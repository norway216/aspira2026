.PHONY: all build build-cc build-pn clean test lint run-cc run-pn docker-up docker-down

# Build both binaries
all: build

build: build-cc build-pn

build-cc:
	go build -o bin/control-center ./cmd/control-center

build-pn:
	go build -o bin/proxy-node ./cmd/proxy-node

# Run locally (with defaults)
run-cc:
	go run ./cmd/control-center

run-pn:
	go run ./cmd/proxy-node

# Run with example configuration
run-cc-dev:
	LISTEN_ADDR=:8080 DATABASE_DRIVER=sqlite DATABASE_DSN=traffic_lab.db go run ./cmd/control-center

run-pn-dev:
	NODE_ID=node-a LISTEN_ADDR=:10801 NODE_SECRET=node-secret-a METRICS_LISTEN_ADDR=:9200 go run ./cmd/proxy-node

# Run two proxy nodes for multi-node testing
run-pn-dev2:
	NODE_ID=node-b LISTEN_ADDR=:10802 NODE_SECRET=node-secret-b METRICS_LISTEN_ADDR=:9201 go run ./cmd/proxy-node

# Testing
test:
	go test -race -count=1 ./internal/...

test-cover:
	go test -race -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html

# Linting
lint:
	go vet ./...
	go mod tidy

# Clean
clean:
	rm -rf bin/
	rm -f traffic_lab.db
	rm -f coverage.out coverage.html

# Docker
docker-up:
	docker compose -f deploy/docker-compose.yml up --build -d

docker-down:
	docker compose -f deploy/docker-compose.yml down

docker-logs:
	docker compose -f deploy/docker-compose.yml logs -f

# Download deps
deps:
	go mod download
	go mod tidy

# Quick smoke test (requires control-center running on :8080)
smoke-test:
	@echo "=== Testing control center health ==="
	curl -s http://localhost:8080/api/v1/nodes | head -c 200
	@echo ""
	@echo "=== Testing Prometheus metrics ==="
	curl -s http://localhost:8080/metrics | head -n 20
