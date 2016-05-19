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

package resp

import (
	"fmt"
	"io"
	"strings"
)

// RedisHandler is the interface used by the connection protocol layer to
// hand-off to application specific code
type RedisHandler interface {
	HandleStart(*RedisConnection) error
	HandleEnd(*RedisConnection) error
	HandleNull(*RedisConnection) error
	HandleInteger(*RedisConnection, int) error
	HandleString(*RedisConnection, []byte) error
	HandleBulkString(*RedisConnection, int, io.Reader) error
	HandleArray(*RedisConnection, int) error
	HandleError(*RedisConnection, []byte) error
}

type ValueConsumer struct {
	rconn            *RedisConnection
	seen             int
	max              int
	missingErrFormat string
	nextVal          RedisValue
}

func NewValueConsumer(rconn *RedisConnection, max int, kind string) *ValueConsumer {
	return &ValueConsumer{
		rconn:            rconn,
		seen:             0,
		max:              max,
		missingErrFormat: strings.Join([]string{"missing %s", kind}, " "),
	}
}

func (vc *ValueConsumer) NumRemaining() int {
	if vc.seen > vc.max {
		return 0
	}
	return vc.max - vc.seen
}

func (vc *ValueConsumer) NumValues() int {
	return vc.max
}

func (vc *ValueConsumer) Consume(name string) (RedisValue, error) {
	if vc.seen >= vc.max {
		return NilRedisValue, fmt.Errorf(vc.missingErrFormat, name)
	}
	if err := vc.rconn.Consume(vc); err != nil {
		return vc.nextVal, err
	}
	vc.seen += 1
	return vc.nextVal, nil
}

func (vc *ValueConsumer) HandleNull(_ *RedisConnection) error {
	vc.nextVal = NilRedisValue
	return nil
}

func (vc *ValueConsumer) HandleInteger(_ *RedisConnection, num int) error {
	vc.nextVal = NewIntRedisValue(num)
	return nil
}

func (vc *ValueConsumer) HandleString(_ *RedisConnection, buf []byte) error {
	vc.nextVal = NewBytesRedisValue(buf)
	return nil
}

func (vc *ValueConsumer) HandleBulkString(_ *RedisConnection, n int, r io.Reader) error {
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	vc.nextVal = NewBytesRedisValue(buf)
	return nil
}

func (vc *ValueConsumer) HandleArray(_ *RedisConnection, _ int) error {
	return fmt.Errorf("unexpected RESP array, expected a string or integer")
}

func (vc *ValueConsumer) HandleError(_ *RedisConnection, _ []byte) error {
	return fmt.Errorf("unexpected RESP error, expected a string or integer")
}

func (vc *ValueConsumer) HandleStart(_ *RedisConnection) error {
	return nil
}

func (vc *ValueConsumer) HandleEnd(_ *RedisConnection) error {
	return nil
}

// CmdHandler implements simple command-dispatch:
// - only accepts arrays
// - those arrays must have one or more elements
// - the first element must be a string
type CmdHandler func(*RedisConnection, []byte, *ValueConsumer) error

func (_ CmdHandler) HandleNull(*RedisConnection) error {
	return fmt.Errorf("unexpected RESP null, expected an array")
}

func (_ CmdHandler) HandleInteger(*RedisConnection, int) error {
	return fmt.Errorf("unexpected RESP integer, expected an array")
}

func (_ CmdHandler) HandleString(*RedisConnection, []byte) error {
	return fmt.Errorf("unexpected RESP string, expected an array")
}

func (_ CmdHandler) HandleBulkString(*RedisConnection, int, io.Reader) error {
	return fmt.Errorf("unexpected RESP bulk string, expected an array")
}

func (f CmdHandler) HandleArray(rconn *RedisConnection, n int) error {
	vc := NewValueConsumer(rconn, n, "argument")

	cmdVal, err := vc.Consume("command")
	if err != nil {
		return err
	}
	cmd, ok := cmdVal.GetBytes()
	if !ok {
		return fmt.Errorf("expected command string")
	}

	if err := f(rconn, cmd, vc); err != nil {
		return err
	}

	if vc.NumRemaining() > 0 {
		return fmt.Errorf("too many arguments to %v command", string(cmd))
	}

	return nil
}

func (_ CmdHandler) HandleError(*RedisConnection, []byte) error {
	return fmt.Errorf("unexpected RESP error, expected an array")
}

func (f CmdHandler) HandleStart(rconn *RedisConnection) error {
	return f(rconn, []byte("__start__"), nil)
}

func (f CmdHandler) HandleEnd(rconn *RedisConnection) error {
	return f(rconn, []byte("__end__"), nil)
}

type CmdFunc func(*RedisConnection, *ValueConsumer) error

type cmdMap map[string]CmdFunc

func (cmds cmdMap) handleCommand(rconn *RedisConnection, cmdBuf []byte, vc *ValueConsumer) error {
	cmd := string(cmdBuf)
	cmdFunc, ok := cmds[cmd]
	if !ok && !strings.HasPrefix(cmd, "_") {
		return rconn.WriteError(fmt.Errorf("unimplemented command %#v", cmd))
	}
	if cmdFunc != nil {
		return cmdFunc(rconn, vc)
	}
	return nil
}

func CmdMapHandler(cmds map[string]CmdFunc) CmdHandler {
	return CmdHandler(cmdMap(cmds).handleCommand)
}
