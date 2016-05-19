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

package protocol

import (
	"errors"
	"sync"
)

var errItemBufClosed = errors.New("item buffer closed")

type itemBuf struct {
	sync.Mutex
	ready   chan<- *itemBuf
	closed  bool
	pending bool
	buffer  [][]byte
	takeBuf [][]byte
	// TODO: limit
}

func newItemBuf(ready chan<- *itemBuf) *itemBuf {
	return &itemBuf{
		ready: ready,
	}
}

func (ib *itemBuf) put(items ...[]byte) (int, error) {
	if ib.closed {
		return 0, errItemBufClosed
	}
	ib.buffer = append(ib.buffer, items...)
	return len(items), nil
}

func (ib *itemBuf) HandleItem(item []byte) error {
	ib.Lock()
	n, err := ib.put(item)
	ib.Unlock()
	if n > 0 {
		ib.ready <- ib
	}
	return err
}

func (ib *itemBuf) HandleItems(items [][]byte) error {
	ib.Lock()
	n, err := ib.put(items...)
	ib.Unlock()
	if n > 0 {
		ib.ready <- ib
	}
	return err
}

func (ib *itemBuf) Close() error {
	if !ib.closed {
		ib.Lock()
		ib.closed = true
		ib.Unlock()
	}
	return nil
}

func (ib *itemBuf) drain() [][]byte {
	ib.Lock()
	ib.takeBuf = append(ib.takeBuf[:0], ib.buffer...)
	ib.buffer = ib.buffer[:0]
	ib.Unlock()
	return ib.takeBuf
}
