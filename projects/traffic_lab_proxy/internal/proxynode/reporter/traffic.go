package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
	"github.com/aspira2026/traffic_lab_proxy/internal/proxynode/counter"
)

// TrafficReporter periodically sends traffic statistics to the control center.
type TrafficReporter struct {
	client   *http.Client
	baseURL  string
	nodeID   string
	secret   string
	counter  *counter.TrafficCounter
	interval time.Duration
}

// NewTrafficReporter creates a traffic reporter.
func NewTrafficReporter(baseURL, nodeID, secret string,
	c *counter.TrafficCounter, interval time.Duration) *TrafficReporter {

	return &TrafficReporter{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		baseURL:  baseURL,
		nodeID:   nodeID,
		secret:   secret,
		counter:  c,
		interval: interval,
	}
}

// Start begins periodic traffic reporting.
func (tr *TrafficReporter) Start(ctx context.Context) {
	ticker := time.NewTicker(tr.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Final report before shutdown
			tr.reportTraffic(context.Background())
			return
		case <-ticker.C:
			tr.reportTraffic(ctx)
		}
	}
}

func (tr *TrafficReporter) reportTraffic(ctx context.Context) {
	snapshots := tr.counter.SnapshotAndReset()
	if len(snapshots) == 0 {
		return
	}

	records := make([]common.TrafficRecordDTO, 0, len(snapshots))
	for _, s := range snapshots {
		records = append(records, common.TrafficRecordDTO{
			UserID:          s.UserID,
			UploadBytes:     s.UploadBytes,
			DownloadBytes:   s.DownloadBytes,
			ConnectionCount: int(s.ConnectionCount),
		})
	}

	req := common.TrafficReportRequest{
		NodeID:    tr.nodeID,
		Records:   records,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(req)
	if err != nil {
		slog.Error("traffic reporter: marshal error", "error", err)
		return
	}

	sig := computeHMACReport(body, tr.secret)
	url := fmt.Sprintf("%s/api/v1/traffic/report", tr.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		slog.Error("traffic reporter: create request", "error", err)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Node-Id", tr.nodeID)
	httpReq.Header.Set("X-Node-Signature", sig)

	resp, err := tr.client.Do(httpReq)
	if err != nil {
		slog.Error("traffic reporter: request failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("traffic reporter: non-200 response", "status", resp.StatusCode)
		return
	}

	slog.Debug("traffic reported",
		"records", len(records),
	)
}
