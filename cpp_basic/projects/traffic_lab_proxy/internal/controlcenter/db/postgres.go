package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
)

// PostgresDB implements the DB interface using PostgreSQL.
// It reuses the same interface as SQLiteDB and can serve as a
// drop-in replacement when PostgreSQL is available.
type PostgresDB struct {
	db *sql.DB
}

// NewPostgresDB creates a new PostgreSQL-backed DB.
// dsn is a PostgreSQL connection string, e.g.:
//
//	"host=localhost port=5432 user=trafficlab password=trafficlab dbname=trafficlab sslmode=disable"
func NewPostgresDB(dsn string) (*PostgresDB, error) {
	sqldb, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	sqldb.SetMaxOpenConns(25)
	sqldb.SetMaxIdleConns(10)
	sqldb.SetConnMaxLifetime(5 * time.Minute)

	pdb := &PostgresDB{db: sqldb}

	if err := pdb.initSchema(); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return pdb, nil
}

func (p *PostgresDB) initSchema() error {
	// PostgreSQL-compatible schema (with IF NOT EXISTS)
	schema := `
	CREATE TABLE IF NOT EXISTS users (
	    id SERIAL PRIMARY KEY,
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

	CREATE TABLE IF NOT EXISTS nodes (
	    id SERIAL PRIMARY KEY,
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

	CREATE TABLE IF NOT EXISTS node_status_logs (
	    id SERIAL PRIMARY KEY,
	    node_id VARCHAR(64) NOT NULL,
	    cpu_usage DOUBLE PRECISION NOT NULL,
	    memory_usage DOUBLE PRECISION NOT NULL,
	    current_connections INTEGER NOT NULL,
	    rx_mbps DOUBLE PRECISION NOT NULL,
	    tx_mbps DOUBLE PRECISION NOT NULL,
	    status VARCHAR(32) NOT NULL,
	    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS traffic_logs (
	    id SERIAL PRIMARY KEY,
	    node_id VARCHAR(64) NOT NULL,
	    user_id VARCHAR(64) NOT NULL,
	    upload_bytes BIGINT NOT NULL DEFAULT 0,
	    download_bytes BIGINT NOT NULL DEFAULT 0,
	    connection_count INTEGER NOT NULL DEFAULT 0,
	    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS policies (
	    id SERIAL PRIMARY KEY,
	    user_id VARCHAR(64) NOT NULL,
	    max_rate_mbps INTEGER NOT NULL,
	    burst_mbps INTEGER NOT NULL,
	    max_connections INTEGER NOT NULL,
	    priority INTEGER NOT NULL DEFAULT 0,
	    status VARCHAR(32) NOT NULL DEFAULT 'active',
	    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS scheduler_events (
	    id SERIAL PRIMARY KEY,
	    user_id VARCHAR(64),
	    selected_node_id VARCHAR(64),
	    strategy VARCHAR(64) NOT NULL,
	    reason TEXT,
	    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := p.db.Exec(schema)
	return err
}

func (p *PostgresDB) Close() error { return p.db.Close() }
func (p *PostgresDB) Ping(ctx context.Context) error { return p.db.PingContext(ctx) }

// User methods — delegate to shared SQL (parameter syntax differs with $N)
func (p *PostgresDB) CreateUser(ctx context.Context, user *common.User) error {
	now := time.Now().UTC()
	err := p.db.QueryRowContext(ctx,
		`INSERT INTO users (username, password_hash, token, status, traffic_total, traffic_used, max_rate_mbps, max_connections, expired_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		user.Username, user.PasswordHash, user.Token, user.Status, user.TrafficTotal,
		user.TrafficUsed, user.MaxRateMbps, user.MaxConnections, user.ExpiredAt, now, now,
	).Scan(&user.ID)
	user.CreatedAt = now
	user.UpdatedAt = now
	return err
}

func (p *PostgresDB) GetUser(ctx context.Context, id int64) (*common.User, error) {
	user := &common.User{}
	var expiredAt sql.NullTime
	err := p.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, token, status, traffic_total, traffic_used,
		        max_rate_mbps, max_connections, expired_at, created_at, updated_at
		 FROM users WHERE id=$1`, id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Token, &user.Status,
		&user.TrafficTotal, &user.TrafficUsed, &user.MaxRateMbps, &user.MaxConnections,
		&expiredAt, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, common.NewNotFound("user not found")
	}
	if expiredAt.Valid {
		user.ExpiredAt = &expiredAt.Time
	}
	return user, err
}

func (p *PostgresDB) GetUserByToken(ctx context.Context, token string) (*common.User, error) {
	user := &common.User{}
	var expiredAt sql.NullTime
	err := p.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, token, status, traffic_total, traffic_used,
		        max_rate_mbps, max_connections, expired_at, created_at, updated_at
		 FROM users WHERE token=$1`, token,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Token, &user.Status,
		&user.TrafficTotal, &user.TrafficUsed, &user.MaxRateMbps, &user.MaxConnections,
		&expiredAt, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, common.NewNotFound("user not found")
	}
	return user, err
}

func (p *PostgresDB) GetUserByUsername(ctx context.Context, username string) (*common.User, error) {
	user := &common.User{}
	var expiredAt sql.NullTime
	err := p.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, token, status, traffic_total, traffic_used,
		        max_rate_mbps, max_connections, expired_at, created_at, updated_at
		 FROM users WHERE username=$1`, username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Token, &user.Status,
		&user.TrafficTotal, &user.TrafficUsed, &user.MaxRateMbps, &user.MaxConnections,
		&expiredAt, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, common.NewNotFound("user not found")
	}
	return user, err
}

func (p *PostgresDB) ListUsers(ctx context.Context) ([]*common.User, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, username, password_hash, token, status, traffic_total, traffic_used,
		        max_rate_mbps, max_connections, expired_at, created_at, updated_at FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows)
}

func (p *PostgresDB) UpdateUser(ctx context.Context, user *common.User) error {
	user.UpdatedAt = time.Now().UTC()
	_, err := p.db.ExecContext(ctx,
		`UPDATE users SET username=$1, password_hash=$2, token=$3, status=$4, traffic_total=$5,
		 traffic_used=$6, max_rate_mbps=$7, max_connections=$8, expired_at=$9, updated_at=$10 WHERE id=$11`,
		user.Username, user.PasswordHash, user.Token, user.Status, user.TrafficTotal,
		user.TrafficUsed, user.MaxRateMbps, user.MaxConnections, user.ExpiredAt, user.UpdatedAt, user.ID)
	return err
}

func (p *PostgresDB) DeleteUser(ctx context.Context, id int64) error {
	_, err := p.db.ExecContext(ctx, `DELETE FROM users WHERE id=$1`, id)
	return err
}

func (p *PostgresDB) AddUserTraffic(ctx context.Context, userID string, bytes int64) error {
	_, err := p.db.ExecContext(ctx,
		`UPDATE users SET traffic_used = traffic_used + $1, updated_at = $2 WHERE username = $3`,
		bytes, time.Now().UTC(), userID)
	return err
}

// Node methods
func (p *PostgresDB) RegisterNode(ctx context.Context, node *common.Node) error {
	now := time.Now().UTC()
	node.CreatedAt = now
	node.UpdatedAt = now
	if node.Status == "" {
		node.Status = common.NodeStatusOnline
	}
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO nodes (node_id, name, ip, proxy_port, max_bandwidth_mbps, max_connections,
		 node_secret, weight, status, last_heartbeat, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		node.NodeID, node.Name, node.IP, node.ProxyPort, node.MaxBandwidthMbps,
		node.MaxConnections, node.NodeSecret, node.Weight, node.Status, node.LastHeartbeat, now, now)
	return err
}

func (p *PostgresDB) UpsertNode(ctx context.Context, node *common.Node) error {
	now := time.Now().UTC()
	node.UpdatedAt = now
	res, err := p.db.ExecContext(ctx,
		`UPDATE nodes SET name=$1, ip=$2, proxy_port=$3, max_bandwidth_mbps=$4, max_connections=$5,
		 node_secret=$6, weight=$7, status=$8, last_heartbeat=$9, updated_at=$10 WHERE node_id=$11`,
		node.Name, node.IP, node.ProxyPort, node.MaxBandwidthMbps, node.MaxConnections,
		node.NodeSecret, node.Weight, node.Status, node.LastHeartbeat, now, node.NodeID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return p.RegisterNode(ctx, node)
	}
	return nil
}

func (p *PostgresDB) GetNode(ctx context.Context, nodeID string) (*common.Node, error) {
	node := &common.Node{}
	var lastHeartbeat sql.NullTime
	err := p.db.QueryRowContext(ctx,
		`SELECT id, node_id, name, ip, proxy_port, max_bandwidth_mbps, max_connections,
		        node_secret, weight, status, last_heartbeat, created_at, updated_at
		 FROM nodes WHERE node_id=$1`, nodeID,
	).Scan(&node.ID, &node.NodeID, &node.Name, &node.IP, &node.ProxyPort,
		&node.MaxBandwidthMbps, &node.MaxConnections, &node.NodeSecret, &node.Weight,
		&node.Status, &lastHeartbeat, &node.CreatedAt, &node.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, common.NewNotFound("node not found")
	}
	if lastHeartbeat.Valid {
		node.LastHeartbeat = &lastHeartbeat.Time
	}
	return node, err
}

func (p *PostgresDB) ListNodes(ctx context.Context) ([]*common.Node, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, node_id, name, ip, proxy_port, max_bandwidth_mbps, max_connections,
		        node_secret, weight, status, last_heartbeat, created_at, updated_at FROM nodes ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodesPg(rows)
}

func (p *PostgresDB) ListOnlineNodes(ctx context.Context) ([]*common.Node, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, node_id, name, ip, proxy_port, max_bandwidth_mbps, max_connections,
		        node_secret, weight, status, last_heartbeat, created_at, updated_at
		 FROM nodes WHERE status IN ($1,$2) ORDER BY id`,
		common.NodeStatusOnline, common.NodeStatusDegraded)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodesPg(rows)
}

func (p *PostgresDB) UpdateNodeHeartbeat(ctx context.Context, nodeID string, cpu, mem float64, conns int, rx, tx float64, status string) error {
	now := time.Now().UTC()
	_, err := p.db.ExecContext(ctx,
		`UPDATE nodes SET status=$1, last_heartbeat=$2, updated_at=$3 WHERE node_id=$4`,
		status, now, now, nodeID)
	if err != nil {
		return err
	}
	_, err = p.db.ExecContext(ctx,
		`INSERT INTO node_status_logs (node_id, cpu_usage, memory_usage, current_connections, rx_mbps, tx_mbps, status, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		nodeID, cpu, mem, conns, rx, tx, status, now)
	return err
}

func (p *PostgresDB) UpdateNodeStatus(ctx context.Context, nodeID, status string) error {
	_, err := p.db.ExecContext(ctx,
		`UPDATE nodes SET status=$1, updated_at=$2 WHERE node_id=$3`,
		status, time.Now().UTC(), nodeID)
	return err
}

func (p *PostgresDB) MarkStaleNodesOffline(ctx context.Context, timeoutSec int) (int64, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(timeoutSec) * time.Second)
	res, err := p.db.ExecContext(ctx,
		`UPDATE nodes SET status=$1, updated_at=$2 WHERE status IN ($3,$4) AND (last_heartbeat IS NULL OR last_heartbeat < $5)`,
		common.NodeStatusOffline, time.Now().UTC(), common.NodeStatusOnline, common.NodeStatusDegraded, cutoff)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// Status logs
func (p *PostgresDB) InsertNodeStatusLog(ctx context.Context, log *common.NodeStatus) error {
	log.CreatedAt = time.Now().UTC()
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO node_status_logs (node_id, cpu_usage, memory_usage, current_connections, rx_mbps, tx_mbps, status, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		log.NodeID, log.CPUUsage, log.MemoryUsage, log.CurrentConnections, log.RxMbps, log.TxMbps, log.Status, log.CreatedAt)
	return err
}

func (p *PostgresDB) GetLatestNodeStatus(ctx context.Context, nodeID string) (*common.NodeStatus, error) {
	ns := &common.NodeStatus{}
	err := p.db.QueryRowContext(ctx,
		`SELECT id, node_id, cpu_usage, memory_usage, current_connections, rx_mbps, tx_mbps, status, created_at
		 FROM node_status_logs WHERE node_id=$1 ORDER BY created_at DESC LIMIT 1`, nodeID,
	).Scan(&ns.ID, &ns.NodeID, &ns.CPUUsage, &ns.MemoryUsage, &ns.CurrentConnections, &ns.RxMbps, &ns.TxMbps, &ns.Status, &ns.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return ns, err
}

func (p *PostgresDB) GetLatestNodeStatuses(ctx context.Context) (map[string]*common.NodeStatus, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT DISTINCT ON (node_id) id, node_id, cpu_usage, memory_usage, current_connections,
		        rx_mbps, tx_mbps, status, created_at
		 FROM node_status_logs ORDER BY node_id, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]*common.NodeStatus)
	for rows.Next() {
		ns := &common.NodeStatus{}
		if err := rows.Scan(&ns.ID, &ns.NodeID, &ns.CPUUsage, &ns.MemoryUsage,
			&ns.CurrentConnections, &ns.RxMbps, &ns.TxMbps, &ns.Status, &ns.CreatedAt); err != nil {
			return nil, err
		}
		result[ns.NodeID] = ns
	}
	return result, rows.Err()
}

// Traffic logs
func (p *PostgresDB) InsertTrafficLog(ctx context.Context, log *common.TrafficRecord) error {
	log.CreatedAt = time.Now().UTC()
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO traffic_logs (node_id, user_id, upload_bytes, download_bytes, connection_count, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		log.NodeID, log.UserID, log.UploadBytes, log.DownloadBytes, log.ConnectionCount, log.CreatedAt)
	return err
}

func (p *PostgresDB) BatchInsertTrafficLogs(ctx context.Context, logs []*common.TrafficRecord) error {
	if len(logs) == 0 {
		return nil
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO traffic_logs (node_id, user_id, upload_bytes, download_bytes, connection_count, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	now := time.Now().UTC()
	for _, log := range logs {
		log.CreatedAt = now
		if _, err := stmt.ExecContext(ctx, log.NodeID, log.UserID, log.UploadBytes, log.DownloadBytes, log.ConnectionCount, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *PostgresDB) GetUserTrafficSummary(ctx context.Context, userID string) (*common.TrafficSummary, error) {
	s := &common.TrafficSummary{UserID: userID}
	err := p.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(upload_bytes),0), COALESCE(SUM(download_bytes),0), COALESCE(SUM(connection_count),0)
		 FROM traffic_logs WHERE user_id=$1`, userID,
	).Scan(&s.TotalUpload, &s.TotalDownload, &s.ConnectionCount)
	s.TotalBytes = s.TotalUpload + s.TotalDownload
	return s, err
}

func (p *PostgresDB) GetNodeTrafficSummary(ctx context.Context, nodeID string) (*common.TrafficSummary, error) {
	s := &common.TrafficSummary{NodeID: nodeID}
	err := p.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(upload_bytes),0), COALESCE(SUM(download_bytes),0), COALESCE(SUM(connection_count),0)
		 FROM traffic_logs WHERE node_id=$1`, nodeID,
	).Scan(&s.TotalUpload, &s.TotalDownload, &s.ConnectionCount)
	s.TotalBytes = s.TotalUpload + s.TotalDownload
	return s, err
}

func (p *PostgresDB) ListTrafficLogs(ctx context.Context, limit int) ([]*common.TrafficRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, node_id, user_id, upload_bytes, download_bytes, connection_count, created_at
		 FROM traffic_logs ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []*common.TrafficRecord
	for rows.Next() {
		l := &common.TrafficRecord{}
		if err := rows.Scan(&l.ID, &l.NodeID, &l.UserID, &l.UploadBytes, &l.DownloadBytes, &l.ConnectionCount, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// Policies
func (p *PostgresDB) GetUserPolicy(ctx context.Context, userID string) (*common.Policy, error) {
	pol := &common.Policy{}
	err := p.db.QueryRowContext(ctx,
		`SELECT id, user_id, max_rate_mbps, burst_mbps, max_connections, priority, status, created_at, updated_at
		 FROM policies WHERE user_id=$1 AND status=$2`, userID, common.PolicyStatusActive,
	).Scan(&pol.ID, &pol.UserID, &pol.MaxRateMbps, &pol.BurstMbps, &pol.MaxConnections, &pol.Priority, &pol.Status, &pol.CreatedAt, &pol.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return pol, err
}

func (p *PostgresDB) UpsertPolicy(ctx context.Context, policy *common.Policy) error {
	now := time.Now().UTC()
	res, err := p.db.ExecContext(ctx,
		`UPDATE policies SET max_rate_mbps=$1, burst_mbps=$2, max_connections=$3, priority=$4, status=$5, updated_at=$6
		 WHERE user_id=$7`,
		policy.MaxRateMbps, policy.BurstMbps, policy.MaxConnections, policy.Priority, policy.Status, now, policy.UserID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		policy.CreatedAt = now
		policy.UpdatedAt = now
		_, err = p.db.ExecContext(ctx,
			`INSERT INTO policies (user_id, max_rate_mbps, burst_mbps, max_connections, priority, status, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			policy.UserID, policy.MaxRateMbps, policy.BurstMbps, policy.MaxConnections, policy.Priority, policy.Status, policy.CreatedAt, policy.UpdatedAt)
	}
	return err
}

func (p *PostgresDB) ListPolicies(ctx context.Context) ([]*common.Policy, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, user_id, max_rate_mbps, burst_mbps, max_connections, priority, status, created_at, updated_at FROM policies ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var policies []*common.Policy
	for rows.Next() {
		pol := &common.Policy{}
		if err := rows.Scan(&pol.ID, &pol.UserID, &pol.MaxRateMbps, &pol.BurstMbps, &pol.MaxConnections, &pol.Priority, &pol.Status, &pol.CreatedAt, &pol.UpdatedAt); err != nil {
			return nil, err
		}
		policies = append(policies, pol)
	}
	return policies, rows.Err()
}

func (p *PostgresDB) DeletePolicy(ctx context.Context, userID string) error {
	_, err := p.db.ExecContext(ctx, `DELETE FROM policies WHERE user_id=$1`, userID)
	return err
}

func (p *PostgresDB) GetAllActivePolicies(ctx context.Context) ([]*common.Policy, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, user_id, max_rate_mbps, burst_mbps, max_connections, priority, status, created_at, updated_at
		 FROM policies WHERE status=$1`, common.PolicyStatusActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var policies []*common.Policy
	for rows.Next() {
		pol := &common.Policy{}
		if err := rows.Scan(&pol.ID, &pol.UserID, &pol.MaxRateMbps, &pol.BurstMbps, &pol.MaxConnections, &pol.Priority, &pol.Status, &pol.CreatedAt, &pol.UpdatedAt); err != nil {
			return nil, err
		}
		policies = append(policies, pol)
	}
	return policies, rows.Err()
}

// Scheduler events
func (p *PostgresDB) InsertSchedulerEvent(ctx context.Context, event *common.SchedulerEvent) error {
	event.CreatedAt = time.Now().UTC()
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO scheduler_events (user_id, selected_node_id, strategy, reason, created_at)
		 VALUES ($1,$2,$3,$4,$5)`,
		event.UserID, event.SelectedNodeID, event.Strategy, event.Reason, event.CreatedAt)
	return err
}

func (p *PostgresDB) ListSchedulerEvents(ctx context.Context, limit int) ([]*common.SchedulerEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, user_id, selected_node_id, strategy, reason, created_at
		 FROM scheduler_events ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []*common.SchedulerEvent
	for rows.Next() {
		e := &common.SchedulerEvent{}
		var userID, nodeID sql.NullString
		if err := rows.Scan(&e.ID, &userID, &nodeID, &e.Strategy, &e.Reason, &e.CreatedAt); err != nil {
			return nil, err
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

// Dashboard
func (p *PostgresDB) GetDashboardStats(ctx context.Context) (*common.DashboardStats, error) {
	stats := &common.DashboardStats{}
	rows, err := p.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM nodes GROUP BY status`)
	if err != nil {
		return nil, err
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
	p.db.QueryRowContext(ctx,
		`SELECT COUNT(*), SUM(CASE WHEN status=$1 THEN 1 ELSE 0 END) FROM users`,
		common.UserStatusActive).Scan(&stats.TotalUsers, &stats.ActiveUsers)
	p.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(upload_bytes),0), COALESCE(SUM(download_bytes),0)
		 FROM traffic_logs WHERE created_at > $1`,
		time.Now().UTC().Add(-24*time.Hour),
	).Scan(&stats.TotalUpload, &stats.TotalDownload)
	statuses, _ := p.GetLatestNodeStatuses(ctx)
	for _, ns := range statuses {
		stats.ActiveConnections += ns.CurrentConnections
	}
	return stats, nil
}

// ────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────

func scanUsers(rows *sql.Rows) ([]*common.User, error) {
	var users []*common.User
	for rows.Next() {
		user := &common.User{}
		var expiredAt sql.NullTime
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Token,
			&user.Status, &user.TrafficTotal, &user.TrafficUsed, &user.MaxRateMbps,
			&user.MaxConnections, &expiredAt, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		if expiredAt.Valid {
			user.ExpiredAt = &expiredAt.Time
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func scanNodesPg(rows *sql.Rows) ([]*common.Node, error) {
	var nodes []*common.Node
	for rows.Next() {
		node := &common.Node{}
		var lastHeartbeat sql.NullTime
		if err := rows.Scan(&node.ID, &node.NodeID, &node.Name, &node.IP, &node.ProxyPort,
			&node.MaxBandwidthMbps, &node.MaxConnections, &node.NodeSecret, &node.Weight,
			&node.Status, &lastHeartbeat, &node.CreatedAt, &node.UpdatedAt); err != nil {
			return nil, err
		}
		if lastHeartbeat.Valid {
			node.LastHeartbeat = &lastHeartbeat.Time
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}
