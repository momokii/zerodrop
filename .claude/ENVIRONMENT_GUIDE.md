# Environment Guide — ZeroDrop Terminal

> Environment definitions and agent behavior for ZeroDrop Terminal v1.0/v1.1.

---

## Environment Definitions

| Environment | Purpose | Characteristics |
|---|---|---|
| `development` | Local development and feature work | Mock Printer, verbose logging, hot reload optional |
| `staging` | Pre-production validation | Real printer (if available), production-like config |
| `production` | Live system | USB Printer, minimal logging, hardened config |

**Active environment**: Not explicitly controlled — determined by `PRINTER_TYPE` and `LOG_ENABLED` variables.

---

## Agent Behavior by Environment

### Development (Local work)

- Verbose logging acceptable for debugging
- Mock Printer used by default (`PRINTER_TYPE=mock`)
- The agent may proceed with standard workflow without extra confirmation
- Hot reload available via air if configured: `air` (not required)

### Staging (Pre-production testing)

- The agent must **never** run destructive commands without confirmation
- Any proposed change must be presented as a written plan first
- Flag explicitly that operating in staging context
- Use real USB printer if available (`PRINTER_TYPE=usb`)

### Production (Live system)

- The agent must **never** directly modify production config or secrets
- The agent must **never** run destructive commands without explicit written confirmation
- Any proposed change must be presented as a written plan first
- Flag explicitly that operating in production context

---

## Environment Variables

### Required Variables

```bash
# Printer type (required)
PRINTER_TYPE=mock    # or "usb"

# USB device path (required if PRINTER_TYPE=usb)
PRINTER_DEVICE=/dev/usb/lp0
```

### Optional Variables

```bash
# Public key output path (default: ./data/public_key.pem)
PUBLIC_KEY_PATH=./data/public_key.pem

# Private key path (default: ./data/private_key.pem)
# Private key is saved on first run and reused across restarts.
PRIVATE_KEY_PATH=./data/private_key.pem

# Force key pair regeneration (default: false)
KEY_ROTATE=false

# Admin dashboard authentication token (required for /admin access)
# Set this in .env — never commit to version control.
ADMIN_TOKEN=your-secure-random-token-here

# Rate limiting (default: 5 per hour)
RATE_LIMIT_REQUESTS_PER_HOUR=5

# Structured logging (default: false)
LOG_ENABLED=false
```

---

## Environment Startup Commands

### Development

```bash
# 1. Set environment variables
export PRINTER_TYPE=mock

# 2. Run tests
go test ./... -v -race -cover

# 3. Build binary
go build -o bin/zerodrop ./cmd/zerodrop

# 4. Run application
./bin/zerodrop
# or with custom public key path:
PUBLIC_KEY_PATH=./test_key.pem ./bin/zerodrop

# 5. Test API endpoint
curl http://localhost:8080/key
curl -X POST http://localhost:8080/drop -H "Content-Type: application/json" -d '{"payload":"ZD1:test"}'
curl http://localhost:8080/health
```

### Production (via Docker)

```bash
# 1. Build image
docker build -t zerodrop:latest .

# 2. Run with USB printer
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# 3. View logs
docker-compose logs -f zerodrop

# 4. Health check
curl http://localhost:8080/health
```

---

## Test Commands

### Unit Tests

```bash
# Run all tests
go test ./... -v

# Run with race detection
go test ./... -race

# Run with coverage
go test ./... -cover

# Run specific package
go test ./pkg/crypto -v

# Run with coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests

```bash
# Run with Mock Printer
PRINTER_TYPE=mock go test ./...

# Run tests with verbose output
go test ./... -v -race -cover
```

### Linting

```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Run golangci-lint (if installed)
golangci-lint run

# Check for vulnerabilities
govulncheck ./...
```

---

## Docker Compose Environment Pattern

### Files

- **`docker-compose.yml`** — base service definitions
- **`docker-compose.override.yml`** — development overrides (hot reload, debug ports)
- **`docker-compose.prod.yml`** — production overrides (no volume mounts, restart policies)

### Commands

```bash
# Development (automatic — docker-compose.override.yml loaded by default)
docker-compose up

# Production (explicit — only base + prod override)
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# Staging (prod override with staging .env)
docker-compose -f docker-compose.yml -f docker-compose.prod.yml --env-file .env.staging up -d

# Stop
docker-compose down

# View logs
docker-compose logs -f
```

---

## `.env` File Pattern

```
.env.example        # Committed — all keys with placeholders
.env                # Never committed — development values
.env.staging        # Never committed — staging secrets
.env.production     # Never committed — production secrets
```

### `.env.example` Requirements

```bash
# Printer type: mock or usb
PRINTER_TYPE=mock

# USB device path (required if PRINTER_TYPE=usb)
# PRINTER_DEVICE=/dev/usb/lp0

# Public key output path (default: ./data/public_key.pem)
# PUBLIC_KEY_PATH=./data/public_key.pem

# Private key path (default: ./data/private_key.pem)
# PRIVATE_KEY_PATH=./data/private_key.pem

# Force key pair regeneration on next startup (default: false)
# KEY_ROTATE=false

# Admin dashboard authentication token (required for /admin access)
# ADMIN_TOKEN=change-me-to-a-secure-random-string

# Rate limiting: requests per IP per hour (default: 5)
# RATE_LIMIT_REQUESTS_PER_HOUR=5

# Structured logging: true or false (default: false)
# LOG_ENABLED=false
```

---

## Pre-Commit Environment Checks

Before the first commit of any session, verify:

1. [ ] `.env` is listed in `.gitignore`
2. [ ] `.env.example` exists and is up to date
3. [ ] No real secrets are present in any file about to be committed
4. [ ] `go test ./...` passes
5. [ ] `go fmt ./...` has been run
6. [ ] `go vet ./...` passes
7. [ ] `govulncheck ./...` passes (no vulnerabilities)
