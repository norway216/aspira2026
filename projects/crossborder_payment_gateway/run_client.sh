#!/bin/bash
# ============================================================
#  Aspira Pay — Multi-Currency Continuous Trader (Go)
#  8 workers, 8 currencies, 10,000,000 balance each
# ============================================================
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLIENT_DIR="$SCRIPT_DIR/client"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# Defaults
GATEWAY="${GATEWAY_URL:-http://localhost:8080}"
BATCH_SIZE="${BATCH_SIZE:-3}"
BATCH_PAUSE="${BATCH_PAUSE:-800ms}"
REPORT_EVERY="${REPORT_EVERY:-10s}"
BALANCE="${BALANCE:-10000000}"

banner() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║   Aspira Pay — Multi-Currency Trader (8 Workers)        ║"
    echo "╚══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --target=URL       Gateway URL (default: http://localhost:8080)"
    echo "  --batch=N          Txns per batch per worker (default: 5)"
    echo "  --pause=D          Pause between batches (default: 1s)"
    echo "  --report=D         Stats report interval (default: 10s)"
    echo "  --balance=N        Initial balance per account (default: 10000000)"
    echo "  --username=USER    Login username (default: admin)"
    echo "  --password=PASS    Login password (default: admin123)"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Default"
    echo "  $0 --batch=10 --pause=500ms           # High throughput"
    echo "  $0 --batch=20 --pause=2s --target=... # Large batches"
    echo ""
    echo "Workers (8 currencies):"
    echo "  USD acct-001   HKD acct-013   SGD acct-012   JPY acct-006"
    echo "  AUD acct-010   EUR acct-005   CNY acct-002   CAD acct-009"
    echo ""
    echo "Each account holds $BALANCE in native currency."
}

# Parse custom args
for arg in "$@"; do
    case "$arg" in
        --target=*)    GATEWAY="${arg#*=}" ;;
        --batch=*)     BATCH_SIZE="${arg#*=}" ;;
        --pause=*)     BATCH_PAUSE="${arg#*=}" ;;
        --report=*)    REPORT_EVERY="${arg#*=}" ;;
        --balance=*)   BALANCE="${arg#*=}" ;;
        --username=*)  LOGIN_USER="${arg#*=}" ;;
        --password=*)  LOGIN_PASS="${arg#*=}" ;;
        -h|--help)     usage; exit 0 ;;
    esac
done

# -----------------------------------------------------------
# Build
# -----------------------------------------------------------
build() {
    echo -e "${YELLOW}[BUILD]${NC} Building Go client..."
    cd "$CLIENT_DIR"
    export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
    go mod tidy 2>&1 | tail -1
    go build -ldflags="-s -w" -o client . 2>&1
    if [ -f client ]; then
        local size; size=$(du -h client | cut -f1)
        echo -e "  ${GREEN}✓${NC} Build OK ($size)"
    else
        echo -e "  ${RED}✗${NC} Build failed"
        exit 1
    fi
}

# -----------------------------------------------------------
# Check gateway reachable
# -----------------------------------------------------------
check_gateway() {
    echo -ne "${YELLOW}[CHECK]${NC} Gateway $GATEWAY ... "
    if curl -s -o /dev/null --connect-timeout 3 \
        -X POST "$GATEWAY/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}' 2>/dev/null; then
        echo -e "${GREEN}OK${NC}"
    else
        echo -e "${RED}DOWN${NC}"
        echo "  Start: cd gateway && ./gateway"
        exit 1
    fi
}

# -----------------------------------------------------------
# Main
# -----------------------------------------------------------
main() {
    banner

    [ ! -f "$CLIENT_DIR/client" ] && build
    check_gateway

    echo ""
    echo -e "${GREEN}[START]${NC} 8 workers | batch=$BATCH_SIZE | pause=$BATCH_PAUSE"
    echo "  USD HKD SGD JPY AUD EUR CNY CAD"
    echo ""

    cd "$CLIENT_DIR"
    exec ./client \
        --target="$GATEWAY" \
        --batch="$BATCH_SIZE" \
        --pause="$BATCH_PAUSE" \
        --balance="$BALANCE" \
        --report="$REPORT_EVERY" \
        --username="${LOGIN_USER:-admin}" \
        --password="${LOGIN_PASS:-admin123}"
}

main "$@"
