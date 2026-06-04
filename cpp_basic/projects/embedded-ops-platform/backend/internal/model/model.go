package model

import "time"

// User represents a platform user.
type User struct {
	ID        int64     `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Password  string    `json:"-" db:"password_hash"`
	Email     string    `json:"email" db:"email"`
	Role      string    `json:"role" db:"role"` // admin, operator, viewer, auditor
	Enabled   bool      `json:"enabled" db:"enabled"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Device represents a managed embedded device.
type Device struct {
	ID            int64     `json:"id" db:"id"`
	DeviceID      string    `json:"device_id" db:"device_id"`
	DeviceName    string    `json:"device_name" db:"device_name"`
	Hostname      string    `json:"hostname" db:"hostname"`
	IPAddress     string    `json:"ip_address" db:"ip_address"`
	MACAddress    string    `json:"mac_address" db:"mac_address"`
	BoardType     string    `json:"board_type" db:"board_type"`
	Arch          string    `json:"arch" db:"arch"`
	OSName        string    `json:"os_name" db:"os_name"`
	OSVersion     string    `json:"os_version" db:"os_version"`
	KernelVersion string    `json:"kernel_version" db:"kernel_version"`
	BSPVersion    string    `json:"bsp_version" db:"bsp_version"`
	AgentVersion  string    `json:"agent_version" db:"agent_version"`
	Status        string    `json:"status" db:"status"` // online, offline
	LastSeen      time.Time `json:"last_seen" db:"last_seen"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// AgentCredential stores agent authentication tokens.
type AgentCredential struct {
	ID             int64     `json:"id" db:"id"`
	DeviceID       string    `json:"device_id" db:"device_id"`
	TokenHash      string    `json:"-" db:"token_hash"`
	CertFingerprint string   `json:"cert_fingerprint,omitempty" db:"cert_fingerprint"`
	Enabled        bool      `json:"enabled" db:"enabled"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// DeviceMetric represents a device metric data point.
type DeviceMetric struct {
	ID           int64     `json:"id" db:"id"`
	DeviceID     string    `json:"device_id" db:"device_id"`
	CPUUsage     float64   `json:"cpu_usage" db:"cpu_usage"`
	MemoryUsage  float64   `json:"memory_usage" db:"memory_usage"`
	DiskUsage    float64   `json:"disk_usage" db:"disk_usage"`
	LoadAvg1m    float64   `json:"load_avg_1m" db:"load_avg_1m"`
	Temperature  float64   `json:"temperature" db:"temperature"`
	GPULoad      float64   `json:"gpu_load" db:"gpu_load"`
	NetworkRX    int64     `json:"network_rx_bytes" db:"network_rx_bytes"`
	NetworkTX    int64     `json:"network_tx_bytes" db:"network_tx_bytes"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// CommandTask represents a remote command to be executed on a device.
type CommandTask struct {
	ID         int64     `json:"id" db:"id"`
	TaskID     string    `json:"task_id" db:"task_id"`
	DeviceID   string    `json:"device_id" db:"device_id"`
	Action     string    `json:"action" db:"action"`
	Params     string    `json:"params" db:"params"` // JSON string
	Status     string    `json:"status" db:"status"` // pending, running, completed, failed
	Stdout     string    `json:"stdout,omitempty" db:"stdout"`
	Stderr     string    `json:"stderr,omitempty" db:"stderr"`
	ExitCode   int       `json:"exit_code,omitempty" db:"exit_code"`
	CreatedBy  string    `json:"created_by" db:"created_by"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty" db:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty" db:"finished_at"`
}

// DeviceLog represents a log entry from a device.
type DeviceLog struct {
	ID        int64     `json:"id" db:"id"`
	DeviceID  string    `json:"device_id" db:"device_id"`
	LogType   string    `json:"log_type" db:"log_type"`
	Level     string    `json:"level" db:"level"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Alert represents an alert for a device.
type Alert struct {
	ID         int64      `json:"id" db:"id"`
	DeviceID   string     `json:"device_id" db:"device_id"`
	AlertType  string     `json:"alert_type" db:"alert_type"`
	Severity   string     `json:"severity" db:"severity"`
	Message    string     `json:"message" db:"message"`
	Status     string     `json:"status" db:"status"` // open, acknowledged, resolved
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
}

// --- API Request/Response types ---

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type AgentRegisterRequest struct {
	RegisterCode string   `json:"register_code" binding:"required"`
	Hostname     string   `json:"hostname"`
	MachineID    string   `json:"machine_id"`
	MACAddresses []string `json:"mac_addresses"`
	Arch         string   `json:"arch"`
	OSName       string   `json:"os_name"`
	KernelVersion string  `json:"kernel_version"`
	BoardType    string   `json:"board_type"`
	AgentVersion string   `json:"agent_version"`
}

type AgentRegisterResponse struct {
	DeviceID    string `json:"device_id"`
	AgentToken  string `json:"agent_token"`
	ServerTime  int64  `json:"server_time"`
	ConfigVersion int  `json:"config_version"`
}

type HeartbeatRequest struct {
	DeviceID    string `json:"device_id" binding:"required"`
	Timestamp   int64  `json:"timestamp"`
	UptimeSec   int64  `json:"uptime_sec"`
	AgentVersion string `json:"agent_version"`
	AppStatus   string `json:"app_status"`
}

type MetricsRequest struct {
	DeviceID   string  `json:"device_id" binding:"required"`
	Timestamp  int64   `json:"timestamp"`
	CPUUsage   float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage  float64 `json:"disk_usage"`
	LoadAvg1m  float64 `json:"load_avg_1m"`
	Temperature float64 `json:"temperature"`
	GPULoad    float64 `json:"gpu_load"`
	NetworkRX  int64   `json:"network_rx_bytes"`
	NetworkTX  int64   `json:"network_tx_bytes"`
}

type CommandRequest struct {
	Action string                 `json:"action" binding:"required"`
	Params map[string]interface{} `json:"params"`
}

type CommandResultRequest struct {
	TaskID   string `json:"task_id" binding:"required"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Status   string `json:"status"` // completed, failed
}

// RegisterCode represents a one-time registration code.
type RegisterCode struct {
	Code      string    `json:"code"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}