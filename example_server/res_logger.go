// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"net/http"
	"net/http/httptest"
	"text/template"

	"github.com/uber-go/gwr/source"
)

type resLogger struct {
	handler http.Handler
	watcher source.GenericDataWatcher
}

type resInfo struct {
	Code        int    `json:"code"`
	Bytes       int    `json:"bytes"`
	ContentType string `json:"content_type"`
}

var resLogTextTemplate = template.Must(template.New("res_logger_text").Parse(`
{{ define "item" }}{{ .Code }} {{ .Bytes }} {{ .ContentType }}{{ end }}
`))

func (rl *resLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rl.watcher.Active()

	rec := httptest.NewRecorder()
	rl.handler.ServeHTTP(rec, r)

	rl.watcher.HandleItem(resInfo{
		Code:        rec.Code,
		Bytes:       rec.Body.Len(),
		ContentType: rec.HeaderMap.Get("Content-Type"),
	})

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

func (rl *resLogger) SetWatcher(watcher source.GenericDataWatcher) {
	rl.watcher = watcher
}
