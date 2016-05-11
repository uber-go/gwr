package main

import (
	"log"
	"net/http"

	gwr "github.com/uber-go/gwr"
)

func main() {
	srv := gwr.NewConfiguredServer(gwr.Config{Enabled: true})
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
	defer srv.Stop()

	resLog := &resLogger{handler: http.DefaultServeMux}
	reqLog := &reqLogger{handler: resLog}

	gwr.AddGenericDataSource(reqLog)
	gwr.AddGenericDataSource(resLog)

	log.Fatal(http.ListenAndServe(":8080", reqLog))
}
