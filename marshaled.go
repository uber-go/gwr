package gwr

import (
	"errors"
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
// more formats for it.
//
// MarshaledDataSource implements:
// - DataSource to satisfy DataSources and low level protocols
// - ItemDataSource so that higher level protocols may add their own framing
// - GenericDataWatcher inwardly to the wrapped GenericDataSource
type MarshaledDataSource struct {
	source      GenericDataSource
	formats     map[string]GenericDataFormat
	formatNames []string
	watchers    map[string]*marshaledWatcher
	watching    bool
	itemChan    chan interface{}
	itemsChan   chan []interface{}
}

// GenericDataWatcher is the interface for the watcher passed to
// GenericDataSource.Watch. Both single-item and batch methods are provided.
type GenericDataWatcher interface {
	// HandleItem is called with a single item of generic unmarshaled data.
	HandleItem(item interface{}) bool

	// HandleItem is called with a batch of generic unmarshaled data.
	HandleItems(items []interface{}) bool
}

// GenericDataSource is a format-agnostic data source
type GenericDataSource interface {
	// Name must return the name of the data source; see DataSource.Name.
	Name() string

	// Attrs returns any descriptors of the generic data source; see
	// DataSource.Name.
	Attrs() map[string]interface{}

	// TextTemplate returns the text/template that is used to construct a
	// TemplatedMarshal to implement the "text" format for this data source.
	TextTemplate() *template.Template

	// Get should return any data available for the data source.  A nil value
	// should  result in a ErrNotGetable.  If a generic data source wants a
	// marshaled null value, its Get must return a non-nil interface value.
	Get() interface{}

	// GetInit should return any initial data to send to a new watch stream.
	// Similarly to Get a nil value will not be marshaled, but no error will be
	// returned to the Watch request.
	GetInit() interface{}

	// Watch sets the current (singular!) watcher.  Implementations must call
	// the passed watcher until it returns false, or until a new watcher is
	// passed by a future call of Watch.
	Watch(GenericDataWatcher)
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
	source   GenericDataSource
	format   GenericDataFormat
	dfw      defaultFrameWatcher
	watchers []ItemWatcher
}

func newMarshaledWatcher(source GenericDataSource, format GenericDataFormat) *marshaledWatcher {
	mw := &marshaledWatcher{source: source, format: format}
	mw.dfw.format = format
	return mw
}

func (mw *marshaledWatcher) Close() error {
	var errs []error
	for _, watcher := range mw.watchers {
		if closer, ok := watcher.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				if errs == nil {
					errs = make([]error, 0, len(mw.watchers))
				}
				errs = append(errs, err)
			}
		}
	}
	mw.watchers = mw.watchers[:0]
	return MultiErr(errs).AsError()
}

func (mw *marshaledWatcher) init(w io.Writer) error {
	if err := mw.dfw.init(mw.source.GetInit(), w); err != nil {
		return err
	}
	if len(mw.dfw.writers) == 1 {
		mw.watchers = append(mw.watchers, &mw.dfw)
	}
	return nil
}

func (mw *marshaledWatcher) initItems(iw ItemWatcher) error {
	if data := mw.source.GetInit(); data != nil {
		if buf, err := mw.format.MarshalInit(data); err != nil {
			log.Printf("initial marshaling error %v", err)
			return err
		} else if err := iw.HandleItem(buf); err != nil {
			return err
		}
	}
	mw.watchers = append(mw.watchers, iw)
	return nil
}

func (mw *marshaledWatcher) emit(item interface{}) bool {
	if len(mw.watchers) == 0 {
		return false
	}
	data, err := mw.format.MarshalItem(item)
	if err != nil {
		log.Printf("item marshaling error %v", err)
		return false
	}

	var failed []int // TODO: could carry this rather than allocate on failure
	for i, iw := range mw.watchers {
		if err := iw.HandleItem(data); err != nil {
			if failed == nil {
				failed = make([]int, 0, len(mw.watchers))
			}
			failed = append(failed, i)
		}
	}
	if len(failed) == 0 {
		return true
	}

	var (
		okay   []ItemWatcher
		remain = len(mw.watchers) - len(failed)
	)
	if remain > 0 {
		okay = make([]ItemWatcher, 0, remain)
	}
	for i, iw := range mw.watchers {
		if i != failed[0] {
			okay = append(okay, iw)
		}
		if i >= failed[0] {
			failed = failed[1:]
			if len(failed) == 0 {
				if j := i + 1; j < len(mw.watchers) {
					okay = append(okay, mw.watchers[j:]...)
				}
				break
			}
		}
	}
	mw.watchers = okay

	return len(mw.watchers) != 0
}

func (mw *marshaledWatcher) emitBatch(items []interface{}) bool {
	if len(mw.watchers) == 0 {
		return false
	}

	data := make([][]byte, len(items))
	for i, item := range items {
		buf, err := mw.format.MarshalItem(item)
		if err != nil {
			log.Printf("item marshaling error %v", err)
			return false
		}
		data[i] = buf
	}

	var failed []int // TODO: could carry this rather than allocate on failure
	for i, iw := range mw.watchers {
		if err := iw.HandleItems(data); err != nil {
			if failed == nil {
				failed = make([]int, 0, len(mw.watchers))
			}
			failed = append(failed, i)
		}
	}
	if len(failed) == 0 {
		return true
	}

	var (
		okay   []ItemWatcher
		remain = len(mw.watchers) - len(failed)
	)
	if remain > 0 {
		okay = make([]ItemWatcher, 0, remain)
	}
	for i, iw := range mw.watchers {
		if i != failed[0] {
			okay = append(okay, iw)
		}
		if i >= failed[0] {
			failed = failed[1:]
			if len(failed) == 0 {
				if j := i + 1; j < len(mw.watchers) {
					okay = append(okay, mw.watchers[j:]...)
				}
				break
			}
		}
	}
	mw.watchers = okay

	return len(mw.watchers) != 0
}

// NewMarshaledDataSource creates a MarshaledDataSource for a given
// format-agnostic data source and a map of marshalers
func NewMarshaledDataSource(
	source GenericDataSource,
	formats map[string]GenericDataFormat,
) *MarshaledDataSource {
	if len(formats) == 0 {
		formats = make(map[string]GenericDataFormat)
	}

	// standard json protocol
	if formats["json"] == nil {
		formats["json"] = LDJSONMarshal
	}

	// convenience templated text protocol
	if tt := source.TextTemplate(); tt != nil && formats["text"] == nil {
		formats["text"] = NewTemplatedMarshal(tt)
	}

	// TODO: source should be able to declare some formats in addition to any
	// integratgor

	var formatNames []string
	watchers := make(map[string]*marshaledWatcher, len(formats))
	for name, format := range formats {
		formatNames = append(formatNames, name)
		watchers[name] = newMarshaledWatcher(source, format)
	}
	return &MarshaledDataSource{
		source:      source,
		formats:     formats,
		formatNames: formatNames,
		watchers:    watchers,
	}
}

// Name passes through the GenericDataSource.Name()
func (mds *MarshaledDataSource) Name() string {
	return mds.source.Name()
}

// Formats returns the list of supported format names.
func (mds *MarshaledDataSource) Formats() []string {
	return mds.formatNames
}

// Attrs returns arbitrary description information about the data source.
func (mds *MarshaledDataSource) Attrs() map[string]interface{} {
	// TODO: support per-format Attrs?
	return mds.source.Attrs()
}

// Get marshals data source's Get data to the writer
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

// Watch marshals any data source GetInit data to the writer, and then
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

	mds.startWatching()

	return nil
}

// WatchItems marshals any data source GetInit data as a single item to the
// ItemWatcher's HandleItem method.  The watcher is then retained and future
// items are marshaled to its HandleItem method.
func (mds *MarshaledDataSource) WatchItems(formatName string, iw ItemWatcher) error {
	watcher, ok := mds.watchers[strings.ToLower(formatName)]
	if !ok {
		return ErrUnsupportedFormat
	}

	if err := watcher.initItems(iw); err != nil {
		return err
	}

	mds.startWatching()

	return nil
}

func (mds *MarshaledDataSource) startWatching() {
	// TODO: we probably need synchronized access to watching and co
	// TODO: we could optimize the only-one-format-being-watched case
	if mds.watching {
		return
	}
	mds.source.Watch(mds)
	// TODO: tune size
	mds.itemChan = make(chan interface{}, 100)
	mds.itemsChan = make(chan []interface{}, 100)
	mds.watching = true
	go mds.processItemChan()
}

func (mds *MarshaledDataSource) stopWatching() {
	// TODO: we probably need synchronized access to watching and co
	if !mds.watching {
		return
	}
	mds.watching = false
	for _, watcher := range mds.watchers {
		watcher.Close()
	}
}

func (mds *MarshaledDataSource) processItemChan() {
	for mds.watching {
		any := false

		select {
		case item := <-mds.itemChan:
			for _, watcher := range mds.watchers {
				if watcher.emit(item) {
					any = true
				}
			}

		case items := <-mds.itemsChan:
			for _, watcher := range mds.watchers {
				if watcher.emitBatch(items) {
					any = true
				}
			}
		}

		if !any {
			mds.stopWatching()
		}
	}
	mds.itemChan = nil
	mds.itemsChan = nil
}

// HandleItem implements GenericDataWatcher.HandleItem by passing the item to
// all current marshaledWatchers.
func (mds *MarshaledDataSource) HandleItem(item interface{}) bool {
	if !mds.watching {
		return false
	}
	select {
	case mds.itemChan <- item:
		return true
	default:
		mds.stopWatching()
		return false
	}
}

// HandleItems implements GenericDataWatcher.HandleItems by passing the batch
// to all current marshaledWatchers.
func (mds *MarshaledDataSource) HandleItems(items []interface{}) bool {
	if !mds.watching {
		return false
	}
	select {
	case mds.itemsChan <- items:
		return true
	default:
		mds.stopWatching()
		return false
	}
}

var errDefaultFrameWatcherDone = errors.New("all defaultFrameWatcher writers done")

type defaultFrameWatcher struct {
	format  GenericDataFormat
	writers []io.Writer
}

func (dfw *defaultFrameWatcher) init(data interface{}, w io.Writer) error {
	if data != nil {
		buf, err := dfw.format.MarshalInit(data)
		if err != nil {
			log.Printf("initial marshaling error %v", err)
			return err
		}
		buf, err = dfw.format.FrameItem(buf)
		if err != nil {
			log.Printf("initial framing error %v", err)
			return err
		}
		if _, err := w.Write(buf); err != nil {
			return err
		}
	}
	dfw.writers = append(dfw.writers, w)
	return nil
}

func (dfw *defaultFrameWatcher) HandleItem(item []byte) error {
	if len(dfw.writers) == 0 {
		return errDefaultFrameWatcherDone
	}
	if buf, err := dfw.format.FrameItem(item); err != nil {
		log.Printf("item framing error %v", err)
		return err
	} else if err := dfw.writeToAll(buf); err != nil {
		return err
	}
	return nil
}

func (dfw *defaultFrameWatcher) HandleItems(items [][]byte) error {
	if len(dfw.writers) == 0 {
		return errDefaultFrameWatcherDone
	}
	for _, item := range items {
		if buf, err := dfw.format.FrameItem(item); err != nil {
			log.Printf("item framing error %v", err)
			return err
		} else if err := dfw.writeToAll(buf); err != nil {
			return err
		}
	}
	return nil
}

func (dfw *defaultFrameWatcher) Close() error {
	var errs []error
	for _, writer := range dfw.writers {
		if closer, ok := writer.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				if errs == nil {
					errs = make([]error, 0, len(dfw.writers))
				}
				errs = append(errs, err)
			}
		}
	}
	dfw.writers = dfw.writers[:0]
	return MultiErr(errs).AsError()
}

func (dfw *defaultFrameWatcher) writeToAll(buf []byte) error {
	// TODO: avoid blocking fan out, parallelize; error back-propagation then
	// needs to happen over another channel

	var failed []int // TODO: could carry this rather than allocate on failure
	for i, w := range dfw.writers {
		if _, err := w.Write(buf); err != nil {
			if failed == nil {
				failed = make([]int, 0, len(dfw.writers))
			}
			failed = append(failed, i)
		}
	}
	if len(failed) == 0 {
		return nil
	}

	var (
		okay   []io.Writer
		remain = len(dfw.writers) - len(failed)
	)
	if remain > 0 {
		okay = make([]io.Writer, 0, remain)
	}
	for i, w := range dfw.writers {
		if i != failed[0] {
			okay = append(okay, w)
		}
		if i >= failed[0] {
			failed = failed[1:]
			if len(failed) == 0 {
				if j := i + 1; j < len(dfw.writers) {
					okay = append(okay, dfw.writers[j:]...)
				}
				break
			}
		}
	}
	dfw.writers = okay

	if len(dfw.writers) == 0 {
		return errDefaultFrameWatcherDone
	}
	return nil
}
