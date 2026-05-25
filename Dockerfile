# ZeroDrop Terminal - Production Dockerfile
# Multi-stage build for minimal image size

# Build stage
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o zerodrop ./cmd/zerodrop

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS (if needed for future features)
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 zerodrop && \
    adduser -D -u 1000 -G zerodrop zerodrop

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/zerodrop .

# Copy static files (reader.html)
COPY --chown=zerodrop:zerodrop static/ static/

# Create directory for public key (if not using volume)
RUN mkdir -p /app/data && chown -R zerodrop:zerodrop /app/data

# Switch to non-root user
USER zerodrop

# Environment variables
ENV PRINTER_TYPE=mock
ENV PUBLIC_KEY_PATH=/app/data/public_key.pem
ENV LOG_ENABLED=false

# Expose HTTP port
EXPOSE 8080

# Health check (wget -q exits 0 for any HTTP response, non-zero on connection failure)
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:8080/health || exit 1

# Run the binary
ENTRYPOINT ["./zerodrop"]
