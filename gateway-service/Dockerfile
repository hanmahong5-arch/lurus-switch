# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files first for better caching
COPY gateway-service/go.mod gateway-service/go.sum ./gateway-service/
COPY lurus-common/go.mod lurus-common/go.sum ./lurus-common/

# Download dependencies
WORKDIR /app/gateway-service
RUN go mod download

# Copy source code
WORKDIR /app
COPY lurus-common/ ./lurus-common/
COPY gateway-service/ ./gateway-service/

# Build
WORKDIR /app/gateway-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gateway-service ./cmd/gateway

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Copy binary and config
COPY --from=builder /gateway-service .
COPY --from=builder /app/gateway-service/configs/config.yaml ./configs/

# Environment
ENV TZ=Asia/Shanghai

# Expose port
EXPOSE 18100

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:18100/health || exit 1

# Run
ENTRYPOINT ["/app/gateway-service"]
CMD ["-conf", "/app/configs/config.yaml"]
