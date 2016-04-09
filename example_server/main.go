package main

import (
	"log"
	"net/http"

	"code.uber.internal/personal/joshua/gwr"
	"code.uber.internal/personal/joshua/gwr/protocol"
)

func main() {
	resLog := &resLogger{handler: http.DefaultServeMux}
	reqLog := &reqLogger{handler: resLog}

	gwr.AddMarshaledDataSource(reqLog)
	gwr.AddMarshaledDataSource(resLog)
	go func() {
		log.Fatal(http.ListenAndServe(":4040", reqLog))
	}()

	go func() {
		log.Fatal(protocol.NewRedisServer(&gwr.DefaultDataSources).ListenAndServe(":6379"))
	}()

	select {}
}
