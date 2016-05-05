package resp

import (
	"fmt"
	"net"
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

// IsFirstByteRespTag returns true if the first byte in the passed slice is a
// valid RESP tag character (i.e. if it is one of "-:+$*").
func IsFirstByteRespTag(p []byte) bool {
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
