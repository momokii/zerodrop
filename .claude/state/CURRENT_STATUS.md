## Project Phase

**v1.1 Implementation Complete — Ready for Merge** on `feature/v1.1-admin-dashboard` branch

---

## Completed

- [x] .claude/ agent infrastructure initialized
- [x] PRD-001: ZeroDrop Terminal v1.0 (Initial Implementation) completed
- [x] PRD-001 v1.1: Frontend stack revised to React + Vite + shadcn/ui
- [x] M-01: Project Bootstrap & Crypto Foundation
- [x] M-02: API & Spooler Core
- [x] M-03: Printer Interface & Reader
- [x] M-04: USB Printer & Health Check
- [x] M-05: Frontend & Production Readiness
- [x] v1.1 Implementation Plan written (`docs/plans/v1.1-admin-dashboard.md`)
- [x] v1.1-T1: Persistent Key Pair Storage
- [x] v1.1-T2: Spooler Metrics Collection
- [x] v1.1-T3: PrinterManager — Multi-Printer Detection & Selection
- [x] v1.1-T4: Admin Authentication
- [x] v1.1-T5: Admin API Endpoints (8 endpoints at /api/admin/*)
- [x] v1.1-T6: Admin Dashboard Frontend (/admin React route)
- [x] v1.1-T7: Private Key QR Scan in reader.html

---

## In Progress

- None — v1.1 complete, awaiting merge to main

---

## Blocked

- None

---

## Open Questions

- ~~**OQ-001**: Which 58mm thermal printer models for production testing?~~ → **RESOLVED**: Auto-detection implemented
- ~~**OQ-002**: Acceptable spooler drain timeout for graceful shutdown?~~ → **RESOLVED**: 30 seconds
- ~~**OQ-003**: Should structured logging be enabled by default or opt-in?~~ → **RESOLVED**: Opt-in (default: false)

---

## Tech Stack

- Go backend, React + Vite + shadcn/ui frontend, Docker Compose
- Crypto: Curve25519 (X25519) via `crypto/ecdh`, Web Crypto API for browser
- Architecture: Zero-knowledge, ephemeral processing, no database persistence

---

## Security Notes

- Zero-knowledge guarantee is non-negotiable: server never possesses plaintext or private key
- Burn Protocol with `runtime.KeepAlive()` is critical for memory hygiene
- QR version header (`ZD1:`) enables forward compatibility
- Key fingerprinting (SHA-256) provides operator verification against key substitution
- Frontend uses Web Crypto API (browser-native) — no external crypto libraries
- All 80 tests passing with race detection enabled (up from 26 in v1.0)
- `govulncheck` shows no known vulnerabilities
- Frontend dependencies have only dev-time vulnerabilities (esbuild/vite not in production)
- Admin auth uses constant-time token comparison with session cookies (default 24h expiry, configurable via `ADMIN_SESSION_TTL`)
- Login rate-limited: 10 attempts per 15 minutes per IP

---

## Session History

| Session | Date | Summary |
|---|---|---|---|
| 1 | 2026-05-11 | Bootstrap: created full .claude/ agent infrastructure |
| 2 | 2026-05-11 | PRD creation: drafted and approved PRD-001 for ZeroDrop Terminal v1.0 |
| 3 | 2026-05-11 | PRD revision v1.1: Frontend stack changed to React + Vite + shadcn/ui |
| 4 | 2026-05-11 | M-01 implementation: Go module, pkg/crypto, pkg/config |
| 5 | 2026-05-11 | M-02 implementation: pkg/api, pkg/spooler, pkg/observability |
| 6 | 2026-05-11 | M-03 implementation: pkg/printer, static/reader.html |
| 7 | 2026-05-11 | Standards update: All `.claude/` files updated with Go standards |
| 8 | 2026-05-12 | M-04 implementation: USB printer auto-detection, health check, Docker |
| 9 | 2026-05-12 | M-05 implementation: React + Vite + shadcn/ui frontend, Web Crypto API, production build, SPA serving |
| 10 | 2026-05-23 | ECIES crypto chain: real X25519 ECDH + AES-256-GCM, QR ESC/POS rasterization, health 503, payload 250→400 |
| 11 | 2026-05-24 | SPKI public key format fix, LOG_ENABLED confirmed, docs audit |
| 12 | 2026-05-24 | Traefik removal, Docker-only deployment Makefile, docs cleanup |
| 13 | 2026-05-24 | Rate limiter middleware, 429 responses, reverse proxy recommendation |
| 14-16 | 2026-05-25 | Bug fixes (make dev foreground, health check wget), Docker frontend build |
| 17-20 | 2026-05-28 | Non-secure context detection, TLS self-signed cert, TLS error suppression, PKCS#8 DER fix |
| 21 | 2026-05-30 | JWK format switch, log ordering fix, fingerprint independence, paste listeners |
| 22 | 2026-05-31 | PEM non-blocking fallback, raw base64 paste support, extractJWK label handling |
| 23 | 2026-06-04 | Docker .env fix, QR interleaving fix, PNG QR files, dual PEM+JWK QR, v1.1 plan written |
| 24 | 2026-06-06 | v1.1 implementation complete on feature/v1.1-admin-dashboard. 80 tests passing. |
| 25 | 2026-06-08 | Doc audit: fixed test counts (43→80), updated project structure, README, OVERVIEW, ENVIRONMENT_GUIDE, SECURITY_STANDARDS, PRD; added missing env vars and v1.1 feature descriptions. All docs now reflect v1.1 state. |
| 3 | 2026-05-11 | PRD revision v1.1: Frontend stack changed to React + Vite + shadcn/ui |
| 4 | 2026-05-11 | M-01 implementation: Go module, pkg/crypto, pkg/config |
| 5 | 2026-05-11 | M-02 implementation: pkg/api, pkg/spooler, pkg/observability |
| 6 | 2026-05-11 | M-03 implementation: pkg/printer, static/reader.html |
| 7 | 2026-05-11 | Standards update: All `.claude/` files updated with Go standards |
| 8 | 2026-05-12 | M-04 implementation: USB printer auto-detection, health check, Docker |
| 9 | 2026-05-12 | M-05 implementation: React + Vite + shadcn/ui frontend, Web Crypto API, production build, SPA serving |
| 10 | 2026-05-23 | ECIES crypto chain: real X25519 ECDH + AES-256-GCM encryption in frontend and reader.html, QR ESC/POS rasterization (pkg/qr), health check 503, payload limit 250→400, Docker USB group_add; full PRD audit & fixes |
| 11 | 2026-05-24 | SPKI public key format fix: SavePublicKeyToFile uses x509.MarshalPKIXPublicKey instead of publicKey.Bytes() so Web Crypto API importKey("spki",...) succeeds. GetPublicKeyFingerprint hashes SPKI DER to match frontend. LOG_ENABLED confirmed working. Docs audit and update. |
| 12 | 2026-05-24 | **Traefik removal**: Deleted infrastructure/traefik/ and docker-compose.traefik.yml. Rewrote Makefile.deploy as Docker-only (removed binary/systemd/udev/firewall/rollback targets). Cleaned Traefik refs from README.md, docs/OVERVIEW.md, TESTING.md, CLAUDE.md, .claude/*.md, .env.example. All docs updated to Docker-only architecture. |
| 13 | 2026-05-24 | **Rate limiter middleware**: Added per-IP sliding window rate limiter in pkg/api/server.go using existing RateLimitRequestsPerHour/RateLimitBurst config. Applied to all API endpoints. Returns HTTP 429 when exceeded. Restored RATE_LIMIT_ vars in .env.example with reverse proxy recommendation. Updated docs to reflect: built-in basic rate limiting + deploy behind nginx/caddy for production security. |
| 14 | 2026-05-25 | **Bug fixes**: Fixed `make dev` running `go run` in background (`&`) causing Ctrl+C to leave zombie processes holding port 8080 — now runs in foreground so SIGINT reaches the shutdown handler. Fixed `make stop` to also kill `go run` and `go-build` processes. Fixed error logging to print actual error details (e.g. "address already in use") even when LOG_ENABLED=false. Fixed docker-compose.override.yml removing `build.target: builder` which skipped the Dockerfile runtime stage, producing images without ENTRYPOINT that exited immediately. |
| 15 | 2026-05-25 | **Docker health check + rate limiter fixes**: Fixed Docker HEALTHCHECK using `wget --spider` (not supported by Alpine's busybox wget) — changed to `wget -q -O /dev/null`. Excluded `/health` endpoint from rate limiter so Docker health checks don't get rate-limited (429) and mark container unhealthy. Port exposure was working correctly — previous "no ports" issue was caused by zombie `make dev` process holding host port 8080. |
| 16 | 2026-05-25 | **Docker frontend build + make dev warning**: Added `frontend-builder` stage to Dockerfile (node:20-alpine) that builds the React SPA during docker build — frontend now ships inside the Docker image. `make dev` now warns when `frontend/dist/` is missing with instructions to run `make build-frontend`. Docker image size: 16.6MB. |

---

## Codebase Statistics

- **Total packages**: 8 Go packages (added `pkg/qr/`) + 1 React frontend
- **Total tests**: 80 passing (Go backend) — up from 26 in v1.0
- **Go dependencies**: 2 (gorilla/mux, skip2/go-qrcode)
- **Frontend dependencies**: 343 packages (344 with audit) + 257KB jsQR offline library
- **Vulnerabilities**: 0 Go, 2 moderate (frontend dev-time only)
- **Race conditions**: 0 (go test -race clean)
- **Production build size**: 
  - Backend binary: 8.9MB
  - Frontend dist: 152.9KB JS + 15.5KB CSS + 506B HTML

---

## New in M-05 (Complete)

### Frontend Stack
- **React 18.3** with TypeScript
- **Vite 5.4** for fast development and optimized builds
- **Tailwind CSS 3.4** for styling with design tokens
- **shadcn/ui** components (copied into codebase, no npm dependency)

### Components Created (18 files)
- `src/lib/crypto.ts` — Web Crypto API utilities
- `src/lib/api.ts` — API client functions
- `src/lib/utils.ts` — Utility functions
- `src/components/ui/` — 6 shadcn/ui components
- `src/App.tsx` — Main application with 4-step encryption flow
- `src/main.tsx` — Entry point
- `src/index.css` — Tailwind + custom styles
- Configuration files (vite, tsconfig, tailwind, eslint, postcss, package)

### Features Implemented
- Fetches server public key and displays SHA-256 fingerprint
- Encrypts messages in browser using Web Crypto API (X25519 ECDH + AES-256-GCM ECIES)
- Validates payload before submission (400 char limit, ZD1: prefix, base64)
- Shows server health and printer status in real-time
- Zero-knowledge guarantee explained in UI
- Responsive design with gradient backgrounds
- Copy-to-clipboard functionality for public key
- Error handling and status messages throughout flow

### Backend Integration
- SPA serving from Go backend (http.FileServer)
- SPA fallback handler for client-side routing
- API routes work with both `/api/*` and direct paths for compatibility
- Frontend assets served efficiently (gzip compressed)

### Production Build
- Frontend: 152.9KB JS (50KB gzipped) + 15.5KB CSS (4KB gzipped)
- Backend: 8.9MB single binary
- No external CDN dependencies
- Self-contained deployment

---

---

## Post M-05 — ECIES Crypto Chain & QR Print (Session 10)

### What Changed

The encryption chain was upgraded from stub/simulated to **real ECIES** (Elliptic Curve Integrated Encryption Scheme) across all three layers — frontend, server, and offline reader:

| Layer | Before | After |
|-------|--------|-------|
| **Frontend (`crypto.ts`)** | X25519 ECDH key gen stub, concatenated key+btoa as ciphertext | Real X25519 ECDH + AES-256-GCM encryption via Web Crypto API, PEM parser (`parsePEM`), `encryptData()` returns `ZD1:base64(ephPubKey(32)+iv(12)+ciphertextWithTag)` |
| **Server (`pkg/qr/qr.go`)** | Ciphertext printed as raw text | QR code generated via `go-qrcode` (Medium error correction), rasterized to ESC/POS GS v 0 bit-image commands |
| **Offline Reader (`reader.html`)** | Stub that threw "not implemented" | Real ECDH decryption with jsQR camera scanning, `static/jsqr.min.js` saved locally for zero-network operation |

### New Files Created

| File | Purpose |
|------|---------|
| `pkg/qr/qr.go` | QR code generation + ESC/POS GS v 0 rasterization for thermal printers |
| `static/jsqr.min.js` | jsQR v1.4.0 QR decoding library (saved locally for offline use) |

### Updated Files

| File | What Changed |
|------|-------------|
| `pkg/printer/mock.go` | QR ESC/POS commands + hex preview in logs instead of raw ciphertext |
| `pkg/printer/usb.go` | Writes QR GS v 0 raster commands to USB device; auto-detection fallback path for `PRINTER_DEVICE` (now optional) |
| `pkg/crypto/crypto.go` | Private key logged as QR PNG (via `qr.GenerateQRPNG`) + PEM text fallback |
| `pkg/api/server.go` | `/health` returns HTTP 503 when `IsAvailable()` returns false (was always 200) |
| `frontend/src/lib/crypto.ts` | **Rewritten**: real X25519 ECDH + AES-256-GCM, PEM parser (`parsePEM`), `decryptData()` |
| `frontend/src/App.tsx` | Fixed PEM import bug (raw PEM string → SPKI), status messages per PRD spec |
| `frontend/src/lib/api.ts` | Payload limit: 250 → 400 characters |
| `static/reader.html` | jsQR continuous camera scan wired, real ECDH + AES-256-GCM decrypt |
| `docker-compose.prod.yml` | Added `group_add: [dialout, lp]` for USB printer device permissions |

### Protocol Specification

- **Algorithm**: X25519 ECDH (Curve25519) + AES-256-GCM
- **Payload format**: `ZD1:base64(ephemeralPubKeyRaw(32) + iv(12) + aesCiphertextWithTag)`
- **Max payload**: 400 characters (was 250)
- **Max plaintext**: ~185 characters ASCII
- **Server /health**: Returns HTTP 503 when printer unavailable (via `IsAvailable()` from the `Printer` interface)

### PRD Audit

Full PRD-001 audit conducted after implementation. Results:
- **20/39** FRs: **FULLY** implemented
- **9/39** FRs: **PARTIAL** (now all resolved)
- **7/39** FRs: **NOT IMPLEMENTED** (now all resolved)
- **3/39** FRs: **MISSING from code** (FR-14, FR-15, FR-21 — now resolved)

The PRD was corrected to fix inaccuracies: FR-007 (400-char limit), FR-024 (split into submission + reader), FR-030 (IsAvailable-based 503), FR-034 (PRINTER_DEVICE optional with auto-detection).

---

## Quick Start

### Development

```bash
# Backend
go build -o bin/zerodrop ./cmd/zerodrop
PRINTER_TYPE=mock ./bin/zerodrop

# Frontend (dev server with proxy)
cd frontend
npm install
npm run dev  # Runs on :3000, proxies API to :8080
```

### Production (Single Binary)

```bash
# Build frontend
cd frontend
npm install
npm run build

# Build backend with embedded frontend
cd ..
go build -o bin/zerodrop ./cmd/zerodrop

# Run (serves frontend on :8080)
PRINTER_TYPE=mock ./bin/zerodrop

# Access at http://localhost:8080
```

### Production (Docker)

```bash
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

---

## Verification

All tests passing:
- ✅ 23 Go tests with race detection
- ✅ Frontend builds successfully
- ✅ Backend serves SPA correctly
- ✅ API endpoints work (GET /key, POST /drop, GET /health)
- ✅ No Go vulnerabilities
- ✅ Frontend vulnerabilities are dev-time only (esbuild/vite not in production build)

---

## Session 11 — SPKI Public Key Format Fix (2026-05-24)

### What Changed

The `SavePublicKeyToFile` function was writing raw 32-byte X25519 bytes into the PEM file. The frontend's `crypto.subtle.importKey("spki", ...)` requires full SPKI DER structure (algorithm OID + key). Fixed by using `x509.MarshalPKIXPublicKey(publicKey)` instead of `publicKey.Bytes()`.

Similarly, `GetPublicKeyFingerprint` now hashes the SPKI DER bytes so the Go server and frontend fingerprints match for operator verification.

### Root Cause
- `publicKey.Bytes()` returns raw 32-byte X25519 public key — not valid SPKI DER
- This caused `importKey("spki", ...)` to fail with "encryption failed" on the frontend
- Old PEM: `CkcyzxMs8aRKMlSagZJqO5mSjW2zgsqRMImcEShWHi8=` (raw bytes, ~70 bytes)
- New PEM: `MCowBQYDK2VuAyEA...` (SPKI DER with X25519 OID 1.3.101.110, ~113 bytes)

### Files Changed
| File | Change |
|------|--------|
| `pkg/crypto/crypto.go` | Added `crypto/x509` import. `SavePublicKeyToFile`: `x509.MarshalPKIXPublicKey` instead of `publicKey.Bytes()`. `GetPublicKeyFingerprint`: same SPKI DER hash. |

## Session 21 — Log Ordering, Fingerprint Independence, Paste Listeners (2026-05-30)

### What Changed

**Log interleaving fix**: Consolidated all `LogPrivateKeyAsQR` stdout writes into a single buffer (`strings.Builder`) to prevent interleaving between `fmt.Fprintf(os.Stdout, ...)` and `log.Printf(os.Stderr, ...)` output in Docker logs. Commit includes this fix.

**Fingerprint independence**: Separated `extractPublicKey` (private key import) from `updateFingerprint` (SPKI import + SHA-256 hash). Fingerprint failure now shows "Unavailable" instead of throwing an error that blocks decryption. The private key import succeeds independently — if fingerprint calculation fails (e.g., `importKey("spki", ...)` not supported for X25519), the decrypt button remains enabled.

**Paste event listeners**: Added `paste` event listeners (with 50ms defer for value to settle) to both the private key and payload textareas, as a fallback for browsers where `input` events don't fire reliably on paste.

**Cache busting**: Added `/reader-v2.html` route alias as an explicit cache-busting URL (`Cache-Control: no-cache, no-store, must-revalidate` headers). Added version badge (v3 → v4) with ISO timestamp for visual freshness confirmation.

### Files Changed

| File | Change |
|------|--------|
| `pkg/crypto/crypto.go` | Consolidated stdout writes into single buffer via `strings.Builder` |
| `static/reader.html` | Separated fingerprint from key import (v4), paste listeners, cache meta tags |

### Last Updated

2026-05-30 — **Log ordering & fingerprint independence**: Log output consolidated to single buffer preventing stdout/stderr interleaving in Docker. Fingerprint calculation separated from private key import — failures show "Unavailable" instead of blocking decryption. Paste event listeners on both textareas. Reader version v4.

## Session 22 — PEM Non-Blocking, Raw Base64, Makefile Rebuild (2026-05-31)

### What Changed

**PEM non-blocking**: PEM import failures no longer block decryption. `loadPrivateKeyPEM` catches errors without disabling the button — if a prior JWK import already set `privateKey`, decrypt still works. Fingerprint shows "Unavailable" on PEM failure.

**Raw base64 paste**: Added `looksLikeBase64()` detection. Pasting just the raw PEM body (without `-----BEGIN`/`-----END` headers) auto-wraps in PEM markers and attempts import.

**Label-wrapped JWK**: `extractJWK()` now handles pastes with surrounding label lines (`=== PRIVATE KEY (JWK) === ... === END PRIVATE KEY JWK ===`), extracting the JSON between first `{` and last `}`.

**Makefile rebuild targets**: `docker-up` now supports `REBUILD=true` flag to add `--build` to `up -d`. New `docker-up-rebuild` and `docker-restart-rebuild` targets. Rebuilt Docker image with all latest changes baked in.

### Files Changed

| File | Change |
|------|--------|
| `static/reader.html` | PEM non-blocking (v7), raw base64 paste support, label-wrapped JWK extraction |
| `pkg/crypto/crypto.go` | Log labels: "JWK - Recommended", "PEM - Alternative" |
| `Makefile` | `docker-up-rebuild`, `docker-restart-rebuild`, `REBUILD=true` flag |
| `README.md` | Updated docs for new Makefile targets |
| `CLAUDE.md` | Session 22 history added |

### Last Updated

2026-05-31 — **PEM non-blocking + Makefile rebuild**: PEM import failures are non-blocking. Raw base64 paste auto-wraps as PEM. `extractJWK` handles label-wrapped pastes. `make docker-up-rebuild` and `make docker-restart-rebuild` added. Docker image rebuilt with all latest changes.

2026-05-28 — **TLS support with self-signed certificate**: Added `TLS_ENABLED` env var (default `false`). When enabled, the server generates an ECDSA P-256 self-signed certificate at startup and serves HTTPS on port 8080 instead of HTTP. Makes `crypto.subtle` available from other devices (HTTPS = secure context). Browsers show security warning — click "Advanced" → "Proceed to site". Docker HEALTHCHECK uses `--no-check-certificate`. Commit `TBD`. Config: `TLS_ENABLED=true`.

2026-05-28 — **Non-secure context detection fix**: Added early `window.crypto.subtle` check in App.tsx init() to catch the case where the page is accessed over plain HTTP from a non-localhost device. Web Crypto API (`crypto.subtle`) is only available in secure contexts (HTTPS or localhost). Previously showed a cryptic "Cannot read properties of undefined (reading 'digest')" error — now shows a clear message explaining HTTPS/localhost requirement. Frontend rebuilt in Docker image. Commit `3fc2407`. **Still blocked by browser security model**: this is a browser-enforced restriction and cannot be bypassed without HTTPS or localhost.

2026-05-25 — **Docker frontend build**: Added node:20-alpine frontend-builder stage to Dockerfile, React SPA now built during docker build (image: 16.6MB). `make dev` warns when frontend/dist/ is missing.

2026-05-25 — **Docker health check + rate limiter fixes**: Fixed HEALTHCHECK using `wget --spider` (broken in busybox) → `wget -q -O /dev/null`. Excluded `/health` from rate limiter so Docker health checks don't 429. Zombie `make dev` processes holding port 8080 was root cause of "no ports" issue.

2026-05-25 — **Bug Fixes**: Fixed `make dev` foreground (removed `&`), `make stop` kills `go run`/`go-build` processes, error logging details in LOG_ENABLED=false mode, docker-compose.override.yml `build.target: builder` removed to fix Docker entrypoint.

2026-05-24 — **Rate Limiter Middleware**: Added per-IP sliding window rate limiter wired to existing config. Returns HTTP 429 when exceeded. Docs updated with reverse proxy recommendation for production.

2026-05-24 — **Traefik Removal & Docker-Only Deployment**: Deleted infrastructure/traefik/, docker-compose.traefik.yml. Makefile.deploy rewritten as Docker-only. All docs cleaned of Traefik references.

2026-05-24 — **SPKI Format Fix & Docs Audit**: `SavePublicKeyToFile` now produces proper SPKI DER for Web Crypto API compatibility. All docs updated to reflect latest state. Structured JSON logging confirmed working with `LOG_ENABLED=true`.

2026-05-23 — **ECIES Crypto Chain Complete**: Real X25519 ECDH + AES-256-GCM encryption across all layers. QR ESC/POS rasterization. Health check 503. Payload limit 250→400. Docker USB permissions. Full PRD audit complete. All 39 FRs resolved.

2026-05-18 — **Deployment Makefile Complete**: Created `Makefile.deploy` with 30+ production operations targets. Includes deploy (binary/Docker), build, run/stop, health/status, backup/restore, security checks, system setup (systemd/udev/firewall), and rollback operations. Documented in DECISIONS_LOG.md and TASK_QUEUE.md.

2026-05-12 — **M-05 COMPLETE**: ZeroDrop Terminal v1.0 ready for production deployment. All 5 milestones implemented. Frontend (React + Vite + shadcn/ui) complete with Web Crypto API integration. Backend serves SPA with fallback handler. Production builds verified.

---

## v1.1 — Admin Dashboard & Key Persistence (2026-06-06 COMPLETE)

### Branch

`feature/v1.1-admin-dashboard` — ready for merge to main.

### Task Summary

| Task | Description | Status |
|------|-------------|--------|
| 1 | Persistent Key Pair Storage — changed from ephemeral (new keys every restart → data loss) to persistent keys saved to disk (0600). Private key burned from RAM after initial QR display, never loaded on subsequent starts. Preserves zero-knowledge guarantee while preventing data loss on restart. Security rationale documented in README.md Security Model section. | **DONE** |
| 2 | Spooler Metrics Collection | **DONE** |
| 3 | PrinterManager — Multi-Printer Detection & Selection | **DONE** |
| 4 | Admin Authentication (`ADMIN_TOKEN`, session cookie, login rate limit) | **DONE** |
| 5 | Admin API Endpoints (8 endpoints at `/api/admin/*`) | **DONE** |
| 6 | Admin Dashboard Frontend (`/admin` React route) | **DONE** |
| 7 | Private Key QR Scan in reader.html | **DONE** |

### New Files

| File | Purpose |
|------|---------|
| `pkg/spooler/metrics.go` | Thread-safe Metrics struct (queue depth, processed/failed, duration) |
| `pkg/printer/manager.go` | PrinterManager — detect, select, switch printers at runtime |
| `pkg/printer/manager_test.go` | PrinterManager tests |
| `pkg/api/middleware.go` | Admin auth middleware, session cookies, login rate limiting |
| `pkg/api/middleware_test.go` | Middleware tests |
| `pkg/api/admin.go` | 8 admin API endpoint handlers |
| `pkg/api/admin_test.go` | Admin API tests |
| `frontend/src/pages/Admin.tsx` | Admin dashboard React page |
| `frontend/src/lib/admin-api.ts` | Admin API client functions |

### New Environment Variables

| Variable | Description |
|----------|-------------|
| `ADMIN_TOKEN` | Shared secret for admin auth (set in `.env`) |
| `ADMIN_SESSION_TTL` | Admin session lifetime (Go duration, default: `24h`) |
| `KEY_ROTATE` | Set `true` to force key regeneration (default: `false`) |
| `PRIVATE_KEY_PATH` | Path to save/load private key PEM (default: `./data/private_key.pem`) |

### Security Rationale — Why Persistent Keys Don't Break Zero-Knowledge

The shift from ephemeral to persistent keys was operationally necessary (restart = data loss in v1.0) and does not compromise security:

| Concern | Why It's Fine |
|---------|---------------|
| Private key on disk | 0600 permissions, same threat profile as SSH/TLS keys. Root access exposes all secrets on any system. |
| Private key in RAM | Only loaded once (first boot QR display), then burned. Never loaded on subsequent starts. |
| Server can decrypt? | No — server never opens the key file after initial provisioning. |
| Key rotation | `KEY_ROTATE=true` forces fresh keys. Old ciphertext becomes undecryptable — operator chooses when. |

**Red alert:** If someone gets root on your server, they can read `private_key.pem`. This is the same as them reading your SSH host key, TLS private key, or database password file. ZeroDrop has always assumed the server is in a physically controlled environment. The zero-knowledge guarantee during active submission/printing is **unchanged from v1.0**.

### Last Updated

2026-06-08 — **Doc audit (session 25)**: Fixed stale test counts (43→80) across all docs. Updated README.md project structure (added `pages/`, spooler test files). Added missing env vars (`ADMIN_SESSION_TTL`, `TLS_ENABLED`) to ENVIRONMENT_GUIDE.md and SECURITY_STANDARDS.md. Added v1.1 features (admin dashboard, persistent keys) to OVERVIEW.md. Fixed Traefik reference in PRD. All docs now reflect v1.1 state. 80 tests passing.
