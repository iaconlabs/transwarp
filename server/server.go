// Package server provides a production-ready HTTP server wrapper with support
// for graceful shutdown and configuration defaults.
package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/profe-ajedrez/transwarp/router"
)

const (
	defaultTimeout      = 5 * time.Second
	defaultReadTimeout  = 10 * time.Second
	defaultWriteTimeout = 15 * time.Second
	defaultIdleTimeout  = 60 * time.Second
)

// Config defines the timeouts and address for the HTTP server.
type Config struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// Server wraps the standard [http.Server] to work with Transwarp adapters.
type Server struct {
	cfg        Config
	httpServer *http.Server
	adapter    router.Router
	ln         net.Listener
	addr       string
	mu         sync.RWMutex
	ready      chan struct{}
}

// New initializes a new Server with the given config and router adapter.
func New(cfg Config, adapter router.Router) *Server {
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = defaultReadTimeout
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = defaultWriteTimeout
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = defaultIdleTimeout
	}

	s := &Server{
		cfg:     cfg,
		adapter: adapter,
		ready:   make(chan struct{}),
	}

	s.httpServer = &http.Server{
		Addr:         cfg.Addr,
		Handler:      adapter,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return s
}

// Start runs the HTTP server. This call is blocking until the server is closed.
func (s *Server) Start(ctx context.Context) error {
	// 1. Define a ListenConfig to perform context-aware listening.
	lc := net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp", s.cfg.Addr)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.ln = ln
	s.addr = ln.Addr().String()
	s.mu.Unlock()

	close(s.ready) // Notify that Addr() is now available

	// 2. Pass the listener to the internal http.Server
	err = s.httpServer.Serve(ln)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the server without interrupting active connections.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Addr returns the network address the server is listening on.
// It waits for the server to be ready, making it safe for use in tests with dynamic ports.
func (s *Server) Addr() string {
	select {
	case <-s.ready:
		// Server initialized
	case <-time.After(defaultTimeout):
		// Safety timeout
		return ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.addr
}
