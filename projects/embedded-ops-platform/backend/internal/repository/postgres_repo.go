package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/embedded-ops-platform/backend/internal/model"
	"github.com/embedded-ops-platform/backend/pkg/config"
	"github.com/embedded-ops-platform/backend/pkg/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepo implements all database operations.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo creates a new PostgreSQL repository.
func NewPostgresRepo(ctx context.Context, cfg config.DatabaseConfig) (*PostgresRepo, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connectivity
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Connected to PostgreSQL", logger.Int("max_conns", cfg.MaxOpenConns))
	return &PostgresRepo{pool: pool}, nil
}

// Close closes the database connection pool.
func (r *PostgresRepo) Close() {
	r.pool.Close()
}

// InitSchema creates the database tables if they don't exist.
func (r *PostgresRepo) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		username VARCHAR(128) UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		email VARCHAR(256),
		role VARCHAR(32) DEFAULT 'viewer',
		enabled BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS devices (
		id BIGSERIAL PRIMARY KEY,
		device_id VARCHAR(128) UNIQUE NOT NULL,
		device_name VARCHAR(128),
		hostname VARCHAR(128),
		ip_address VARCHAR(64),
		mac_address VARCHAR(128),
		board_type VARCHAR(64),
		arch VARCHAR(64),
		os_name VARCHAR(128),
		os_version VARCHAR(128),
		kernel_version VARCHAR(128),
		bsp_version VARCHAR(128),
		agent_version VARCHAR(64),
		status VARCHAR(32) DEFAULT 'offline',
		last_seen TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS agent_credentials (
		id BIGSERIAL PRIMARY KEY,
		device_id VARCHAR(128) UNIQUE NOT NULL,
		token_hash TEXT NOT NULL,
		cert_fingerprint VARCHAR(256),
		enabled BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS device_metrics (
		id BIGSERIAL PRIMARY KEY,
		device_id VARCHAR(128) NOT NULL,
		cpu_usage DOUBLE PRECISION,
		memory_usage DOUBLE PRECISION,
		disk_usage DOUBLE PRECISION,
		load_avg_1m DOUBLE PRECISION,
		temperature DOUBLE PRECISION,
		gpu_load DOUBLE PRECISION,
		network_rx_bytes BIGINT,
		network_tx_bytes BIGINT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS command_tasks (
		id BIGSERIAL PRIMARY KEY,
		task_id VARCHAR(128) UNIQUE NOT NULL,
		device_id VARCHAR(128) NOT NULL,
		action VARCHAR(128) NOT NULL,
		params JSONB,
		status VARCHAR(32) DEFAULT 'pending',
		stdout TEXT,
		stderr TEXT,
		exit_code INT,
		created_by VARCHAR(128),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		started_at TIMESTAMP,
		finished_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS device_logs (
		id BIGSERIAL PRIMARY KEY,
		device_id VARCHAR(128) NOT NULL,
		log_type VARCHAR(64),
		level VARCHAR(32),
		content TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS alerts (
		id BIGSERIAL PRIMARY KEY,
		device_id VARCHAR(128),
		alert_type VARCHAR(128),
		severity VARCHAR(32),
		message TEXT,
		status VARCHAR(32) DEFAULT 'open',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		resolved_at TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_devices_device_id ON devices(device_id);
	CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);
	CREATE INDEX IF NOT EXISTS idx_device_metrics_device_id ON device_metrics(device_id);
	CREATE INDEX IF NOT EXISTS idx_device_metrics_created_at ON device_metrics(created_at);
	CREATE INDEX IF NOT EXISTS idx_command_tasks_device_id ON command_tasks(device_id);
	CREATE INDEX IF NOT EXISTS idx_command_tasks_status ON command_tasks(status);
	CREATE INDEX IF NOT EXISTS idx_device_logs_device_id ON device_logs(device_id);
	CREATE INDEX IF NOT EXISTS idx_alerts_device_id ON alerts(device_id);
	CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
	`

	_, err := r.pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	logger.Info("Database schema initialized")
	return nil
}

// SeedDefaultData creates default admin user and a test register code if they don't exist.
func (r *PostgresRepo) SeedDefaultData(ctx context.Context) error {
	// Check if admin user already exists
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE username = 'admin'`).Scan(&count)
	if err != nil || count > 0 {
		return nil // already seeded or error (will try again next time)
	}

	// Insert default admin user with bcrypt hash for 'admin123' (cost=10)
	_, err = r.pool.Exec(ctx, `
		INSERT INTO users (username, password_hash, email, role, enabled)
		VALUES ('admin', '$2a$10$.wgux0E7n0ZqN2xnrjKGVOJugltZuwcVn0dXqxziRD85.D37sJH.6', 'admin@embedded-ops.local', 'admin', true)
		ON CONFLICT (username) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to seed default user: %w", err)
	}
	return nil
}

// --- User operations ---

func (r *PostgresRepo) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	user := &model.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, email, role, enabled, created_at, updated_at
		 FROM users WHERE username = $1`, username).
		Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Role, &user.Enabled, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

func (r *PostgresRepo) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	user := &model.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, email, role, enabled, created_at, updated_at
		 FROM users WHERE id = $1`, userID).
		Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Role, &user.Enabled, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

// --- Device operations ---

func (r *PostgresRepo) CreateDevice(ctx context.Context, d *model.Device) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO devices (device_id, device_name, hostname, ip_address, mac_address, board_type, arch, os_name, os_version, kernel_version, bsp_version, agent_version, status, last_seen, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NOW(),NOW())
		 ON CONFLICT (device_id) DO UPDATE SET
		   hostname=EXCLUDED.hostname, ip_address=EXCLUDED.ip_address, mac_address=EXCLUDED.mac_address,
		   board_type=EXCLUDED.board_type, arch=EXCLUDED.arch, os_name=EXCLUDED.os_name,
		   os_version=EXCLUDED.os_version, kernel_version=EXCLUDED.kernel_version,
		   agent_version=EXCLUDED.agent_version, updated_at=NOW()`,
		d.DeviceID, d.DeviceName, d.Hostname, d.IPAddress, d.MACAddress, d.BoardType,
		d.Arch, d.OSName, d.OSVersion, d.KernelVersion, d.BSPVersion, d.AgentVersion,
		d.Status, d.LastSeen)
	return err
}

func (r *PostgresRepo) GetDeviceByID(ctx context.Context, deviceID string) (*model.Device, error) {
	d := &model.Device{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, device_id, device_name, hostname, ip_address, mac_address, board_type, arch,
		        os_name, os_version, kernel_version, bsp_version, agent_version, status, last_seen, created_at, updated_at
		 FROM devices WHERE device_id = $1`, deviceID).
		Scan(&d.ID, &d.DeviceID, &d.DeviceName, &d.Hostname, &d.IPAddress, &d.MACAddress,
			&d.BoardType, &d.Arch, &d.OSName, &d.OSVersion, &d.KernelVersion, &d.BSPVersion,
			&d.AgentVersion, &d.Status, &d.LastSeen, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}
	return d, nil
}

func (r *PostgresRepo) ListDevices(ctx context.Context, offset, limit int) ([]*model.Device, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM devices`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, device_name, hostname, ip_address, mac_address, board_type, arch,
		        os_name, os_version, kernel_version, bsp_version, agent_version, status, last_seen, created_at, updated_at
		 FROM devices ORDER BY last_seen DESC NULLS LAST LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var devices []*model.Device
	for rows.Next() {
		d := &model.Device{}
		if err := rows.Scan(&d.ID, &d.DeviceID, &d.DeviceName, &d.Hostname, &d.IPAddress, &d.MACAddress,
			&d.BoardType, &d.Arch, &d.OSName, &d.OSVersion, &d.KernelVersion, &d.BSPVersion,
			&d.AgentVersion, &d.Status, &d.LastSeen, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, err
		}
		devices = append(devices, d)
	}
	return devices, total, nil
}

func (r *PostgresRepo) UpdateDeviceStatus(ctx context.Context, deviceID, status string, lastSeen time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE devices SET status = $2, last_seen = $3, updated_at = NOW() WHERE device_id = $1`,
		deviceID, status, lastSeen)
	return err
}

func (r *PostgresRepo) BatchUpdateLastSeen(ctx context.Context, updates map[string]time.Time) error {
	if len(updates) == 0 {
		return nil
	}
	// Batch update one by one for simplicity; in production use COPY or UNNEST
	for deviceID, t := range updates {
		if _, err := r.pool.Exec(ctx,
			`UPDATE devices SET last_seen = $2, status = 'online', updated_at = NOW() WHERE device_id = $1`,
			deviceID, t); err != nil {
			logger.Warn("Failed to batch update device",
				logger.String("device_id", deviceID),
				logger.ErrField(err))
		}
	}
	return nil
}

// --- Agent Credential operations ---

func (r *PostgresRepo) CreateAgentCredential(ctx context.Context, cred *model.AgentCredential) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO agent_credentials (device_id, token_hash, cert_fingerprint, enabled, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,NOW(),NOW())
		 ON CONFLICT (device_id) DO UPDATE SET token_hash=EXCLUDED.token_hash, updated_at=NOW()`,
		cred.DeviceID, cred.TokenHash, cred.CertFingerprint, cred.Enabled)
	return err
}

func (r *PostgresRepo) GetAgentCredential(ctx context.Context, deviceID string) (*model.AgentCredential, error) {
	cred := &model.AgentCredential{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, device_id, token_hash, cert_fingerprint, enabled, created_at, updated_at
		 FROM agent_credentials WHERE device_id = $1 AND enabled = true`, deviceID).
		Scan(&cred.ID, &cred.DeviceID, &cred.TokenHash, &cred.CertFingerprint, &cred.Enabled, &cred.CreatedAt, &cred.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("credential not found: %w", err)
	}
	return cred, nil
}

// --- Metric operations ---

func (r *PostgresRepo) InsertMetric(ctx context.Context, m *model.DeviceMetric) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO device_metrics (device_id, cpu_usage, memory_usage, disk_usage, load_avg_1m, temperature, gpu_load, network_rx_bytes, network_tx_bytes, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())`,
		m.DeviceID, m.CPUUsage, m.MemoryUsage, m.DiskUsage, m.LoadAvg1m, m.Temperature, m.GPULoad, m.NetworkRX, m.NetworkTX)
	return err
}

func (r *PostgresRepo) BatchInsertMetrics(ctx context.Context, metrics []*model.DeviceMetric) error {
	if len(metrics) == 0 {
		return nil
	}
	for _, m := range metrics {
		if err := r.InsertMetric(ctx, m); err != nil {
			logger.Warn("Failed to insert metric",
				logger.String("device_id", m.DeviceID),
				logger.ErrField(err))
		}
	}
	return nil
}

func (r *PostgresRepo) GetLatestMetrics(ctx context.Context, deviceID string) (*model.DeviceMetric, error) {
	m := &model.DeviceMetric{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, device_id, cpu_usage, memory_usage, disk_usage, load_avg_1m, temperature, gpu_load, network_rx_bytes, network_tx_bytes, created_at
		 FROM device_metrics WHERE device_id = $1 ORDER BY created_at DESC LIMIT 1`, deviceID).
		Scan(&m.ID, &m.DeviceID, &m.CPUUsage, &m.MemoryUsage, &m.DiskUsage, &m.LoadAvg1m,
			&m.Temperature, &m.GPULoad, &m.NetworkRX, &m.NetworkTX, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("metrics not found: %w", err)
	}
	return m, nil
}

func (r *PostgresRepo) GetMetricsHistory(ctx context.Context, deviceID string, since time.Time, limit int) ([]*model.DeviceMetric, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, cpu_usage, memory_usage, disk_usage, load_avg_1m, temperature, gpu_load, network_rx_bytes, network_tx_bytes, created_at
		 FROM device_metrics WHERE device_id = $1 AND created_at >= $2 ORDER BY created_at DESC LIMIT $3`,
		deviceID, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*model.DeviceMetric
	for rows.Next() {
		m := &model.DeviceMetric{}
		if err := rows.Scan(&m.ID, &m.DeviceID, &m.CPUUsage, &m.MemoryUsage, &m.DiskUsage,
			&m.LoadAvg1m, &m.Temperature, &m.GPULoad, &m.NetworkRX, &m.NetworkTX, &m.CreatedAt); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}

// --- Command Task operations ---

func (r *PostgresRepo) CreateCommandTask(ctx context.Context, task *model.CommandTask) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO command_tasks (task_id, device_id, action, params, status, created_by, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,NOW())`,
		task.TaskID, task.DeviceID, task.Action, task.Params, task.Status, task.CreatedBy)
	return err
}

func (r *PostgresRepo) GetPendingTask(ctx context.Context, deviceID string) (*model.CommandTask, error) {
	task := &model.CommandTask{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, task_id, device_id, action, params, status, created_at
		 FROM command_tasks WHERE device_id = $1 AND status = 'pending' ORDER BY created_at ASC LIMIT 1
		 FOR UPDATE SKIP LOCKED`, deviceID).
		Scan(&task.ID, &task.TaskID, &task.DeviceID, &task.Action, &task.Params, &task.Status, &task.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("no pending task: %w", err)
	}
	return task, nil
}

func (r *PostgresRepo) UpdateCommandTask(ctx context.Context, taskID, status, stdout, stderr string, exitCode int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE command_tasks SET status = $2, stdout = $3, stderr = $4, exit_code = $5, finished_at = NOW()
		 WHERE task_id = $1 AND status = 'running'`,
		taskID, status, stdout, stderr, exitCode)
	return err
}

func (r *PostgresRepo) SetTaskRunning(ctx context.Context, taskID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE command_tasks SET status = 'running', started_at = NOW() WHERE task_id = $1 AND status = 'pending'`,
		taskID)
	return err
}

func (r *PostgresRepo) ListDeviceCommands(ctx context.Context, deviceID string, limit int) ([]*model.CommandTask, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, task_id, device_id, action, params, status, stdout, stderr, exit_code, created_by, created_at, started_at, finished_at
		 FROM command_tasks WHERE device_id = $1 ORDER BY created_at DESC LIMIT $2`, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*model.CommandTask
	for rows.Next() {
		t := &model.CommandTask{}
		if err := rows.Scan(&t.ID, &t.TaskID, &t.DeviceID, &t.Action, &t.Params, &t.Status,
			&t.Stdout, &t.Stderr, &t.ExitCode, &t.CreatedBy, &t.CreatedAt, &t.StartedAt, &t.FinishedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// --- Log operations ---

func (r *PostgresRepo) InsertLog(ctx context.Context, log *model.DeviceLog) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO device_logs (device_id, log_type, level, content, created_at)
		 VALUES ($1,$2,$3,$4,NOW())`,
		log.DeviceID, log.LogType, log.Level, log.Content)
	return err
}

func (r *PostgresRepo) GetDeviceLogs(ctx context.Context, deviceID, logType string, limit int) ([]*model.DeviceLog, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, log_type, level, content, created_at
		 FROM device_logs WHERE device_id = $1 AND ($2 = '' OR log_type = $2)
		 ORDER BY created_at DESC LIMIT $3`, deviceID, logType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*model.DeviceLog
	for rows.Next() {
		l := &model.DeviceLog{}
		if err := rows.Scan(&l.ID, &l.DeviceID, &l.LogType, &l.Level, &l.Content, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// --- Alert operations ---

func (r *PostgresRepo) CreateAlert(ctx context.Context, alert *model.Alert) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO alerts (device_id, alert_type, severity, message, status, created_at)
		 VALUES ($1,$2,$3,$4,$5,NOW())`,
		alert.DeviceID, alert.AlertType, alert.Severity, alert.Message, alert.Status)
	return err
}

func (r *PostgresRepo) ListAlerts(ctx context.Context, offset, limit int) ([]*model.Alert, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM alerts WHERE status = 'open'`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, alert_type, severity, message, status, created_at, resolved_at
		 FROM alerts WHERE status = 'open' ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var alerts []*model.Alert
	for rows.Next() {
		a := &model.Alert{}
		if err := rows.Scan(&a.ID, &a.DeviceID, &a.AlertType, &a.Severity, &a.Message, &a.Status, &a.CreatedAt, &a.ResolvedAt); err != nil {
			return nil, 0, err
		}
		alerts = append(alerts, a)
	}
	return alerts, total, nil
}

func (r *PostgresRepo) ResolveAlert(ctx context.Context, alertID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE alerts SET status = 'resolved', resolved_at = NOW() WHERE id = $1`, alertID)
	return err
}

// CheckDeviceExists checks if a device_id exists (for registration dedup).
func (r *PostgresRepo) CheckDeviceExists(ctx context.Context, deviceID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM devices WHERE device_id = $1)`, deviceID).Scan(&exists)
	return exists, err
}

// GetDeviceIDByMAC returns a device_id for a given MAC address, if one exists.
func (r *PostgresRepo) GetDeviceIDByMAC(ctx context.Context, mac string) (string, error) {
	var deviceID string
	err := r.pool.QueryRow(ctx,
		`SELECT device_id FROM devices WHERE mac_address LIKE $1 LIMIT 1`, "%"+mac+"%").Scan(&deviceID)
	if err != nil {
		return "", fmt.Errorf("device not found for MAC: %w", err)
	}
	return deviceID, nil
}