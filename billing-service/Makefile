.PHONY: all build run test clean docker proto

# Variables
APP_NAME := billing-service
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_FLAGS := -ldflags="-w -s -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Default target
all: build

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	go build $(GO_FLAGS) -o bin/$(APP_NAME) ./cmd/billing

# Run the application
run:
	@echo "Running $(APP_NAME)..."
	go run ./cmd/billing -conf configs/config.yaml

# Run with debug mode
run-debug:
	@echo "Running $(APP_NAME) in debug mode..."
	DEBUG=true go run ./cmd/billing -conf configs/config.yaml

# Run tests
test:
	@echo "Running tests..."
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run ./...

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download

# Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .

# Build Docker image with latest tag
docker-latest:
	@echo "Building Docker image with latest tag..."
	docker build -t $(APP_NAME):latest .

# Push Docker image
docker-push:
	@echo "Pushing Docker image..."
	docker push $(APP_NAME):$(VERSION)

# Run in Docker
docker-run:
	@echo "Running in Docker..."
	docker run -p 18103:18103 --rm $(APP_NAME):latest

# Generate wire dependency injection
wire:
	@echo "Generating wire..."
	wire ./internal/...

# Database migration (if using migration tool)
migrate-up:
	@echo "Running migrations..."
	# migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	@echo "Rolling back migrations..."
	# migrate -path migrations -database "$(DATABASE_URL)" down 1

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  run           - Run the application"
	@echo "  run-debug     - Run with debug logging"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  clean         - Clean build artifacts"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  tidy          - Tidy dependencies"
	@echo "  deps          - Download dependencies"
	@echo "  docker        - Build Docker image"
	@echo "  docker-run    - Run in Docker"
	@echo "  wire          - Generate wire DI"
	@echo "  help          - Show this help"
