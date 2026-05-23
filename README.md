# ZeroDrop Terminal

> **Air-gapped, zero-knowledge secure credential delivery terminal.**

Encrypt sensitive data in your browser using Web Crypto API, transmit it to a server that **cannot decrypt it**, and receive the ciphertext as a physical QR code printout via 58mm thermal printer. Recipients decrypt the printout offline using a standalone HTML file — no network required.

---

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [How It Works](#how-it-works)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Quick Start](#quick-start)
  - [Prerequisites](#prerequisites)
  - [Development (Backend)](#development-backend)
  - [Development (Frontend)](#development-frontend)
  - [Production (Docker)](#production-docker)
- [API Reference](#api-reference)
- [Configuration](#configuration)
- [Security Model](#security-model)
- [USB Printer Support](#usb-printer-support)
- [Testing](#testing)
- [Deployment](#deployment)
  - [Docker Deployment](#docker-deployment)
  - [Binary Deployment](#binary-deployment)
  - [System Setup](#system-setup)
- [Offline Decryption](#offline-decryption)
- [Development](#development)
  - [Makefile Targets](#makefile-targets)
  - [CI/CD Pipeline](#cicd-pipeline)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)

---

## Features

- **Zero-Knowledge Encryption** — Messages are encrypted in the browser before transmission. The server never possesses plaintext or the private key.
- **Physical QR Code Output** — Encrypted payloads are printed as scannable QR codes on 58mm thermal paper via ESC/POS protocol.
- **Ephemeral Processing** — No database. Data exists in RAM only during the print job, then is securely zeroed.
- **Burn Protocol** — The private key is generated at startup, logged for the operator as a scannable QR code, then destroyed from server memory.
- **Offline Decryption** — Recipients use `static/reader.html` with jsQR camera scanning and real X25519 ECDH + AES-256-GCM decryption — no external dependencies, no network calls.
- **Asynchronous Print Spooler** — Buffered Go channel worker pool with retry logic (3 attempts, exponential backoff) and graceful shutdown draining.
- **Hardware Abstraction** — Supports Mock Printer (stdout logging) and USB Printer (auto-detection of 10+ models with graceful fallback).
- **Production-Grade Infrastructure** — Docker Compose deployment with Traefik reverse proxy, rate limiting (5 req/hr/IP), TLS termination, and health checks.
- **Dark Mode UI** — Modern React frontend with shadcn/ui components and Tailwind CSS, supporting both light and dark themes.

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    SUBMITTER                         │
│  ┌─────────────────────────────────────────────┐    │
│  │         Browser (Web Crypto API)             │    │
│  │                                               │    │
│  │  1. Fetch server public key (GET /key)        │    │
│  │  2. Encrypt plaintext with X25519 ECDH        │    │
│  │  3. Submit encrypted payload (POST /drop)     │    │
│  └──────────┬────────────────────────────────────┘    │
└─────────────┼────────────────────────────────────────┘
              │ Encrypted ciphertext (server cannot read)
              ▼
┌──────────────────────────────────────────────────────┐
│  Go Server (port 8080)                                │
│                                                        │
│  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────────┐ │
│  │  API   │  │Spooler │  │Printer │  │Observability│ │
│  │ /key   │─▶│ channel│─▶│  impl  │  │  Logger     │ │
│  │ /drop  │  │  queue │  │(mock  │  │  Shutdown   │ │
│  │ /health│  │        │  │ /usb)  │  │  Handler    │ │
│  └────────┘  └────────┘  └────────┘  └────────────┘ │
│                                                        │
│  ┌──────────────────────────────────────────────────┐ │
│  │  Crypto (server-side)                             │ │
│  │  • Generate X25519 key pair on startup            │ │
│  │  • Log private key as QR to stdout (operator)     │ │
│  │  • Burn Protocol: zero private key from memory    │ │
│  │  • SHA-256 public key fingerprint for verification│ │
│  └──────────────────────────────────────────────────┘ │
└──────────────────────┬───────────────────────────────┘
                       │ ESC/POS commands
                       ▼
┌──────────────────────────────────────────────────────┐
│  58mm Thermal Printer (USB)                           │
│  • QR code printout on thermal paper                  │
│  • Auto paper feed and cut                            │
└──────────────────────────────────────────────────────┘
                       │
                       ▼ (physical handoff)
┌──────────────────────────────────────────────────────┐
│  RECIPIENT (offline)                                  │
│  ┌────────────────────────────────────────────────┐ │
│  │  static/reader.html                             │ │
│  │  • Scan QR code via device camera               │ │
│  │  • Paste private key (PEM format)               │ │
│  │  • Decrypt offline using Web Crypto API          │ │
│  │  • No network required                           │ │
│  └────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────┘
```

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **No database** | Eliminates persistent attack surface. Data exists only during the print job. |
| **Burn Protocol** | Server generates a key pair, logs the private key for the operator, then destroys it. Uses `runtime.KeepAlive()` to prevent compiler optimization of the zeroing. |
| **Ephemeral key per restart** | Each server restart generates a fresh key pair. Old ciphertexts can only be decrypted by the corresponding private key. |
| **ZD1: payload prefix** | Forward-compatible version header for the QR payload format. |
| **Dual API paths** | Both `/api/*` and legacy `/*` routes are supported for backward compatibility. |
| **Buffered spooler channel** | Non-blocking job submission with 10-job buffer. Returns 503 if spooler is full. |
| **ECIES protocol** | X25519 ECDH + AES-256-GCM across all layers. Payload: `ZD1:base64(ephPubKey+iv+ciphertextWithTag)`. Ephemeral public key enables stateless offline decryption. |

---

## How It Works

1. **Operator starts the server** — On first boot, an X25519 key pair is generated. The public key is saved to disk; the private key is logged as a QR code to stdout, then destroyed from memory via the Burn Protocol.
2. **Submitter opens the web interface** — The React SPA fetches the server's public key (`GET /key`) and displays its SHA-256 fingerprint for verification.
3. **Submitter encrypts a message** — The plaintext is encrypted in-browser using Web Crypto API (X25519 ECDH) and prefixed with `ZD1:` for versioning.
4. **Submitter sends the payload** — The encrypted ciphertext is posted to the server (`POST /drop`). The server cannot decrypt it — it only receives the encrypted blob.
5. **Server queues the print job** — The payload enters a buffered Go channel. A worker picks it up, sends ESC/POS commands to the thermal printer, and prints the QR code.
6. **Memory is zeroed** — After printing, the payload buffer is securely overwritten in memory.
7. **Recipient scans and decrypts** — Using `static/reader.html` (fully offline), the recipient scans the QR code with their camera, pastes their private key, and decrypts the message.

---

## Tech Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| **Backend** | Go 1.26+ | HTTP server, crypto, printer control, job spooler |
| **Crypto** | Curve25519 (X25519 ECDH) + AES-256-GCM | ECIES encryption chain (Go `crypto/ecdh` + Web Crypto API) |
| **HTTP Router** | gorilla/mux | Route handling and SPA fallback |
| **QR Code** | go-qrcode (skip2) | QR code generation in Go |
| **Frontend** | React 18 + TypeScript | Web-based submission portal |
| **Build Tool** | Vite 5 | Frontend bundling and dev server |
| **UI Library** | shadcn/ui + Tailwind CSS 3 | Component system with dark mode |
| **Styling** | PostCSS + Autoprefixer | CSS processing |
| **Containers** | Docker + Docker Compose | Deployment and infrastructure |
| **Reverse Proxy** | Traefik v2.10 | TLS termination, rate limiting, health checks |
| **Printer Protocol** | ESC/POS | Thermal printer communication |
| **QR Decoding (Offline)** | jsQR v1.4.0 | Local QR scanning in reader.html (no network) |
| **Testing** | Go testing framework | Unit and integration tests |

---

## Project Structure

```
zerodrop/
├── cmd/
│   └── zerodrop/
│       └── main.go              # Application entry point, wiring, graceful shutdown
├── pkg/
│   ├── api/
│   │   └── server.go            # HTTP server, route handlers, SPA serving, health 503
│   ├── config/
│   │   ├── config.go            # Environment variable loading and validation
│   │   └── config_test.go       # Config tests (9 test cases)
│   ├── crypto/
│   │   ├── crypto.go            # X25519 key generation, PEM, Burn Protocol, fingerprinting
│   │   └── crypto_test.go       # Crypto tests (6 test cases)
│   ├── observability/
│   │   └── observability.go     # Structured JSON logger, shutdown handler
│   ├── printer/
│   │   ├── printer.go           # Printer interface, HealthChecker, Reconnector
│   │   ├── mock.go              # MockPrinter (QR ESC/POS + hex preview for testing/CI)
│   │   ├── usb.go               # USBPrinter with auto-detection of 10+ printer models
│   │   ├── usb_test.go          # USB printer tests
│   │   └── printer_test.go      # Printer interface tests
│   ├── qr/
│   │   └── qr.go                # QR code generation + ESC/POS GS v 0 rasterization
│   └── spooler/
│       └── spooler.go           # Buffered channel worker pool, retry logic, memory zeroing
├── frontend/
│   ├── src/
│   │   ├── main.tsx             # React entry point
│   │   ├── App.tsx              # Main application (encrypt → submit → success flow)
│   │   ├── index.css            # Tailwind CSS with shadcn/ui theme variables
│   │   ├── vite-env.d.ts        # Environment type declarations
│   │   ├── lib/
│   │   │   ├── api.ts           # API client (fetchPublicKey, submitPayload, checkHealth)
│   │   │   ├── crypto.ts        # Web Crypto API: ECDH key generation, encryption, fingerprint
│   │   │   └── utils.ts         # Utilities: cn(), clipboard, file download, formatting
│   │   └── components/
│   │       └── ui/
│   │           ├── alert.tsx    # shadcn/ui Alert component
│   │           ├── button.tsx   # shadcn/ui Button component
│   │           ├── card.tsx     # shadcn/ui Card component
│   │           ├── input.tsx    # shadcn/ui Input component
│   │           ├── label.tsx    # shadcn/ui Label component
│   │           └── textarea.tsx # shadcn/ui Textarea component
│   ├── dist/                    # Production frontend build output
│   ├── index.html               # HTML entry point
│   ├── vite.config.ts           # Vite configuration with API proxy
│   ├── tailwind.config.js       # Tailwind CSS configuration
│   ├── tsconfig.json            # TypeScript configuration
│   ├── postcss.config.js        # PostCSS configuration
│   └── package.json             # Node dependencies and scripts
├── static/
│   ├── reader.html              # Offline QR code decryption utility (works offline, no CDN)
│   └── jsqr.min.js              # jsQR v1.4.0 — local QR decoding for offline reader
├── infrastructure/
│   └── traefik/
│       └── traefik.yml          # Traefik static configuration (HTTPS, rate limiting)
├── docs/
│   ├── OVERVIEW.md              # Stakeholder-friendly project overview
│   └── prd/
│       └── PRD-001-zerodrop-terminal-v1.0.md  # Product Requirements Document
├── data/
│   └── public_key.pem           # Generated public key (created at runtime)
├── .claude/                     # Agent infrastructure and state tracking
│   ├── README.md                # Agent orientation
│   ├── AGENT_RULES.md           # Non-negotiable behavioral rules
│   ├── HOW_TO_RESUME.md         # Session startup protocol
│   ├── CODING_STANDARDS.md      # Go coding conventions
│   ├── SECURITY_STANDARDS.md    # Security guidelines
│   ├── ENVIRONMENT_GUIDE.md     # Environment definitions and commands
│   ├── state/
│   │   ├── CURRENT_STATUS.md    # Current project status
│   │   ├── TASK_QUEUE.md        # Implementation backlog
│   │   └── DECISIONS_LOG.md     # Architecture decision records
│   └── templates/               # Implementation checklists
├── Dockerfile                   # Multi-stage Alpine Docker build
├── docker-compose.yml           # Base Docker Compose configuration
├── docker-compose.prod.yml      # Production Docker Compose overrides
├── docker-compose.traefik.yml   # Traefik Docker Compose integration
├── docker-compose.override.yml  # Development Docker Compose overrides
├── Makefile                     # Development automation (build, test, deploy, ops)
├── Makefile.deploy              # Production deployment Makefile
├── go.mod                       # Go module definition
├── go.sum                       # Go dependency checksums
├── .env.example                 # Environment variable template
├── .gitignore                   # Git ignore rules
└── CLAUDE.md                    # Project-level Claude agent instructions
```

---

## Quick Start

### Prerequisites

| Dependency | Version | Purpose |
|-----------|---------|---------|
| Go | 1.26+ | Backend compilation and runtime |
| Node.js | 20+ | Frontend development |
| Docker | 20.10+ | Containerized deployment |
| Docker Compose | 2.0+ | Multi-service orchestration |
| 58mm thermal printer | USB | Physical print output (optional for dev) |

### Development (Backend)

```bash
# Run all tests with race detection and coverage
go test ./... -v -race -cover

# Build the binary
go build -o bin/zerodrop ./cmd/zerodrop

# Run with Mock Printer (logs to stdout, no hardware needed)
PRINTER_TYPE=mock ./bin/zerodrop

# Run with USB Printer (auto-detect)
PRINTER_TYPE=usb PRINTER_DEVICE="" ./bin/zerodrop

# Run with USB Printer (specific device)
PRINTER_TYPE=usb PRINTER_DEVICE=/dev/usb/lp0 ./bin/zerodrop
```

The backend serves:
- The API at `http://localhost:8080` (and `/api/*` prefix)
- The frontend SPA at `http://localhost:8080` (auto-serves `frontend/dist/`)
- The `static/reader.html` at `http://localhost:8080/reader.html`

### Development (Frontend)

```bash
cd frontend

# Install dependencies
npm install

# Run dev server (proxies API calls to backend on :8080)
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Type checking (no emit)
npm run type-check

# Lint
npm run lint
```

The frontend dev server runs on `http://localhost:3000` and proxies `/key`, `/drop`, and `/health` to `http://localhost:8080`.

### Production (Docker)

```bash
# Build and start all services (development mode)
docker-compose up -d

# Build and start all services (production mode with Traefik)
docker-compose -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.traefik.yml up -d

# View logs
docker-compose logs -f zerodrop

# Stop services
docker-compose down

# Using Makefile
make docker-up          # Development mode
make docker-up-prod     # Production mode with Traefik
make docker-logs        # View logs
make docker-down        # Stop services
make docker-restart     # Restart services
```

---

## API Reference

All API endpoints are available under both `/api/*` and legacy `/*` paths.

### `GET /key`

Retrieves the server's Curve25519 public key in PEM format.

**Response:**
```
Content-Type: text/plain

-----BEGIN PUBLIC KEY-----
<base64-encoded key bytes>
-----END PUBLIC KEY-----
```

**Status codes:**
- `200` — Public key returned successfully
- `500` — Failed to read public key file

---

### `POST /drop`

Submits an encrypted payload for printing as a QR code.

**Request body:**
```json
{
  "payload": "ZD1:<base64_encoded_ciphertext>"
}
```

**Validation rules:**
- Maximum payload length: 400 characters (including `ZD1:` prefix)
- Must start with `ZD1:` version header (for forward compatibility)
- Remainder must be valid base64 encoding

**Response:**
```json
{
  "status": "queued",
  "message": "Payload queued for printing"
}
```

**Status codes:**
- `202` — Payload accepted and queued for printing
- `400` — Invalid payload (bad JSON, missing `ZD1:` prefix, invalid base64, exceeds length limit)
- `503` — Server busy (spooler queue full)

---

### `GET /health`

Returns the server health status, including printer information.

**Response:**
```json
{
  "status": "healthy",
  "service": "zerodrop-terminal",
  "printer": {
    "type": "mock",
    "available": true,
    "status": "healthy"
  }
}
```

The `printer` object varies by printer type:
- **MockPrinter**: `type: "mock"`, `available`, `status`
- **USBPrinter**: `type: "usb"`, `available`, `device_path`, `model`, `mode`

**Status codes:**
- `200` — Server is healthy and printer is available
- `503` — Server is running but printer is unavailable (payloads will be rejected)

---

## Configuration

All configuration is via environment variables.

| Variable | Required | Default | Valid Values | Description |
|----------|----------|---------|-------------|-------------|
| `PRINTER_TYPE` | **Yes** | — | `mock`, `usb`, `tcp` | Printer implementation |
| `PRINTER_DEVICE` | No* | `""` (auto-detect) | Device path or empty | USB printer device path (empty = auto-detect from `/dev/usb/lp*`, `/dev/lp*`, `/dev/ttyUSB*`) |
| `PUBLIC_KEY_PATH` | No | `./data/public_key.pem` | File path | Where to save/load the public key |
| `RATE_LIMIT_REQUESTS_PER_HOUR` | No | `5` | Integer ≥ 1 | Max requests per IP per hour (Traefik) |
| `RATE_LIMIT_BURST` | No | `1` | Integer ≥ 1 | Burst capacity for rate limiter |
| `LOG_ENABLED` | No | `false` | `true`, `false` | Enable structured JSON logging |

*\*Required for USB printer when auto-detection fails.*

**Rate Limiting** is enforced by Traefik at the reverse proxy level:
- 5 requests per IP per hour
- Burst capacity of 1
- Returns `429 Too Many Requests` when exceeded

**Structured Logging** (when `LOG_ENABLED=true`) outputs JSON log entries:
```json
{"timestamp":"2026-05-23T12:00:00Z","level":"INFO","message":"ZeroDrop Terminal starting","fields":{"version":"1.0","printer_type":"mock"}}
```

---

## Security Model

### Zero-Knowledge Guarantee

The server **never** possesses either the plaintext payload or the private key needed to decrypt it:

1. **Key generation** — On startup, the server generates an X25519 key pair
2. **Public key distribution** — The public key is served via `GET /key` and saved to disk
3. **Private key logging** — The private key is logged as a scannable QR code for the operator
4. **Burn Protocol** — The private key is explicitly zeroed from memory using `runtime.KeepAlive()` to prevent compiler optimization
5. **Client-side encryption** — All encryption happens in the browser using Web Crypto API
6. **Server ignorance** — The server receives only the encrypted ciphertext, which it cannot decrypt

### Memory Hygiene

- All payload buffers are explicitly zeroed (`for i := range payload { payload[i] = 0 }`) after printing
- `runtime.KeepAlive()` prevents the compiler from optimizing away the zeroing
- The private key is set to `nil` after zeroing to assist garbage collection

### Security Measures

| Measure | Implementation |
|---------|---------------|
| No persistent storage | No database. RAM-only processing. |
| Forward compatibility | All payloads prefixed with `ZD1:` version header |
| Key fingerprinting | SHA-256 hash of public key printed on startup for operator verification |
| Rate limiting | 5 requests per IP per hour via Traefik (mitigates DDoS) |
| Non-root container | Docker runs as `zerodrop` user (UID 1000) |
| Read-only filesystem | Container root filesystem can be set read-only in production |
| No-new-privileges | Docker security option prevents privilege escalation |
| Resource limits | Production deploy: 0.5 CPU, 128MB memory limit |
| Graceful shutdown | 30-second drain timeout for print queue completion |

### Threat Model

| Threat | Mitigation |
|--------|-----------|
| Server compromise | Server cannot decrypt stored payloads (no private key) |
| Network interception | Payload is encrypted before transmission |
| Physical printer access | Only ciphertext is printed; no plaintext exposed |
| Database breach | No database exists |
| DoS attack | Rate limiting at reverse proxy level |
| Memory dump | Ephemeral buffers zeroed with compiler-proof technique |

---

## USB Printer Support

### Auto-Detection

USB printers are auto-detected by scanning common device paths (`/dev/usb/lp0-2`, `/dev/usblp0-2`) and identifying the hardware through sysfs (`idVendor`/`idProduct`).

### Supported Models

| Vendor ID | Product ID | Model |
|-----------|-----------|-------|
| `1504` | `0006` | POS-5890 / Generic ESC/POS |
| `04b8` | `0202` | Epson TM-T88 (compatible) |
| `04b8` | `0203` | Epson TM-T88II |
| `0416` | `5011` | Rongta RP58 |
| `0456` | `0808` | XPrinter XP-58III |
| `0493` | `b002` | Citizen CT-S310 |
| `0519` | `0001` | Star Micronics TSP650 |
| `0dd4` | `01a5` | BCST Printers |
| `20d1` | `0001` | Gainscha |
| `0fe6` | `811e` | Zjiang |
| `0418` | `0156` | Custom VG205 |

### Fallback Behavior

If no USB printer is found or initialization fails, the system **gracefully falls back to Mock Printer** (logging output to stdout), ensuring the service remains operational without hardware.

### UDEV Rules

For production use, install udev rules to grant the service user access to the printer:

```bash
# The Makefile.deploy target generates and installs these rules:
make -f Makefile.deploy setup-udev

# This creates /tmp/99-zerodrop-printer.rules with rules for all supported printers
# To install:
sudo cp /tmp/99-zerodrop-printer.rules /etc/udev/rules.d/
sudo udevadm control --reload-rules
sudo udevadm trigger
```

---

## Testing

```bash
# Run all tests with verbose output
go test ./... -v

# Run tests with race detection (CI standard)
go test ./... -race -v

# Run tests with coverage report
go test ./... -cover -coverprofile=coverage.out -v
go tool cover -html=coverage.out

# Run specific package tests
go test ./pkg/crypto -v
go test ./pkg/config -v

# Run integration tests (starts server, checks health and key endpoints)
make test-integration

# Full CI pipeline (checks + race tests + build all)
make ci
```

### Test Coverage Areas

| Package | Tests | Key Coverage |
|---------|-------|-------------|
| `pkg/config` | 9 | Default values, env var parsing, validation of printer type, device path, rate limits, log flag |
| `pkg/crypto` | 6 | Key pair generation, PEM file save, QR logging, Burn Protocol memory zeroing, fingerprint format, key initialization |
| `pkg/printer` | 2 test files (14 tests) | Mock printer operations, USB printer auto-detection, health check, device identification |
| `pkg/qr` | — | QR code generation + ESC/POS rasterization (no test file yet) |
| Integration | — | Server startup, health endpoint (including 503), key endpoint |

---

## Deployment

### Docker Deployment

```bash
# Using the main Makefile (recommended for development)
make deploy-dev      # Deploy in development mode
make deploy          # Full production deployment (checks + build + deploy)

# Or using the deploy Makefile
make -f Makefile.deploy deploy           # Full production deployment
make -f Makefile.deploy deploy-binary    # Deploy as single binary
make -f Makefile.deploy deploy-update    # Update with backup

# Manual Docker deployment
docker-compose -f docker-compose.yml \
               -f docker-compose.prod.yml \
               -f docker-compose.traefik.yml \
               up -d
```

### Binary Deployment

```bash
# Build everything
make build-all

# Run as standalone binary
make run

# Or manually:
PRINTER_TYPE=usb LOG_ENABLED=false ./bin/zerodrop
```

### System Setup

The project includes automation for production system setup via `Makefile.deploy`:

```bash
# Install systemd service
make -f Makefile.deploy setup-systemd

# Configure USB printer permissions
make -f Makefile.deploy setup-udev

# Create dedicated service user
make -f Makefile.deploy setup-user

# Configure firewall rules
make -f Makefile.deploy setup-firewall

# Backup and restore
make -f Makefile.deploy backup-key
make -f Makefile.deploy backup-config
make -f Makefile.deploy restore-key

# Security checks
make -f Makefile.deploy check-secure
make -f Makefile.deploy scan-vulns
make -f Makefile.deploy update-deps
```

### Production Checks

```bash
# Quick health check
curl http://localhost:8080/health
curl http://localhost:8080/key

# Full health check with printer details
make -f Makefile.deploy health-detailed

# Monitor continuously
make monitor          # Health check every 5 seconds

# Service status
make -f Makefile.deploy status

# View logs
make logs
docker-compose logs -f zerodrop
```

---

## Offline Decryption

The `static/reader.html` file is a standalone, fully offline decryption tool:

- **No external dependencies** — Pure HTML/CSS/JavaScript, no CDN links, no npm packages (jsQR saved locally)
- **Camera QR scanning** — Uses jsQR v1.4.0 (saved locally as `static/jsqr.min.js`) to decode QR codes from the device camera via WebRTC `getUserMedia`
- **Real ECDH decryption** — Implements full X25519 ECDH + AES-256-GCM decryption using Web Crypto API (same algorithm as the submission portal)
- **PEM key import** — Supports importing private keys in PEM format (PKCS#8) with proper binary DER parsing
- **Zero network calls** — Opens and works entirely offline after the initial page load
- **Key fingerprinting** — Displays the SHA-256 fingerprint of the derived public key for verification

**Usage:**
1. Open `static/reader.html` in a browser (works offline)
2. Click "Start Camera" and scan the QR code printed by ZeroDrop
3. Paste the private key (logged as QR code during server startup, or copy from operator console output)
4. Click "Decrypt" to reveal the plaintext message

**Payload format:** `ZD1:base64(ephemeralPubKey(32) + iv(12) + aesCiphertextWithTag)` — the ephemeral public key enables stateless decryption without any server involvement.

---

## Development

### Makefile Targets

The main `Makefile` provides comprehensive development automation:

| Category | Target | Description |
|----------|--------|-------------|
| **Setup** | `install` | Install all dependencies (Go + Node) |
| | `install-go` | Install Go dependencies only |
| | `install-node` | Install Node dependencies only |
| **Development** | `dev` | Start development server (Mock Printer) |
| | `dev-usb` | Start development server (USB Printer) |
| | `dev-frontend` | Start frontend dev server with API proxy |
| | `run` | Run production binary |
| **Build** | `build` | Build both backend and frontend |
| | `build-backend` | Build Go binary |
| | `build-frontend` | Build React frontend for production |
| | `build-all` | Clean build of all components |
| **Testing** | `test` | Run all unit tests |
| | `test-race` | Run tests with race detection |
| | `test-coverage` | Run tests with coverage report |
| | `test-integration` | Run integration tests against live server |
| **Docker** | `docker-build` | Build Docker images |
| | `docker-up` | Start services (dev mode) |
| | `docker-up-prod` | Start services (production mode) |
| | `docker-down` | Stop services |
| | `docker-logs` | View service logs |
| | `docker-clean` | Remove Docker resources |
| **Quality** | `check` | Run all checks (deps, tests) |
| | `check-deps` | Verify Go and Node are installed |
| | `format` | Run `go fmt` |
| | `lint` / `vet` | Run `go vet` |
| **Operations** | `stop` | Stop running server |
| | `restart` | Restart server |
| | `status` | Show running status |
| | `logs` | Show server logs |
| | `health` | Check server health |
| **Utilities** | `gen-key` | Generate new key pair |
| | `deps` | Show dependency tree |
| | `version` | Show version information |
| | `backup` | Backup public key |
| | `monitor` | Continuous health monitoring |
| **CI/CD** | `ci` | Full CI pipeline (check + race test + build) |
| | `ci-test` | CI: tests with race detection |
| | `ci-build` | CI: build all components |
| | `ci-deploy` | CI: test, build, and deploy to production |

### CI/CD Pipeline

```bash
# Run the full CI pipeline
make ci

# This executes:
# 1. Dependency checks
# 2. All tests with race detection
# 3. Build backend and frontend
```

For automated deployment:

```bash
make ci-deploy   # Test → Build → Deploy to production (Docker)
```

---

## Documentation

| Document | Location | Audience |
|----------|----------|----------|
| Product Requirements | `docs/prd/PRD-001-zerodrop-terminal-v1.0.md` | Product and engineering teams |
| Project Overview | `docs/OVERVIEW.md` | Stakeholders, non-technical readers |
| Architecture Decisions | `.claude/state/DECISIONS_LOG.md` | Engineering team |
| Coding Standards | `.claude/CODING_STANDARDS.md` | Go developers |
| Security Standards | `.claude/SECURITY_STANDARDS.md` | Security review |
| Environment Guide | `.claude/ENVIRONMENT_GUIDE.md` | Operations team |
| Offline Reader | `static/reader.html` | Recipients (self-documenting UI) |

---

## Contributing

This is a security-focused application. All contributions must:

1. **Pass all tests** with race detection enabled (`go test ./... -race`)
2. **Pass `govulncheck`** with no vulnerabilities
3. **Maintain zero-knowledge guarantee** — Server must never possess plaintext or private key
4. **Follow Go coding standards** as defined in `.claude/CODING_STANDARDS.md`
5. **Follow security standards** as defined in `.claude/SECURITY_STANDARDS.md`
6. **Update documentation** — Keep `.claude/` files in sync with any changes
7. **Zero memory leaks** — All sensitive buffers must be explicitly zeroed after use

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run tests: `make check test-race`
5. Run security checks: `make -f Makefile.deploy check-secure`
6. Submit a pull request

---

## License

MIT License — See [LICENSE](./LICENSE) file for details.

---

<div align="center">
<strong>ZeroDrop Terminal v1.0</strong> — Zero-Knowledge Secure Credential Delivery<br>
Built with Go, React, Web Crypto API, and ESC/POS thermal printers.
</div>
