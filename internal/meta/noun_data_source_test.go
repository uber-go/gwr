package meta_test

import (
	"bufio"
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
	name  string
	attrs map[string]interface{}
	tmpl  *template.Template
}

func (dds *dummyDataSource) Name() string {
	return dds.name
}

func (dds *dummyDataSource) Attrs() map[string]interface{} {
	return dds.attrs
}

func (dds *dummyDataSource) TextTemplate() *template.Template {
	return dds.tmpl
}

func (dds *dummyDataSource) Get() interface{} {
	return nil
}

func (dds *dummyDataSource) GetInit() interface{} {
	return nil
}

func (dds *dummyDataSource) SetWatcher(watcher source.GenericDataWatcher) {
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

	// verify init data
	assertJSONScanLine(t, sc,
		`{"/meta/nouns":{"formats":["json","text"],"attrs":null}}`,
		"should get /meta/nouns initially")

	// add a data source, observe it
	assert.NoError(t, dss.Add(marshaled.NewDataSource(&dummyDataSource{
		name:  "/foo",
		attrs: map[string]interface{}{"aKey": "aVal"},
		tmpl:  nil,
	}, nil)), "no add error expected")
	assertJSONScanLine(t, sc,
		`{"name":"/foo","type":"add","info":{"formats":["json"],"attrs":{"aKey":"aVal"}}}`,
		"should get an add event for /foo")

	// add another data source, observe it
	assert.NoError(t, dss.Add(marshaled.NewDataSource(&dummyDataSource{
		name:  "/bar",
		attrs: map[string]interface{}{"bKey": 123},
		tmpl:  template.Must(template.New("bar_tmpl").Parse("")),
	}, nil)), "no add error expected")
	assertJSONScanLine(t, sc,
		`{"name":"/bar","type":"add","info":{"formats":["json","text"],"attrs":{"bKey":123}}}`,
		"should get an add event for /bar")

	// remove the /foo data source, observe it
	assert.NotNil(t, dss.Remove("/foo"), "expected a removed data source")
	assertJSONScanLine(t, sc,
		`{"name":"/foo","type":"remove"}`,
		"should get a remove event for /foo")

	// remove the /bar data source, observe it
	assert.NotNil(t, dss.Remove("/bar"), "expected a removed data source")
	assertJSONScanLine(t, sc,
		`{"name":"/bar","type":"remove"}`,
		"should get a remove event for /bar")

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
