package reporter

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
	"github.com/aspira2026/traffic_lab_proxy/internal/proxynode/counter"
	"github.com/aspira2026/traffic_lab_proxy/internal/proxynode/proxy"
)

// HeartbeatReporter periodically sends node health status to the control center.
type HeartbeatReporter struct {
	client    *http.Client
	baseURL   string
	nodeID    string
	secret    string
	connMgr   *proxy.ConnectionManager
	counter   *counter.TrafficCounter
	interval  time.Duration
}

// NewHeartbeatReporter creates a heartbeat reporter.
func NewHeartbeatReporter(baseURL, nodeID, secret string,
	connMgr *proxy.ConnectionManager, counter *counter.TrafficCounter,
	interval time.Duration) *HeartbeatReporter {

	return &HeartbeatReporter{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL:  baseURL,
		nodeID:   nodeID,
		secret:   secret,
		connMgr:  connMgr,
		counter:  counter,
		interval: interval,
	}
}

// Start begins periodic heartbeat reporting.
func (hr *HeartbeatReporter) Start(ctx context.Context) {
	// Send first heartbeat immediately
	hr.sendHeartbeat(ctx)

	ticker := time.NewTicker(hr.interval)
	defer ticker.Stop()

	backoff := hr.interval
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := hr.sendHeartbeat(ctx); err != nil {
				slog.Error("heartbeat failed", "error", err)
				// Exponential backoff on failure
				backoff *= 2
				if backoff > 60*time.Second {
					backoff = 60 * time.Second
				}
				ticker.Reset(backoff)
			} else {
				backoff = hr.interval
				ticker.Reset(backoff)
			}
		}
	}
}

func (hr *HeartbeatReporter) sendHeartbeat(ctx context.Context) error {
	cpu := getCPUUsage()
	mem := getMemoryUsage()
	conns := hr.connMgr.CurrentConnections()
	rxMbps := hr.counter.NodeRxMbps() / 2 // Rough split between rx/tx
	txMbps := hr.counter.NodeRxMbps() / 2

	status := common.NodeStatusOnline
	bandwidthRatio := hr.connMgr.Utilization()
	if cpu > 85 || mem > 90 || bandwidthRatio > 0.9 {
		status = common.NodeStatusDegraded
	}

	req := common.HeartbeatRequest{
		NodeID:             hr.nodeID,
		CPUUsage:           cpu,
		MemoryUsage:        mem,
		CurrentConnections: conns,
		RxMbps:             rxMbps,
		TxMbps:             txMbps,
		Status:             status,
		Timestamp:          time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal heartbeat: %w", err)
	}

	sig := computeHMACReport(body, hr.secret)
	url := fmt.Sprintf("%s/api/v1/nodes/heartbeat", hr.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Node-Id", hr.nodeID)
	httpReq.Header.Set("X-Node-Signature", sig)

	resp, err := hr.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat returned status %d", resp.StatusCode)
	}

	// Update metrics
	proxy.CPUUsage.Set(cpu)
	proxy.MemoryUsage.Set(mem)
	proxy.RxMbps.Set(rxMbps)
	proxy.TxMbps.Set(txMbps)
	proxy.CurrentConnections.Set(float64(conns))

	return nil
}

// getCPUUsage reads CPU usage from /proc/stat (Linux-specific).
func getCPUUsage() float64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return 0
			}

			var total, idle float64
			for i, f := range fields[1:] {
				v, _ := strconv.ParseFloat(f, 64)
				total += v
				if i == 3 { // idle is the 4th field
					idle = v
				}
			}

			if total > 0 {
				return (1 - idle/total) * 100
			}
		}
	}
	return 0
}

// getMemoryUsage reads memory usage from /proc/meminfo (Linux-specific).
func getMemoryUsage() float64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}

	var total, available float64
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				total, _ = strconv.ParseFloat(fields[1], 64)
			}
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				available, _ = strconv.ParseFloat(fields[1], 64)
			}
		}
	}

	if total > 0 {
		return ((total - available) / total) * 100
	}
	return 0
}

// getGoMemoryUsage returns the Go runtime memory usage as percentage of system memory.
// This is a fallback when /proc/meminfo is unavailable.
func getGoMemoryUsage() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) / float64(m.Sys) * 100
}

func computeHMACReport(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
