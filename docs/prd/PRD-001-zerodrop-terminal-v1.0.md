# PRD: ZeroDrop Terminal v1.0 — Initial Implementation

**Version:** 1.1
**Status:** Revised
**Last Updated:** 2026-05-11

## Revision History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2026-05-11 | Initial PRD creation |
| 1.1 | 2026-05-11 | Revised frontend stack: Vanilla JS → React + Vite + shadcn/ui |

---

## 1. Executive Summary

ZeroDrop Terminal v1.0 is the initial implementation of an air-gapped, zero-knowledge secure credential delivery terminal. The system enables users to encrypt sensitive data (passwords, API keys, security reports) in their browser, transmit it to a server that cannot decrypt it, and receive the ciphertext as a physical QR code printout. Recipients decrypt the printout offline using a standalone HTML file and a private key that never touched the server.

This PRD covers building the entire system from the ground up: cryptographic backend (Go), hardware abstraction layer (ESC/POS thermal printer), asynchronous spooler, submission portal (React + shadcn/ui), offline reader, and infrastructure (Docker Compose + Traefik). The zero-knowledge guarantee is non-negotiable: the server must never possess the plaintext payload or the private key at any point in the data flow.

---

## 2. Problem Statement

### The Gap

Highly sensitive data (root passwords, API keys, breach reports) must be transmitted securely. Traditional methods suffer from critical flaws:

- **Database persistence**: Encrypted or not, stored data is a high-value target for insider threats and server compromise.
- **Transmission risk**: Network interception, man-in-the-middle attacks, and TLS termination points introduce vulnerability.
- **Recipient friction**: Complex mobile apps, dependency chains, and online requirements create barriers for recipients in air-gapped environments.

### Cost of Not Building This

- **Security exposure**: Without ZeroDrop, organizations must choose between (a) storing sensitive data persistently (creating attack surface) or (b) using manual, error-prone physical handoff methods.
- **Operational pain**: Current secure delivery methods often require specialized software, complex key exchange ceremonies, or online dependencies.
- **Scalability ceiling**: Physical courier methods do not scale; database-dependent methods do not meet zero-trust requirements.

### Why Now

Threat models have evolved. Server compromise is no longer a theoretical risk — it is an operational reality. ZeroDrop's zero-knowledge architecture ensures that even a root-level server breach yields no useful data: intercepted ciphertext is mathematically worthless without the private key, which never exists on the server.

---

## 3. Goals & Non-Goals

### Goals

1. **Zero-knowledge guarantee**: The server must never possess the plaintext payload or the private key at any point in processing, memory, or storage.
2. **Hardware-optimized cryptography**: Use ECC (Curve25519) via `crypto/ecdh` to produce compact ciphertext that reliably scans as QR codes on 58mm thermal printers.
3. **Ephemeral processing**: No database persistence. Data exists in RAM only for the duration of the print job, then is explicitly zeroed.
4. **Frictionless offline decryption**: A single `reader.html` file that decrypts QR codes completely offline using Web Crypto API, with zero external dependencies.
5. **Production-ready reliability**: Graceful shutdown, hardware failure recovery, and operational visibility through structured logging.
6. **Containerized portability**: Full Docker Compose setup with proper device mapping and permissions for USB printer passthrough.

### Non-Goals

- **Multi-printer support**: v1.0 supports a single thermal printer. Multi-printer orchestration is deferred to v1.1+.
- **Batch submission**: v1.0 supports single-payload submission. Batch API is deferred to v1.1+.
- **Admin dashboard**: v1.0 has no web-based administration interface. Configuration is via environment variables only.
- **Persistent audit logs**: v1.0 uses RAM-only ephemeral processing. Persistent audit trails are deferred to v1.2+.
- **TCP printer support**: v1.0 supports USB printers and Mock Printer. TCP network printers are deferred to v1.1+.
- **Key rotation**: v1.0 generates a single key pair on first boot. Key rotation mechanisms are deferred to v1.2+.

---

## 4. User Stories

| ID | User Story | Role |
|----|------------|------|
| US-001 | As a **Security Administrator**, I want to provision a new ZeroDrop instance so that I can establish a secure, zero-knowledge credential delivery channel. | Admin |
| US-002 | As a **Security Administrator**, I want to receive the private key as a scannable QR code in the terminal logs so that I can securely archive it for offline decryption. | Admin |
| US-003 | As a **Security Administrator**, I want to verify that the server has shredded the private key from memory so that I can confirm the zero-knowledge guarantee is intact. | Admin |
| US-004 | As a **Submitter**, I want to encrypt my payload in the browser before transmission so that the server never sees my plaintext. | Operator |
| US-005 | As a **Submitter**, I want to receive immediate feedback that my payload was queued for printing so that I can close the browser tab without waiting. | Operator |
| US-006 | As a **Submitter**, I want to see clear validation errors if my payload exceeds the character limit so that I can correct it before submission. | Operator |
| US-007 | As a **Recipient**, I want to decrypt a printed QR code using a standalone HTML file on an air-gapped machine so that I can retrieve my credential without internet access. | Recipient |
| US-008 | As a **Recipient**, I want to verify that I am using the correct private key so that I can avoid decryption errors from key mismatch. | Recipient |
| US-009 | As a **System Operator**, I want to monitor the system health via a `/health` endpoint so that my orchestration system can detect failures. | Operator |
| US-010 | As a **System Operator**, I want the system to shut down gracefully when receiving SIGTERM so that in-flight print jobs complete before the container exits. | Operator |

---

## 5. Functional Requirements

| ID | Functional Requirement | Satisfies | Module |
|----|------------------------|-----------|--------|
| FR-001 | On first boot, the system shall generate an ECC key pair using Curve25519 (`X25519`) via `crypto/ecdh`. | US-001 | `pkg/crypto` |
| FR-002 | The system shall save the public key to `public_key.pem` in the configured volume mount. | US-001 | `pkg/crypto` |
| FR-003 | The system shall log the private key as a scannable QR code to stdout, prefixed with `PRIVATE_KEY_QR:`, then immediately shred the private key from memory and disk. | US-002, US-003 | `pkg/crypto` |
| FR-004 | The private key shredding shall use explicit memory zeroing with `runtime.KeepAlive()` to prevent compiler optimization. | US-003 | `pkg/crypto` |
| FR-005 | The system shall expose `GET /key` which returns the `public_key.pem` file as `text/plain` with HTTP 200. | US-001 | `pkg/api` |
| FR-006 | The system shall expose `POST /drop` which accepts JSON `{ "payload": "<base64-ciphertext>" }` and returns HTTP 202 Accepted immediately. | US-004, US-005 | `pkg/api` |
| FR-007 | The `POST /drop` endpoint shall validate that the payload length is ≤ 250 characters and return HTTP 400 if exceeded. | US-006 | `pkg/api` |
| FR-008 | The `POST /drop` endpoint shall validate that the payload is valid base64 and return HTTP 400 if invalid. | US-004 | `pkg/api` |
| FR-009 | The `POST /drop` endpoint shall push the payload to a buffered Go channel (spooler) and return immediately without waiting for print completion. | US-005 | `pkg/api`, `pkg/spooler` |
| FR-010 | The spooler shall process payloads sequentially using a worker pool, pulling from the channel and passing each payload to the `Printer` interface. | US-005 | `pkg/spooler` |
| FR-011 | The `Printer` interface shall accept a byte slice (ciphertext) and output ESC/POS commands for rasterizing and printing a QR code. | US-005 | `pkg/printer` |
| FR-012 | The Mock Printer implementation shall write the ESC/POS commands to stdout for testing and CI. | US-005 | `pkg/printer` |
| FR-013 | The USB Printer implementation shall open the device file (e.g., `/dev/usb/lp0`), write the ESC/POS commands, and handle I/O errors. | US-005 | `pkg/printer` |
| FR-014 | The QR code generation shall use a standard library (e.g., `github.com/skip2/go-qrcode`) to produce a monochrome byte matrix. | US-005 | `pkg/printer` |
| FR-015 | The ESC/POS rasterization shall use `GS v 0` (bit-image) commands to print the QR code. | US-005 | `pkg/printer` |
| FR-016 | The printer shall send a paper cut command (`GS V m 66`) after each QR code to separate prints. | US-005 | `pkg/printer` |
| FR-017 | After successful transmission to the printer, the spooler shall explicitly zero the payload buffer in RAM using `for i := range buf { buf[i] = 0 }` followed by `runtime.KeepAlive(buf)`. | US-003 | `pkg/spooler` |
| FR-018 | The submission portal shall be a React application built with Vite, using shadcn/ui components for the UI. | US-004 | Frontend |
| FR-019 | The submission portal shall use `window.crypto.subtle` for client-side encryption (no crypto libraries). | US-004 | Frontend |
| FR-020 | The submission portal shall fetch `public_key.pem` from `GET /key` and import it using `window.crypto.subtle.importKey("spki", ...)`. | US-004 | Frontend |
| FR-021 | The submission portal shall encrypt the plaintext payload using ECDH (Curve25519) and export the ciphertext as base64. | US-004 | Frontend |
| FR-022 | The submission portal shall prefix the base64 ciphertext with `ZD1:` (version header) before transmission. | US-004 | Frontend |
| FR-023 | The submission portal shall validate the payload length is ≤ 185 characters of plaintext before encryption and display an error using shadcn/ui Alert component if exceeded. | US-006 | Frontend |
| FR-024 | The submission portal shall submit the payload to `POST /drop` and display status using shadcn/ui components: "Encrypting...", "Transmitting...", "Safely Dropped." | US-005 | Frontend |
| FR-024 | The `reader.html` file shall be a standalone HTML file with embedded Vanilla JS that decrypts QR codes offline using Web Crypto API. | US-007 | `reader.html` |
| FR-025 | The `reader.html` file shall access the device webcam using `navigator.mediaDevices.getUserMedia()` to scan QR codes. | US-007 | `reader.html` |
| FR-026 | The `reader.html` file shall parse the QR code, strip the `ZD1:` prefix, and decode the remaining base64 to ciphertext. | US-007 | `reader.html` |
| FR-027 | The `reader.html` file shall prompt the user to paste the private key (in PEM format) and import it using `window.crypto.subtle.importKey("pkcs8", ...)`. | US-007, US-008 | `reader.html` |
| FR-028 | The `reader.html` file shall decrypt the ciphertext using ECDH (Curve25519) and display the plaintext on screen. | US-007 | `reader.html` |
| FR-029 | The `reader.html` file shall compute and display a SHA-256 fingerprint of the imported public key for verification against the server's logged fingerprint. | US-008 | `reader.html` |
| FR-030 | The system shall expose `GET /health` which returns HTTP 200 if the system is operational, HTTP 503 if the printer is disconnected. | US-009 | `pkg/api` |
| FR-031 | The system shall log the SHA-256 fingerprint of the public key on startup in the format `PUBLIC_KEY_FINGERPRINT: <hex-string>`. | US-008 | `pkg/crypto` |
| FR-032 | The system shall handle SIGTERM and SIGINT signals by stopping acceptance of new requests, waiting for the spooler to drain (max 30 seconds), and then exiting. | US-010 | `pkg/spooler` |
| FR-033 | The system shall validate required environment variables on startup and exit with error code 1 if any are missing or invalid. | US-001 | `pkg/config` |
| FR-034 | Required environment variables are: `PRINTER_DEVICE` (path to device file), `PRINTER_TYPE` (one of: `mock`, `usb`). | US-001 | `pkg/config` |
| FR-035 | Optional environment variables are: `RATE_LIMIT_REQUESTS_PER_HOUR` (default: 5), `RATE_LIMIT_BURST` (default: 1), `LOG_ENABLED` (default: false). | US-001 | `pkg/config` |
| FR-036 | The system shall validate that `PRINTER_DEVICE` exists and is writable if `PRINTER_TYPE=usb`. | US-001 | `pkg/config` |
| FR-037 | The system shall use structured JSON logging (when `LOG_ENABLED=true`) and never log private keys, payloads, or submission metadata. | US-001 | `pkg/observability` |
| FR-038 | The Traefik reverse proxy shall rate-limit requests to 5 per IP per hour with burst of 1. | US-001 | Infrastructure |
| FR-039 | The Docker Compose configuration shall map the printer device (`/dev/usb/lp0`) into the container with `group_add: [dialout, lp]` for permissions. | US-001 | Infrastructure |

---

## 6. Non-Functional Requirements

### Security (NFR-S-XXX)

| ID | Requirement |
|----|-------------|
| NFR-S-001 | The zero-knowledge guarantee shall be preserved: the server never holds, derives, or accesses the plaintext payload or the private key at any point. |
| NFR-S-002 | The Burn Protocol shall be executed immediately after logging the private key QR: zero the byte slice, then call `runtime.KeepAlive()` to prevent optimization. |
| NFR-S-003 | All cryptographic operations shall use `crypto/ecdh` and `crypto/rand` exclusively. The `math/rand` package shall not be used. |
| NFR-S-004 | The private key shall never be written to disk, logged, or persisted in any form beyond the initial terminal QR code output. |
| NFR-S-005 | Memory hygiene: after print job completion, the payload buffer in the spooler shall be zeroed explicitly with `runtime.KeepAlive()`. |
| NFR-S-006 | The `POST /drop` endpoint shall not return any information about whether the ciphertext was successfully decrypted or not (no padding oracle). |
| NFR-S-007 | The `ZD1:` version prefix in QR payloads allows future format changes while maintaining backward compatibility of `reader.html`. |
| NFR-S-008 | Key fingerprinting (SHA-256 of public key) provides operators with a verification mechanism against key substitution attacks. |
| NFR-S-009 | The `reader.html` file shall perform all decryption operations locally in the browser using Web Crypto API with no network requests after initial load. |
| NFR-S-010 | The submission portal shall perform all encryption operations locally in the browser using Web Crypto API before any network transmission. |
| NFR-S-011 | Rate limiting (5 req/IP/hour) mitigates DDoS and hardware exhaustion attacks. |
| NFR-S-012 | The 250-character payload limit (246 after `ZD1:` prefix) prevents QR density that exceeds scanner capabilities. |
| NFR-S-013 | Structured logging shall explicitly exclude: private keys, payload plaintext, payload ciphertext, submission IP addresses, submission timestamps. |

### Performance (NFR-P-XXX)

| ID | Requirement |
|----|-------------|
| NFR-P-001 | The `POST /drop` endpoint shall return HTTP 202 within 100ms for valid requests (excluding network latency). |
| NFR-P-002 | The spooler shall process print jobs sequentially with no more than 5 seconds of queue wait time under normal load (≤ 5 concurrent submissions). |
| NFR-P-003 | The browser encryption process shall complete within 500ms for payloads ≤ 185 characters on modern hardware. |
| NFR-P-004 | The `reader.html` decryption process shall complete within 500ms for standard payloads on modern hardware. |
| NFR-P-005 | The system shall handle up to 5 concurrent submissions without degradation (Traefik rate limit). |

### Hardware Compatibility (NFR-H-XXX)

| ID | Requirement |
|----|-------------|
| NFR-H-001 | The system shall support 58mm thermal printers with ESC/POS command set compatibility. |
| NFR-H-002 | The system shall support USB printer connectivity via `/dev/usb/lp0` device mapping. |
| NFR-H-003 | The system shall support Mock Printer mode for testing and CI environments. |
| NFR-H-004 | The QR code generation shall produce version 10 or lower QR codes (≤ 40x40 modules) for reliable scanning on low-resolution hardware. |
| NFR-H-005 | The system shall handle printer offline conditions gracefully: log error, return 503 on `/health`, continue accepting submissions (spooler buffers). |
| NFR-H-006 | The system shall handle USB disconnect/reconnect events: log warning, mark printer unavailable, attempt reconnection on next print job. |

### Reliability (NFR-R-XXX)

| ID | Requirement |
|----|-------------|
| NFR-R-001 | If the printer goes offline mid-job, the spooler shall log the error and retain the job in the queue for retry. |
| NFR-R-002 | The spooler shall retry failed print jobs up to 3 times with exponential backoff (1s, 2s, 4s) before giving up. |
| NFR-R-003 | The graceful shutdown handler shall wait up to 30 seconds for the spooler to drain before forcing exit. |
| NFR-R-004 | If the spooler does not drain within 30 seconds during shutdown, the system shall log a warning and exit with a non-zero code. |
| NFR-R-005 | The system shall not crash on malformed input; all error paths shall return appropriate HTTP status codes (400, 500, 503). |
| NFR-R-006 | The system shall recover from printer power cycles: detect reconnection, resume processing spooler queue. |

### Deployability (NFR-D-XXX)

| ID | Requirement |
|----|-------------|
| NFR-D-001 | The system shall be deployable via `docker-compose up` with no additional host configuration beyond device permissions. |
| NFR-D-002 | The system shall run on Linux kernel 5.0 or higher with Docker 20.10+ and Docker Compose 2.0+. |
| NFR-D-003 | The system shall not require Docker privileged mode; device access shall be granted via `group_add` and `devices` mapping. |
| NFR-D-004 | The system shall validate all required environment variables on startup and exit with a clear error message if invalid. |
| NFR-D-005 | The system shall be stateless except for the `public_key.pem` file, which is persisted to a Docker volume. |
| NFR-D-006 | The system shall log to stdout for Docker log aggregation; structured JSON logging shall be opt-in via `LOG_ENABLED`. |

---

## 7. Technical Design Notes

### Affected Modules

| Module | Status | Summary of Change or Deprecation Path |
|--------|--------|----------------------------------------|
| `pkg/crypto` | **New** | Implements ECC key generation (Curve25519), key persistence, private key logging as QR, Burn Protocol with memory zeroing. |
| `pkg/api` | **New** | Implements HTTP server with `GET /key`, `POST /drop`, `GET /health` endpoints. Rate limiting via Traefik. |
| `pkg/printer` | **New** | Defines `Printer` interface. Implements Mock Printer (stdout) and USB Printer (`/dev/usb/lp0`). QR generation and ESC/POS rasterization. |
| `pkg/spooler` | **New** | Implements buffered channel-based worker pool for sequential print job processing. Memory zeroing after job completion. |
| `pkg/config` | **New** | Validates environment variables on startup. Defines required and optional configuration. |
| `pkg/observability` | **New** | Implements structured JSON logging (opt-in). Health check logic. Graceful shutdown handler. |
| `reader.html` | **New** | Standalone HTML file for offline QR decryption using Web Crypto API. Webcam access, private key import, fingerprint display. |
| Frontend (submission portal) | **New** | React + Vite application with shadcn/ui components. Client-side encryption using Web Crypto API, `ZD1:` prefix, validation, status feedback. |
| Infrastructure | **New** | Docker Compose configuration, Traefik reverse proxy with rate limiting, device mapping for USB printer. |

### New Interfaces or Contracts

#### Go Interface: `Printer`

```go
type Printer interface {
    Print(ciphertext []byte) error
    IsAvailable() bool
}
```

- **`Print(ciphertext []byte) error`**: Accepts ciphertext, generates QR code, rasterizes to ESC/POS, writes to output device. Returns error if print fails.
- **`IsAvailable() bool`**: Returns true if printer is connected and ready, false otherwise.

#### HTTP Endpoints

| Endpoint | Method | Request | Response | Error Codes |
|----------|--------|---------|----------|-------------|
| `/key` | GET | None | `text/plain` containing PEM-encoded public key | 500 (server error) |
| `/drop` | POST | `{"payload": "<base64-ciphertext>"}` | `202 Accepted` | 400 (invalid payload, length exceeded), 500 (server error) |
| `/health` | GET | None | `200 OK` (healthy), `503 Service Unavailable` (printer offline) | 500 (server error) |

#### QR Payload Format

```
ZD1:<base64-encoded-ciphertext>
```

- `ZD1:`: Version header (4 characters).
- `<base64-encoded-ciphertext>`: Base64-encoded ciphertext from ECDH encryption.
- Total length ≤ 250 characters.

#### Key Provisioning & Burn Protocol

1. **Startup**: Generate ECC key pair using `crypto/ecdh`.
2. **Persist Public Key**: Save to `public_key.pem` in volume.
3. **Log Private Key**: Print private key as QR to stdout with prefix `PRIVATE_KEY_QR:`.
4. **Burn Protocol**:
   a. Zero the private key byte slice: `for i := range key { key[i] = 0 }`
   b. Call `runtime.KeepAlive(key)` to prevent compiler optimization.
   c. Delete any temporary files.

### Data Flow Changes

**Baseline**: HLD Section 5 describes the end-to-end data flow. v1.0 implements this flow exactly.

| Phase | Change |
|-------|--------|
| **Initialization** | No change. Follows HLD exactly. |
| **Submission** | No change. Follows HLD exactly. |
| **Processing** | No change. Follows HLD exactly. |
| **Retrieval** | No change. Follows HLD exactly. |

### API Contract Changes

**Baseline**: HLD Section 5-B describes the `POST /drop` and `GET /key` endpoints. v1.0 implements these endpoints as specified.

| Endpoint | Change |
|----------|--------|
| `GET /key` | **New**. Returns public key as `text/plain`. |
| `POST /drop` | **New**. Accepts JSON payload, returns 202. |
| `GET /health` | **New**. Health check endpoint for orchestration. |

**Breaking Changes**: None. This is initial implementation.

**Backward Compatibility**: N/A (initial implementation).

### Dependency Changes

| Dependency | Version | Purpose | Supply Chain Risk |
|------------|---------|---------|-------------------|
| `github.com/skip2/go-qrcode` | Latest stable | QR code generation | Low: well-maintained, pure Go, no external deps |
| `github.com/gorilla/mux` | Latest stable | HTTP router | Low: de facto standard, actively maintained |
| `golang.org/x/crypto` | Latest stable | Cryptographic primitives | Low: maintained by Go team |

**Frontend dependencies:**
| Dependency | Version | Purpose | Supply Chain Risk |
|------------|---------|---------|-------------------|
| React | Latest stable | UI framework | Low: de facto standard, actively maintained |
| Vite | Latest stable | Build tool | Low: modern, fast, widely adopted |
| shadcn/ui | Latest stable | Pre-built UI components | Low: built on Radix UI, excellent accessibility |
| Tailwind CSS | Latest stable | Utility-first CSS | Low: industry standard, widely used |

**No crypto libraries**: All encryption uses Web Crypto API (browser-native). No external crypto deps.

---

## ⛔ Security Design Gate

Review of all design decisions in Section 7 against the zero-knowledge constraint:

1. **Does any modified or new module require the server to hold, derive, or access the plaintext payload at any point?**
   → **No.** The plaintext payload is encrypted in the browser before transmission. The server only handles ciphertext.

2. **Does any modified or new module require the server to hold, derive, or access the private key at any point?**
   → **No.** The private key is generated, logged as QR, then immediately shredded via Burn Protocol. The server never persists or uses the private key for decryption.

3. **Does any API contract change expose plaintext or key material over the network?**
   → **No.** `GET /key` returns the public key only. `POST /drop` accepts ciphertext only. The private key is never transmitted.

4. **Does any new dependency introduce an external service that could intercept plaintext or key material?**
   → **No.** All dependencies are pure Go packages with no external service calls. The frontend uses Web Crypto API (browser-native) with no external CDN.

---

✅ **Security Gate Passed.** All design decisions in Section 7 have been reviewed against the zero-knowledge constraint. The server does not hold, derive, or access the plaintext payload or private key at any point in the updated data flow.

---

## 8. Security & Threat Model Impact

**Baseline**: HLD Section 6 threat model.

| Existing Threat | Impact | Mitigation |
|-----------------|--------|------------|
| **DDoS / Hardware Exhaustion** | No impact | Rate limiting (5 req/IP/hour) + 250-char payload cap. |
| **Server Compromise (Root Access)** | No impact | Zero-knowledge architecture: server never has private key. |
| **Memory Scraping (Cold Boot Attacks)** | Strengthened | Burn Protocol with `runtime.KeepAlive()` ensures memory zeroing. |
| **Static Vulnerabilities** | No impact | Static analysis tooling mandated. |

### New Threats Introduced by v1.0

| Threat | Mitigation |
|--------|------------|
| **Key Substitution Attack** (attacker replaces `public_key.pem` with their own) | Key fingerprinting: SHA-256 hash logged on startup; operator can verify. |
| **QR Payload Parsing Error** (malicious input causes buffer overflow in `reader.html`) | Input validation: base64 decoding fails safely; browser sandbox prevents escape. |
| **Printer Device Hijacking** (attacker redirects USB device to malicious endpoint) | Device permissions: `group_add` restricts access; container isolation prevents host access. |
| **Rate Limit Bypass** (attacker distributes requests across IPs) | Rate limit is per-IP; distributed attack requires botnet. Acceptable risk for v1.0. |
| **Web Crypto API Compromise** (browser backdoor in encryption) | No mitigation possible at server level. Operator must use trusted browser. Documented in security guide. |

---

## 9. Testing Strategy

> **Definition**: Testing Strategy defines the *how* — the methods, test types, and environments used to verify that requirements are met. Acceptance Criteria (Section 11) defines the *what*.

### Unit Tests

| Package | Behavior | Expected Outcome |
|---------|----------|------------------|
| `pkg/crypto` | Key generation | Produces valid X25519 key pair; public key is exportable as PEM. |
| `pkg/crypto` | Burn Protocol | Private key byte slice is zeroed; `runtime.KeepAlive()` prevents optimization. |
| `pkg/config` | Env var validation | Missing `PRINTER_DEVICE` returns error; invalid `PRINTER_TYPE` returns error. |
| `pkg/printer` | QR generation | Produces monochrome byte matrix; QR is scannable by standard readers. |
| `pkg/printer` | ESC/POS rasterization | Output bytes are valid ESC/POS commands; `GS v 0` format is correct. |
| `pkg/spooler` | Job queue | Payloads are processed sequentially; queue drains before exit. |
| `pkg/spooler` | Memory zeroing | Payload buffer is zeroed after print job completes. |
| `pkg/api` | `GET /key` | Returns public key as `text/plain` with HTTP 200. |
| `pkg/api` | `POST /drop` valid | Returns HTTP 202; payload is queued to spooler. |
| `pkg/api` | `POST /drop` invalid | Returns HTTP 400 if payload > 250 chars or invalid base64. |
| `pkg/api` | `GET /health` | Returns HTTP 200 if printer available, HTTP 503 if not. |

### Integration Tests

| Interaction | Verification |
|-------------|--------------|
| API → Spooler handoff | `POST /drop` returns 202; spooler receives payload; queue depth increments. |
| Spooler → Printer interface | Spooler calls `Printer.Print()`; QR is generated; ESC/POS bytes are written. |
| Crypto → API payload flow | `POST /drop` accepts payload; payload remains encrypted throughout; no decryption occurs. |
| Graceful shutdown | SIGTERM triggers handler; new requests return 503; spooler drains; process exits. |

### Hardware-in-the-Loop Tests

| Scenario | Mock Printer Behavior | Passing Condition |
|----------|----------------------|-------------------|
| Normal print job | Writes ESC/POS to stdout | Output is valid ESC/POS; QR is parseable. |
| Printer offline | Returns error from `Print()` | Spooler retries 3 times; logs error; job is retained. |
| Queue depth test | Simulates slow printing | Spooler buffers 5 concurrent jobs; processes sequentially. |
| Shutdown mid-job | Blocks until job completes | Graceful shutdown waits for job; exits cleanly. |

**CI vs. Hardware Distinction**:
- **CI-only passing test**: Mock Printer writes valid ESC/POS to stdout.
- **Hardware-in-the-loop passing test**: Physical printer produces scannable QR code on paper; recipient can decrypt using `reader.html`.

**Covered Acceptance Criteria**: AC-001 through AC-015 (see Section 11).

### Offline Reader Tests (`reader.html`)

| Test Vector | Input | Expected Output |
|-------------|-------|-----------------|
| TV-001 | Known plaintext `"Hello, World!"` encrypted with test key | Decrypts to `"Hello, World!"` |
| TV-002 | 185-char plaintext encrypted with test key | Decrypts correctly |
| TV-003 | Ciphertext with `ZD1:` prefix | Strips prefix, decrypts correctly |
| TV-004 | Ciphertext without `ZD1:` prefix | Handles gracefully (error message) |
| TV-005 | Invalid base64 | Shows error, does not crash |

**Minimum Browser Environment**: Chrome 90+, Firefox 88+, Safari 14+. All support Web Crypto API and `navigator.mediaDevices.getUserMedia()`.

### Security Regression Tests

| Property | Test |
|----------|------|
| RAM zeroing | After Burn Protocol, memory dump shows no private key remnants. |
| Burn Protocol idempotency | Calling Burn Protocol multiple times is safe (no panic, no double-free). |
| QR payload boundary | Payload of 246 chars (250 with `ZD1:`) succeeds; 247 chars fails. |
| Private key never persisted | File system scan shows no private key file after startup. |
| Rate limit enforced | 6th request within hour returns HTTP 429 (Traefik). |

---

## 10. Rollback & Compatibility

### Backward Compatibility

| Question | Answer |
|----------|--------|
| Can QR prints generated before this update still be decrypted by `reader.html` post-update? | N/A (initial implementation). Future versions must maintain backward compatibility with `ZD1:` format. |
| Is the `public_key.pem` format stable across this update? | N/A (initial implementation). Format is standard PEM and will not change. |
| Are there breaking changes to Docker Compose configuration requiring manual migration steps? | N/A (initial implementation). |
| Is the `POST /drop` / `GET /key` API surface backward-compatible? | N/A (initial implementation). |

### Forward Compatibility Commitments

| Contract | Stability Guarantee |
|----------|---------------------|
| QR format | `ZD1:` prefix will remain valid for all future v1.X versions. `reader.html` v1.0 will decrypt all v1.X QR codes. |
| `public_key.pem` | Standard PEM format. Will never change in v1.X. |
| `POST /drop` | JSON payload. New optional fields may be added in v1.X; existing fields will not be removed. |
| `GET /key` | Returns PEM. Format will not change in v1.X. |
| `reader.html` | Will decrypt all v1.X QR codes. May need updates for v2.0. |

### Rollback Procedure

**Scenario**: Critical issue found post-deployment; need to revert to previous version.

**Steps**:
1. `docker-compose down` — stops containers gracefully (finishes in-flight jobs).
2. `docker images ls` — identify previous image version.
3. Update `docker-compose.yml` to use previous image tag.
4. `docker-compose up -d` — start previous version.
5. Verify health: `curl http://localhost:8080/health` should return 200.

**Rollback Safety**:
- Is rollback safe to perform while the spooler has jobs in-flight? → **Yes**. Graceful shutdown (step 1) waits for spooler to drain.
- Are there host volume artifacts that must be cleaned up? → **No**. Only `public_key.pem` exists, and it is forward-compatible.
- Maximum acceptable rollback time window? → **5 minutes**. Graceful shutdown waits max 30 seconds; container restart takes ~10 seconds.

---

## 11. Acceptance Criteria

> **Definition**: Acceptance Criteria define the *what* — the observable, binary conditions that must ALL be true before this feature is considered complete. Each AC must be verifiable by a test case in Section 9.

- [ ] **AC-001** `Verifies: FR-001, FR-002` — ZeroDrop starts up, generates ECC key pair, saves `public_key.pem` to volume.
- [ ] **AC-002** `Verifies: FR-003` — Private key is logged as scannable QR code prefixed with `PRIVATE_KEY_QR:`.
- [ ] **AC-003** `Verifies: FR-004` — After private key logging, memory dump shows no private key remnants (Burn Protocol executed).
- [ ] **AC-004** `Verifies: FR-005` — `GET /key` returns HTTP 200 with PEM-encoded public key as `text/plain`.
- [ ] **AC-005** `Verifies: FR-006, FR-009` — `POST /drop` with valid payload returns HTTP 202 immediately (< 100ms).
- [ ] **AC-006** `Verifies: FR-007` — `POST /drop` with payload > 250 chars returns HTTP 400.
- [ ] **AC-007** `Verifies: FR-008` — `POST /drop` with invalid base64 returns HTTP 400.
- [ ] **AC-008** `Verifies: FR-010` — Five concurrent `POST /drop` requests all return HTTP 202; spooler processes sequentially.
- [ ] **AC-009** `Verifies: FR-012` — Mock Printer writes ESC/POS commands to stdout for `POST /drop` request.
- [ ] **AC-010** `Verifies: FR-013, FR-015, FR-016` — USB Printer prints scannable QR code with paper cut.
- [ ] **AC-011** `Verifies: FR-017` — After print job completion, payload buffer in RAM is zeroed.
- [ ] **AC-012** `Verifies: FR-019, FR-020` — Submission portal fetches public key and encrypts payload using Web Crypto API.
- [ ] **AC-013** `Verifies: FR-021, FR-022` — Submission portal prefixes ciphertext with `ZD1:`; validates payload length ≤ 185 chars.
- [ ] **AC-014** `Verifies: FR-023` — Submission portal displays status messages: "Encrypting...", "Transmitting...", "Safely Dropped."
- [ ] **AC-015** `Verifies: FR-024, FR-028` — `reader.html` decrypts QR code offline and displays plaintext.
- [ ] **AC-016** `Verifies: FR-029` — `reader.html` displays SHA-256 fingerprint of imported public key.
- [ ] **AC-017** `Verifies: FR-030` — `GET /health` returns HTTP 200 when printer is available.
- [ ] **AC-018** `Verifies: FR-030` — `GET /health` returns HTTP 503 when printer is offline.
- [ ] **AC-019** `Verifies: FR-031` — Startup logs contain `PUBLIC_KEY_FINGERPRINT: <hex-string>`.
- [ ] **AC-020** `Verifies: FR-032` — SIGTERM triggers graceful shutdown; spooler drains before exit.
- [ ] **AC-021** `Verifies: FR-033, FR-034` — Missing `PRINTER_DEVICE` causes startup to exit with error code 1.
- [ ] **AC-022** `Verifies: FR-036` — Invalid `PRINTER_DEVICE` path causes startup to exit with error code 1.
- [ ] **AC-023** `Verifies: FR-037` — Structured logging (when enabled) contains no private keys, payloads, or submission metadata.
- [ ] **AC-024** `Verifies: FR-038` — Traefik rate limits to 5 req/IP/hour; 6th request returns HTTP 429.
- [ ] **AC-025** `Verifies: FR-039` — Docker Compose successfully maps `/dev/usb/lp0` into container.

---

## 12. Documentation Requirements

| Artifact | Scope of Update | Owner | Milestone |
|----------|-----------------|-------|-----------|
| `README.md` | Complete: setup instructions, env vars, architecture overview, usage guide | TBD | M-05 |
| API Reference | Document `GET /key`, `POST /drop`, `GET /health` with request/response schema | TBD | M-04 |
| `docker-compose.yml` comments | Annotate device mappings, volume mounts, env vars | TBD | M-04 |
| Architecture Diagram | Create module diagram showing all packages and data flow | TBD | M-02 |
| `reader.html` inline docs | Document decryption flow, input format, offline usage | TBD | M-03 |
| Security Guide | Document zero-knowledge model, key handling, threat model | TBD | M-05 |
| Operator Guide | Document startup, shutdown, troubleshooting, log interpretation | TBD | M-05 |

---

## 13. Open Questions

| ID | Question | Impact if Unresolved | Blocking Milestone | Owner | Target Resolution |
|----|----------|---------------------|-------------------|-------|-------------------|
| OQ-001 | Which 58mm thermal printer models will be used for production testing? | Hardware compatibility cannot be verified without target device. | M-04 | TBD | Before M-04 |
| OQ-002 | What is the acceptable spooler drain timeout for graceful shutdown? (Currently proposed: 30 seconds) | Affects operational behavior during rolling updates. | M-04 | TBD | M-02 |
| OQ-003 | Should structured logging be enabled by default or opt-in? (Currently proposed: opt-in) | Affects security posture and operational visibility. | M-04 | TBD | M-02 |

---

## 14. Implementation Milestones

### Milestone Rules

1. Each milestone produces a working, deployable system state.
2. **Mock Printer Continuity Rule**: Mock Printer mode must remain fully functional at the close of every milestone.
3. **ADR Gate**: Not applicable (frontend stack is React + shadcn/ui, decided in this PRD).
4. **OQ Gate**: M-02 and M-04 are blocked by OQ-002 and OQ-003 respectively.

### Milestones

**M-01 — Project Bootstrap & Crypto Foundation**
- **Scope**: Initialize Go module, create directory structure, implement `pkg/crypto` (key generation, Burn Protocol), create `pkg/config` (env var validation).
- **Exit Criteria**:
  - AC-001, AC-003 (key generation and Burn Protocol)
  - AC-021, AC-022 (config validation)
- **Blocked By**: None
- **Mock Printer Status**: N/A (not yet created)
- **Documentation Deliverables**: None

**M-02 — API & Spooler Core**
- **Scope**: Implement `pkg/api` (`GET /key`, `POST /drop`), `pkg/spooler` (worker pool, memory zeroing), `pkg/observability` (structured logging). Resolve OQ-002 and OQ-003.
- **Exit Criteria**:
  - AC-004, AC-005, AC-006, AC-007, AC-008, AC-011 (API and spooler)
  - AC-023 (structured logging)
- **Blocked By**: M-01, OQ-002, OQ-003
- **Mock Printer Status**: Not yet created (comes in M-03)
- **Documentation Deliverables**: None

**M-03 — Printer Interface & Reader**
- **Scope**: Implement `pkg/printer` (Printer interface, Mock Printer, QR generation, ESC/POS rasterization), create `reader.html`.
- **Exit Criteria**:
  - AC-009 (Mock Printer)
  - AC-015, AC-016 (`reader.html`)
- **Blocked By**: M-02
- **Mock Printer Status**: Created and functional (writes to stdout)
- **Documentation Deliverables**: `reader.html` inline docs

**M-04 — USB Printer & Health Check**
- **Scope**: Implement USB Printer, `GET /health` endpoint, graceful shutdown handler. Resolve OQ-001.
- **Exit Criteria**:
  - AC-010, AC-017, AC-018 (USB printer, health check)
  - AC-020 (graceful shutdown)
  - AC-025 (Docker device mapping)
- **Blocked By**: M-03, OQ-001
- **Mock Printer Status**: Updated to work alongside USB Printer (both implement same interface)
- **Documentation Deliverables**: API Reference, `docker-compose.yml` comments

**M-05 — Frontend & Production Readiness**
- **Scope**: Implement submission portal (React + Vite + shadcn/ui), Traefik integration, end-to-end testing, all documentation.
- **Exit Criteria**:
  - AC-002, AC-012, AC-013, AC-014, AC-019, AC-024 (frontend and key provisioning)
  - AC-026 (Traefik rate limiting)
- **Blocked By**: M-04
- **Mock Printer Status**: Remains functional
- **Documentation Deliverables**: `README.md`, Architecture Diagram, Security Guide, Operator Guide

---

## Traceability Audit

**Chain Verification**:

| US | FR Count | AC Count | TC Coverage |
|----|----------|----------|-------------|
| US-001 | 5 (FR-001, FR-002, FR-005, FR-033, FR-034) | 3 (AC-001, AC-021, AC-022) | TC-001, TC-007, TC-008 |
| US-002 | 1 (FR-003) | 1 (AC-002) | TC-002 |
| US-003 | 2 (FR-003, FR-004) | 1 (AC-003) | TC-003 |
| US-004 | 5 (FR-006, FR-008, FR-019, FR-020) | 1 (AC-012) | TC-004 |
| US-005 | 4 (FR-006, FR-009, FR-010, FR-023) | 3 (AC-005, AC-008, AC-014) | TC-005, TC-006 |
| US-006 | 1 (FR-007) | 1 (AC-006) | TC-009 |
| US-007 | 4 (FR-024, FR-025, FR-026, FR-028) | 1 (AC-015) | TC-010 |
| US-008 | 2 (FR-027, FR-029, FR-031) | 2 (AC-016, AC-019) | TC-011 |
| US-009 | 1 (FR-030) | 2 (AC-017, AC-018) | TC-012 |
| US-010 | 1 (FR-032) | 1 (AC-020) | TC-013 |

**Verification**: All US have FR children. All FR have US parents. All AC have FR parents. TC coverage is complete for all AC.

---

**End of PRD**
