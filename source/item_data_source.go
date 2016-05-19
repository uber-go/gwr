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

// ItemDataSource is an interface implemented by a data source to provide
// marshaled but unframed streams of Watch items.  When implemented protocol
// libraries can add protocol-specific builtin framing.
type ItemDataSource interface {
	// WatchItems implementations have all of the semantics of
	// DataSource.Watch, just over an ItemWatcher instead of an io.Writer.
	//
	// If the data source naturally has batches of items on hand it may call
	// watcher.HandleItems.
	//
	// Passed watchers must be discarded after either HandleItem or
	// HandleItems returns a non-nil error.
	WatchItems(format string, watcher ItemWatcher) error
}

// ItemWatcher is the interface passed to ItemSource.WatchItems.  Any
// error returned by either HandleItem or HandleItems indicates that this
// watcher should not be called with more items.
type ItemWatcher interface {
	// HandleItem gets a single marshaled item, and should return any framing
	// or write error.
	HandleItem(item []byte) error

	// HandleItems gets a batch of marshaled items, and should return any
	// framing or write error.
	HandleItems(items [][]byte) error
}

// ItemWatcherFunc is a convenience type for watching with a simple
// per-item function.  The item function should return any framing or write
// error.
type ItemWatcherFunc func([]byte) error

// HandleItem just calls the wrapped function.
func (itemFunc ItemWatcherFunc) HandleItem(item []byte) error {
	return itemFunc(item)
}

// HandleItems calls the wrapped function for each item in the batch, stopping
// on and returning the first error.
func (itemFunc ItemWatcherFunc) HandleItems(items [][]byte) error {
	for _, item := range items {
		if err := itemFunc(item); err != nil {
			return err
		}
	}
	return nil
}

// ItemWatcherBatchFunc is a convenience type for watching with a simple
// batch function.  The batch function should return any framing or write
// error.
type ItemWatcherBatchFunc func([][]byte) error

// HandleItem calls the wrapped function with a singleton batch.
func (batchFunc ItemWatcherBatchFunc) HandleItem(item []byte) error {
	return batchFunc([][]byte{item})
}

// HandleItems just calls the wrapped function.
func (batchFunc ItemWatcherBatchFunc) HandleItems(items [][]byte) error {
	return batchFunc(items)
}
