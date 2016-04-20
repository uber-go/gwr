package protocol

import (
	"net/http"

	"github.com/uber-go/gwr"
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

// ListenAndServe starts an "auto" protocol server that will respond to HTTP or
// RESP on the given hostPort.
func ListenAndServe(hostPort string, dss *gwr.DataSources) error {
	if dss == nil {
		dss = &gwr.DefaultDataSources
	}
	return NewAutoServer(dss).ListenAndServe(hostPort)
}
