#!/bin/bash
# ============================================================
#  LAN Monitor - 一键部署启动脚本
#  One-click deployment and launch script
# ============================================================

set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_NAME="lan-monitor"
BIN_PATH="$PROJECT_DIR/$BIN_NAME"
CONFIG_PATH="$PROJECT_DIR/configs/config.yaml"
DATA_DIR="$PROJECT_DIR/data"
PID_FILE="$PROJECT_DIR/.lan-monitor.pid"
LOG_FILE="$PROJECT_DIR/lan-monitor.log"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

banner() {
    echo -e "${BLUE}"
    echo "╔══════════════════════════════════════════════╗"
    echo "║   🖥️  LAN Monitor - 局域网设备监控管理系统  ║"
    echo "║      One-Click Deploy & Launch Script       ║"
    echo "╚══════════════════════════════════════════════╝"
    echo -e "${NC}"
}

log_info()  { echo -e "${GREEN}[INFO]${NC}  $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()  { echo -e "${BLUE}[STEP]${NC}  $1"; }

# ----------------------------------------------------------
# 1. Check environment
# ----------------------------------------------------------
check_env() {
    log_step "检查运行环境..."

    # Check OS
    case "$(uname -s)" in
        Linux)  log_info "操作系统: Linux" ;;
        Darwin) log_info "操作系统: macOS" ;;
        *)      log_warn "未知操作系统: $(uname -s)，可能不兼容" ;;
    esac

    # Check architecture
    log_info "架构: $(uname -m)"

    # Check if port is available
    PORT=$(grep -oP 'addr:\s*":?\K\d+' "$CONFIG_PATH" 2>/dev/null || echo "8080")
    if command -v ss &>/dev/null; then
        if ss -tlnp | grep -q ":$PORT "; then
            log_warn "端口 $PORT 已被占用"
            log_warn "占用进程: $(ss -tlnp | grep ":$PORT " | awk '{print $NF}')"
        else
            log_info "端口 $PORT 可用"
        fi
    elif command -v lsof &>/dev/null; then
        if lsof -i :$PORT &>/dev/null; then
            log_warn "端口 $PORT 已被占用"
        else
            log_info "端口 $PORT 可用"
        fi
    fi

    # Check if arp-scan is available (optional, improves scanning)
    if command -v arp-scan &>/dev/null; then
        log_info "arp-scan 可用 (增强扫描功能)"
    else
        log_warn "arp-scan 未安装，ARP扫描将使用系统内置方法"
        log_warn "安装建议: sudo apt install arp-scan  (Debian/Ubuntu)"
        log_warn "         sudo yum install arp-scan  (RHEL/CentOS)"
    fi
}

# ----------------------------------------------------------
# 2. Build the application
# ----------------------------------------------------------
build_app() {
    log_step "编译应用程序..."

    cd "$PROJECT_DIR"

    if [ ! -f "go.mod" ]; then
        log_error "go.mod 文件不存在，请确保在项目根目录运行此脚本"
        exit 1
    fi

    if ! command -v go &>/dev/null; then
        log_error "Go 编译器未安装，请先安装 Go 1.21+"
        log_error "下载地址: https://go.dev/dl/"
        exit 1
    fi

    GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+')
    log_info "Go 版本: $GO_VERSION"

    log_info "正在编译 (首次可能需要下载依赖)..."

    # Try building with standard Go proxy
    if go build -ldflags="-s -w" -o "$BIN_PATH" . 2>&1 | tee /tmp/lan-monitor-build.log; then
        log_info "编译成功"
        ls -lh "$BIN_PATH"
    else
        # If network proxy fails, try with local cache
        log_warn "标准编译失败，尝试使用本地缓存..."
        if GONOSUMCHECK='*' GONOSUMDB='*' GOFLAGS=-mod=mod GOPROXY=off go build -ldflags="-s -w" -o "$BIN_PATH" . 2>&1; then
            log_info "使用本地缓存编译成功"
            ls -lh "$BIN_PATH"
        else
            log_error "编译失败，请检查网络连接或 Go 环境"
            log_error "编译日志: /tmp/lan-monitor-build.log"
            exit 1
        fi
    fi
}

# ----------------------------------------------------------
# 3. Setup directories
# ----------------------------------------------------------
setup_dirs() {
    log_step "创建运行时目录..."

    mkdir -p "$DATA_DIR"
    log_info "数据目录: $DATA_DIR"

    # Create default config if not exists
    if [ ! -f "$CONFIG_PATH" ]; then
        log_warn "配置文件不存在，创建默认配置..."
        mkdir -p "$(dirname "$CONFIG_PATH")"
        cat > "$CONFIG_PATH" << 'EOF'
server:
  addr: ":8080"
  mode: "release"

scan:
  enabled: true
  interval_seconds: 60
  subnets:
    - "auto"
  methods:
    - "arp"
    - "ping"
    - "tcp"
  worker_count: 100
  timeout_ms: 800
  offline_threshold: 3
  tcp_ports:
    - 22
    - 80
    - 443
    - 8080
    - 3389

database:
  path: "./data/lan_monitor.db"

jwt:
  secret: "lan-monitor-secret-change-in-production"
  access_token_expire_minutes: 60
  refresh_token_expire_hours: 168

alert:
  new_device_enabled: true
  offline_enabled: true
  high_traffic_enabled: false
  high_traffic_threshold_mbps: 100
EOF
        log_info "默认配置已创建: $CONFIG_PATH"
        log_warn "请修改 jwt.secret 为随机字符串以确保安全"
    fi
}

# ----------------------------------------------------------
# 4. Stop existing instance
# ----------------------------------------------------------
stop_app() {
    if [ -f "$PID_FILE" ]; then
        OLD_PID=$(cat "$PID_FILE")
        if kill -0 "$OLD_PID" 2>/dev/null; then
            log_step "停止旧进程 (PID: $OLD_PID)..."
            kill "$OLD_PID" 2>/dev/null
            # Wait for graceful shutdown
            for i in $(seq 1 10); do
                if ! kill -0 "$OLD_PID" 2>/dev/null; then
                    log_info "旧进程已停止"
                    break
                fi
                sleep 1
            done
            # Force kill if still running
            if kill -0 "$OLD_PID" 2>/dev/null; then
                log_warn "强制终止进程..."
                kill -9 "$OLD_PID" 2>/dev/null
            fi
        fi
        rm -f "$PID_FILE"
    fi
}

# ----------------------------------------------------------
# 5. Start the application
# ----------------------------------------------------------
start_app() {
    log_step "启动 LAN Monitor..."

    cd "$PROJECT_DIR"

    # Check binary
    if [ ! -f "$BIN_PATH" ]; then
        build_app
    fi

    if [ ! -x "$BIN_PATH" ]; then
        chmod +x "$BIN_PATH"
    fi

    # Start in background
    nohup "$BIN_PATH" -config "$CONFIG_PATH" >> "$LOG_FILE" 2>&1 &
    APP_PID=$!
    echo "$APP_PID" > "$PID_FILE"

    # Wait for startup
    log_info "等待服务启动..."
    sleep 2

    if kill -0 "$APP_PID" 2>/dev/null; then
        log_info "============================================"
        log_info "  ✅ LAN Monitor 启动成功!"
        log_info "  PID: $APP_PID"
        log_info "  地址: http://localhost:$PORT"
        log_info "  日志: $LOG_FILE"
        log_info "  默认账号: admin / admin123"
        log_info "============================================"
        echo ""
        log_info "查看日志: tail -f $LOG_FILE"
        log_info "停止服务: $0 stop"
        log_info "查看状态: $0 status"
    else
        log_error "启动失败，请查看日志: $LOG_FILE"
        tail -20 "$LOG_FILE"
        exit 1
    fi
}

# ----------------------------------------------------------
# 6. Show status
# ----------------------------------------------------------
show_status() {
    echo ""
    echo -e "${BLUE}=== LAN Monitor 状态 ===${NC}"

    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
            echo -e "  状态: ${GREEN}运行中${NC}"
            echo "  PID:  $PID"

            # Uptime
            if command -v ps &>/dev/null; then
                UPTIME=$(ps -o etime= -p "$PID" 2>/dev/null | xargs)
                echo "  运行时间: $UPTIME"
            fi

            # Memory usage
            if command -v ps &>/dev/null; then
                MEM=$(ps -o rss= -p "$PID" 2>/dev/null | xargs)
                if [ -n "$MEM" ]; then
                    MEM_MB=$((MEM / 1024))
                    echo "  内存使用: ${MEM_MB}MB"
                fi
            fi

            # Port status
            PORT=$(grep -oP 'addr:\s*":?\K\d+' "$CONFIG_PATH" 2>/dev/null || echo "8080")
            echo "  监听端口: $PORT"
            echo "  访问地址: http://localhost:$PORT"

            # Test HTTP endpoint
            if command -v curl &>/dev/null; then
                HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:$PORT/api/v1/dashboard" \
                    -H "Authorization: Bearer test" 2>/dev/null || echo "N/A")
                if [ "$HTTP_CODE" != "000" ]; then
                    echo "  HTTP 状态: $HTTP_CODE (API 可访问)"
                else
                    echo "  HTTP 状态: 无法连接"
                fi
            fi
        else
            echo -e "  状态: ${RED}未运行${NC} (PID 文件存在但进程已终止)"
        fi
    else
        echo -e "  状态: ${RED}未运行${NC}"
    fi

    # Show data stats
    if [ -f "$DATA_DIR/lan_monitor.db" ]; then
        DB_SIZE=$(ls -lh "$DATA_DIR/lan_monitor.db" | awk '{print $5}')
        echo "  数据库大小: $DB_SIZE"
    fi

    echo ""
}

# ----------------------------------------------------------
# 7. Show logs
# ----------------------------------------------------------
show_logs() {
    LINES=${1:-50}
    if [ -f "$LOG_FILE" ]; then
        echo -e "${BLUE}=== 最近 $LINES 行日志 ===${NC}"
        tail -n "$LINES" "$LOG_FILE"
    else
        log_warn "日志文件不存在: $LOG_FILE"
    fi
}

# ----------------------------------------------------------
# 8. Restart
# ----------------------------------------------------------
restart_app() {
    stop_app
    start_app
}

# ----------------------------------------------------------
# Main
# ----------------------------------------------------------
banner

case "${1:-start}" in
    start)
        check_env
        setup_dirs
        stop_app
        start_app
        ;;

    stop)
        log_step "停止 LAN Monitor..."
        stop_app
        log_info "服务已停止"
        ;;

    restart)
        check_env
        setup_dirs
        restart_app
        ;;

    status)
        show_status
        ;;

    build)
        build_app
        ;;

    logs)
        show_logs "${2:-50}"
        ;;

    dev)
        # Development mode: build and run in foreground
        log_step "开发模式启动..."
        check_env
        setup_dirs
        build_app
        log_info "正在前台运行 (Ctrl+C 停止)..."
        cd "$PROJECT_DIR"
        exec "$BIN_PATH" -config "$CONFIG_PATH"
        ;;

    *)
        echo "用法: $0 {start|stop|restart|status|build|logs|dev}"
        echo ""
        echo "  start    - 一键部署并启动 (默认)"
        echo "  stop     - 停止服务"
        echo "  restart  - 重启服务"
        echo "  status   - 查看运行状态"
        echo "  build    - 仅编译"
        echo "  logs     - 查看日志 (可追加行数: $0 logs 100)"
        echo "  dev      - 开发模式 (前台运行)"
        echo ""
        echo "示例:"
        echo "  $0              # 一键部署启动"
        echo "  $0 status       # 查看状态"
        echo "  $0 logs 100     # 查看最近100行日志"
        echo "  $0 restart      # 重启服务"
        exit 1
        ;;
esac
