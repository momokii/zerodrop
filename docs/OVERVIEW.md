# ZeroDrop Terminal — Simple Overview

**Version:** 1.0  
**Last Updated:** 2026-05-11

---

## What Is ZeroDrop Terminal?

ZeroDrop is a **secure credential delivery terminal** that lets you send sensitive information (passwords, API keys, security reports) to someone else without:

- Storing the data on a server
- The server being able to read your data
- The data being intercepted during transmission

It works by:
1. **Encrypting** your message in your browser before sending
2. **Printing** the encrypted message as a QR code on a thermal printer
3. **Burning** (destroying) the encryption key immediately after
4. **Recipient** scans the QR code with their phone/computer and decrypts it offline

**The server never sees your plaintext message or the decryption key.** Even if the server is hacked, the data is useless.

---

## Main Features

### 1. Zero-Knowledge Encryption
- Your message is encrypted in your browser using military-grade cryptography (Curve25519)
- The server only receives encrypted gibberish — it cannot decrypt your message
- The decryption key is printed as a QR code, then destroyed from the server

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

### 6. Production Ready
- Docker Compose setup — one command to start
- Traefik reverse proxy with rate limiting (prevents abuse)
- Structured logging (optional, privacy-conscious)
- Health check endpoint for monitoring

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
│ 6. Shred key    │
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
- **Reverse Proxy:** Traefik (handles HTTPS, rate limiting)
- **Hardware:** USB thermal printer, Linux server
- **Isolation:** Container runs without privileged access

---

## Security Guarantees

1. **Zero-Knowledge:** Server never possesses plaintext or private key
2. **Memory Hygiene:** All sensitive data is zeroed from RAM after use
3. **No Database:** Data is never stored — exists only during printing
4. **Rate Limited:** 5 requests per IP per hour (prevents abuse)
5. **Forward Compatible:** QR format versioned (`ZD1:`) for future upgrades

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
2. Print private key QR code to logs
3. Display public key fingerprint
4. Shred private key from server

---

## Summary

**ZeroDrop Terminal** is a secure, physical delivery system for sensitive information. It encrypts data in your browser, prints it as a QR code, and lets recipients decrypt it offline — all without the server ever being able to read your message.

**Perfect for:** Passwords, API keys, security credentials, emergency access codes.

**Built for:** Organizations that need secure credential delivery without database risks.

**Technology:** Go backend, React frontend, Docker deployment, thermal printer output.
