package protocol

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strings"

	"code.uber.internal/personal/joshua/gwr"
)

var formatContetTypes = map[string]string{
	"json": "application/json",
	"text": "text/plain",
	"html": "text/html",
}

func init() {
	http.Handle("/gwr/", NewHTTPRest(&gwr.DefaultDataSources, "/gwr"))
}

// HTTPRest implements http.Handler to host a collection of data sources
// REST-fully.
type HTTPRest struct {
	defaultFormats []string
	prefix         string
	dss            *gwr.DataSources
}

// NewHTTPRest returns an http.Handler to host the data sources REST-fully at a
// given prefix.
func NewHTTPRest(dss *gwr.DataSources, prefix string) *HTTPRest {
	return &HTTPRest{
		defaultFormats: []string{"text", "json"},
		prefix:         prefix,
		dss:            dss,
	}
}

func (hndl *HTTPRest) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := hndl.routeSource(w, r); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "500 Internal Server Error\n")
		log.Printf("data source serve failed: %v\n", err)
		// XXX log
		return
	}
}

func (hndl *HTTPRest) routeSource(w http.ResponseWriter, r *http.Request) error {
	name := r.URL.Path[len(hndl.prefix):]

	source := hndl.dss.Get(name)
	if source == nil {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, "404 Not Found\nNo such data source\n")
		return nil
	}

	if err := hndl.routeVerb(source, w, r); err != nil {
		return err
	}

	return nil
}

func (hndl *HTTPRest) routeVerb(
	source gwr.DataSource,
	w http.ResponseWriter,
	r *http.Request,
) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	switch strings.ToLower(r.Method) {
	case "get":
		if err := hndl.doGet(source, w, r); err != nil {
			return err
		}

	case "watch":
		if err := hndl.doWatch(source, w, r); err != nil {
			return err
		}

	default:
		w.Header().Set("Allow", "GET, WATCH")
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, "405 Invalid Method\n")
	}
	return nil
}

func (hndl *HTTPRest) doGet(
	source gwr.DataSource,
	w http.ResponseWriter,
	r *http.Request,
) error {
	formatName, err := hndl.determineFormat(source, w, r)
	if len(formatName) == 0 || err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := source.Get(formatName, &buf); err != nil {
		return err
	}

	if contetType, ok := formatContetTypes[formatName]; ok {
		w.Header().Set("Content-Type", contetType)
	} else {
		w.Header().Set("Content-Type", "application/octet")
	}

	w.WriteHeader(http.StatusOK)
	if _, err := buf.WriteTo(w); err != nil {
		return err
	}

	return nil
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
	source gwr.DataSource,
	w http.ResponseWriter,
	r *http.Request,
) error {
	formatName, err := hndl.determineFormat(source, w, r)
	if len(formatName) == 0 || err != nil {
		return err
	}

	ready := make(chan *chanBuf, 1)
	var buf = chanBuf{ready: ready}
	defer buf.close()

	if err := source.Watch(formatName, &buf); err != nil {
		return err
	}

	if contetType, ok := formatContetTypes[formatName]; ok {
		w.Header().Set("Content-Type", contetType)
	} else {
		w.Header().Set("Content-Type", "application/octet")
	}
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
	source gwr.DataSource,
	w http.ResponseWriter,
	r *http.Request,
) (string, error) {
	// TODO: some people like Accepts negotiation

	formats := source.Formats()

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
