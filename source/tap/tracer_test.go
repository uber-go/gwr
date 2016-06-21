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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/uber-go/gwr/internal/test"
	"github.com/uber-go/gwr/source/tap"
)

func TestTracer_collatz(t *testing.T) {
	tracer := tap.NewTracer("test")
	wat := test.NewWatcher()
	tracer.SetWatcher(wat)
	sc := tracer.Scope("collatzTest").Open()
	n := collatz(5, sc)
	sc.Close(n)
	require.Equal(t, 1, n)
	assert.Equal(t, recodeTimeField(wat.AllStrings()), []string{
		"--> t0 [1::1] collatzTest: ",
		"--> t1 [1:1:2] collatz: 5",
		"<-- t2 [1:1:2] collatz: 16",
		"--> t3 [1:2:3] collatz: 16",
		"<-- t4 [1:2:3] collatz: 8",
		"--> t5 [1:3:4] collatz: 8",
		"<-- t6 [1:3:4] collatz: 4",
		"--> t7 [1:4:5] collatz: 4",
		"<-- t8 [1:4:5] collatz: 2",
		"--> t9 [1:5:6] collatz: 2",
		"<-- t10 [1:5:6] collatz: 1",
		"<-- t11 [1::1] collatzTest: 1",
	})
}

func recodeTimeField(strs []string) []string {
	for n, str := range strs {
		var head string
		rest := str
		for k := 0; k < 5; k++ {
			i := strings.IndexByte(rest, ' ')
			if i < 0 {
				continue
			}
			if len(head) == 0 {
				head = rest[:i]
			}
			rest = rest[i+1:]
		}
		strs[n] = strings.Join([]string{head, rest}, fmt.Sprintf(" t%d ", n))
	}
	return strs
}

func collatz(n int, sc *tap.TraceScope) int {
	if n <= 1 {
		return n
	}
	sc = sc.Sub("collatz").Open(n)
	if n&1 == 1 {
		n = 3*n + 1
		sc.Close(n)
		return collatz(n, sc)
	}
	n = n >> 1
	sc.Close(n)
	return collatz(n, sc)
}
