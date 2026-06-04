# TrafficLab Proxy Control Platform

多节点代理流量调度实验平台 - Multi-node Traffic Scheduling and Proxy Control Platform

## Overview

TrafficLab is an experimental platform for learning and verifying network systems design. It implements a multi-node proxy architecture with traffic forwarding, rate limiting, load scheduling, and monitoring.

## Architecture

```
Client → Proxy Node (TCP CONNECT) → Target Server
                ↕ (heartbeat/traffic/policy)
           Control Center (REST API + Web Dashboard)
                ↕
        Prometheus + Grafana (monitoring)
```

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose (optional)

### Local Development

```bash
# Build both binaries
make build

# Start control center (with SQLite)
make run-cc-dev

# In another terminal, start proxy node 1
make run-pn-dev

# In another terminal, start proxy node 2
make run-pn-dev2
```

### Docker Deployment

```bash
# Start all services
make docker-up

# View logs
make docker-logs

# Stop all services
make docker-down
```

### Access Points

| Service | URL |
|---------|-----|
| Control Center API | http://localhost:8080/api/v1/ |
| Web Dashboard | http://localhost:8080/admin/dashboard |
| Proxy Node 1 | localhost:10801 |
| Proxy Node 2 | localhost:10802 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 (admin/admin) |
| Target Nginx | http://localhost:8088 |

### Web Dashboard Login

Default credentials: `admin` / `admin123`

## Usage

### 1. Create a proxy user

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -u "admin:admin123" \
  -d '{"username":"user1","password":"pass1","max_rate_mbps":10,"max_connections":5}'
```

### 2. Use the proxy

```bash
# The token is returned when creating the user
curl -x "http://user1:<TOKEN>@localhost:10801" http://target-server/
```

### 3. Check traffic statistics

```bash
curl -u "admin:admin123" http://localhost:8080/api/v1/traffic/users
```

## Scheduling Strategies

| Strategy | Description |
|----------|-------------|
| `round_robin` | Simple round-robin across nodes |
| `least_connections` | Pick node with fewest connections |
| `least_bandwidth` | Pick node with lowest bandwidth utilization |
| `weighted_round_robin` | Weighted distribution based on node capacity |
| `composite` | Composite score: CPU×0.25 + Mem×0.15 + BW×0.40 + Conn×0.20 |

Change strategy via API or Web Dashboard:
```bash
curl -X POST "http://localhost:8080/api/v1/scheduler/strategy?strategy=least_connections" \
  -u "admin:admin123"
```

## Environment Variables

### Control Center

| Variable | Default | Description |
|----------|---------|-------------|
| `LISTEN_ADDR` | `:8080` | HTTP listen address |
| `DATABASE_DRIVER` | `sqlite` | Database driver (sqlite/postgres) |
| `DATABASE_DSN` | `traffic_lab.db` | Database connection string |
| `ADMIN_USER` | `admin` | Admin username |
| `ADMIN_PASS` | `admin123` | Admin password |
| `SCHEDULER_STRATEGY` | `composite` | Default scheduler strategy |
| `NODE_STALE_TIMEOUT` | `30s` | Time before marking node offline |

### Proxy Node

| Variable | Default | Description |
|----------|---------|-------------|
| `NODE_ID` | `node-01` | Unique node identifier |
| `LISTEN_ADDR` | `:10801` | Proxy listen address |
| `CONTROL_CENTER_URL` | `http://localhost:8080` | Control Center URL |
| `NODE_SECRET` | `...` | HMAC secret for node auth |
| `MAX_CONNECTIONS` | `1000` | Max concurrent connections |
| `MAX_BANDWIDTH_MBPS` | `100` | Max node bandwidth |

## Testing

```bash
# Run all tests with race detector
make test

# Integration test script
./scripts/test_traffic.sh http://localhost:8080 http://localhost:10801

# Test proxy throughput with curl
curl -x "http://user1:TOKEN@localhost:10801" -o /dev/null -w "Speed: %{speed_download} bytes/s\n" http://localhost:8088/test/1m
```

## Project Structure

```
traffic_lab_proxy/
├── cmd/
│   ├── control-center/   # Control Center entry point
│   └── proxy-node/       # Proxy Node entry point
├── internal/
│   ├── common/           # Shared types, config, errors
│   ├── controlcenter/    # Control Center implementation
│   │   ├── api/          # REST API handlers + router
│   │   ├── db/           # Database layer (SQLite/PostgreSQL)
│   │   ├── scheduler/    # Load balancing strategies
│   │   └── web/          # Web dashboard handler
│   └── proxynode/        # Proxy Node implementation
│       ├── proxy/        # TCP proxy + connection mgr
│       ├── limiter/      # Token Bucket rate limiter
│       ├── auth/         # User authentication
│       ├── counter/      # Traffic statistics
│       ├── reporter/     # Heartbeat + traffic reporting
│       └── policy/       # Policy fetching + caching
├── web/                  # Frontend (templates + static)
├── database/             # SQL schema
├── deploy/               # Docker + monitoring configs
└── scripts/              # Test and utility scripts
```

## License

This project is for educational and experimental purposes only.
