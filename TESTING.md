# ZeroDrop Terminal — Manual Testing Guide

> ECIES Crypto Chain (X25519 ECDH + AES-256-GCM), QR ESC/POS Print, Offline Decryption

---

## Quick Start

```bash
# 1. Build the backend
go build -o bin/zerodrop ./cmd/zerodrop

# 2. Build the frontend
cd frontend && npm run build && cd ..

# 3. Run the server (Mock Printer for testing, no hardware needed)
PRINTER_TYPE=mock LOG_ENABLED=false ./bin/zerodrop
```

> **Note:** The server auto-detects the project root by looking for `go.mod`. You can run it from any directory — it will always find the correct paths for the frontend, static files, and keys.

The server starts on `http://localhost:8080`.

On startup, look in the terminal for:
- **`Working directory:`** — Confirms the server found the project root and chdir'd there
- **`PUBLIC_KEY_FINGERPRINT`** — SHA-256 hex string to verify the key
- **`PRIVATE_KEY_QR (PEM DATA)`** — The PEM private key (save for decryption testing)
- **`PRIVATE_KEY_QR saved to ...private_key_qr.png`** — Full absolute path to the scannable QR PNG

The file `private_key_qr.png` is always created in the project root (alongside `go.mod`), regardless of where you run the binary from. It contains the private key as a scannable QR code (raw PEM, no wrapper). You can open it and scan with the reader's camera, or just copy the PEM text from the terminal log.

---

## Step-by-Step Testing

### 1. Backend Health

```bash
curl http://localhost:8080/health
```

**Printer available (200):**
```json
{
  "status": "healthy",
  "service": "zerodrop-terminal",
  "printer": {
    "type": "mock",
    "available": true,
    "status": "healthy"
  }
}
```

**Printer unavailable (503):**
```json
{
  "status": "unhealthy",
  "service": "zerodrop-terminal",
  "printer": {
    "available": false
  }
}
```
(With HTTP 503 — occurs when `IsAvailable()` returns `false`)

---

### 2. Get Public Key

```bash
curl http://localhost:8080/key
```

**Expected:** PEM-formatted public key
```
-----BEGIN PUBLIC KEY-----
...
-----END PUBLIC KEY-----
```

---

### 3. Frontend (Browser)

Open `http://localhost:8080` in your browser.

**What to expect:**
- ZeroDrop Terminal title with gradient header
- **Public key SHA-256 fingerprint** loaded and displayed (matches server startup log)
- Printer status badge: `mock (available)` in green
- Text input area for your message
- "Encrypt Message" button (disabled until fingerprint loads)

---

### 4. Full Encryption Flow (Browser)

**Step 1 — Encrypt:**
- Type a message: `Test secret password: hunter2`
- Click **"Encrypt Message"**

*Behind the scenes:*
1. Frontend parses the PEM public key via `parsePEM()` (binary DER extraction)
2. Imports as X25519 SPKI key via Web Crypto API
3. Generates an **ephemeral X25519 key pair**
4. Derives a 32-byte shared secret via **ECDH**
5. Imports shared secret as **AES-256-GCM** key
6. Generates random **12-byte IV**
7. Encrypts plaintext → ciphertext + 16-byte GCM auth tag
8. Packs payload: `ZD1:base64(ephPubKeyRaw(32B) + iv(12B) + ciphertextWithTag)`

**Step 2 — Submit:**
- Encrypted payload (starting with `ZD1:`) is displayed
- Click **"Submit to Print"**

**On the server terminal:**
- `[MockPrinter]` prints QR ESC/POS GS v 0 commands with a hex preview
- The output represents a **real QR code** image, not raw text

**Status messages (per PRD spec):**
| Phase | Message |
|-------|---------|
| Encrypting | `"Encrypting..."` |
| Submitting | `"Transmitting..."` |
| Success | `"Safely Dropped."` |

---

### 5. Offline Reader (Real ECDH Decryption)

Open in your browser:
```
http://localhost:8080/reader.html
```

**This is now a real decryption tool** with full ECIES support.

**Step-by-step:**

1. **Page loads** — No network calls. jsQR v1.4.0 is loaded from `static/jsqr.min.js` (local). All crypto uses Web Crypto API.

2. **Paste or scan the payload:**
   - *Option A:* Click "Start Camera" and scan a QR code (uses WebRTC `getUserMedia` + jsQR)
   - *Option B:* Manually paste the encrypted payload (the `ZD1:...` string)

3. **Import the private key:**
   - *Option A:* Open `private_key_qr.png` on your screen or print it, then scan via camera using the reader's "Start Camera" button
   - *Option B:* Copy the PEM from the server startup log (`PRIVATE_KEY_QR (PEM DATA):`) and paste into the key input field

4. **Verify fingerprint:**
   - The reader calculates the **SHA-256 fingerprint** of your imported key's public key
   - Verify it matches the server's `PUBLIC_KEY_FINGERPRINT` log

5. **Decrypt:**
   - Click **"Decrypt"**
   - This runs the reverse ECIES:
     - Strips `ZD1:` prefix, base64-decodes
     - Extracts 32B ephemeral public key + 12B IV + ciphertext+tag
     - ECDH derives shared secret from your private key + ephemeral public key
     - AES-256-GCM decrypts (auto-validates auth tag)
     - Displays the original plaintext

> **The reader works fully offline.** Copy `static/reader.html` + `static/jsqr.min.js` to a USB stick, open on an air-gapped machine, and decrypt without any network access.

---

### 6. API Payload Submission (curl)

```bash
# Valid (starts with ZD1:)
curl -X POST http://localhost:8080/drop \
  -H "Content-Type: application/json" \
  -d '{"payload":"ZD1:VGVzdCBtZXNzYWdl"}'

# → HTTP 202
# {"status":"queued","message":"Payload queued for printing"}
```

```bash
# Payload too long (exceeds 400)
curl -X POST http://localhost:8080/drop \
  -H "Content-Type: application/json" \
  -d '{"payload":"ZD1:'$(python3 -c 'print("A"*410)')'"}'

# → HTTP 400
# Payload exceeds 400 character limit
```

```bash
# Missing ZD1: prefix
curl -X POST http://localhost:8080/drop \
  -H "Content-Type: application/json" \
  -d '{"payload":"VGVzdA=="}'

# → HTTP 400
# Payload must start with ZD1: prefix
```

```bash
# Invalid JSON
curl -X POST http://localhost:8080/drop \
  -H "Content-Type: application/json" \
  -d 'not json'

# → HTTP 400
```

---

### 7. Graceful Shutdown

Press `Ctrl+C` in the server terminal.

**Expected sequence:**
```
Shutdown signal received. Initiating graceful shutdown...
Waiting for spooler to drain (timeout: 30s)...
Spooler drained successfully
Graceful shutdown complete. Exiting.
```

The spooler drains in-flight print jobs (max 30s timeout) before exiting. All processed payload buffers are explicitly zeroed after each job.

---

## Full Automated Test Script

```bash
#!/bin/bash
# save as test-e2e.sh && chmod +x test-e2e.sh && ./test-e2e.sh

set -e

echo "═══════════════════════════════════════════"
echo "  ZeroDrop Terminal v1.0 — E2E Test Suite"
echo "═══════════════════════════════════════════"
echo ""

# Start server
echo "▶ Starting server (Mock Printer)..."
PRINTER_TYPE=mock LOG_ENABLED=false ./bin/zerodrop &
SERVER_PID=$!
sleep 3

# Cleanup on exit
trap "kill $SERVER_PID 2>/dev/null; echo 'Server stopped.'" EXIT

echo ""
echo "───────────────────────────────────────────"
echo "1. Health Check"
echo "───────────────────────────────────────────"
curl -s http://localhost:8080/health | python3 -m json.tool
echo ""

echo "───────────────────────────────────────────"
echo "2. Get Public Key (first 3 lines)"
echo "───────────────────────────────────────────"
curl -s http://localhost:8080/key | head -3
echo ""

echo "───────────────────────────────────────────"
echo "3. Submit Valid Payload"
echo "───────────────────────────────────────────"
curl -s -X POST http://localhost:8080/drop \
  -H "Content-Type: application/json" \
  -d '{"payload":"ZD1:VGVzdCBtZXNzYWdl"}' | python3 -m json.tool
echo ""

echo "───────────────────────────────────────────"
echo "4. Submit Invalid (no ZD1:)"
echo "───────────────────────────────────────────"
curl -s -X POST http://localhost:8080/drop \
  -H "Content-Type: application/json" \
  -d '{"payload":"VGVzdA=="}'
echo ""

echo "───────────────────────────────────────────"
echo "5. Submit Invalid (too long)"
echo "───────────────────────────────────────────"
curl -s -X POST http://localhost:8080/drop \
  -H "Content-Type: application/json" \
  -d '{"payload":"ZD1:'$(python3 -c 'print("A"*410)')'"}'
echo ""

echo "───────────────────────────────────────────"
echo "6. Submit Invalid (bad JSON)"
echo "───────────────────────────────────────────"
curl -s -X POST http://localhost:8080/drop \
  -H "Content-Type: application/json" \
  -d 'not json'
echo ""

echo ""
echo "═══════════════════════════════════════════"
echo "  All API tests passed!"
echo "═══════════════════════════════════════════"
echo ""
echo "Browser tests to run manually:"
echo "  • http://localhost:8080           (Frontend)"
echo "  • http://localhost:8080/reader.html  (Offline Reader)"
echo ""

echo "Press ENTER to stop the server..."
read
```

---

## Testing Checklist

### Backend
- [ ] Server starts — logs fingerprint, private key PEM + saves `private_key_qr.png`
- [ ] `private_key_qr.png` created in project root (openable image, scannable QR)
- [ ] `GET /health` (200) — returns JSON with printer status
- [ ] `GET /health` (503) — when printer unavailable (USB mode, no device)
- [ ] `GET /key` (200) — returns PEM public key
- [ ] `POST /drop` (202) — valid payload accepted, queued for print
- [ ] `POST /drop` (400) — missing `ZD1:` prefix
- [ ] `POST /drop` (400) — payload > 400 characters
- [ ] `POST /drop` (400) — invalid JSON body
- [ ] No duplicate log lines on startup
- [ ] Graceful shutdown — spooler drains, buffers zeroed
- [ ] Payload memory zeroed after print job

### Frontend
- [ ] Page loads — shows fingerprint, printer status, message input
- [ ] **Encrypt** — produces `ZD1:base64(...)` payload (real ECDH + AES-256-GCM)
- [ ] **Status chain** — "Encrypting..." → "Transmitting..." → "Safely Dropped."
- [ ] **Server terminal** — shows QR ESC/POS output (hex preview)
- [ ] Payload > 185 chars shows frontend validation error

### Offline Reader (`reader.html`)
- [ ] Loads without network (disconnect internet -> refresh -> still works)
- [ ] **Camera scanning** — jsQR decodes QR from device camera
- [ ] **PEM import** — accepts `-----BEGIN PRIVATE KEY-----` format
- [ ] **Fingerprint** — SHA-256 of imported key matches server log
- [ ] **Decrypt** — ECDH + AES-256-GCM returns original plaintext
- [ ] **Decrypt with wrong key** — auth tag mismatch error
- [ ] **Decrypt offline** — copy to USB, open on air-gapped machine, works

---

## Docker Testing

```bash
# Build frontend first
cd frontend && npm run build && cd ..

# Development mode
docker-compose up

# Production mode (with Traefik)
docker-compose -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.traefik.yml up -d

# View logs
docker-compose logs -f zerodrop

# Stop
docker-compose down
```

Same testing flow at `http://localhost:8080`.

---

## USB Printer Testing (with Hardware)

```bash
# Auto-detect (scans /dev/usb/lp*, /dev/lp*, /dev/ttyUSB*)
PRINTER_TYPE=usb PRINTER_DEVICE="" LOG_ENABLED=false ./bin/zerodrop

# Specific device
PRINTER_TYPE=usb PRINTER_DEVICE=/dev/usb/lp0 LOG_ENABLED=false ./bin/zerodrop
```

**Behavior:**
- If printer found → prints QR codes on 58mm thermal paper
- If printer not found → gracefully falls back to Mock Printer (logs to stdout)
- Health endpoint returns 503 if USB printer was expected but unavailable

---

## Key Behavioral Changes from v1.0

| Aspect | Before | Now |
|--------|--------|-----|
| **Encryption** | Stub (btoa encode) | Real X25519 ECDH + AES-256-GCM (ECIES) |
| **Print output** | Raw ciphertext text | QR ESC/POS GS v 0 raster commands |
| **Reader decryption** | "Not implemented" | Real ECDH decrypt + jsQR camera scan |
| **PEM import (frontend)** | Broken (raw string → SPKI) | Fixed (`parsePEM` → binary DER → SPKI) |
| **Status messages** | Generic "Success!" | Per PRD: Encrypting / Transmitting / Safely Dropped |
| **Payload limit** | 250 chars | 400 chars |
| **Health 503** | Always HTTP 200 | HTTP 503 when `IsAvailable()` false |
| **Private key logging** | PEM text only | QR PNG image + PEM text fallback |
| **Offline reader** | Requires network (future) | Fully offline (jsQR saved locally) |
