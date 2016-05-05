package source

import "text/template"

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

	// SetWatcher sets the current (singular!) watcher.  Implementations must
	// call the passed watcher until it returns false, or until a new watcher
	// is passed by a future call of SetWatcher.
	SetWatcher(GenericDataWatcher)
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
