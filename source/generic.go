package source

import "text/template"

// GenericDataWatcher is the interface for the watcher passed to
// GenericDataSource.SetWatcher.  Both single-item and batch methods are
// provided.
type GenericDataWatcher interface {
	Active() bool

	// HandleItem is called with a single item of generic unmarshaled data.
	HandleItem(item interface{}) bool

	// HandleItem is called with a batch of generic unmarshaled data.
	HandleItems(items []interface{}) bool
}

// GenericDataSource is a format-agnostic data source
type GenericDataSource interface {
	// Name must return the name of the data source; see DataSource.Name.
	Name() string

	// TextTemplate returns the text/template that is used to construct a
	// TemplatedMarshal to implement the "text" format for this data source.
	TextTemplate() *template.Template
}

// GetableDataSource is the interface implemented by GenericDataSources that
// support Get.  If a GenericDataSource does not implement GetableDataSource,
// then any gets for it return source.ErrNotGetable.
type GetableDataSource interface {
	GenericDataSource

	// Get should return any data available for the data source.
	Get() interface{}
}

// WatchableDataSource is the interface implemented by GenericDataSources that
// support Watch.  If a GenericDataSource does not implement
// WatchableDataSource, then any watches for it return source.ErrNotWatchable.
type WatchableDataSource interface {
	GenericDataSource

	// SetWatcher sets the watcher.
	//
	// Implementations should retain a reference to the last passed watcher,
	// and need not retain multiple; in the usual case this method will only be
	// called once per data source lifecycle.
	//
	// Implementations should pass items to watcher.HandleItem and/or
	// watcher.HandleItems methods.
	//
	// Implementations may use watcher.Active to avoid building items which
	// would just be thrown out by a call to HandleItem(s).
	SetWatcher(watcher GenericDataWatcher)
}

// ActivateWatchableDataSource is an optional interface that
// WatchableDataSources may implement to get notified about source activation.
type ActivateWatchableDataSource interface {
	WatchableDataSource

	// Activate gets called when the GenericDataWatcher transitions from
	// inactive to active.  It may be used by implementations to start or
	// trigger any resources needed to generate items to pass to the set
	// GenericDataWatcher.
	Activate()
}

// WatchInitableDataSource is the interface that a WatchableDataSource should
// implement if it wants to provide an initial data item to all new watch
// streams.
type WatchInitableDataSource interface {
	WatchableDataSource

	// GetInit should returns initial data to send to new watch streams.
	WatchInit() interface{}
}

// GenericDataFormat provides both a data marshaling protocol and a framing
// protocol for the watch stream.  Any marshaling or framing error should cause
// a break in any watch streams subscribed to this format.
type GenericDataFormat interface {
	// Marshal serializes the passed data from GenericDataSource.Get.
	MarshalGet(interface{}) ([]byte, error)

	// Marshal serializes the passed data from GenericDataSource.GetInit.
	MarshalInit(interface{}) ([]byte, error)

	// MarshalItem serializes data passed to a GenericDataWatcher.
	MarshalItem(interface{}) ([]byte, error)

	// FrameItem wraps a MarshalItem-ed byte buffer for a watch stream.
	FrameItem([]byte) ([]byte, error)
}
