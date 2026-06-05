package models

import "time"

type ScanTask struct {
	ID          uint       `json:"id"`
	Subnet      string     `json:"subnet"`
	ScanType    string     `json:"scan_type"`
	Status      string     `json:"status"`
	StartedAt   *time.Time `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at"`
	TotalIPs    int        `json:"total_ips"`
	OnlineCount int        `json:"online_count"`
	ErrorMsg    string     `json:"error_message"`
}

type ScanConfig struct {
	ID               uint   `json:"id"`
	Subnets          string `json:"subnets"`
	Methods          string `json:"methods"`
	IntervalSeconds  int    `json:"interval_seconds"`
	WorkerCount      int    `json:"worker_count"`
	TimeoutMs        int    `json:"timeout_ms"`
	OfflineThreshold int    `json:"offline_threshold"`
	TCPPorts         string `json:"tcp_ports"`
	Enabled          bool   `json:"enabled"`
}
