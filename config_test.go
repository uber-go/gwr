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
