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

import (
	"errors"
	"io"
)

var (
	// ErrUnsupportedFormat should be returned by DataSource.Get and
	// DataSource.Watch if the requested format is not supported.
	ErrUnsupportedFormat = errors.New("unsupported format")

	// ErrNotGetable should be returned by DataSource.Get if the data source
	// does not support get.
	ErrNotGetable = errors.New("get not supported, data source is watch-only")

	// ErrNotWatchable should be returned by DataSource.Get if the data source
	// does not support watch.
	ErrNotWatchable = errors.New("watch not supported, data source is get-only")
)

// DataSource is the low-level interface implemented by all data sources.
//
// On formats, implementanions:
// - must implement format == "json"
// - should implement format == "text"
// - may implement any other formats that make sense for them
//
// Further implementation requirements are listed within the interface
// functions' documentation.
type DataSource interface {
	// Name returns the unique identifier (GWR noun path) for this source.
	Name() string

	// Formats returns a list of supported format name strings.  All
	// implemented formats must be listed.  At least "json" must be supported.
	Formats() []string

	// Attrs returnts arbitrary descriptive data about the data source.  This
	// data is exposed be the /meta/nouns data source.
	//
	// TODO: standardize and document common fields; current ideas include:
	// - affording get-only or watch-only
	// - affording sampling config (%-age, N-per-t, etc)
	// - whether this data source is lossy (likely the default) or whether
	//   attempts should be taken to drop no item (opt-in edge case); this
	//   could be used internally at least to switch between an implementation
	//   that drops data when buffers fill, or blocks and provides back
	//   pressure.
	Attrs() map[string]interface{}

	// Get implementations:
	// - may return ErrNotGetable if get is not supported by the data source
	// - if the format is not support then ErrUnsupportedFormat must be returned
	// - must format and write any available data to the supplied io.Writer
	// - should return any write error
	Get(format string, w io.Writer) error

	// Watch implementations:
	// - may return ErrNotWatchable if watch is not supported by the data
	//   source
	// - if the format is not support then ErrUnsupportedFormat must be returned
	// - may format and write initial data to the supplied io.Writer; any
	//   initial write error must be returned
	// - may retain and write to the supplied io.Writer indefinately until it
	//   returns a write error
	//
	// Note that at this level, data sources are responsible for both item
	// marshalling and stream framing.
	//
	// Framing for the required "json" format is as follows:
	// - JSON must be encoded in compact (no intermediate whitespace) form
	// - each JSON record must be separated by a newline "\n"
	//
	// Framing for the required "text" format is as follows:
	// - any initial stream data should be followed by a blank line (double new
	//   line "\n\n")
	// - items should be separated by newlines
	// - if an item's text form takes up multiple lines, it should either use
	//   indentation or a double blank line to separate itself from siblings
	Watch(format string, w io.Writer) error
}

// DrainableSource is a DataSource that can be drained.  Draining a source
// should flush any unsent data, and then close any remaining Watch writers.
type DrainableSource interface {
	DataSource
	Drain()
}

// TODO: should add a ClosableSource so that DataSources.Remove can close any
// active watchers etc.
