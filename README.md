# ZeroDrop Terminal v1.0

Air-gapped, zero-knowledge secure credential delivery terminal. Users encrypt sensitive data in their browser using Web Crypto API, transmit it to a server that cannot decrypt it, and receive the ciphertext as a physical QR code printout via 58mm thermal printer. Recipients decrypt the printout offline using a standalone HTML file.

## Architecture

- **Zero-knowledge guarantee**: Server never possesses plaintext payload or private key
- **Ephemeral processing**: No database. Data exists in RAM only during print job, then is zeroed
- **Hardware abstraction**: Supports Mock Printer (stdout) and USB Printer (auto-detection)
- **Asynchronous spooler**: Buffered Go channel worker pool for sequential print processing
- **Client-side encryption**: All crypto happens in browser using Web Crypto API
- **Offline decryption**: `static/reader.html` works completely offline with no external dependencies

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.26+ |
| Crypto | Curve25519 (X25519) via `crypto/ecdh` |
| Frontend | React + Vite + shadcn/ui + Tailwind CSS |
| Infrastructure | Docker Compose + Traefik |
| Hardware | 58mm thermal printer (ESC/POS), USB |
| Testing | Go testing framework + Mock Printer |

## Quick Start

### Prerequisites

- Go 1.26+
- Node.js 20+ (for frontend development)
- Docker & Docker Compose (for production deployment)

### Development (Backend)

```bash
# Run tests
go test ./... -v -race -cover

# Build binary
go build -o bin/zerodrop ./cmd/zerodrop

# Run with Mock Printer
PRINTER_TYPE=mock ./bin/zerodrop

# Run with USB Printer (auto-detect)
PRINTER_TYPE=usb PRINTER_DEVICE="" ./bin/zerodrop

# Run with USB Printer (specific device)
PRINTER_TYPE=usb PRINTER_DEVICE=/dev/usb/lp0 ./bin/zerodrop
```

### Development (Frontend)

```bash
cd frontend

# Install dependencies
npm install

# Run dev server (proxies to backend on :8080)
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

### Production (Docker)

```bash
# Build and start all services
docker-compose -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.traefik.yml up -d

# View logs
docker-compose logs -f zerodrop

# Stop
docker-compose down
```

## API Endpoints

- **GET /key** — Retrieve server's public key (PEM format)
- **POST /drop** — Submit encrypted payload for printing
  ```json
  {
    "payload": "ZD1:<base64_encoded_ciphertext>"
  }
  ```
- **GET /health** — Health check (includes printer status)

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PRINTER_TYPE` | Yes | mock | Printer type: `mock` or `usb` |
| `PRINTER_DEVICE` | No* | /dev/usb/lp0 | USB device path (empty = auto-detect) |
| `PUBLIC_KEY_PATH` | No | ./data/public_key.pem | Path to public key PEM |
| `RATE_LIMIT_REQUESTS_PER_HOUR` | No | 5 | Rate limit per IP per hour |
| `LOG_ENABLED` | No | false | Enable structured JSON logging |

*Required for USB printer if auto-detection fails

## Security

- **Zero-knowledge**: Server cannot decrypt messages
- **Burn Protocol**: Private key destroyed from memory after startup
- **Memory hygiene**: All payload buffers zeroed after print job
- **Rate limiting**: 5 requests per IP per hour (Traefik)
- **No database**: Ephemeral RAM-only processing
- **Web Crypto API**: Browser-native encryption, no external libraries

## USB Printer Support

Auto-detects 10+ common 58mm thermal printers:
- POS-5890 / Generic ESC/POS (1504:0006)
- Epson TM-T88 series (04b8:0202, 04b8:0203)
- Rongta RP58 (0416:5011)
- XPrinter XP-58III (0456:0808)
- And 6 more models

If no USB printer is found, gracefully falls back to Mock Printer.

## Testing

```bash
# Run all tests with race detection
go test ./... -v -race -cover

# Run specific package tests
go test ./pkg/crypto -v

# Run with coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Documentation

- **PRD**: `docs/prd/PRD-001-zerodrop-terminal-v1.0.md` — Complete requirements
- **Overview**: `docs/OVERVIEW.md` — Stakeholder-friendly explanation
- **Reader**: `static/reader.html` — Offline decryption utility

## License

MIT License — See LICENSE file for details

## Contributing

This is a security-focused application. All contributions must:
1. Pass all tests with race detection enabled
2. Pass `govulncheck` with no vulnerabilities
3. Maintain zero-knowledge guarantee
4. Follow Go coding standards in `.claude/CODING_STANDARDS.md`
5. Follow security standards in `.claude/SECURITY_STANDARDS.md`
