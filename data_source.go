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

package gwr

import (
	"github.com/uber-go/gwr/internal/marshaled"
	"github.com/uber-go/gwr/internal/meta"
	"github.com/uber-go/gwr/source"
)

// DefaultDataSources is default data sources registry which data sources are
// added to by the module-level Add* functions.  It is used by all of the
// protocol servers if no data sources are provided.
var DefaultDataSources *source.DataSources

func init() {
	DefaultDataSources = source.NewDataSources()
	metaNouns := meta.NewNounDataSource(DefaultDataSources)
	DefaultDataSources.Add(marshaled.NewDataSource(metaNouns, nil))
	DefaultDataSources.SetObserver(metaNouns)
}

// AddDataSource adds a data source to the default data sources registry.  It
// returns an error if there's already a data source defined with the same
// name.
func AddDataSource(ds source.DataSource) error {
	return DefaultDataSources.Add(ds)
}

// AddGenericDataSource adds a generic data source to the default data sources
// registry.  It returns an error if there's already a data source defined with
// the same name.
func AddGenericDataSource(gds source.GenericDataSource) error {
	mds := marshaled.NewDataSource(gds, nil)
	return DefaultDataSources.Add(mds)
}
