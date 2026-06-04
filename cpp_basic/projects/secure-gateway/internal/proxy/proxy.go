package proxy

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/secure-gateway/internal/metrics"
	"github.com/secure-gateway/pkg/logger"
)

// ProxyConfig holds configuration for the traffic proxy.
type ProxyConfig struct {
	ListenAddr   string
	TLSEnabled   bool
	TLSCert      string
	TLSKey       string
	IdleTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	TargetAddr   string // Default target address (or use policy-based routing)
}

// Proxy handles TCP/TLS traffic forwarding.
type Proxy struct {
	config    ProxyConfig
	stats     *metrics.ConnectionStats
	activeConns sync.Map
	listener     net.Listener
	shutdown     chan struct{}
	connCount    int64
}

// NewProxy creates a new TCP/TLS proxy.
func NewProxy(config ProxyConfig, stats *metrics.ConnectionStats) *Proxy {
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 120 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 30 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 30 * time.Second
	}

	return &Proxy{
		config:   config,
		stats:    stats,
		shutdown: make(chan struct{}),
	}
}

// Start begins listening for incoming connections.
func (p *Proxy) Start(ctx context.Context) error {
	var err error

	if p.config.TLSEnabled {
		cert, err := tls.LoadX509KeyPair(p.config.TLSCert, p.config.TLSKey)
		if err != nil {
			return err
		}
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		p.listener, err = tls.Listen("tcp", p.config.ListenAddr, tlsCfg)
		logger.Info("TLS proxy listening", "addr", p.config.ListenAddr)
	} else {
		p.listener, err = net.Listen("tcp", p.config.ListenAddr)
		logger.Info("TCP proxy listening", "addr", p.config.ListenAddr)
	}
	if err != nil {
		return err
	}

	go p.acceptLoop(ctx)
	return nil
}

// Stop gracefully shuts down the proxy.
func (p *Proxy) Stop() {
	close(p.shutdown)
	if p.listener != nil {
		p.listener.Close()
	}
}

// ActiveConnCount returns the current number of active connections.
func (p *Proxy) ActiveConnCount() int64 {
	return atomic.LoadInt64(&p.connCount)
}

func (p *Proxy) acceptLoop(ctx context.Context) {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			select {
			case <-p.shutdown:
				return
			default:
				logger.Error("Proxy accept error", "error", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}

		select {
		case <-p.shutdown:
			conn.Close()
			return
		default:
		}

		atomic.AddInt64(&p.connCount, 1)
		p.stats.IncActiveConns()
		go p.handleConnection(conn)
	}
}

func (p *Proxy) handleConnection(clientConn net.Conn) {
	defer func() {
		clientConn.Close()
		atomic.AddInt64(&p.connCount, -1)
		p.stats.DecActiveConns()
	}()

	// Set initial deadlines
	if p.config.ReadTimeout > 0 {
		clientConn.SetReadDeadline(time.Now().Add(p.config.ReadTimeout))
	}

	// Generate a connection ID for logging
	connID := time.Now().UnixNano()
	remoteAddr := clientConn.RemoteAddr().String()
	logger.Debug("New connection", "id", connID, "remote", remoteAddr)

	// In a full implementation, this is where we would:
	// 1. Authenticate the client (verify token)
	// 2. Look up the target service based on policy
	// 3. Dial the target

	// For now, use default target if configured, otherwise echo for testing
	targetAddr := p.config.TargetAddr
	if targetAddr == "" {
		// Echo mode for testing
		p.handleEcho(clientConn, connID)
		return
	}

	// Dial target
	targetConn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		logger.Error("Failed to dial target", "target", targetAddr, "error", err)
		p.stats.IncErrors()
		return
	}
	defer targetConn.Close()

	// Reset read timeout for streaming
	clientConn.SetReadDeadline(time.Time{})

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		written, err := p.copyWithStats(targetConn, clientConn, true)
		if err != nil && err != io.EOF {
			logger.Debug("Copy client->target error", "id", connID, "error", err)
		}
		logger.Debug("Client->Target transfer complete", "id", connID, "bytes", written)
	}()

	go func() {
		defer wg.Done()
		written, err := p.copyWithStats(clientConn, targetConn, false)
		if err != nil && err != io.EOF {
			logger.Debug("Copy target->client error", "id", connID, "error", err)
		}
		logger.Debug("Target->Client transfer complete", "id", connID, "bytes", written)
	}()

	wg.Wait()
	logger.Debug("Connection closed", "id", connID)
}

func (p *Proxy) copyWithStats(dst io.Writer, src io.Reader, isRx bool) (int64, error) {
	buf := make([]byte, 32*1024) // 32KB buffer
	var total int64

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			if nw > 0 {
				total += int64(nw)
				if isRx {
					p.stats.AddRxBytes(int64(nw))
				} else {
					p.stats.AddTxBytes(int64(nw))
				}
			}
			if ew != nil {
				return total, ew
			}
			if nr != nw {
				return total, io.ErrShortWrite
			}
		}
		if er != nil {
			return total, er
		}
	}
}

func (p *Proxy) handleEcho(conn net.Conn, connID int64) {
	logger.Debug("Echo mode for connection", "id", connID)
	buf := make([]byte, 32*1024)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			p.stats.AddRxBytes(int64(n))
			wn, werr := conn.Write(buf[:n])
			if wn > 0 {
				p.stats.AddTxBytes(int64(wn))
			}
			if werr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

// GetStats returns a snapshot of connection statistics.
func (p *Proxy) GetStats() (activeConns, rxPerSec, txPerSec int64) {
	return p.stats.Snapshot()
}