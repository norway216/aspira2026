package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
	"github.com/aspira2026/traffic_lab_proxy/internal/proxynode/auth"
	"github.com/aspira2026/traffic_lab_proxy/internal/proxynode/counter"
	"github.com/aspira2026/traffic_lab_proxy/internal/proxynode/limiter"
	"github.com/aspira2026/traffic_lab_proxy/internal/proxynode/policy"
	"github.com/aspira2026/traffic_lab_proxy/internal/proxynode/proxy"
	"github.com/aspira2026/traffic_lab_proxy/internal/proxynode/reporter"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := common.LoadProxyNodeConfig()

	slog.Info("starting proxy node",
		"node_id", cfg.NodeID,
		"listen", cfg.ListenAddr,
		"control_center", cfg.ControlCenterURL,
		"metrics", cfg.MetricsListenAddr,
	)

	// ── Initialize components ──────────────────────────

	authenticator := auth.NewAuthenticator()
	rateLimiter := limiter.NewPerUserLimiter()
	trafficCounter := counter.NewTrafficCounter()
	connManager := proxy.NewConnectionManager(cfg.MaxConnections)
	policyCache := policy.NewCache(cfg.NodeID)

	proxyHandler := proxy.NewProxy(
		authenticator,
		rateLimiter,
		trafficCounter,
		connManager,
		proxy.ProxyConfig{
			BufferSize:     cfg.ProxyBufferSize,
			IdleTimeout:    5 * time.Minute,
			ConnectTimeout: cfg.TargetConnectTimeout,
			MaxConnections: cfg.MaxConnections,
		},
	)

	// ── Start reporters ───────────────────────────────

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Policy fetcher
	policyFetcher := policy.NewPolicyFetcher(
		cfg.ControlCenterURL,
		cfg.NodeID,
		cfg.NodeSecret,
		policyCache,
		authenticator,
		rateLimiter,
		cfg.PolicyFetchInterval,
	)
	go policyFetcher.Start(ctx)

	// Heartbeat reporter
	heartbeatReporter := reporter.NewHeartbeatReporter(
		cfg.ControlCenterURL,
		cfg.NodeID,
		cfg.NodeSecret,
		connManager,
		trafficCounter,
		cfg.HeartbeatInterval,
	)
	go heartbeatReporter.Start(ctx)

	// Traffic reporter
	trafficReporter := reporter.NewTrafficReporter(
		cfg.ControlCenterURL,
		cfg.NodeID,
		cfg.NodeSecret,
		trafficCounter,
		cfg.TrafficReportInterval,
	)
	go trafficReporter.Start(ctx)

	// ── Register with control center ──────────────────

	// Wait for policy fetch to complete first
	time.Sleep(500 * time.Millisecond)

	// The registration happens via the heartbeat mechanism - the first
	// heartbeat will trigger node registration on the control center side.

	// ── Start Prometheus metrics server ───────────────

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	metricsSrv := &http.Server{
		Addr:    cfg.MetricsListenAddr,
		Handler: metricsMux,
	}

	go func() {
		slog.Info("metrics server starting", "addr", cfg.MetricsListenAddr)
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server error", "error", err)
		}
	}()

	// ── Start TCP proxy server ────────────────────────

	proxySrv := proxy.NewServer(proxyHandler, proxy.ServerConfig{
		ListenAddr:     cfg.ListenAddr,
		MaxConnections: cfg.MaxConnections,
	})

	// Handle graceful shutdown
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-sigCtx.Done()
		slog.Info("shutting down proxy node...")

		// Cancel context for reporters
		cancel()

		// Shutdown proxy server with 30s drain
		proxySrv.Shutdown(30 * time.Second)

		// Shutdown metrics server
		metricsShutdownCtx, metricsCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer metricsCancel()
		metricsSrv.Shutdown(metricsShutdownCtx)
	}()

	// Start proxy server (blocks until shutdown)
	if err := proxySrv.Start(ctx); err != nil {
		slog.Error("proxy server error", "error", err)
		os.Exit(1)
	}

	slog.Info("proxy node stopped")
}
