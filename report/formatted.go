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

package report

import (
	"errors"

	"github.com/uber-go/gwr/source"
)

var errReporterClosed = errors.New("reporter closed")
var errRawSource = errors.New("raw sources unsupported, only item data sources")

// FormattedReporter reports observed items from a data source to a formatting
// function.  Only works with sources that support the "text" format.
//
// For example to send a source to stdandard output:
//     rep := NewPrintfReporter(someSource, fmt.Printf)
//     if err := rep.Start(); err != nil {
//         panic(err)
//     }
//     defer rep.Stop()
type FormattedReporter interface {
	source.ItemWatcher
	Source() source.DataSource
	Start() error
	Stop()
}

// logfReporter is a FormattedReporter that targets a log formatting function.
type logfReporter struct {
	src     source.DataSource
	logf    func(format string, args ...interface{})
	stopped bool
}

// NewLogfReporter creates a FormattedReporter around a log formatting
// function.  Log formatting functions are not expected to return an error, and
// are expected to handle their own framing concerns (e.g. adding a trailing
// newline).
func NewLogfReporter(
	src source.DataSource,
	logf func(format string, args ...interface{}),
) FormattedReporter {
	return &logfReporter{
		src:  src,
		logf: logf,
	}
}

// Source returns the target source.
func (rep *logfReporter) Source() source.DataSource {
	return rep.src
}

// Start clears any stop flag, and starts watching the data source.
func (rep *logfReporter) Start() error {
	var err error
	rep.stopped = false
	if isrc, ok := rep.src.(source.ItemDataSource); ok {
		err = isrc.WatchItems("text", rep)
	} else {
		err = errRawSource
	}
	if err != nil {
		rep.stopped = true
	}
	return err
}

// Stop sets a flag internally so that the next HandleItem(s) will return an
// error, removing the watcher resource.
func (rep *logfReporter) Stop() {
	rep.stopped = true
}

// HandleItem outputs the item to the logging function with a source-name
// prefix.
func (rep *logfReporter) HandleItem(item []byte) error {
	if rep.stopped {
		return errReporterClosed
	}
	rep.logf("%s: %s", rep.src.Name(), item)
	return nil
}

// HandleItems outputs all items to the logging function with a source-name
// prefix on each item.
func (rep *logfReporter) HandleItems(items [][]byte) error {
	if rep.stopped {
		return errReporterClosed
	}
	name := rep.src.Name()
	for _, item := range items {
		rep.logf("%s: %s", name, item)
	}
	return nil
}

// printfReporter is a FormattedReporter that targets a log formatting function.
type printfReporter struct {
	src     source.DataSource
	printf  func(format string, args ...interface{}) (int, error)
	stopped bool
}

// NewPrintfReporter creates a new FormattedReporter around a raw
// fmt.Printf-family formatting function.  The formatting function is expected
// to return a number and error in package fmt style.  NewPrintfReporter will
// append a newline to passed format strings, since the print formatting
// function is expected to not do so.
func NewPrintfReporter(
	src source.DataSource,
	printf func(format string, args ...interface{}) (int, error),
) FormattedReporter {
	return &printfReporter{
		src:    src,
		printf: printf,
	}
}

// Source returns the target source.
func (rep *printfReporter) Source() source.DataSource {
	return rep.src
}

// Start clears any stop flag, and starts watching the data source.
func (rep *printfReporter) Start() error {
	var err error
	rep.stopped = false
	if isrc, ok := rep.src.(source.ItemDataSource); ok {
		err = isrc.WatchItems("text", rep)
	} else {
		err = errRawSource
	}
	if err != nil {
		rep.stopped = true
	}
	return err
}

// Stop sets a flag internally so that the next HandleItem(s) will return an
// error, removing the watcher resource.
func (rep *printfReporter) Stop() {
	rep.stopped = true
}

// HandleItem outputs the item to the printf function with a source-name
// prefix and trailing newline.
func (rep *printfReporter) HandleItem(item []byte) error {
	if rep.stopped {
		return errReporterClosed
	}
	if _, err := rep.printf("%s: %s\n", rep.src.Name(), item); err != nil {
		rep.stopped = true
		return err
	}
	return nil
}

// HandleItems outputs all items to the logging function with a source-name
// prefix and trailing newline on each item.
func (rep *printfReporter) HandleItems(items [][]byte) error {
	if rep.stopped {
		return errReporterClosed
	}
	name := rep.src.Name()
	for _, item := range items {
		if _, err := rep.printf("%s: %s\n", name, item); err != nil {
			rep.stopped = true
			return err
		}
	}
	return nil
}
