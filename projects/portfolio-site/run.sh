#!/usr/bin/env bash
#
# Aspira Studio — 一键运行脚本
# 用法: ./run.sh [命令]
#   命令:
#     start    构建并启动服务 (默认)
#     dev      开发模式，文件变更自动重载
#     stop     停止服务
#     restart  重启服务
#     build    仅构建
#     clean    清理构建产物
#     status   查看服务状态
#     logs     查看日志
#     test     运行测试
#     help     显示帮助

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_NAME="portfolio-site"
SERVER_BIN="${SCRIPT_DIR}/portfolio-server"
PID_FILE="${SCRIPT_DIR}/.server.pid"
LOG_DIR="${SCRIPT_DIR}/logs"
LOG_FILE="${LOG_DIR}/server.log"
DATA_DIR="${SCRIPT_DIR}/data"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认配置
export SERVER_ADDR="${SERVER_ADDR:-:8080}"
export DB_DRIVER="${DB_DRIVER:-sqlite3}"
export DB_DSN="${DB_DSN:-${DATA_DIR}/portfolio.db}"
export SITE_TITLE="${SITE_TITLE:-Aspira Studio}"
export SESSION_SECRET="${SESSION_SECRET:-$(openssl rand -hex 32 2>/dev/null || echo 'dev-secret-change-me')}"

log_info()  { echo -e "${GREEN}[INFO]${NC}  $(date '+%H:%M:%S') $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $(date '+%H:%M:%S') $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $(date '+%H:%M:%S') $*"; }

# 检查是否已安装必要的运行时
check_prereqs() {
    if ! command -v go &>/dev/null; then
        log_error "未找到 Go 编译器，请先安装 Go 1.22+"
        exit 1
    fi
    log_info "Go 版本: $(go version)"
}

# 创建必要的目录
init_dirs() {
    mkdir -p "${LOG_DIR}" "${DATA_DIR}"
    mkdir -p "${SCRIPT_DIR}/web/static/uploads"
}

# 下载依赖（使用本地缓存）
download_deps() {
    if [ ! -f "${SCRIPT_DIR}/go.sum" ] || [ "${SCRIPT_DIR}/go.sum" -ot "${SCRIPT_DIR}/go.mod" ]; then
        log_info "下载依赖..."
        GONOSUMCHECK='*' GONOSUMDB='*' \
            GOPROXY="file://${GOPATH:-$HOME/go}/pkg/mod/cache/download,direct" \
            go mod download 2>&1 | while read -r line; do
            log_info "  $line"
        done
    fi
}

# 构建
build() {
    log_info "构建 ${PROJECT_NAME}..."
    init_dirs

    # 获取构建信息
    local build_time build_commit
    build_time="$(date -u '+%Y-%m-%d_%H:%M:%S')"
    build_commit="$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"

    go build -o "${SERVER_BIN}" \
        -ldflags="-s -w -X main.buildTime=${build_time} -X main.buildCommit=${build_commit}" \
        ./cmd/server/

    log_info "构建完成: ${SERVER_BIN} ($(du -h "${SERVER_BIN}" | cut -f1))"
}

# 启动服务（守护进程模式）
start_server() {
    if is_running; then
        log_warn "服务已在运行中 (PID: $(cat "${PID_FILE}"))"
        return 0
    fi

    if [ ! -x "${SERVER_BIN}" ]; then
        build
    fi

    log_info "启动 ${PROJECT_NAME}..."
    local host="${SERVER_ADDR}"
    log_info "  地址: http://localhost${host}"
    log_info "  管理后台: http://localhost${host}/admin"
    log_info "  账号: admin / admin123"
    log_info "  日志: ${LOG_FILE}"

    # 后台启动，重定向输出到日志文件
    nohup "${SERVER_BIN}" >> "${LOG_FILE}" 2>&1 &
    local pid=$!
    echo "${pid}" > "${PID_FILE}"

    # 等待就绪
    sleep 1
    if is_running; then
        log_info "服务启动成功 (PID: ${pid})"
    else
        log_error "服务启动失败，请查看日志: tail -f ${LOG_FILE}"
        rm -f "${PID_FILE}"
        exit 1
    fi
}

# 开发模式（前台运行，含基础文件监控）
dev_mode() {
    log_info "开发模式启动 (前台运行)..."

    # 首次构建
    if [ ! -x "${SERVER_BIN}" ]; then
        build
    fi

    log_info "监听文件变更... (Ctrl+C 退出)"
    log_info "  地址: http://localhost${SERVER_ADDR}"

    # 使用 inotifywait 监控文件变更（如果可用）
    if command -v inotifywait &>/dev/null; then
        # 后台启动服务
        "${SERVER_BIN}" &
        local server_pid=$!
        trap "kill ${server_pid} 2>/dev/null; exit 0" INT TERM

        while true; do
            inotifywait -r -e modify,create,delete \
                --exclude '(\.git|data|logs|portfolio-server)' \
                "${SCRIPT_DIR}" 2>/dev/null
            log_info "检测到文件变更，重新构建..."
            build && {
                kill "${server_pid}" 2>/dev/null
                sleep 0.5
                "${SERVER_BIN}" &
                server_pid=$!
                log_info "热重载完成"
            }
        done
    else
        # 无 inotifywait，直接前台运行
        log_warn "未安装 inotify-tools，无法自动重载 (apt install inotify-tools)"
        exec "${SERVER_BIN}"
    fi
}

# 停止服务
stop_server() {
    if ! is_running; then
        log_info "服务未运行"
        rm -f "${PID_FILE}"
        return 0
    fi

    local pid
    pid="$(cat "${PID_FILE}")"
    log_info "停止服务 (PID: ${pid})..."

    kill "${pid}" 2>/dev/null || true
    sleep 1

    # 如果还没结束，强制杀掉
    if kill -0 "${pid}" 2>/dev/null; then
        log_warn "强制终止..."
        kill -9 "${pid}" 2>/dev/null || true
    fi

    rm -f "${PID_FILE}"
    log_info "服务已停止"
}

# 重启服务
restart_server() {
    stop_server
    sleep 1
    start_server
}

# 检查服务是否运行
is_running() {
    if [ -f "${PID_FILE}" ]; then
        local pid
        pid="$(cat "${PID_FILE}")"
        if kill -0 "${pid}" 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# 查看状态
show_status() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    echo -e "${BLUE}  Aspira Studio — Go Web System${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    echo ""
    if is_running; then
        local pid
        pid="$(cat "${PID_FILE}")"
        echo -e "  状态:     ${GREEN}运行中${NC} (PID: ${pid})"
        echo "  地址:     http://localhost${SERVER_ADDR}"
        echo "  管理后台:  http://localhost${SERVER_ADDR}/admin"

        # 检查服务健康状态
        local health
        health="$(curl -s -o /dev/null -w '%{http_code}' "http://localhost${SERVER_ADDR}" 2>/dev/null || echo 'N/A')"
        echo "  健康检查:  HTTP ${health}"

        # CPU/内存
        if command -v ps &>/dev/null; then
            local rss cpu
            rss="$(ps -o rss= -p "${pid}" 2>/dev/null | tr -d ' ' || echo 'N/A')"
            cpu="$(ps -o %cpu= -p "${pid}" 2>/dev/null | tr -d ' ' || echo 'N/A')"
            echo "  内存:      ${rss}KB"
            echo "  CPU:       ${cpu}%"
        fi
    else
        echo -e "  状态:     ${RED}未运行${NC}"
    fi
    echo ""
    echo "  日志文件:  ${LOG_FILE}"
    echo "  数据库:    ${DB_DSN}"
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    echo ""
}

# 查看日志
show_logs() {
    if [ -f "${LOG_FILE}" ]; then
        tail -f "${LOG_FILE}"
    else
        log_info "日志文件尚未生成"
    fi
}

# 清理
clean() {
    log_info "清理构建产物..."
    rm -f "${SERVER_BIN}" "${PID_FILE}"
    log_info "清理完成"
}

# 帮助
show_help() {
    echo ""
    echo "Aspira Studio — 一键运行脚本"
    echo ""
    echo "用法: ./run.sh [命令]"
    echo ""
    echo "命令:"
    echo "  start      构建并启动服务 (默认)"
    echo "  dev        开发模式，文件变更自动重载"
    echo "  stop       停止服务"
    echo "  restart    重启服务"
    echo "  build      仅构建二进制"
    echo "  clean      清理构建产物"
    echo "  status     查看服务状态"
    echo "  logs       查看实时日志"
    echo "  test       运行测试"
    echo "  help       显示此帮助"
    echo ""
    echo "环境变量:"
    echo "  SERVER_ADDR        监听地址 (默认 :8080)"
    echo "  DB_DSN             数据库路径 (默认 data/portfolio.db)"
    echo "  SITE_TITLE         站点标题 (默认 Aspira Studio)"
    echo "  SESSION_SECRET     会话密钥"
    echo ""
    echo "示例:"
    echo "  ./run.sh start"
    echo "  SERVER_ADDR=:9090 ./run.sh dev"
    echo ""
}

# 主逻辑
main() {
    cd "${SCRIPT_DIR}"
    local cmd="${1:-start}"

    case "${cmd}" in
        start)   check_prereqs; init_dirs; download_deps; start_server ;;
        dev)     check_prereqs; init_dirs; download_deps; dev_mode ;;
        stop)    stop_server ;;
        restart) restart_server ;;
        build)   check_prereqs; init_dirs; download_deps; build ;;
        clean)   stop_server 2>/dev/null; clean ;;
        status)  show_status ;;
        logs)    show_logs ;;
        test)    go test ./... -v -count=1 ;;
        help|-h|--help) show_help ;;
        *)
            log_error "未知命令: ${cmd}"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
