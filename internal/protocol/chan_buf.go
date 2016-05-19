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
	"bytes"
	"errors"
	"io"
	"sync"
)

var errBufClosed = errors.New("buffer closed")

type chanBuf struct {
	sync.Mutex
	bytes.Buffer
	ready   chan<- *chanBuf
	closed  bool
	pending bool
	p       []byte
	// TODO: limit
}

func (cb *chanBuf) Reset() {
	cb.pending = false
	cb.Buffer.Reset()
}

func (cb *chanBuf) Write(p []byte) (int, error) {
	var send bool
	cb.Lock()

	if cb.closed {
		cb.Unlock()
		return 0, errBufClosed
	}

	n, err := cb.Buffer.Write(p)
	if n > 0 && !cb.pending {
		cb.pending = true
		send = true
	}
	cb.Unlock()

	if send {
		cb.ready <- cb
	}
	return n, err
}

func (cb *chanBuf) Close() error {
	if !cb.closed {
		cb.Lock()
		cb.closed = true
		cb.Unlock()
	}
	return nil
}

func (cb *chanBuf) writeTo(w io.Writer) (int, error) {
	return w.Write(cb.drain())
}

func (cb *chanBuf) drain() []byte {
	cb.Lock()
	if cap(cb.p) < cb.Len() {
		cb.p = make([]byte, cb.Len())
	}

	n := copy(cb.p[:cap(cb.p)], cb.Bytes())
	cb.p = cb.p[:n]
	cb.Reset()
	cb.Unlock()
	return cb.p
}
