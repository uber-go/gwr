package main

import (
	"log"
	"net/http"

	"code.uber.internal/personal/joshua/gwr"
)

func main() {
	resLog := &resLogger{handler: http.DefaultServeMux}
	reqLog := &reqLogger{handler: resLog}

	gwr.AddMarshaledDataSource(reqLog)
	gwr.AddMarshaledDataSource(resLog)
	log.Fatal(http.ListenAndServe(":4040", reqLog))
}
