# syntax=docker/dockerfile:1

ARG GO_VERSION=1.27-alpine

# ==========================================
# 1. Build Stage
# ==========================================
FROM golang:${GO_VERSION} AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/bin/bookstore-api \
    ./cmd/main.go

# ==========================================
# 2. Production Runtime Stage
# ==========================================
FROM alpine:3.21 AS runner

# Install runtime dependencies:
# - ca-certificates: TLS/HTTPS support for external APIs (Google OAuth, Biteship, Midtrans)
# - tzdata: Required by cmd/main.go for time.LoadLocation("Asia/Jakarta")
# - curl: For container health check
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user and group
RUN addgroup -S appgroup -g 1001 && \
    adduser -S appuser -u 1001 -G appgroup

WORKDIR /app

# Create necessary directories and set permissions
RUN mkdir -p /app/public /app/logs && \
    chown -R appuser:appgroup /app

# Copy binary from builder stage
COPY --from=builder --chown=appuser:appgroup /app/bin/bookstore-api /app/bookstore-api

# Copy static assets and migrations
COPY --chown=appuser:appgroup public /app/public
COPY --chown=appuser:appgroup migrations /app/migrations

# Switch to non-root user for security
USER appuser

# Expose default HTTP port
EXPOSE 8000

# Container healthcheck
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8000/health || exit 1

# Run the binary
ENTRYPOINT ["/app/bookstore-api"]
