#Requires -Version 5.1
# Quick deployment script for Lurus Switch

param(
    [string]$ServerIP = "115.190.239.146",
    [string]$User = "root"
)

$RemoteDir = "/opt/lurus"

function Write-Step { param($msg) Write-Host ">>> $msg" -ForegroundColor Yellow }
function Write-OK { param($msg) Write-Host "[OK] $msg" -ForegroundColor Green }

Write-Host "`nLurus Switch Deployment to $ServerIP`n" -ForegroundColor Green

# Step 0: Check SSH
Write-Step "Step 0: Checking SSH connection..."
$test = ssh -o BatchMode=yes -o ConnectTimeout=5 $User@$ServerIP 'echo OK' 2>&1
if ($test -notlike "*OK*") {
    Write-Host "`nSSH key not configured. Please run this command in Git Bash:`n" -ForegroundColor Yellow
    Write-Host "cat ~/.ssh/id_rsa.pub | ssh $User@$ServerIP `"mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys`"`n" -ForegroundColor Cyan
    Write-Host "Password: GGsuperman1211`n" -ForegroundColor Green
    Write-Host "Press any key after running the command..." -ForegroundColor Yellow
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
}
Write-OK "SSH ready"

# Step 1: Check environment
Write-Step "Step 1: Checking server environment..."
ssh $User@$ServerIP "uname -a && docker --version"
Write-OK "Environment checked"

# Step 2: Create directories
Write-Step "Step 2: Creating directories..."
ssh $User@$ServerIP "mkdir -p $RemoteDir/deploy $RemoteDir/data $RemoteDir/logs"
Write-OK "Directories created"

# Step 3: Copy files
Write-Step "Step 3: Copying files..."
scp docker-compose.dev.yaml ${User}@${ServerIP}:${RemoteDir}/
scp -r deploy ${User}@${ServerIP}:${RemoteDir}/
Write-OK "Files copied"

# Step 4: Start infrastructure
Write-Step "Step 4: Starting infrastructure services..."
ssh $User@$ServerIP "cd $RemoteDir && docker compose -f docker-compose.dev.yaml up -d postgres redis nats clickhouse"
Write-Host "Waiting 30 seconds for services to start..." -ForegroundColor Cyan
Start-Sleep -Seconds 30

# Step 5: Start observability
Write-Step "Step 5: Starting observability services..."
ssh $User@$ServerIP "cd $RemoteDir && docker compose -f docker-compose.dev.yaml up -d jaeger prometheus grafana"
Start-Sleep -Seconds 10

# Step 6: Check status
Write-Step "Step 6: Checking status..."
ssh $User@$ServerIP "docker ps"

Write-Host "`nDeployment Complete!`n" -ForegroundColor Green
Write-Host "Access:" -ForegroundColor Cyan
Write-Host "  Grafana:    http://${ServerIP}:3000 (admin/admin)"
Write-Host "  Prometheus: http://${ServerIP}:9090"
Write-Host "  Jaeger:     http://${ServerIP}:16686"
Write-Host "`n"
