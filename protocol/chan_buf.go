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
