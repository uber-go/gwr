package protocol

import (
	"net/http"

	"github.com/uber-go/gwr"
	"github.com/uber-go/gwr/protocol/resp"
	"github.com/uber-go/gwr/protocol/stacked"
)

// ListenAndServeResp starts a resp protocol gwr server.
func ListenAndServeResp(hostPort string, dss *gwr.DataSources) error {
	if dss == nil {
		dss = &gwr.DefaultDataSources
	}
	return NewRedisServer(dss).ListenAndServe(hostPort)
}

// ListenAndServeHTTP starts an http protocol gwr server.
func ListenAndServeHTTP(hostPort string, dss *gwr.DataSources) error {
	if dss == nil {
		dss = &gwr.DefaultDataSources
	}
	return http.ListenAndServe(hostPort, NewHTTPRest(dss, ""))
}

// TODO: support environment variable and/or flag for port(s)

// NewServer creates an "auto" protocol server that will respond to HTTP or
// RESP requests.
func NewServer(dss *gwr.DataSources) *stacked.Server {
	if dss == nil {
		dss = &gwr.DefaultDataSources
	}
	hh := NewHTTPRest(dss, "")
	rh := NewRedisHandler(dss)
	return resp.WrapHTTPHandler(rh, hh)
}

// ListenAndServe starts an "auto" protocol server that will respond to HTTP or
// RESP on the given hostPort.
func ListenAndServe(hostPort string, dss *gwr.DataSources) error {
	return NewServer(dss).ListenAndServe(hostPort)
}
