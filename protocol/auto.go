package protocol

import (
	"bufio"
	"net"

	"github.com/uber-go/gwr"
	"github.com/uber-go/gwr/protocol/resp"
	"github.com/uber-go/gwr/protocol/stacked"
)

// NewAutoServer creates a new stacked server that first tries to detect and
// serve the RESP protocol (if the first byte is one of "-:+$*").  Failing
// that, it falls back to serving HTTP.
func NewAutoServer(dss *gwr.DataSources) *stacked.Server {
	return &stacked.Server{[]stacked.Detector{
		respDetector(NewRedisHandler(dss)),
		stacked.DefaultHTTPHandler(NewHTTPRest(dss, "")),
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

func respDetector(respConnHandler resp.RedisHandler) stacked.Detector {
	hndl := stacked.ConnHandlerFunc(func(conn net.Conn, bufr *bufio.Reader) {
		resp.NewRedisConnection(conn, bufr).Handle(respConnHandler)
	})
	return stacked.Detector{
		Needed:  1,
		Test:    isFirstByteRespTag,
		Handler: hndl,
	}
}
