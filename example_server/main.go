package main

import (
	"log"
	"net/http"

	"code.uber.internal/personal/joshua/gwr"
)

func main() {
	reqLog := &reqLogger{handler: http.DefaultServeMux}

	gwr.AddMarshaledDataSource(reqLog)
	log.Fatal(http.ListenAndServe(":4040", reqLog))
}
