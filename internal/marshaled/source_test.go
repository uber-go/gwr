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

package marshaled_test

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/uber-go/gwr/internal/marshaled"
	"github.com/uber-go/gwr/source"
)

var (
	debug = flag.Bool("debug", false, "Enable debug prints")
	delay = flag.Duration("delay", 20*time.Microsecond, "How long to delay before generating a new item")
)

var words = []string{"the", "quick", "brown", "fox", "jump", "over", "the", "lazy", "hound"}

type benchData struct {
	Mess string `json:"word"`
}

type testDataSource struct {
	watcher   source.GenericDataWatcher
	activated chan struct{}
}

func (tds *testDataSource) Name() string {
	return "/test"
}

func (tds *testDataSource) TextTemplate() *template.Template {
	return nil
}

func (tds *testDataSource) SetWatcher(watcher source.GenericDataWatcher) {
	tds.watcher = watcher
}

func (tds *testDataSource) Activate() {
	tds.activated <- struct{}{}
}

func (tds *testDataSource) emit(item interface{}) {
	if tds.watcher.Active() {
		tds.watcher.HandleItem(item)
	}
}

func (tds *testDataSource) hasActivated() bool {
	select {
	case <-tds.activated:
		return true
	default:
		return false
	}
}

func TestDataSource_Watch_activation(t *testing.T) {
	tds := &testDataSource{}
	tds.activated = make(chan struct{}, 1)
	mds := marshaled.NewDataSource(tds, nil)

	var ps pipeSet
	defer ps.close()

	require.NoError(t, ps.watch(mds, "json"))
	assert.True(t, tds.hasActivated(), "first watcher causes activation")

	// observe one
	tds.emit(map[string]interface{}{"hello": "world"})
	ps.assertGotJSON(t, 1, `{"hello":"world"}`)

	require.NoError(t, ps.watch(mds, "json"))
	assert.False(t, tds.hasActivated(), "second watcher does not cause activation")

	// observe two
	tds.emit(map[string]interface{}{"hello": "world2"})
	ps.assertGotJSON(t, 2, `{"hello":"world2"}`)

	// TODO: further testing has synchronization needs: need to be able to wait
	// for mds to "drain" when it's watcher-less before moving on to next phase
}

func BenchmarkDataSource_Watch_json(b *testing.B) {
	// TODO: parallel variant(s)

	tds := &testDataSource{}
	tds.activated = make(chan struct{}, 1)
	mds := marshaled.NewDataSource(tds, nil)

	var ps pipeSet
	defer ps.close()

	lines := make(chan string, 1000000)

	var sg sync.WaitGroup
	sg.Add(1)

	go func() {
		reconn := 0
		require.NoError(b, ps.watch(mds, "json"))
		sg.Done()
		for {
			sc := ps.scs[len(ps.scs)-1]
			for sc.Scan() {
				lines <- sc.Text()
			}
			if ps.closed {
				break
			}
			reconn++
			require.NoError(b, ps.watch(mds, "json"))
		}
		b.Logf("line scanner done, %d reconns\n", reconn)
		close(lines)
	}()

	go func() {
		sg.Wait()
		for i := 0; i < b.N; i++ {
			if *debug {
				fmt.Printf("put %d\n", i)
			}
			word := words[i%len(words)]
			mess := fmt.Sprintf("[%d]: %q", i, word)
			tds.emit(benchData{Mess: mess})
			<-time.After(*delay)
		}
	}()

	expected := b.N
	got := 0

	// 10% more than N * delay
	to := time.After(time.Duration(int(1.1 * float64(int(*delay)*b.N))))

loop:
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				break loop
			}
			require.True(b, line[0] == '{' && line[len(line)-1] == '}', "looks like a json object")
			if *debug {
				fmt.Printf("got %q\n", line)
			}
			// done if we've seen expected number of lines
			got++
			if got >= expected {
				break loop
			}

		case <-to:
			break loop
		}

	}

	ps.close()
	p := 100.0 * float64(got) / float64(expected)
	b.Logf("saw %d/%d (%.1f%%) records\n", got, expected, p)
}

type pipeSet struct {
	rs     []*os.File
	scs    []*bufio.Scanner
	closed bool // TODO: atomic?
}

func (ps *pipeSet) watch(src source.DataSource, format string) error {
	w, err := ps.add()
	if err == nil {
		err = src.Watch(format, w)
	}
	return err
}

func (ps *pipeSet) add() (*os.File, error) {
	r, w, err := os.Pipe()
	if err == nil {
		ps.rs = append(ps.rs, r)
		ps.scs = append(ps.scs, bufio.NewScanner(r))
	}
	return w, err
}

func (ps *pipeSet) closeOne(i int) error {
	if i >= len(ps.rs) {
		return fmt.Errorf("invalid index")
	}
	r := ps.rs[i]
	ps.rs = append(ps.rs[:i], ps.rs[i+1:]...)
	ps.scs = append(ps.scs[:i], ps.scs[i+1:]...)
	return r.Close()
}

func (ps *pipeSet) close() {
	ps.closed = true
	for _, r := range ps.rs {
		r.Close()
	}
}

func (ps *pipeSet) assertGotJSON(t *testing.T, n int, expected string, msgAndArgs ...interface{}) {
	var m int
	for _, sc := range ps.scs {
		assertJSONScanLine(t, sc, expected, msgAndArgs...)
		m++
	}
	assert.Equal(t, n, m, msgAndArgs...)
}

func assertJSONScanLine(t *testing.T, sc *bufio.Scanner, expected string, msgAndArgs ...interface{}) {
	if !sc.Scan() {
		assert.Fail(t, "expected to scan a JSON line", msgAndArgs...)
	} else {
		expected = strings.Join([]string{expected, "\n"}, "")
		assert.JSONEq(t, expected, sc.Text(), msgAndArgs...)
	}
	assert.NoError(t, sc.Err())
}
