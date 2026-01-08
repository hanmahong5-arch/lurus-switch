# Lurus Switch Windows Services Installation Script
# Run as Administrator

param(
    [switch]$PostgreSQL,
    [switch]$NATS,
    [switch]$Caddy,
    [switch]$All
)

$ErrorActionPreference = "Stop"
$ProgressPreference = 'SilentlyContinue'

$InstallDir = "D:\install"
$ServicesDir = "D:\services"

# Create directories
if (-not (Test-Path $InstallDir)) { New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null }
if (-not (Test-Path $ServicesDir)) { New-Item -ItemType Directory -Path $ServicesDir -Force | Out-Null }

function Install-PostgreSQL {
    Write-Host "`n=== Installing PostgreSQL ===" -ForegroundColor Cyan

    $pgVersion = "16.6-1"
    $pgUrl = "https://get.enterprisedb.com/postgresql/postgresql-$pgVersion-windows-x64.exe"
    $pgInstaller = "$InstallDir\postgresql-$pgVersion.exe"
    $pgDataDir = "D:\data\postgresql"
    $pgPassword = "lurus_pg_password_2026"

    if (Get-Service -Name "postgresql*" -ErrorAction SilentlyContinue) {
        Write-Host "PostgreSQL is already installed as a service" -ForegroundColor Yellow
        return
    }

    if (-not (Test-Path $pgInstaller)) {
        Write-Host "Downloading PostgreSQL $pgVersion..." -ForegroundColor Green
        Invoke-WebRequest -Uri $pgUrl -OutFile $pgInstaller
    }

    Write-Host "Installing PostgreSQL (this may take a few minutes)..." -ForegroundColor Green

    # Silent installation
    $args = @(
        "--mode", "unattended",
        "--superpassword", $pgPassword,
        "--servicename", "postgresql",
        "--datadir", $pgDataDir,
        "--prefix", "D:\PostgreSQL",
        "--serverport", "5432"
    )

    Start-Process -FilePath $pgInstaller -ArgumentList $args -Wait -NoNewWindow

    Write-Host "PostgreSQL installed successfully!" -ForegroundColor Green
    Write-Host "  Data directory: $pgDataDir"
    Write-Host "  Port: 5432"
    Write-Host "  Password: $pgPassword"
}

function Install-NATS {
    Write-Host "`n=== Installing NATS Server ===" -ForegroundColor Cyan

    $natsVersion = "2.10.24"
    $natsUrl = "https://github.com/nats-io/nats-server/releases/download/v$natsVersion/nats-server-v$natsVersion-windows-amd64.zip"
    $natsZip = "$InstallDir\nats-server-$natsVersion.zip"
    $natsDir = "$ServicesDir\nats"

    if (Test-Path "$natsDir\nats-server.exe") {
        Write-Host "NATS Server is already installed at $natsDir" -ForegroundColor Yellow
        return
    }

    if (-not (Test-Path $natsZip)) {
        Write-Host "Downloading NATS Server $natsVersion..." -ForegroundColor Green
        Invoke-WebRequest -Uri $natsUrl -OutFile $natsZip
    }

    Write-Host "Extracting NATS Server..." -ForegroundColor Green
    Expand-Archive -Path $natsZip -DestinationPath $InstallDir -Force

    # Move to services directory
    New-Item -ItemType Directory -Path $natsDir -Force | Out-Null
    Move-Item "$InstallDir\nats-server-v$natsVersion-windows-amd64\nats-server.exe" "$natsDir\" -Force

    # Create data directory
    $natsDataDir = "D:\data\nats"
    New-Item -ItemType Directory -Path $natsDataDir -Force | Out-Null

    # Create configuration file
    $natsConfig = @"
# NATS Server Configuration for Lurus Switch
port: 4222
http_port: 8222

# JetStream configuration
jetstream {
    store_dir: "D:/data/nats/jetstream"
    max_mem: 1G
    max_file: 10G
}

# Logging
log_file: "D:/data/nats/nats.log"
debug: false
trace: false
"@

    Set-Content -Path "$natsDir\nats-server.conf" -Value $natsConfig -Encoding UTF8

    Write-Host "NATS Server installed successfully!" -ForegroundColor Green
    Write-Host "  Install directory: $natsDir"
    Write-Host "  Config file: $natsDir\nats-server.conf"
    Write-Host "  Data directory: $natsDataDir"
    Write-Host ""
    Write-Host "To start NATS manually: $natsDir\nats-server.exe -c $natsDir\nats-server.conf"
}

function Install-Caddy {
    Write-Host "`n=== Installing Caddy ===" -ForegroundColor Cyan

    $caddyUrl = "https://caddyserver.com/api/download?os=windows&arch=amd64"
    $caddyDir = "$ServicesDir\caddy"
    $caddyExe = "$caddyDir\caddy.exe"

    if (Test-Path $caddyExe) {
        Write-Host "Caddy is already installed at $caddyDir" -ForegroundColor Yellow
        return
    }

    New-Item -ItemType Directory -Path $caddyDir -Force | Out-Null

    Write-Host "Downloading Caddy..." -ForegroundColor Green
    Invoke-WebRequest -Uri $caddyUrl -OutFile $caddyExe

    # Create data directories
    $caddyDataDir = "D:\data\caddy"
    New-Item -ItemType Directory -Path $caddyDataDir -Force | Out-Null
    New-Item -ItemType Directory -Path "$caddyDataDir\certificates" -Force | Out-Null
    New-Item -ItemType Directory -Path "$caddyDataDir\logs" -Force | Out-Null

    # Copy Caddyfile if exists
    $sourceCaddyfile = "D:\tools\lurus-switch\deploy\caddy\Caddyfile"
    if (Test-Path $sourceCaddyfile) {
        Copy-Item $sourceCaddyfile "$caddyDir\Caddyfile" -Force
        Write-Host "  Copied Caddyfile from deploy/caddy/"
    }

    Write-Host "Caddy installed successfully!" -ForegroundColor Green
    Write-Host "  Install directory: $caddyDir"
    Write-Host "  Data directory: $caddyDataDir"
    Write-Host ""
    Write-Host "To start Caddy manually: $caddyExe run --config $caddyDir\Caddyfile"
}

# Main execution
if ($All -or (-not $PostgreSQL -and -not $NATS -and -not $Caddy)) {
    # Install all if no specific flag or -All specified
    Install-PostgreSQL
    Install-NATS
    Install-Caddy
} else {
    if ($PostgreSQL) { Install-PostgreSQL }
    if ($NATS) { Install-NATS }
    if ($Caddy) { Install-Caddy }
}

Write-Host "`n=== Installation Complete ===" -ForegroundColor Cyan
Write-Host @"

Next steps:
1. Start PostgreSQL service:
   Start-Service postgresql

2. Create databases:
   psql -U postgres -f D:\tools\lurus-switch\deploy\postgres\init-databases.sql

3. Start NATS:
   D:\services\nats\nats-server.exe -c D:\services\nats\nats-server.conf

4. Configure Caddyfile with your domain settings, then start:
   D:\services\caddy\caddy.exe run --config D:\services\caddy\Caddyfile

5. Build and start Go services (new-api, gateway-service, sync-service)
"@
