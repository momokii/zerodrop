# ZeroDrop Terminal v1.2 — Upgrade Recommendations

> **Date:** 2026-06-23
> **Scope:** Comprehensive codebase audit + structural hardening implementation

---

## Project Understanding

ZeroDrop Terminal is an **air-gapped, zero-knowledge secure credential delivery terminal**. The core workflow:

1. **Submitter** encrypts plaintext in-browser via Web Crypto API (X25519 ECDH + AES-256-GCM)
2. **Server** receives only ciphertext — cannot decrypt (private key never in RAM after boot)
3. **Printer** outputs the ciphertext as a scannable QR code on thermal paper
4. **Recipient** scans the QR code offline using `reader.html` and decrypts with the private key

**Zero-knowledge is the non-negotiable foundation.** The server never possesses the plaintext or the private key during operation.

### Current State (v1.1 + structural upgrades)

| Layer | Status |
|-------|--------|
| Go backend (crypto, API, spooler, printer, qr, config, observability) | ✅ Production-grade, 120+ tests, 50 security checks |
| React frontend (encryption portal, admin dashboard) | ✅ Complete with shadcn/ui |
| Offline reader (reader.html) | ✅ jsQR + ECDH decrypt, camera fallback fixed |
| Persistent key pairs | ✅ Private key saved to disk (0600), never loaded after boot |
| Admin dashboard | ✅ Token auth, printer management, key management, spooler metrics |
| CI/CD pipeline | ✅ 3 jobs (backend, frontend, Docker smoke test) |
| Error handling | ✅ Typed APIError across all handlers |
| Middleware | ✅ RequestID, CORS, SecurityHeaders, RequestLogger, Gzip |
| HTTP server | ✅ Read/Write/Idle timeouts |
| Health endpoint | ✅ Version, uptime, goroutines, memory stats |

---

## Implemented in This Session (v1.2 Structural)

These changes have been implemented and committed:

### 1. Security Headers Middleware
- **CSP**: `default-src 'self'` with `script-src 'self' 'unsafe-inline' blob:`, `frame-ancestors 'none'`
- **X-Frame-Options**: `DENY`
- **X-Content-Type-Options**: `nosniff`
- **Referrer-Policy**: `strict-origin-when-cross-origin`
- **Permissions-Policy**: camera/microphone/geolocation disabled, fullscreen restricted to self
- **X-XSS-Protection**: `0` (deprecated but explicit)

### 2. Request Logging Middleware
- Logs every request: `[request_id] METHOD /path STATUS duration`
- Captures status code via `statusWriter` wrapper
- Compatible with `http.Hijacker` for future WebSocket support

### 3. Gzip Compression Middleware
- Compresses API responses when client sends `Accept-Encoding: gzip`
- Passthrough for non-gzip clients (no overhead)
- ~80% reduction on JSON payloads

### 4. HTTP Server Timeouts
- **ReadTimeout**: 10s (prevents slow-client attacks)
- **WriteTimeout**: 15s (prevents hung connections)
- **IdleTimeout**: 60s (keep-alive connections)
- Applied to both HTTP and TLS server

### 5. Health Endpoint Enhancement
- Added: `version`, `uptime`, `started_at` (RFC3339), `goroutines`, `memory_alloc`, `memory_sys`, `gc_cycles`
- Operators can now monitor memory pressure and goroutine leaks via `/health`

### 6. Camera Fallback (reader.html, earlier session)
- `facingMode: 'environment'` → `OverconstrainedError` → `facingMode: 'user'` fallback
- Explicit `OverconstrainedError` detection in error messages

**Files changed:**
- `pkg/api/middleware.go` — SecurityHeaders, RequestLogger, GzipMiddleware, statusWriter, gzipResponseWriter
- `pkg/api/server.go` — serverVersion constant, startTime tracking, middleware wiring, handler timeouts, enhanced health

---

## Groundbreaking Feature Recommendations

These are not implemented — they represent the next frontier for ZeroDrop.

### ⭐ P1: "Try It" QR Preview Mode
**Impact: High | Effort: Low-Medium**
Add an interactive demo endpoint (`GET /preview?payload=...`) that renders the QR code as an HTML `<img>` or SVG in the browser. Operators can test payload formatting without wasting thermal paper. The SPA already has the crypto primitives — just needs a QR preview component that calls `pkg/qr` on the backend or renders client-side.

**Why groundbreaking:** Every thermal printer owner knows the pain of wasted paper during testing. This eliminates it.

### ⭐ P1: Print Notification Webhook
**Impact: High | Effort: Medium**
When a print job succeeds, fire a configurable webhook (HTTP POST to a URL) with metadata: `{ "job_id", "timestamp", "status", "payload_size" }`. No payload content ever leaves the server. Useful for:
- Slack/Teams notification: "Credential delivered"
- SIEM integration: audit trail of successful prints
- Incident response automation: trigger downstream workflows

### ⭐ P2: Batch QR Splitting
**Impact: Medium | Effort: Medium-High**
For payloads >185 chars plaintext, split across multiple QR codes (chunked). reader.html scans sequentially and reassembles. Enables delivery of SSH private keys, PGP keys, or full security reports.

### ⭐ P2: Recipient Lockbox (Per-Recipient Keys)
**Impact: Medium | Effort: High**
Generate a unique encryption key per recipient. The server stores an encrypted "lockbox" — only the intended recipient can decrypt with their specific key. Turns ZeroDrop from "one key for everyone" into "each recipient gets their own key."

### 🟡 P3: Dead Man's Switch / Timed Payload
**Impact: Low-Medium | Effort: Medium**
Payload auto-destructs if not printed within N hours (configurable). Server holds ciphertext in an ephemeral TTL map instead of immediately queuing for print. Useful for time-sensitive credentials (e.g., "this root password is only valid until 5 PM").

### 🟡 P3: Prometheus Metrics Endpoint
**Impact: Low-Medium | Effort: Low**
Expose `GET /metrics` with spooler depth, print duration histogram, total payloads, error rate. The `pkg/spooler/metrics.go` already tracks this data — just needs a `/metrics` endpoint in Prometheus text format (or a `/health` extension with `?format=prometheus`).

---

## Structural Upgrade Recommendations (Future)

These are incremental improvements, not groundbreaking features.

### MEDIUM Priority

1. **Integration test framework** — Start real server, submit payload, verify spooler processes it end-to-end
2. **Structured error responses on admin login** — Return JSON for all admin error paths (currently mixed)
3. **Graceful printer degradation** — If printer.Print hangs, timeout and mark printer unavailable (currently blocks spooler worker)
4. **Context propagation through spooler** — Pass `context.Context` through processJob for cancellation/timeout on printer operations

### LOW Priority

5. **go 1.26 `http.ServeMux` migration** — `gorilla/mux` is fine but Go 1.22+ ServeMux has sufficient patterns; could drop dependency
6. **Frontend error boundaries** — Wrap React app in ErrorBoundary for graceful crash recovery
7. **Makefile lint target** — Wire `go vet`, `staticcheck`, `govulncheck` into a single target
8. **Docker healthcheck enhancement** — Check spooler depth and printer availability in HEALTHCHECK
9. **`.env.example` updates** — Document new env vars as they're added
10. **Frontend build embedding** — Use Go 1.26 `embed` package to embed frontend dist in binary

---

## Senior Engineer Observations

What a seasoned Go/React developer would notice on first review:

### What's Done Well 👍
- Clean package boundaries with clear interfaces (Printer, PrinterProvider, HealthChecker)
- Thread-safe metrics with `sync/atomic` pattern
- Burn Protocol with `runtime.KeepAlive()` — correct and well-documented
- Proper SPKI/PKCS#8 PEM handling for Web Crypto API compatibility
- Dual-format QR output (PEM + JWK) for maximum client compatibility
- Session auth with HttpOnly cookies, constant-time comparison, login rate limiting
- Graceful shutdown with 30s spooler drain
- CI/CD pipeline with race detection, vulnerability scanning, Docker build
- 50 security checks in `make check-security`

### What Needs Attention 🔧
- **Spooler blocks on printer.Print()** — no timeout on the actual print call. If a USB printer hangs mid-job, the entire spooler worker is stuck. Use a goroutine with timeout for printer I/O.
- **No context propagation** — request contexts don't flow into the spooler. If the client disconnects, the spooler still processes the job. This is acceptable for ZeroDrop's use case (print jobs should complete even if client disconnects) but should be a conscious decision.
- **Login rate limiter uses global mutex** — fine for single-server, but the map grows unbounded. Add periodic cleanup (already done for API rate limiter, but login rate limiter also needs it).
- **Test coverage lacks integration tests** — all tests use mocks. No test starts the server, submits a payload, and verifies the spooler processes it. This would catch wiring bugs.
- **Session store is in-memory only** — survives restarts? No. Acceptable for single-server, but if the server restarts, all sessions are lost (operator must re-login). This is actually fine — sessions should be ephemeral.
