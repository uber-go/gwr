package marshaled

import (
	"io"
	"log"
	"sort"
	"strings"

	"github.com/uber-go/gwr/source"
)

// TODO: punts on any locking concerns

// NOTE: This approach is perhaps overfit to the json module's marshalling
// mindset.  A better interface (for performance) would work by passing a
// writer to the specific encoder, rather than a []byte-returning Marshal
// function.  This would be possible perhaps using something like
// io.MultiWriter.

// DataSource wraps a format-agnostic data source and provides one or
// more formats for it.
//
// DataSource implements:
// - DataSource to satisfy DataSources and low level protocols
// - ItemDataSource so that higher level protocols may add their own framing
// - GenericDataWatcher inwardly to the wrapped GenericDataSource
type DataSource struct {
	// TODO: better to have alternate implementations for each combination
	// rather than one with these nil checks
	source      source.GenericDataSource
	getSource   source.GetableDataSource
	watchSource source.WatchableDataSource
	watiSource  source.WatchInitableDataSource

	formats     map[string]source.GenericDataFormat
	formatNames []string
	watchers    map[string]*marshaledWatcher
	watching    bool
	itemChan    chan interface{}
	itemsChan   chan []interface{}
}

// NewDataSource creates a DataSource for a given format-agnostic data source
// and a map of marshalers
func NewDataSource(
	src source.GenericDataSource,
	formats map[string]source.GenericDataFormat,
) *DataSource {
	if len(formats) == 0 {
		formats = make(map[string]source.GenericDataFormat)
	}

	// standard json protocol
	if formats["json"] == nil {
		formats["json"] = LDJSONMarshal
	}

	// convenience templated text protocol
	if tt := src.TextTemplate(); tt != nil && formats["text"] == nil {
		formats["text"] = NewTemplatedMarshal(tt)
	}

	// TODO: source should be able to declare some formats in addition to any
	// integratgor

	ds := &DataSource{
		source:   src,
		formats:  formats,
		watchers: make(map[string]*marshaledWatcher, len(formats)),
	}
	ds.getSource, _ = src.(source.GetableDataSource)
	ds.watchSource, _ = src.(source.WatchableDataSource)
	ds.watiSource, _ = src.(source.WatchInitableDataSource)
	for name, format := range formats {
		ds.formatNames = append(ds.formatNames, name)
		ds.watchers[name] = newMarshaledWatcher(ds, format)
	}
	sort.Strings(ds.formatNames)
	return ds
}

// Name passes through the GenericDataSource.Name()
func (mds *DataSource) Name() string {
	return mds.source.Name()
}

// Formats returns the list of supported format names.
func (mds *DataSource) Formats() []string {
	return mds.formatNames
}

// Attrs returns arbitrary description information about the data source.
func (mds *DataSource) Attrs() map[string]interface{} {
	// TODO: support per-format Attrs?
	// TODO: any support for per-source Attrs?
	return nil
}

// Get marshals data source's Get data to the writer
func (mds *DataSource) Get(formatName string, w io.Writer) error {
	if mds.getSource == nil {
		return source.ErrNotGetable
	}
	format, ok := mds.formats[strings.ToLower(formatName)]
	if !ok {
		return source.ErrUnsupportedFormat
	}
	data := mds.getSource.Get()
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
func (mds *DataSource) Watch(formatName string, w io.Writer) error {
	if mds.watchSource == nil {
		return source.ErrNotWatchable
	}
	watcher, ok := mds.watchers[strings.ToLower(formatName)]
	if !ok {
		return source.ErrUnsupportedFormat
	}
	if err := watcher.init(w); err != nil {
		return err
	}
	return mds.startWatching()
}

// WatchItems marshals any data source GetInit data as a single item to the
// ItemWatcher's HandleItem method.  The watcher is then retained and future
// items are marshaled to its HandleItem method.
func (mds *DataSource) WatchItems(formatName string, iw source.ItemWatcher) error {
	if mds.watchSource == nil {
		return source.ErrNotWatchable
	}
	watcher, ok := mds.watchers[strings.ToLower(formatName)]
	if !ok {
		return source.ErrUnsupportedFormat
	}
	if err := watcher.initItems(iw); err != nil {
		return err
	}
	return mds.startWatching()
}

func (mds *DataSource) startWatching() error {
	// TODO: we probably need synchronized access to watching and co
	// TODO: we could optimize the only-one-format-being-watched case
	if mds.watching {
		return nil
	}
	mds.watchSource.SetWatcher(mds)
	// TODO: tune size
	mds.itemChan = make(chan interface{}, 100)
	mds.itemsChan = make(chan []interface{}, 100)
	mds.watching = true
	go mds.processItemChan()
	return nil
}

func (mds *DataSource) stopWatching() {
	// TODO: we probably need synchronized access to watching and co
	if !mds.watching {
		return
	}
	mds.watching = false
	for _, watcher := range mds.watchers {
		watcher.Close()
	}
}

func (mds *DataSource) processItemChan() {
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
func (mds *DataSource) HandleItem(item interface{}) bool {
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
func (mds *DataSource) HandleItems(items []interface{}) bool {
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
