package protocol

import (
	"log"
	"net/http"

	"code.uber.internal/personal/joshua/gwr"
)

// TODO: write a http+resp server that heuristically detects the protocol

// ListenAndServeResp starts a redis protocol gwr server.
func ListenAndServeResp(hostPort string) error {
	return NewRedisServer(&gwr.DefaultDataSources).ListenAndServe(hostPort)
}

// ListenAndServeHTTP starts an http protocol gwr server.
func ListenAndServeHTTP(hostPort string) error {
	return http.ListenAndServe(hostPort, NewHTTPRest(&gwr.DefaultDataSources, ""))
}

// ProtoListenAndServe maps protocol names to listenAndServe func(hostPort
// string) error.
var ProtoListenAndServe = map[string]func(string) error{
	"resp": ListenAndServeResp,
	"http": ListenAndServeHTTP,
}

// DefaultProtoHostPorts defines which protocol(s) and port(s) GWR listens on
// by default.
//
// TODO: change based on environment variables, maybe also flags?
var DefaultProtoHostPorts = map[string]string{
	"http": ":4040",
	"resp": ":4041",
}

// ListenAndServe starts one or more servers given a map of protocol name to
// hostPort string.  Any errors are passed to log.Fatal.
func ListenAndServe(protoHostPorts map[string]string) {
	if len(protoHostPorts) == 0 {
		protoHostPorts = DefaultProtoHostPorts
	} else {
		for proto := range protoHostPorts {
			if ProtoListenAndServe[proto] == nil {
				log.Fatalf("invalid protocol %v", proto)
			}
		}
	}

	if len(protoHostPorts) == 1 {
		for proto, hostPort := range protoHostPorts {
			listenAndServe := ProtoListenAndServe[proto]
			log.Fatal(listenAndServe(hostPort))
			return
		}
	}

	for proto, hostPort := range protoHostPorts {
		go func(proto, hostPort string) {
			listenAndServe := ProtoListenAndServe[proto]
			log.Fatal(listenAndServe(hostPort))
		}(proto, hostPort)
	}
	select {}
}
