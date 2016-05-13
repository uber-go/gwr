package main

import (
	"log"
	"net/http"

	gwr "github.com/uber-go/gwr"
)

func main() {
	if err := gwr.Configure(nil); err != nil {
		log.Fatal(err)
	}

	resLog := &resLogger{handler: http.DefaultServeMux}
	reqLog := &reqLogger{handler: resLog}

	gwr.AddGenericDataSource(reqLog)
	gwr.AddGenericDataSource(resLog)

	log.Fatal(http.ListenAndServe(":8080", reqLog))
}
