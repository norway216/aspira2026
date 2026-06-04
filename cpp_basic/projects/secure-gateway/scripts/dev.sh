#!/bin/bash
#
# Development startup script for Secure Gateway
# Starts Docker services and Web frontend
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "====================================="
echo "  Secure Gateway - Development Mode"
echo "====================================="
echo ""

# Check prerequisites
echo "[1/4] Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    echo "ERROR: docker is not installed"
    exit 1
fi

if ! docker compose version &> /dev/null; then
    echo "ERROR: docker compose is not available"
    exit 1
fi

if ! command -v node &> /dev/null; then
    echo "ERROR: node is not installed"
    exit 1
fi

echo "  ✓ docker $(docker --version | cut -d' ' -f3 | tr -d ',')"
echo "  ✓ docker compose $(docker compose version --short)"
echo "  ✓ node $(node --version)"

# Build and start backend services
echo ""
echo "[2/4] Starting backend services (PostgreSQL, Redis, API Gateway, Traffic Node)..."
cd "$PROJECT_DIR"

# Check if go.sum exists, if not generate it
if [ ! -f "go.sum" ]; then
    echo "  → Generating go.sum..."
    docker run --rm -v "$PROJECT_DIR":/app -w /app golang:1.22-alpine sh -c "apk add --no-cache git && go mod tidy" 2>&1 | tail -1
fi

docker compose up -d postgres redis
echo "  Waiting for databases..."
sleep 5

docker compose up -d api-gateway traffic-node
echo "  ✓ Backend services started"
echo "  ✓ API Gateway: http://localhost:8080"
echo "  ✓ Health Check: http://localhost:8080/health"

# Build and start web frontend
echo ""
echo "[3/4] Starting Web Frontend..."
cd "$PROJECT_DIR/web"

if [ ! -d "node_modules" ]; then
    echo "  → Installing npm dependencies..."
    npm install
fi

# Build for Nginx production (for docker compose)
npm run build 2>/dev/null || echo "  ⚠ Frontend build skipped (may need npm install)"

docker compose up -d web
echo "  ✓ Web: http://localhost:80"

echo ""
echo "[4/4] Verifying services..."
sleep 3

# Check service health
if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
    echo "  ✓ API Gateway is healthy"
else
    echo "  ⚠ API Gateway health check failed (may still be starting)"
fi

echo ""
echo "====================================="
echo "  System is ready!"
echo ""
echo "  Web UI:       http://localhost:80"
echo "  API:          http://localhost:8080"
echo "  DB (PG):      localhost:5432"
echo "  Redis:        localhost:6379"
echo "  Default Login: admin / admin123"
echo ""
echo "  To stop: docker compose down"
echo "  To view logs: docker compose logs -f"
echo "====================================="