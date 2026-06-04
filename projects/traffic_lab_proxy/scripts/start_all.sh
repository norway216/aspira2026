#!/bin/bash
# TrafficLab - 一键启动所有核心服务
# 用法: ./scripts/start_all.sh [--no-target]
set -e

# ── 配色 ────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'

# ── 默认端口 ────────────────────────────────────────
CC_PORT=8080
CC_METRICS_PORT=8090
PROXY_PORT=10801
PROXY_METRICS_PORT=9200
TARGET_PORT=18888
NODE_ID="node-01"
NODE_SECRET="node-secret-local"

# ── 解析参数 ────────────────────────────────────────
NO_TARGET=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --no-target) NO_TARGET=true; shift ;;
    -h|--help)
      echo "TrafficLab 一键启动脚本"
      echo "用法: $0 [--no-target]"
      echo "  --no-target  不启动目标测试服务器"
      exit 0 ;;
    *) echo "未知参数: $1"; exit 1 ;;
  esac
done

# ── 清理函数 ────────────────────────────────────────
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DB_FILE="$PROJECT_DIR/data/trafficlab.db"
LOG_DIR="$PROJECT_DIR/data/logs"
PID_DIR="$PROJECT_DIR/data/pids"

mkdir -p "$LOG_DIR" "$PID_DIR" "$(dirname "$DB_FILE")"

cleanup() {
    echo -e "\n${YELLOW}[INFO]${NC} 正在停止所有服务..."
    for f in "$PID_DIR"/*.pid; do
        [[ -f "$f" ]] && kill "$(cat "$f")" 2>/dev/null && echo "  已停止 $(basename "$f" .pid)"
    done
    # 确保端口释放
    for port in $CC_PORT $PROXY_PORT $TARGET_PORT; do
        lsof -ti :$port 2>/dev/null | xargs -r kill 2>/dev/null || true
    done
    rm -rf "$PID_DIR"
    echo -e "${GREEN}[OK]${NC} 所有服务已停止"
}
trap cleanup EXIT INT TERM

# ── 检查依赖 ────────────────────────────────────────
for cmd in go curl python3; do
    command -v $cmd >/dev/null 2>&1 || { echo -e "${RED}[ERRO]${NC} 缺少命令: $cmd"; exit 1; }
done

# ── 构建 ──────────────────────────────────────────
echo -e "${GREEN}═══ TrafficLab 一键启动 ═══${NC}"
echo ""

cd "$PROJECT_DIR"

echo -e "${CYAN}[1/5]${NC} 构建二进制文件..."
go build -o /tmp/trafficlab-cc ./cmd/control-center 2>/dev/null
go build -o /tmp/trafficlab-pn ./cmd/proxy-node 2>/dev/null
echo -e "      ${GREEN}✅ 构建完成${NC}"

# ── 启动目标测试服务器 ─────────────────────────────
if $NO_TARGET; then
    echo -e "${CYAN}[2/5]${NC} 跳过目标测试服务器"
else
    echo -e "${CYAN}[2/5]${NC} 启动目标测试服务器 (port $TARGET_PORT)..."
    mkdir -p /tmp/trafficlab-target
    echo "<html><body><h1>TrafficLab Target Server</h1><p>若看到此页面则代理转发成功</p></body></html>" > /tmp/trafficlab-target/index.html
    python3 -c "
import http.server, os
os.chdir('/tmp/trafficlab-target')
s = http.server.HTTPServer(('', $TARGET_PORT), http.server.SimpleHTTPRequestHandler)
s.serve_forever()
" > "$LOG_DIR/target.log" 2>&1 &
    echo $! > "$PID_DIR/target.pid"
    echo -e "      ${GREEN}✅ 目标服务器已启动${NC}"
fi

# ── 启动控制中心 ─────────────────────────────────
echo -e "${CYAN}[3/5]${NC} 启动控制中心 (port $CC_PORT)..."
LISTEN_ADDR=:$CC_PORT \
DATABASE_DSN="$DB_FILE" \
WEB_ROOT="$PROJECT_DIR/web" \
ADMIN_USER=admin \
ADMIN_PASS=admin123 \
SCHEDULER_STRATEGY=composite \
NODE_STALE_TIMEOUT=30s \
/tmp/trafficlab-cc > "$LOG_DIR/control-center.log" 2>&1 &
echo $! > "$PID_DIR/control-center.pid"

# 等待控制中心就绪
for i in $(seq 1 30); do
    if curl -sf http://localhost:$CC_PORT/metrics > /dev/null 2>&1; then
        echo -e "      ${GREEN}✅ 控制中心已就绪${NC}"
        break
    fi
    if [[ $i -eq 30 ]]; then
        echo -e "      ${RED}❌ 控制中心启动超时${NC}"
        exit 1
    fi
    sleep 1
done

# ── 初始化用户和节点 ──────────────────────────────
echo -e "${CYAN}[4/5]${NC} 初始化测试用户和节点..."

# 创建测试用户
USER_TOKEN=""
CREATE_RESP=$(curl -sf -u admin:admin123 -X POST http://localhost:$CC_PORT/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"testpass","max_rate_mbps":50,"max_connections":20,"traffic_total":0}' 2>/dev/null) || true

if echo "$CREATE_RESP" | grep -q '"success":true'; then
    USER_TOKEN=$(echo "$CREATE_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])" 2>/dev/null)
    echo -e "      用户 testuser 已创建, token: ${USER_TOKEN:0:12}..."
else
    # 用户可能已存在，获取 token
    USER_TOKEN=$(curl -sf -u admin:admin123 http://localhost:$CC_PORT/api/v1/users 2>/dev/null | \
      python3 -c "import sys,json; d=json.load(sys.stdin)['data']; print([u['token'] for u in d if u['username']=='testuser'][0] if d else '')" 2>/dev/null) || true
    if [[ -z "$USER_TOKEN" ]]; then
        echo -e "      ${YELLOW}⚠️  无法创建或获取测试用户${NC}"
    else
        echo -e "      使用已存在的测试用户, token: ${USER_TOKEN:0:12}..."
    fi
fi

# 注册节点
curl -sf -u admin:admin123 -X POST http://localhost:$CC_PORT/api/v1/nodes/register \
  -H "Content-Type: application/json" \
  -d "{\"node_id\":\"$NODE_ID\",\"name\":\"Local Proxy\",\"ip\":\"127.0.0.1\",\"proxy_port\":$PROXY_PORT,\"max_bandwidth_mbps\":100,\"max_connections\":1000,\"node_secret\":\"$NODE_SECRET\"}" \
  > /dev/null 2>&1 || true
echo -e "      节点 $NODE_ID 已注册"

# ── 启动代理节点 ─────────────────────────────────
echo -e "${CYAN}[5/5]${NC} 启动代理节点 (proxy:$PROXY_PORT, metrics:$PROXY_METRICS_PORT)..."

NODE_ID="$NODE_ID" \
LISTEN_ADDR=":$PROXY_PORT" \
NODE_SECRET="$NODE_SECRET" \
CONTROL_CENTER_URL="http://localhost:$CC_PORT" \
METRICS_LISTEN_ADDR=":$PROXY_METRICS_PORT" \
HEARTBEAT_INTERVAL=10s \
TRAFFIC_REPORT_INTERVAL=15s \
POLICY_FETCH_INTERVAL=30s \
MAX_CONNECTIONS=1000 \
/tmp/trafficlab-pn > "$LOG_DIR/proxy-node.log" 2>&1 &
echo $! > "$PID_DIR/proxy-node.pid"

# 等待代理节点就绪
for i in $(seq 1 20); do
    if curl -sf http://localhost:$PROXY_METRICS_PORT/metrics > /dev/null 2>&1; then
        echo -e "      ${GREEN}✅ 代理节点已就绪${NC}"
        break
    fi
    if [[ $i -eq 20 ]]; then
        echo -e "      ${YELLOW}⚠️  代理节点启动超时（检查日志: $LOG_DIR/proxy-node.log）${NC}"
    fi
    sleep 1
done

# ── 验证 ──────────────────────────────────────────
echo ""
NODE_STATUS=$(curl -sf -u admin:admin123 http://localhost:$CC_PORT/api/v1/nodes 2>/dev/null | \
  python3 -c "import sys,json; nodes=json.load(sys.stdin)['data']; print(nodes[0]['status'] if nodes else 'unknown')" 2>/dev/null || echo "unknown")
SCHED_CHECK=$(curl -sf -u admin:admin123 "http://localhost:$CC_PORT/api/v1/schedule?user_id=testuser" 2>/dev/null | \
  python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['node_id'] if d.get('success') else 'FAIL')" 2>/dev/null || echo "FAIL")

# ── 打印信息 ──────────────────────────────────────
echo -e "${GREEN}══════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}  TrafficLab 启动成功！${NC}"
echo -e "${GREEN}══════════════════════════════════════════════════════${NC}"
echo ""
echo -e "  ${CYAN}控制中心 API${NC}    http://localhost:$CC_PORT/api/v1/"
echo -e "  ${CYAN}Web 管理面板${NC}    http://localhost:$CC_PORT/admin/dashboard"
echo -e "  ${CYAN}        登录${NC}    admin / admin123"
echo ""
echo -e "  ${CYAN}代理节点${NC}        localhost:$PROXY_PORT"
echo -e "  ${CYAN}测试用户${NC}        testuser / testpass"
if [[ -n "$USER_TOKEN" ]]; then
echo -e "  ${CYAN}用户 Token${NC}      $USER_TOKEN"
fi
echo ""
if ! $NO_TARGET; then
echo -e "  ${CYAN}目标服务器${NC}      http://localhost:$TARGET_PORT"
echo ""
echo -e "  ${GREEN}快速测试:${NC}"
echo -e "  ${YELLOW}# 直接访问目标服务器${NC}"
echo "  curl http://localhost:$TARGET_PORT/"
echo ""
if [[ -n "$USER_TOKEN" ]]; then
echo -e "  ${YELLOW}# 通过代理访问（验证转发+认证）${NC}"
echo "  curl --proxytunnel -x \"http://testuser:${USER_TOKEN}@127.0.0.1:$PROXY_PORT\" http://localhost:$TARGET_PORT/"
fi
echo ""
fi
echo -e "  ${CYAN}Prometheus${NC}      http://localhost:$CC_PORT/metrics"
echo -e "  ${CYAN}代理节点指标${NC}    http://localhost:$PROXY_METRICS_PORT/metrics"
echo ""
echo -e "  ${CYAN}日志文件${NC}        $LOG_DIR/"
echo ""
echo -e "  ${YELLOW}按 Ctrl+C 停止所有服务${NC}"
echo ""

# ── 保持运行 ──────────────────────────────────────
while true; do
    # 检查所有服务是否正常
    if ! kill -0 "$(cat "$PID_DIR/control-center.pid")" 2>/dev/null; then
        echo -e "${RED}[ERRO] 控制中心异常退出！${NC}"
        exit 1
    fi
    if ! kill -0 "$(cat "$PID_DIR/proxy-node.pid")" 2>/dev/null; then
        echo -e "${RED}[ERRO] 代理节点异常退出！${NC}"
        exit 1
    fi
    sleep 5
done
