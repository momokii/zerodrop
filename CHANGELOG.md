# Changelog

All notable changes to ZeroDrop Terminal are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] — 2026-06-09

### Added
- **Admin Dashboard** (`/admin`): Real-time spooler monitoring, printer management,
  key management (fingerprint, download, rotate)
- **Persistent Key Pair**: Private key saved to disk (`0600`) on first run,
  reused across restarts. Burn Protocol still runs on first boot.
- **Spooler Metrics**: Thread-safe tracking of queue depth, processed/failed counts,
  print duration
- **PrinterManager**: Auto-detect all connected printers, switch active printer
  at runtime via admin API
- **Admin Authentication**: `ADMIN_TOKEN` env var, session cookies
  (HttpOnly, SameSite), constant-time comparison, login rate limiting
- **Admin API**: 8 endpoints at `/api/admin/*` for monitoring and management
- **Key Grant Step-Up Auth**: Re-authentication required for sensitive operations
- **Private Key QR Scan**: Camera-based private key import in `reader.html`
- **Configurable Session TTL**: `ADMIN_SESSION_TTL` env var (default: `24h`)
- **Security Verification Suite**: 50 automated security checks via `make check-security`
- **Custom Modal Dialogs**: Replaced `confirm()` with custom modals in reader and admin

### Changed
- **Key lifecycle**: Ephemeral (v1.0) → Persistent (v1.1). Keys survive restarts.
- **Docker deployment**: Traefik removed, Docker-only deployment
- **Reader cache busting**: Added `/reader-v2.html` route with no-cache headers
- **Payload display**: Human-readable elapsed time in admin metrics

### Security
- 50 automated security checks verifying zero-knowledge guarantees
- Login rate limiting: 10 attempts per 15 minutes per IP
- Admin session cookies: HMAC-signed, HttpOnly, SameSite
- Key rotation available via `KEY_ROTATE=true` or admin dashboard

### Fixed
- SPA handler no longer silently swallows unregistered API routes
- Printer interface deduplication across packages
- Admin logout properly invalidates server-side session
- USB printer detection in admin dashboard
- Key rotation path and error handling

## [1.0.0] — 2026-05-12

### Added
- **Zero-knowledge encryption**: X25519 ECDH + AES-256-GCM (ECIES) across all layers
- **Go backend**: HTTP server with `GET /key`, `POST /drop`, `GET /health` endpoints
- **React frontend**: Submission portal with Web Crypto API encryption
- **Offline reader**: `static/reader.html` with jsQR camera scanning
- **ESC/POS thermal printer support**: USB auto-detection for 10+ printer models
- **Asynchronous spooler**: Buffered Go channel with retry logic (3 attempts, exponential backoff)
- **Burn Protocol**: Private key zeroed from RAM with `runtime.KeepAlive()`
- **Graceful shutdown**: 30-second spooler drain timeout on SIGTERM/SIGINT
- **Rate limiting**: Per-IP sliding window (5 req/hr default)
- **Structured logging**: Opt-in JSON logging (`LOG_ENABLED`)
- **SPI public key format**: SPKI DER for Web Crypto API compatibility
- **Docker Compose**: Multi-stage Alpine build, non-root user, USB device mapping
- **Makefile**: Development automation (build, test, deploy, ops)
- **Deployment Makefile**: Production operations (deploy, backup, security checks)

### Security
- Server never possesses plaintext or private key
- All payload buffers zeroed after print job
- `ZD1:` version header for forward compatibility
- SHA-256 key fingerprinting for operator verification
- Rate limiting at application level
