package proxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/aspira2026/traffic_lab_proxy/internal/proxynode/auth"
	"github.com/aspira2026/traffic_lab_proxy/internal/proxynode/counter"
	"github.com/aspira2026/traffic_lab_proxy/internal/proxynode/limiter"
)

const (
	copyBufferSize       = 32 * 1024
	idleTimeout          = 5 * time.Minute
	targetConnectTimeout = 10 * time.Second
)

// Proxy handles TCP connections through the full lifecycle.
type Proxy struct {
	auth    *auth.Authenticator
	limiter *limiter.PerUserLimiter
	counter *counter.TrafficCounter
	connMgr *ConnectionManager
	cfg     ProxyConfig
}

type ProxyConfig struct {
	BufferSize     int
	IdleTimeout    time.Duration
	ConnectTimeout time.Duration
	MaxConnections int
}

func DefaultProxyConfig() ProxyConfig {
	return ProxyConfig{
		BufferSize:     copyBufferSize,
		IdleTimeout:    idleTimeout,
		ConnectTimeout: targetConnectTimeout,
		MaxConnections: 1000,
	}
}

func NewProxy(authHandler *auth.Authenticator, rateLimiter *limiter.PerUserLimiter,
	trafficCounter *counter.TrafficCounter, connManager *ConnectionManager,
	cfg ProxyConfig) *Proxy {

	return &Proxy{
		auth:    authHandler,
		limiter: rateLimiter,
		counter: trafficCounter,
		connMgr: connManager,
		cfg:     cfg,
	}
}

func (p *Proxy) HandleConnection(ctx context.Context, clientConn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("proxy handler panic", "panic", r)
			ConnectionErrors.Inc()
		}
	}()

	// Acquire connection slot
	connID, err := p.connMgr.Acquire(ctx)
	if err != nil {
		ConnectionErrors.Inc()
		clientConn.Close()
		return
	}
	defer p.connMgr.Release(connID)

	clientConn.SetDeadline(time.Now().Add(p.cfg.IdleTimeout))

	// Step 1: Authenticate
	userID, target, err := p.auth.Authenticate(clientConn)
	if err != nil {
		slog.Warn("authentication failed", "error", err, "remote", clientConn.RemoteAddr())
		ConnectionErrors.Inc()
		writeProxyError(clientConn, "401 Unauthorized", err.Error())
		return
	}
	p.connMgr.SetUserID(connID, userID)
	p.connMgr.SetTarget(connID, target)

	// Step 2: Check connection limit
	if !p.limiter.AllowConnection(userID) {
		ConnectionErrors.Inc()
		writeProxyError(clientConn, "429 Too Many Requests", "connection limit exceeded")
		return
	}
	defer p.limiter.ReleaseConnection(userID)
	p.counter.AddConnection(userID)

	// Step 3: Connect to target
	targetConn, err := net.DialTimeout("tcp", target, p.cfg.ConnectTimeout)
	if err != nil {
		slog.Warn("target connect failed", "target", target, "error", err)
		ConnectionErrors.Inc()
		writeProxyError(clientConn, "502 Bad Gateway", fmt.Sprintf("cannot connect to %s: %v", target, err))
		return
	}
	defer targetConn.Close()

	// Step 4: Send 200
	if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		return
	}

	clientConn.SetDeadline(time.Time{})
	targetConn.SetDeadline(time.Time{})

	// Step 5: Bidirectional copy
	// Use a non-cancelling context — we don't want one direction to kill the other
	connCtx, _ := context.WithCancel(ctx) // deliberately ignoring cancel

	var wg sync.WaitGroup
	bufSize := p.cfg.BufferSize

	// Upload: client -> target
	wg.Add(1)
	go func() {
		defer wg.Done()
		cr := p.counter.NewCountingReader(userID, clientConn, true)
		buf := make([]byte, bufSize)
		_, copyErr := io.CopyBuffer(targetConn, cr, buf)
		if copyErr != nil && copyErr != io.EOF && !isConnectionClosed(copyErr) {
			slog.Debug("upload finished", "error", copyErr)
		}
		// Signal half-close on target
		if tc, ok := targetConn.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}()

	// Download: target -> client (with rate limiting)
	wg.Add(1)
	go func() {
		defer wg.Done()
		cr := p.counter.NewCountingReader(userID, targetConn, false)
		lw := p.limiter.NewRateLimitedWriter(connCtx, userID, clientConn)
		buf := make([]byte, bufSize)
		_, copyErr := io.CopyBuffer(lw, cr, buf)
		if copyErr != nil && copyErr != io.EOF && !isConnectionClosed(copyErr) {
			slog.Debug("download finished", "error", copyErr)
		}
	}()

	wg.Wait()
	CurrentConnections.Set(float64(p.connMgr.CurrentConnections()))
}

func writeProxyError(conn net.Conn, status, msg string) {
	response := fmt.Sprintf("HTTP/1.1 %s\r\nContent-Type: text/plain\r\n\r\n%s\r\n", status, msg)
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	conn.Write([]byte(response))
}

func isConnectionClosed(err error) bool {
	if err == nil || err == io.EOF {
		return true
	}
	if opErr, ok := err.(*net.OpError); ok {
		return opErr.Err.Error() == "use of closed network connection"
	}
	return false
}

var _ = context.Canceled
