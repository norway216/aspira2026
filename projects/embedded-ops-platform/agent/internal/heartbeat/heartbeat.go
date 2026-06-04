package heartbeat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Sender handles heartbeat and metric sending to the server.
type Sender struct {
	serverURL string
	deviceID  string
	token     string
	client    *http.Client
	startTime time.Time
}

// NewSender creates a new heartbeat sender.
func NewSender(serverURL, deviceID, token string) *Sender {
	return &Sender{
		serverURL: serverURL,
		deviceID:  deviceID,
		token:     token,
		client:    &http.Client{Timeout: 10 * time.Second},
		startTime: time.Now(),
	}
}

// SendHeartbeat sends a heartbeat to the server.
func (s *Sender) SendHeartbeat() error {
	payload := map[string]interface{}{
		"device_id":     s.deviceID,
		"timestamp":     time.Now().Unix(),
		"uptime_sec":    int64(time.Since(s.startTime).Seconds()),
		"agent_version": "1.0.0",
		"app_status":    "running",
	}

	return s.post("/api/v1/agents/heartbeat", payload)
}

// SendMetrics sends metrics to the server.
func (s *Sender) SendMetrics(cpu, mem, disk, load, temp float64, rx, tx int64) error {
	payload := map[string]interface{}{
		"device_id":       s.deviceID,
		"timestamp":       time.Now().Unix(),
		"cpu_usage":       cpu,
		"memory_usage":    mem,
		"disk_usage":      disk,
		"load_avg_1m":     load,
		"temperature":     temp,
		"network_rx_bytes": rx,
		"network_tx_bytes": tx,
	}

	return s.post("/api/v1/agents/metrics", payload)
}

// PullTasks fetches pending command tasks from the server.
func (s *Sender) PullTasks() ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/agents/tasks/pull", s.serverURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Device-ID", s.deviceID)
	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Tasks []map[string]interface{} `json:"tasks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode tasks response: %w", err)
	}

	return result.Tasks, nil
}

// SubmitTaskResult sends the result of a completed command task.
func (s *Sender) SubmitTaskResult(taskID, status, stdout, stderr string, exitCode int) error {
	payload := map[string]interface{}{
		"task_id":   taskID,
		"status":    status,
		"stdout":    stdout,
		"stderr":    stderr,
		"exit_code": exitCode,
	}

	url := fmt.Sprintf("%s/api/v1/agents/tasks/%s/result", s.serverURL, taskID)
	return s.postWithURL(url, payload)
}

func (s *Sender) post(path string, payload interface{}) error {
	url := fmt.Sprintf("%s%s", s.serverURL, path)
	return s.postWithURL(url, payload)
}

func (s *Sender) postWithURL(url string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-ID", s.deviceID)
	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}