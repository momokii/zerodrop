## Task Queue

> Implementation backlog for ZeroDrop Terminal v1.0, as defined in PRD-001.
> **All milestones complete!**

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
| Scope               | Implement `pkg/printer` (Printer interface, Mock Printer, QR simulation), create `static/reader.html`. |
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
| Scope               | Implement USB Printer with auto-detection, `GET /health` endpoint, graceful shutdown, Docker configuration. |
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
| Scope               | Implement submission portal (React + Vite + shadcn/ui), Web Crypto API encryption, SPA serving, documentation. |
| Acceptance Criteria | AC-002, AC-012, AC-013, AC-014, AC-019 (frontend and encryption); AC-024 (Traefik rate limiting)            |
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
5. **Infrastructure**: Docker Compose + Traefik with rate limiting
6. **Documentation**: Complete README, PRD, and `.claude/` standards

---

### Deployment Options

1. **Single Binary** (simplest):
   ```bash
   make -f Makefile.deploy deploy-binary
   make -f Makefile.deploy run-binary
   ```

2. **Docker Compose** (recommended for production):
   ```bash
   make -f Makefile.deploy deploy
   ```

3. **Systemd Service** (for Linux servers):
   ```bash
   make -f Makefile.deploy setup-user
   make -f Makefile.deploy setup-udev
   make -f Makefile.deploy setup-systemd
   ```

### Deployment Makefile

Created `Makefile.deploy` with comprehensive production operations:

**Quick Deploy:**
- `make -f Makefile.deploy deploy` — Full production deployment (Docker)
- `make -f Makefile.deploy deploy-binary` — Deploy as single binary
- `make -f Makefile.deploy deploy-update` — Update with backup
- `make -f Makefile.deploy deploy-rollback` — Rollback to previous version

**Build:**
- `make -f Makefile.deploy build-all` — Build all (binary + Docker)
- `make -f Makefile.deploy build-binary` — Build production binary
- `make -f Makefile.deploy build-docker` — Build Docker images

**Operations:**
- `make -f Makefile.deploy run-binary` / `run-docker` — Start services
- `make -f Makefile.deploy stop-docker` / `restart-docker` — Manage Docker
- `make -f Makefile.deploy health` / `health-detailed` — Health checks
- `make -f Makefile.deploy status` — Show service status
- `make -f Makefile.deploy logs` / `logs-docker` — View logs

**Backup & Restore:**
- `make -f Makefile.deploy backup-key` / `restore-key` — Key management
- `make -f Makefile.deploy backup-config` / `restore-config` — Configuration backup
- `make -f Makefile.deploy rollback-archive` — Create rollback archive
- `make -f Makefile.deploy rollback-list` — List rollback archives
- `make -f Makefile.deploy rollback-clean` — Clean old archives

**Security:**
- `make -f Makefile.deploy check-secure` — Run security checks (vet, vulnscan, race detection)
- `make -f Makefile.deploy update-deps` — Update and audit dependencies
- `make -f Makefile.deploy scan-vulns` — Scan for vulnerabilities

**System Setup:**
- `make -f Makefile.deploy setup-systemd` — Generate systemd service file
- `make -f Makefile.deploy setup-udev` — Generate USB printer udev rules
- `make -f Makefile.deploy setup-user` — Create dedicated service user
- `make -f Makefile.deploy setup-firewall` — Firewall configuration

---

### Future Enhancements (v1.1+)

- Full ECDH encryption in `reader.html`
- Multi-printer support
- Admin dashboard for print queue monitoring
- Metrics export (Prometheus)
- Enhanced QR code with error correction
- Mobile app for on-the-go decryption
