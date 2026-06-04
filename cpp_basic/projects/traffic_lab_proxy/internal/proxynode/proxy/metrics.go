package proxy

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CurrentConnections tracks the number of active proxy connections.
	CurrentConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "proxy_node_connections_current",
		Help: "Current number of active proxy connections",
	})

	// RxBytesTotal counts total bytes received (upload from client to proxy).
	RxBytesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "proxy_node_rx_bytes_total",
		Help: "Total bytes received from clients (upload)",
	})

	// TxBytesTotal counts total bytes transmitted (download from proxy to client).
	TxBytesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "proxy_node_tx_bytes_total",
		Help: "Total bytes transmitted to clients (download)",
	})

	// RxMbps tracks the current receive rate in Mbps.
	RxMbps = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "proxy_node_rx_mbps",
		Help: "Current receive bandwidth in Mbps",
	})

	// TxMbps tracks the current transmit rate in Mbps.
	TxMbps = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "proxy_node_tx_mbps",
		Help: "Current transmit bandwidth in Mbps",
	})

	// CPUUsage reports the proxy node CPU usage.
	CPUUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "proxy_node_cpu_usage_percent",
		Help: "Proxy node CPU usage percentage",
	})

	// MemoryUsage reports the proxy node memory usage.
	MemoryUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "proxy_node_memory_usage_percent",
		Help: "Proxy node memory usage percentage",
	})

	// UserRxBytes tracks per-user received bytes.
	UserRxBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "proxy_user_rx_bytes_total",
		Help: "Total bytes received per user",
	}, []string{"user_id"})

	// UserTxBytes tracks per-user transmitted bytes.
	UserTxBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "proxy_user_tx_bytes_total",
		Help: "Total bytes transmitted per user",
	}, []string{"user_id"})

	// RateLimitedTotal counts rate limiting events.
	RateLimitedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "proxy_user_rate_limited_total",
		Help: "Total rate limiting events by user",
	}, []string{"user_id"})

	// ConnectionErrors counts connection-level errors.
	ConnectionErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "proxy_connection_errors_total",
		Help: "Total connection errors",
	})
)
