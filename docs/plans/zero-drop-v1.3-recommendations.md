# ZeroDrop Terminal — Post-v1.2 Upgrade Recommendations

> **Date:** 2026-06-23
> **Status:** Recommendations only — no implementation started
> **Scope:** Gaps and improvements identified after v1.2 structural/security upgrades

This document captures all recommended upgrades beyond v1.2. Use it as the canonical reference when asked "what should we do next?" or "give me upgrade recommendations."

---

## Project Purpose (Recap)

ZeroDrop Terminal is an **air-gapped, zero-knowledge secure credential delivery terminal**:

1. **Submitter** encrypts plaintext in-browser via Web Crypto API (X25519 ECDH + AES-256-GCM)
2. **Server** receives only ciphertext — **cannot decrypt** (private key never in RAM after boot)
3. **Printer** outputs the ciphertext as a scannable QR code on 58mm thermal paper
4. **Recipient** scans the QR code offline using `reader.html` and decrypts with the private key

**Zero-knowledge is non-negotiable.** The server must never possess the plaintext or the private key during operation.

---

### Current State (v1.2 Complete)

| Layer | Status |
|-------|--------|
| Go backend (crypto, API, spooler, printer, qr, config, observability) | ✅ Production-grade, 120+ tests, 50 security checks |
| React frontend (encryption portal, admin dashboard) | ✅ Complete with shadcn/ui |
| Offline reader (reader.html) | ✅ jsQR + ECDH decrypt, camera fallback, PEM+JWK import |
| Persistent key pairs | ✅ Private key saved to disk (0600), never loaded after boot |
| Burn Protocol | ✅ `runtime.KeepAlive()` prevents compiler optimization |
| Admin dashboard | ✅ Token auth, printer management, key management, spooler metrics |
| CI/CD pipeline | ✅ 3 jobs (backend test+vet+vulncheck, frontend lint+type-check+build, Docker smoke) |
| Error handling | ✅ Typed APIError across all handlers |
| Middleware chain | ✅ RequestID → SecurityHeaders → Gzip → RequestLogger → CORS |
| HTTP server timeouts | ✅ Read 10s / Write 15s / Idle 60s |
| Health endpoint | ✅ Version 1.2.0, uptime, goroutines, memory stats |
| Security verification | ✅ 50 automated checks via `make check-security` |

---

## 🔴 P0 — Must Fix (Operational Risk)

### 1. Printer Hang Protection

**Problem**: The spooler (`pkg/spooler/spooler.go`) calls `printer.Print()` with **no timeout**. If a USB printer malfunctions mid-job (paper jam, power loss, buffer stall), the worker goroutine blocks forever. The spooler stops processing all subsequent jobs until the server restarts.

**Fix**: Wrap the print call in a goroutine with a configurable timeout (default 30s). If the printer doesn't respond within the timeout:
- Mark the printer as unavailable
- Return the job as failed (with retry)
- Continue processing remaining jobs

**Files**: `pkg/spooler/spooler.go`
**Effort**: Low
**Risk if not fixed**: Silent production outage — prints stop working, no alert, user only discovers when the next person tries to print.

---

## 🟠 P1 — High Priority (Security & Reliability)

### 2. Frontend Crypto Tests

**Problem**: The browser-side encryption in `frontend/src/lib/crypto.ts` implements the full ECIES chain (X25519 ECDH + AES-256-GCM) — this is the **core of the zero-knowledge guarantee**. Yet there are **zero test files** anywhere in `frontend/`. A regression in browser API behavior, an edge case in PEM parsing, or a bug in base64 encoding would go undetected until a recipient fails to decrypt.

**Fix**: Add Vitest (Vite-native, minimal config) and test:
- `crypto.ts` — encrypt/decrypt roundtrip, PEM parsing, JWK import, fingerprint derivation, edge cases (malformed PEM, empty payload, large payload)
- `api.ts` — API client with mocked fetch for success/error/network-failure paths
- `admin-api.ts` — Admin API client (login, status, metrics, etc.)

**Files**: New `frontend/src/lib/*.test.ts`
**Effort**: Low-Medium (Vitest works with Vite out of the box)
**Risk if not fixed**: Encryption could silently break on a browser update. You'd only notice when production credentials can't be decrypted.

### 3. Login Rate Limiter — Unbounded Memory Growth

**Problem**: The login rate limiter in `pkg/api/middleware.go` stores entries in an in-memory map with **no cleanup**. Every unique IP that hits the login endpoint creates a permanent entry. The **API rate limiter** already has periodic cleanup; the login rate limiter does not.

**Fix**: Add periodic sweep of entries older than the 15-minute window. Same pattern as the API rate limiter cleanup goroutine.

**Files**: `pkg/api/middleware.go`
**Effort**: Trivial
**Risk if not fixed**: Slow unbounded memory growth in long-running deployments with many unique source IPs.

---

## 🟡 P2 — Medium Priority (Quality & Confidence)

### 4. Integration Tests

**Problem**: All 120+ tests use mocks. No test starts the real server, submits a real payload, and verifies the spooler processes it. Wiring bugs (middleware ordering, route registration, handler→spooler wiring) go undetected until runtime.

**Fix**: Add an integration test that:
1. Starts the server via `httptest.NewServer`
2. Hits `GET /key` → verify 200 + PEM body
3. Hits `POST /drop` → verify 202 + spooler processes the job
4. Hits `GET /health` → verify JSON shape (version, uptime, printer info)
5. Verifies spooler metrics increment after successful print

**Files**: New `pkg/api/integration_test.go`
**Effort**: Low
**Benefit**: Catches wiring/regression bugs that unit tests miss entirely.

### 5. Embed Frontend in Go Binary (`go:embed`)

**Problem**: The Go binary reads `frontend/dist/` from disk at runtime via `os.DirFS()`. If you move the binary or run it from a different working directory, the SPA breaks with 404s. Deployment requires ensuring `frontend/dist/` is in the right place relative to the binary.

**Fix**: Use `//go:embed frontend/dist/*` to embed the production build directory at compile time. Serve via `http.FS(embedFS)` instead of `os.DirFS("./frontend/dist")`.

```go
//go:embed frontend/dist/*
var frontendAssets embed.FS
// Serve via http.FS(frontendAssets) instead of os.DirFS("./frontend/dist")
```

**Files**: `pkg/api/server.go`, `cmd/zerodrop/main.go`
**Effort**: Low (one file change)
**Benefit**: Single-file deployment — no companion `frontend/dist/` directory needed. The binary is truly self-contained.

### 6. Printer Health Timeout in Health Check

**Problem**: The health endpoint calls `printer.IsAvailable()` which (for USB printers) tries to open the device. If the device is in a bad state, this can block the health endpoint response.

**Fix**: Wrap `IsAvailable()` with a short timeout (1-2s). If the device check hangs, return `available: false` with a `"timeout"` status rather than blocking the health response.

**Files**: `pkg/printer/usb.go`, `pkg/api/server.go`
**Effort**: Low
**Benefit**: Health endpoint remains responsive even when the printer is in a broken state.

---

## 🟢 P3 — Low Priority (Features & Polish)

### 7. TCP Network Printer Support

**Problem**: Only Mock (stdout) and USB (local device) printers are supported. Many deployments have network thermal printers or use Ethernet-to-USB adapters.

**Fix**: Add `pkg/printer/tcp.go` implementing the `Printer` interface. Dial `host:port`, send ESC/POS commands, close connection. The existing `PrinterManager` and `PrinterProvider` pattern makes this straightforward.

**New env var**: `PRINTER_TCP_ADDR` (e.g., `192.168.1.100:9100`)
**Files**: `pkg/printer/tcp.go` (new), `pkg/printer/printer.go` (register type)
**Effort**: Low-Medium (follows existing USB pattern)
**Benefit**: Covers network printer deployments without hardware changes.

### 8. Prometheus Metrics Endpoint (`GET /metrics`)

**Problem**: Spooler metrics are tracked in a thread-safe struct (`pkg/spooler/metrics.go`) but only exposed via the admin API (`GET /api/admin/metrics`). Operators using Prometheus for monitoring can't scrape this data.

**Fix**: Expose `GET /metrics` in Prometheus text format with:
- `zerodrop_spooler_queue_depth`
- `zerodrop_spooler_total_processed`
- `zerodrop_spooler_total_failed`
- `zerodrop_spooler_print_duration_seconds`

**Files**: `pkg/api/server.go` (new handler)
**Effort**: Low (data already exists, just needs formatting)
**Benefit**: Integration with existing monitoring infrastructure.

### 9. Frontend Error Boundaries

**Problem**: The React app crashes silently on uncaught errors. If `init()` fails (e.g., Web Crypto API unavailable on HTTP), the user sees a white screen or cryptic console error instead of a helpful message.

**Fix**: Wrap the app in a React Error Boundary component with:
- Clear error message explaining what went wrong
- Recovery button ("Retry" or "Reload")
- Technical details hidden behind expandable section

**Files**: New `frontend/src/components/ErrorBoundary.tsx`, `frontend/src/App.tsx`
**Effort**: Low
**Benefit**: Better UX when things go wrong — users see an explanation instead of a blank screen.

### 10. Canary/Batch QR Splitting

**Problem**: The 400-char payload limit (~185 chars plaintext) restricts what can be delivered. SSH private keys, PGP keys, and full security reports exceed this limit.

**Fix**: Split large payloads across multiple QR codes (chunked mode):
- Server emits N sequential QR codes
- Each QR has a header: `ZD1:chunk-N-of-M:base64(...)`
- `reader.html` scans sequentially and reassembles

**Files**: `pkg/qr/qr.go`, `pkg/spooler/spooler.go`, `static/reader.html`
**Effort**: Medium-High (touches all 3 layers)
**Benefit**: Enables delivery of any-size credential material.

### 11. Print Notification Webhook

**Problem**: Operators have no way to be notified when a print job completes. They must manually check the admin dashboard or server logs.

**Fix**: When a print succeeds/fails, fire a configurable webhook (HTTP POST) with metadata:
```json
{ "job_id": "...", "timestamp": "...", "status": "success|failed", "payload_size": 123 }
```
No payload content ever leaves the server.

**Files**: `pkg/spooler/spooler.go` (webhook call after job), `pkg/config/config.go` (new `WEBHOOK_URL` env var)
**Effort**: Medium
**Benefit**: Enables Slack/Teams notifications, SIEM integration, or downstream automation.

### 12. QR Preview Mode ("Try It")

**Problem**: Operators waste thermal paper testing payload formatting during setup or debugging.

**Fix**: Add `GET /preview?payload=ZD1:...` that returns the QR code as an inline HTML `<img>` (PNG base64) or SVG. No printing, no spooler — just visual preview in the browser.

**Files**: `pkg/api/server.go` (new handler using existing `pkg/qr`), `frontend/src/App.tsx` (preview button)
**Effort**: Low-Medium (reuses existing `GenerateQRPNG`)
**Benefit**: Eliminates wasted paper during testing.

---

## Already Considered (From v1.2 Recommendations Doc)

These items from `docs/plans/zero-drop-v1.2-upgrade-recommendations.md` remain valid but are lower priority:

| Item | Why Hold |
|------|----------|
| `gorilla/mux` → stdlib `http.ServeMux` | Works fine. Dropping a dependency is nice but not urgent. |
| `Makefile` lint target (`go vet` + `staticcheck` + `govulncheck`) | Useful but `make check-security` already covers most of it. |
| Context propagation through spooler | Print-after-disconnect is intentional behavior. Not a bug. |
| Batch QR splitting | Listed as P3 item 10 above. |
| Recipient lockbox (per-recipient keys) | Architecture change, high effort, unclear demand. |
| Dead man's switch / timed payload | Novel feature but niche use case. |
| Prometheus `/metrics` | Listed as P3 item 8 above. |

---

## Explicitly Not Recommended

| Idea | Why Not |
|------|---------|
| Switch to a database | Violates ephemeral/zero-knowledge architecture. RAM-only is a feature. |
| Add user auth system | Admin token is appropriate for LAN/air-gapped. User auth adds complexity without value. |
| External crypto dependencies | Web Crypto API + Go stdlib is battle-tested. More deps = more attack surface. |
| Kubernetes orchestration | Overkill for a single-service terminal with USB printer. |
| Full TypeScript end-to-end types | Works fine as-is. Shared types package is maintenance overhead at this scale. |

---

## Recommendation Priority Summary

```
P0 ── Printer hang protection          ⬅ Do this first
P1 ── Frontend crypto tests            ⬅ Security blind spot
P1 ── Login rate limiter cleanup       ⬅ Cheap fix, prevents leak
P2 ── Integration tests                ⬅ Catches wiring bugs
P2 ── go:embed frontend assets         ⬅ Single binary deployment
P2 ── Printer health timeout           ⬅ Keeps health endpoint responsive
P3 ── TCP printer support              ⬅ Network deployment coverage
P3 ── Prometheus /metrics              ⬅ Monitoring integration
P3 ── Frontend error boundaries        ⬅ Better UX on failure
P3 ── Batch QR splitting               ⬅ Large credential support
P3 ── Print webhook                    ⬅ Operator notifications
P3 ── QR preview mode                  ⬅ Saves paper during testing
```
