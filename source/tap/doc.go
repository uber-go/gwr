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
Package tap provides a simple item emitter source and a tracing source.

TODO: break up this package: split emitter from tracer

Emitter

The Emitter source is useful for adding watchable-sources for existing
data in your application, and where it wasn't worth it to define a
specialized source.

All emitter sources will be named like "/tap/...", to emphasize their generic
nature.  The normal use case here is for adding adhoc taps into existing
program data.

Tracer

The Tracer source is useful for tracing program execution.  It can be used
to trace things like function calls, goroutine work units, and anything else
where you can define a scope of work.

All tracer sources will be named like "/tap/trace/...".  A default tracer is
provided at "/tap/trace", however the normal usage pattern is to declare
one-or-more tracers within a package, and to name the appropriately to the area
of the code that is traced.

*/
package tap
