# Makefile for ZeroDrop Terminal v1.0
# Docker-focused build, test, and deployment automation

.PHONY: help
help:
	@echo "ZeroDrop Terminal v1.0 — Makefile Targets"
	@echo ""
	@echo "DOCKER (primary workflow):"
	@echo "  make docker-build        - Build Docker images"
	@echo "  make docker-up           - Start services (auto-detects usb/mock from .env)"
	@echo "  make docker-up-rebuild   - Rebuild image + start services (same as: make docker-up REBUILD=true)"
	@echo "  make docker-up-prod      - Start services (production mode)"
	@echo "  make docker-up-prod-rebuild - Rebuild + start production (USB printer)"
	@echo "  make docker-down         - Stop services"
	@echo "  make docker-logs         - View service logs"
	@echo "  make docker-restart      - Restart services (no rebuild)"
	@echo "  make docker-restart-rebuild - Restart services with rebuild"
	@echo "  make docker-clean        - Remove all Docker resources"
	@echo ""
	@echo "READER (standalone offline decryptor):"
	@echo "  make serve-reader        - Serve reader.html locally (port 8081)"
	@echo "  make docker-reader-build - Build standalone reader Docker image"
	@echo "  make docker-reader-up    - Start reader container (port 8081)"
	@echo "  make docker-reader-down  - Stop reader container"
	@echo ""
	@echo "DEVELOPMENT (local):"
	@echo "  make dev                 - Start dev server (Mock Printer)"
	@echo "  make dev-usb             - Start dev server (USB Printer)"
	@echo "  make dev-frontend        - Start frontend dev server"
	@echo ""
	@echo "BUILD:"
	@echo "  make build               - Build backend binary"
	@echo "  make build-frontend      - Build frontend for production"
	@echo "  make build-all           - Build backend + frontend"
	@echo ""
	@echo "TESTING:"
	@echo "  make test                - Run all tests"
	@echo "  make test-race           - Run tests with race detection"
	@echo "  make test-coverage       - Run tests with coverage report"
	@echo "  make test-integration    - Run integration tests"
	@echo "  make check-security      - Run full security verification suite (Go + code scan)"
	@echo ""
	@echo "QUALITY:"
	@echo "  make check               - Run all checks (deps + tests)"
	@echo "  make check-deps          - Verify Go and Node are installed"
	@echo "  make format              - Run go fmt"
	@echo "  make vet                 - Run go vet"
	@echo ""
	@echo "OPERATIONS:"
	@echo "  make stop                - Stop running server"
	@echo "  make restart             - Restart server"
	@echo "  make clean               - Clean build artifacts"
	@echo "  make status              - Show running status"
	@echo "  make logs                - Show server logs"
	@echo "  make health              - Check server health"
	@echo ""
	@echo "DEPLOYMENT (Docker):"
	@echo "  make deploy              - Build Docker image + start production"
	@echo "  make deploy-dev          - Build Docker image + start development"
	@echo ""
	@echo "UTILITIES:"
	@echo "  make deps                - Show dependency tree"
	@echo "  make version             - Show version info"
	@echo "  make monitor             - Health check every 5 seconds"
	@echo "  make gen-key             - Generate new key pair"
	@echo ""

# =============================================================================
# Variables
# =============================================================================

BINARY_NAME=zerodrop
BUILD_DIR=bin
FRONTEND_DIR=frontend
FRONTEND_DIST=$(FRONTEND_DIR)/dist
GO_CMD=go
DOCKER_COMPOSE=docker compose

# Auto-detect configuration from .env (for compose file selection)
-include .env

# Environment (can be overridden via command line; defaults if .env absent)
PRINTER_TYPE?=mock
LOG_ENABLED?=false
PUBLIC_KEY_PATH?=./data/public_key.pem

# Automatically include production compose override when PRINTER_TYPE=usb
ifeq ($(PRINTER_TYPE),usb)
  COMPOSE_OVERRIDE = -f docker-compose.prod.yml
else
  COMPOSE_OVERRIDE =
endif

# =============================================================================
# Docker Targets (primary workflow)
# =============================================================================

.PHONY: docker-build docker-up docker-up-rebuild docker-up-prod docker-up-prod-rebuild docker-down docker-logs docker-restart docker-restart-rebuild docker-clean

docker-build:
	@echo "→ Building Docker image..."
	$(DOCKER_COMPOSE) build

# Usage: make docker-up          (quick start, no rebuild)
#        make docker-up-rebuild  (rebuild image then start)
#        make docker-up REBUILD=true  (same as above)
docker-up:
	@echo "→ Starting services ($(PRINTER_TYPE) mode)..."
	$(DOCKER_COMPOSE) -f docker-compose.yml $(COMPOSE_OVERRIDE) up -d $(if $(filter true,$(REBUILD)),--build,)
	@echo "✓ Services running at http://localhost:8080 ($(PRINTER_TYPE) mode)"
	@echo "  Logs: make docker-logs"

docker-up-rebuild:
	$(MAKE) docker-up REBUILD=true

docker-up-prod:
	@echo "→ Starting services (production)..."
	$(DOCKER_COMPOSE) -f docker-compose.yml -f docker-compose.prod.yml up -d
	@echo "✓ Production services running at http://localhost:8080"

docker-up-prod-rebuild:
	@echo "→ Rebuilding and starting services (production)..."
	$(DOCKER_COMPOSE) -f docker-compose.yml -f docker-compose.prod.yml up -d --build
	@echo "✓ Production services running at http://localhost:8080"

docker-down:
	@echo "→ Stopping services..."
	$(DOCKER_COMPOSE) down
	@echo "✓ Services stopped"

docker-logs:
	@echo "→ Service logs (Ctrl+C to stop)..."
	$(DOCKER_COMPOSE) logs -f

docker-restart: docker-down docker-up
	@echo "✓ Services restarted"

docker-restart-rebuild: docker-down docker-up-rebuild
	@echo "✓ Services restarted (with rebuild)"

docker-clean:
	@echo "→ Cleaning Docker resources..."
	@$(DOCKER_COMPOSE) down -v 2>/dev/null || true
	docker system prune -f
	@echo "✓ Docker resources cleaned"

# =============================================================================
# Reader Targets (standalone offline decryptor)
# =============================================================================

.PHONY: serve-reader docker-reader-build docker-reader-up docker-reader-down

serve-reader:
	@echo "→ Starting ZeroDrop Reader on http://localhost:8081..."
	@echo "  Open http://localhost:8081/reader.html in your browser"
	@echo "  Press Ctrl+C to stop"
	python3 -m http.server 8081 --directory .

docker-reader-build:
	@echo "→ Building standalone reader Docker image..."
	docker build -t zerodrop-reader:latest -f Dockerfile.reader .
	@echo "✓ Reader image built: zerodrop-reader:latest"

docker-reader-up:
	@echo "→ Starting reader container on http://localhost:8081..."
	$(DOCKER_COMPOSE) -f docker-compose.reader.yml up -d
	@echo "✓ Reader running at http://localhost:8081/reader.html"

docker-reader-down:
	@echo "→ Stopping reader container..."
	$(DOCKER_COMPOSE) -f docker-compose.reader.yml down
	@echo "✓ Reader stopped"

# =============================================================================
# Development Targets (local binary)
# =============================================================================

.PHONY: dev dev-usb dev-frontend

dev: stop
	@echo "→ Starting development server (Mock Printer)..."
	@echo "  Press Ctrl+C to stop"
	@if [ ! -d "$(FRONTEND_DIST)" ]; then \
		echo "  ⚠️  frontend/dist/ not found — run 'make build-frontend' first for the UI"; \
		echo "  API endpoints will still work (http://localhost:8080/health, /key)"; \
	fi
	PRINTER_TYPE=mock LOG_ENABLED=false go run ./cmd/zerodrop

dev-usb: stop
	@echo "→ Starting development server (USB Printer)..."
	@echo "  Press Ctrl+C to stop"
	PRINTER_TYPE=usb LOG_ENABLED=false go run ./cmd/zerodrop

dev-hot:
	@echo "→ Starting development server with hot reload (air)..."
	@if ! command -v air > /dev/null 2>&1; then \
		echo "  ⚠️  air not installed. Install it: go install github.com/air-verse/air@latest"; \
		echo "  Falling back to regular dev server..."; \
		$(MAKE) dev; \
		exit 0; \
	fi
	@if [ ! -d "$(FRONTEND_DIST)" ]; then \
		echo "  ⚠️  frontend/dist/ not found — run 'make build-frontend' first for the UI"; \
	fi
	PRINTER_TYPE=mock LOG_ENABLED=false air

dev-frontend:
	@echo "→ Starting frontend dev server..."
	cd $(FRONTEND_DIR) && npm run dev

# =============================================================================
# Build Targets
# =============================================================================

.PHONY: build build-backend build-frontend build-all

build-backend:
	@echo "→ Building backend binary..."
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/zerodrop
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)

build-frontend:
	@echo "→ Building frontend for production..."
	cd $(FRONTEND_DIR) && npm run build

build: build-backend build-frontend
	@echo "✓ Backend and frontend built"

build-all: clean
	$(MAKE) build-backend build-frontend
	@echo "✓ All components built"

# =============================================================================
# Testing Targets
# =============================================================================

.PHONY: test test-race test-coverage test-integration check-security

test:
	@echo "→ Running tests..."
	$(GO_CMD) test ./... -v

test-race:
	@echo "→ Running tests with race detection..."
	$(GO_CMD) test ./... -race -v

test-coverage:
	@echo "→ Running tests with coverage..."
	$(GO_CMD) test ./... -cover -coverprofile=coverage.out -v
	@go tool cover -html=coverage.out
	@echo "✓ Coverage report: coverage/index.html"

test-integration:
	@echo "→ Starting server for integration testing..."
	PRINTER_TYPE=mock LOG_ENABLED=false timeout 5 ./$(BUILD_DIR)/$(BINARY_NAME) &
	@sleep 2
	@curl -s http://localhost:8080/health | grep -q "healthy" || (echo "✗ Health check failed" && exit 1)
	@curl -s http://localhost:8080/key | grep -q "PUBLIC KEY" || (echo "✗ Key endpoint failed" && exit 1)
	@pkill -f $(BINARY_NAME) 2>/dev/null || true
	@echo "✓ Integration tests passed"

check-security:
	@echo "=== ZeroDrop Security Verification ==="
	@echo ""
	@echo "--- Go Security Tests ---"
	@$(GO_CMD) test -v -run "TestSecurity_" -count=1 ./... 2>&1 | tee /tmp/zerodrop-security-go.log || true
	@echo ""
	@echo "--- Code Scanning Checks ---"
	@bash scripts/security-scan.sh 2>&1 | tee /tmp/zerodrop-security-scan.log || true
	@echo ""
	@bash scripts/security-report.sh /tmp/zerodrop-security-go.log /tmp/zerodrop-security-scan.log

# =============================================================================
# Quality Targets
# =============================================================================

.PHONY: check check-deps format vet

check: check-deps test
	$(MAKE) test

check-deps:
	@echo "→ Checking dependencies..."
	@which $(GO_CMD) > /dev/null || (echo "✗ Go not installed" && exit 1)
	@which npm > /dev/null || (echo "✗ npm not installed" && exit 1)
	@docker compose version > /dev/null 2>&1 || (echo "✗ docker compose not installed (need Docker with compose plugin)" && exit 1)
	@echo "✓ All dependencies satisfied"

format:
	@echo "→ Formatting Go code..."
	$(GO_CMD) fmt ./...

vet:
	@echo "→ Running go vet..."
	$(GO_CMD) vet ./...

# =============================================================================
# Deployment Targets (Docker-only)
# =============================================================================

.PHONY: deploy deploy-dev

deploy: docker-build docker-up-prod
	@echo "✓ Deployed to production"
	@echo "  Access: http://localhost:8080"

deploy-dev: docker-build docker-up
	@echo "✓ Deployed to development"
	@echo "  Access: http://localhost:8080"

# =============================================================================
# Operations Targets
# =============================================================================

.PHONY: stop restart clean status logs health

stop:
	@echo "→ Stopping server..."
	@-pkill -f $(BUILD_DIR)/$(BINARY_NAME) 2>/dev/null || true
	@-pkill -f "go run ./cmd/$(BINARY_NAME)" 2>/dev/null || true
	@-pkill -f "go-build.*$(BINARY_NAME)" 2>/dev/null || true
	@sleep 1
	@printf "  Server: "; pgrep -f "$(BINARY_NAME)" > /dev/null 2>&1 && echo "running (try: kill -9)" || echo "stopped"

restart: stop build-all
	@echo "→ Restarting..."

clean:
	@echo "→ Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(FRONTEND_DIST)
	@echo "✓ Build artifacts cleaned"

status:
	@echo "→ Running status:"
	@printf "  Binary: "; pgrep -f $(BUILD_DIR)/$(BINARY_NAME) > /dev/null && echo "running" || echo "stopped"
	@printf "  Docker: "; docker ps --format '{{.Names}}' 2>/dev/null | grep -q zerodrop && echo "running" || echo "stopped"

logs:
	@echo "→ Server logs (last 20 lines):"
	@if [ -f zerodrop.log ]; then \
		tail -20 zerodrop.log; \
	else \
		echo "  No log file found"; \
	fi

health:
	@echo "→ Health check..."
	@curl -sf http://localhost:8080/health && echo "✓ Server is healthy" || (echo "✗ Server not running" && exit 1)

# =============================================================================
# Utility Targets
# =============================================================================

.PHONY: deps version monitor gen-key

deps:
	@echo "→ Dependency tree:"
	@echo "  Go:"; $(GO_CMD) mod graph
	@echo "  Node:"; cd $(FRONTEND_DIR) && npm list --depth=0 2>/dev/null

version:
	@echo "→ Version information:"
	@echo "  Go: $$(go version 2>/dev/null)"
	@echo "  Node: $$(node --version 2>/dev/null || echo 'not installed')"

monitor:
	@echo "→ Monitoring server (check health every 5s)..."
	@while true; do \
		curl -sf http://localhost:8080/health > /dev/null 2>&1 && echo "$$(date +%H:%M:%S) ✓ Healthy" || echo "$$(date +%H:%M:%S) ✗ Unhealthy"; \
		sleep 5; \
	done

gen-key:
	@echo "→ Generating new key pair..."
	go run ./cmd/zerodrop
	@echo "⚠️  IMPORTANT: Scan the PRIVATE_KEY_QR above and save it securely!"
	@echo "   The private key will be destroyed from server memory after this session."
