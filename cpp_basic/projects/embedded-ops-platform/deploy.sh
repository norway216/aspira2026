#!/usr/bin/env bash
#
# Embedded Ops Platform — Unified Build & Deploy Script
# ======================================================
# One script to build, deploy, and manage the entire platform:
#   - Go backend API server
#   - Vue 3 frontend (Naive UI + ECharts)
#   - PostgreSQL + Redis infrastructure
#   - Go Agent (for embedded device management)
#
# Usage:
#   ./deploy.sh                    # Full Docker deployment (backend + frontend + DB)
#   ./deploy.sh --build            # Build all Docker images
#   ./deploy.sh --start            # Start all services
#   ./deploy.sh --stop             # Stop all services
#   ./deploy.sh --restart          # Restart all services
#   ./deploy.sh --clean            # Stop and remove all data volumes
#   ./deploy.sh --logs             # Follow logs from all services
#   ./deploy.sh --status           # Show service status
#
#   ./deploy.sh --build-local      # Build backend + agent binaries locally (no Docker)
#   ./deploy.sh --build-agent-arm  # Cross-compile agent for ARM64 (aarch64)
#   ./deploy.sh --build-agent-arm32 # Cross-compile agent for ARM32 (armv7l)
#   ./deploy.sh --run-local        # Run backend locally (requires postgres+redis)
#   ./deploy.sh --run-agent        # Run agent locally (connects to server)
#
#   ./deploy.sh --agent-package    # Create agent deployment tarball for embedded devices
#   ./deploy.sh --stop-native      # Stop services started with native mode
#
#   ./deploy.sh --help             # Show this help
#
# Environment variables:
#   DB_USER, DB_PASSWORD, DB_NAME  — PostgreSQL credentials
#   SERVER_PORT                     — Backend port (default: 8080)
#   FRONTEND_PORT                   — Frontend port (default: 80)
#   JWT_SECRET                      — JWT signing secret
#   GIN_MODE                        — "release" or "debug"
#
# Architecture:
#   Frontend (Vue 3) ─── Nginx (proxy) ─── Backend (Go/Gin) ─── PostgreSQL + Redis
#                                                                    │
#   Embedded Device ─── Agent (Go) ─── HTTPS/WSS ─── Backend ───────┘
#

set -euo pipefail

# ─── Color Output ──────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
log_success() { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $*"; }
log_step()    { echo -e "\n${CYAN}═══ $* ═══${NC}"; }

# ─── Script Location ──────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# ─── Configuration ────────────────────────────────────────────────────────────
COMPOSE_FILE="docker-compose.yml"
ENV_FILE=".env"
AGENT_DIR="agent"
BACKEND_DIR="backend"
FRONTEND_DIR="frontend"

# ─── Parse Arguments ─────────────────────────────────────────────────────────
BUILD=false
BUILD_LOCAL=false
BUILD_AGENT_ARM=false
BUILD_AGENT_ARM32=false
RUN_LOCAL=false
RUN_AGENT=false
AGENT_PACKAGE=false
STOP_NATIVE=false
START=false
STOP=false
RESTART=false
CLEAN=false
FOLLOW_LOGS=false
SHOW_STATUS=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --build)            BUILD=true; shift ;;
    --start)            START=true; shift ;;
    --stop)             STOP=true; shift ;;
    --restart)          RESTART=true; shift ;;
    --clean)            CLEAN=true; shift ;;
    --logs)             FOLLOW_LOGS=true; shift ;;
    --status)           SHOW_STATUS=true; shift ;;
    --build-local)      BUILD_LOCAL=true; shift ;;
    --build-agent-arm)  BUILD_AGENT_ARM=true; shift ;;
    --build-agent-arm32) BUILD_AGENT_ARM32=true; shift ;;
    --run-local)        RUN_LOCAL=true; shift ;;
    --run-agent)        RUN_AGENT=true; shift ;;
    --agent-package)    AGENT_PACKAGE=true; shift ;;
    --stop-native)      STOP_NATIVE=true; shift ;;
    --help|-h)
      sed -n '/^#/p' "$0" | sed 's/^# //; s/^#$//' | head -n -1
      exit 0
      ;;
    *)
      log_error "Unknown option: $1"
      echo "Usage: $0 [--build|--start|--stop|--restart|--clean|--logs|--status|--build-local|--build-agent-arm|--run-local|--run-agent|--agent-package|--help]"
      exit 1
      ;;
  esac
done

# ─── Detect Docker Compose ───────────────────────────────────────────────────
detect_compose() {
  if docker compose version &>/dev/null; then
    DOCKER_COMPOSE="docker compose"
    return 0
  elif command -v docker-compose &>/dev/null; then
    DOCKER_COMPOSE="docker-compose"
    return 0
  else
    log_error "Docker Compose not found."
    log_info "Install Docker Compose:"
    log_info "  sudo apt install docker-compose-plugin   # Ubuntu/Debian"
    log_info "  or: sudo curl -SL https://github.com/docker/compose/releases/latest/download/docker-compose-linux-x86_64 -o /usr/local/bin/docker-compose && sudo chmod +x /usr/local/bin/docker-compose"
    log_info ""
    log_info "Without Docker Compose, use local build:  $0 --build-local"
    log_info "Or build individual images with:  docker build -f deploy/Dockerfile.backend -t embedded-ops-backend ."
    return 1
  fi
}

# ─── Preflight Checks ────────────────────────────────────────────────────────
preflight_checks() {
  log_step "Preflight Checks"

  if ! command -v docker &>/dev/null; then
    log_error "Docker is not installed."
    exit 1
  fi
  log_success "Docker: $(docker --version)"

  detect_compose || exit 1
  log_success "Docker Compose ready"

  if ! docker info &>/dev/null; then
    log_error "Docker daemon is not running."
    exit 1
  fi
  log_success "Docker daemon is running"

  # Check required files
  local required_files=(
    "$COMPOSE_FILE"
    "deploy/Dockerfile.backend"
    "deploy/Dockerfile.frontend"
    "deploy/nginx.conf"
    "$BACKEND_DIR/go.mod"
    "$FRONTEND_DIR/package.json"
  )
  for file in "${required_files[@]}"; do
    if [[ ! -f "$file" ]]; then
      log_error "Required file not found: $file"
      exit 1
    fi
  done
  log_success "All required project files present"

  # Check disk space
  local available_space
  available_space=$(df --output=avail "$SCRIPT_DIR" | tail -1)
  if [[ "$available_space" -lt 1048576 ]]; then
    log_warn "Low disk space: $(( available_space / 1024 )) MB available."
  else
    log_success "Disk space: $(( available_space / 1024 / 1024 )) GB available"
  fi
}

# ─── Setup Environment ──────────────────────────────────────────────────────
setup_env() {
  log_step "Environment Setup"

  if [[ ! -f "$ENV_FILE" ]]; then
    cat > "$ENV_FILE" <<- EOF
DB_USER=embedded_ops
DB_PASSWORD=embedded_ops_secret
DB_NAME=embedded_ops
JWT_SECRET=embedded-ops-jwt-secret-$(date +%s)
GIN_MODE=release
SERVER_PORT=8080
FRONTEND_PORT=80
EOF
    log_success "Created .env file with generated secrets"
  else
    log_success "Using existing .env file"
  fi

  # Source env vars
  set -a
  source "$ENV_FILE"
  set +a
}

# ─── Build Docker Images ─────────────────────────────────────────────────────
build_images() {
  log_step "Building Docker Images"

  detect_compose || exit 1

  log_info "Building backend image..."
  $DOCKER_COMPOSE build backend 2>&1 | tail -5
  log_success "Backend image built"

  log_info "Building frontend image..."
  $DOCKER_COMPOSE build frontend 2>&1 | tail -5
  log_success "Frontend image built"

  log_success "All images built. Run '$0 --start' to launch."
}

# ─── Start Services ─────────────────────────────────────────────────────────
start_services() {
  log_step "Starting Services"

  detect_compose || exit 1

  log_info "Starting all services..."
  $DOCKER_COMPOSE up -d --remove-orphans

  log_info "Waiting for services to become healthy (max 60s)..."
  local max_attempts=20
  local attempt=0
  while [[ $attempt -lt $max_attempts ]]; do
    if $DOCKER_COMPOSE ps 2>/dev/null | grep -q 'healthy'; then
      local all_healthy=true
      # Check each expected service
      for svc in postgres redis backend; do
        if ! $DOCKER_COMPOSE ps "$svc" 2>/dev/null | grep -q 'healthy'; then
          all_healthy=false
        fi
      done
      if $all_healthy; then
        break
      fi
    fi
    attempt=$((attempt + 1))
    sleep 3
  done

  echo ""
  $DOCKER_COMPOSE ps 2>/dev/null
  echo ""

  local backend_port="${SERVER_PORT:-8080}"
  local frontend_port="${FRONTEND_PORT:-80}"

  log_success "Services started!"
  echo ""
  echo -e "  ${GREEN}Frontend:${NC}  http://localhost:${frontend_port}"
  echo -e "  ${GREEN}Backend:${NC}   http://localhost:${backend_port}/health"
  echo -e "  ${GREEN}Login:${NC}    admin / admin123"
  echo ""
  echo "  Commands:"
  echo "    $0 --logs     Follow logs"
  echo "    $0 --status   Show status"
  echo "    $0 --stop     Stop services"
  echo "    $0 --clean    Remove all data"
}

# ─── Stop Services ──────────────────────────────────────────────────────────
stop_services() {
  log_step "Stopping Services"
  detect_compose || exit 1
  $DOCKER_COMPOSE down
  log_success "All services stopped"
}

# ─── Clean All ──────────────────────────────────────────────────────────────
clean_all() {
  log_step "Clean All Data"
  log_warn "This will DELETE ALL DATA including the database!"
  echo -n "Type 'yes' to confirm: "
  read -r confirm
  if [[ "$confirm" != "yes" ]]; then
    log_info "Clean cancelled"
    exit 0
  fi

  detect_compose || exit 1
  $DOCKER_COMPOSE down -v
  docker volume rm -f embedded-ops-platform_postgres-data 2>/dev/null || true
  docker volume rm -f embedded-ops-platform_redis-data 2>/dev/null || true
  rm -f agent/identity.json 2>/dev/null || true
  log_success "Clean complete. Run '$0' to start fresh."
}

# ─── Status ─────────────────────────────────────────────────────────────────
show_status() {
  log_step "Service Status"
  detect_compose || true  # Non-fatal: just show what we can
  $DOCKER_COMPOSE ps 2>/dev/null || true
  echo ""

  # Check backend health
  local backend_port="${SERVER_PORT:-8080}"
  if curl -sf "http://localhost:${backend_port}/health" >/dev/null 2>&1; then
    log_success "Backend is healthy"
    curl -s "http://localhost:${backend_port}/health" | python3 -m json.tool 2>/dev/null || true
  else
    log_warn "Backend is not responding"
  fi
}

# ─── Follow Logs ────────────────────────────────────────────────────────────
follow_logs() {
  log_step "Following Logs (Ctrl+C to stop)"
  detect_compose || exit 1
  $DOCKER_COMPOSE logs -f --tail=100
}

# ─── Local Build (no Docker) ─────────────────────────────────────────────────
build_local() {
  log_step "Local Build (No Docker)"

  # Check Go
  if ! command -v go &>/dev/null; then
    log_error "Go is not installed."
    exit 1
  fi
  log_success "Go: $(go version)"

  # Build backend
  log_info "Building backend..."
  cd "$BACKEND_DIR"
  GOPROXY=https://goproxy.cn,direct go mod tidy 2>&1 | tail -3
  CGO_ENABLED=0 go build -ldflags="-w -s" -o "${SCRIPT_DIR}/bin/embedded-ops-server" ./cmd/server
  cd "$SCRIPT_DIR"
  log_success "Backend binary: bin/embedded-ops-server ($(ls -lh bin/embedded-ops-server | awk '{print $5}'))"

  # Build agent
  log_info "Building agent..."
  cd "$AGENT_DIR"
  GOPROXY=https://goproxy.cn,direct go mod tidy 2>&1 | tail -3
  CGO_ENABLED=0 go build -ldflags="-w -s" -o "${SCRIPT_DIR}/bin/embedded-agent" ./cmd/agent
  cd "$SCRIPT_DIR"
  log_success "Agent binary: bin/embedded-agent ($(ls -lh bin/embedded-agent | awk '{print $5}'))"

  echo ""
  log_success "Local build complete!"
  log_info "Binaries in: bin/"
  echo ""
  echo "  Run backend:  ./bin/embedded-ops-server"
  echo "  Run agent:    ./bin/embedded-agent -config deploy/agent-config.yaml"
}

# ─── Cross-compile Agent ────────────────────────────────────────────────────
build_agent_arm() {
  local arch="$1"
  local goarch
  case "$arch" in
    arm64|aarch64) goarch="arm64" ;;
    arm32|armv7l)  goarch="arm" ;;
    *) log_error "Unknown arch: $arch"; exit 1 ;;
  esac

  log_step "Cross-compiling Agent for $goarch"

  cd "$AGENT_DIR"
  GOPROXY=https://goproxy.cn,direct go mod tidy 2>&1 | tail -3

  local out="${SCRIPT_DIR}/bin/embedded-agent-${arch}"

  if [[ "$goarch" == "arm" ]]; then
    CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 \
      go build -ldflags="-w -s" -o "$out" ./cmd/agent
  else
    CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" \
      go build -ldflags="-w -s" -o "$out" ./cmd/agent
  fi

  cd "$SCRIPT_DIR"
  log_success "Agent binary: bin/embedded-agent-${arch} ($(ls -lh "$out" | awk '{print $5}'))"
}

# ─── Run Locally ─────────────────────────────────────────────────────────────
run_local() {
  log_step "Running Backend Locally"

  if [[ ! -f "bin/embedded-ops-server" ]]; then
    log_error "Binary not found. Run '$0 --build-local' first."
    exit 1
  fi

  # Source env
  if [[ -f "$ENV_FILE" ]]; then
    set -a; source "$ENV_FILE"; set +a
  fi

  export DB_HOST="${DB_HOST:-localhost}"
  export DB_PORT="${DB_PORT:-5432}"
  export DB_USER="${DB_USER:-embedded_ops}"
  export DB_PASSWORD="${DB_PASSWORD:-embedded_ops_secret}"
  export DB_NAME="${DB_NAME:-embedded_ops}"
  export DB_SSLMODE="${DB_SSLMODE:-disable}"
  export REDIS_HOST="${REDIS_HOST:-localhost}"
  export REDIS_PORT="${REDIS_PORT:-6379}"
  export REDIS_PASSWORD="${REDIS_PASSWORD:-}"
  export SERVER_HOST="${SERVER_HOST:-0.0.0.0}"
  export SERVER_PORT="${SERVER_PORT:-8080}"
  export GIN_MODE="${GIN_MODE:-debug}"
  export JWT_SECRET="${JWT_SECRET:-dev-secret}"

  log_info "Starting server on :${SERVER_PORT} (GIN_MODE=${GIN_MODE})"
  log_info "Requires PostgreSQL at ${DB_HOST}:${DB_PORT} and Redis at ${REDIS_HOST}:${REDIS_PORT}"
  exec ./bin/embedded-ops-server
}

# ─── Run Agent Locally ──────────────────────────────────────────────────────
run_agent() {
  log_step "Running Agent Locally"

  if [[ ! -f "bin/embedded-agent" ]]; then
    log_error "Agent binary not found. Run '$0 --build-local' first."
    exit 1
  fi

  local config="${1:-deploy/agent-config.yaml}"
  log_info "Using config: $config"
  exec ./bin/embedded-agent -config "$config"
}

# ─── Agent Deployment Package ───────────────────────────────────────────────
agent_package() {
  log_step "Creating Agent Deployment Package"

  local pkg_dir="bin/agent-package"
  mkdir -p "$pkg_dir"/{bin,config,systemd}

  # Copy binaries
  cp bin/embedded-agent "$pkg_dir/bin/" 2>/dev/null || log_warn "No x86_64 agent binary"
  cp bin/embedded-agent-arm64 "$pkg_dir/bin/" 2>/dev/null || log_info "No ARM64 binary (run --build-agent-arm first)"
  cp bin/embedded-agent-arm32 "$pkg_dir/bin/" 2>/dev/null || log_info "No ARM32 binary (run --build-agent-arm32 first)"

  # Copy config
  cp deploy/agent-config.yaml "$pkg_dir/config/config.yaml"

  # Create systemd service file
  cat > "$pkg_dir/systemd/embedded-agent.service" << 'SYSTEMD'
[Unit]
Description=Embedded Device Management Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/embedded-agent -config /etc/embedded-agent/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SYSTEMD

  # Create install script
  cat > "$pkg_dir/install.sh" << 'INSTALL'
#!/bin/sh
# Embedded Agent Install Script
set -e

ARCH=$(uname -m)
case "$ARCH" in
  aarch64) BIN="embedded-agent-arm64" ;;
  armv7l)  BIN="embedded-agent-arm32" ;;
  x86_64)  BIN="embedded-agent" ;;
  *) echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

echo "Installing Embedded Agent for $ARCH..."

install -m 0755 bin/$BIN /usr/local/bin/embedded-agent
install -d /etc/embedded-agent /var/lib/embedded-agent /var/log
install -m 0644 config/config.yaml /etc/embedded-agent/config.yaml
install -m 0644 systemd/embedded-agent.service /etc/systemd/system/

systemctl daemon-reload
systemctl enable embedded-agent
systemctl start embedded-agent

echo "Agent installed and started. Check status with: systemctl status embedded-agent"
INSTALL
chmod +x "$pkg_dir/install.sh"

  # Create tarball
  cd bin
  tar czf agent-package.tar.gz agent-package/
  cd "$SCRIPT_DIR"

  log_success "Agent package created: bin/agent-package.tar.gz ($(ls -lh bin/agent-package.tar.gz | awk '{print $5}'))"
  echo ""
  echo "  Deploy on target device:"
  echo "    tar xzf agent-package.tar.gz"
  echo "    cd agent-package"
  echo "    sudo ./install.sh"
}

# ─── Native Run (no Docker Compose) ──────────────────────────────────────────
start_native() {
  log_step "Native Mode (Docker for DB + Local Binaries)"

  setup_env

  # Ensure Docker network exists
  docker network create embedded-ops-net 2>/dev/null || true

  # Start PostgreSQL if not running
  if ! docker ps --format '{{.Names}}' | grep -q 'embedded-ops-postgres'; then
    log_info "Starting PostgreSQL..."
    docker rm -f embedded-ops-postgres 2>/dev/null || true
    docker run -d --name embedded-ops-postgres \
      --network embedded-ops-net \
      -e POSTGRES_USER="${DB_USER:-embedded_ops}" \
      -e POSTGRES_PASSWORD="${DB_PASSWORD:-embedded_ops_secret}" \
      -e POSTGRES_DB="${DB_NAME:-embedded_ops}" \
      -p 127.0.0.1:5432:5432 \
      postgres:16-alpine 2>&1 | head -1
  else
    log_success "PostgreSQL already running"
  fi

  # Start Redis if not running
  if ! docker ps --format '{{.Names}}' | grep -q 'embedded-ops-redis'; then
    log_info "Starting Redis..."
    docker rm -f embedded-ops-redis 2>/dev/null || true
    docker run -d --name embedded-ops-redis \
      --network embedded-ops-net \
      -p 127.0.0.1:6379:6379 \
      redis:7-alpine redis-server --appendonly yes --save 60 1000 2>&1 | head -1
  else
    log_success "Redis already running"
  fi

  # Wait for PostgreSQL
  log_info "Waiting for PostgreSQL..."
  for i in $(seq 1 20); do
    if docker exec embedded-ops-postgres pg_isready -U "${DB_USER:-embedded_ops}" -d "${DB_NAME:-embedded_ops}" 2>/dev/null | grep -q accepting; then
      log_success "PostgreSQL ready"
      break
    fi
    sleep 1
  done

  # Wait for Redis
  log_info "Waiting for Redis..."
  for i in $(seq 1 10); do
    if docker exec embedded-ops-redis redis-cli ping 2>/dev/null | grep -q PONG; then
      log_success "Redis ready"
      break
    fi
    sleep 1
  done

  # Build backend if needed
  if [[ ! -f "bin/embedded-ops-server" ]]; then
    build_local
  fi

  # Kill old backend
  pkill -f embedded-ops-server 2>/dev/null || true
  sleep 1

  # Start backend
  log_info "Starting backend on :${SERVER_PORT:-8080}..."
  export DB_HOST="${DB_HOST:-localhost}"
  export DB_PORT="${DB_PORT:-5432}"
  export REDIS_HOST="${REDIS_HOST:-localhost}"
  export REDIS_PORT="${REDIS_PORT:-6379}"
  export SERVER_HOST="${SERVER_HOST:-0.0.0.0}"
  export SERVER_PORT="${SERVER_PORT:-8080}"
  export GIN_MODE="${GIN_MODE:-release}"
  export JWT_SECRET="${JWT_SECRET:-dev-secret}"

  nohup ./bin/embedded-ops-server > /tmp/embedded-ops-backend.log 2>&1 &
  local backend_pid=$!
  echo $backend_pid > /tmp/embedded-ops-backend.pid
  sleep 2

  if curl -s http://localhost:${SERVER_PORT:-8080}/health >/dev/null 2>&1; then
    log_success "Backend running on http://localhost:${SERVER_PORT:-8080}"
  else
    log_warn "Backend may still be starting (check /tmp/embedded-ops-backend.log)"
  fi

  # Start Agent
  if [[ -f "bin/embedded-agent" ]]; then
    log_info "Starting Agent..."
    pkill -f embedded-agent 2>/dev/null || true
    sleep 1
    # Clean old identity to force fresh registration (DB was recreated)
    rm -f /tmp/embedded-agent-identity.json 2>/dev/null || true

    cat > /tmp/agent-native.yaml << EOF
server:
  url: "http://localhost:${SERVER_PORT:-8080}"
  websocket_url: "ws://localhost:${SERVER_PORT:-8080}/ws"
agent:
  device_name: ""
  register_code: "embedded-ops-first-run"
  data_dir: "/tmp/embedded-agent-data"
  log_file: "/tmp/embedded-agent.log"
security:
  tls_verify: false
  token_file: "/tmp/embedded-agent-identity.json"
collector:
  heartbeat_interval_sec: 5
  metrics_interval_sec: 10
command:
  enable_remote_command: true
  allow_reboot: false
EOF
    mkdir -p /tmp/embedded-agent-data
    nohup ./bin/embedded-agent -config /tmp/agent-native.yaml > /tmp/embedded-ops-agent.log 2>&1 &
    echo $! > /tmp/embedded-ops-agent.pid
    sleep 3
    log_success "Agent running (device auto-registers)"
  fi

  echo ""
  log_success "=== Platform Started ==="
  echo ""
  echo -e "  ${GREEN}Backend:${NC}   http://localhost:${SERVER_PORT:-8080}/health"
  echo -e "  ${GREEN}Login:${NC}    admin / admin123"
  echo ""
  echo "  Start frontend:  cd frontend && npm run dev"
  echo "  Stop all:        $0 --stop-native"
}

stop_native() {
  log_step "Stopping Native Services"

  log_info "Stopping backend..."
  if [[ -f /tmp/embedded-ops-backend.pid ]]; then
    kill $(cat /tmp/embedded-ops-backend.pid) 2>/dev/null || true
    rm -f /tmp/embedded-ops-backend.pid
  fi
  pkill -f embedded-ops-server 2>/dev/null || true

  log_info "Stopping agent..."
  if [[ -f /tmp/embedded-ops-agent.pid ]]; then
    kill $(cat /tmp/embedded-ops-agent.pid) 2>/dev/null || true
    rm -f /tmp/embedded-ops-agent.pid
  fi
  pkill -f embedded-agent 2>/dev/null || true

  log_info "Stopping Docker containers..."
  docker stop embedded-ops-postgres embedded-ops-redis 2>/dev/null || true
  docker rm embedded-ops-postgres embedded-ops-redis 2>/dev/null || true

  log_success "All services stopped"
}

# ─── Main ──────────────────────────────────────────────────────────────────
main() {
  echo ""
  echo -e "${CYAN}╔══════════════════════════════════════════════════╗${NC}"
  echo -e "${CYAN}║     Embedded Ops Platform — Deploy System       ║${NC}"
  echo -e "${CYAN}║   Distributed Embedded Device Web Ops Platform  ║${NC}"
  echo -e "${CYAN}╚══════════════════════════════════════════════════╝${NC}"
  echo ""

  # Single-action commands
  if [[ "$SHOW_STATUS" == "true" ]]; then
    show_status; exit 0
  fi
  if [[ "$STOP" == "true" ]]; then
    stop_services; exit 0
  fi
  if [[ "$STOP_NATIVE" == "true" ]]; then
    stop_native; exit 0
  fi
  if [[ "$CLEAN" == "true" ]]; then
    clean_all; exit 0
  fi
  if [[ "$FOLLOW_LOGS" == "true" ]]; then
    follow_logs; exit 0
  fi
  if [[ "$BUILD_LOCAL" == "true" ]]; then
    build_local; exit 0
  fi
  if [[ "$BUILD_AGENT_ARM" == "true" ]]; then
    build_agent_arm "arm64"; exit 0
  fi
  if [[ "$BUILD_AGENT_ARM32" == "true" ]]; then
    build_agent_arm "arm32"; exit 0
  fi
  if [[ "$RUN_LOCAL" == "true" ]]; then
    run_local; exit 0
  fi
  if [[ "$RUN_AGENT" == "true" ]]; then
    run_agent; exit 0
  fi
  if [[ "$AGENT_PACKAGE" == "true" ]]; then
    agent_package; exit 0
  fi
  if [[ "$BUILD" == "true" ]]; then
    preflight_checks; setup_env; build_images; exit 0
  fi
  if [[ "$START" == "true" ]]; then
    setup_env; start_services; exit 0
  fi
  if [[ "$RESTART" == "true" ]]; then
    stop_services; setup_env; start_services; exit 0
  fi

  # Default: smart deploy — try Docker Compose first, fall back to native
  if docker compose version &>/dev/null || command -v docker-compose &>/dev/null; then
    preflight_checks
    setup_env
    build_images
    start_services
  else
    log_warn "Docker Compose not available — using native mode (Docker for DB + local binaries)"
    echo ""
    start_native
  fi

  echo ""
  log_success "Deployment complete!"
  echo ""
}

main
