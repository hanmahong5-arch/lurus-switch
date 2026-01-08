#!/bin/bash
# Lurus Switch Toolchain Installation Script (Linux/macOS)
# Run: ./scripts/install-tools.sh

set -e

echo "=== Lurus Switch Toolchain Installation ==="
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Check Go version
if command -v go &> /dev/null; then
    GO_VERSION=$(go version)
    echo -e "${GREEN}[OK]${NC} Go installed: $GO_VERSION"
else
    echo -e "${RED}[ERROR]${NC} Go is not installed. Please install Go 1.24+ first."
    exit 1
fi

# Check protoc
if command -v protoc &> /dev/null; then
    PROTOC_VERSION=$(protoc --version)
    echo -e "${GREEN}[OK]${NC} protoc installed: $PROTOC_VERSION"
else
    echo -e "${YELLOW}[WARN]${NC} protoc not found. Installing..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        brew install protobuf
    else
        # Linux - download from GitHub
        PROTOC_VERSION="25.1"
        curl -LO "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip"
        unzip -o "protoc-${PROTOC_VERSION}-linux-x86_64.zip" -d $HOME/.local
        rm "protoc-${PROTOC_VERSION}-linux-x86_64.zip"
        echo "export PATH=\$HOME/.local/bin:\$PATH" >> ~/.bashrc
    fi
fi

echo ""
echo -e "${CYAN}Installing Go tools...${NC}"
echo ""

# Install Kratos CLI
echo "Installing Kratos CLI..."
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest && \
    echo -e "${GREEN}[OK]${NC} Kratos CLI installed" || \
    echo -e "${RED}[ERROR]${NC} Failed to install Kratos CLI"

# Install Hertz CLI (hz)
echo "Installing Hertz CLI (hz)..."
go install github.com/cloudwego/hertz/cmd/hz@latest && \
    echo -e "${GREEN}[OK]${NC} Hertz CLI installed" || \
    echo -e "${RED}[ERROR]${NC} Failed to install Hertz CLI"

# Install Wire
echo "Installing Wire..."
go install github.com/google/wire/cmd/wire@latest && \
    echo -e "${GREEN}[OK]${NC} Wire installed" || \
    echo -e "${RED}[ERROR]${NC} Failed to install Wire"

# Install protoc-gen-go
echo "Installing protoc-gen-go..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    echo -e "${GREEN}[OK]${NC} protoc-gen-go installed" || \
    echo -e "${RED}[ERROR]${NC} Failed to install protoc-gen-go"

# Install protoc-gen-go-grpc
echo "Installing protoc-gen-go-grpc..."
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
    echo -e "${GREEN}[OK]${NC} protoc-gen-go-grpc installed" || \
    echo -e "${RED}[ERROR]${NC} Failed to install protoc-gen-go-grpc"

# Install Kratos protoc plugins
echo "Installing Kratos protoc plugins..."
go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest && \
go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest && \
    echo -e "${GREEN}[OK]${NC} Kratos protoc plugins installed" || \
    echo -e "${RED}[ERROR]${NC} Failed to install Kratos plugins"

# Install buf (optional - for proto linting)
echo "Installing buf (proto linting)..."
go install github.com/bufbuild/buf/cmd/buf@latest && \
    echo -e "${GREEN}[OK]${NC} buf installed" || \
    echo -e "${YELLOW}[WARN]${NC} Failed to install buf (optional)"

echo ""
echo -e "${CYAN}=== Verification ===${NC}"
echo ""

# Verify installations
verify_tool() {
    if command -v $1 &> /dev/null; then
        echo -e "${GREEN}[OK]${NC} $1 ready"
    else
        echo -e "${YELLOW}[WARN]${NC} $1 may not be in PATH"
    fi
}

verify_tool kratos
verify_tool hz
verify_tool wire
verify_tool protoc-gen-go
verify_tool protoc-gen-go-grpc

echo ""
echo -e "${GREEN}=== Installation Complete ===${NC}"
echo ""
echo -e "${CYAN}Next steps:${NC}"
echo "  1. Start infrastructure: cd deploy && docker-compose up -d"
echo "  2. Create services:"
echo "     - kratos new provider-service"
echo "     - kratos new log-service"
echo "     - kratos new billing-service"
echo "     - hz new -mod gateway-service"
echo ""
