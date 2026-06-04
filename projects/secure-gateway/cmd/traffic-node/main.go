package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/secure-gateway/internal/config"
	"github.com/secure-gateway/internal/metrics"
	"github.com/secure-gateway/internal/model"
	"github.com/secure-gateway/internal/proxy"
	"github.com/secure-gateway/pkg/logger"
)

func main() {
	configPath := flag.String("config", "configs/traffic-node.yaml", "Path to config file")
	flag.Parse()

	// Load config
	cfg, err := config.LoadTrafficNodeConfig(*configPath)
	if err != nil {
		// Try alternate path
		cfg, err = config.LoadTrafficNodeConfig("/app/configs/traffic-node.yaml")
		if err != nil {
			logger.Init("info", "")
			logger.Fatal("Failed to load config", "error", err)
		}
	}

	// Init logger
	logger.Init(cfg.Log.Level, cfg.Log.File)
	defer logger.Sync()

	logger.Info("Starting Traffic Node",
		"node_id", cfg.Node.NodeID,
		"region", cfg.Node.Region,
		"listen", cfg.Server.Listen,
	)

	// Init connection stats
	stats := metrics.NewConnectionStats()

	// Register with control plane
	registerNode(cfg)

	// Start heartbeat
	go heartbeatLoop(cfg, stats)

	// Start proxy
	idleTimeout, _ := time.ParseDuration(cfg.Limits.IdleTimeout)
	readTimeout, _ := time.ParseDuration(cfg.Limits.ReadTimeout)
	writeTimeout, _ := time.ParseDuration(cfg.Limits.WriteTimeout)

	if readTimeout == 0 {
		readTimeout = 30 * time.Second
	}
	if writeTimeout == 0 {
		writeTimeout = 30 * time.Second
	}

	proxyCfg := proxy.ProxyConfig{
		ListenAddr:   cfg.Server.Listen,
		TLSEnabled:   cfg.Server.TLSEnabled,
		TLSCert:      cfg.Server.TLSCert,
		TLSKey:       cfg.Server.TLSKey,
		IdleTimeout:  idleTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	p := proxy.NewProxy(proxyCfg, stats)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := p.Start(ctx); err != nil {
		logger.Fatal("Failed to start proxy", "error", err)
	}

	// Start metrics HTTP server for Prometheus
	go startMetricsServer(cfg, stats)

	logger.Info("Traffic Node started successfully", "node_id", cfg.Node.NodeID)

	// Wait for shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down Traffic Node...")
	p.Stop()
	cancel()
	logger.Info("Traffic Node stopped")
}

func registerNode(cfg *config.TrafficNodeConfig) {
	req := model.NodeRegisterRequest{
		NodeID:         cfg.Node.NodeID,
		Name:           cfg.Node.Name,
		Region:         cfg.Node.Region,
		PublicAddr:     cfg.Node.PublicAddr,
		Secret:         cfg.Node.Secret,
		MaxConnections: cfg.Node.MaxConnections,
	}

	body, _ := json.Marshal(req)
	url := fmt.Sprintf("%s/api/v1/nodes/register", cfg.Control.Endpoint)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		logger.Warn("Node registration failed (will retry)", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		logger.Info("Node registered successfully", "node_id", cfg.Node.NodeID)
	} else {
		logger.Warn("Node registration returned non-OK", "status", resp.StatusCode)
	}
}

func heartbeatLoop(cfg *config.TrafficNodeConfig, stats *metrics.ConnectionStats) {
	interval, _ := time.ParseDuration(cfg.Control.HeartbeatInterval)
	if interval == 0 {
		interval = 5 * time.Second
	}

	// Simulate some CPU/Mem usage for demo
	var cpuCounter float64 = 20.0
	var memCounter float64 = 35.0

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		activeConns, rxPerSec, txPerSec := stats.Snapshot()

		// Simulate slight variations in CPU/memory for demo
		cpuCounter += (float64(activeConns) * 0.5 - cpuCounter) * 0.1
		memCounter += (35.0 + float64(activeConns)*0.1 - memCounter) * 0.05

		req := model.NodeHeartbeatRequest{
			NodeID:            cfg.Node.NodeID,
			CPUUsage:          cpuCounter,
			MemUsage:          memCounter,
			ActiveConnections: int(activeConns),
			RxBytesPerSec:     rxPerSec,
			TxBytesPerSec:     txPerSec,
		}

		body, _ := json.Marshal(req)
		url := fmt.Sprintf("%s/api/v1/nodes/heartbeat", cfg.Control.Endpoint)

		resp, err := http.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			logger.Warn("Heartbeat failed", "error", err)
			continue
		}
		resp.Body.Close()

		if activeConns > 0 || rxPerSec > 0 || txPerSec > 0 {
			logger.Debug("Heartbeat sent",
				"connections", activeConns,
				"rx_per_sec", rxPerSec,
				"tx_per_sec", txPerSec,
			)
		}
	}
}

func startMetricsServer(cfg *config.TrafficNodeConfig, stats *metrics.ConnectionStats) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		activeConns, totalConns, rxBytes, txBytes, authFailed, errors := stats.GetAll()

		fmt.Fprintf(w, "# HELP node_active_connections Current active connections\n")
		fmt.Fprintf(w, "# TYPE node_active_connections gauge\n")
		fmt.Fprintf(w, "node_active_connections{node_id=\"%s\"} %d\n", cfg.Node.NodeID, activeConns)

		fmt.Fprintf(w, "# HELP node_total_connections Total connections handled\n")
		fmt.Fprintf(w, "# TYPE node_total_connections counter\n")
		fmt.Fprintf(w, "node_total_connections{node_id=\"%s\"} %d\n", cfg.Node.NodeID, totalConns)

		fmt.Fprintf(w, "# HELP node_rx_bytes_total Total bytes received\n")
		fmt.Fprintf(w, "# TYPE node_rx_bytes_total counter\n")
		fmt.Fprintf(w, "node_rx_bytes_total{node_id=\"%s\"} %d\n", cfg.Node.NodeID, rxBytes)

		fmt.Fprintf(w, "# HELP node_tx_bytes_total Total bytes transmitted\n")
		fmt.Fprintf(w, "# TYPE node_tx_bytes_total counter\n")
		fmt.Fprintf(w, "node_tx_bytes_total{node_id=\"%s\"} %d\n", cfg.Node.NodeID, txBytes)

		fmt.Fprintf(w, "# HELP node_auth_failed_total Authentication failures\n")
		fmt.Fprintf(w, "# TYPE node_auth_failed_total counter\n")
		fmt.Fprintf(w, "node_auth_failed_total{node_id=\"%s\"} %d\n", cfg.Node.NodeID, authFailed)

		fmt.Fprintf(w, "# HELP node_connection_errors_total Connection errors\n")
		fmt.Fprintf(w, "# TYPE node_connection_errors_total counter\n")
		fmt.Fprintf(w, "node_connection_errors_total{node_id=\"%s\"} %d\n", cfg.Node.NodeID, errors)
	})

	addr := ":9100"
	logger.Info("Starting metrics server", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("Metrics server error", "error", err)
	}
}