package proxy

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"
)

// Server is the TCP proxy server that listens for client connections.
type Server struct {
	listener net.Listener
	proxy    *Proxy
	cfg      ServerConfig
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// ServerConfig holds the server-level configuration.
type ServerConfig struct {
	ListenAddr     string
	MaxConnections int
}

// NewServer creates a new proxy server.
func NewServer(proxy *Proxy, cfg ServerConfig) *Server {
	return &Server{
		proxy: proxy,
		cfg:   cfg,
	}
}

// Start begins listening for client connections. Blocks until the context is cancelled
// or an error occurs on the listener.
func (s *Server) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	var err error
	s.listener, err = net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return err
	}

	slog.Info("proxy server listening", "addr", s.cfg.ListenAddr)

	// Accept loop
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return nil // Graceful shutdown
			default:
				slog.Error("accept error", "error", err)
				continue
			}
		}

		// Handle connection in a goroutine
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.proxy.HandleConnection(s.ctx, conn)
			conn.Close()
		}()
	}
}

// Shutdown gracefully stops the server. It stops accepting new connections,
// then waits for active connections to drain within the given deadline.
func (s *Server) Shutdown(deadline time.Duration) error {
	slog.Info("proxy server shutting down...")

	// Stop accepting new connections
	if s.cancel != nil {
		s.cancel()
	}

	if s.listener != nil {
		s.listener.Close()
	}

	// Wait for active connections with deadline
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("all connections drained gracefully")
	case <-time.After(deadline):
		slog.Warn("shutdown deadline exceeded, forcing close",
			"remaining_connections", s.proxy.connMgr.CurrentConnections())
	}

	return nil
}

// ListenAddr returns the address the server is listening on.
func (s *Server) ListenAddr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.cfg.ListenAddr
}
