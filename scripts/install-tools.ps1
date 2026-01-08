# Lurus Switch Toolchain Installation Script (Windows PowerShell)
# Run: .\scripts\install-tools.ps1

$ErrorActionPreference = "Stop"

Write-Host "=== Lurus Switch Toolchain Installation ===" -ForegroundColor Cyan
Write-Host ""

# Check Go version
$goVersion = go version 2>$null
if (-not $goVersion) {
    Write-Host "[ERROR] Go is not installed. Please install Go 1.24+ first." -ForegroundColor Red
    exit 1
}
Write-Host "[OK] Go installed: $goVersion" -ForegroundColor Green

# Check protoc
$protocVersion = protoc --version 2>$null
if (-not $protocVersion) {
    Write-Host "[WARN] protoc not found. Installing via winget..." -ForegroundColor Yellow
    winget install Google.Protobuf --silent
} else {
    Write-Host "[OK] protoc installed: $protocVersion" -ForegroundColor Green
}

Write-Host ""
Write-Host "Installing Go tools..." -ForegroundColor Cyan
Write-Host ""

# Install Kratos CLI
Write-Host "Installing Kratos CLI..." -ForegroundColor White
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] Kratos CLI installed" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Failed to install Kratos CLI" -ForegroundColor Red
}

# Install Hertz CLI (hz)
Write-Host "Installing Hertz CLI (hz)..." -ForegroundColor White
go install github.com/cloudwego/hertz/cmd/hz@latest
if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] Hertz CLI installed" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Failed to install Hertz CLI" -ForegroundColor Red
}

# Install Wire
Write-Host "Installing Wire..." -ForegroundColor White
go install github.com/google/wire/cmd/wire@latest
if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] Wire installed" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Failed to install Wire" -ForegroundColor Red
}

# Install protoc-gen-go
Write-Host "Installing protoc-gen-go..." -ForegroundColor White
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] protoc-gen-go installed" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Failed to install protoc-gen-go" -ForegroundColor Red
}

# Install protoc-gen-go-grpc
Write-Host "Installing protoc-gen-go-grpc..." -ForegroundColor White
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] protoc-gen-go-grpc installed" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Failed to install protoc-gen-go-grpc" -ForegroundColor Red
}

# Install Kratos protoc plugins
Write-Host "Installing Kratos protoc plugins..." -ForegroundColor White
go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest
if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] Kratos protoc plugins installed" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Failed to install Kratos plugins" -ForegroundColor Red
}

# Install buf (optional - for proto linting)
Write-Host "Installing buf (proto linting)..." -ForegroundColor White
go install github.com/bufbuild/buf/cmd/buf@latest
if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] buf installed" -ForegroundColor Green
} else {
    Write-Host "[WARN] Failed to install buf (optional)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Verification ===" -ForegroundColor Cyan
Write-Host ""

# Verify installations
$tools = @(
    @{Name="kratos"; Check="kratos --version"},
    @{Name="hz"; Check="hz --version"},
    @{Name="wire"; Check="wire help"},
    @{Name="protoc-gen-go"; Check="protoc-gen-go --version"},
    @{Name="protoc-gen-go-grpc"; Check="protoc-gen-go-grpc --version"}
)

foreach ($tool in $tools) {
    $result = Invoke-Expression $tool.Check 2>$null
    if ($LASTEXITCODE -eq 0 -or $result) {
        Write-Host "[OK] $($tool.Name) ready" -ForegroundColor Green
    } else {
        Write-Host "[WARN] $($tool.Name) may not be in PATH" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "=== Installation Complete ===" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "  1. Start infrastructure: cd deploy && docker-compose up -d"
Write-Host "  2. Create services:"
Write-Host "     - kratos new provider-service"
Write-Host "     - kratos new log-service"
Write-Host "     - kratos new billing-service"
Write-Host "     - hz new -mod gateway-service"
Write-Host ""
