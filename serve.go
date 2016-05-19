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

package gwr

import (
	"bufio"
	"errors"
	"net"
	"net/http"

	"github.com/uber-go/gwr/internal/protocol"
	"github.com/uber-go/gwr/internal/resp"
	"github.com/uber-go/gwr/source"

	"github.com/uber-common/stacked"
)

var errNoServer = errors.New("no server configured")

type indirectServer struct {
	cs **ConfiguredServer
}

func (is indirectServer) Addr() net.Addr {
	srv := *(is.cs)
	if srv == nil {
		return nil
	}
	return srv.Addr()
}

func (is indirectServer) StartOn(laddr string) error {
	srv := *(is.cs)
	if srv == nil {
		return errNoServer
	}
	return srv.StartOn(laddr)
}

func (is indirectServer) Stop() error {
	srv := *(is.cs)
	if srv == nil {
		return errNoServer
	}
	return srv.Stop()
}

func init() {
	hh := protocol.NewHTTPRest(DefaultDataSources, "/gwr", indirectServer{&theServer})
	http.Handle("/gwr/", hh)
}

// ListenAndServeResp starts a resp protocol gwr server.
func ListenAndServeResp(hostPort string, dss *source.DataSources) error {
	if dss == nil {
		dss = DefaultDataSources
	}
	return protocol.NewRedisServer(dss).ListenAndServe(hostPort)
}

// ListenAndServeHTTP starts an http protocol gwr server.
func ListenAndServeHTTP(hostPort string, dss *source.DataSources) error {
	if dss == nil {
		dss = DefaultDataSources
	}
	hh := protocol.NewHTTPRest(dss, "", indirectServer{&theServer})
	return http.ListenAndServe(hostPort, hh)
}

// NewServer creates an "auto" protocol server that will respond to HTTP or
// RESP requests.
func NewServer(dss *source.DataSources) stacked.Server {
	if dss == nil {
		dss = DefaultDataSources
	}
	hh := protocol.NewHTTPRest(dss, "", indirectServer{&theServer})
	rh := protocol.NewRedisHandler(dss)
	return stacked.NewServer(
		respDetector(rh),
		stacked.DefaultHTTPHandler(hh),
	)
}

func respDetector(respHandler resp.RedisHandler) stacked.Detector {
	hndl := stacked.HandlerFunc(func(conn net.Conn, bufr *bufio.Reader) {
		resp.NewRedisConnection(conn, bufr).Handle(respHandler)
	})
	return stacked.Detector{
		Needed:  1,
		Test:    resp.IsFirstByteRespTag,
		Handler: hndl,
	}
}

// ListenAndServe starts an "auto" protocol server that will respond to HTTP or
// RESP on the given hostPort.
func ListenAndServe(hostPort string, dss *source.DataSources) error {
	return NewServer(dss).ListenAndServe(hostPort)
}
