package models

import "time"

type AuditLog struct {
	ID         int64     `json:"id" db:"id"`
	UserID     string    `json:"user_id,omitempty" db:"user_id"`
	Action     string    `json:"action" db:"action"`
	Resource   string    `json:"resource" db:"resource"`
	ResourceID string    `json:"resource_id" db:"resource_id"`
	IPAddr     string    `json:"ip_addr" db:"ip_addr"`
	UserAgent  string    `json:"user_agent" db:"user_agent"`
	Detail     string    `json:"detail" db:"detail"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type AuditLogListResponse struct {
	Logs     []AuditLog `json:"logs"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}
