# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files first for better caching
COPY billing-service/go.mod billing-service/go.sum ./billing-service/
COPY lurus-common/go.mod lurus-common/go.sum ./lurus-common/

# Download dependencies
WORKDIR /app/billing-service
RUN go mod download

# Copy source code
WORKDIR /app
COPY lurus-common/ ./lurus-common/
COPY billing-service/ ./billing-service/

# Build
WORKDIR /app/billing-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /billing-service ./cmd/billing

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Copy binary and config
COPY --from=builder /billing-service .
COPY --from=builder /app/billing-service/configs/config.yaml ./configs/

# Environment
ENV TZ=Asia/Shanghai

# Expose port
EXPOSE 18103

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:18103/health || exit 1

# Run
ENTRYPOINT ["/app/billing-service"]
CMD ["-conf", "/app/configs/config.yaml"]
