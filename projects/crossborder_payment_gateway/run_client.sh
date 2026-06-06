#!/bin/bash
set -e

# ============================================================
#  Aspira Payment Gateway — Benchmark Client Runner
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLIENT_DIR="$SCRIPT_DIR/client"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

banner() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║     Aspira Payment Gateway — Benchmark Client           ║"
    echo "╚══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

usage() {
    echo "Usage: $0 [preset|options...]"
    echo ""
    echo "Presets:"
    echo "  quick       Quick smoke test (5 workers, 10s, unlimited rate)"
    echo "  standard    Standard benchmark (20 workers, 500 TPS, 30s)"
    echo "  stress      Stress test (100 workers, unlimited rate, 60s)"
    echo "  mixed       Mixed workload (20 workers, 100 TPS, 30s)"
    echo "  query       Read-only query test (10 workers, unlimited, 10s)"
    echo ""
    echo "Custom options (forwarded to benchmark client):"
    echo "  --mode=MODE          payment | query | mixed (default: payment)"
    echo "  --concurrency=N      Number of workers (default: 20)"
    echo "  --rate=N             Target TPS, 0=unlimited (default: 0)"
    echo "  --duration=D         Test duration (default: 30s)"
    echo "  --ramp-up=D          Ramp-up time (default: 5s)"
    echo "  --target=URL         Gateway URL (default: http://localhost:8080)"
    echo "  --username=USER      Login username (default: admin)"
    echo "  --password=PASS      Login password (default: admin123)"
    echo "  --report=FILE        Save JSON report to file"
    echo ""
    echo "Examples:"
    echo "  $0 quick"
    echo "  $0 standard"
    echo "  $0 --mode=payment --concurrency=50 --rate=1000 --duration=60s"
    echo "  $0 --mode=mixed --concurrency=20 --duration=30s --report=results.json"
}

log_info()  { echo -e "${GREEN}[INFO]${NC}  $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; }

# -----------------------------------------------------------
# Build
# -----------------------------------------------------------
build() {
    log_info "Building benchmark client..."
    cd "$CLIENT_DIR"

    export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
    export GONOSUMCHECK='*'

    go mod tidy 2>&1 | tail -1
    go build -ldflags="-s -w" -o client . 2>&1

    if [ -f client ]; then
        local size=$(du -h client | cut -f1)
        log_info "Build successful (binary: $size)"
    else
        echo -e "${RED}[ERROR]${NC} Build failed"
        exit 1
    fi
}

# -----------------------------------------------------------
# Check if gateway is reachable
# -----------------------------------------------------------
check_gateway() {
    local target="${1:-http://localhost:8080}"
    log_info "Checking gateway at $target..."

    if curl -s -o /dev/null -w "%{http_code}" --connect-timeout 3 "$target/api/v1/auth/login" \
        -X POST -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}' 2>/dev/null | grep -q "200"; then
        log_info "Gateway is reachable"
        return 0
    else
        log_warn "Gateway is not reachable at $target"
        echo "  Make sure the gateway is running:"
        echo "    cd $(dirname "$SCRIPT_DIR") && ./run.sh start"
        return 1
    fi
}

# -----------------------------------------------------------
# Main
# -----------------------------------------------------------
main() {
    banner

    cd "$CLIENT_DIR"

    # Build if needed
    if [ ! -f "$CLIENT_DIR/client" ]; then
        build
    fi

    local cmd="${1:-help}"
    local extra_args=""

    case "$cmd" in
        quick)
            log_info "Preset: Quick smoke test"
            extra_args="--mode=payment --concurrency=5 --rate=0 --duration=10s --ramp-up=2s"
            ;;
        standard)
            log_info "Preset: Standard benchmark"
            extra_args="--mode=payment --concurrency=20 --rate=500 --duration=30s"
            ;;
        stress)
            log_info "Preset: Stress test"
            extra_args="--mode=payment --concurrency=100 --rate=0 --duration=60s --ramp-up=10s"
            ;;
        mixed)
            log_info "Preset: Mixed workload"
            extra_args="--mode=mixed --concurrency=20 --rate=100 --duration=30s"
            ;;
        query)
            log_info "Preset: Query-only test"
            extra_args="--mode=query --concurrency=10 --rate=0 --duration=10s"
            ;;
        help|--help|-h)
            usage
            exit 0
            ;;
        --*)
            # Custom options forwarded directly
            extra_args="$*"
            ;;
        *)
            log_info "Running with arguments: $*"
            extra_args="$*"
            ;;
    esac

    # Check gateway unless user specified custom target
    local target_flag=$(echo "$extra_args" | grep -o '\-\-target=[^ ]*' || true)
    if [ -z "$target_flag" ]; then
        check_gateway "http://localhost:8080" || true
    fi

    echo ""
    log_info "Starting benchmark..."
    echo ""

    # Run the client
    ./client $extra_args

    exit_code=$?

    echo ""
    if [ $exit_code -eq 0 ]; then
        log_info "Benchmark completed successfully"
    else
        log_warn "Benchmark exited with code $exit_code"
    fi
}

main "$@"
