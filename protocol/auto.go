package protocol

import (
	"bufio"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/uber-go/gwr"
	"github.com/uber-go/gwr/protocol/resp"
)

// AutoServer hosts both HTTP and RESP protocol.
type AutoServer struct {
	httpHandler http.Handler
	respHandler resp.RedisHandler
	httpConns   chan<- *bufConn
}

// NewAutoServer creates a new auto server around a redis handler and FIXME HOW HTTP.
func NewAutoServer(dss *gwr.DataSources) *AutoServer {
	return &AutoServer{
		httpHandler: NewHTTPRest(dss, ""),
		respHandler: NewRedisHandler(dss),
	}
}

// ListenAndServe opens a listening TCP socket, and calls Serve on it.
func (h AutoServer) ListenAndServe(hostPort string) error {
	ln, err := net.Listen("tcp", hostPort)
	if err != nil {
		return err
	}
	return h.Serve(ln)
}

// Serve runs a handling loop on a listening socket.
func (h AutoServer) Serve(ln net.Listener) error {
	httpConns := make(chan *bufConn)
	bufLn := bufListener{
		addr: ln.Addr(),
		inc:  httpConns,
	}
	srv := http.Server{
		Handler: h.httpHandler,
	}
	h.httpConns = httpConns
	go srv.Serve(&bufLn)
	for {
		conn, err := ln.Accept()
		if err != nil {
			// TODO: deal better with accept errors
			log.Printf("ERROR: accept error: %v", err)
			continue
		}
		go h.handleConnection(conn)
	}
	close(h.httpConns) // TODO: maybe panic with a handleConnection that's trying to send
	bufLn.Close()
	return nil
}

// handleConnection handles an incoming connection by peeking at the first byte
// to decide if its RESP or HTTP protocol.
func (h AutoServer) handleConnection(conn net.Conn) {
	bufr := bufio.NewReader(conn)
	p, err := bufr.Peek(1)
	if err != nil {
		log.Print(err)
		return
	}

	switch p[0] {
	case '-':
		fallthrough
	case ':':
		fallthrough
	case '+':
		fallthrough
	case '$':
		fallthrough
	case '*':
		resp.NewRedisConnection(conn, bufr).Handle(h.respHandler)

	default:
		h.httpConns <- &bufConn{conn, bufr}
	}
}

var errBufListenerClosed = errors.New("bufListener closed")

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
