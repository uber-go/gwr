package gwr_test

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"text/template"

	gwr "github.com/uber-go/gwr"
	"github.com/uber-go/gwr/source"
)

type accessLogger struct {
	handler http.Handler
	watcher source.GenericDataWatcher
}

func logged(handler http.Handler) *accessLogger {
	if handler == nil {
		handler = http.DefaultServeMux
	}
	return &accessLogger{
		handler: handler,
	}
}

type accessEntry struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Query       string `json:"query"`
	Code        int    `json:"code"`
	Bytes       int    `json:"bytes"`
	ContentType string `json:"content_type"`
}

var accessLogTextTemplate = template.Must(template.New("req_logger_text").Parse(`
{{- define "item" -}}
{{ .Method }} {{ .Path }}{{ if .Query }}?{{ .Query }}{{ end }} {{ .Code }} {{ .Bytes }} {{ .ContentType }}
{{- end -}}
`))

func (al *accessLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	watcher := al.watcher
	if watcher == nil {
		al.handler.ServeHTTP(w, r)
		return
	}

	rec := httptest.NewRecorder()
	al.handler.ServeHTTP(rec, r)
	bytes := rec.Body.Len()

	// finishing work first...
	hdr := w.Header()
	for key, vals := range rec.HeaderMap {
		hdr[key] = vals
	}
	w.WriteHeader(rec.Code)
	if _, err := rec.Body.WriteTo(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// ...then emitting the entry may afford slightly less overhead; most
	// overhead, like marshalling, should be deferred by the gwr library,
	// watcher.HandleItem is supposed to be fast enough to not need a channel
	// indirection within each source.
	if !watcher.HandleItem(accessEntry{
		Method:      r.Method,
		Path:        r.URL.Path,
		Query:       r.URL.RawQuery,
		Code:        rec.Code,
		Bytes:       bytes,
		ContentType: rec.HeaderMap.Get("Content-Type"),
	}) {
		al.watcher = nil
	}
}

func (al *accessLogger) Name() string {
	return "/access_log"
}

func (al *accessLogger) TextTemplate() *template.Template {
	return accessLogTextTemplate
}

func (al *accessLogger) SetWatcher(watcher source.GenericDataWatcher) {
	al.watcher = watcher
}

// TODO: this has become more test than example; maybe just make it a test?

func Example_httpserver_accesslog() {
	// Uses :0 for no conflict in test.
	if err := gwr.Configure(&gwr.Config{ListenAddr: ":0"}); err != nil {
		log.Fatal(err)
	}
	defer gwr.DefaultServer().Stop()
	gwrAddr := gwr.DefaultServer().Addr()

	// a handler so we get more than just 404s
	http.HandleFunc("/foo", func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, "Ok ;-)\n"); err != nil {
			panic(err.Error())
		}
	})

	// wraps the default serve mux in an access logging gwr source...
	loggedHTTPHandler := logged(nil)

	// ...which we then register with gwr
	gwr.AddGenericDataSource(loggedHTTPHandler)

	// Again note the :0 pattern complicates things more than normal; this is
	// just the default http server
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}
	svcAddr := ln.Addr()
	go http.Serve(ln, loggedHTTPHandler)

	// make two requests, one should get a 200, the other a 404; we should see
	// only their output since no watchers are going yet
	fmt.Println("start")
	httpGetStdout("http://%s/foo?num=1", svcAddr)
	httpGetStdout("http://%s/bar?num=2", svcAddr)

	// start a json watch on the access log; NOTE we don't care about any copy
	// error because normal termination here is closed-mid-read; TODO we could
	// tighten this to log fatal any non "closed" error
	jsonLines := newHTTPGetChan("JSON", "http://%s/access_log?format=json&watch=1", gwrAddr)
	fmt.Println("\nwatching json")

	// make two requests, now with json watcher
	httpGetStdout("http://%s/foo?num=3", svcAddr)
	httpGetStdout("http://%s/bar?num=4", svcAddr)
	jsonLines.printOne()
	jsonLines.printOne()

	// start a text watch on the access log
	textLines := newHTTPGetChan("TEXT", "http://%s/access_log?format=text&watch=1", gwrAddr)
	fmt.Println("\nwatching text & json")

	// make two requests, now with both watchers
	httpGetStdout("http://%s/foo?num=5", svcAddr)
	httpGetStdout("http://%s/bar?num=6", svcAddr)
	jsonLines.printOne()
	jsonLines.printOne()
	textLines.printOne()
	textLines.printOne()

	// shutdown the json watch
	jsonLines.close()
	fmt.Println("\njust text")

	// make two requests; we should still see the body copies, but only get the text watch data
	httpGetStdout("http://%s/foo?num=7", svcAddr)
	httpGetStdout("http://%s/bar?num=8", svcAddr)
	textLines.printOne()
	textLines.printOne()

	// shutdown the json watch
	textLines.close()
	fmt.Println("\nno watchers")

	// make two requests; we should still see the body copies, but get no watch data for them
	httpGetStdout("http://%s/foo?num=9", svcAddr)
	httpGetStdout("http://%s/bar?num=10", svcAddr)

	// output: start
	// Ok ;-)
	// 404 page not found
	//
	// watching json
	// Ok ;-)
	// 404 page not found
	// JSON: {"method":"GET","path":"/foo","query":"num=3","code":200,"bytes":7,"content_type":"text/plain; charset=utf-8"}
	// JSON: {"method":"GET","path":"/bar","query":"num=4","code":404,"bytes":19,"content_type":"text/plain; charset=utf-8"}
	//
	// watching text & json
	// Ok ;-)
	// 404 page not found
	// JSON: {"method":"GET","path":"/foo","query":"num=5","code":200,"bytes":7,"content_type":"text/plain; charset=utf-8"}
	// JSON: {"method":"GET","path":"/bar","query":"num=6","code":404,"bytes":19,"content_type":"text/plain; charset=utf-8"}
	// TEXT: GET /foo?num=5 200 7 text/plain; charset=utf-8
	// TEXT: GET /bar?num=6 404 19 text/plain; charset=utf-8
	// JSON: CLOSE
	//
	// just text
	// Ok ;-)
	// 404 page not found
	// TEXT: GET /foo?num=7 200 7 text/plain; charset=utf-8
	// TEXT: GET /bar?num=8 404 19 text/plain; charset=utf-8
	// TEXT: CLOSE
	//
	// no watchers
	// Ok ;-)
	// 404 page not found
}

// test brevity conveniences

func httpGetStdout(format string, args ...interface{}) {
	url := fmt.Sprintf(format, args...)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
		log.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		log.Fatal(err)
	}
}

type httpGetChan struct {
	tag string
	c   chan string
	r   *http.Response
}

func newHTTPGetChan(tag string, format string, args ...interface{}) *httpGetChan {
	url := fmt.Sprintf(format, args...)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	hc := &httpGetChan{
		tag: tag,
		c:   make(chan string),
		r:   resp,
	}
	go hc.scanLines()
	return hc
}

func (hc *httpGetChan) printOne() {
	if _, err := fmt.Printf("%s: %s\n", hc.tag, <-hc.c); err != nil {
		log.Fatal(err)
	}
}

func (hc *httpGetChan) close() {
	if err := hc.r.Body.Close(); err != nil {
		log.Fatal(err)
	}
	hc.printOne()
}

func (hc *httpGetChan) scanLines() {
	s := bufio.NewScanner(hc.r.Body)
	for s.Scan() {
		hc.c <- s.Text()
	}
	hc.c <- "CLOSE"
	close(hc.c)
	// NOTE: s.Err() intentionally not checking since this is a "just"
	// function; in particular, we expect to get a closed-during-read error
}
