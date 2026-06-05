package models

import "time"

type Device struct {
	ID                    uint       `json:"id"`
	IP                    string     `json:"ip"`
	MAC                   string     `json:"mac"`
	Hostname              string     `json:"hostname"`
	Vendor                string     `json:"vendor"`
	DeviceType            string     `json:"device_type"`
	Label                 string     `json:"label"`
	Owner                 string     `json:"owner"`
	Department            string     `json:"department"`
	Status                string     `json:"status"`
	FirstSeen             time.Time  `json:"first_seen"`
	LastSeen              time.Time  `json:"last_seen"`
	LastOnlineAt          *time.Time `json:"last_online_at"`
	LastOfflineAt         *time.Time `json:"last_offline_at"`
	OnlineDurationSeconds int64      `json:"online_duration_seconds"`
	OfflineCount          int        `json:"offline_count"`
	FailCount             int        `json:"fail_count"`
	ConsecutiveFailures   int        `json:"consecutive_failures"`
	RiskLevel             string     `json:"risk_level"`
	Note                  string     `json:"note"`
	OpenPorts             string     `json:"open_ports"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type DeviceEvent struct {
	ID          uint      `json:"id"`
	DeviceID    uint      `json:"device_id"`
	DeviceMAC   string    `json:"device_mac"`
	EventType   string    `json:"event_type"`
	OldStatus   string    `json:"old_status"`
	NewStatus   string    `json:"new_status"`
	EventTime   time.Time `json:"event_time"`
	Description string    `json:"description"`
}

type OnlineSession struct {
	ID          uint       `json:"id"`
	DeviceID    uint       `json:"device_id"`
	OnlineAt    time.Time  `json:"online_at"`
	OfflineAt   *time.Time `json:"offline_at"`
	DurationSec int64      `json:"duration_sec"`
}
