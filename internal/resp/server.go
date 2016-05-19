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
