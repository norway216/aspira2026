package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Node metrics
	NodesOnline = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "control_nodes_online_total",
		Help: "Number of online proxy nodes",
	})
	NodesOffline = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "control_nodes_offline_total",
		Help: "Number of offline proxy nodes",
	})
	NodesDegraded = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "control_nodes_degraded_total",
		Help: "Number of degraded proxy nodes",
	})

	// API metrics
	APIRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "control_api_requests_total",
		Help: "Total number of API requests",
	}, []string{"method", "path", "status"})
	APIErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "control_api_errors_total",
		Help: "Total number of API errors",
	}, []string{"method", "path"})

	// Traffic metrics
	UserTrafficBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "control_user_traffic_used_bytes",
		Help: "Total traffic used per user in bytes",
	}, []string{"user_id"})
)

// MetricsMiddleware records Prometheus metrics for each HTTP request.
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(ww, r)

		statusLabel := http.StatusText(ww.statusCode)
		if statusLabel == "" {
			statusLabel = "unknown"
		}

		APIRequestsTotal.WithLabelValues(r.Method, r.URL.Path, statusLabel).Inc()
		if ww.statusCode >= 400 {
			APIErrorsTotal.WithLabelValues(r.Method, r.URL.Path).Inc()
		}
	})
}
