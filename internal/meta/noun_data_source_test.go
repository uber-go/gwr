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

package meta_test

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"testing"
	"text/template"

	"github.com/uber-go/gwr/internal/marshaled"
	"github.com/uber-go/gwr/internal/meta"
	"github.com/uber-go/gwr/source"

	"github.com/stretchr/testify/assert"
)

type dummyDataSource struct {
	name string
	tmpl *template.Template
}

func (dds *dummyDataSource) Name() string {
	return dds.name
}

func (dds *dummyDataSource) TextTemplate() *template.Template {
	return dds.tmpl
}

func setup() *source.DataSources {
	dss := source.NewDataSources()
	nds := meta.NewNounDataSource(dss)
	dss.Add(marshaled.NewDataSource(nds, nil))
	dss.SetObserver(nds)
	return dss
}

func TestNounDataSource_Watch(t *testing.T) {
	dss := setup()
	mds := dss.Get("/meta/nouns")

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	// buf := newInternalRWC()
	sc := bufio.NewScanner(r)
	if err := mds.Watch("json", w); err != nil {
		t.Fatal(err)
	}

	getText := func() string {
		var buf bytes.Buffer
		mds.Get("text", &buf)
		return buf.String()
	}

	// verify init data
	assertJSONScanLine(t, sc,
		`{"/meta/nouns":{"formats":["json","text"],"attrs":null}}`,
		"should get /meta/nouns initially")
	assert.Equal(t, getText(), "Data Sources:\n"+
		"/meta/nouns formats: [json text]\n")

	// add a data source, observe it
	assert.NoError(t, dss.Add(marshaled.NewDataSource(&dummyDataSource{
		name: "/foo",
		tmpl: nil,
	}, nil)), "no add error expected")
	assertJSONScanLine(t, sc,
		`{"name":"/foo","type":"add","info":{"formats":["json","text"],"attrs":null}}`,
		"should get an add event for /foo")
	assert.Equal(t, getText(), "Data Sources:\n"+
		"/foo formats: [json text]\n"+
		"/meta/nouns formats: [json text]\n")

	// add another data source, observe it
	assert.NoError(t, dss.Add(marshaled.NewDataSource(&dummyDataSource{
		name: "/bar",
		tmpl: template.Must(template.New("bar_tmpl").Parse("")),
	}, nil)), "no add error expected")
	assertJSONScanLine(t, sc,
		`{"name":"/bar","type":"add","info":{"formats":["json","text"],"attrs":null}}`,
		"should get an add event for /bar")
	assert.Equal(t, getText(), "Data Sources:\n"+
		"/bar formats: [json text]\n"+
		"/foo formats: [json text]\n"+
		"/meta/nouns formats: [json text]\n")

	// remove the /foo data source, observe it
	assert.NotNil(t, dss.Remove("/foo"), "expected a removed data source")
	assertJSONScanLine(t, sc,
		`{"name":"/foo","type":"remove"}`,
		"should get a remove event for /foo")
	assert.Equal(t, getText(), "Data Sources:\n"+
		"/bar formats: [json text]\n"+
		"/meta/nouns formats: [json text]\n")

	// remove the /bar data source, observe it
	assert.NotNil(t, dss.Remove("/bar"), "expected a removed data source")
	assertJSONScanLine(t, sc,
		`{"name":"/bar","type":"remove"}`,
		"should get a remove event for /bar")
	assert.Equal(t, getText(), "Data Sources:\n"+
		"/meta/nouns formats: [json text]\n")

	// shutdown the watch stream
	assert.NoError(t, r.Close())
	assert.False(t, sc.Scan(), "no more scan")
}

func assertJSONScanLine(t *testing.T, sc *bufio.Scanner, expected string, msgAndArgs ...interface{}) {
	if !sc.Scan() {
		assert.Fail(t, "expected to scan a JSON line", msgAndArgs...)
	} else {
		expected = strings.Join([]string{expected, "\n"}, "")
		assert.JSONEq(t, sc.Text(), expected, msgAndArgs...)
	}
	assert.NoError(t, sc.Err())
}
