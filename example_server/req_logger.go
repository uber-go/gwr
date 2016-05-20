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
{{ define "item" }}{{ .Method }} {{ .Path }} {{ .Query }}{{ end }}
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

func (rl *reqLogger) SetWatcher(watcher source.GenericDataWatcher) {
	rl.watcher = watcher
}
