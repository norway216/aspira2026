package common

import "time"

// ────────────────────────────────────────────────────────────
// Database Models
// ────────────────────────────────────────────────────────────

// User represents a proxy user account.
type User struct {
	ID             int64      `json:"id"`
	Username       string     `json:"username"`
	PasswordHash   string     `json:"-"`
	Token          string     `json:"token,omitempty"`
	Status         string     `json:"status"`
	TrafficTotal   int64      `json:"traffic_total"`
	TrafficUsed    int64      `json:"traffic_used"`
	MaxRateMbps    int        `json:"max_rate_mbps"`
	MaxConnections int        `json:"max_connections"`
	ExpiredAt      *time.Time `json:"expired_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Node represents a registered proxy node.
type Node struct {
	ID               int64      `json:"id"`
	NodeID           string     `json:"node_id"`
	Name             string     `json:"name"`
	IP               string     `json:"ip"`
	ProxyPort        int        `json:"proxy_port"`
	MaxBandwidthMbps int        `json:"max_bandwidth_mbps"`
	MaxConnections   int        `json:"max_connections"`
	NodeSecret       string     `json:"-"`
	Weight           int        `json:"weight"`
	Status           string     `json:"status"`
	LastHeartbeat    *time.Time `json:"last_heartbeat,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// NodeStatus represents a snapshot of node health reported via heartbeat.
type NodeStatus struct {
	ID                 int64     `json:"id"`
	NodeID             string    `json:"node_id"`
	CPUUsage           float64   `json:"cpu_usage"`
	MemoryUsage        float64   `json:"memory_usage"`
	CurrentConnections int       `json:"current_connections"`
	RxMbps             float64   `json:"rx_mbps"`
	TxMbps             float64   `json:"tx_mbps"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
}

// TrafficRecord represents a traffic accounting record for a user on a node.
type TrafficRecord struct {
	ID              int64     `json:"id"`
	NodeID          string    `json:"node_id"`
	UserID          string    `json:"user_id"`
	UploadBytes     int64     `json:"upload_bytes"`
	DownloadBytes   int64     `json:"download_bytes"`
	ConnectionCount int       `json:"connection_count"`
	CreatedAt       time.Time `json:"created_at"`
}

// Policy defines rate limiting and connection limits for a specific user.
type Policy struct {
	ID             int64     `json:"id"`
	UserID         string    `json:"user_id"`
	MaxRateMbps    int       `json:"max_rate_mbps"`
	BurstMbps      int       `json:"burst_mbps"`
	MaxConnections int       `json:"max_connections"`
	Priority       int       `json:"priority"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SchedulerEvent records each scheduling decision for audit and debugging.
type SchedulerEvent struct {
	ID             int64     `json:"id"`
	UserID         string    `json:"user_id,omitempty"`
	SelectedNodeID string    `json:"selected_node_id,omitempty"`
	Strategy       string    `json:"strategy"`
	Reason         string    `json:"reason,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// ────────────────────────────────────────────────────────────
// API Request DTOs
// ────────────────────────────────────────────────────────────

// RegisterRequest is sent by a proxy node to register itself.
type RegisterRequest struct {
	NodeID           string `json:"node_id"`
	Name             string `json:"name"`
	IP               string `json:"ip"`
	ProxyPort        int    `json:"proxy_port"`
	MaxBandwidthMbps int    `json:"max_bandwidth_mbps"`
	MaxConnections   int    `json:"max_connections"`
	NodeSecret       string `json:"node_secret"`
}

// HeartbeatRequest is sent by a proxy node periodically.
type HeartbeatRequest struct {
	NodeID             string  `json:"node_id"`
	CPUUsage           float64 `json:"cpu_usage"`
	MemoryUsage        float64 `json:"memory_usage"`
	CurrentConnections int     `json:"current_connections"`
	RxMbps             float64 `json:"rx_mbps"`
	TxMbps             float64 `json:"tx_mbps"`
	Status             string  `json:"status"`
	Timestamp          string  `json:"timestamp"`
}

// TrafficReportRequest is sent by a proxy node to report user traffic.
type TrafficReportRequest struct {
	NodeID    string          `json:"node_id"`
	Records   []TrafficRecordDTO `json:"records"`
	Timestamp string          `json:"timestamp"`
}

// TrafficRecordDTO is a traffic record within a report request.
type TrafficRecordDTO struct {
	UserID          string `json:"user_id"`
	UploadBytes     int64  `json:"upload_bytes"`
	DownloadBytes   int64  `json:"download_bytes"`
	ConnectionCount int    `json:"connection_count"`
}

// ────────────────────────────────────────────────────────────
// API Response DTOs
// ────────────────────────────────────────────────────────────

// APIResponse is the standard JSON response wrapper.
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PolicyResponse is returned when a node fetches its policies.
type PolicyResponse struct {
	NodeID       string              `json:"node_id"`
	GlobalPolicy GlobalPolicy        `json:"global_policy"`
	UserPolicies []UserPolicySummary `json:"user_policies"`
}

// GlobalPolicy contains node-level policy limits.
type GlobalPolicy struct {
	MaxNodeBandwidthMbps int `json:"max_node_bandwidth_mbps"`
	MaxNodeConnections   int `json:"max_node_connections"`
}

// UserPolicySummary is a per-user policy sent to proxy nodes.
type UserPolicySummary struct {
	UserID         string `json:"user_id"`
	Token          string `json:"token"`
	MaxRateMbps    int    `json:"max_rate_mbps"`
	BurstMbps      int    `json:"burst_mbps"`
	MaxConnections int    `json:"max_connections"`
	Status         string `json:"status"`
}

// CreateUserRequest is sent to create a new user.
type CreateUserRequest struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	MaxRateMbps    int    `json:"max_rate_mbps"`
	MaxConnections int    `json:"max_connections"`
	TrafficTotal   int64  `json:"traffic_total"`
}

// UpdateUserRequest is sent to update an existing user.
type UpdateUserRequest struct {
	Status         *string `json:"status,omitempty"`
	MaxRateMbps    *int    `json:"max_rate_mbps,omitempty"`
	MaxConnections *int    `json:"max_connections,omitempty"`
	TrafficTotal   *int64  `json:"traffic_total,omitempty"`
	Password       *string `json:"password,omitempty"`
}

// LoginRequest is the admin login payload.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ────────────────────────────────────────────────────────────
// Aggregate / Summary types
// ────────────────────────────────────────────────────────────

// TrafficSummary aggregates traffic data for a user or node.
type TrafficSummary struct {
	UserID          string `json:"user_id,omitempty"`
	NodeID          string `json:"node_id,omitempty"`
	TotalUpload     int64  `json:"total_upload"`
	TotalDownload   int64  `json:"total_download"`
	TotalBytes      int64  `json:"total_bytes"`
	ConnectionCount int    `json:"connection_count"`
}

// DashboardStats holds key metrics for the admin dashboard overview.
type DashboardStats struct {
	OnlineNodes     int   `json:"online_nodes"`
	OfflineNodes    int   `json:"offline_nodes"`
	DegradedNodes   int   `json:"degraded_nodes"`
	TotalNodes      int   `json:"total_nodes"`
	ActiveUsers     int   `json:"active_users"`
	TotalUsers      int   `json:"total_users"`
	TotalUpload     int64 `json:"total_upload"`
	TotalDownload   int64 `json:"total_download"`
	ActiveConnections int `json:"active_connections"`
}
