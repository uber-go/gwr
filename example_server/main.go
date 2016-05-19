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
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	gwr "github.com/uber-go/gwr"
	"github.com/uber-go/gwr/source/tap"
)

func main() {
	if err := gwr.Configure(nil); err != nil {
		log.Fatal(err)
	}

	resLog := &resLogger{handler: http.DefaultServeMux}
	reqLog := &reqLogger{handler: resLog}

	gwr.AddGenericDataSource(reqLog)
	gwr.AddGenericDataSource(resLog)

	fb := fibber{
		naive: tap.AddNewTracer("fib/naive"),
	}
	http.HandleFunc("/fib/naive", fb.handleNaive)

	log.Fatal(http.ListenAndServe(":8080", reqLog))
}

type fibber struct {
	naive *tap.Tracer
}

func (fb *fibber) handleNaive(w http.ResponseWriter, r *http.Request) {
	trc := fb.naive.Scope("handleNaive").Open(map[string]interface{}{
		"method": r.Method,
		"proto":  r.Proto,
		"url":    r.URL,
		"header": r.Header,
		"host":   r.Host,
	})
	defer trc.Close()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		trc.ErrorName("ParseForm", err)
		return
	}
	trc.Info("parsed form", r.Form)

	i, err := strconv.Atoi(r.Form.Get("n"))
	if err != nil {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		trc.ErrorName("Atoi", err)
		return
	}

	n := naiveFib(i, trc)
	io.WriteString(w, fmt.Sprintf("fib(%d) = %d\n", i, n))
}

func naiveFib(i int, trc *tap.TraceScope) (n int) {
	sc := trc.Sub("naiveFib").Open(i)
	defer func() {
		sc.Close(n)
	}()

	if i <= 0 {
		n = 0
		return
	}

	if i <= 2 {
		n = 1
		return
	}

	n = naiveFib(i-1, sc) + naiveFib(i-2, sc)
	return
}
