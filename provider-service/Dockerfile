# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files first for better caching
COPY provider-service/go.mod provider-service/go.sum ./provider-service/
COPY lurus-common/go.mod lurus-common/go.sum ./lurus-common/

# Download dependencies
WORKDIR /app/provider-service
RUN go mod download

# Copy source code
WORKDIR /app
COPY lurus-common/ ./lurus-common/
COPY provider-service/ ./provider-service/

# Build
WORKDIR /app/provider-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /provider-service ./cmd/provider

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Copy binary and config
COPY --from=builder /provider-service .
COPY --from=builder /app/provider-service/configs/config.yaml ./configs/

# Environment
ENV TZ=Asia/Shanghai

# Expose ports (HTTP and gRPC)
EXPOSE 18101 19101

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:18101/health || exit 1

# Run
ENTRYPOINT ["/app/provider-service"]
CMD ["-conf", "/app/configs/config.yaml"]
