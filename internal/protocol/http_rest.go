package protocol

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/uber-go/gwr/internal/meta"
	"github.com/uber-go/gwr/source"
)

// Servable is a minimal server interface supported by the "/listen" facility.
type Servable interface {
	Addr() net.Addr
	StartOn(string) error
	Stop() error
}

var formatContetTypes = map[string]string{
	"json": "application/json",
	"text": "text/plain",
	"html": "text/html",
}

func contentTypeFor(formatName string) string {
	if contetType, ok := formatContetTypes[formatName]; ok {
		return contetType
	}
	return "application/octet"
}

// HTTPRest implements http.Handler to host a collection of data sources
// REST-fully.
type HTTPRest struct {
	defaultFormats []string
	prefix         string
	dss            *source.DataSources
	srv            Servable
}

// NewHTTPRest returns an http.Handler to host the data sources REST-fully at a
// given prefix.
//
// If a non-nil servable is passed, then a /listen convenience endpoint will be
// provided to afford server discovery and lifecycle management.
func NewHTTPRest(dss *source.DataSources, prefix string, srv Servable) *HTTPRest {
	return &HTTPRest{
		defaultFormats: []string{"text", "json"},
		prefix:         prefix,
		dss:            dss,
		srv:            srv,
	}
}

func (hndl *HTTPRest) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := hndl.routeSource(w, r); err != nil {
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		log.Printf("data source serve failed: %v\n", err)
		// XXX log
		return
	}
}

func (hndl *HTTPRest) doListen(w http.ResponseWriter, r *http.Request) error {
	// TODO: this could be "just" another meta source, if sources had a way to
	// define custom actions, e.g. to tell it to go listen
	switch strings.ToLower(r.Method) {
	case "get":
		addr := hndl.srv.Addr()
		if addr == nil {
			http.Error(w,
				"503 Not Listening\nServer not started, POST an address to start it.",
				http.StatusServiceUnavailable)
			return nil
		}
		io.WriteString(w, fmt.Sprintf("%v\n", addr))

	case "post":
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			if err := r.ParseMultipartForm(1024); err != nil {
				return err
			}
		} else if err := r.ParseForm(); err != nil {
			return err
		}

		if r.Form.Get("stop") != "" {
			if hndl.srv.Addr() == nil {
				io.WriteString(w, "not running\n")
			} else if err := hndl.srv.Stop(); err != nil {
				io.WriteString(w, err.Error())
			} else {
				io.WriteString(w, "stopped\n")
			}
			return nil
		}

		if laddr := r.Form.Get("address"); laddr == "" {
			http.Error(w, "400 Missing \"address\" form value.", http.StatusBadRequest)
			return nil
		} else if err := hndl.srv.StartOn(laddr); err != nil {
			http.Error(w,
				fmt.Sprintf("503 Unable to start server\nstart failed: %s", err.Error()),
				http.StatusServiceUnavailable)
			return nil
		}

		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, fmt.Sprintf("%v\n", hndl.srv.Addr()))

	default:
		w.Header().Set("Allow", "GET, POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, "405 Invalid Method\n")
	}

	return nil
}

func (hndl *HTTPRest) routeSource(w http.ResponseWriter, r *http.Request) error {
	path := r.URL.Path[len(hndl.prefix):]
	if hndl.srv != nil && path == "/listen" {
		return hndl.doListen(w, r)
	}

	var src source.DataSource
	if len(path) == 0 || path == "/" {
		src = hndl.dss.Get(meta.NounsName)
	} else {
		src = hndl.dss.Get(path)
	}
	if src == nil {
		http.NotFound(w, r)
		return nil
	}
	return hndl.routeVerb(src, w, r)
}

func (hndl *HTTPRest) routeVerb(
	src source.DataSource,
	w http.ResponseWriter,
	r *http.Request,
) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	switch strings.ToLower(r.Method) {
	case "get":
		if r.Form.Get("watch") != "" {
			// convenience for http clients that don't easily support custom
			// method strings
			return hndl.doWatch(src, w, r)
		}
		return hndl.doGet(src, w, r)

	case "watch":
		return hndl.doWatch(src, w, r)

	default:
		w.Header().Set("Allow", "GET, WATCH")
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, "405 Invalid Method\n")
	}
	return nil
}

func (hndl *HTTPRest) doGet(
	src source.DataSource,
	w http.ResponseWriter,
	r *http.Request,
) error {
	formatName, err := hndl.determineFormat(src, w, r)
	if len(formatName) == 0 || err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := src.Get(formatName, &buf); err == source.ErrNotGetable {
		http.Error(w, "501 source does not support Get", http.StatusNotImplemented)
		return nil
	} else if err != nil {
		return err
	}

	if contetType, ok := formatContetTypes[formatName]; ok {
		w.Header().Set("Content-Type", contetType)
	} else {
		w.Header().Set("Content-Type", "application/octet")
	}

	w.WriteHeader(http.StatusOK)
	_, err = buf.WriteTo(w)
	return err
}

type flushWriter struct {
	w io.Writer
	f http.Flusher
}

func (fw *flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	fw.f.Flush()
	return n, err
}

func (hndl *HTTPRest) doWatch(
	src source.DataSource,
	w http.ResponseWriter,
	r *http.Request,
) error {
	formatName, err := hndl.determineFormat(src, w, r)
	if len(formatName) == 0 || err != nil {
		return err
	}

	ready := make(chan *chanBuf, 1)
	var buf = chanBuf{ready: ready}
	defer buf.Close()

	if err := src.Watch(formatName, &buf); err == source.ErrNotWatchable {
		http.Error(w, "501 source does not support Watch", http.StatusNotImplemented)
		return nil
	} else if err != nil {
		return err
	}

	w.Header().Set("Content-Type", contentTypeFor(formatName))
	w.Header().Set("Transfer-Encoding", "chunked")

	w.WriteHeader(http.StatusOK)

	var fw io.Writer = w

	if f, _ := w.(http.Flusher); f != nil {
		f.Flush()
		fw = &flushWriter{w, f}
	}

	var cn <-chan bool
	if cnr, ok := w.(http.CloseNotifier); ok {
		cn = cnr.CloseNotify()
	}

	for {
		select {
		case <-ready:
			if _, err := buf.writeTo(fw); err != nil {
				return err
			}
		case <-cn:
			// TODO: don't get this, why
			return nil
		}
	}
}

func (hndl *HTTPRest) determineFormat(
	src source.DataSource,
	w http.ResponseWriter,
	r *http.Request,
) (string, error) {
	// TODO: some people like Accepts negotiation

	formats := src.Formats()

	formatName := r.Form.Get("format")
	if len(formatName) != 0 {
		for _, availFormat := range formats {
			if strings.EqualFold(formatName, availFormat) {
				return availFormat, nil
			}
		}
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "400 Bad Request\nUnsupported Format\n")
		return "", nil
	}

	for _, defaultFormat := range hndl.defaultFormats {
		for _, availFormat := range formats {
			if strings.EqualFold(availFormat, defaultFormat) {
				return availFormat, nil
			}
		}
	}

	return formats[0], nil
}
