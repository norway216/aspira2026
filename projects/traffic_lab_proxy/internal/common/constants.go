package common

// Node status constants
const (
	NodeStatusOnline   = "online"
	NodeStatusOffline  = "offline"
	NodeStatusDegraded = "degraded"
	NodeStatusDisabled = "disabled"
)

// User status constants
const (
	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"
	UserStatusExpired  = "expired"
)

// Policy status constants
const (
	PolicyStatusActive   = "active"
	PolicyStatusInactive = "inactive"
)

// Scheduler strategy names
const (
	SchedulerRoundRobin     = "round_robin"
	SchedulerLeastConn      = "least_connections"
	SchedulerLeastBw        = "least_bandwidth"
	SchedulerWeightedRR     = "weighted_round_robin"
	SchedulerComposite      = "composite"
)

// Default configuration values
const (
	DefaultControlCenterPort    = 8080
	DefaultProxyNodePort        = 10801
	DefaultMetricsPort          = 9200
	DefaultHeartbeatInterval    = 10  // seconds
	DefaultTrafficReportInterval = 15  // seconds
	DefaultPolicyFetchInterval  = 30  // seconds
	DefaultNodeStaleTimeout     = 30  // seconds
	DefaultMaxConnections       = 1000
	DefaultMaxBandwidthMbps     = 100
	DefaultProxyBufferSize      = 32768 // 32KB
	DefaultTargetConnectTimeout = 10    // seconds
	DefaultDatabaseDriver       = "sqlite"
	DefaultDatabaseDSN          = "traffic_lab.db"
	DefaultSchedulerStrategy    = SchedulerComposite
	DefaultSessionSecret        = "change-me-in-production-use-a-random-secret"
	DefaultAdminUsername        = "admin"
	DefaultAdminPassword        = "admin123"
)
