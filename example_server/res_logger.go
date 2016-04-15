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
	watcher := rl.watcher
	if watcher == nil {
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
	if !watcher(info) {
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

func (rl *resLogger) Attrs() map[string]interface{} {
	return nil
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
