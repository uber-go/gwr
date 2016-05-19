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
	"bufio"
	"fmt"
	"io"
	"net"
)

// RedisConnection is the protocol reading and writing layer
type RedisConnection struct {
	Conn   net.Conn
	reader *bufio.Reader
	// TODO: use a bufio.Writer too
}

// NewRedisConnection creates a redis connection around an existing net.Conn
// and an optional reader (e.g. a bufio.Reader that's already been created
// around the net.Conn).  If r is nill, conn is used instead.
func NewRedisConnection(conn net.Conn, r io.Reader) *RedisConnection {
	if r == nil {
		r = conn
	}
	return &RedisConnection{
		Conn:   conn,
		reader: bufio.NewReader(r),
	}
}

// Close closes the underlying connection.
func (rconn *RedisConnection) Close() error {
	return rconn.Conn.Close()
}

// Handle runs the passed handler until the connection ends or errors.
func (rconn *RedisConnection) Handle(handler RedisHandler) {
	if err := rconn.handle(handler); err != nil {
		fmt.Printf("redis handler error from %v: %v\n", rconn.Conn.RemoteAddr(), err)
	}
	if err := rconn.Close(); err != nil {
		fmt.Printf("error closing connection from %v: %v\n", rconn.Conn.RemoteAddr(), err)
	}
}

func (rconn *RedisConnection) handle(handler RedisHandler) error {
	if err := handler.HandleStart(rconn); err != nil {
		return err
	}

	for {
		err := rconn.Consume(handler)
		if err != nil {
			if err != io.EOF {
				rconn.WriteError(err)
			}
			break
		}
	}

	return handler.HandleEnd(rconn)
}

// Consume reads one element from the connection and passes it to the given handler.
func (rconn *RedisConnection) Consume(handler RedisHandler) error {
	c, err := rconn.reader.ReadByte()
	if err != nil {
		return err
	}

	switch c {
	case '-':
		return rconn.consumeError(handler)

	case ':':
		return rconn.consumeInteger(handler)

	case '+':
		return rconn.consumeShortString(handler)

	case '$':
		return rconn.consumeBulkString(handler)

	case '*':
		return rconn.consumeArray(handler)

	default:
		return fmt.Errorf("unknown RESP type %#v", string(c))
	}
}

func (rconn *RedisConnection) consumeError(handler RedisHandler) error {
	buf, err := rconn.scanLine()
	if err != nil {
		return err
	}
	return handler.HandleError(rconn, buf)
}

func (rconn *RedisConnection) consumeInteger(handler RedisHandler) error {
	n, err := rconn.readInteger()
	if err != nil {
		return err
	}
	return handler.HandleInteger(rconn, n)
}

func (rconn *RedisConnection) consumeShortString(handler RedisHandler) error {
	buf, err := rconn.scanLine()
	if err != nil {
		return err
	}
	return handler.HandleString(rconn, buf)
}

func (rconn *RedisConnection) consumeBulkString(handler RedisHandler) error {
	n, err := rconn.readInteger()
	if err != nil {
		return err
	}

	if n < 0 {
		return handler.HandleNull(rconn)
	}

	strReader := io.LimitReader(rconn.reader, int64(n))
	if err := handler.HandleBulkString(rconn, n, strReader); err != nil {
		return err
	}

	c, err := rconn.reader.ReadByte()
	if err != nil {
		return err
	}
	if c != '\r' {
		return fmt.Errorf("missing CR")
	}

	c, err = rconn.reader.ReadByte()
	if err != nil {
		return err
	}
	if c != '\n' {
		return fmt.Errorf("missing LF after CR")
	}

	return nil
}

func (rconn *RedisConnection) consumeArray(handler RedisHandler) error {
	n, err := rconn.readInteger()
	if err != nil {
		return err
	}

	if n < 0 {
		return handler.HandleNull(rconn)
	}

	return handler.HandleArray(rconn, n)
}

func (rconn *RedisConnection) readUInteger() (uint, error) {
	n, err := rconn.scanNumbers(0)
	if err != nil {
		return n, err
	}

	c, err := rconn.reader.ReadByte()
	if err != nil {
		return n, err
	}
	if c != '\n' {
		return n, fmt.Errorf("missing LF after CR")
	}

	return n, nil
}

func (rconn *RedisConnection) readInteger() (int, error) {
	n := 0

	c, err := rconn.reader.ReadByte()
	if err != nil {
		return n, err
	}

	if c == '-' {
		nu, err := rconn.scanNumbers(0)
		n = -int(nu)
		if err != nil {
			return n, err
		}
	} else if c != '\r' {
		// if c > '9' {
		// 	return n, fmt.Errorf("unexpected byte %v while scanning integer, expected [0-9]", c)
		// }
		nu, err := rconn.scanNumbers(uint(c - '0'))
		n = int(nu)
		if err != nil {
			return n, err
		}
	}

	c, err = rconn.reader.ReadByte()
	if err != nil {
		return n, err
	}
	if c != '\n' {
		return n, fmt.Errorf("missing LF after CR")
	}

	return n, nil
}

func (rconn *RedisConnection) scanNumbers(n uint) (uint, error) {
	for {
		c, err := rconn.reader.ReadByte()
		if err != nil {
			return n, err
		}
		if c == '\r' {
			break
		}
		// if c > '9' {
		// 	return n, fmt.Errorf("unexpected byte %v while scanning integer, expected [0-9]", c)
		// }

		n = 10*n + uint(c-'0')
	}

	return n, nil
}

func (rconn *RedisConnection) scanLine() ([]byte, error) {
	buf, err := rconn.reader.ReadBytes('\r')
	if err != nil {
		return buf, err
	}

	c, err := rconn.reader.ReadByte()
	if err != nil {
		return buf, err
	}
	if c != '\n' {
		return buf, fmt.Errorf("missing LF after CR")
	}

	return buf, nil
}

// WriteArrayHeader writes a "*N\r\n" array header.
func (rconn *RedisConnection) WriteArrayHeader(num int) error {
	return rconn.writef("*%v\r\n", num)
}

// WriteInteger writes a ":N\r\n" integer literal
func (rconn *RedisConnection) WriteInteger(num int) error {
	return rconn.writef(":%v\r\n", num)
}

// WriteNull writes a "$-1\r\n" null string
func (rconn *RedisConnection) WriteNull() error {
	return rconn.write([]byte("$-1\r\n"))
}

// WriteNullArray writes a "*-1\r\n" null array
func (rconn *RedisConnection) WriteNullArray() error {
	return rconn.write([]byte("*-1\r\n"))
}

// WriteBulkBytes writes "$N\r\n...\r\n" bulk string from a byte slice.
func (rconn *RedisConnection) WriteBulkBytes(buf []byte) error {
	n := len(buf)
	if n == 0 {
		return rconn.write([]byte("$0\r\n\r\n"))
	}

	if err := rconn.writef("$%v\r\n", n); err != nil {
		return err
	}

	if _, err := rconn.Conn.Write(buf); err != nil {
		return err
	}

	return rconn.write([]byte("\r\n"))
}

// WriteBulkStringHeader writes a "$N\r\n" bulk string header.
func (rconn *RedisConnection) WriteBulkStringHeader(n int) error {
	return rconn.writef("$%v\r\n", n)
}

// WriteBulkStringFooter writes a "\r\n" bulk string footer.
func (rconn *RedisConnection) WriteBulkStringFooter() error {
	return rconn.write([]byte("\r\n"))
}

// WriteBulkString writes a "$N\r\n...\r\n" bulk string.
func (rconn *RedisConnection) WriteBulkString(str string) error {
	n := len(str)
	if n == 0 {
		return rconn.write([]byte("$0\r\n\r\n"))
	}
	return rconn.writef("$%v\r\n%v\r\n", n, str)
}

// WriteSimpleString writes a "+...\r\n" simple string.
func (rconn *RedisConnection) WriteSimpleString(str string) error {
	return rconn.writef("+%v\r\n", str)
}

// WriteSimpleBytes writes a "+...\r\n" simple string froma byte slice.
func (rconn *RedisConnection) WriteSimpleBytes(b []byte) error {
	return rconn.writef("+%s\r\n", b)
}

// WriteError writes a "-ERR...\r\n" error.
func (rconn *RedisConnection) WriteError(err error) error {
	return rconn.writef("-ERR %v\r\n", err)
}

// WriteErrorBytes writes a "-...\r\n" error from a byte slice.
func (rconn *RedisConnection) WriteErrorBytes(b []byte) error {
	if _, err := rconn.Conn.Write([]byte("-")); err != nil {
		return err
	}
	if _, err := rconn.Conn.Write(b); err != nil {
		return err
	}
	if _, err := rconn.Conn.Write([]byte("\r\n")); err != nil {
		return err
	}
	return nil
}

// WriteErrorString writes a "-TYPE ...\r\n" error from a string type and body.
func (rconn *RedisConnection) WriteErrorString(errType, str string) error {
	return rconn.writef("-%v %v\r\n", errType, str)
}

func (rconn *RedisConnection) writef(format string, a ...interface{}) error {
	_, err := fmt.Fprintf(rconn.Conn, format, a...)
	return err
}

func (rconn *RedisConnection) write(buf []byte) error {
	if _, err := rconn.Conn.Write(buf); err != nil {
		return err
	}
	return nil
}
