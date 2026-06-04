-- Traffic Lab Proxy Control Platform - Database Schema
-- Compatible with SQLite (and PostgreSQL with minor adjustments)

-- Users table
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

-- Nodes table
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

-- Node status logs table
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

-- Traffic logs table
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

-- Policies table
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

-- Scheduler events table
CREATE TABLE IF NOT EXISTS scheduler_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id VARCHAR(64),
    selected_node_id VARCHAR(64),
    strategy VARCHAR(64) NOT NULL,
    reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_scheduler_events_created_at ON scheduler_events(created_at);
CREATE INDEX IF NOT EXISTS idx_scheduler_events_strategy ON scheduler_events(strategy);
