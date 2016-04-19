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

func (ib *itemBuf) close() {
	ib.Lock()
	ib.closed = true
	ib.Unlock()
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

func (ib *itemBuf) drain() [][]byte {
	ib.Lock()
	ib.takeBuf = append(ib.takeBuf[:0], ib.buffer...)
	ib.buffer = ib.buffer[:0]
	ib.Unlock()
	return ib.takeBuf
}
