# ZeroDrop — Agent Orientation

## What Is This Repository?

**ZeroDrop Terminal** is an air-gapped, zero-knowledge secure credential delivery terminal. Users encrypt sensitive data (passwords, API keys, security reports) in their browser using Web Crypto API, transmit it to a server that cannot decrypt it, and receive the ciphertext as a physical QR code printout via 58mm thermal printer. Recipients decrypt the printout offline using a standalone `reader.html` file.

**Tech stack:** Go backend, React + Vite + shadcn/ui frontend, Docker Compose, Curve25519 cryptography.

**Current phase:** **v1.1 Complete** — Admin Dashboard & Key Persistence (all 7 tasks implemented)

---

## Agent Orientation Sequence

Any agent arriving at this project cold **must** read these files in this exact order before taking any action:

1. **`HOW_TO_RESUME.md`** — The numbered step-by-step protocol for session startup
2. **`state/CURRENT_STATUS.md`** — What is done, in progress, and blocked right now
3. **`state/TASK_QUEUE.md`** — The 5 implementation milestones from PRD-001
4. **`docs/prd/PRD-001-zerodrop-terminal-v1.0.md`** — Complete Product Requirements Document
5. **`AGENT_RULES.md`** — Non-negotiable behavioral rules (mandatory every session)
6. **`CODING_STANDARDS.md`** — Go-specific coding conventions (updated for this project)
7. **`SECURITY_STANDARDS.md`** — Security requirements (mandatory before any implementation)
8. **`ENVIRONMENT_GUIDE.md`** — Environment definitions and Go development commands

---

## Directory Map

```
.claude/
├── README.md                 ← You are here
├── settings.json             # Tool permissions (allow/deny rules)
├── AGENT_RULES.md            # Mandatory behavioral rules
├── CODING_STANDARDS.md       # Go-specific coding conventions
├── SECURITY_STANDARDS.md     # Security requirements for all implementation
├── ENVIRONMENT_GUIDE.md      # Environment definitions and Go commands
├── HOW_TO_RESUME.md          # Session startup protocol
├── state/
│   ├── CURRENT_STATUS.md     # Living state: done / in progress / blocked
│   ├── TASK_QUEUE.md         # 5 implementation milestones (ALL DONE ✅)
│   └── DECISIONS_LOG.md      # Record of 40 decisions from PRD + v1.0 + v1.1 implementation
└── templates/
    ├── new_feature.md        # Checklist: implementing a new feature
    ├── new_endpoint.md       # Checklist: adding an API endpoint
    ├── new_test.md           # Checklist: writing a test scenario
    └── bug_fix.md            # Checklist: investigating and fixing a bug

docs/
├── prd/
│   └── PRD-001-zerodrop-terminal-v1.0.md  # Complete PRD for v1.0
├── plans/
│   └── v1.1-admin-dashboard.md  # v1.1 implementation plan (7 tasks)
└── OVERVIEW.md             # Simple explanation for stakeholders

pkg/                          # Go packages (M-01 through M-04 complete)
├── crypto/                   # ✅ ECC key generation, Burn Protocol, persistent key pair
├── api/                      # ✅ HTTP server, endpoints, health check (503), SPA serving, admin API
├── printer/                  # ✅ Printer interface, Mock Printer, USB Printer (auto-detect), PrinterManager
├── qr/                       # ✅ QR code generation + ESC/POS GS v 0 rasterization
├── spooler/                  # ✅ Asynchronous print job queue + metrics
├── config/                   # ✅ Environment variable validation
└── observability/            # ✅ Structured logging, health check, graceful shutdown

static/                       # Web assets
├── reader.html               # ✅ Offline decryption utility (jsQR + real ECDH decrypt)
└── jsqr.min.js               # ✅ jsQR v1.4.0 — local QR decoding for offline reader

cmd/
└── zerodrop/
    └── main.go               # ✅ Application entry point

Makefile                     # ✅ Development Makefile
Makefile.deploy              # ✅ Production deployment Makefile

frontend/                     # ✅ React + Vite + shadcn/ui (M-05 complete)
├── src/
│   ├── components/
│   │   └── ui/              # ✅ shadcn/ui components (Button, Card, Input, etc.)
│   ├── lib/
│   │   ├── crypto.ts        # ✅ Web Crypto API utilities
│   │   ├── api.ts           # ✅ API client functions
│   │   └── utils.ts         # ✅ Utility functions
│   ├── App.tsx              # ✅ Main application component
│   ├── main.tsx             # ✅ Entry point
│   └── index.css            # ✅ Tailwind CSS + custom styles
├── index.html               # ✅ HTML template
├── package.json             # ✅ Dependencies
├── vite.config.ts           # ✅ Vite configuration
├── tsconfig.json            # ✅ TypeScript configuration
├── tailwind.config.js       # ✅ Tailwind configuration
└── dist/                    # ✅ Production build (served by Go backend)

infrastructure/               # Docker and deployment (M-04)
└── (future: monitoring, logging)

# Docker files (M-04)
Dockerfile                   # ✅ Multi-stage build
docker-compose.yml           # ✅ Base configuration
docker-compose.override.yml   # ✅ Development overrides
docker-compose.prod.yml       # ✅ Production overrides

.env.example                 # ✅ Environment variable template
```

---

## Current Task State

- **Status file:** `state/CURRENT_STATUS.md` — **v1.1 Complete** — Admin Dashboard & Key Persistence implemented
- **Task backlog:** `state/TASK_QUEUE.md` — 5 v1.0 milestones (ALL DONE ✅) + 7 v1.1 tasks (ALL DONE ✅)
- **Decision history:** `state/DECISIONS_LOG.md` — 38 decisions logged (31 v1.0 + 7 v1.1)
- **PRD:** `docs/prd/PRD-001-zerodrop-terminal-v1.0.md` — updated with 400-char limit, FR-024 split, FR-030/FR-034 corrections
- **v1.1 Plan:** `docs/plans/v1.1-admin-dashboard.md` — 7 tasks (ALL IMPLEMENTED ✅)

**Project Status:** ZeroDrop Terminal v1.1 is **COMPLETE** on `feature/v1.1-admin-dashboard` branch. 43 tests passing, frontend builds successfully.

---

## Self-Update Policy

All `.claude/` files are living documents and **must** be updated as the project evolves. At the end of every working session, the agent must:

- Update `state/CURRENT_STATUS.md` with session summary and timestamp
- Update `state/TASK_QUEUE.md` with completed/in-progress tasks
- Update `state/DECISIONS_LOG.md` with any new decisions
- Update relevant standards files as tech stack decisions are made
- Update this README if directory structure changes

This is not optional. Keeping `.claude/` accurate is part of every task.

---

## Important Notes

### Security (Critical)

- **Zero-knowledge is non-negotiable**: Server must never possess plaintext payload or private key at any point during operation
- **Burn Protocol**: Must use `runtime.KeepAlive()` after zeroing private key from memory
- **Persistent keys (v1.1)**: Private key saved to disk (0600) so restarts don't destroy access to previously encrypted data. Only loaded into RAM once (first boot), then burned. Never loaded on subsequent starts — see README.md Security Model for full rationale
- **Memory hygiene**: All payload buffers zeroed after print job completion
- **No database**: Ephemeral RAM-only processing
- **Rate limiting**: Applied at application level

### Architecture Decisions

- **Single printer**: v1.0 supports one thermal printer (multi-printer via PrinterManager in v1.1)
- **Persistent keys**: v1.1 saves private key to disk on first run, reuses across restarts (D-001)
- **React + shadcn/ui**: Pre-built components for professional UI, fast implementation, simple to maintain
- **ECIES protocol**: X25519 ECDH + AES-256-GCM across all layers; payload = `ZD1:base64(ephPubKey+iv+ciphertextWithTag)`
- **QR version header**: `ZD1:` prefix for forward compatibility
- **QR rasterization**: go-qrcode (Medium error correction) + ESC/POS GS v 0 bit-image commands
- **Key fingerprinting**: SHA-256 hash logged on startup for operator verification
- **Health check 503**: `/health` returns 503 when printer unavailable via `IsAvailable()`
- **Graceful shutdown**: 30-second spooler drain timeout (configurable)
- **USB auto-detection**: Scans for 10+ thermal printer models, falls back to Mock Printer

### Tech Stack

| Component | Technology | Status |
|-----------|------------|--------|
| Backend | Go 1.26+ | ✅ Implemented |
| Crypto | X25519 ECDH + AES-256-GCM (ECIES) via Go `crypto/ecdh` + Web Crypto API | ✅ Implemented |
| API | gorilla/mux | ✅ Implemented |
| QR Generation | go-qrcode (skip2) + ESC/POS GS v 0 rasterization | ✅ Implemented |
| Spooler | Buffered channel + worker pool | ✅ Implemented |
| Printer | Mock Printer + USB Printer (auto-detect) | ✅ Implemented |
| Health Check | Enhanced with printer status + 503 when unavailable | ✅ Implemented |
| SPA Serving | Go http.FileServer with fallback | ✅ Implemented |
| Docker | Multi-stage build, Alpine-based | ✅ Implemented |

| Frontend | React + Vite + shadcn/ui + TypeScript | ✅ Implemented |
| Admin Dashboard | React SPA at /admin, session auth, printer mgmt, key mgmt | ✅ v1.1 |
| Web Crypto | X25519 ECDH + AES-256-GCM encryption/decryption in browser | ✅ Implemented |
| Offline Decoding | jsQR v1.4.0 (local file, no network) | ✅ Implemented |
| Testing | Go testing framework + Mock Printer | ✅ 43 tests passing |
| Hardware | 58mm thermal printer (ESC/POS), USB | ✅ Supported |

### Test Coverage

- **Total tests**: 120 passing (80 functional + 40 security)
- **Race detection**: Clean (`go test -race ./...`)
- **Vulnerabilities**: 0 (govulncheck clean)
- **Coverage**: Good for security-critical packages

### Development Commands

```bash
# Backend tests
go test ./... -v -race -cover

# Backend build
go build -o bin/zerodrop ./cmd/zerodrop

# Run backend (serves frontend at root)
PRINTER_TYPE=mock ./bin/zerodrop

# Run backend (USB Printer with auto-detection)
PRINTER_TYPE=usb PRINTER_DEVICE="" ./bin/zerodrop

# Frontend development (optional - for development only)
cd frontend
npm install
npm run dev        # Dev server on :3000 with proxy to :8080
npm run build      # Production build (already done)
npm run preview    # Preview production build

# Production (single binary)
# Frontend already built in frontend/dist/
# Just run the backend:
PRINTER_TYPE=mock ./bin/zerodrop
# Access at http://localhost:8080

# Docker Compose (production)
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# Using Deployment Makefile (production operations)
make -f Makefile.deploy deploy              # Full deployment
make -f Makefile.deploy check-secure       # Security checks
make -f Makefile.deploy status             # Service status
make -f Makefile.deploy logs-docker        # Docker logs
make -f Makefile.deploy backup-key         # Backup public key

# Lint
go fmt ./...
go vet ./...
govulncheck ./...
```

### Production Deployment

ZeroDrop Terminal v1.0 is ready for production deployment:

**Using Deployment Makefile (recommended):**
```bash
# Quick deployment (Docker)
make -f Makefile.deploy deploy
```

**Manual deployment:**

1. **Docker Compose**:
   ```bash
   docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
   ```

2. **Single Binary**:
   ```bash
   # Build frontend (already done)
   cd frontend && npm run build && cd ..

   # Build backend
   go build -o bin/zerodrop ./cmd/zerodrop

   # Run
   PRINTER_TYPE=usb PRINTER_DEVICE="" ./bin/zerodrop
   ```

3. **Access**:
   - Frontend: `http://localhost:8080/`
   - API: `http://localhost:8080/key`, `/drop`, `/health`

### USB Printer Auto-Detection

The system automatically detects common 58mm thermal printers:
- POS-5890 / Generic ESC/POS (VID:PID 1504:0006)
- Epson TM-T88 series (04b8:0202, 04b8:0203)
- Rongta RP58 (0416:5011)
- XPrinter XP-58III (0456:0808)
- And 6 more common models

If no USB printer is found, the system gracefully falls back to Mock Printer.

### Production Build Sizes

- **Backend binary**: 8.9MB (single file, no dependencies)
- **Frontend**: 152.9KB JS (50KB gzipped) + 15.5KB CSS (4KB gzipped) + 506B HTML
- **Total**: ~170KB frontend assets + 8.9MB backend = ~9MB total deployment
