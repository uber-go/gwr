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

// Info is a convenience info descriptor about a data source.
type Info struct {
	Formats []string               `json:"formats"`
	Attrs   map[string]interface{} `json:"attrs"`
}

// GetInfo returns a structure that contains format and other information about
// a given data source.
func GetInfo(ds DataSource) Info {
	return Info{
		Formats: ds.Formats(),
		Attrs:   ds.Attrs(),
	}
}

// Info returns a map of info about all sources.
func (dss *DataSources) Info() map[string]Info {
	info := make(map[string]Info, len(dss.sources))
	for name, ds := range dss.sources {
		info[name] = GetInfo(ds)
	}
	return info
}
