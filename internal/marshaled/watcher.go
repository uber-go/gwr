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
	"errors"
	"io"
	"log"

	"github.com/uber-go/gwr/internal"
	"github.com/uber-go/gwr/source"
)

var errDefaultFrameWatcherDone = errors.New("all defaultFrameWatcher writers done")

// marshaledWatcher manages all of the low level io.Writers for a given format.
// Instances are created once for each DataSource.
//
// DataSource then manages calling marshaledWatcher.emit for each data item as
// long as there is one valid io.Writer for a given format.  Once the last
// marshaledWatcher goes idle, the underlying GenericDataSource watch is ended.
type marshaledWatcher struct {
	source   *DataSource
	format   source.GenericDataFormat
	dfw      defaultFrameWatcher
	watchers []source.ItemWatcher
}

func newMarshaledWatcher(src *DataSource, format source.GenericDataFormat) *marshaledWatcher {
	mw := &marshaledWatcher{source: src, format: format}
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
	return internal.MultiErr(errs).AsError()
}

func (mw *marshaledWatcher) init(w io.Writer) error {
	if mw.source.watiSource != nil {
		initData := mw.source.watiSource.WatchInit()
		if err := mw.dfw.writeInitData(initData, w); err != nil {
			return err
		}
	}
	mw.dfw.writers = append(mw.dfw.writers, w)
	if len(mw.dfw.writers) == 1 {
		mw.watchers = append(mw.watchers, &mw.dfw)
	}
	return nil
}

func (mw *marshaledWatcher) initItems(iw source.ItemWatcher) error {
	if mw.source.watiSource != nil {
		initData := mw.source.watiSource.WatchInit()
		if buf, err := mw.format.MarshalInit(initData); err != nil {
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
		okay   []source.ItemWatcher
		remain = len(mw.watchers) - len(failed)
	)
	if remain > 0 {
		okay = make([]source.ItemWatcher, 0, remain)
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
		okay   []source.ItemWatcher
		remain = len(mw.watchers) - len(failed)
	)
	if remain > 0 {
		okay = make([]source.ItemWatcher, 0, remain)
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

type defaultFrameWatcher struct {
	format  source.GenericDataFormat
	writers []io.Writer
}

func (dfw *defaultFrameWatcher) writeInitData(data interface{}, w io.Writer) error {
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
	return nil
}

func (dfw *defaultFrameWatcher) HandleItem(item []byte) error {
	if len(dfw.writers) == 0 {
		return errDefaultFrameWatcherDone
	}
	buf, err := dfw.format.FrameItem(item)
	if err != nil {
		log.Printf("item framing error %v", err)
		return err
	}
	if err := dfw.writeToAll(buf); err != nil {
		return err
	}
	return nil
}

func (dfw *defaultFrameWatcher) HandleItems(items [][]byte) error {
	if len(dfw.writers) == 0 {
		return errDefaultFrameWatcherDone
	}
	for _, item := range items {
		buf, err := dfw.format.FrameItem(item)
		if err != nil {
			log.Printf("item framing error %v", err)
			return err
		}
		if err := dfw.writeToAll(buf); err != nil {
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
	return internal.MultiErr(errs).AsError()
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
