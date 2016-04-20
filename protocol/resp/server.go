package resp

import (
	"fmt"
	"net"
)

type RedisServer struct {
	consumer RedisHandler
}

func NewRedisServer(consumer RedisHandler) *RedisServer {
	return &RedisServer{
		consumer: consumer,
	}
}

func (h RedisServer) ListenAndServe(hostPort string) error {
	ln, err := net.Listen("tcp", hostPort)
	if err != nil {
		return err
	}
	return h.Serve(ln)
}

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
