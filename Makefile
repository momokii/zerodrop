# Makefile for ZeroDrop Terminal v1.0
# Comprehensive build, test, and deployment automation

.PHONY: help
help:
	@echo "ZeroDrop Terminal v1.0 — Makefile Targets"
	@echo ""
	@echo "Setup:"
	@echo "  make install           - Install all dependencies (Go + Node)"
	@echo "  make install-deps       - Install Go and Node dependencies"
	@echo "  make install-go          - Install Go dependencies only"
	@echo "  make install-node        - Install Node dependencies only"
	@echo ""
	@echo "Development:"
	@echo "  make dev                - Start development server (Mock Printer)"
	@echo "  make dev-usb             - Start development server (USB Printer auto-detect)"
	@echo "  make dev-frontend       - Start frontend dev server (proxies to backend)"
	@echo "  make run                - Run production binary (serves frontend + backend)"
	@echo ""
	@echo "Build:"
	@echo "  make build              - Build backend binary"
	@echo "  make build-frontend       - Build frontend for production"
	@echo "  make build-all           - Build both backend and frontend"
	@echo ""
	@echo "Testing:"
	@echo "  make test               - Run all tests"
	@echo "  make test-race          - Run tests with race detection"
	@echo "  make test-coverage      - Run tests with coverage report"
	@echo "  make test-integration   - Run integration tests"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build        - Build Docker images"
	@echo "  make docker-up           - Start services (dev)"
	@echo "  make docker-up-prod      - Start services (production)"
	@echo "  make docker-down         - Stop services"
	@echo "  make docker-logs         - View service logs"
	@echo "  make docker-clean        - Remove Docker images and containers"
	@echo "  make docker-restart      - Restart services"
	@echo ""
	@echo "System:"
	@echo "  make health             - Check server health"
	@echo "  make check              - Run all checks (tests, formatting, vet)"
	@echo "  make check-deps         - Check if dependencies are satisfied"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make clean-all           - Clean everything (including downloads)"
	@echo ""
	@echo "Operations:"
	@echo "  make stop                - Stop running server"
	@echo "  make restart             - Restart server"
	@echo "  make status             - Show running status"
	@echo "  make logs                - Show server logs"
	@echo ""
	@echo "Utilities:"
	@echo "  make deps               - Show dependency tree"
	@echo "  make version            - Show version info"
	@echo "  make gen-key            - Generate new key pair (WARN: destroys old key)"
	@echo ""

# =============================================================================
# Variables
# =============================================================================

BINARY_NAME=zerodrop
BUILD_DIR=bin
FRONTEND_DIR=frontend
FRONTEND_DIST=$(FRONTEND_DIR)/dist
GO_CMD=go
GO_FLAGS=-v
NPM_CMD=npm
GO_EXAMPLE?=go run ./cmd/zerodrop
DOCKER_COMPOSE=docker-compose
DOCKER_COMPOSE_FILES=--file docker-compose.yml
DOCKER_PROD_FILES=--file docker-compose.yml --file docker-compose.prod.yml

# Environment variables (can be overridden)
PRINTER_TYPE?=mock
LOG_ENABLED?=false
PUBLIC_KEY_PATH?=./data/public_key.pem
PRINTER_DEVICE?=

# Build flags
LDFLAGS?=-ldflags="-w -s"
CGO_ENABLED?=0

# =============================================================================
# Setup Targets
# =============================================================================

.PHONY: install install-deps install-go install-node
install: install-deps
	@echo "→ Installing all dependencies..."

install-deps: install-go install-node
	@echo "✓ All dependencies installed"

install-go:
	@echo "→ Installing Go dependencies..."
	@$(GO_CMD) mod download
	@$(GO_CMD) mod tidy
	@echo "✓ Go dependencies installed"

install-node:
	@echo "→ Installing Node dependencies..."
	cd $(FRONTEND_DIR) && $(NPM_CMD) install
	@echo "✓ Node dependencies installed"

# =============================================================================
# Development Targets
# =============================================================================

.PHONY: dev dev-usb dev-frontend
dev: stop
	@echo "→ Starting development server (Mock Printer)..."
	PRINTER_TYPE=mock LOG_ENABLED=false $(GO_EXAMPLE) &
	@sleep 2
	@echo "✓ Development server running on http://localhost:8080"
	@echo "  Frontend: http://localhost:8080"
	@echo "  Press Ctrl+C to stop"

dev-usb: stop
	@echo "→ Starting development server (USB Printer, auto-detect)..."
	PRINTER_TYPE=usb PRINTER_DEVICE="" LOG_ENABLED=false $(GO_EXAMPLE) &
	@sleep 2
	@echo "✓ Development server running on http://localhost:8080"
	@echo "  USB Printer will be auto-detected"
	@echo "  Falls back to Mock Printer if no printer found"
	@echo "  Press Ctrl+C to stop"

dev-frontend:
	@echo "→ Starting frontend development server..."
	cd $(FRONTEND_DIR) && $(NPM_CMD) run dev

.PHONY: run
run: build-all stop
	@echo "→ Running production binary (serves frontend + backend)..."
	PRINTER_TYPE=$(PRINTER_TYPE) LOG_ENABLED=$(LOG_ENABLED) PUBLIC_KEY_PATH=$(PUBLIC_KEY_PATH) PRINTER_DEVICE=$(PRINTER_DEVICE) ./$(BUILD_DIR)/$(BINARY_NAME) &
	@sleep 2
	@echo "✓ Production server running on http://localhost:8080"
	@echo "  Press Ctrl+C to stop"

# =============================================================================
# Build Targets
# =============================================================================

.PHONY: build build-backend build-frontend build-all
build: build-backend build-frontend
	@echo "✓ Backend and frontend built"

build-backend:
	@echo "→ Building backend binary..."
	CGO_ENABLED=$(CGO_ENABLED) $(GO_CMD) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/zerodrop
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)
	@echo "✓ Backend built: $(BUILD_DIR)/$(BINARY_NAME)"

build-frontend:
	@echo "→ Building frontend for production..."
	cd $(FRONTEND_DIR) && $(NPM_CMD) run build
	@echo "✓ Frontend built in $(FRONTEND_DIST)"

build-all: clean
	@echo "→ Building all components..."
	$(MAKE) build-backend build-frontend
	@echo "✓ All components built"

# =============================================================================
# Testing Targets
# =============================================================================

.PHONY: test test-race test-coverage test-integration
test:
	@echo "→ Running tests..."
	$(GO_CMD) test ./... -v

test-race:
	@echo "→ Running tests with race detection..."
	$(GO_CMD) test ./... -race -v

test-coverage:
	@echo "→ Running tests with coverage..."
	$(GO_CMD) test ./... -cover -coverprofile=coverage.out -v
	@go tool cover -html -coverage.out
	@echo "✓ Coverage report: coverage/index.html"

test-integration:
	@echo "→ Starting server for integration testing..."
	PRINTER_TYPE=mock LOG_ENABLED=false timeout 5 ./$(BUILD_DIR)/$(BINARY_NAME) &
	@sleep 2
	@curl -s http://localhost:8080/health | grep -q "healthy" || (echo "✗ Health check failed" && exit 1)
	@curl -s http://localhost:8080/key | grep -q "PUBLIC KEY" || (echo "✗ Key endpoint failed" && exit 1)
	@pkill -f $(BINARY_NAME) 2>/dev/null || true
	@echo "✓ Integration tests passed"

# =============================================================================
# Docker Targets
# =============================================================================

.PHONY: docker-build docker-up docker-up-prod docker-down docker-logs docker-clean docker-restart
docker-build:
	@echo "→ Building Docker images..."
	$(DOCKER_COMPOSE) build

docker-up:
	@echo "→ Starting Docker services (development mode)..."
	$(DOCKER_COMPOSE) $(DOCKER_COMPOSE_FILES) up -d
	@echo "✓ Services running"
	@echo "  API: http://localhost:8080"
	@echo "  Frontend: http://localhost:8080"
	@echo "  Logs: make docker-logs"

docker-up-prod:
	@echo "→ Starting Docker services (production mode)..."
	$(DOCKER_COMPOSE) $(DOCKER_PROD_FILES) up -d
	@echo "✓ Production services running"
	@echo "  API: http://localhost:8080"
	@echo "  Logs: make docker-logs"

docker-down:
	@echo "→ Stopping Docker services..."
	$(DOCKER_COMPOSE) down
	@echo "✓ Services stopped"

docker-logs:
	@echo "→ Showing Docker service logs..."
	$(DOCKER_COMPOSE) logs -f

docker-clean:
	@echo "→ Cleaning Docker resources..."
	@$(DOCKER_COMPOSE) down -v
	docker system prune -f
	@echo "✓ Docker resources cleaned"

docker-restart: docker-down docker-up
	@echo "→ Restarting Docker services..."

# =============================================================================
# System Check Targets
# =============================================================================

.PHONY: health check check check-deps version status
health: stop
	@echo "→ Checking server health..."
	@curl -sf http://localhost:8080/health || (echo "✗ Server not running" && exit 1)
	@echo "✓ Server is healthy"

check: check-deps test
	@echo "→ Running all checks..."
	@$(MAKE) check-deps test

check-deps:
	@echo "→ Checking dependencies..."
	@which $(GO_CMD) > /dev/null || (echo "✗ Go not installed" && exit 1)
	@which $(NPM_CMD) > /dev/null || (echo "✗ npm not installed" && exit 1)
	@$(GO_CMD) version | grep -q "go version" || (echo "✗ Go version check failed" && exit 1)
	@$(NPM_CMD) --version > /dev/null || (echo "✗ npm version check failed" && exit 1)
	@echo "✓ All dependencies satisfied"

version:
	@echo "→ Version information:"
	@echo "  Go version: " $$(GO_CMD) version
	@echo "  npm version:" $$(NPM_CMD) --version
	@echo "  Node version:" $$(node --version 2>/dev/null || echo "  (not installed)")
	@echo "  Docker:" $$(docker version --format '{{.Version}}' 2>/dev/null || echo "  (not installed)")

status:
	@echo "→ Checking running status..."
	@pgrep -f $(BUILD_DIR)/$(BINARY_NAME) > /dev/null && \
		(echo "✓ Server is running (PID: $$(pgrep -f $(BUILD_DIR)/$(BINARY_NAME))") && \
		echo "  URL: http://localhost:8080") || \
	(echo "✗ Server is not running")
	@echo ""
	@pgrep -f "vite" > /dev/null && \
		(echo "✓ Frontend dev server is running (PID: $$(pgrep -f "vite")")" && \
		echo "  URL: http://localhost:3000") || \
	(echo "✓ Frontend dev server is not running")
	@echo ""
	@$(DOCKER_COMPOSE) ps 2>/dev/null | grep -q "Up" && \
		echo "✓ Docker services are running" || \
		echo "✓ Docker services are stopped"

# =============================================================================
# Operations Targets
# =============================================================================

.PHONY: stop restart clean logs clean-all
stop:
	@echo "→ Stopping server..."
	@-pkill -f $(BUILD_DIR)/$(BINARY_NAME) 2>/dev/null && echo "✓ Server stopped" || echo "✓ Server was not running"

restart: stop run
	@echo "→ Restarting server..."

logs:
	@echo "→ Server logs (from current session):"
	@echo "  (Run with 'tail -f zerodrop.log' for live logs in production)"
	@if [ -f zerodrop.log ]; then \
		tail -20 zerodrop.log; \
	else \
		echo "  No log file found"; \
	fi

clean:
	@echo "→ Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(FRONTEND_DIST)
	@echo "✓ Build artifacts cleaned"

clean-all: clean
	@echo "→ Cleaning everything..."
	rm -rf $(BUILD_DIR)
	rm -rf $(FRONTEND_DIST)
	rm -rf $(FRONTEND_DIR)/node_modules
	rm -f coverage.out
	rm -f zerodrop.log
	@echo "✓ All artifacts cleaned"

# =============================================================================
# Utility Targets
# =============================================================================

.PHONY: deps gen-key format lint vet deps-go deps-node
deps:
	@echo "→ Dependency tree:"
	@echo "  Go:"
	@$(GO_CMD) mod graph
	@echo ""
	@echo "  Node:"
	@cd $(FRONTEND_DIR) && $(NPM_CMD) list --depth=0

deps-go:
	@echo "→ Go dependency tree:"
	@$(GO_CMD) mod graph

deps-node:
	@echo "→ Node dependency tree:"
	@cd $(FRONTEND_DIR) && $(NPM_CMD) list --depth=0

gen-key:
	@echo "→ Generating new key pair..."
	@$(GO_CMD) run ./cmd/zerodrop
	@echo "⚠️  IMPORTANT: Scan the PRIVATE_KEY_QR above and save it securely!"
	@echo "   The private key will be destroyed from server memory after this session."

format:
	@echo "→ Formatting Go code..."
	@$(GO_CMD) fmt ./...

lint:
	@echo "→ Linting Go code..."
	@$(GO_CMD) vet ./...

vet:
	@echo "→ Running go vet..."
	@$(GO_CMD) vet ./...

# =============================================================================
# Production Deployment Targets
# =============================================================================

.PHONY: deploy deploy-dev deploy-prod
deploy: check build-all docker-up-prod
	@echo "→ Deploying to production..."
	@echo "  Building all components..."
	@$(MAKE) build-all
	@echo "  Starting services..."
	@$(DOCKER_COMPOSE) $(DOCKER_PROD_FILES) up -d
	@echo "✓ Deployed to production"
	@echo "  Access: http://localhost:8080"

deploy-dev: check docker-up
	@echo "→ Deploying to development..."
	@echo "  Starting services..."
	@$(DOCKER_COMPOSE) up -d
	@echo "✓ Deployed to development"
	@echo "  Access: http://localhost:8080"

# =============================================================================
# Database/Cleanup Targets
# =============================================================================

.PHONY: reset hard-reset
reset: clean-all
	@echo "→ Resetting repository..."
	@echo "  WARNING: This will remove all build artifacts and downloads"
	@$(MAKE) clean-all

hard-reset: reset
	@echo "→ Hard reset repository..."
	@echo "  WARNING: This will remove all artifacts, dependencies, and node_modules"
	@rm -rf $(FRONTEND_DIR)/node_modules
	@echo "  Hard reset complete"

# =============================================================================
# Production Operations
# =============================================================================

.PHONY: backup backup-key restore-key
backup:
	@echo "→ Backing up public key..."
	@if [ -f $(PUBLIC_KEY_PATH) ]; then \
		cp $(PUBLIC_KEY_PATH) $(PUBLIC_KEY_PATH).backup; \
		echo "✓ Public key backed up to $(PUBLIC_KEY_PATH).backup"; \
	else \
		echo "✗ Public key not found"; \
	fi

restore-key:
	@echo "→ Restoring public key from backup..."
	@if [ -f $(PUBLIC_KEY_PATH).backup ]; then \
		cp $(PUBLIC_KEY_PATH).backup $(PUBLIC_KEY_PATH); \
		echo "✓ Public key restored from backup"; \
	else \
		echo "✗ Backup not found"; \
	fi

# =============================================================================
# Monitoring Targets
# =============================================================================

.PHONY: monitor watch monitor-test
monitor:
	@echo "→ Monitoring server (check health every 5s)..."
	@while true; do \
		curl -sf http://localhost:8080/health > /dev/null 2>&1 && echo "✓ Healthy" || echo "✗ Unhealthy"; \
		sleep 5; \
	done

watch:
	@echo "→ Watching for file changes (requires entr)..."
	@echo "  Install: go install github.com/arkadi/v1 at latest"
	@echo "  Run: entr -r go run ./cmd/zerodrop ./..."

watch-test:
	@echo "→ Running tests on file changes (requires entr)..."
	@echo "  Install: go install github.com/arkadi/v1 at latest"
	@echo "  Run: entr -r go test ./..."

# =============================================================================
# CI/CD Targets
# =============================================================================

.PHONY: ci ci-test ci-build ci-deploy
ci: check test-race build-all
	@echo "→ Running CI pipeline..."
	@$(MAKE) check test-race build-all

ci-test: check test-race
	@echo "→ CI: Running tests with race detection..."
	@$(MAKE) check test-race

ci-build: build-all
	@echo "→ CI: Building all components..."
	@$(MAKE) build-all

ci-deploy: ci-test ci-build docker-up-prod
	@echo "→ CI: Deploying to production..."
	@$(DOCKER_COMPOSE) $(DOCKER_PROD_FILES) up -d

# =============================================================================
# Targets with special conditions
# =============================================================================

.PHONY: FORCE
FORCE:
	@echo "→ Force rebuild of all components..."
	@$(MAKE) clean-all build-all

# Don't create intermediate files from source files
.PHONY: $(BUILD_DIR)/$(BINARY_NAME)
$(BUILD_DIR)/$(BINARY_NAME): build-backend

# Keep frontend dist up-to-date
$(FRONTEND_DIST): build-frontend