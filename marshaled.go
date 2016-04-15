package gwr

import (
	"io"
	"log"
	"strings"
	"text/template"
)

// TODO: punts on any locking concerns
// TODO: .emit(interface{}) vs chan interface{}

// NOTE: This approach is perhaps overfit to the json module's marshalling
// mindset.  A better interface (for performance) would work by passing a
// writer to the specific encoder, rather than a []byte-returning Marshal
// function.  This would be possible perhaps using something like
// io.MultiWriter.

// MarshaledDataSource wraps a format-agnostic data source and provides one or
// more formats for it
type MarshaledDataSource struct {
	source      GenericDataSource
	formats     map[string]GenericDataFormat
	formatNames []string
	watchers    map[string]*marshaledWatcher
	watching    bool
}

// GenericDataWatcher is a type alias for the function signature passed to
// source.Watch.
type GenericDataWatcher func(interface{}) bool

// GenericDataSource is a format-agnostic data source
type GenericDataSource interface {
	// Info returns a description of the data source
	Info() GenericDataSourceInfo

	// Get should return any data available for the data source.  A nil value
	// should  result in a ErrNotGetable.  If a generic data source wants a
	// marshaled null value, its Get must return a non-nil interface value.
	Get() interface{}

	// GetInit should return any inital data to send to a new watch stream.
	// Similarly to Get a nil value will not be marshaled, but no error will be
	// returned to the Watch request.
	GetInit() interface{}

	// Watch sets the current (singular!) watcher.  Implementations must call
	// the passed watcher until it returns false, or until a new watcher is
	// passed by a future call of Watch.
	Watch(GenericDataWatcher)
}

// GenericDataSourceInfo describes a format-agnostic data source
type GenericDataSourceInfo struct {
	Name         string
	Attrs        map[string]interface{}
	TextTemplate *template.Template
}

// GenericDataFormat provides both a data marshaling protocol and a framing
// protocol for the watch stream.  Any marshaling or framing error should cause
// a break in any watch streams subscribed to this format.
type GenericDataFormat interface {
	// Marshal serializes the passed data from GenericDataSource.Get.
	MarshalGet(interface{}) ([]byte, error)

	// Marshal serializes the passed data from GenericDataSource.GetInit.
	MarshalInit(interface{}) ([]byte, error)

	// Marshal serializes data passed to a GenericDataWatcher.
	MarshalItem(interface{}) ([]byte, error)

	// FrameItem wraps a MarshalItem-ed byte buffer for a watch stream.
	FrameItem([]byte) ([]byte, error)
}

// marshaledWatcher manages all of the low level io.Writers for a given format.
// Instances are created once for each MarshaledDataSource.
//
// MarshaledDataSource then manages calling marshaledWatcher.emit for each data
// item as long as there is one valid io.Writer for a given format.  Once the
// last marshaledWatcher goes idle, the underlying GenericDataSource watch is
// ended.
type marshaledWatcher struct {
	source  GenericDataSource
	format  GenericDataFormat
	writers []io.Writer
}

func (gw *marshaledWatcher) init(w io.Writer) error {
	if data := gw.source.GetInit(); data != nil {
		format := gw.format
		buf, err := format.MarshalInit(data)
		if err != nil {
			log.Printf("inital marshaling error %v", err)
			return err
		}
		buf, err = format.FrameItem(buf)
		if err != nil {
			log.Printf("inital framing error %v", err)
			return err
		}
		_, err = w.Write(buf)
		if err != nil {
			return err
		}
	}
	gw.writers = append(gw.writers, w)
	return nil
}

func (gw *marshaledWatcher) emit(data interface{}) bool {
	if len(gw.writers) == 0 {
		return false
	}
	buf, err := gw.format.MarshalItem(data)
	if err != nil {
		log.Printf("item marshaling error %v", err)
		return false
	}
	buf, err = gw.format.FrameItem(buf)
	if err != nil {
		log.Printf("item framing error %v", err)
		return false
	}

	// TODO: avoid blocking fan out, parallelize; error back-propagation then
	// needs to happen over another channel

	var failed []int // TODO: could carry this rather than allocate on failure
	for i, w := range gw.writers {
		if _, err := w.Write(buf); err != nil {
			if failed == nil {
				failed = make([]int, 0, len(gw.writers))
			}
			failed = append(failed, i)
		}
	}
	if len(failed) == 0 {
		return true
	}

	var (
		okay []io.Writer
		remain = len(gw.writers) - len(failed)
	)
	if remain > 0 {
		okay = make([]io.Writer, 0, remain)
	}
	for i, w := range gw.writers {
		if i != failed[0] {
			okay = append(okay, w)
		}
		if i >= failed[0] {
			failed = failed[1:]
			if len(failed) == 0 {
				if j := i + 1; j < len(gw.writers) {
					okay = append(okay, gw.writers[j:]...)
				}
				break
			}
		}
	}
	gw.writers = okay

	return len(gw.writers) != 0
}

// NewMarshaledDataSource creates a MarshaledDataSource for a given
// format-agnostic data source and a map of marshalers
func NewMarshaledDataSource(
	source GenericDataSource,
	formats map[string]GenericDataFormat,
) *MarshaledDataSource {
	var formatNames []string

	// we need room for json and text defaults plus any specified
	n := len(formats)
	if formats["json"] == nil {
		n++
	}
	if formats["text"] == nil {
		// may over estimate by one if source has no TextTemplate; probably not
		// a big deal
		n++
	}
	watchers := make(map[string]*marshaledWatcher, n)

	// standard json protocol
	if formats["json"] == nil {
		formatNames = append(formatNames, "json")
		watchers["json"] = &marshaledWatcher{
			source:  source,
			format:  LDJSONMarshal,
			writers: nil,
		}
	}

	// convenience templated text protocol
	if tt := source.Info().TextTemplate; tt != nil && formats["text"] == nil {
		formatNames = append(formatNames, "text")
		watchers["text"] = &marshaledWatcher{
			source:  source,
			format:  NewTemplatedMarshal(tt),
			writers: nil,
		}
	}

	// TODO: source should be able to declare some formats in addition to any
	// integratgor

	for name, format := range formats {
		formatNames = append(formatNames, name)
		watchers[name] = &marshaledWatcher{
			source:  source,
			format:  format,
			writers: nil,
		}
	}

	return &MarshaledDataSource{
		source:      source,
		formats:     formats,
		formatNames: formatNames,
		watchers:    watchers,
	}
}

// Info returns the generic data source description, plus any format specific
// description
func (mds *MarshaledDataSource) Info() DataSourceInfo {
	info := mds.source.Info()
	// TODO: any need for per-format Attrs?
	return DataSourceInfo{
		Name:    info.Name,
		Formats: mds.formatNames,
		Attrs:   info.Attrs,
	}
}

// Get marshals the agnostic data source's Get data to the writer
func (mds *MarshaledDataSource) Get(formatName string, w io.Writer) error {
	format, ok := mds.formats[strings.ToLower(formatName)]
	if !ok {
		return ErrUnsupportedFormat
	}
	data := mds.source.Get()
	if data == nil {
		return ErrNotGetable
	}
	buf, err := format.MarshalGet(data)
	if err != nil {
		log.Printf("get marshaling error %v", err)
		return err
	}
	_, err = w.Write(buf)
	return err
}

// Watch marshals any agnostic data source GetInit data to the writer, and then
// retains a reference to the writer so that any future agnostic data source
// Watch(emit)'ed data gets marshaled to it as well
func (mds *MarshaledDataSource) Watch(formatName string, w io.Writer) error {
	watcher, ok := mds.watchers[strings.ToLower(formatName)]
	if !ok {
		return ErrUnsupportedFormat
	}

	if err := watcher.init(w); err != nil {
		return err
	}

	// TODO: we could optimize the only-one-format-being-watched case
	if !mds.watching {
		mds.source.Watch(mds.emit)
		mds.watching = true
	}

	return nil
}

func (mds *MarshaledDataSource) emit(data interface{}) bool {
	if !mds.watching {
		return false
	}
	any := false
	for _, watcher := range mds.watchers {
		if watcher.emit(data) {
			any = true
		}
	}
	if !any {
		mds.watching = false
	}
	return any
}
