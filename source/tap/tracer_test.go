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
	assert.Equal(t, []string{
		">>> t0 [1::1] collatzTest: ",
		">>> t1 [1:1:2] collatz: 5",
		"<<< t2 [1:1:2] collatz: 16",
		">>> t3 [1:2:3] collatz: 16",
		"<<< t4 [1:2:3] collatz: 8",
		">>> t5 [1:3:4] collatz: 8",
		"<<< t6 [1:3:4] collatz: 4",
		">>> t7 [1:4:5] collatz: 4",
		"<<< t8 [1:4:5] collatz: 2",
		">>> t9 [1:5:6] collatz: 2",
		"<<< t10 [1:5:6] collatz: 1",
		"<<< t11 [1::1] collatzTest: 1",
	}, recodeTimeField(wat.AllStrings()))
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
