.PHONY: build run test clean wire docker

# Variables
BINARY_NAME=provider-service
CONFIG_PATH=configs/config.yaml

# Build
build:
	go build -o $(BINARY_NAME) ./cmd/provider

# Build for Linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/provider

# Run
run:
	go run ./cmd/provider -conf $(CONFIG_PATH)

# Test
test:
	go test -v ./...

# Test with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean
clean:
	rm -f $(BINARY_NAME) coverage.out coverage.html

# Generate Wire
wire:
	cd cmd/provider && wire

# Go mod tidy
tidy:
	go mod tidy

# Lint
lint:
	golangci-lint run ./...

# Docker build
docker:
	cd .. && docker build -f provider-service/Dockerfile -t provider-service:latest .

# Docker run
docker-run:
	docker run -p 18101:18101 -p 19101:19101 provider-service:latest

# Generate protobuf
proto:
	protoc --go_out=. --go-grpc_out=. api/provider/v1/provider.proto

# All
all: tidy build test
