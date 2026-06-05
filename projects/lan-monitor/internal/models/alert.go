package models

import "time"

type Alert struct {
	ID         uint       `json:"id"`
	DeviceID   *uint      `json:"device_id"`
	DeviceMAC  string     `json:"device_mac"`
	AlertType  string     `json:"alert_type"`
	Level      string     `json:"level"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
	ResolvedBy *uint      `json:"resolved_by"`
}

type AuditLog struct {
	ID           uint      `json:"id"`
	UserID       *uint     `json:"user_id"`
	Username     string    `json:"username"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	IP           string    `json:"ip"`
	UserAgent    string    `json:"user_agent"`
	Detail       string    `json:"detail"`
	CreatedAt    time.Time `json:"created_at"`
}
