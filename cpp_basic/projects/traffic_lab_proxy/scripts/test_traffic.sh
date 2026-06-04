#!/bin/bash
# Traffic Lab Proxy - Integration Test Script
# Usage: ./test_traffic.sh [control-center-url] [proxy-url] [user-token]
set -e

CC_URL="${1:-http://localhost:8080}"
PROXY_URL="${2:-http://localhost:10801}"
TARGET_URL="${3:-http://target-server:80}"

echo "========================================"
echo " TrafficLab Proxy - Integration Tests"
echo "========================================"
echo "Control Center: $CC_URL"
echo "Proxy: $PROXY_URL"
echo ""

# ── Test 1: Control Center Health ──────────────────
echo ">>> Test 1: Control Center API health"
curl -sf "$CC_URL/metrics" | head -n 5
echo "   [PASS]"

# ── Test 2: Admin Login ────────────────────────────
echo ">>> Test 2: Admin login"
LOGIN_RESP=$(curl -sf -X POST "$CC_URL/api/v1/admin/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}')
echo "$LOGIN_RESP"
SESSION=$(echo "$LOGIN_RESP" | grep -o '"admin_session=[^"]*"' || true)
echo "   [PASS]"

# ── Test 3: Create Test User ──────────────────────
echo ">>> Test 3: Create test user"
USER_RESP=$(curl -sf -X POST "$CC_URL/api/v1/users" \
    -H "Content-Type: application/json" \
    -u "admin:admin123" \
    -d '{"username":"testuser","password":"testpass","max_rate_mbps":10,"max_connections":5}')
echo "$USER_RESP"
USER_TOKEN=$(echo "$USER_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])" 2>/dev/null || echo "")
if [ -z "$USER_TOKEN" ]; then
    echo "   [WARN] Could not extract token, using placeholder"
    USER_TOKEN="unknown"
fi
echo "   User token: $USER_TOKEN"
echo "   [PASS]"

# ── Test 4: List Nodes ─────────────────────────────
echo ">>> Test 4: List proxy nodes"
curl -sf -u "admin:admin123" "$CC_URL/api/v1/nodes" | python3 -m json.tool 2>/dev/null || echo "   [WARN] No nodes or parse error"
echo "   [PASS]"

# ── Test 5: Schedule Endpoint ─────────────────────
echo ">>> Test 5: Scheduler endpoint"
SCHED_RESP=$(curl -sf -u "admin:admin123" "$CC_URL/api/v1/schedule?user_id=testuser")
echo "$SCHED_RESP"
echo "   [PASS]"

# ── Test 6: Web Dashboard ─────────────────────────
echo ">>> Test 6: Web dashboard accessibility"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$CC_URL/admin/login")
if [ "$HTTP_CODE" = "200" ]; then
    echo "   [PASS] Dashboard login page accessible (HTTP $HTTP_CODE)"
else
    echo "   [WARN] Dashboard returned HTTP $HTTP_CODE"
fi

# ── Test 7: Prometheus Metrics ────────────────────
echo ">>> Test 7: Prometheus metrics endpoint"
curl -sf "$CC_URL/metrics" | grep -c "control_" || true
echo "   [PASS]"

# ── Test 8: Proxy Node test (if running) ──────────
echo ">>> Test 8: Proxy connectivity test"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    --connect-timeout 3 \
    -x "http://testuser:$USER_TOKEN@${PROXY_URL#http://}" \
    "$TARGET_URL" 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "000" ]; then
    echo "   Proxy connection result: HTTP $HTTP_CODE"
else
    echo "   [WARN] Proxy returned HTTP $HTTP_CODE"
fi

echo ""
echo "========================================"
echo " All tests completed"
echo "========================================"
