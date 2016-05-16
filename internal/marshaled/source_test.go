package marshaled_test

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/uber-go/gwr/internal/marshaled"
	"github.com/uber-go/gwr/source"
)

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
		fmt.Printf("active <- %v\n", item)
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

	watchit := func() {
		w, err := ps.add()
		require.NoError(t, err)
		require.NoError(t, mds.Watch("json", w))
	}

	// first watcher causes activation
	watchit()
	assert.True(t, tds.hasActivated())

	// observe one
	tds.emit(map[string]interface{}{"hello": "world"})
	ps.assertGotJSON(t, 1, `{"hello":"world"}`)

	// second watcher does not cause activation
	watchit()
	assert.False(t, tds.hasActivated())

	// observe two
	tds.emit(map[string]interface{}{"hello": "world2"})
	ps.assertGotJSON(t, 2, `{"hello":"world2"}`)

	// TODO: further testing has synchronization needs: need to be able to wait
	// for mds to "drain" when it's watcher-less before moving on to next phase
}

type pipeSet struct {
	rs  []*os.File
	scs []*bufio.Scanner
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
