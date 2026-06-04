package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/secure-gateway/internal/model"
	"github.com/secure-gateway/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

// DB wraps the sql.DB connection pool.
type DB struct {
	*sql.DB
}

// New creates a new database connection.
func New(driver, dsn string) (*DB, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("database open: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database ping: %w", err)
	}

	logger.Info("Database connected", "driver", driver)
	return &DB{db}, nil
}

// AutoMigrate creates tables if they don't exist.
func (db *DB) AutoMigrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		username VARCHAR(64) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(32) NOT NULL DEFAULT 'user',
		status VARCHAR(32) NOT NULL DEFAULT 'active',
		traffic_quota_bytes BIGINT NOT NULL DEFAULT 0,
		traffic_used_bytes BIGINT NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS nodes (
		id BIGSERIAL PRIMARY KEY,
		node_id VARCHAR(128) NOT NULL UNIQUE,
		name VARCHAR(128) NOT NULL,
		region VARCHAR(64) NOT NULL,
		public_addr VARCHAR(255) NOT NULL,
		status VARCHAR(32) NOT NULL DEFAULT 'offline',
		weight INT NOT NULL DEFAULT 100,
		max_connections INT NOT NULL DEFAULT 10000,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS node_metrics (
		id BIGSERIAL PRIMARY KEY,
		node_id VARCHAR(128) NOT NULL,
		cpu_usage DOUBLE PRECISION NOT NULL,
		mem_usage DOUBLE PRECISION NOT NULL,
		active_connections INT NOT NULL,
		rx_bytes_per_sec BIGINT NOT NULL,
		tx_bytes_per_sec BIGINT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS traffic_records (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT NOT NULL,
		node_id VARCHAR(128) NOT NULL,
		rx_bytes BIGINT NOT NULL DEFAULT 0,
		tx_bytes BIGINT NOT NULL DEFAULT 0,
		duration_seconds BIGINT NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT,
		action VARCHAR(128) NOT NULL,
		resource VARCHAR(255),
		ip_addr VARCHAR(64),
		user_agent TEXT,
		detail TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_node_metrics_node_id ON node_metrics(node_id);
	CREATE INDEX IF NOT EXISTS idx_traffic_records_user_id ON traffic_records(user_id);
	CREATE INDEX IF NOT EXISTS idx_traffic_records_created_at ON traffic_records(created_at);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	logger.Info("Database migration completed")
	return nil
}

// SeedDefaultAdmin inserts a default admin user if no users exist.
func (db *DB) SeedDefaultAdmin() error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		"INSERT INTO users (username, password_hash, role, status) VALUES ($1, $2, $3, $4)",
		"admin", string(hash), model.RoleAdmin, model.StatusActive,
	)
	if err != nil {
		return err
	}
	logger.Info("Default admin user created (admin / admin123)")
	return nil
}

// --- User operations ---

func (db *DB) GetUserByUsername(username string) (*model.User, error) {
	u := &model.User{}
	err := db.QueryRow(
		"SELECT id, username, password_hash, role, status, traffic_quota_bytes, traffic_used_bytes, created_at, updated_at FROM users WHERE username = $1",
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.Status, &u.TrafficQuotaBytes, &u.TrafficUsedBytes, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) GetUserByID(id int64) (*model.User, error) {
	u := &model.User{}
	err := db.QueryRow(
		"SELECT id, username, password_hash, role, status, traffic_quota_bytes, traffic_used_bytes, created_at, updated_at FROM users WHERE id = $1",
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.Status, &u.TrafficQuotaBytes, &u.TrafficUsedBytes, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) ListUsers() ([]*model.User, error) {
	rows, err := db.Query(
		"SELECT id, username, role, status, traffic_quota_bytes, traffic_used_bytes, created_at, updated_at FROM users ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Status, &u.TrafficQuotaBytes, &u.TrafficUsedBytes, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (db *DB) CreateUser(req *model.CreateUserRequest) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	role := req.Role
	if role == "" {
		role = model.RoleUser
	}

	u := &model.User{}
	err = db.QueryRow(
		`INSERT INTO users (username, password_hash, role, traffic_quota_bytes)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, username, role, status, traffic_quota_bytes, traffic_used_bytes, created_at, updated_at`,
		req.Username, string(hash), role, req.Quota,
	).Scan(&u.ID, &u.Username, &u.Role, &u.Status, &u.TrafficQuotaBytes, &u.TrafficUsedBytes, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) UpdateUser(id int64, req *model.UpdateUserRequest) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if req.Password != nil && *req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		_, err = tx.Exec("UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", string(hash), id)
		if err != nil {
			return err
		}
	}
	if req.Role != nil {
		_, err = tx.Exec("UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2", *req.Role, id)
		if err != nil {
			return err
		}
	}
	if req.Status != nil {
		_, err = tx.Exec("UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2", *req.Status, id)
		if err != nil {
			return err
		}
	}
	if req.Quota != nil {
		_, err = tx.Exec("UPDATE users SET traffic_quota_bytes = $1, updated_at = NOW() WHERE id = $2", *req.Quota, id)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) DeleteUser(id int64) error {
	_, err := db.Exec("DELETE FROM users WHERE id = $1", id)
	return err
}

// --- Node operations ---

func (db *DB) GetNodeByNodeID(nodeID string) (*model.Node, error) {
	n := &model.Node{}
	err := db.QueryRow(
		"SELECT id, node_id, name, region, public_addr, status, weight, max_connections, created_at, updated_at FROM nodes WHERE node_id = $1",
		nodeID,
	).Scan(&n.ID, &n.NodeID, &n.Name, &n.Region, &n.PublicAddr, &n.Status, &n.Weight, &n.MaxConnections, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (db *DB) ListNodes() ([]*model.Node, error) {
	rows, err := db.Query(
		"SELECT id, node_id, name, region, public_addr, status, weight, max_connections, created_at, updated_at FROM nodes ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*model.Node
	for rows.Next() {
		n := &model.Node{}
		err := rows.Scan(&n.ID, &n.NodeID, &n.Name, &n.Region, &n.PublicAddr, &n.Status, &n.Weight, &n.MaxConnections, &n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

func (db *DB) RegisterNode(req *model.NodeRegisterRequest) (*model.Node, error) {
	existing, err := db.GetNodeByNodeID(req.NodeID)
	if err == nil {
		// Node already exists - update it
		_, err = db.Exec(
			"UPDATE nodes SET name=$1, region=$2, public_addr=$3, status=$4, max_connections=$5, updated_at=NOW() WHERE node_id=$6",
			req.Name, req.Region, req.PublicAddr, model.NodeStatusOnline, req.MaxConnections, req.NodeID,
		)
		if err != nil {
			return nil, err
		}
		existing.Status = model.NodeStatusOnline
		existing.PublicAddr = req.PublicAddr
		return existing, nil
	}

	// New node
	maxConns := req.MaxConnections
	if maxConns <= 0 {
		maxConns = 10000
	}

	n := &model.Node{}
	err = db.QueryRow(
		`INSERT INTO nodes (node_id, name, region, public_addr, status, max_connections)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, node_id, name, region, public_addr, status, weight, max_connections, created_at, updated_at`,
		req.NodeID, req.Name, req.Region, req.PublicAddr, model.NodeStatusOnline, maxConns,
	).Scan(&n.ID, &n.NodeID, &n.Name, &n.Region, &n.PublicAddr, &n.Status, &n.Weight, &n.MaxConnections, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (db *DB) UpdateNodeStatus(nodeID, status string) error {
	_, err := db.Exec("UPDATE nodes SET status = $1, updated_at = NOW() WHERE node_id = $2", status, nodeID)
	return err
}

// --- Node Metrics operations ---

func (db *DB) InsertNodeMetrics(m *model.NodeMetrics) error {
	_, err := db.Exec(
		"INSERT INTO node_metrics (node_id, cpu_usage, mem_usage, active_connections, rx_bytes_per_sec, tx_bytes_per_sec) VALUES ($1,$2,$3,$4,$5,$6)",
		m.NodeID, m.CPUUsage, m.MemUsage, m.ActiveConnections, m.RxBytesPerSec, m.TxBytesPerSec,
	)
	return err
}

func (db *DB) GetLatestNodeMetrics(nodeID string) (*model.NodeMetrics, error) {
	m := &model.NodeMetrics{}
	err := db.QueryRow(
		"SELECT id, node_id, cpu_usage, mem_usage, active_connections, rx_bytes_per_sec, tx_bytes_per_sec, created_at FROM node_metrics WHERE node_id = $1 ORDER BY created_at DESC LIMIT 1",
		nodeID,
	).Scan(&m.ID, &m.NodeID, &m.CPUUsage, &m.MemUsage, &m.ActiveConnections, &m.RxBytesPerSec, &m.TxBytesPerSec, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// --- Traffic operations ---

func (db *DB) InsertTrafficRecord(r *model.TrafficRecord) error {
	_, err := db.Exec(
		"INSERT INTO traffic_records (user_id, node_id, rx_bytes, tx_bytes, duration_seconds) VALUES ($1,$2,$3,$4,$5)",
		r.UserID, r.NodeID, r.RxBytes, r.TxBytes, r.DurationSeconds,
	)
	return err
}

func (db *DB) GetTodayTrafficBytes() (int64, error) {
	var total sql.NullInt64
	err := db.QueryRow(
		"SELECT COALESCE(SUM(rx_bytes + tx_bytes), 0) FROM traffic_records WHERE created_at >= CURRENT_DATE",
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total.Int64, nil
}

// --- Audit log operations ---

func (db *DB) InsertAuditLog(log *model.AuditLog) error {
	_, err := db.Exec(
		"INSERT INTO audit_logs (user_id, action, resource, ip_addr, user_agent, detail) VALUES ($1,$2,$3,$4,$5,$6)",
		log.UserID, log.Action, log.Resource, log.IPAddr, log.UserAgent, log.Detail,
	)
	return err
}

func (db *DB) ListAuditLogs(limit int) ([]*model.AuditLog, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Query(
		"SELECT id, user_id, action, resource, ip_addr, user_agent, detail, created_at FROM audit_logs ORDER BY created_at DESC LIMIT $1",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*model.AuditLog
	for rows.Next() {
		l := &model.AuditLog{}
		err := rows.Scan(&l.ID, &l.UserID, &l.Action, &l.Resource, &l.IPAddr, &l.UserAgent, &l.Detail, &l.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// --- Dashboard ---

func (db *DB) GetDashboardStats() (*model.DashboardStats, error) {
	stats := &model.DashboardStats{}

	err := db.QueryRow("SELECT COUNT(*) FROM nodes WHERE status = 'online'").Scan(&stats.OnlineNodes)
	if err != nil {
		return nil, err
	}

	var totalConns sql.NullInt64
	err = db.QueryRow("SELECT COALESCE(SUM(active_connections), 0) FROM node_metrics n1 WHERE created_at = (SELECT MAX(created_at) FROM node_metrics n2 WHERE n2.node_id = n1.node_id)").Scan(&totalConns)
	if err != nil {
		// non-fatal
		_ = err
	}
	stats.TotalConnections = int(totalConns.Int64)

	todayBytes, err := db.GetTodayTrafficBytes()
	if err == nil {
		stats.TodayTrafficBytes = todayBytes
	}

	var cpu, mem sql.NullFloat64
	err = db.QueryRow("SELECT COALESCE(AVG(cpu_usage), 0), COALESCE(AVG(mem_usage), 0) FROM node_metrics n1 WHERE created_at = (SELECT MAX(created_at) FROM node_metrics n2 WHERE n2.node_id = n1.node_id)").Scan(&cpu, &mem)
	if err == nil {
		stats.AvgCPUUsage = cpu.Float64
		stats.AvgMemUsage = mem.Float64
	}

	return stats, nil
}