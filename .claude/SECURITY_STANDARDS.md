# Security Standards — ZeroDrop Terminal

> **Mandatory reference document** governing all security decisions for ZeroDrop Terminal v1.1.
> **Zero-knowledge architecture**: Server must never possess plaintext payload or private key.

---

## Zero-Knowledge Guarantee (Non-Negotiable)

- **Server must never possess plaintext payload** — ciphertext is queued and printed as-is
- **Server must never possess private key in RAM after initial setup** — v1.1 saves to disk on first run, only loaded during first-run QR display, then burned
- **No decryption capability** — server cannot decrypt messages even if compromised
- **No database** — ephemeral RAM-only processing, no persistence (v1.1 persistent key file is not a database — it is a PEM file with 0600 permissions)
- **Burn Protocol** — all sensitive memory must be zeroed with `runtime.KeepAlive()`

---

## Secrets & Environment Variable Management

- **Never hardcode secrets, API keys, tokens, passwords** — not even in test files
- All secrets managed via **environment variables** loaded from `.env` files
- **`.env` files excluded from version control** via `.gitignore`
- **`.env.example` must exist** with all variable names, placeholders, and descriptions
- **Never log, print, or expose** environment variable values
- **Verify `.env` is listed in `.gitignore`** before writing code that reads it

### ZeroDrop-specific variables

| Variable | Required | Description |
|----------|----------|-------------|
| `PRINTER_TYPE` | Yes | `mock` or `usb` |
| `PRINTER_DEVICE` | If USB | Device path (e.g., `/dev/usb/lp0`) |
| `PUBLIC_KEY_PATH` | No | Path to save public key PEM (default: `./data/public_key.pem`) |
| `PRIVATE_KEY_PATH` | No | Path to save/load private key PEM (default: `./data/private_key.pem`) |
| `ADMIN_TOKEN` | For admin | Shared secret for admin dashboard auth (set in `.env`) |
| `ADMIN_SESSION_TTL` | No | Admin session lifetime (Go duration, default: `24h`) |
| `KEY_ROTATE` | No | Set `true` to force key pair regeneration on next startup (default: `false`) |
| `TLS_ENABLED` | No | Enable built-in self-signed HTTPS (default: `false`) |
| `RATE_LIMIT_REQUESTS_PER_HOUR` | No | Per-IP API rate limit (default: 5) |
| `LOG_ENABLED` | No | Structured logging opt-in (default: false) |

---

## Environment Configuration

- **Three environments**: `development`, `staging`, `production`
- **`APP_ENV` variable** controls environment-specific behavior
- **Configuration driven by environment variables** — never hardcoded conditionals
- **Debug mode off by default** — only enabled via explicit environment variable
- **Production uses Mock Printer** if USB not configured (fallback for testing)

---

## Go-Specific Cryptography

### Approved Crypto Stack

- **Key exchange**: X25519 via `crypto/ecdh` (Curve25519)
- **Random generation**: `crypto/rand` — NEVER `math/rand`
- **Hashing**: SHA-256 via `crypto/sha256`
- **Constant-time comparison**: `crypto/subtle.ConstantTimeCompare()`

### Prohibited Patterns

- **Never implement custom crypto** — use stdlib only
- **Never use deprecated algorithms** — MD5, SHA1, RC4, DES
- **Never use `math/rand` for security** — only `crypto/rand`
- **Never hardcode IVs, nonces, or keys** — always generate randomly

### Burn Protocol Implementation

```go
// Critical: Use runtime.KeepAlive() to prevent compiler optimization
func BurnProtocol(buf []byte) {
    for i := range buf {
        buf[i] = 0
    }
    runtime.KeepAlive(buf) // Prevents optimization
}
```

---

## Input Validation & Sanitization

- **All external input validated at boundary layer** — HTTP handlers before business logic
- **Never trust client-supplied data** for authorization decisions
- **Reject and return clear error** for invalid input — no silent coercion
- **ZeroDrop-specific validations**:
  - `POST /drop` payload: max 400 characters, valid base64, optional `ZD1:` prefix
  - `PRINTER_DEVICE`: must exist and be writable for USB type
- **Use validation library** — consider `go-playground/validator` for complex validation

---

## HTTP & API Security

### Required Headers

```go
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("Content-Security-Policy", "default-src 'self'")
```

### Endpoint Security

- **`GET /key`**: Returns public key (no auth, read-only)
- **`POST /drop`**: Accepts encrypted payload (no decryption, rate-limited)
- **`GET /health`**: Health check (no auth, read-only)
- **`/api/admin/*`**: 8 admin endpoints (session auth required, login rate-limited)

### Rate Limiting

- **Global API rate limiter**: Built-in per-IP sliding window (5 req/hr default, `/health` excluded)
- **Admin login rate limiter**: 10 attempts per 15 minutes per IP (separate from global limiter)
- **No authentication on public endpoints** — zero-knowledge design prevents user tracking
- **Degrade gracefully** — return 429 when rate-limited

### Admin Authentication (v1.1)

- **Token**: `ADMIN_TOKEN` env var, compared with `crypto/subtle.ConstantTimeCompare` (timing-attack resistant)
- **Session**: HMAC-signed cookie, `HttpOnly`, `SameSite=Lax`, scoped to `/api/admin` path
- **Expiry**: 24 hours from login, server-side tracking
- **Login rate limit**: 10 attempts per 15 minutes per source IP
- **Key download**: `GET /api/admin/key` and `GET /api/admin/key/qr` expose private key — acceptable because admin has server disk access
- **Key rotation**: `POST /api/admin/key/rotate` forces new key pair generation (admin-only)

---

## Dependency Security

### Before Adding Dependencies

- **Run `govulncheck ./...`** to check for known vulnerabilities
- **Prefer stdlib** — only add external deps when absolutely necessary
- **Well-maintained packages** — check recent commits, issues, PRs
- **Pin versions** — avoid open-ended ranges

### Current Dependencies

| Package | Version | Purpose | Security Audit |
|---------|---------|---------|----------------|
| `github.com/gorilla/mux` | v1.8.1 | HTTP router | Passed govulncheck |
| `github.com/skip2/go-qrcode` | v0.0.0-20200617195104 | QR generation | Passed govulncheck |

### Adding New Dependencies

1. Run `go get <package>@<version>`
2. Run `govulncheck ./...`
3. Log decision in `state/DECISIONS_LOG.md`
4. Update `SECURITY_STANDARDS.md` table above

---

## Memory Security

### Buffer Zeroing

- **All payload buffers zeroed after print job** — spooler responsibility
- **Private key zeroed after QR logging** — Burn Protocol (v1.1: applies after first-run QR display, subsequent starts do not load private key to RAM)
- **Use `runtime.KeepAlive()`** to prevent compiler optimization
- **No sensitive data in logs** — never log keys, ciphertext, plaintext

### Persistent Key Storage (v1.1)

- **Private key saved to `data/private_key.pem` on first run** with 0600 permissions
- **Subsequent starts reuse key pair** — private key NOT loaded to RAM
- **`KEY_ROTATE=true` forces regeneration** — old key file is overwritten
- **Docker**: Non-root user + `no-new-privileges` protects key file
- **Tradeoff accepted**: For LAN/air-gapped deployment, disk access implies physical control. Ephemeral keys would risk data loss on crash.

### Memory Safety

- **Run tests with `-race` flag** — detect data races
- **Use `defer` for cleanup** — ensure resources released
- **Avoid `unsafe` package** — unless absolutely necessary

---

## Logging & Monitoring

### Logging Rules

- **Never log sensitive data** — no keys, ciphertext, plaintext
- **Log security events** — auth failures, input validation failures, rate limit hits
- **Structured logging** — JSON format for parsing
- **Opt-in only** — `LOG_ENABLED=false` by default

### Safe Logging Example

```go
// ✅ Safe: No sensitive data
log.Printf("[api] drop request received from %s, payload length: %d", r.RemoteAddr, len(payload))

// ❌ Unsafe: Logs ciphertext
log.Printf("[api] drop request with payload: %s", string(payload))
```

---

## Hardware Security (USB Printer)

- **Device permissions** — configure udev rules, avoid privileged containers
- **Device validation** — check device exists and is writable on startup
- **Fallback to Mock Printer** — if USB unavailable, fail gracefully
- **No privileged mode** — use device mapping in Docker Compose

---

## Docker & Container Security

- **Never run as root** — use non-root user in Dockerfile
- **Do not expose unnecessary ports** — only HTTP port
- **Never commit `.env` files** — use Docker secrets or env injection
- **Use specific image tags** — never `latest` in production
- **Scan base images** — use `docker scan` or Trivy
- **Minimize image size** — multi-stage builds, slim base images
- **Device mapping for USB** — use `devices:` in docker-compose, NOT `privileged: true`

---

## Web Crypto API (Frontend)

- **Browser-native crypto only** — Web Crypto API, no external libraries
- **Client-side encryption** — plaintext never leaves browser
- **No external CDN deps** — self-contained `reader.html`
- **HTTPS required** — for production deployment

---

## Security Testing

- **Unit tests** — all crypto functions, input validation
- **Integration tests** — API endpoints with mock printer
- **Race detection** — `go test -race ./...`
- **Vulnerability scanning** — `govulncheck ./...` before commits
- **Manual security review** — before M-05 (production readiness)

---

## Incident Response (Future)

- **Security issues** — report via GitHub Security Advisories
- **Key compromise** — generate new key pair, redistribute public key
- **Server compromise** — zero-knowledge design limits exposure (ciphertext only)
