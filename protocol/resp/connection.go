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
}

func NewRedisConnection(conn net.Conn) *RedisConnection {
	return &RedisConnection{
		Conn:   conn,
		reader: bufio.NewReader(conn),
	}
}

func (rconn *RedisConnection) Close() error {
	return rconn.Conn.Close()
}

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

	return nil
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

func (rconn *RedisConnection) WriteArrayHeader(num int) error {
	return rconn.writef("*%v\r\n", num)
}

func (rconn *RedisConnection) WriteInteger(num int) error {
	return rconn.writef(":%v\r\n", num)
}

func (rconn *RedisConnection) WriteNull() error {
	return rconn.write([]byte("$-1\r\n"))
}

func (rconn *RedisConnection) WriteNullArray() error {
	return rconn.write([]byte("*-1\r\n"))
}

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

func (rconn *RedisConnection) WriteBulkStringHeader(n int) error {
	return rconn.writef("$%v\r\n", n)
}

func (rconn *RedisConnection) WriteBulkStringFooter() error {
	return rconn.write([]byte("\r\n"))
}

func (rconn *RedisConnection) WriteBulkString(str string) error {
	n := len(str)
	if n == 0 {
		return rconn.write([]byte("$0\r\n\r\n"))
	}
	return rconn.writef("$%v\r\n%v\r\n", n, str)
}

func (rconn *RedisConnection) WriteSimpleString(str string) error {
	return rconn.writef("+%v\r\n", str)
}

func (rconn *RedisConnection) WriteSimpleBytes(b []byte) error {
	return rconn.writef("+%s\r\n", b)
}

func (rconn *RedisConnection) WriteError(err error) error {
	return rconn.writef("-ERR %v\r\n", err)
}

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

func (rconn *RedisConnection) WriteErrorString(errType string, str string) error {
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
