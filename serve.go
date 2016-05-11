package gwr

import (
	"bufio"
	"net"
	"net/http"

	"github.com/uber-go/gwr/internal/protocol"
	"github.com/uber-go/gwr/internal/resp"
	"github.com/uber-go/gwr/source"

	"github.com/uber-common/stacked"
)

func init() {
	http.Handle("/gwr/", protocol.NewHTTPRest(DefaultDataSources, "/gwr"))
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
	return http.ListenAndServe(hostPort, protocol.NewHTTPRest(dss, ""))
}

// NewServer creates an "auto" protocol server that will respond to HTTP or
// RESP requests.
func NewServer(dss *source.DataSources) stacked.Server {
	if dss == nil {
		dss = DefaultDataSources
	}
	hh := protocol.NewHTTPRest(dss, "")
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
