package stacked

import (
	"bufio"
	"errors"
	"net"
	"time"
)

var errBufListenerClosed = errors.New("bufListener closed")

// bufListener implements net.Listener around a chan *bufConn.  It is useful
// for wrapping ListenServers to implement BufferdConnHandler.
type bufListener struct {
	addr   net.Addr
	inc    chan *bufConn
	closed bool
}

// Accept waits for and returns the next connection to the listener.
func (bl *bufListener) Accept() (net.Conn, error) {
	if bl.closed {
		return nil, errBufListenerClosed
	}
	if conn := <-bl.inc; conn != nil {
		return conn, nil
	}
	bl.closed = true
	return nil, errBufListenerClosed
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (bl *bufListener) Close() error {
	bl.closed = true
	// TODO: complete
	return nil
}

// Addr returns the listener's network address.
func (bl *bufListener) Addr() net.Addr {
	return bl.addr
}

// bufConn implements net.Conn around a net.Conn and a bufio.Reader by first
// draining the buffered reader, and then switching over to raw Reads on the
// connection.  All other net.Conn methods are simply passed through to the
// connection.
type bufConn struct {
	conn net.Conn
	bufr *bufio.Reader
}

// Read reads data by first draining the buffered reader, and then passes
// through to the underlying connection.
func (bufc *bufConn) Read(b []byte) (int, error) {
	if bufc.bufr == nil {
		return bufc.conn.Read(b)
	}

	n := len(b)
	if m := bufc.bufr.Buffered(); m < n {
		n = m
	}
	p, err := bufc.bufr.Peek(n)
	n = copy(b, p)
	bufc.bufr.Discard(n)
	if err == bufio.ErrBufferFull {
		err = nil
	}
	if bufc.bufr.Buffered() == 0 {
		// TODO: useful? bufc.bufr.Reset(nil)
		bufc.bufr = nil
	}
	return n, err
}

// Write writes data to the connection.
func (bufc *bufConn) Write(b []byte) (int, error) {
	return bufc.conn.Write(b)
}

// Close closes the connection.
func (bufc *bufConn) Close() error {
	return bufc.conn.Close()
}

// LocalAddr returns the local network address.
func (bufc *bufConn) LocalAddr() net.Addr {
	return bufc.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (bufc *bufConn) RemoteAddr() net.Addr {
	return bufc.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated with the
// connection.
func (bufc *bufConn) SetDeadline(t time.Time) error {
	return bufc.conn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls.
func (bufc *bufConn) SetReadDeadline(t time.Time) error {
	return bufc.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls.
func (bufc *bufConn) SetWriteDeadline(t time.Time) error {
	return bufc.conn.SetWriteDeadline(t)
}
