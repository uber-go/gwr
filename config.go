package gwr

import (
	"errors"
	"net"
	"os"
	"sync/atomic"

	"github.com/uber-common/stacked"
)

var (
	// ErrAlreadyConfigured is returned by gwr.Configure when called more than
	// once.
	ErrAlreadyConfigured = errors.New("gwr already configured")

	// ErrAlreadyStarted is returned by ConfiguredServer.Start if the server is
	// already listening.
	ErrAlreadyStarted = errors.New("gwr server already started")
)

// Config defines configuration for GWR.  For now this only defines server
// configuration; however once we have reporting support we'll add something
// ReportingCofiguration here.
type Config struct {
	// Enabled controls whether GWR is enabled or not, it defaults true.
	// Currently this only controls whether ConfiguredServer starting.
	Enabled *bool `yaml:"enabled"`

	// ListenAddr controls what address ConfiguredServer will listen on.  It is
	// superceded by the $GWR_LISTEN environment variable, and defaults to
	// ":4040" if neither is set.
	ListenAddr string `yaml:"listen"`
}

var theServer *ConfiguredServer

// Configure sets up the gwr library and starts any resources (like a listening
// server) if enabled.
// - if nil config is passed, it's a convenience for &gwr.Config{}
// - if called more than once, ErrAlreadyConfigured is returned
// - otherwise any ConfiguredServer.Start error is returned.
func Configure(config *Config) error {
	if theServer != nil {
		return ErrAlreadyConfigured
	}
	if config == nil {
		config = &Config{}
	}
	theServer = NewConfiguredServer(*config)
	return theServer.Start()
}

// Enabled returns true if the gwr library is configured and enabled.
func Enabled() bool {
	if theServer == nil {
		return false
	}
	return theServer.Enabled()
}

// DefaultServer returns the configured gwr server, or nil if Configure
// hasn't been called yet.
func DefaultServer() *ConfiguredServer {
	return theServer
}

type serverConfig struct {
	enabled    bool
	listenAddr string
}

var defaultServerConfig = serverConfig{
	enabled:    true,
	listenAddr: ":4040",
}

// ConfiguredServer manages the lifecycle of a configured GWR server, as
// created by gwr.NewServer.
type ConfiguredServer struct {
	config   serverConfig
	stacked  stacked.Server
	ln       net.Listener
	stopping uint32
	done     chan error
}

// NewConfiguredServer creates a new ConfiguredServer for a given config.
func NewConfiguredServer(cfg Config) *ConfiguredServer {
	srv := &ConfiguredServer{
		config:  defaultServerConfig,
		stacked: NewServer(DefaultDataSources),
	}

	if cfg.Enabled != nil {
		srv.config.enabled = *cfg.Enabled
	}

	if envListen := os.Getenv("GWR_LISTEN"); envListen != "" {
		srv.config.listenAddr = envListen
	} else if cfg.ListenAddr != "" {
		srv.config.listenAddr = cfg.ListenAddr
	}

	return srv
}

// Enabled return true if the server is enabled.
func (srv *ConfiguredServer) Enabled() bool {
	return srv.config.enabled
}

// ListenAddr returns the configured listen address string.
func (srv *ConfiguredServer) ListenAddr() string {
	return srv.config.listenAddr
}

// Addr returns the current listening address, if any.
func (srv *ConfiguredServer) Addr() net.Addr {
	if srv.ln == nil {
		return nil
	}
	return srv.ln.Addr()
}

// Start starts the server by creating the listener and a server goroutine to
// accept connections.
// - if not enabled, noops and returns nil
// - if already listening, returns ErrAlreadyStarted
// - otherwise any net.Listen error is returned.
func (srv *ConfiguredServer) Start() error {
	if !srv.config.enabled {
		return nil
	}

	if srv.ln != nil {
		return ErrAlreadyStarted
	}

	ln, err := net.Listen("tcp", srv.config.listenAddr)
	if err != nil {
		return err
	}

	srv.ln = ln
	srv.done = make(chan error, 1)
	go func(ln net.Listener, done chan<- error) {
		err := srv.stacked.Serve(ln)
		if atomic.LoadUint32(&srv.stopping) == 0 {
			done <- err
		} else {
			done <- nil
		}
	}(srv.ln, srv.done)
	return nil
}

// Stop closes the current listener and shuts down the server goroutine started
// by Start (if any).
func (srv *ConfiguredServer) Stop() error {
	if srv.ln == nil {
		return nil
	}
	if !atomic.CompareAndSwapUint32(&srv.stopping, 0, 1) {
		return nil
	}
	ln, done := srv.ln, srv.done
	srv.ln, srv.done = nil, nil
	err := ln.Close()
	if serveErr := <-done; err == nil && serveErr != nil {
		err = serveErr
	}
	atomic.CompareAndSwapUint32(&srv.stopping, 1, 0)
	return err
}
