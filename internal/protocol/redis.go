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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/uber-go/gwr/internal/resp"
	"github.com/uber-go/gwr/source"
)

// NewRedisServer creates a new redis server to provide access to a collection
// of gwr data sources.
func NewRedisServer(sources *source.DataSources) *resp.RedisServer {
	handler := NewRedisHandler(sources)
	return resp.NewRedisServer(handler)
}

// NewRedisHandler creates a new redis handler for a given collection of gwr
// data sources for use with the resp package.
func NewRedisHandler(sources *source.DataSources) resp.RedisHandler {
	model := respModel{
		sources:  sources,
		sessions: make(map[*resp.RedisConnection]*respSession, 1),
	}
	return resp.CmdMapHandler(map[string]resp.CmdFunc{
		"ls":      model.handleLs,
		"get":     model.handleGet,
		"watch":   model.handleWatch,
		"monitor": model.handleMonitor,
		"__end__": model.handleEnd,
	})
}

type respModel struct {
	sources  *source.DataSources
	sessions map[*resp.RedisConnection]*respSession
}

type respSession struct {
	watches     map[string]string
	stopMonitor chan struct{}
}

func (rm *respModel) session(rconn *resp.RedisConnection) *respSession {
	if session, ok := rm.sessions[rconn]; ok {
		return session
	}
	session := &respSession{
		watches:     make(map[string]string, 1),
		stopMonitor: make(chan struct{}, 1),
	}
	rm.sessions[rconn] = session
	return session
}

func (rm *respModel) handleLs(rconn *resp.RedisConnection, vc *resp.ValueConsumer) error {
	// TODO: implement optional path argument
	// TODO: maybe custom format

	if vc.NumRemaining() > 0 {
		return fmt.Errorf("too many arguments to ls")
	}

	return rm.doGet(rconn, rm.sources.Get("/meta/nouns"), "text")
}

func (rm *respModel) handleGet(rconn *resp.RedisConnection, vc *resp.ValueConsumer) error {
	source, err := rm.consumeSource(rconn, vc)
	if err != nil {
		return err
	}

	format, err := rm.consumeFormat(rconn, vc)
	if err != nil {
		return err
	}

	if vc.NumRemaining() > 0 {
		return fmt.Errorf("too many arguments to get")
	}

	return rm.doGet(rconn, source, format)
}

func (rm *respModel) doGet(rconn *resp.RedisConnection, source source.DataSource, format string) error {
	var buf bytes.Buffer
	if err := source.Get(format, &buf); err != nil {
		return err
	}

	switch format {
	case "text":
		lines := strings.Split(buf.String(), "\n")
		if i := len(lines) - 1; len(lines[i]) == 0 {
			lines = lines[:i]
		}
		if err := rconn.WriteArrayHeader(len(lines)); err != nil {
			return err
		}
		for _, line := range lines {
			if err := rconn.WriteSimpleString(line); err != nil {
				return err
			}
		}
	default:
		return rconn.WriteBulkBytes(buf.Bytes())
	}

	return nil
}

func (rm *respModel) handleWatch(rconn *resp.RedisConnection, vc *resp.ValueConsumer) error {
	session := rm.session(rconn)

	source, err := rm.consumeSource(rconn, vc)
	if err != nil {
		return err
	}

	format, err := rm.consumeFormat(rconn, vc)
	if err != nil {
		return err
	}

	if vc.NumRemaining() > 0 {
		return fmt.Errorf("too many arguments to watch")
	}

	name := source.Name()
	session.watches[name] = format

	return rconn.WriteSimpleString("OK")
}

func (rm *respModel) handleMonitor(rconn *resp.RedisConnection, vc *resp.ValueConsumer) error {
	session := rm.session(rconn)

	for vc.NumRemaining() > 0 {
		source, err := rm.consumeSource(rconn, vc)
		if err != nil {
			return err
		}

		format, err := rm.consumeFormat(rconn, vc)
		if err != nil {
			return err
		}

		name := source.Name()
		session.watches[name] = format
	}

	if len(session.watches) == 0 {
		return fmt.Errorf("no watches set, monitor likely to be uninteresting")
	}

	go rm.doWatch(rconn)

	return nil
}

func (rm *respModel) doWatch(rconn *resp.RedisConnection) error {
	type bufInfoEntry struct {
		name, format string
	}

	session := rm.session(rconn)
	bufs := make([]*chanBuf, 0, len(session.watches))
	itemBufs := make([]*itemBuf, 0, len(session.watches))
	bufInfo := make(map[*chanBuf]bufInfoEntry, len(session.watches))
	itemBufInfo := make(map[*itemBuf]bufInfoEntry, len(session.watches))
	bufReady := make(chan *chanBuf, len(session.watches))
	itemBufReady := make(chan *itemBuf, len(session.watches))
	defer func() {
		for _, buf := range bufs {
			buf.Close()
		}
		for _, itemBuf := range itemBufs {
			itemBuf.Close()
		}
	}()

	for name, format := range session.watches {
		src := rm.sources.Get(name)
		if src == nil {
			continue
		}
		if itemSource, ok := src.(source.ItemDataSource); ok {
			itemBuf := newItemBuf(itemBufReady)
			itemBufs = append(itemBufs, itemBuf)
			itemBufInfo[itemBuf] = bufInfoEntry{
				name:   name,
				format: strings.ToLower(format),
			}
			itemSource.WatchItems(format, itemBuf)
		} else {
			buf := &chanBuf{ready: bufReady}
			bufs = append(bufs, buf)
			bufInfo[buf] = bufInfoEntry{
				name:   name,
				format: strings.ToLower(format),
			}
			src.Watch(format, buf)
		}
	}

	var write func(*resp.RedisConnection, *chanBuf, string, string) error
	var writeItems func(*resp.RedisConnection, *itemBuf, string, string) error

	if len(session.watches) == 1 {
		write = rm.writeSingleWatchData
		writeItems = rm.writeSingleWatchItem
	} else {
		write = rm.writeMultiWatchData
		writeItems = rm.writeMultiWatchItem
	}

	for {
		select {
		case <-session.stopMonitor:
			return nil
		case buf := <-bufReady:
			info := bufInfo[buf]
			if err := write(rconn, buf, info.name, info.format); err != nil {
				return err
			}
		case itemBuf := <-itemBufReady:
			info := itemBufInfo[itemBuf]
			if err := writeItems(rconn, itemBuf, info.name, info.format); err != nil {
				return err
			}
		}
	}
}

type multiJSONMessage struct {
	Name string           `json:"name"`
	Data *json.RawMessage `json:"data"`
}

func (rm *respModel) writeSingleWatchItem(rconn *resp.RedisConnection, itemBuf *itemBuf, name, format string) error {
	switch format {
	case "text":
		for _, line := range itemBuf.drain() {
			// TODO: still need to split and send individual lines?
			if err := rconn.WriteSimpleBytes(line); err != nil {
				return err
			}
		}

	default:
		for _, buf := range itemBuf.drain() {
			if err := rconn.WriteBulkBytes(buf); err != nil {
				return err
			}
		}
	}
	return nil
}

func (rm *respModel) writeMultiWatchItem(rconn *resp.RedisConnection, itemBuf *itemBuf, name, format string) error {
	switch format {
	case "text":
		for _, buf := range itemBuf.drain() {
			// TODO: still need to split and send individual lines?
			line := fmt.Sprintf("%s> %s", name, buf)
			if err := rconn.WriteSimpleString(line); err != nil {
				return err
			}
		}

	case "json":
		for _, buf := range itemBuf.drain() {
			if buf, err := json.Marshal(multiJSONMessage{
				Name: name,
				Data: (*json.RawMessage)(&buf),
			}); err != nil {
				return err
			} else if err := rconn.WriteBulkBytes(buf); err != nil {
				return err
			}
		}

	default:
		for _, buf := range itemBuf.drain() {
			if err := rconn.WriteArrayHeader(2); err != nil {
				return err
			}
			if err := rconn.WriteSimpleString(name); err != nil {
				return err
			}
			if err := rconn.WriteBulkBytes(buf); err != nil {
				return err
			}
		}
	}
	return nil
}

// TODO: can we re-use code b/w *WatchData and *WatchItems?

func (rm *respModel) writeSingleWatchData(rconn *resp.RedisConnection, buf *chanBuf, name, format string) error {
	switch format {
	case "text":
		buf.Lock()
		for {
			line, doneErr := buf.ReadString('\n')
			if doneErr == nil {
				line = line[:len(line)-1]
			}
			if len(line) > 0 || doneErr == nil {
				if err := rconn.WriteSimpleString(line); err != nil {
					return err
				}
			}
			if doneErr != nil {
				break
			}
		}
		buf.Reset()
		buf.Unlock()

	case "json":
		buf.Lock()
		for {
			line, doneErr := buf.ReadString('\n')
			if len(line) > 0 {
				line = line[:len(line)-1]
			}
			if len(line) > 0 {
				if err := rconn.WriteBulkString(line); err != nil {
					return err
				}
			}
			if doneErr != nil {
				break
			}
		}
		buf.Reset()
		buf.Unlock()

	default:
		b := buf.drain()
		if err := rconn.WriteBulkBytes(b); err != nil {
			return err
		}
	}

	return nil
}

func (rm *respModel) writeMultiWatchData(rconn *resp.RedisConnection, buf *chanBuf, name, format string) error {
	switch format {
	case "text":
		buf.Lock()
		for {
			line, doneErr := buf.ReadString('\n')
			if doneErr == nil {
				line = line[:len(line)-1]
			}
			if len(line) > 0 || doneErr == nil {
				line = fmt.Sprintf("%s> %s", name, line)
				if err := rconn.WriteSimpleString(line); err != nil {
					return err
				}
			}
			if doneErr != nil {
				break
			}
		}
		buf.Reset()
		buf.Unlock()

	case "json":
		buf.Lock()
		for {
			line, doneErr := buf.ReadString('\n')
			if doneErr == nil {
				line = line[:len(line)-1]
			}
			if len(line) > 0 {
				data := []byte(line)
				if buf, err := json.Marshal(multiJSONMessage{
					Name: name,
					Data: (*json.RawMessage)(&data),
				}); err != nil {
					return err
				} else if err := rconn.WriteBulkBytes(buf); err != nil {
					return err
				}
			}
			if doneErr != nil {
				break
			}
		}
		buf.Reset()
		buf.Unlock()

	default:
		b := buf.drain()
		if err := rconn.WriteArrayHeader(2); err != nil {
			return err
		}
		if err := rconn.WriteSimpleString(name); err != nil {
			return err
		}
		if err := rconn.WriteBulkBytes(b); err != nil {
			return err
		}
	}

	return nil
}

func (rm *respModel) handleEnd(rconn *resp.RedisConnection, vc *resp.ValueConsumer) error {
	session, ok := rm.sessions[rconn]
	if !ok {
		return nil
	}

	session.stopMonitor <- struct{}{}

	delete(rm.sessions, rconn)
	return nil
}

func (rm *respModel) consumeSource(rconn *resp.RedisConnection, vc *resp.ValueConsumer) (source.DataSource, error) {
	nameRV, err := vc.Consume("name")
	if err != nil {
		return nil, err
	}
	name, ok := nameRV.GetString()
	if !ok {
		return nil, fmt.Errorf("name argument not a string")
	}
	source := rm.sources.Get(name)
	if source == nil {
		return nil, fmt.Errorf("no such data source")
	}
	return source, nil
}

func (rm *respModel) consumeFormat(rconn *resp.RedisConnection, vc *resp.ValueConsumer) (string, error) {
	if vc.NumRemaining() == 0 {
		return "text", nil // XXX default
	}
	rv, err := vc.Consume("format")
	if err != nil {
		return "", err
	}
	format, ok := rv.GetString()
	if !ok {
		return "", fmt.Errorf("format argument not a string")
	}
	return format, nil
}
