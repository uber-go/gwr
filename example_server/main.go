package main

import (
	"net/http"

	"github.com/uber-go/gwr"
	gwrProto "github.com/uber-go/gwr/protocol"
)

func main() {
	go func() {
		gwrProto.ListenAndServe(nil, nil)
	}()

	resLog := &resLogger{handler: http.DefaultServeMux}
	reqLog := &reqLogger{handler: resLog}

	gwr.AddMarshaledDataSource(reqLog)
	gwr.AddMarshaledDataSource(resLog)

	http.ListenAndServe(":8080", reqLog)
}
