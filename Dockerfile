# Build stage
ARG GO_VERSION=1.24
FROM golang:${GO_VERSION}-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    make \
    upx

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies with retry logic
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && \
    go mod verify

# Copy source code
COPY . .

# Build the application with optimizations
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev} -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -trimpath \
    -o bot \
    cmd/bot/main.go

# Compress binary with UPX
RUN upx --best --lzma bot

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    dumb-init && \
    update-ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bot .

# Copy configs directory (Stage 4)
COPY --from=builder /app/configs ./configs

# Create non-root user with specific UID/GID
RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Health check (optional - adjust endpoint if needed)
# HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
#   CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Use dumb-init to properly handle signals
ENTRYPOINT ["/usr/bin/dumb-init", "--"]

# Run the application
CMD ["./bot"]

# Labels for metadata
LABEL maintainer="your-email@example.com" \
      description="Crypto Trading Bot with DCA and Auto-Sell strategies" \
      version="${VERSION:-dev}"
