package resp

import (
	"bufio"
	"fmt"
	"net"
	"net/http"

	"github.com/uber-go/gwr/internal/stacked"
)

// RedisServer serves a RedisHandler on a listening socket.
type RedisServer struct {
	consumer RedisHandler
}

// NewRedisServer creates a new RedisServer.
func NewRedisServer(consumer RedisHandler) *RedisServer {
	return &RedisServer{
		consumer: consumer,
	}
}

// ListenAndServe listens on the given hostPort, and then serves on that
// listener.
func (h RedisServer) ListenAndServe(hostPort string) error {
	ln, err := net.Listen("tcp", hostPort)
	if err != nil {
		return err
	}
	return h.Serve(ln)
}

// Serve serves the given listener.
func (h RedisServer) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			// TODO: deal better with accept errors
			fmt.Printf("ERROR: accept error: %v", err)
			continue
		}
		go NewRedisConnection(conn, nil).Handle(h.consumer)
	}
	return nil
}

// WrapHTTPHandler bundles together a RedisHandler and an http.Handler into a
// dual protocol server.  The server detects the RESP protocol based on the
// first byte (if it is one of "-:+$*").
func WrapHTTPHandler(respHandler RedisHandler, httpHandler http.Handler) *stacked.Server {
	return &stacked.Server{[]stacked.Detector{
		respDetector(respHandler),
		stacked.DefaultHTTPHandler(httpHandler),
	}}
}

func isFirstByteRespTag(p []byte) bool {
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
		return true
	default:
		return false
	}
}

func respDetector(respHandler RedisHandler) stacked.Detector {
	hndl := stacked.ConnHandlerFunc(func(conn net.Conn, bufr *bufio.Reader) {
		NewRedisConnection(conn, bufr).Handle(respHandler)
	})
	return stacked.Detector{
		Needed:  1,
		Test:    isFirstByteRespTag,
		Handler: hndl,
	}
}
