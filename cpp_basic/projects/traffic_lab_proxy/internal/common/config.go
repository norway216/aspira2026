package common

import (
	"os"
	"strconv"
	"time"
)

// ────────────────────────────────────────────────────────────
// Control Center Configuration
// ────────────────────────────────────────────────────────────

type ControlCenterConfig struct {
	ListenAddr        string
	DatabaseDriver    string
	DatabaseDSN       string
	AdminUser         string
	AdminPass         string
	SessionSecret     string
	SchedulerStrategy string
	NodeStaleTimeout  time.Duration
	LogLevel          string
}

func DefaultControlCenterConfig() *ControlCenterConfig {
	return &ControlCenterConfig{
		ListenAddr:        ":" + strconv.Itoa(DefaultControlCenterPort),
		DatabaseDriver:    DefaultDatabaseDriver,
		DatabaseDSN:       DefaultDatabaseDSN,
		AdminUser:         DefaultAdminUsername,
		AdminPass:         DefaultAdminPassword,
		SessionSecret:     DefaultSessionSecret,
		SchedulerStrategy: DefaultSchedulerStrategy,
		NodeStaleTimeout:  DefaultNodeStaleTimeout * time.Second,
		LogLevel:          "info",
	}
}

func LoadControlCenterConfig() *ControlCenterConfig {
	cfg := DefaultControlCenterConfig()

	if v := os.Getenv("LISTEN_ADDR"); v != "" {
		cfg.ListenAddr = v
	}
	if v := os.Getenv("DATABASE_DRIVER"); v != "" {
		cfg.DatabaseDriver = v
	}
	if v := os.Getenv("DATABASE_DSN"); v != "" {
		cfg.DatabaseDSN = v
	}
	if v := os.Getenv("ADMIN_USER"); v != "" {
		cfg.AdminUser = v
	}
	if v := os.Getenv("ADMIN_PASS"); v != "" {
		cfg.AdminPass = v
	}
	if v := os.Getenv("SESSION_SECRET"); v != "" {
		cfg.SessionSecret = v
	}
	if v := os.Getenv("SCHEDULER_STRATEGY"); v != "" {
		cfg.SchedulerStrategy = v
	}
	if v := os.Getenv("NODE_STALE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.NodeStaleTimeout = d
		}
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	return cfg
}

// ────────────────────────────────────────────────────────────
// Proxy Node Configuration
// ────────────────────────────────────────────────────────────

type ProxyNodeConfig struct {
	NodeID                string
	ListenAddr            string
	ControlCenterURL      string
	NodeSecret            string
	MetricsListenAddr     string
	HeartbeatInterval     time.Duration
	TrafficReportInterval time.Duration
	PolicyFetchInterval   time.Duration
	MaxConnections        int
	MaxBandwidthMbps      int
	TargetConnectTimeout  time.Duration
	ProxyBufferSize       int
}

func DefaultProxyNodeConfig() *ProxyNodeConfig {
	return &ProxyNodeConfig{
		NodeID:                "node-01",
		ListenAddr:            ":" + strconv.Itoa(DefaultProxyNodePort),
		ControlCenterURL:      "http://localhost:" + strconv.Itoa(DefaultControlCenterPort),
		NodeSecret:            "node-secret-change-me",
		MetricsListenAddr:     ":" + strconv.Itoa(DefaultMetricsPort),
		HeartbeatInterval:     DefaultHeartbeatInterval * time.Second,
		TrafficReportInterval: DefaultTrafficReportInterval * time.Second,
		PolicyFetchInterval:   DefaultPolicyFetchInterval * time.Second,
		MaxConnections:        DefaultMaxConnections,
		MaxBandwidthMbps:      DefaultMaxBandwidthMbps,
		TargetConnectTimeout:  DefaultTargetConnectTimeout * time.Second,
		ProxyBufferSize:       DefaultProxyBufferSize,
	}
}

func LoadProxyNodeConfig() *ProxyNodeConfig {
	cfg := DefaultProxyNodeConfig()

	if v := os.Getenv("NODE_ID"); v != "" {
		cfg.NodeID = v
	}
	if v := os.Getenv("LISTEN_ADDR"); v != "" {
		cfg.ListenAddr = v
	}
	if v := os.Getenv("CONTROL_CENTER_URL"); v != "" {
		cfg.ControlCenterURL = v
	}
	if v := os.Getenv("NODE_SECRET"); v != "" {
		cfg.NodeSecret = v
	}
	if v := os.Getenv("METRICS_LISTEN_ADDR"); v != "" {
		cfg.MetricsListenAddr = v
	}
	if v := os.Getenv("HEARTBEAT_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.HeartbeatInterval = d
		}
	}
	if v := os.Getenv("TRAFFIC_REPORT_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.TrafficReportInterval = d
		}
	}
	if v := os.Getenv("POLICY_FETCH_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.PolicyFetchInterval = d
		}
	}
	if v := os.Getenv("MAX_CONNECTIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxConnections = n
		}
	}
	if v := os.Getenv("MAX_BANDWIDTH_MBPS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxBandwidthMbps = n
		}
	}
	if v := os.Getenv("TARGET_CONNECT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.TargetConnectTimeout = d
		}
	}
	if v := os.Getenv("PROXY_BUFFER_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.ProxyBufferSize = n
		}
	}
	return cfg
}
