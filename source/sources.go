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

package source

import "errors"

var ErrSourceAlreadyDefined = errors.New("data source already defined")

// DataSourcesObserver is an interface to observe data sources changes.
//
// Observation happens after after the source has been added (resp. removed).
type DataSourcesObserver interface {
	SourceAdded(ds DataSource)
	SourceRemoved(ds DataSource)
}

// DataSources is a flat collection of DataSources
// with a meta introspection data source.
type DataSources struct {
	sources map[string]DataSource
	obs     DataSourcesObserver
}

// NewDataSources creates a DataSources structure
// an sets up its "/meta/nouns" data source.
func NewDataSources() *DataSources {
	dss := &DataSources{
		sources: make(map[string]DataSource, 2),
	}
	return dss
}

// SetObserver sets the (single!) observer of data source changes; if nil is
// passed, observation is disabled.
func (dss *DataSources) SetObserver(obs DataSourcesObserver) {
	dss.obs = obs
}

// Get returns the named data source or nil if none is defined.
func (dss *DataSources) Get(name string) DataSource {
	source, ok := dss.sources[name]
	if ok {
		return source
	}
	return nil
}

// Add a DataSource, if none is already defined for the given name.
func (dss *DataSources) Add(ds DataSource) error {
	name := ds.Name()
	if _, ok := dss.sources[name]; ok {
		return ErrSourceAlreadyDefined
	}
	dss.sources[name] = ds
	if dss.obs != nil {
		dss.obs.SourceAdded(ds)
	}
	return nil
}

// Remove a DataSource by name, if any exsits.  Returns the source removed, nil
// if none was defined.
func (dss *DataSources) Remove(name string) DataSource {
	ds, ok := dss.sources[name]
	if ok {
		delete(dss.sources, name)
		if dss.obs != nil {
			dss.obs.SourceRemoved(ds)
		}
	}
	return ds
}
