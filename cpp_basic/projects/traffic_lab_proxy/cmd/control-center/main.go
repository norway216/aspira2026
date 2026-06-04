package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
	"github.com/aspira2026/traffic_lab_proxy/internal/controlcenter/api"
	"github.com/aspira2026/traffic_lab_proxy/internal/controlcenter/db"
	"github.com/aspira2026/traffic_lab_proxy/internal/controlcenter/scheduler"
)

func main() {
	// Setup structured logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Load configuration
	cfg := common.LoadControlCenterConfig()
	slog.Info("starting control center",
		"listen", cfg.ListenAddr,
		"db_driver", cfg.DatabaseDriver,
		"db_dsn", cfg.DatabaseDSN,
	)

	// Initialize database
	database, err := db.NewDBFromConfig(cfg)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := database.Ping(context.Background()); err != nil {
		slog.Error("database ping failed", "error", err)
		os.Exit(1)
	}
	slog.Info("database connected")

	// Initialize scheduler engine
	eng := scheduler.NewEngine(database, cfg.SchedulerStrategy)
	slog.Info("scheduler initialized", "strategy", eng.CurrentStrategy())

	// Create router
	router := api.NewRouter(database, cfg, eng)

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start stale node detector
	go staleNodeDetector(ctx, database, cfg.NodeStaleTimeout)

	// Start server
	go func() {
		slog.Info("HTTP server starting", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	slog.Info("shutting down...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	slog.Info("control center stopped")
}

// staleNodeDetector periodically marks nodes as offline if they haven't sent
// a heartbeat within the configured timeout.
func staleNodeDetector(ctx context.Context, database db.DB, timeout time.Duration) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := database.MarkStaleNodesOffline(ctx, int(timeout.Seconds()))
			if err != nil {
				slog.Error("stale node detector error", "error", err)
				continue
			}
			if n > 0 {
				slog.Warn("marked stale nodes offline", "count", n)
				// Update Prometheus metrics
				nodes, _ := database.ListNodes(ctx)
				updateNodeMetrics(nodes)
			}
		}
	}
}

func updateNodeMetrics(nodes []*common.Node) {
	online := 0
	offline := 0
	degraded := 0
	for _, n := range nodes {
		switch n.Status {
		case common.NodeStatusOnline:
			online++
		case common.NodeStatusOffline:
			offline++
		case common.NodeStatusDegraded:
			degraded++
		}
	}
	api.NodesOnline.Set(float64(online))
	api.NodesOffline.Set(float64(offline))
	api.NodesDegraded.Set(float64(degraded))
}
