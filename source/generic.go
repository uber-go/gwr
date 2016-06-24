// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

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
}

// TextTemplatedSource is implemented by generic data sources to provide a
// convenience template for the "text" format.
type TextTemplatedSource interface {
	// TextTemplate returns the text/template that is used to construct a
	// TemplatedMarshal to implement the "text" format for this data source.
	TextTemplate() *template.Template
}

// GenericDataSourceFormats is implemented by generic data sources to define
// additional formats beyond the default json and templated text ones.
type GenericDataSourceFormats interface {
	Formats() map[string]GenericDataFormat
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
	// MarshalGet serializes the passed data from GenericDataSource.Get.
	MarshalGet(interface{}) ([]byte, error)

	// MarshalInit serializes the passed data from GenericDataSource.GetInit.
	MarshalInit(interface{}) ([]byte, error)

	// MarshalItem serializes data passed to a GenericDataWatcher.
	MarshalItem(interface{}) ([]byte, error)

	// FrameItem wraps a MarshalItem-ed byte buffer for a watch stream.
	FrameItem([]byte) ([]byte, error)
}
