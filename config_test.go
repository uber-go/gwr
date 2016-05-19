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

package gwr_test

import (
	"os"
	"testing"

	"github.com/uber-go/gwr"

	"github.com/stretchr/testify/assert"
)

func TestConfiguredServer(t *testing.T) {
	os.Unsetenv("GWR_LISTEN")
	srv := gwr.NewConfiguredServer(gwr.Config{})
	assert.Equal(t, srv.ListenAddr(), "", "no default listen address")

	os.Setenv("GWR_LISTEN", ":1234")
	srv = gwr.NewConfiguredServer(gwr.Config{})
	assert.Equal(t, srv.ListenAddr(), ":1234", "env variable works")
	os.Unsetenv("GWR_LISTEN")

	srv = gwr.NewConfiguredServer(gwr.Config{ListenAddr: ":0"})
	assert.Equal(t, srv.ListenAddr(), ":0", "server took ListenAddr config")

	assert.NoError(t, srv.Start(), "no start error")
	assert.Equal(t, srv.Start(), gwr.ErrAlreadyStarted, "double start error")
	assert.NotNil(t, srv.Addr(), "non-nil addr after start")
	assert.NoError(t, srv.Stop(), "no stop error")
	assert.Nil(t, srv.Addr(), "nil addr after stop")
	assert.NoError(t, srv.Stop(), "stop is idempotent")
}
