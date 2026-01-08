.PHONY: build run test clean docker

# Application name
APP_NAME := gateway-service
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go build flags
LDFLAGS := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)

# Build the application
build:
	go build -ldflags "$(LDFLAGS)" -o $(APP_NAME) ./cmd/gateway

# Build for Windows
build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(APP_NAME).exe ./cmd/gateway

# Build for Linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(APP_NAME) ./cmd/gateway

# Run the application
run:
	go run ./cmd/gateway -conf configs/config.yaml

# Run tests
test:
	go test -v -cover ./...

# Clean build artifacts
clean:
	rm -f $(APP_NAME) $(APP_NAME).exe
	go clean -cache

# Update dependencies
deps:
	go mod tidy
	go mod verify

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Lint code
lint:
	golangci-lint run ./...

# Docker build
docker:
	docker build -t $(APP_NAME):$(VERSION) .

# Docker run
docker-run:
	docker run -p 18100:18100 \
		-e NATS_URL=nats://host.docker.internal:4222 \
		-e PROVIDER_ENDPOINT=http://host.docker.internal:18101 \
		-e BILLING_ENDPOINT=http://host.docker.internal:18103 \
		$(APP_NAME):$(VERSION)

# Generate Hertz code (if using hz CLI)
generate:
	hz update -idl api/gateway.thrift

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build for current platform"
	@echo "  build-windows  - Build for Windows"
	@echo "  build-linux    - Build for Linux"
	@echo "  run            - Run the application"
	@echo "  test           - Run tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Update dependencies"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  docker         - Build Docker image"
	@echo "  docker-run     - Run Docker container"
