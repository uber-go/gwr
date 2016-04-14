package main

import (
	"net/http"
	"net/http/httptest"
	"text/template"

	"code.uber.internal/personal/joshua/gwr"
)

type resLogger struct {
	handler http.Handler
	watcher gwr.GenericDataWatcher
}

type resInfo struct {
	Code        int    `json:"code"`
	Bytes       int    `json:"bytes"`
	ContentType string `json:"content_type"`
}

var resLogTextTemplate = template.Must(template.New("res_logger_text").Parse(`
{{- define "item" -}}
{{ .Code }} {{ .Bytes }} {{ .ContentType }}
{{ end -}}
`))

func (rl *resLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if rl.watcher == nil {
		rl.handler.ServeHTTP(w, r)
		return
	}

	rec := httptest.NewRecorder()
	rl.handler.ServeHTTP(rec, r)

	info := resInfo{
		Code:        rec.Code,
		Bytes:       rec.Body.Len(),
		ContentType: rec.HeaderMap.Get("Content-Type"),
	}
	if !rl.watcher(info) {
		rl.watcher = nil
	}

	hdr := w.Header()
	for key, vals := range rec.HeaderMap {
		hdr[key] = vals
	}
	w.WriteHeader(rec.Code)
	rec.Body.WriteTo(w)
}

func (rl *resLogger) Name() string {
	return "/response_log"
}

func (rl *resLogger) TextTemplate() *template.Template {
	return resLogTextTemplate
}

func (rl *resLogger) Info() gwr.GenericDataSourceInfo {
	return gwr.GenericDataSourceInfo{
		// TODO: afford watch-only nature
	}
}

func (rl *resLogger) Get() interface{} {
	return nil
}

func (rl *resLogger) GetInit() interface{} {
	return nil
}

func (rl *resLogger) Watch(watcher gwr.GenericDataWatcher) {
	rl.watcher = watcher
}
