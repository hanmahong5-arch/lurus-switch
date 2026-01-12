#!/usr/bin/env pwsh
# PowerShell Deployment Script for Lurus Switch
# Usage: .\Deploy-ToServer.ps1 -SetupSSH -DeployInfra -DeployServices

param(
    [string]$ServerIP = "115.190.239.146",
    [string]$ServerUser = "root",
    [string]$ServerPassword = "GGsuperman1211",
    [switch]$SetupSSH,
    [switch]$CheckEnv,
    [switch]$DeployInfra,
    [switch]$DeployServices,
    [switch]$DeployAll
)

$ErrorActionPreference = "Continue"
$RemoteDir = "/opt/lurus"

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host ">>> $Message" -ForegroundColor $Color
}

function Invoke-SSHCommand {
    param(
        [string]$Command,
        [string]$Description = "",
        [switch]$UsePassword
    )

    if ($Description) {
        Write-ColorOutput $Description "Yellow"
    }

    if ($UsePassword) {
        # Use echo to pipe password (not recommended for production)
        $result = echo $ServerPassword | ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 "${ServerUser}@${ServerIP}" "$Command" 2>&1
    } else {
        $result = ssh -o BatchMode=yes -o ConnectTimeout=10 "${ServerUser}@${ServerIP}" "$Command" 2>&1
    }

    if ($LASTEXITCODE -ne 0 -and !$UsePassword) {
        Write-ColorOutput "SSH command failed (may need password)" "Red"
        return $null
    }

    return $result
}

# Setup SSH Key Authentication
if ($SetupSSH -or $DeployAll) {
    Write-ColorOutput "=== Setting up SSH Key Authentication ===" "Green"

    # Test if SSH key auth already works
    $testResult = Invoke-SSHCommand "echo 'OK'" "Testing SSH connection"

    if ($null -eq $testResult -or $testResult -notcontains "OK") {
        Write-ColorOutput "SSH key not configured. Setting up..." "Yellow"

        # Read public key
        $pubKey = Get-Content "$env:USERPROFILE\.ssh\id_rsa.pub" -Raw -ErrorAction Stop

        # Create authorized_keys entry
        $setupCmd = "mkdir -p ~/.ssh && chmod 700 ~/.ssh && echo '$pubKey' >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && echo 'SSH key added'"

        Write-ColorOutput "Copying SSH public key to server..." "Yellow"
        Write-Host "You may need to enter password: $ServerPassword" -ForegroundColor DarkGray

        # Use interactive SSH for this one-time setup
        $sshCmd = "ssh -o StrictHostKeyChecking=no ${ServerUser}@${ServerIP} `"$setupCmd`""
        Write-Host "Please run this command and enter password when prompted:" -ForegroundColor Cyan
        Write-Host $sshCmd -ForegroundColor White
        Write-Host ""
        Write-Host "Password: $ServerPassword" -ForegroundColor Green
        Write-Host ""
        Write-Host "After running the command, press Enter to continue..." -ForegroundColor Yellow
        $null = Read-Host

        # Test again
        $testResult2 = Invoke-SSHCommand "echo 'SSH OK'" "Testing SSH after key setup"
        if ($testResult2 -contains "SSH OK") {
            Write-ColorOutput "✓ SSH key authentication working!" "Green"
        } else {
            Write-ColorOutput "SSH key setup may have failed. Please check." "Red"
            exit 1
        }
    } else {
        Write-ColorOutput "✓ SSH key authentication already configured" "Green"
    }
}

# Check Environment
if ($CheckEnv -or $DeployAll) {
    Write-ColorOutput "`n=== Checking Server Environment ===" "Green"

    Invoke-SSHCommand "uname -a" "OS Information"
    Invoke-SSHCommand "cat /etc/os-release | head -5" "Linux Distribution"
    Invoke-SSHCommand "docker --version" "Docker Version"
    Invoke-SSHCommand "docker compose version 2>&1 || docker-compose --version" "Docker Compose Version"
    Invoke-SSHCommand "free -h | grep -E 'Mem|Swap'" "Memory"
    Invoke-SSHCommand "df -h / | tail -1" "Disk Space"
    Invoke-SSHCommand "docker ps -a --format 'table {{.Names}}\t{{.Status}}' 2>&1 || echo 'No containers'" "Existing Containers"
}

# Deploy Infrastructure
if ($DeployInfra -or $DeployAll) {
    Write-ColorOutput "`n=== Deploying Infrastructure ===" "Green"

    # Create directories
    Invoke-SSHCommand "mkdir -p $RemoteDir/{deploy,data,logs,backup}" "Creating directories"

    # Copy files
    Write-ColorOutput "Copying configuration files..." "Yellow"
    scp -o StrictHostKeyChecking=no -r docker-compose.dev.yaml "${ServerUser}@${ServerIP}:${RemoteDir}/"
    scp -o StrictHostKeyChecking=no -r deploy "${ServerUser}@${ServerIP}:${RemoteDir}/"

    # Create .env file
    Write-ColorOutput "Creating environment file..." "Yellow"
    $envContent = @"
POSTGRES_USER=lurus
POSTGRES_PASSWORD=lurus_prod_$(openssl rand -hex 16)
POSTGRES_DB=lurus
CLICKHOUSE_USER=lurus
CLICKHOUSE_PASSWORD=lurus_click_$(openssl rand -hex 16)
CLICKHOUSE_DB=lurus_logs
GF_SECURITY_ADMIN_PASSWORD=admin
"@
    $envContent | Out-File -FilePath "$env:TEMP\lurus.env" -Encoding UTF8
    scp -o StrictHostKeyChecking=no "$env:TEMP\lurus.env" "${ServerUser}@${ServerIP}:${RemoteDir}/.env"

    # Pull images
    Invoke-SSHCommand "cd $RemoteDir && docker compose -f docker-compose.dev.yaml pull" "Pulling Docker images"

    # Start infrastructure
    Invoke-SSHCommand "cd $RemoteDir && docker compose -f docker-compose.dev.yaml up -d postgres redis nats clickhouse consul" "Starting infrastructure services"

    Write-ColorOutput "Waiting 20 seconds for services to initialize..." "Yellow"
    Start-Sleep -Seconds 20

    # Check status
    Invoke-SSHCommand "docker ps --format 'table {{.Names}}\t{{.Status}}'" "Checking container status"

    # Initialize databases
    Write-ColorOutput "Initializing databases..." "Yellow"
    Invoke-SSHCommand "docker exec lurus-postgres psql -U lurus -c '\l'" "Listing databases"
}

# Deploy Microservices
if ($DeployServices -or $DeployAll) {
    Write-ColorOutput "`n=== Deploying Microservices ===" "Green"

    # Note: This requires built images
    Write-ColorOutput "Starting all microservices..." "Yellow"
    Invoke-SSHCommand "cd $RemoteDir && docker compose -f docker-compose.dev.yaml up -d" "Starting all services"

    Start-Sleep -Seconds 15

    Invoke-SSHCommand "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'" "Final container status"
}

Write-ColorOutput "`n=== Deployment Summary ===" "Green"
Write-Host "Server: $ServerIP" -ForegroundColor Cyan
Write-Host ""
Write-Host "Access the server:" -ForegroundColor Yellow
Write-Host "  ssh ${ServerUser}@${ServerIP}" -ForegroundColor White
Write-Host ""
Write-Host "Check services:" -ForegroundColor Yellow
Write-Host "  docker ps" -ForegroundColor White
Write-Host "  docker logs -f lurus-postgres" -ForegroundColor White
Write-Host ""
Write-Host "Service URLs (if accessible):" -ForegroundColor Yellow
Write-Host "  Grafana: http://${ServerIP}:3000 (admin/admin)" -ForegroundColor White
Write-Host "  Prometheus: http://${ServerIP}:9090" -ForegroundColor White
Write-Host "  Jaeger: http://${ServerIP}:16686" -ForegroundColor White
Write-Host "  Consul: http://${ServerIP}:8500" -ForegroundColor White
