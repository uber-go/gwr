package main

import (
	"net/http"

	"code.uber.internal/personal/joshua/gwr"
	gwrProto "code.uber.internal/personal/joshua/gwr/protocol"
)

func main() {
	go func() {
		gwrProto.ListenAndServe(map[string]string{
			"http": ":4040",
			"resp": ":4041",
		})
	}()

	resLog := &resLogger{handler: http.DefaultServeMux}
	reqLog := &reqLogger{handler: resLog}

	gwr.AddMarshaledDataSource(reqLog)
	gwr.AddMarshaledDataSource(resLog)

	http.ListenAndServe(":8080", reqLog)
}
