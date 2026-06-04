-- Seed data for Secure Gateway
-- Default admin password: admin123 (bcrypt hash)
-- The hash below is for "admin123"

INSERT INTO users (username, password_hash, role, status)
VALUES ('admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'admin', 'active')
ON CONFLICT (username) DO NOTHING;

INSERT INTO users (username, password_hash, role, status, traffic_quota_bytes)
VALUES ('demo', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'user', 'active', 1073741824)
ON CONFLICT (username) DO NOTHING;

-- Sample node for testing
INSERT INTO nodes (node_id, name, region, public_addr, status, max_connections)
VALUES ('edge-node-01', 'Edge Node 01', 'local-lab', 'traffic-node:9443', 'online', 10000)
ON CONFLICT (node_id) DO NOTHING;