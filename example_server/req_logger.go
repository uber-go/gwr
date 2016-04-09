package main

import (
	"net/http"
	"text/template"

	"code.uber.internal/personal/joshua/gwr"
)

type reqLogger struct {
	handler http.Handler
	watcher gwr.GenericDataWatcher
}

func logged(handler http.Handler) *reqLogger {
	return &reqLogger{
		handler: handler,
	}
}

var reqLogTextTemplate = template.Must(template.New("req_logger_text").Parse(`
{{- define "item" -}}
{{ .Method }} {{ .Path }} {{ .Query }}
{{ end -}}
`))

type reqInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Query  string `json:"query"`
}

func (rl *reqLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if rl.watcher != nil {
		info := reqInfo{
			Method: r.Method,
			Path:   r.URL.Path,
			Query:  r.URL.RawQuery,
		}
		if !rl.watcher(info) {
			rl.watcher = nil
		}
	}
	rl.handler.ServeHTTP(w, r)
}

func (rl *reqLogger) Info() gwr.GenericDataSourceInfo {
	return gwr.GenericDataSourceInfo{
		Name: "/request_log",
		// TODO: afford watch-only nature
		TextTemplate: reqLogTextTemplate,
	}
}

func (rl *reqLogger) Get() interface{} {
	return nil
}

func (rl *reqLogger) GetInit() interface{} {
	return nil
}

func (rl *reqLogger) Watch(watcher gwr.GenericDataWatcher) {
	rl.watcher = watcher
}
