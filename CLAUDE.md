# ZeroDrop Terminal

Air-gapped, zero-knowledge secure credential delivery terminal. Encrypt sensitive data in your browser, transmit it to a server that cannot decrypt it, and receive the ciphertext as a physical QR code printout. Recipients decrypt offline using a standalone HTML file.

---

## Agent Infrastructure

All agent rules, standards, state tracking, and templates live in `.claude/`.

**On every session start**, follow the orientation sequence in `.claude/HOW_TO_RESUME.md` — read the files listed there in order before taking any action.

### Guideline Map

Every `.claude/` file is a living document. The agent must consult and update them as the project evolves.

| File | When to Read | What It Governs |
|------|-------------|-----------------|
| `.claude/README.md` | Session start | Master orientation — project description, directory map, orientation sequence |
| `.claude/AGENT_RULES.md` | Session start, mandatory | Non-negotiable behavioral rules: scope lock, dependency guard, zero-regression, security, environment awareness, session end checklist |
| `.claude/HOW_TO_RESUME.md` | Session start, mandatory | 11-step numbered protocol for session startup — do not skip steps |
| `.claude/CODING_STANDARDS.md` | Before writing any code | Go-specific naming conventions, error handling, code structure, testing, documentation |
| `.claude/SECURITY_STANDARDS.md` | Before writing any code | Secrets management, input validation, dependency audits, data protection, HTTP/API security, container security, logging |
| `.claude/ENVIRONMENT_GUIDE.md` | Before running any command | Environment definitions (dev/staging/prod), agent behavior per env, Go development commands, Docker Compose pattern |
| `.claude/state/CURRENT_STATUS.md` | Session start | What is done, in progress, and blocked right now — the ground truth |
| `.claude/state/TASK_QUEUE.md` | Session start | Prioritized implementation backlog with acceptance criteria |
| `.claude/state/DECISIONS_LOG.md` | When decisions are made | Permanent record of architectural/technical/product decisions with rationale |
| `docs/prd/PRD-001-zerodrop-terminal-v1.0.md` | Before implementing any feature | Complete Product Requirements Document for v1.0 — defines all requirements, milestones, and acceptance criteria |

### Template Checklists

Use these when implementing the corresponding task type:

| Template | Use When |
|----------|----------|
| `.claude/templates/new_feature.md` | Building any new feature |
| `.claude/templates/new_endpoint.md` | Adding an API endpoint |
| `.claude/templates/new_test.md` | Writing a new test scenario |
| `.claude/templates/bug_fix.md` | Investigating and fixing a bug |

---

## Auto Self-Update Rules

**This is mandatory. Not optional.** The agent must keep all `.claude/` files accurate and relevant at all times.

### When to Auto-Update

The agent must automatically update `.claude/` files whenever any of these triggers occur:

| Trigger | What to Update |
|---------|---------------|
| **New directory or file structure is created** | `README.md` directory map — add new directories and their purpose. This `CLAUDE.md` — update if top-level structure changes. |
| **New dependency is added** | `DECISIONS_LOG.md` — log the dependency with rationale and vulnerability check result. `SECURITY_STANDARDS.md` — if the dependency introduces new security considerations. |
| **New pattern or convention is established in code** | `CODING_STANDARDS.md` — document the pattern so future sessions follow it. |
| **Architecture decision is made** | `DECISIONS_LOG.md` — log the decision, rationale, and alternatives rejected. `README.md` — update project description and structure. `SECURITY_STANDARDS.md` — update if security architecture changed. |
| **Docker or infrastructure setup is created** | `ENVIRONMENT_GUIDE.md` — replace placeholder commands with real, verified commands. `SECURITY_STANDARDS.md` — add container-specific security rules. |
| **New environment variable is introduced** | `ENVIRONMENT_GUIDE.md` — document it. Ensure `.env.example` is updated. |
| **Test framework is configured** | `CODING_STANDARDS.md` — add real test command and naming conventions. `ENVIRONMENT_GUIDE.md` — add test command to startup section. |
| **Bug is fixed or feature is completed** | `state/CURRENT_STATUS.md` — update completed/in-progress. `state/TASK_QUEUE.md` — mark task done, add discovered tasks. |
| **Session ends** | `state/CURRENT_STATUS.md` — session summary and timestamp. `state/TASK_QUEUE.md` — reflect any changes. `state/DECISIONS_LOG.md` — log any decisions made. Any `.claude/` file whose content became stale or inaccurate during the session. |

### Self-Update Verification

At the end of every session, before closing, the agent must run this verification:

- [ ] Does the project description in this `CLAUDE.md` still match reality?
- [ ] Does the directory map in `.claude/README.md` reflect the actual file structure?
- [ ] Do the commands in `.claude/ENVIRONMENT_GUIDE.md` actually work?
- [ ] Do the conventions in `.claude/CODING_STANDARDS.md` match what's actually in the codebase?
- [ ] Are there any new security considerations not captured in `.claude/SECURITY_STANDARDS.md`?
- [ ] Is `.claude/state/CURRENT_STATUS.md` accurate and timestamped?
- [ ] Are all completed tasks marked done in `.claude/state/TASK_QUEUE.md`?

If any answer is "no", fix it before ending the session.

---

## Current Status

**Phase:** M-05 Complete — ZeroDrop Terminal v1.0 Production Ready

### Product Definition

**ZeroDrop Terminal** is an air-gapped, zero-knowledge secure credential delivery terminal. Users encrypt sensitive data (passwords, API keys, security reports) in their browser using Web Crypto API, transmit it to a server that cannot decrypt it, and receive the ciphertext as a physical QR code printout via 58mm thermal printer. Recipients decrypt the printout offline using a standalone `reader.html` file and a private key that never touched the server.

### Tech Stack (Determined)

| Component | Technology | Status |
|-----------|------------|--------|
| **Backend** | Go 1.26+ | ✅ Implemented |
| **Crypto** | Curve25519 (X25519) via `crypto/ecdh`, `crypto/rand` only | ✅ Implemented |
| **Frontend** | React + Vite + shadcn/ui + Tailwind CSS | ✅ Implemented |
| **Infrastructure** | Docker Compose + Traefik (reverse proxy, rate limiting) | ✅ Implemented |
| **Hardware** | 58mm thermal printer (ESC/POS), USB connectivity | ✅ Supported |
| **Testing** | Go testing framework + Mock Printer for CI | ✅ 23 tests passing |

### Architecture

- **Zero-knowledge guarantee**: Server never possesses plaintext payload or private key
- **Ephemeral processing**: No database. Data exists in RAM only during print job, then is zeroed
- **Hardware abstraction**: `Printer` interface supports Mock Printer (stdout) and USB Printer (auto-detection)
- **Asynchronous spooler**: Buffered Go channel worker pool for sequential print processing
- **Client-side encryption**: All crypto happens in browser using Web Crypto API
- **Offline decryption**: `static/reader.html` works completely offline with no external dependencies
- **SPA serving**: Go backend serves React frontend with client-side routing fallback

### Implementation Status

| Milestone | Status | Description |
|-----------|--------|-------------|
| M-01 | ✅ DONE | Project Bootstrap & Crypto Foundation |
| M-02 | ✅ DONE | API & Spooler Core |
| M-03 | ✅ DONE | Printer Interface & Reader |
| M-04 | ✅ DONE | USB Printer & Health Check |
| M-05 | ✅ DONE | Frontend & Production Readiness |

### Open Questions

- ~~**OQ-001**: Which 58mm thermal printer models for production testing?~~ → **RESOLVED**: Auto-detection implemented, supports 10+ models
- ~~**OQ-002**: Acceptable spooler drain timeout for graceful shutdown?~~ → **RESOLVED**: 30 seconds
- ~~**OQ-003**: Should structured logging be enabled by default or opt-in?~~ → **RESOLVED**: Opt-in (default: false)

### Documentation

- **PRD**: `docs/prd/PRD-001-zerodrop-terminal-v1.0.md` (complete)
- **Overview**: `docs/OVERVIEW.md` (stakeholder-friendly explanation)
- **Decisions Log**: `.claude/state/DECISIONS_LOG.md` (24 decisions recorded)
- **Task Queue**: `.claude/state/TASK_QUEUE.md` (5 milestones, all complete)
- **Standards**: All `.claude/` files updated with Go-specific conventions

### Security Notes (Critical)

- **Zero-knowledge is non-negotiable**: Server must never possess plaintext payload or private key at any point
- **Burn Protocol**: Must use `runtime.KeepAlive()` after zeroing private key to prevent compiler optimization
- **QR version header**: All payloads prefixed with `ZD1:` for forward compatibility
- **Key fingerprinting**: SHA-256 hash of public key logged on startup for operator verification
- **Memory hygiene**: All payload buffers zeroed after print job completion
- **Rate limiting**: 5 requests per IP per hour (Traefik) to mitigate DDoS
- **Web Crypto API**: Browser-native encryption, no external libraries

### Development Commands

```bash
# Run tests (with race detection and coverage)
go test ./... -v -race -cover

# Build binary
go build -o bin/zerodrop ./cmd/zerodrop

# Run application (serves frontend + backend)
PRINTER_TYPE=mock ./bin/zerodrop

# Run application (USB Printer with auto-detection)
PRINTER_TYPE=usb PRINTER_DEVICE="" ./bin/zerodrop

# Run application (USB Printer with specific device)
PRINTER_TYPE=usb PRINTER_DEVICE=/dev/usb/lp0 ./bin/zerodrop

# Frontend development (optional - for development only)
cd frontend && npm run dev

# Lint and security checks
go fmt ./...
go vet ./...
```

### Session History

| Session | Date | Summary |
|---|---|---|
| 1 | 2026-05-11 | Bootstrap: created full .claude/ agent infrastructure (settings, rules, standards, state tracking, templates) |
| 2 | 2026-05-11 | PRD creation: drafted and approved PRD-001 for ZeroDrop Terminal v1.0 with 5 implementation milestones. Tech stack determined. |
| 3 | 2026-05-11 | PRD revision v1.1: Frontend stack changed from Vanilla JS to React + Vite + shadcn/ui for better UX |
| 4 | 2026-05-11 | M-01 implementation: Go module initialized, pkg/crypto (key generation, Burn Protocol), pkg/config (env var validation) |
| 5 | 2026-05-11 | M-02 implementation: pkg/api (endpoints), pkg/spooler (worker pool), pkg/observability (logging, graceful shutdown) |
| 6 | 2026-05-11 | M-03 implementation: pkg/printer (Printer interface, Mock Printer), static/reader.html (offline decryption) |
| 7 | 2026-05-11 | Standards update: All `.claude/` files updated with Go-specific conventions, security standards, and environment commands. `.gitignore` created. |
| 8 | 2026-05-12 | M-04 implementation: USB printer with auto-detection, enhanced health check, Docker configuration, Traefik integration |
| 9 | 2026-05-12 | M-05 implementation: React + Vite + shadcn/ui frontend, Web Crypto API integration, SPA serving, production build. **All 5 milestones complete — ZeroDrop Terminal v1.0 production ready.** |
