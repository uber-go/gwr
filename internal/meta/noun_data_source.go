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

package meta

import (
	"strings"
	"text/template"

	"github.com/uber-go/gwr/source"
)

// NounsName is the name of the meta nouns data source.
const NounsName = "/meta/nouns"

var nounsTextTemplate = template.Must(template.New("meta_nouns_text").Parse(strings.TrimSpace(`
{{ define "get" }}Data Sources:{{ range $name, $info := . }}{{ $name }} formats: {{ $info.Formats }}
{{ end }}{{ end }}
`)))

// NounDataSource provides a data source that describes other data sources.  It
// is used to implement the "/meta/nouns" data source.
type NounDataSource struct {
	sources *source.DataSources
	watcher source.GenericDataWatcher
}

// NewNounDataSource creates a new data source that gets information on other
// data sources and streams updates about them.
func NewNounDataSource(dss *source.DataSources) *NounDataSource {
	return &NounDataSource{
		sources: dss,
	}
}

// Name returns the static "/meta/nouns" string; currently using more than one
// NounDataSource in a single DataSources is unsupported.
func (nds *NounDataSource) Name() string {
	return NounsName
}

// TextTemplate returns a text/template to implement the GenericDataSource with
// a "text" format option.
func (nds *NounDataSource) TextTemplate() *template.Template {
	return nounsTextTemplate
}

// Get returns all currently knows data sources.
func (nds *NounDataSource) Get() interface{} {
	return nds.sources.Info()
}

// WatchInit returns identical data to Get so that all Watch streams start out
// with a snapshot of the world.
func (nds *NounDataSource) WatchInit() interface{} {
	return nds.Get()
}

// SetWatcher implements GenericDataSource by retaining a reference to the
// passed watcher.  Updates are later sent to the watcher when new data sources
// are added and removed.
func (nds *NounDataSource) SetWatcher(watcher source.GenericDataWatcher) {
	nds.watcher = watcher
}

// SourceAdded is called whenever a source is added to the DataSources.
func (nds *NounDataSource) SourceAdded(ds source.DataSource) {
	if !nds.watcher.Active() {
		return
	}
	nds.watcher.HandleItem(struct {
		Type string      `json:"type"`
		Name string      `json:"name"`
		Info source.Info `json:"info"`
	}{"add", ds.Name(), source.GetInfo(ds)})
}

// SourceRemoved is called whenever a source is removed from the DataSources.
func (nds *NounDataSource) SourceRemoved(ds source.DataSource) {
	if !nds.watcher.Active() {
		return
	}
	nds.watcher.HandleItem(struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}{"remove", ds.Name()})
}
