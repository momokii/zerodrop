## Task Queue

> Implementation backlog for ZeroDrop Terminal.
> **v1.0: All milestones complete! | v1.1: Planning complete, ready for implementation**

---

### Template Format

| Field               | Value                                               |
|---------------------|-----------------------------------------------------|
| Task ID             | TASK-001                                            |
| Name                | [Task name]                                         |
| Priority            | High / Medium / Low                                 |
| Status              | TODO / IN PROGRESS / DONE / BLOCKED                 |
| Complexity          | S / M / L                                           |
| Depends On          | [Task IDs this task requires to be done first]      |
| Scope               | [Exact description of what must be built]           |
| Acceptance Criteria | [What "done" looks like, measurable]                |
| Security Concerns   | [Any security considerations specific to this task] |

---

### Backlog

#### Milestone 1: Project Bootstrap & Crypto Foundation

| Field               | Value                                                                    |
|---------------------|--------------------------------------------------------------------------|
| Task ID             | M-01                                                                     |
| Name                | Project Bootstrap & Crypto Foundation                                    |
| Priority            | High                                                                     |
| Status              | **DONE**                                                                 |
| Complexity          | M                                                                        |
| Depends On          | None                                                                     |
| Scope               | Initialize Go module, create directory structure, implement `pkg/crypto` (key generation, Burn Protocol), create `pkg/config` (env var validation). |
| Acceptance Criteria | AC-001, AC-003 (key generation and Burn Protocol); AC-021, AC-022 (config validation) |
| Security Concerns   | Burn Protocol must use `runtime.KeepAlive()` to prevent compiler optimization. Private key must never be persisted. |

---

#### Milestone 2: API & Spooler Core

| Field               | Value                                                                                                    |
|---------------------|----------------------------------------------------------------------------------------------------------|
| Task ID             | M-02                                                                                                     |
| Name                | API & Spooler Core                                                                                       |
| Priority            | High                                                                                                     |
| Status              | **DONE**                                                                                                 |
| Complexity          | L                                                                                                        |
| Depends On          | M-01                                                                                                    |
| Scope               | Implement `pkg/api` (`GET /key`, `POST /drop`), `pkg/spooler` (worker pool, memory zeroing), `pkg/observability` (structured logging). |
| Acceptance Criteria | AC-004, AC-005, AC-006, AC-007, AC-008, AC-011 (API and spooler); AC-023 (structured logging) |
| Security Concerns   | Spooler must zero payload buffers after print jobs. API must not decrypt payloads. Logging must exclude sensitive data. |

---

#### Milestone 3: Printer Interface & Reader

| Field               | Value                                                                                      |
|---------------------|--------------------------------------------------------------------------------------------|
| Task ID             | M-03                                                                                       |
| Name                | Printer Interface & Reader                                                                 |
| Priority            | High                                                                                       |
| Status              | **DONE**                                                                                   |
| Complexity          | L                                                                                          |
| Depends On          | M-02                                                                                       |
| Scope               | Implement `pkg/printer` (Printer interface, Mock Printer, QR ciphertext logging), create `static/reader.html` stub. QR generation deferred to M-04; real ECDH decrypt added post M-05. |
| Acceptance Criteria | AC-009 (Mock Printer); AC-015, AC-016 (`reader.html`)                                      |
| Security Concerns   | `reader.html` must perform all decryption locally. No external dependencies. QR format must match spec. |

---

#### Milestone 4: USB Printer & Health Check

| Field               | Value                                                                                             |
|---------------------|---------------------------------------------------------------------------------------------------|
| Task ID             | M-04                                                                                              |
| Name                | USB Printer & Health Check                                                                         |
| Priority            | High                                                                                              |
| Status              | **DONE**                                                                                           |
| Complexity          | M                                                                                                 |
| Depends On          | M-03                                                                                              |
| Scope               | Implement USB Printer with auto-detection, `GET /health` endpoint (enhanced post M-05 with 503), graceful shutdown, Docker configuration, QR ESC/POS rasterization in `pkg/qr/qr.go`. |
| Acceptance Criteria | AC-010, AC-017, AC-018 (USB printer, health check); AC-020 (graceful shutdown); AC-025 (Docker) |
| Security Concerns   | USB device permissions must be configured correctly (no privileged mode). Graceful shutdown must drain spooler. |

---

#### Milestone 5: Frontend & Production Readiness

| Field               | Value                                                                                                              |
|---------------------|--------------------------------------------------------------------------------------------------------------------|
| Task ID             | M-05                                                                                                               |
| Name                | Frontend & Production Readiness                                                                                    |
| Priority            | High                                                                                                               |
| Status              | **DONE**                                                                                                           |
| Complexity          | L                                                                                                                  |
| Depends On          | M-04                                                                                                               |
| Scope               | Implement submission portal (React + Vite + shadcn/ui), Web Crypto API encryption (real X25519 ECDH + AES-256-GCM added post M-05), SPA serving, documentation. |
| Acceptance Criteria | AC-002, AC-012, AC-013, AC-014, AC-019 (frontend and encryption)            |
| Security Concerns   | Submission portal must encrypt in browser using Web Crypto API. No external CDN dependencies. SPA served by backend. |

---

### Task Summary

- **Total Milestones**: 5
- **Completed**: 5 (M-01, M-02, M-03, M-04, M-05) ✅
- **In Progress**: 0
- **Blocked**: 0
- **Dependencies**: M-01 → M-02 → M-03 → M-04 → M-05 (all complete)

---

### Test Coverage

- **Total tests**: 23 passed
- **Packages tested**: 7 (`pkg/api`, `pkg/config`, `pkg/crypto`, `pkg/observability`, `pkg/printer`, `pkg/spooler`, `cmd/zerodrop`)
- **Coverage**: Good for security-critical packages
- **Race detection**: Clean (`go test -race ./...`)
- **Vulnerabilities**: 0 (`govulncheck ./...`)
- **Frontend**: Production build verified, SPA serving functional

---

### Production Ready

ZeroDrop Terminal v1.0 is **ready for production deployment**:

1. **Backend**: Single 8.9MB binary, no external runtime dependencies
2. **Frontend**: React SPA embedded in backend, no CDN required
3. **Security**: Zero-knowledge architecture, Web Crypto API, Burn Protocol
4. **Hardware**: USB printer auto-detection with Mock Printer fallback
5. **Infrastructure**: Docker Compose with secure defaults
6. **Documentation**: Complete README, PRD, and `.claude/` standards

---

### Deployment Options

1. **Docker Compose** (recommended for production):
   ```bash
   make -f Makefile.deploy deploy
   ```

### Deployment Makefile

Created `Makefile.deploy` with comprehensive production operations:

**Quick Deploy:**
- `make -f Makefile.deploy deploy` — Full production deployment (Docker)
- `make -f Makefile.deploy deploy-update` — Update with backup
- `make -f Makefile.deploy deploy-rollback` — Rollback to previous version

**Build:**
- `make -f Makefile.deploy build-docker` — Build Docker images

**Operations:**
- `make -f Makefile.deploy run-docker` — Start Docker services
- `make -f Makefile.deploy stop-docker` / `restart-docker` — Manage Docker
- `make -f Makefile.deploy health` / `health-detailed` — Health checks
- `make -f Makefile.deploy status` — Show service status
- `make -f Makefile.deploy logs` / `logs-docker` — View logs

**Backup & Restore:**
- `make -f Makefile.deploy backup-key` / `restore-key` — Key management
- `make -f Makefile.deploy backup-config` / `restore-config` — Configuration backup

**Security:**
- `make -f Makefile.deploy check-secure` — Run security checks (vet, vulnscan, race detection)
- `make -f Makefile.deploy update-deps` — Update and audit dependencies
- `make -f Makefile.deploy scan-vulns` — Scan for vulnerabilities

---

### Future Enhancements (v1.1+)

- Multi-printer support (multiple USB printers or mixed mock/usb)
- Admin dashboard for print queue monitoring
- Metrics export (Prometheus) for spooler depth, print times, error rates
- Mobile app for on-the-go decryption
- Private key QR scanning from reader.html (scan operator's QR to import private key)
- TCP network printer support (for shared network thermal printers)
- Configurable QR code size / error correction level per job

---

## v1.1 — Admin Dashboard & Key Persistence

> **Plan document:** `docs/plans/v1.1-admin-dashboard.md`
> **Branch:** `feature/v1.1-admin-dashboard` (implementation starts here)
> **Status:** Planning complete, ready for implementation

### v1.1-Task 1: Persistent Key Pair Storage

| Field               | Value                                                                    |
|---------------------|--------------------------------------------------------------------------|
| Task ID             | v1.1-T1                                                                  |
| Name                | Persistent Key Pair Storage                                              |
| Priority            | High                                                                     |
| Status              | TODO                                                                     |
| Complexity          | M                                                                        |
| Depends On          | None                                                                     |
| Scope               | Modify `pkg/crypto/crypto.go` to support persistent key pairs. Save private key to `data/private_key.pem` (0600) on first run. On subsequent starts, reuse existing key pair. Add `KEY_ROTATE` env var to force regeneration. Private key only loaded into RAM during first-run QR display, then burned. |
| Acceptance Criteria | 1. First run generates and saves both keys to `data/`. 2. Subsequent runs reuse existing keys. 3. `KEY_ROTATE=true` forces new key generation. 4. Private key file has 0600 permissions. 5. Existing tests still pass. |
| Security Concerns   | Private key file must be 0600. Docker runs as non-root. Burn Protocol still applies after first-run display. |

### v1.1-Task 2: Spooler Metrics Collection

| Field               | Value                                                                    |
|---------------------|--------------------------------------------------------------------------|
| Task ID             | v1.1-T2                                                                  |
| Name                | Spooler Metrics Collection                                               |
| Priority            | High                                                                     |
| Status              | TODO                                                                     |
| Complexity          | M                                                                        |
| Depends On          | None                                                                     |
| Scope               | Add metrics tracking to `pkg/spooler`: total jobs processed, total failures, current queue depth, average print duration. Expose via `GetMetrics()` method. Thread-safe with sync/atomic. |
| Acceptance Criteria | 1. `GetMetrics()` returns queue depth, total processed, total failed, avg print duration. 2. Metrics update atomically on each job. 3. No race conditions. 4. Existing tests still pass. |
| Security Concerns   | Metrics contain no sensitive data (no payloads, no keys).                |

### v1.1-Task 3: PrinterManager — Multi-Printer Detection & Selection

| Field               | Value                                                                    |
|---------------------|--------------------------------------------------------------------------|
| Task ID             | v1.1-T3                                                                  |
| Name                | PrinterManager — Multi-Printer Detection & Selection                     |
| Priority            | Medium                                                                   |
| Status              | TODO                                                                     |
| Complexity          | M                                                                        |
| Depends On          | None                                                                     |
| Scope               | Create `pkg/printmgr/printmgr.go` with `PrinterManager` struct. Detects all connected USB printers on startup. Holds reference to active printer. Supports runtime switching via `SetActivePrinter(id)`. Spooler gets its printer from the manager. |
| Acceptance Criteria | 1. Detects all connected USB printers. 2. Lists printers with ID, name, path. 3. Can switch active printer at runtime. 4. Falls back to Mock Printer if no USB found. 5. Existing tests still pass. |
| Security Concerns   | Printer switching is admin-only (auth required in Task 4).              |

### v1.1-Task 4: Admin Authentication

| Field               | Value                                                                    |
|---------------------|--------------------------------------------------------------------------|
| Task ID             | v1.1-T4                                                                  |
| Name                | Admin Authentication                                                     |
| Priority            | High                                                                     |
| Status              | TODO                                                                     |
| Complexity          | M                                                                        |
| Depends On          | None                                                                     |
| Scope               | Create `pkg/admin/admin.go`. `ADMIN_TOKEN` env var (required for admin). `POST /api/admin/login` with `{"token": "..."}` returns HMAC-signed session cookie. Constant-time token comparison (`crypto/subtle.ConstantTimeCompare`). All `/api/admin/*` endpoints require valid session. |
| Acceptance Criteria | 1. Valid token returns session cookie. 2. Invalid token returns 401. 3. Admin endpoints reject requests without valid cookie. 4. Token comparison is constant-time. 5. Session expires after configurable duration. |
| Security Concerns   | Constant-time comparison prevents timing attacks. Session cookie HMAC-signed with server secret. `ADMIN_TOKEN` must be set in .env (not committed). |

### v1.1-Task 5: Admin API Endpoints

| Field               | Value                                                                    |
|---------------------|--------------------------------------------------------------------------|
| Task ID             | v1.1-T5                                                                  |
| Name                | Admin API Endpoints                                                      |
| Priority            | High                                                                     |
| Status              | TODO                                                                     |
| Complexity          | L                                                                        |
| Depends On          | v1.1-T1, v1.1-T2, v1.1-T3, v1.1-T4                                     |
| Scope               | Add admin API routes to `pkg/api/server.go`: `GET /api/admin/metrics` (spooler stats), `GET /api/admin/printers` (detected printers), `POST /api/admin/printers/active` (switch active printer), `GET /api/admin/keys` (key info + fingerprint), `POST /api/admin/keys/rotate` (force key rotation). All require admin session. |
| Acceptance Criteria | 1. All endpoints return correct JSON. 2. All endpoints require auth. 3. Printer switching works at runtime. 4. Key rotation generates new pair. 5. Metrics reflect real-time state. |
| Security Concerns   | All admin endpoints behind auth middleware. Key rotation requires admin. |

### v1.1-Task 6: Admin Dashboard Frontend

| Field               | Value                                                                    |
|---------------------|--------------------------------------------------------------------------|
| Task ID             | v1.1-T6                                                                  |
| Name                | Admin Dashboard Frontend                                                 |
| Priority            | High                                                                     |
| Status              | TODO                                                                     |
| Complexity          | L                                                                        |
| Depends On          | v1.1-T5                                                                  |
| Scope               | Add `/admin` React route to frontend. Dashboard shows: spooler metrics (queue depth, jobs processed, failures, avg print time), printer list with dropdown to select active printer, key info (fingerprint, first-generated date), key rotation button. Login screen for token auth. Auto-refresh metrics every 5s. |
| Acceptance Criteria | 1. Login screen accepts admin token. 2. Dashboard shows real-time metrics. 3. Printer dropdown lists detected printers. 4. Key info shows fingerprint. 5. Key rotation works. 6. Responsive layout with shadcn/ui components. |
| Security Concerns   | Token never stored in localStorage — session cookie only. Dashboard only accessible with valid session. |

### v1.1-Task 7: Private Key QR Scan in reader.html

| Field               | Value                                                                    |
|---------------------|--------------------------------------------------------------------------|
| Task ID             | v1.1-T7                                                                  |
| Name                | Private Key QR Scan in reader.html                                       |
| Priority            | Medium                                                                   |
| Status              | TODO                                                                     |
| Complexity          | M                                                                        |
| Depends On          | None                                                                     |
| Scope               | Add camera-based QR scanning to `static/reader.html` for private key import. Uses existing jsQR library. Scan operator's QR code (JWK or PEM format) to import private key without copy-paste. Button to toggle between camera scan and manual paste. |
| Acceptance Criteria | 1. Camera scan detects JWK QR and imports private key. 2. Camera scan detects PEM QR and imports private key. 3. Toggle between scan and paste mode. 4. Existing manual paste still works. 5. Works offline (jsQR already local). |
| Security Concerns   | Camera access requires HTTPS or localhost (same as current reader.html). Private key stays client-side. |

---

### v1.1 Task Summary

- **Total Tasks**: 7
- **Completed**: 0
- **In Progress**: 0
- **Blocked**: 0
- **Dependencies**: T1, T2, T3, T4 (parallel) → T5 → T6; T7 (independent)

---

### Implementation Notes

- **Branch**: All v1.1 work happens on `feature/v1.1-admin-dashboard`
- **Main branch**: Stable v1.0 — no changes during v1.1 development
- **Test strategy**: Each task includes failing test → implement → verify
- **Commit strategy**: Commit after each task with conventional commit messages
- **Docker**: v1.1 changes require updating Dockerfile and docker-compose files
