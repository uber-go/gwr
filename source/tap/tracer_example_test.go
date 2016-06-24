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

package tap_test

import (
	"fmt"
	"strings"

	"github.com/uber-go/gwr"
	"github.com/uber-go/gwr/report"
	"github.com/uber-go/gwr/source"
	"github.com/uber-go/gwr/source/tap"
)

var fibTracer = tap.AddNewTracer("fib")

// untracedFib is a classic recursive fibonacci computation
func untracedFib(n int) int {
	if n < 0 {
		return 0
	}
	if n < 2 {
		return 1
	}
	return untracedFib(n-1) + untracedFib(n-2)
}

// tracedFib is a modified copy of untracedFib that uses and passes along a
// tracing scope.
//
// For more complex functions, you can call other methods on scope like:
// - Info(...) to emit intermediate data
// - Error(err, ...) to emit any error about to be returned
// - ErrorName("name", err, ...) to further specify a name describing the path
//   or cause of the error if it would otherwise be unclear
func tracedFib(n int, scope *tap.TraceScope) (r int) {
	scope = scope.Sub("fib").OpenCall(n)
	defer func() { scope.CloseCall(r) }()

	if n < 0 {
		return 0
	}
	if n < 2 {
		return 1
	}
	return tracedFib(n-1, scope) + tracedFib(n-2, scope)
}

// fib is a wrapper that checks if the fibTracer is active (has any watchers)
// and calls either tracedFib or untracedFib accordingly.
//
// This dual-implementation approach is only one option, you could also:
// - choose to pass along a nil scope, and nil-check it throughout a single
//   implementation path
// - use Tracer.Scope to always create a scope object, all of its record emits
//   will simply go nowhere
func fib(n int) int {
	if scope := fibTracer.MaybeScope("wrapper"); scope != nil {
		scope.Open(n)
		r := tracedFib(n, scope)
		scope.CloseCall(r)
		return r
	}
	return untracedFib(n)
}

func ExampleTracer() {
	// this one won't be traced since there is no watcher yet
	fib(4)

	// this just makes trace ids stable for the test
	tap.ResetTraceID()

	// this causes fibTracer's output to get printed to stdout; the use here is
	// more complicated than you'd have in a real program to get stable test
	// output.
	rep := report.NewPrintfReporter(
		gwr.DefaultDataSources.Get("/tap/trace/fib"),
		(&timeElider{}).printf)
	if err := rep.Start(); err != nil {
		panic(err)
	}
	defer rep.Stop()

	// this one will be traced since there's now a watcher
	fib(5)

	// this flushes and stops all watchers on the reported source; otherwise
	// this function returns too quickly for even one of the emitted trace
	// items to have been printed.
	rep.Source().(source.DrainableSource).Drain()

	// this one won't be traced since Drain deactivated the tracer
	fib(6)

	// Output:
	// /tap/trace/fib: --> DATE TIME_0 [1::1] wrapper: 5
	// /tap/trace/fib: --> DATE TIME_1 [1:1:2] fib(5)
	// /tap/trace/fib: --> DATE TIME_2 [1:2:3] fib(4)
	// /tap/trace/fib: --> DATE TIME_3 [1:3:4] fib(3)
	// /tap/trace/fib: --> DATE TIME_4 [1:4:5] fib(2)
	// /tap/trace/fib: --> DATE TIME_5 [1:5:6] fib(1)
	// /tap/trace/fib: <-- DATE TIME_6 [1:5:6] return 1
	// /tap/trace/fib: --> DATE TIME_7 [1:5:7] fib(0)
	// /tap/trace/fib: <-- DATE TIME_8 [1:5:7] return 1
	// /tap/trace/fib: <-- DATE TIME_9 [1:4:5] return 2
	// /tap/trace/fib: --> DATE TIME_10 [1:4:8] fib(1)
	// /tap/trace/fib: <-- DATE TIME_11 [1:4:8] return 1
	// /tap/trace/fib: <-- DATE TIME_12 [1:3:4] return 3
	// /tap/trace/fib: --> DATE TIME_13 [1:3:9] fib(2)
	// /tap/trace/fib: --> DATE TIME_14 [1:9:10] fib(1)
	// /tap/trace/fib: <-- DATE TIME_15 [1:9:10] return 1
	// /tap/trace/fib: --> DATE TIME_16 [1:9:11] fib(0)
	// /tap/trace/fib: <-- DATE TIME_17 [1:9:11] return 1
	// /tap/trace/fib: <-- DATE TIME_18 [1:3:9] return 2
	// /tap/trace/fib: <-- DATE TIME_19 [1:2:3] return 5
	// /tap/trace/fib: --> DATE TIME_20 [1:2:12] fib(3)
	// /tap/trace/fib: --> DATE TIME_21 [1:12:13] fib(2)
	// /tap/trace/fib: --> DATE TIME_22 [1:13:14] fib(1)
	// /tap/trace/fib: <-- DATE TIME_23 [1:13:14] return 1
	// /tap/trace/fib: --> DATE TIME_24 [1:13:15] fib(0)
	// /tap/trace/fib: <-- DATE TIME_25 [1:13:15] return 1
	// /tap/trace/fib: <-- DATE TIME_26 [1:12:13] return 2
	// /tap/trace/fib: --> DATE TIME_27 [1:12:16] fib(1)
	// /tap/trace/fib: <-- DATE TIME_28 [1:12:16] return 1
	// /tap/trace/fib: <-- DATE TIME_29 [1:2:12] return 3
	// /tap/trace/fib: <-- DATE TIME_30 [1:1:2] return 8
	// /tap/trace/fib: <-- DATE TIME_31 [1::1] return 8
}

// timeElider is here just to normalize output for the test
type timeElider struct {
	n int
}

func (te *timeElider) printf(format string, args ...interface{}) (int, error) {
	s := fmt.Sprintf(format, args...)
	if fields := strings.Split(s, " "); len(fields) > 6 {
		fields[2] = fmt.Sprintf("DATE TIME_%d", te.n)
		te.n++
		copy(fields[3:], fields[6:])
		fields = fields[:len(fields)-3]
		s = strings.Join(fields, " ")
	}
	return fmt.Printf(s)
}
