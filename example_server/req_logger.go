package main

import (
	"net/http"
	"text/template"

	"github.com/uber-go/gwr/source"
)

type reqLogger struct {
	handler http.Handler
	watcher source.GenericDataWatcher
}

func logged(handler http.Handler) *reqLogger {
	return &reqLogger{
		handler: handler,
	}
}

var reqLogTextTemplate = template.Must(template.New("req_logger_text").Parse(`
{{- define "item" -}}
{{ .Method }} {{ .Path }} {{ .Query }}
{{- end -}}
`))

type reqInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Query  string `json:"query"`
}

func (rl *reqLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if watcher := rl.watcher; watcher != nil {
		info := reqInfo{
			Method: r.Method,
			Path:   r.URL.Path,
			Query:  r.URL.RawQuery,
		}
		if !watcher.HandleItem(info) {
			rl.watcher = nil
		}
	}
	rl.handler.ServeHTTP(w, r)
}

func (rl *reqLogger) Name() string {
	return "/request_log"
}

func (rl *reqLogger) TextTemplate() *template.Template {
	return reqLogTextTemplate
}

func (rl *reqLogger) Attrs() map[string]interface{} {
	return nil
}

func (rl *reqLogger) Get() interface{} {
	return nil
}

func (rl *reqLogger) GetInit() interface{} {
	return nil
}

func (rl *reqLogger) Watch(watcher source.GenericDataWatcher) {
	rl.watcher = watcher
}
