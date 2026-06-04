package db

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
)

// SQLiteDB implements the DB interface using SQLite.
type SQLiteDB struct {
	db *sql.DB

	// Prepared statements cache
	stmtMu    sync.RWMutex
	stmtCache map[string]*sql.Stmt
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    token VARCHAR(128) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    traffic_total BIGINT NOT NULL DEFAULT 0,
    traffic_used BIGINT NOT NULL DEFAULT 0,
    max_rate_mbps INTEGER NOT NULL DEFAULT 10,
    max_connections INTEGER NOT NULL DEFAULT 5,
    expired_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_token ON users(token);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

CREATE TABLE IF NOT EXISTS nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    ip VARCHAR(64) NOT NULL,
    proxy_port INTEGER NOT NULL,
    max_bandwidth_mbps INTEGER NOT NULL DEFAULT 100,
    max_connections INTEGER NOT NULL DEFAULT 1000,
    node_secret VARCHAR(128) NOT NULL,
    weight INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(32) NOT NULL DEFAULT 'offline',
    last_heartbeat TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_nodes_node_id ON nodes(node_id);
CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);

CREATE TABLE IF NOT EXISTS node_status_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id VARCHAR(64) NOT NULL,
    cpu_usage REAL NOT NULL,
    memory_usage REAL NOT NULL,
    current_connections INTEGER NOT NULL,
    rx_mbps REAL NOT NULL,
    tx_mbps REAL NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_node_status_logs_node_id ON node_status_logs(node_id);
CREATE INDEX IF NOT EXISTS idx_node_status_logs_created_at ON node_status_logs(created_at);

CREATE TABLE IF NOT EXISTS traffic_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    upload_bytes BIGINT NOT NULL DEFAULT 0,
    download_bytes BIGINT NOT NULL DEFAULT 0,
    connection_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_traffic_logs_node_user ON traffic_logs(node_id, user_id);
CREATE INDEX IF NOT EXISTS idx_traffic_logs_user_id ON traffic_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_traffic_logs_created_at ON traffic_logs(created_at);

CREATE TABLE IF NOT EXISTS policies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id VARCHAR(64) NOT NULL,
    max_rate_mbps INTEGER NOT NULL,
    burst_mbps INTEGER NOT NULL,
    max_connections INTEGER NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_policies_user_id ON policies(user_id);
CREATE INDEX IF NOT EXISTS idx_policies_status ON policies(status);

CREATE TABLE IF NOT EXISTS scheduler_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id VARCHAR(64),
    selected_node_id VARCHAR(64),
    strategy VARCHAR(64) NOT NULL,
    reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_scheduler_events_created_at ON scheduler_events(created_at);
`

// NewSQLiteDB creates a new SQLite-backed DB.
// dsn is typically a file path or ":memory:" for testing.
func NewSQLiteDB(dsn string) (*SQLiteDB, error) {
	// Enable WAL mode and other pragmas for better concurrency
	pragmas := "_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)"
	fullDSN := dsn
	if dsn != ":memory:" {
		fullDSN = dsn + "?" + pragmas
	}

	sqldb, err := sql.Open("sqlite", fullDSN)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite only supports one writer at a time
	sqldb.SetMaxOpenConns(1)
	sqldb.SetMaxIdleConns(1)

	sdb := &SQLiteDB{
		db:        sqldb,
		stmtCache: make(map[string]*sql.Stmt),
	}

	// Initialize schema
	if _, err := sqldb.Exec(schemaSQL); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return sdb, nil
}

func (s *SQLiteDB) Close() error {
	s.stmtMu.Lock()
	for _, stmt := range s.stmtCache {
		stmt.Close()
	}
	s.stmtCache = nil
	s.stmtMu.Unlock()
	return s.db.Close()
}

func (s *SQLiteDB) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// ────────────────────────────────────────────────────────────
// Users
// ────────────────────────────────────────────────────────────

func (s *SQLiteDB) CreateUser(ctx context.Context, user *common.User) error {
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, token, status, traffic_total, traffic_used, max_rate_mbps, max_connections, expired_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.Username, user.PasswordHash, user.Token, user.Status, user.TrafficTotal,
		user.TrafficUsed, user.MaxRateMbps, user.MaxConnections, user.ExpiredAt, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	id, _ := res.LastInsertId()
	user.ID = id
	return nil
}

func (s *SQLiteDB) GetUser(ctx context.Context, id int64) (*common.User, error) {
	user := &common.User{}
	var expiredAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, token, status, traffic_total, traffic_used,
		        max_rate_mbps, max_connections, expired_at, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Token, &user.Status,
		&user.TrafficTotal, &user.TrafficUsed, &user.MaxRateMbps, &user.MaxConnections,
		&expiredAt, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, common.NewNotFound("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if expiredAt.Valid {
		user.ExpiredAt = &expiredAt.Time
	}
	return user, nil
}

func (s *SQLiteDB) GetUserByToken(ctx context.Context, token string) (*common.User, error) {
	user := &common.User{}
	var expiredAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, token, status, traffic_total, traffic_used,
		        max_rate_mbps, max_connections, expired_at, created_at, updated_at
		 FROM users WHERE token = ?`, token,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Token, &user.Status,
		&user.TrafficTotal, &user.TrafficUsed, &user.MaxRateMbps, &user.MaxConnections,
		&expiredAt, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, common.NewNotFound("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user by token: %w", err)
	}
	if expiredAt.Valid {
		user.ExpiredAt = &expiredAt.Time
	}
	return user, nil
}

func (s *SQLiteDB) GetUserByUsername(ctx context.Context, username string) (*common.User, error) {
	user := &common.User{}
	var expiredAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, token, status, traffic_total, traffic_used,
		        max_rate_mbps, max_connections, expired_at, created_at, updated_at
		 FROM users WHERE username = ?`, username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Token, &user.Status,
		&user.TrafficTotal, &user.TrafficUsed, &user.MaxRateMbps, &user.MaxConnections,
		&expiredAt, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, common.NewNotFound("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	if expiredAt.Valid {
		user.ExpiredAt = &expiredAt.Time
	}
	return user, nil
}

func (s *SQLiteDB) ListUsers(ctx context.Context) ([]*common.User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, username, password_hash, token, status, traffic_total, traffic_used,
		        max_rate_mbps, max_connections, expired_at, created_at, updated_at
		 FROM users ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*common.User
	for rows.Next() {
		user := &common.User{}
		var expiredAt sql.NullTime
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Token,
			&user.Status, &user.TrafficTotal, &user.TrafficUsed, &user.MaxRateMbps,
			&user.MaxConnections, &expiredAt, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if expiredAt.Valid {
			user.ExpiredAt = &expiredAt.Time
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *SQLiteDB) UpdateUser(ctx context.Context, user *common.User) error {
	user.UpdatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET username=?, password_hash=?, token=?, status=?, traffic_total=?,
		 traffic_used=?, max_rate_mbps=?, max_connections=?, expired_at=?, updated_at=?
		 WHERE id=?`,
		user.Username, user.PasswordHash, user.Token, user.Status, user.TrafficTotal,
		user.TrafficUsed, user.MaxRateMbps, user.MaxConnections, user.ExpiredAt,
		user.UpdatedAt, user.ID)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (s *SQLiteDB) DeleteUser(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (s *SQLiteDB) AddUserTraffic(ctx context.Context, userID string, bytes int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET traffic_used = traffic_used + ?, updated_at = ? WHERE username = ?`,
		bytes, time.Now().UTC(), userID)
	return err
}

// ────────────────────────────────────────────────────────────
// Nodes
// ────────────────────────────────────────────────────────────

func (s *SQLiteDB) RegisterNode(ctx context.Context, node *common.Node) error {
	now := time.Now().UTC()
	node.CreatedAt = now
	node.UpdatedAt = now
	if node.Status == "" {
		node.Status = common.NodeStatusOnline
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO nodes (node_id, name, ip, proxy_port, max_bandwidth_mbps, max_connections,
		 node_secret, weight, status, last_heartbeat, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		node.NodeID, node.Name, node.IP, node.ProxyPort, node.MaxBandwidthMbps,
		node.MaxConnections, node.NodeSecret, node.Weight, node.Status,
		node.LastHeartbeat, node.CreatedAt, node.UpdatedAt)
	if err != nil {
		return fmt.Errorf("register node: %w", err)
	}
	return nil
}

func (s *SQLiteDB) UpsertNode(ctx context.Context, node *common.Node) error {
	now := time.Now().UTC()
	node.UpdatedAt = now

	res, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET name=?, ip=?, proxy_port=?, max_bandwidth_mbps=?, max_connections=?,
		 node_secret=?, weight=?, status=?, last_heartbeat=?, updated_at=?
		 WHERE node_id=?`,
		node.Name, node.IP, node.ProxyPort, node.MaxBandwidthMbps, node.MaxConnections,
		node.NodeSecret, node.Weight, node.Status, node.LastHeartbeat, node.UpdatedAt,
		node.NodeID)
	if err != nil {
		return fmt.Errorf("upsert node: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return s.RegisterNode(ctx, node)
	}
	return nil
}

func (s *SQLiteDB) GetNode(ctx context.Context, nodeID string) (*common.Node, error) {
	node := &common.Node{}
	var lastHeartbeat sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT id, node_id, name, ip, proxy_port, max_bandwidth_mbps, max_connections,
		        node_secret, weight, status, last_heartbeat, created_at, updated_at
		 FROM nodes WHERE node_id = ?`, nodeID,
	).Scan(&node.ID, &node.NodeID, &node.Name, &node.IP, &node.ProxyPort,
		&node.MaxBandwidthMbps, &node.MaxConnections, &node.NodeSecret, &node.Weight,
		&node.Status, &lastHeartbeat, &node.CreatedAt, &node.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, common.NewNotFound("node not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}
	if lastHeartbeat.Valid {
		node.LastHeartbeat = &lastHeartbeat.Time
	}
	return node, nil
}

func (s *SQLiteDB) ListNodes(ctx context.Context) ([]*common.Node, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, node_id, name, ip, proxy_port, max_bandwidth_mbps, max_connections,
		        node_secret, weight, status, last_heartbeat, created_at, updated_at
		 FROM nodes ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	defer rows.Close()
	return scanNodes(rows)
}

func (s *SQLiteDB) ListOnlineNodes(ctx context.Context) ([]*common.Node, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, node_id, name, ip, proxy_port, max_bandwidth_mbps, max_connections,
		        node_secret, weight, status, last_heartbeat, created_at, updated_at
		 FROM nodes WHERE status IN (?, ?) ORDER BY id`,
		common.NodeStatusOnline, common.NodeStatusDegraded)
	if err != nil {
		return nil, fmt.Errorf("list online nodes: %w", err)
	}
	defer rows.Close()
	return scanNodes(rows)
}

func (s *SQLiteDB) UpdateNodeHeartbeat(ctx context.Context, nodeID string, cpu, mem float64, conns int, rx, tx float64, status string) error {
	now := time.Now().UTC()

	// Update node heartbeat
	_, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET status=?, last_heartbeat=?, updated_at=? WHERE node_id=?`,
		status, now, now, nodeID)
	if err != nil {
		return fmt.Errorf("update node heartbeat: %w", err)
	}

	// Insert status log
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO node_status_logs (node_id, cpu_usage, memory_usage, current_connections, rx_mbps, tx_mbps, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		nodeID, cpu, mem, conns, rx, tx, status, now)
	if err != nil {
		return fmt.Errorf("insert node status log: %w", err)
	}

	return nil
}

func (s *SQLiteDB) UpdateNodeStatus(ctx context.Context, nodeID, status string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET status=?, updated_at=? WHERE node_id=?`,
		status, time.Now().UTC(), nodeID)
	return err
}

func (s *SQLiteDB) MarkStaleNodesOffline(ctx context.Context, timeoutSec int) (int64, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(timeoutSec) * time.Second)
	res, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET status=?, updated_at=? WHERE status IN (?, ?) AND (last_heartbeat IS NULL OR last_heartbeat < ?)`,
		common.NodeStatusOffline, time.Now().UTC(),
		common.NodeStatusOnline, common.NodeStatusDegraded, cutoff)
	if err != nil {
		return 0, fmt.Errorf("mark stale nodes: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ────────────────────────────────────────────────────────────
// Node Status Logs
// ────────────────────────────────────────────────────────────

func (s *SQLiteDB) InsertNodeStatusLog(ctx context.Context, log *common.NodeStatus) error {
	log.CreatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO node_status_logs (node_id, cpu_usage, memory_usage, current_connections, rx_mbps, tx_mbps, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		log.NodeID, log.CPUUsage, log.MemoryUsage, log.CurrentConnections,
		log.RxMbps, log.TxMbps, log.Status, log.CreatedAt)
	return err
}

func (s *SQLiteDB) GetLatestNodeStatus(ctx context.Context, nodeID string) (*common.NodeStatus, error) {
	ns := &common.NodeStatus{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, node_id, cpu_usage, memory_usage, current_connections, rx_mbps, tx_mbps, status, created_at
		 FROM node_status_logs WHERE node_id = ? ORDER BY created_at DESC LIMIT 1`, nodeID,
	).Scan(&ns.ID, &ns.NodeID, &ns.CPUUsage, &ns.MemoryUsage, &ns.CurrentConnections,
		&ns.RxMbps, &ns.TxMbps, &ns.Status, &ns.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest node status: %w", err)
	}
	return ns, nil
}

func (s *SQLiteDB) GetLatestNodeStatuses(ctx context.Context) (map[string]*common.NodeStatus, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT nsl.id, nsl.node_id, nsl.cpu_usage, nsl.memory_usage, nsl.current_connections,
		        nsl.rx_mbps, nsl.tx_mbps, nsl.status, nsl.created_at
		 FROM node_status_logs nsl
		 INNER JOIN (
		     SELECT node_id, MAX(created_at) AS max_created
		     FROM node_status_logs GROUP BY node_id
		 ) latest ON nsl.node_id = latest.node_id AND nsl.created_at = latest.max_created`)
	if err != nil {
		return nil, fmt.Errorf("get latest node statuses: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*common.NodeStatus)
	for rows.Next() {
		ns := &common.NodeStatus{}
		if err := rows.Scan(&ns.ID, &ns.NodeID, &ns.CPUUsage, &ns.MemoryUsage,
			&ns.CurrentConnections, &ns.RxMbps, &ns.TxMbps, &ns.Status, &ns.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan node status: %w", err)
		}
		result[ns.NodeID] = ns
	}
	return result, rows.Err()
}

// ────────────────────────────────────────────────────────────
// Traffic Logs
// ────────────────────────────────────────────────────────────

func (s *SQLiteDB) InsertTrafficLog(ctx context.Context, log *common.TrafficRecord) error {
	log.CreatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO traffic_logs (node_id, user_id, upload_bytes, download_bytes, connection_count, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		log.NodeID, log.UserID, log.UploadBytes, log.DownloadBytes, log.ConnectionCount, log.CreatedAt)
	return err
}

func (s *SQLiteDB) BatchInsertTrafficLogs(ctx context.Context, logs []*common.TrafficRecord) error {
	if len(logs) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO traffic_logs (node_id, user_id, upload_bytes, download_bytes, connection_count, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC()
	for _, log := range logs {
		log.CreatedAt = now
		_, err := stmt.ExecContext(ctx, log.NodeID, log.UserID, log.UploadBytes, log.DownloadBytes, log.ConnectionCount, now)
		if err != nil {
			return fmt.Errorf("exec batch insert: %w", err)
		}
	}

	return tx.Commit()
}

func (s *SQLiteDB) GetUserTrafficSummary(ctx context.Context, userID string) (*common.TrafficSummary, error) {
	summary := &common.TrafficSummary{UserID: userID}
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(upload_bytes),0), COALESCE(SUM(download_bytes),0), COALESCE(SUM(connection_count),0)
		 FROM traffic_logs WHERE user_id = ?`, userID,
	).Scan(&summary.TotalUpload, &summary.TotalDownload, &summary.ConnectionCount)
	if err != nil {
		return nil, fmt.Errorf("get user traffic summary: %w", err)
	}
	summary.TotalBytes = summary.TotalUpload + summary.TotalDownload
	return summary, nil
}

func (s *SQLiteDB) GetNodeTrafficSummary(ctx context.Context, nodeID string) (*common.TrafficSummary, error) {
	summary := &common.TrafficSummary{NodeID: nodeID}
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(upload_bytes),0), COALESCE(SUM(download_bytes),0), COALESCE(SUM(connection_count),0)
		 FROM traffic_logs WHERE node_id = ?`, nodeID,
	).Scan(&summary.TotalUpload, &summary.TotalDownload, &summary.ConnectionCount)
	if err != nil {
		return nil, fmt.Errorf("get node traffic summary: %w", err)
	}
	summary.TotalBytes = summary.TotalUpload + summary.TotalDownload
	return summary, nil
}

func (s *SQLiteDB) ListTrafficLogs(ctx context.Context, limit int) ([]*common.TrafficRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, node_id, user_id, upload_bytes, download_bytes, connection_count, created_at
		 FROM traffic_logs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list traffic logs: %w", err)
	}
	defer rows.Close()

	var logs []*common.TrafficRecord
	for rows.Next() {
		l := &common.TrafficRecord{}
		if err := rows.Scan(&l.ID, &l.NodeID, &l.UserID, &l.UploadBytes, &l.DownloadBytes, &l.ConnectionCount, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan traffic log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// ────────────────────────────────────────────────────────────
// Policies
// ────────────────────────────────────────────────────────────

func (s *SQLiteDB) GetUserPolicy(ctx context.Context, userID string) (*common.Policy, error) {
	p := &common.Policy{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, max_rate_mbps, burst_mbps, max_connections, priority, status, created_at, updated_at
		 FROM policies WHERE user_id = ? AND status = ?`, userID, common.PolicyStatusActive,
	).Scan(&p.ID, &p.UserID, &p.MaxRateMbps, &p.BurstMbps, &p.MaxConnections, &p.Priority, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil // No policy exists, caller should use defaults
	}
	if err != nil {
		return nil, fmt.Errorf("get user policy: %w", err)
	}
	return p, nil
}

func (s *SQLiteDB) UpsertPolicy(ctx context.Context, policy *common.Policy) error {
	now := time.Now().UTC()

	// Try update first
	res, err := s.db.ExecContext(ctx,
		`UPDATE policies SET max_rate_mbps=?, burst_mbps=?, max_connections=?, priority=?, status=?, updated_at=?
		 WHERE user_id=?`,
		policy.MaxRateMbps, policy.BurstMbps, policy.MaxConnections, policy.Priority, policy.Status, now, policy.UserID)
	if err != nil {
		return fmt.Errorf("upsert policy: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		policy.CreatedAt = now
		policy.UpdatedAt = now
		_, err = s.db.ExecContext(ctx,
			`INSERT INTO policies (user_id, max_rate_mbps, burst_mbps, max_connections, priority, status, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			policy.UserID, policy.MaxRateMbps, policy.BurstMbps, policy.MaxConnections,
			policy.Priority, policy.Status, policy.CreatedAt, policy.UpdatedAt)
		if err != nil {
			return fmt.Errorf("insert policy: %w", err)
		}
	}
	return nil
}

func (s *SQLiteDB) ListPolicies(ctx context.Context) ([]*common.Policy, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, max_rate_mbps, burst_mbps, max_connections, priority, status, created_at, updated_at
		 FROM policies ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}
	defer rows.Close()

	var policies []*common.Policy
	for rows.Next() {
		p := &common.Policy{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.MaxRateMbps, &p.BurstMbps, &p.MaxConnections, &p.Priority, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan policy: %w", err)
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

func (s *SQLiteDB) DeletePolicy(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM policies WHERE user_id=?`, userID)
	return err
}

func (s *SQLiteDB) GetAllActivePolicies(ctx context.Context) ([]*common.Policy, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, max_rate_mbps, burst_mbps, max_connections, priority, status, created_at, updated_at
		 FROM policies WHERE status = ?`, common.PolicyStatusActive)
	if err != nil {
		return nil, fmt.Errorf("get all active policies: %w", err)
	}
	defer rows.Close()

	var policies []*common.Policy
	for rows.Next() {
		p := &common.Policy{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.MaxRateMbps, &p.BurstMbps, &p.MaxConnections, &p.Priority, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan policy: %w", err)
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

// ────────────────────────────────────────────────────────────
// Scheduler Events
// ────────────────────────────────────────────────────────────

func (s *SQLiteDB) InsertSchedulerEvent(ctx context.Context, event *common.SchedulerEvent) error {
	event.CreatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO scheduler_events (user_id, selected_node_id, strategy, reason, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		event.UserID, event.SelectedNodeID, event.Strategy, event.Reason, event.CreatedAt)
	return err
}

func (s *SQLiteDB) ListSchedulerEvents(ctx context.Context, limit int) ([]*common.SchedulerEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, selected_node_id, strategy, reason, created_at
		 FROM scheduler_events ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list scheduler events: %w", err)
	}
	defer rows.Close()

	var events []*common.SchedulerEvent
	for rows.Next() {
		e := &common.SchedulerEvent{}
		var userID, nodeID sql.NullString
		if err := rows.Scan(&e.ID, &userID, &nodeID, &e.Strategy, &e.Reason, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan scheduler event: %w", err)
		}
		if userID.Valid {
			e.UserID = userID.String
		}
		if nodeID.Valid {
			e.SelectedNodeID = nodeID.String
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// ────────────────────────────────────────────────────────────
// Dashboard
// ────────────────────────────────────────────────────────────

func (s *SQLiteDB) GetDashboardStats(ctx context.Context) (*common.DashboardStats, error) {
	stats := &common.DashboardStats{}

	// Node counts
	rows, err := s.db.QueryContext(ctx,
		`SELECT status, COUNT(*) FROM nodes GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("get dashboard stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats.TotalNodes += count
		switch status {
		case common.NodeStatusOnline:
			stats.OnlineNodes = count
		case common.NodeStatusOffline:
			stats.OfflineNodes = count
		case common.NodeStatusDegraded:
			stats.DegradedNodes = count
		}
	}

	// User counts
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), SUM(CASE WHEN status=? THEN 1 ELSE 0 END) FROM users`,
		common.UserStatusActive).Scan(&stats.TotalUsers, &stats.ActiveUsers)

	// Traffic totals (last 24h)
	s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(upload_bytes),0), COALESCE(SUM(download_bytes),0)
		 FROM traffic_logs WHERE created_at > ?`,
		time.Now().UTC().Add(-24*time.Hour),
	).Scan(&stats.TotalUpload, &stats.TotalDownload)

	// Active connections estimate (sum from latest node status)
	statuses, _ := s.GetLatestNodeStatuses(ctx)
	for _, ns := range statuses {
		stats.ActiveConnections += ns.CurrentConnections
	}

	return stats, nil
}

// ────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────

func scanNodes(rows *sql.Rows) ([]*common.Node, error) {
	var nodes []*common.Node
	for rows.Next() {
		node := &common.Node{}
		var lastHeartbeat sql.NullTime
		if err := rows.Scan(&node.ID, &node.NodeID, &node.Name, &node.IP, &node.ProxyPort,
			&node.MaxBandwidthMbps, &node.MaxConnections, &node.NodeSecret, &node.Weight,
			&node.Status, &lastHeartbeat, &node.CreatedAt, &node.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		if lastHeartbeat.Valid {
			node.LastHeartbeat = &lastHeartbeat.Time
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}
