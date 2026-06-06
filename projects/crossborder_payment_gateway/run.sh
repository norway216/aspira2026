#!/bin/bash
set -e

# ============================================================
#  Aspira Cross-Border Payment Gateway — 一键启动脚本
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
GATEWAY_DIR="$SCRIPT_DIR/gateway"
DASHBOARD_DIR="$SCRIPT_DIR/dashboard"
DATA_DIR="$GATEWAY_DIR/data"
LOG_FILE="$SCRIPT_DIR/gateway.log"
PID_FILE="$SCRIPT_DIR/gateway.pid"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

banner() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║     Aspira Cross-Border Payment Gateway Launcher         ║"
    echo "╚══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  start       Build and start the gateway (default)"
    echo "  stop        Stop the running gateway"
    echo "  restart     Restart the gateway"
    echo "  status      Show gateway status"
    echo "  logs        Tail gateway logs"
    echo "  clean       Clean build artifacts and database"
    echo "  build       Build only, don't start"
    echo "  dev         Start in dev mode (fresh database, verbose logs)"
    echo "  help        Show this help"
    echo ""
    echo "Examples:"
    echo "  $0              # Start the gateway"
    echo "  $0 dev          # Fresh start with clean database"
    echo "  $0 stop         # Stop the gateway"
    echo "  $0 logs         # Watch real-time logs"
}

log_info()  { echo -e "${GREEN}[INFO]${NC}  $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# -----------------------------------------------------------
# 检查依赖
# -----------------------------------------------------------
check_deps() {
    log_info "Checking dependencies..."

    if ! command -v go &> /dev/null; then
        log_error "Go is not installed. Please install Go 1.22+."
        exit 1
    fi

    GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' | head -1)
    log_info "Go version: $GO_VERSION"

    if [ ! -d "$DASHBOARD_DIR" ]; then
        log_error "Dashboard directory not found: $DASHBOARD_DIR"
        exit 1
    fi
}

# -----------------------------------------------------------
# 构建
# -----------------------------------------------------------
build() {
    log_info "Building gateway..."
    cd "$GATEWAY_DIR"

    # 设置 Go proxy（国内网络环境）
    export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
    export GONOSUMCHECK='*'

    # 下载依赖
    log_info "Downloading dependencies..."
    go mod tidy 2>&1 | tail -3

    # 编译
    log_info "Compiling..."
    go build -ldflags="-s -w" -o gateway . 2>&1

    if [ -f gateway ]; then
        local size=$(du -h gateway | cut -f1)
        log_info "Build successful (binary size: $size)"
    else
        log_error "Build failed"
        exit 1
    fi
}

# -----------------------------------------------------------
# 启动
# -----------------------------------------------------------
start() {
    cd "$GATEWAY_DIR"

    # 检查是否已在运行
    if [ -f "$PID_FILE" ] && kill -0 $(cat "$PID_FILE") 2>/dev/null; then
        log_warn "Gateway is already running (PID: $(cat $PID_FILE))"
        echo "  Dashboard: http://localhost:8080"
        echo "  API:       http://localhost:8080/api/v1"
        echo "  Login:     admin / admin123"
        return 0
    fi

    # 确保 data 目录存在
    mkdir -p "$DATA_DIR"

    # 启动
    log_info "Starting gateway..."
    nohup ./gateway configs/config.yaml > "$LOG_FILE" 2>&1 &
    local pid=$!
    echo $pid > "$PID_FILE"

    # 等待启动
    log_info "Waiting for gateway to be ready..."
    for i in $(seq 1 30); do
        if curl -s http://localhost:8080/api/v1/auth/login \
            -X POST -H "Content-Type: application/json" \
            -d '{"username":"admin","password":"admin123"}' > /dev/null 2>&1; then
            echo ""
            echo -e "${GREEN}╔══════════════════════════════════════════════════════════╗${NC}"
            echo -e "${GREEN}║  Gateway is ready!                                      ║${NC}"
            echo -e "${GREEN}╠══════════════════════════════════════════════════════════╣${NC}"
            echo -e "${GREEN}║${NC}  Dashboard: ${CYAN}http://localhost:8080${NC}                         ${GREEN}║${NC}"
            echo -e "${GREEN}║${NC}  API Base:  ${CYAN}http://localhost:8080/api/v1${NC}                 ${GREEN}║${NC}"
            echo -e "${GREEN}║${NC}  WebSocket: ${CYAN}ws://localhost:8080/ws${NC}                       ${GREEN}║${NC}"
            echo -e "${GREEN}║${NC}  Login:     ${YELLOW}admin / admin123${NC}                             ${GREEN}║${NC}"
            echo -e "${GREEN}║${NC}  PID:       ${pid}                                            ${GREEN}║${NC}"
            echo -e "${GREEN}║${NC}  Logs:      ${LOG_FILE}  ${GREEN}║${NC}"
            echo -e "${GREEN}╚══════════════════════════════════════════════════════════╝${NC}"
            return 0
        fi
        printf "."
        sleep 1
    done

    echo ""
    log_error "Gateway failed to start. Check logs: $LOG_FILE"
    echo ""
    echo "=== Last 20 lines of log ==="
    tail -20 "$LOG_FILE"
    exit 1
}

# -----------------------------------------------------------
# 停止
# -----------------------------------------------------------
stop() {
    if [ ! -f "$PID_FILE" ]; then
        log_warn "No PID file found. Trying to find process on port 8080..."
        local pid=$(lsof -ti:8080 2>/dev/null)
        if [ -n "$pid" ]; then
            kill $pid 2>/dev/null
            sleep 1
            log_info "Killed process $pid on port 8080"
        else
            log_info "Gateway is not running"
        fi
        return 0
    fi

    local pid=$(cat "$PID_FILE")
    if kill -0 $pid 2>/dev/null; then
        log_info "Stopping gateway (PID: $pid)..."
        kill $pid
        sleep 2
        if kill -0 $pid 2>/dev/null; then
            log_warn "Gateway didn't stop gracefully, forcing..."
            kill -9 $pid 2>/dev/null
        fi
        log_info "Gateway stopped"
    else
        log_info "Gateway is not running"
    fi

    rm -f "$PID_FILE"
}

# -----------------------------------------------------------
# 状态
# -----------------------------------------------------------
status() {
    echo ""
    echo -e "${CYAN}=== Gateway Status ===${NC}"
    echo ""

    if [ -f "$PID_FILE" ]; then
        local pid=$(cat "$PID_FILE")
        if kill -0 $pid 2>/dev/null; then
            echo -e "  Status:    ${GREEN}Running${NC} (PID: $pid)"
        else
            echo -e "  Status:    ${RED}Stopped${NC} (stale PID: $pid)"
        fi
    else
        echo -e "  Status:    ${YELLOW}Not running${NC}"
    fi

    # Check port
    local port_pid=$(lsof -ti:8080 2>/dev/null)
    if [ -n "$port_pid" ]; then
        echo "  Port 8080: in use by PID $port_pid"
    else
        echo "  Port 8080: free"
    fi

    # Check database
    if [ -f "$DATA_DIR/payment_gateway.db" ]; then
        local db_size=$(du -h "$DATA_DIR/payment_gateway.db" | cut -f1)
        echo "  Database:  $DATA_DIR/payment_gateway.db ($db_size)"
    else
        echo "  Database:  not found"
    fi

    # Check binary
    if [ -f "$GATEWAY_DIR/gateway" ]; then
        local bin_size=$(du -h "$GATEWAY_DIR/gateway" | cut -f1)
        echo "  Binary:    $GATEWAY_DIR/gateway ($bin_size)"
    else
        echo "  Binary:    not built"
    fi

    echo ""
}

# -----------------------------------------------------------
# 日志
# -----------------------------------------------------------
logs() {
    if [ -f "$LOG_FILE" ]; then
        tail -f "$LOG_FILE"
    else
        log_error "No log file found at $LOG_FILE"
    fi
}

# -----------------------------------------------------------
# 清理
# -----------------------------------------------------------
clean() {
    log_info "Cleaning build artifacts..."
    rm -f "$GATEWAY_DIR/gateway"
    rm -f "$GATEWAY_DIR/data/payment_gateway.db"
    rm -f "$GATEWAY_DIR/data/payment_gateway.db-shm"
    rm -f "$GATEWAY_DIR/data/payment_gateway.db-wal"
    rm -f "$PID_FILE"
    rm -f "$LOG_FILE"
    log_info "Clean complete"
}

# -----------------------------------------------------------
# 开发模式（全新数据库）
# -----------------------------------------------------------
dev() {
    log_info "Starting in DEV mode (fresh database)..."
    stop 2>/dev/null
    clean
    build
    start
}

# -----------------------------------------------------------
# Main
# -----------------------------------------------------------
main() {
    banner

    local cmd="${1:-start}"

    case "$cmd" in
        start)
            check_deps
            if [ ! -f "$GATEWAY_DIR/gateway" ]; then
                build
            fi
            start
            ;;
        stop)
            stop
            ;;
        restart)
            stop
            sleep 1
            check_deps
            build
            start
            ;;
        status)
            status
            ;;
        logs)
            logs
            ;;
        clean)
            stop 2>/dev/null
            clean
            ;;
        build)
            check_deps
            build
            ;;
        dev)
            check_deps
            dev
            ;;
        help|--help|-h)
            usage
            ;;
        *)
            log_error "Unknown command: $cmd"
            usage
            exit 1
            ;;
    esac
}

main "$@"
