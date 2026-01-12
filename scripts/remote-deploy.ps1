# Remote Deployment Script for Lurus Switch
# Usage: .\remote-deploy.ps1

param(
    [string]$ServerIP = "115.190.239.146",
    [string]$User = "root",
    [switch]$CheckEnv,
    [switch]$DeployInfra,
    [switch]$DeployServices,
    [switch]$DeployAll
)

$ErrorActionPreference = "Stop"

function Write-ColorOutput($ForegroundColor) {
    $fc = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    if ($args) {
        Write-Output $args
    }
    $host.UI.RawUI.ForegroundColor = $fc
}

function Invoke-RemoteCommand {
    param(
        [string]$Command,
        [string]$Description
    )

    Write-ColorOutput Yellow ">>> $Description"
    Write-Host "Command: $Command" -ForegroundColor DarkGray

    # Use ssh command (requires SSH client)
    ssh -o StrictHostKeyChecking=no "${User}@${ServerIP}" "$Command"

    if ($LASTEXITCODE -ne 0) {
        Write-ColorOutput Red "Failed: $Description"
        throw "Command execution failed"
    }
}

function Copy-ToRemote {
    param(
        [string]$LocalPath,
        [string]$RemotePath,
        [string]$Description
    )

    Write-ColorOutput Yellow ">>> $Description"
    scp -o StrictHostKeyChecking=no -r "$LocalPath" "${User}@${ServerIP}:${RemotePath}"

    if ($LASTEXITCODE -ne 0) {
        Write-ColorOutput Red "Failed: $Description"
        throw "File copy failed"
    }
}

# Check environment
if ($CheckEnv -or $DeployAll) {
    Write-ColorOutput Green "`n=== Checking Remote Environment ==="

    Invoke-RemoteCommand "uname -a" "Check OS"
    Invoke-RemoteCommand "cat /etc/os-release" "Check Linux Distribution"
    Invoke-RemoteCommand "docker --version" "Check Docker"
    Invoke-RemoteCommand "docker compose version || docker-compose --version" "Check Docker Compose"
    Invoke-RemoteCommand "free -h" "Check Memory"
    Invoke-RemoteCommand "df -h /" "Check Disk Space"
    Invoke-RemoteCommand "docker ps -a" "Check Running Containers"
}

# Deploy infrastructure
if ($DeployInfra -or $DeployAll) {
    Write-ColorOutput Green "`n=== Deploying Infrastructure ==="

    # Create directories
    Invoke-RemoteCommand "mkdir -p /opt/lurus/deploy /opt/lurus/data" "Create deployment directories"

    # Copy docker compose files
    Copy-ToRemote ".\docker-compose.dev.yaml" "/opt/lurus/" "Copy Docker Compose config"
    Copy-ToRemote ".\deploy" "/opt/lurus/" "Copy deployment configs"

    # Start infrastructure services
    Invoke-RemoteCommand "cd /opt/lurus && docker compose -f docker-compose.dev.yaml up -d postgres redis nats clickhouse" "Start infrastructure services"

    # Wait for services to be healthy
    Write-ColorOutput Yellow "Waiting for services to be healthy..."
    Start-Sleep -Seconds 15

    Invoke-RemoteCommand "docker ps" "Check container status"
}

# Deploy microservices
if ($DeployServices -or $DeployAll) {
    Write-ColorOutput Green "`n=== Deploying Microservices ==="

    # Build and deploy services
    # Note: This assumes we have pre-built images or Dockerfiles

    Write-ColorOutput Yellow "Building services..."
    # TODO: Add build commands

    Write-ColorOutput Yellow "Starting services..."
    Invoke-RemoteCommand "cd /opt/lurus && docker compose -f docker-compose.dev.yaml up -d" "Start all services"

    # Wait for services
    Start-Sleep -Seconds 10

    Invoke-RemoteCommand "docker ps" "Check all containers"
}

Write-ColorOutput Green "`n=== Deployment Summary ==="
Write-Host "Server: $ServerIP"
Write-Host "Run 'ssh ${User}@${ServerIP}' to access the server"
Write-Host "Run 'docker ps' to check container status"
Write-Host "Run 'docker logs <container>' to view logs"
