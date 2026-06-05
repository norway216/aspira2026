package models

import "time"

// TrafficRecord stores per-device traffic data points
type TrafficRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	DeviceID  uint      `gorm:"index;not null" json:"device_id"`
	DeviceMAC string    `gorm:"size:64;index" json:"device_mac"`
	RxBytes   int64     `json:"rx_bytes"`
	TxBytes   int64     `json:"tx_bytes"`
	RxRate    float64   `json:"rx_rate"`
	TxRate    float64   `json:"tx_rate"`
	TotalRate float64   `json:"total_rate"`
	Timestamp time.Time `gorm:"index" json:"timestamp"`
}

// SystemTraffic stores overall system network traffic
type SystemTraffic struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RxBytes   int64     `json:"rx_bytes"`
	TxBytes   int64     `json:"tx_bytes"`
	RxRate    float64   `json:"rx_rate"`
	TxRate    float64   `json:"tx_rate"`
	Timestamp time.Time `gorm:"index" json:"timestamp"`
}

// AgentReport represents data reported by a device agent
type AgentReport struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	DeviceID    string     `gorm:"size:128;index" json:"device_id"`
	IP          string    `gorm:"size:64" json:"ip"`
	MAC         string    `gorm:"size:64" json:"mac"`
	RxBytes     int64     `json:"rx_bytes"`
	TxBytes     int64     `json:"tx_bytes"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	DiskUsage   float64   `json:"disk_usage"`
	Temperature float64   `json:"temperature"`
	ReportTime  time.Time `gorm:"index" json:"report_time"`
}
