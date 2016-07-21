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

package marshaled

import (
	"fmt"
	"io"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/uber-go/gwr/source"
)

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
	actiSource  source.ActivateWatchableDataSource

	formats     map[string]source.GenericDataFormat
	formatNames []string
	maxItems    int
	maxBatches  int
	maxWait     time.Duration

	procs     sync.WaitGroup
	watchLock sync.Mutex
	watchers  map[string]*marshaledWatcher
	active    bool
	itemChan  chan interface{}
	itemsChan chan []interface{}
}

func stringIt(item interface{}) ([]byte, error) {
	var s string
	if ss, ok := item.(fmt.Stringer); ok {
		s = ss.String()
	} else {
		s = fmt.Sprintf("%+v", item)
	}
	return []byte(s), nil
}

// NewDataSource creates a DataSource for a given format-agnostic data source
// and a map of marshalers
func NewDataSource(
	src source.GenericDataSource,
	formats map[string]source.GenericDataFormat,
) *DataSource {
	if formats == nil {
		formats = make(map[string]source.GenericDataFormat)
	}

	// source-defined formats
	if fmtsrc, ok := src.(source.GenericDataSourceFormats); ok {
		fmts := fmtsrc.Formats()
		for name, fmt := range fmts {
			formats[name] = fmt
		}
	}

	// standard json protocol
	if formats["json"] == nil {
		formats["json"] = LDJSONMarshal
	}

	// convenience templated text protocol
	if formats["text"] == nil {
		if txtsrc, ok := src.(source.TextTemplatedSource); ok {
			if tt := txtsrc.TextTemplate(); tt != nil {
				formats["text"] = NewTemplatedMarshal(tt)
			}
		}
	}

	// default to just string-ing it
	if formats["text"] == nil {
		formats["text"] = source.GenericDataFormatFunc(stringIt)
	}

	ds := &DataSource{
		source:   src,
		formats:  formats,
		watchers: make(map[string]*marshaledWatcher, len(formats)),
		// TODO: tunable
		maxItems:   100,
		maxBatches: 100,
		maxWait:    100 * time.Microsecond,
	}
	ds.getSource, _ = src.(source.GetableDataSource)
	ds.watchSource, _ = src.(source.WatchableDataSource)
	ds.watiSource, _ = src.(source.WatchInitableDataSource)
	ds.actiSource, _ = src.(source.ActivateWatchableDataSource)
	for name, format := range formats {
		ds.formatNames = append(ds.formatNames, name)
		ds.watchers[name] = newMarshaledWatcher(ds, format)
	}
	sort.Strings(ds.formatNames)

	if ds.watchSource != nil {
		ds.watchSource.SetWatcher(ds)
	}

	return ds
}

// Active returns true if there are any active watchers, false otherwise.  If
// Active returns false, so will any calls to HandleItem and HandleItems.
func (mds *DataSource) Active() bool {
	mds.watchLock.Lock()
	r := mds.active && mds.itemChan != nil && mds.itemsChan != nil
	mds.watchLock.Unlock()
	return r
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

	mds.watchLock.Lock()
	acted := !mds.active
	err := func() error {
		defer mds.watchLock.Unlock()
		watcher, ok := mds.watchers[strings.ToLower(formatName)]
		if !ok {
			return source.ErrUnsupportedFormat
		}
		if err := watcher.init(w); err != nil {
			return err
		}
		if err := mds.startWatching(); err != nil {
			return err
		}
		return nil
	}()

	if err == nil && acted && mds.actiSource != nil {
		mds.actiSource.Activate()
	}
	return err
}

// WatchItems marshals any data source GetInit data as a single item to the
// ItemWatcher's HandleItem method.  The watcher is then retained and future
// items are marshaled to its HandleItem method.
func (mds *DataSource) WatchItems(formatName string, iw source.ItemWatcher) error {
	if mds.watchSource == nil {
		return source.ErrNotWatchable
	}

	mds.watchLock.Lock()
	acted := !mds.active
	err := func() error {
		defer mds.watchLock.Unlock()
		watcher, ok := mds.watchers[strings.ToLower(formatName)]
		if !ok {
			return source.ErrUnsupportedFormat
		}
		if err := watcher.initItems(iw); err != nil {
			return err
		}
		if err := mds.startWatching(); err != nil {
			return err
		}
		return nil
	}()

	if err == nil && acted && mds.actiSource != nil {
		mds.actiSource.Activate()
	}
	return err
}

// startWatching flips the active bit, creates new item channels, and starts a
// processing go routine; it assumes that the watchLock is being held by the
// caller.
func (mds *DataSource) startWatching() error {
	// TODO: we could optimize the only-one-format-being-watched case
	if mds.active {
		return nil
	}
	mds.active = true
	mds.itemChan = make(chan interface{}, mds.maxItems)
	mds.itemsChan = make(chan []interface{}, mds.maxBatches)
	mds.procs.Add(1)
	go mds.processItemChan(mds.itemChan, mds.itemsChan)
	return nil
}

// Drain closes the item channels, and waits for the item processor to finish.
// After drain, any remaining watchers are closed, and the source goes
// inactive.
func (mds *DataSource) Drain() {
	mds.watchLock.Lock()
	any := false
	if mds.itemChan != nil {
		close(mds.itemChan)
		any = true
		mds.itemChan = nil
	}
	if mds.itemsChan != nil {
		close(mds.itemsChan)
		any = true
		mds.itemsChan = nil
	}
	if any {
		mds.watchLock.Unlock()
		mds.procs.Wait()
		mds.watchLock.Lock()
	}
	stop := mds.active
	if stop {
		mds.active = false
	}
	mds.watchLock.Unlock()

	if stop {
		for _, watcher := range mds.watchers {
			watcher.Close()
		}
	}
}

func (mds *DataSource) processItemChan(itemChan chan interface{}, itemsChan chan []interface{}) {
	defer mds.procs.Done()

	stop := false

loop:
	for {
		mds.watchLock.Lock()
		active := mds.active
		watchers := mds.watchers
		mds.watchLock.Unlock()
		if !active {
			break loop
		}
		select {
		case item, ok := <-itemChan:
			if !ok {
				itemChan = nil
				continue loop
			}
			any := false
			for _, watcher := range watchers {
				if watcher.emit(item) {
					any = true
				}
			}
			if !any {
				stop = true
				break loop
			}

		case items, ok := <-itemsChan:
			if !ok {
				itemsChan = nil
				continue loop
			}
			any := false
			for _, watcher := range watchers {
				if watcher.emitBatch(items) {
					any = true
				}
			}
			if !any {
				stop = true
				break loop
			}

		default:
			if itemChan == nil && itemsChan == nil {
				break loop
			}
		}
	}

	mds.watchLock.Lock()
	if mds.itemChan == itemChan {
		mds.itemChan = nil
	}
	if mds.itemsChan == itemsChan {
		mds.itemsChan = nil
	}
	if stop {
		mds.active = false
	}
	mds.watchLock.Unlock()

	if stop {
		for _, watcher := range mds.watchers {
			watcher.Close()
		}
	}
}

// HandleItem implements GenericDataWatcher.HandleItem by passing the item to
// all current marshaledWatchers.
func (mds *DataSource) HandleItem(item interface{}) bool {
	if !mds.Active() {
		return false
	}
	select {
	case mds.itemChan <- item:
		return true
	case <-time.After(mds.maxWait):
		mds.watchLock.Lock()
		if !mds.active {
			mds.watchLock.Unlock()
			return false
		}
		mds.active = false
		mds.watchLock.Unlock()
		for _, watcher := range mds.watchers {
			watcher.Close()
		}
		return false
	}
}

// HandleItems implements GenericDataWatcher.HandleItems by passing the batch
// to all current marshaledWatchers.
func (mds *DataSource) HandleItems(items []interface{}) bool {
	if !mds.Active() {
		return false
	}
	select {
	case mds.itemsChan <- items:
		return true
	case <-time.After(mds.maxWait):
		mds.watchLock.Lock()
		if !mds.active {
			mds.watchLock.Unlock()
			return false
		}
		mds.active = false
		mds.watchLock.Unlock()
		for _, watcher := range mds.watchers {
			watcher.Close()
		}
		return false
	}
}
