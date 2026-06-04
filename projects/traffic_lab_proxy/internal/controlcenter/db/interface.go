package db

import (
	"context"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
)

// DB is the database abstraction for the Control Center.
// All methods accept context.Context for cancellation and timeout support.
type DB interface {
	// ── Users ─────────────────────────────────────────────
	CreateUser(ctx context.Context, user *common.User) error
	GetUser(ctx context.Context, id int64) (*common.User, error)
	GetUserByToken(ctx context.Context, token string) (*common.User, error)
	GetUserByUsername(ctx context.Context, username string) (*common.User, error)
	ListUsers(ctx context.Context) ([]*common.User, error)
	UpdateUser(ctx context.Context, user *common.User) error
	DeleteUser(ctx context.Context, id int64) error
	AddUserTraffic(ctx context.Context, userID string, bytes int64) error

	// ── Nodes ─────────────────────────────────────────────
	RegisterNode(ctx context.Context, node *common.Node) error
	UpsertNode(ctx context.Context, node *common.Node) error
	GetNode(ctx context.Context, nodeID string) (*common.Node, error)
	ListNodes(ctx context.Context) ([]*common.Node, error)
	ListOnlineNodes(ctx context.Context) ([]*common.Node, error)
	UpdateNodeHeartbeat(ctx context.Context, nodeID string, cpu, mem float64, conns int, rx, tx float64, status string) error
	UpdateNodeStatus(ctx context.Context, nodeID, status string) error
	MarkStaleNodesOffline(ctx context.Context, timeoutSec int) (int64, error)

	// ── Node Status Logs ──────────────────────────────────
	InsertNodeStatusLog(ctx context.Context, log *common.NodeStatus) error
	GetLatestNodeStatus(ctx context.Context, nodeID string) (*common.NodeStatus, error)
	GetLatestNodeStatuses(ctx context.Context) (map[string]*common.NodeStatus, error)

	// ── Traffic Logs ──────────────────────────────────────
	InsertTrafficLog(ctx context.Context, log *common.TrafficRecord) error
	BatchInsertTrafficLogs(ctx context.Context, logs []*common.TrafficRecord) error
	GetUserTrafficSummary(ctx context.Context, userID string) (*common.TrafficSummary, error)
	GetNodeTrafficSummary(ctx context.Context, nodeID string) (*common.TrafficSummary, error)
	ListTrafficLogs(ctx context.Context, limit int) ([]*common.TrafficRecord, error)

	// ── Policies ──────────────────────────────────────────
	GetUserPolicy(ctx context.Context, userID string) (*common.Policy, error)
	UpsertPolicy(ctx context.Context, policy *common.Policy) error
	ListPolicies(ctx context.Context) ([]*common.Policy, error)
	DeletePolicy(ctx context.Context, userID string) error
	GetAllActivePolicies(ctx context.Context) ([]*common.Policy, error)

	// ── Scheduler Events ──────────────────────────────────
	InsertSchedulerEvent(ctx context.Context, event *common.SchedulerEvent) error
	ListSchedulerEvents(ctx context.Context, limit int) ([]*common.SchedulerEvent, error)

	// ── Dashboard ─────────────────────────────────────────
	GetDashboardStats(ctx context.Context) (*common.DashboardStats, error)

	// ── Lifecycle ─────────────────────────────────────────
	Ping(ctx context.Context) error
	Close() error
}
