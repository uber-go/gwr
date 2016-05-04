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
