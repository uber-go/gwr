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

/*

Package gwr provides on demand operational data sources in Go.  A typical use
is adding in-depth debug tracing to your program that only turns on when there
are any active consumer(s).

Basics

A gwr data source is a named subset data that is Get-able and/or Watch-able.
The gwr library can then build reporting on top of Watch-able sources, or by
falling back to polling Get-able sources.

For example a request log source would be naturally Watch-able for future
requests as they come in.  An implementation could go further and add a
"last-10" buffer to also become Get-able.

Integrating

To bootstrap gwr and start its server listening on port 4040:

	gwr.Configure(&gwr.Config{ListenAddr: ":4040"})


GWR also adds a handler to the default http server; so if you already have a
default http server like:

	log.Fatal(http.ListenAndServe(":8080", nil))

Then gwr will already be accessible at "/gwr/..." on port 8080; you should
still call gwr.Configure:

	gwr.Configure(nil)

*/
package gwr
