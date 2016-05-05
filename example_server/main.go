package main

import (
	"log"
	"net/http"

	"github.com/uber-go/gwr"
)

func main() {
	go func() {
		log.Fatal(gwr.ListenAndServe(":4040", nil))
	}()

	resLog := &resLogger{handler: http.DefaultServeMux}
	reqLog := &reqLogger{handler: resLog}

	gwr.AddGenericDataSource(reqLog)
	gwr.AddGenericDataSource(resLog)

	log.Fatal(http.ListenAndServe(":8080", reqLog))
}
