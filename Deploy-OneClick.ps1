#Requires -Version 5.1
<#
.SYNOPSIS
    One-click deployment script for Lurus Switch to remote server
.DESCRIPTION
    This script automates the deployment of Lurus Switch microservices infrastructure
    to server 115.190.239.146
#>

param(
    [string]$ServerIP = "115.190.239.146",
    [string]$User = "root",
    [switch]$SkipSSHSetup
)

$ErrorActionPreference = "Continue"
$Password = "GGsuperman1211"
$RemoteDir = "/opt/lurus"

# Color functions
function Write-Step { param($msg) Write-Host ">>> $msg" -ForegroundColor Yellow }
function Write-Success { param($msg) Write-Host "✓ $msg" -ForegroundColor Green }
function Write-Info { param($msg) Write-Host $msg -ForegroundColor Cyan }

Write-Host "`n================================================" -ForegroundColor Green
Write-Host "  Lurus Switch 一键部署脚本" -ForegroundColor Green
Write-Host "  目标服务器: $ServerIP" -ForegroundColor Green
Write-Host "================================================`n" -ForegroundColor Green

# Step 0: Check SSH key setup
if (-not $SkipSSHSetup) {
    Write-Step "步骤 0: 检查 SSH 密钥配置"

    $testCmd = "ssh -o BatchMode=yes -o ConnectTimeout=5 $User@$ServerIP 'echo OK' 2>&1"
    $testResult = Invoke-Expression $testCmd

    if ($testResult -notlike "*OK*") {
        Write-Host "`n需要配置 SSH 密钥。请在新的 Git Bash 或 PowerShell 窗口中执行以下命令:" -ForegroundColor Yellow
        Write-Host "`n命令:" -ForegroundColor Cyan
        $sshKeyCmd = "cat ~/.ssh/id_rsa.pub | ssh $User@$ServerIP `"mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys`""
        Write-Host $sshKeyCmd -ForegroundColor White

        Write-Host "`n密码: " -NoNewline -ForegroundColor Yellow
        Write-Host $Password -ForegroundColor Green

        Write-Host "`n执行完成后,按任意键继续..." -ForegroundColor Yellow
        $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")

        # Verify again
        $testResult2 = Invoke-Expression $testCmd
        if ($testResult2 -notlike "*OK*") {
            Write-Host "`n✗ SSH 密钥配置失败,请检查后重试" -ForegroundColor Red
            exit 1
        }
    }

    Write-Success "SSH 密钥已配置"
}

# Step 1: Check environment
Write-Step "`n步骤 1: 检查服务器环境"
ssh $User@$ServerIP @"
echo '=== 系统信息 ===' &&
uname -a &&
cat /etc/os-release | head -3 &&
echo '' &&
echo '=== Docker 版本 ===' &&
docker --version &&
docker compose version &&
echo '' &&
echo '=== 资源情况 ===' &&
echo '内存:' && free -h | grep Mem &&
echo '磁盘:' && df -h / | tail -1
"@

if ($LASTEXITCODE -ne 0) {
    Write-Host "环境检查失败" -ForegroundColor Red
    exit 1
}

Write-Info "`n继续部署? (按 Y 继续, 其他键取消)"
$confirm = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
if ($confirm.Character -ne 'y' -and $confirm.Character -ne 'Y') {
    Write-Host "部署已取消" -ForegroundColor Yellow
    exit 0
}

# Step 2: Create directories
Write-Step "`n步骤 2: 创建部署目录"
ssh $User@$ServerIP "mkdir -p $RemoteDir/{deploy,data,logs,backup}"
Write-Success "目录创建完成"

# Step 3: Copy files
Write-Step "`n步骤 3: 复制配置文件"
scp docker-compose.dev.yaml ${User}@${ServerIP}:${RemoteDir}/
scp -r deploy ${User}@${ServerIP}:${RemoteDir}/ 2>$null
if (Test-Path "prometheus.yml") {
    scp prometheus.yml ${User}@${ServerIP}:${RemoteDir}/
}
Write-Success "文件复制完成"

# Step 4: Create .env file
Write-Step "`n步骤 4: 创建环境配置文件"
$envContent = @"
POSTGRES_USER=lurus
POSTGRES_PASSWORD=lurus_prod_$(Get-Random)
POSTGRES_DB=lurus
CLICKHOUSE_USER=lurus
CLICKHOUSE_PASSWORD=lurus_click_$(Get-Random)
CLICKHOUSE_DB=lurus_logs
GF_SECURITY_ADMIN_PASSWORD=admin
"@

$envContent | Out-File -FilePath "$env:TEMP\lurus.env" -Encoding UTF8 -NoNewline
scp "$env:TEMP\lurus.env" ${User}@${ServerIP}:${RemoteDir}/.env
Remove-Item "$env:TEMP\lurus.env"
Write-Success "环境配置完成"

# Step 5: Pull images
Write-Step "`n步骤 5: 拉取 Docker 镜像 (可能需要几分钟)"
ssh $User@$ServerIP "cd $RemoteDir && docker compose -f docker-compose.dev.yaml pull"
Write-Success "镜像拉取完成"

# Step 6: Start infrastructure
Write-Step "`n步骤 6: 启动基础设施服务 (PostgreSQL, Redis, NATS, ClickHouse)"
ssh $User@$ServerIP "cd $RemoteDir && docker compose -f docker-compose.dev.yaml up -d postgres redis nats clickhouse consul"

Write-Info "等待服务初始化 (30秒)..."
1..30 | ForEach-Object {
    Write-Progress -Activity "等待服务启动" -Status "$_ / 30" -PercentComplete ($_ / 30 * 100)
    Start-Sleep -Seconds 1
}

# Step 7: Start observability services
Write-Step "`n步骤 7: 启动监控服务 (Prometheus, Grafana, Jaeger)"
ssh $User@$ServerIP "cd $RemoteDir && docker compose -f docker-compose.dev.yaml up -d jaeger prometheus grafana alertmanager"

Write-Info "等待监控服务启动 (10秒)..."
Start-Sleep -Seconds 10

# Step 8: Check status
Write-Step "`n步骤 8: 检查部署状态"
ssh $User@$ServerIP "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"

# Step 9: Initialize databases
Write-Step "`n步骤 9: 初始化数据库"
ssh $User@$ServerIP "docker exec lurus-postgres psql -U lurus -c '\l'"

# Summary
Write-Host "`n================================================" -ForegroundColor Green
Write-Host "  部署完成!" -ForegroundColor Green
Write-Host "================================================`n" -ForegroundColor Green

Write-Host "服务访问地址:" -ForegroundColor Cyan
Write-Host "  Grafana:    http://${ServerIP}:3000 (admin/admin)" -ForegroundColor White
Write-Host "  Prometheus: http://${ServerIP}:9090" -ForegroundColor White
Write-Host "  Jaeger:     http://${ServerIP}:16686" -ForegroundColor White
Write-Host "  Consul:     http://${ServerIP}:8500" -ForegroundColor White

Write-Host "`n数据库连接:" -ForegroundColor Cyan
Write-Host "  PostgreSQL: ${ServerIP}:5432" -ForegroundColor White
Write-Host "  Redis:      ${ServerIP}:6379" -ForegroundColor White
Write-Host "  NATS:       ${ServerIP}:4222" -ForegroundColor White
Write-Host "  ClickHouse: ${ServerIP}:8123 (HTTP), ${ServerIP}:9000 (Native)" -ForegroundColor White

Write-Host "`n查看日志:" -ForegroundColor Cyan
Write-Host "  ssh $User@$ServerIP 'docker logs -f <container-name>'" -ForegroundColor White

Write-Host "`n后续步骤:" -ForegroundColor Cyan
Write-Host "  1. 访问 Grafana 查看监控面板" -ForegroundColor White
Write-Host "  2. 构建并部署微服务 (参见 DEPLOYMENT-STEPS.md)" -ForegroundColor White
Write-Host "  3. 配置域名和 HTTPS (可选)" -ForegroundColor White

Write-Host "`n"
