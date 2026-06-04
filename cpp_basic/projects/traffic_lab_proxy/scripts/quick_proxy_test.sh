#!/bin/bash
# Quick proxy forward test
set -e

cleanup() {
  kill $CC_PID $PN_PID $TG_PID 2>/dev/null || true
  wait $CC_PID 2>/dev/null || true
  wait $PN_PID 2>/dev/null || true
  wait $TG_PID 2>/dev/null || true
  rm -f /tmp/qpt_cc.db /tmp/qpt_page.txt
}
trap cleanup EXIT

echo "=== 1. Build ==="
go build -o /tmp/qpt-cc ./cmd/control-center 2>&1
go build -o /tmp/qpt-pn ./cmd/proxy-node 2>&1

echo "=== 2. Start services ==="
WEB_ROOT=web DATABASE_DSN=/tmp/qpt_cc.db LISTEN_ADDR=:51080 /tmp/qpt-cc &
CC_PID=$!
sleep 2

echo "hello_from_proxy_test" > /tmp/qpt_page.txt
python3 -c "import http.server; s=http.server.HTTPServer(('', 51999), http.server.SimpleHTTPRequestHandler); s.serve_forever()" &
TG_PID=$!
sleep 1

echo "=== 3. Create user ==="
USER_RESP=$(curl -sf -u admin:admin123 -X POST http://localhost:51080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"username":"pxy","password":"pass","max_rate_mbps":100,"max_connections":50}')
TOKEN=$(echo "$USER_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")
echo "Token: $TOKEN"

echo "=== 4. Register node ==="
curl -sf -u admin:admin123 -X POST http://localhost:51080/api/v1/nodes/register \
  -H "Content-Type: application/json" \
  -d '{"node_id":"n1","name":"N1","ip":"127.0.0.1","proxy_port":51901,"max_bandwidth_mbps":100,"max_connections":100,"node_secret":"s1"}' > /dev/null

echo "=== 5. Start proxy node ==="
NODE_ID=n1 LISTEN_ADDR=:51901 NODE_SECRET=s1 CONTROL_CENTER_URL=http://localhost:51080 \
  METRICS_LISTEN_ADDR=:51920 HEARTBEAT_INTERVAL=60s POLICY_FETCH_INTERVAL=60s \
  /tmp/qpt-pn &
PN_PID=$!
sleep 4

echo "=== 6. Test proxy (--proxytunnel) ==="
RESULT=$(curl -sf --proxytunnel --connect-timeout 5 --max-time 10 \
  -x "http://pxy:${TOKEN}@127.0.0.1:51901" \
  http://localhost:51999/tmp/qpt_page.txt 2>&1)
if echo "$RESULT" | grep -q "hello_from_proxy_test"; then
  echo "[PASS] Proxy forwarding works: $RESULT"
else
  echo "[FAIL] Got: $RESULT"
  echo "DEBUG: checking proxy metrics..."
  curl -sf http://localhost:51920/metrics 2>&1 | grep proxy_node
fi

echo ""
echo "=== Test complete ==="
