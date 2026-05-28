# ZeroDrop Terminal - Production Dockerfile
# Multi-stage build for minimal image size

# =============================================================================
# Stage 1: Build Go backend binary
# =============================================================================
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o zerodrop ./cmd/zerodrop

# =============================================================================
# Stage 2: Build frontend SPA
# =============================================================================
FROM node:20-alpine AS frontend-builder

WORKDIR /build

# Cache dependencies separately for faster rebuilds
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci

# Build frontend
COPY frontend/ .
RUN npm run build

# =============================================================================
# Stage 3: Runtime image
# =============================================================================
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 1000 zerodrop && \
    adduser -D -u 1000 -G zerodrop zerodrop

WORKDIR /app

# Copy backend binary
COPY --from=builder /build/zerodrop .

# Copy frontend dist (built by frontend-builder)
COPY --from=frontend-builder /build/dist/ frontend/dist/

# Copy static files (reader.html)
COPY --chown=zerodrop:zerodrop static/ static/

RUN mkdir -p /app/data && chown -R zerodrop:zerodrop /app/data

USER zerodrop

ENV PRINTER_TYPE=mock
ENV PUBLIC_KEY_PATH=/app/data/public_key.pem
ENV LOG_ENABLED=false

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q --no-check-certificate -O /dev/null https://localhost:8080/health 2>/dev/null || wget -q -O /dev/null http://localhost:8080/health 2>/dev/null || exit 1

ENTRYPOINT ["./zerodrop"]
