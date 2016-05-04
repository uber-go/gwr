package stacked

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"time"
)

// ListenServer is the minimal downstream server interface, e.g. implemented by
// http.Server.
type ListenServer interface {
	Serve(ln net.Listener) error
}

// ConnHandler is a higher level interface implemented by handlers that can
// re-use the buffered reader.
type ConnHandler interface {
	ServeConnection(conn net.Conn, bufr *bufio.Reader)
	// TODO: maybe shift to a struct{net.Conn, bufio.Reader}
}

// ConnHandlerFunc is a convenience type for implementing simple ConnHandlers.
type ConnHandlerFunc func(conn net.Conn, bufr *bufio.Reader)

// ServeConnection simply calls the function
func (bchf ConnHandlerFunc) ServeConnection(conn net.Conn, bufr *bufio.Reader) {
	bchf(conn, bufr)
}

// ListenServerHandler creates a compatability ConnHandler for
func ListenServerHandler(srv ListenServer) ConnHandler {
	bc := make(chan *bufConn)
	cbs := &connBufShim{Server: srv, bufConns: bc}
	cbs.bufLn.inc = bc
	return cbs
}

type connBufShim struct {
	Server   ListenServer
	bufLn    bufListener
	bufConns chan<- *bufConn
}

// ServeConnection simply puts a new bufConn onto bufConns for distribution by
// bufLn.Accept.
func (cbs *connBufShim) ServeConnection(conn net.Conn, bufr *bufio.Reader) {
	cbs.bufConns <- &bufConn{conn, bufr}
}

func (cbs *connBufShim) setAddr(addr net.Addr) {
	cbs.bufLn.addr = addr
}

func (cbs *connBufShim) close() {
	close(cbs.bufConns) // TODO: may panic with a ServeConnection that's trying to send
	cbs.bufLn.Close()
}

// Detector is a ConnHandler along with a detection function and how
// many bytes it needs to decide.
type Detector struct {
	// Needed is how many bytes are needed for the Test function
	Needed int

	// Test will be called with at least Needed bytes.  If this function
	// returns true, then the connection will be given to Handler.
	Test func(b []byte) bool

	Handler ConnHandler
}

// DefaultHTTPHandler creates a FallthroughDetector around an http.Handler.
func DefaultHTTPHandler(hndl http.Handler) Detector {
	handler := ListenServerHandler(&http.Server{
		Handler: hndl,
	})
	return FallthroughDetector(handler)
}

// FallthroughDetector returns a Detector whose Test function always returns
// true.  No bytes are needed for tautology.
func FallthroughDetector(hndl ConnHandler) Detector {
	return Detector{
		Needed:  0,
		Test:    func([]byte) bool { return true },
		Handler: hndl,
	}
}

// Server serves one or more Detectors.  The first one whose Test function
// returns true wins.
type Server struct {
	Detectors []Detector
}

// ListenAndServe opens a listening TCP socket, and calls Serve on it.
func (srv *Server) ListenAndServe(hostPort string) error {
	ln, err := net.Listen("tcp", hostPort)
	if err != nil {
		return err
	}
	return srv.Serve(ln)
}

// Serve runs a handling loop on a listening socket.
func (srv *Server) Serve(ln net.Listener) error {
	srv.startDetectors(ln.Addr())
	defer srv.closeDetectors()

	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Printf("stacked: Accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = 0

		go srv.handleConnection(conn)
	}
	return nil
}

func (srv *Server) startDetectors(addr net.Addr) {
	for _, det := range srv.Detectors {
		// TODO: is this a more useful interface?
		if cbs, ok := det.Handler.(*connBufShim); ok {
			cbs.setAddr(addr)
			// TODO: this requires that no cbs be shared between two srv.Serve
			// calls... which seems natural for simple uasge, but may be too
			// restrictive
			go cbs.Server.Serve(&cbs.bufLn)
			// TODO: handle any Serve error? wrap that in cbs.XXX?
		}
	}
}

func (srv *Server) closeDetectors() {
	for _, det := range srv.Detectors {
		// TODO: is this a more useful interface?
		if cbs, ok := det.Handler.(*connBufShim); ok {
			cbs.close()
		}
	}
}

func (srv *Server) handleConnection(conn net.Conn) {
	// TODO: suspect could do better in slow case where we don't have any
	// initial bytes yet... bufr doesn't seem to have a mechanism to wait for X
	// bytes to be available, that then lets us give them all back
	size := 512
	for _, det := range srv.Detectors {
		if det.Needed > size {
			size = det.Needed
		}
	}
	bufr := bufio.NewReaderSize(conn, size)
	i := 0
	for k := 0; k < 10; k++ {
		for ; i < len(srv.Detectors); i++ {
			det := srv.Detectors[i]
			if b, _ := bufr.Peek(det.Needed); len(b) < det.Needed {
				break
			} else if det.Test(b) {
				det.Handler.ServeConnection(conn, bufr)
				return
			}
		}
	}
	log.Printf("stacked: no detector wanted the connection")
	conn.Close()
}
