# ZeroDrop Terminal — Simple Overview

**Version:** 1.1  
**Last Updated:** 2026-06-08

---

## What Is ZeroDrop Terminal?

ZeroDrop is a **secure credential delivery terminal** that lets you send sensitive information (passwords, API keys, security reports) to someone else without:

- Storing the data on a server
- The server being able to read your data
- The data being intercepted during transmission

It works by:
1. **Encrypting** your message in your browser before sending
2. **Printing** the encrypted message as a QR code on a thermal printer
3. **Burning** (destroying) the encryption key from server memory after the initial QR display — the key file stays on disk (0600 permissions) so it survives restarts without being loaded into RAM
4. **Recipient** scans the QR code with their phone/computer and decrypts it offline

**The server never sees your plaintext message or the decryption key.** Even if the server is hacked, the data is useless.

---

## Main Features

### 1. Zero-Knowledge Encryption
- Your message is encrypted in your browser using military-grade cryptography (Curve25519)
- The server only receives encrypted gibberish — it cannot decrypt your message
- The decryption key is printed as a QR code once (on first boot), then destroyed from server memory. The key file stays on disk so previously encrypted messages remain decryptable across restarts — but the server never loads it into RAM again

### 2. Physical QR Code Printout
- Encrypted message is printed as a scannable QR code on 58mm thermal paper
- Each print is separated with automatic paper cutting
- QR codes are designed to work even with low-quality scanners

### 3. Offline Decryption
- Recipients use a simple HTML file (`reader.html`) to decrypt QR codes
- Works completely offline — no internet connection needed
- Uses device camera to scan QR codes
- Paste your private key to decrypt

### 4. Simple Submission Portal
- Clean, modern web interface built with React
- Type your message → Encrypt → Submit
- Real-time validation (character limits, formatting)
- Status feedback: "Encrypting...", "Transmitting...", "Safely Dropped"

### 5. Hardware Integration
- Works with standard 58mm thermal receipt printers
- USB connectivity (plug and play)
- Automatic print queue — handles multiple submissions
- Graceful shutdown — finishes printing before stopping

### 6. Admin Dashboard (v1.1)
- Web UI at `/admin` for monitoring and management
- Token-based authentication with session cookies
- Live spooler metrics with auto-refresh
- Printer management — detect and switch printers at runtime
- Key management — view fingerprint, download, rotate

### 7. Persistent Key Pair (v1.1)
- Private key saved to disk on first run (0600 permissions)
- Survives restarts — old encrypted QR codes remain decryptable
- Key rotation via `KEY_ROTATE=true` or admin dashboard
- Burn Protocol still runs — key zeroed from RAM after initial QR display

### 8. Production Ready
- Docker Compose setup — one command to start
- Structured logging (optional, privacy-conscious)
- Health check endpoint with printer status

---

## How It Works (Simple Flow)

```
┌─────────────────┐
│   SUBMITTER     │
│                 │
│ 1. Type message │
│ 2. Encrypt in   │
│    browser      │
│ 3. Submit       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   SERVER        │
│                 │
│ 4. Receive      │
│    encrypted    │
│    data         │
│ 5. Print QR     │
│    code         │
│ 6. Burn key     │
│    from RAM     │
│    (disk stays) │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   PRINTER       │
│                 │
│ 7. Print QR     │
│    code on      │
│    thermal paper│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   RECIPIENT     │
│                 │
│ 8. Scan QR      │
│    code         │
│ 9. Paste key    │
│ 10. Decrypt     │
│     offline     │
└─────────────────┘
```

**Key Point:** The server never knows your message or has the key to decrypt it.

---

## Use Cases

| Use Case | How ZeroDrop Helps |
|----------|-------------------|
| **Password handoff** | Send a temporary password securely without email/chat |
| **API key delivery** | Deliver API credentials to developers without database storage |
| **Security reports** | Share sensitive findings with offline teams |
| **Emergency access** | Provide one-time credentials to air-gapped systems |
| **Audit trails** | Physical paper trail of what was delivered |

---

## What Makes ZeroDrop Different?

| Traditional Methods | ZeroDrop |
|---------------------|----------|
| Email/database stores data | **No storage** — data exists only during printing |
| Server can read your data | **Zero-knowledge** — server cannot decrypt |
| Network interception risk | **Encrypted in browser** before transmission |
| Requires internet to decrypt | **Offline decryption** — works without network |
| Complex mobile apps | **Simple HTML file** for decryption |
| Persistent attack surface | **Ephemeral** — data shredded after printing |

---

## Technical Overview (Simplified)

### Frontend (What Users See)
- **Technology:** React + Vite + shadcn/ui
- **Encryption:** Web Crypto API (browser-native, no plugins)
- **Submission Portal:** Clean web form for typing messages
- **Reader Tool:** Standalone HTML file for decrypting QR codes

### Backend (What Runs on Server)
- **Language:** Go (fast, secure, reliable)
- **Cryptography:** Curve25519 (military-grade, compact output)
- **Printer Support:** ESC/POS thermal printers (58mm receipt printers)
- **Job Queue:** Handles multiple submissions automatically

### Infrastructure (How It's Deployed)
- **Containers:** Docker Compose (one command to start everything)

- **Hardware:** USB thermal printer, Linux server
- **Isolation:** Container runs without privileged access

---

## Security Guarantees

1. **Zero-Knowledge:** Server never possesses plaintext or private key during operation
2. **Memory Hygiene:** All sensitive data is zeroed from RAM after use — private key is burned from memory after initial QR display
3. **Persistent Key:** The key pair is saved to disk (0600 permissions) so previously encrypted data survives restarts, but the private key is never loaded into server RAM after the first boot
4. **Admin Authentication:** Token-based auth with session cookies (HttpOnly, SameSite), constant-time comparison, login rate-limited (10 attempts/15 min/IP)
5. **No Database:** Data is never stored — exists only during printing
6. **Rate Limited:** Built-in per-IP rate limiting (5 req/hr default) + separate admin login rate limiter. Deploy behind a reverse proxy for production-grade protection.
7. **Forward Compatible:** QR format versioned (`ZD1:`) for future upgrades

---

## What You Need to Run ZeroDrop

### Hardware
- Linux server or VM (Docker capable)
- 58mm thermal receipt printer (USB)
- Computer/device with browser (for submitting and receiving)

### Software
- Docker 20.10+
- Docker Compose 2.0+

### Optional
- Webcam or phone camera (for scanning QR codes)
- Air-gapped computer (for offline decryption)

---

## Quick Start (When Ready)

```bash
# Clone repository
git clone <repo-url>
cd zerodrop

# Start everything
docker-compose up -d

# Access submission portal
open http://localhost:8080

# View logs
docker-compose logs -f
```

First boot will:
1. Generate encryption key pair
2. Save private key to disk (0600 permissions) for persistence across restarts
3. Print private key QR code to logs
4. Display public key fingerprint
5. Shred private key from server memory (file stays on disk)

**On subsequent restarts**, the existing key pair is reused. The private key never re-enters server memory — only the public key is loaded. This means old encrypted QR codes remain decryptable after a restart.

---

## Summary

**ZeroDrop Terminal** is a secure, physical delivery system for sensitive information. It encrypts data in your browser, prints it as a QR code, and lets recipients decrypt it offline — all without the server ever being able to read your message.

**Perfect for:** Passwords, API keys, security credentials, emergency access codes.

**Built for:** Organizations that need secure credential delivery without database risks.

**Technology:** Go backend, React frontend, Docker deployment, thermal printer output.
