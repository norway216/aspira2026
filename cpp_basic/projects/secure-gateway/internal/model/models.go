package model

import "time"

// User represents a system user.
type User struct {
	ID               int64     `json:"id" db:"id"`
	Username         string    `json:"username" db:"username"`
	PasswordHash     string    `json:"-" db:"password_hash"`
	Role             string    `json:"role" db:"role"`
	Status           string    `json:"status" db:"status"`
	TrafficQuotaBytes int64    `json:"traffic_quota_bytes" db:"traffic_quota_bytes"`
	TrafficUsedBytes  int64    `json:"traffic_used_bytes" db:"traffic_used_bytes"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// Node represents a traffic forwarding node.
type Node struct {
	ID             int64     `json:"id" db:"id"`
	NodeID         string    `json:"node_id" db:"node_id"`
	Name           string    `json:"name" db:"name"`
	Region         string    `json:"region" db:"region"`
	PublicAddr     string    `json:"public_addr" db:"public_addr"`
	Status         string    `json:"status" db:"status"`
	Weight         int       `json:"weight" db:"weight"`
	MaxConnections int       `json:"max_connections" db:"max_connections"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// NodeMetrics represents real-time metrics for a node.
type NodeMetrics struct {
	ID                int64     `json:"id" db:"id"`
	NodeID            string    `json:"node_id" db:"node_id"`
	CPUUsage          float64   `json:"cpu_usage" db:"cpu_usage"`
	MemUsage          float64   `json:"mem_usage" db:"mem_usage"`
	ActiveConnections int       `json:"active_connections" db:"active_connections"`
	RxBytesPerSec     int64     `json:"rx_bytes_per_sec" db:"rx_bytes_per_sec"`
	TxBytesPerSec     int64     `json:"tx_bytes_per_sec" db:"tx_bytes_per_sec"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// TrafficRecord represents a traffic usage record for billing/quota.
type TrafficRecord struct {
	ID              int64     `json:"id" db:"id"`
	UserID          int64     `json:"user_id" db:"user_id"`
	NodeID          string    `json:"node_id" db:"node_id"`
	RxBytes         int64     `json:"rx_bytes" db:"rx_bytes"`
	TxBytes         int64     `json:"tx_bytes" db:"tx_bytes"`
	DurationSeconds int64     `json:"duration_seconds" db:"duration_seconds"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID        int64     `json:"id" db:"id"`
	UserID    *int64    `json:"user_id,omitempty" db:"user_id"`
	Action    string    `json:"action" db:"action"`
	Resource  string    `json:"resource,omitempty" db:"resource"`
	IPAddr    string    `json:"ip_addr,omitempty" db:"ip_addr"`
	UserAgent string    `json:"user_agent,omitempty" db:"user_agent"`
	Detail    string    `json:"detail,omitempty" db:"detail"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// --- API request/response types ---

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role"`
	Quota    int64  `json:"quota_bytes"`
}

type UpdateUserRequest struct {
	Password *string `json:"password,omitempty"`
	Role     *string `json:"role,omitempty"`
	Status   *string `json:"status,omitempty"`
	Quota    *int64  `json:"quota_bytes,omitempty"`
}

type NodeRegisterRequest struct {
	NodeID       string `json:"node_id" binding:"required"`
	Name         string `json:"name" binding:"required"`
	Region       string `json:"region" binding:"required"`
	PublicAddr   string `json:"public_addr" binding:"required"`
	Secret       string `json:"secret" binding:"required"`
	MaxConnections int  `json:"max_connections"`
}

type NodeHeartbeatRequest struct {
	NodeID            string  `json:"node_id" binding:"required"`
	CPUUsage          float64 `json:"cpu_usage"`
	MemUsage          float64 `json:"mem_usage"`
	ActiveConnections int     `json:"active_connections"`
	RxBytesPerSec     int64   `json:"rx_bytes_per_sec"`
	TxBytesPerSec     int64   `json:"tx_bytes_per_sec"`
}

type DashboardStats struct {
	OnlineNodes        int     `json:"online_nodes"`
	TotalConnections   int     `json:"total_connections"`
	TodayTrafficBytes  int64   `json:"today_traffic_bytes"`
	AlertCount         int     `json:"alert_count"`
	AvgCPUUsage        float64 `json:"avg_cpu_usage"`
	AvgMemUsage        float64 `json:"avg_mem_usage"`
}

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

const (
	RoleAdmin = "admin"
	RoleUser  = "user"

	StatusActive   = "active"
	StatusDisabled = "disabled"

	NodeStatusOnline  = "online"
	NodeStatusOffline = "offline"
	NodeStatusDraining = "draining"
)