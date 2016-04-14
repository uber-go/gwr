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

// MarshaledDataSource wraps a format-agnostic data source and provides one
// or more formats for it
type MarshaledDataSource struct {
	source      GenericDataSource
	formats     map[string]GenericDataMarshal
	formatNames []string
	watchers    map[string]*genericWatcher
	watching    bool
}

// GenericDataWatcher is a type alias for the function signature passed to
// source.Watch.  The contract is that the source should call the last-passed
// watcher until it returns false.  A new watcher passed by a later call to
// source.Watch supercedes any previously passed watcher.
//
// TODO: consider renaming source.Watch to source.SetWatch to better afford it
// singular nature.
type GenericDataWatcher func(interface{}) bool

// GenericDataSource is a format-agnostic data source
type GenericDataSource interface {
	Info() GenericDataSourceInfo
	Get() interface{}
	GetInit() interface{}
	Watch(GenericDataWatcher)
}

// GenericDataSourceInfo describes a format-agnostic data source
type GenericDataSourceInfo struct {
	Name         string
	Attrs        map[string]interface{}
	TextTemplate *template.Template
}

// GenericDataMarshal provides both a data marshaling protocol and a framing
// protocol for the watch stream
type GenericDataMarshal interface {
	// TODO: need? Info() map[string]interface{}
	Marshal(interface{}) ([]byte, error)
	FrameInit(interface{}) ([]byte, error)
	Frame(interface{}) ([]byte, error)
}

type genericWatcher struct {
	source  GenericDataSource
	format  GenericDataMarshal
	writers []io.Writer
}

func (gw *genericWatcher) init(w io.Writer) error {
	if data := gw.source.GetInit(); data != nil {
		format := gw.format
		buf, err := format.FrameInit(data)
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

func (gw *genericWatcher) emit(data interface{}) bool {
	if len(gw.writers) == 0 {
		return false
	}

	buf, err := gw.format.Frame(data)
	if err != nil {
		log.Printf("framing error %v", err)
		return false
	}
	gw.writers = writeToEach(buf, gw.writers)
	return len(gw.writers) != 0
}

// NewMarshaledDataSource creates a MarshaledDataSource for a given
// format-agnostic data source and a map of marshalers
func NewMarshaledDataSource(
	source GenericDataSource,
	formats map[string]GenericDataMarshal,
) *MarshaledDataSource {
	if len(formats) == 0 {
		formats = make(map[string]GenericDataMarshal)
		formats["json"] = LDJSONMarshal
		if info := source.Info(); info.TextTemplate != nil {
			formats["text"] = NewTemplatedMarshal(info.TextTemplate)
		}
	}

	var formatNames []string
	watchers := make(map[string]*genericWatcher, len(formats))
	for name, format := range formats {
		formatNames = append(formatNames, name)
		watchers[name] = &genericWatcher{
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
	buf, err := format.Marshal(data)
	if err != nil {
		log.Printf("marshaling error %v", err)
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

	if !mds.watching {
		mds.source.Watch(mds.emit)
		mds.watching = true
	}

	return nil
}

func (mds *MarshaledDataSource) emit(data interface{}) bool {
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
