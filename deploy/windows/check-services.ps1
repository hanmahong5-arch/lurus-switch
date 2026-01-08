# Check existing services for Lurus Switch
Write-Host "=== Checking Lurus Switch Prerequisites ===" -ForegroundColor Cyan

# Check services
Write-Host "`n[Services]" -ForegroundColor Yellow
$services = @('postgresql*', 'memurai*', 'redis*', 'nats*', 'docker')
foreach ($svc in $services) {
    $found = Get-Service -Name $svc -ErrorAction SilentlyContinue
    if ($found) {
        Write-Host "  [OK] $($found.DisplayName): $($found.Status)" -ForegroundColor Green
    } else {
        Write-Host "  [--] ${svc} - Not installed" -ForegroundColor Gray
    }
}

# Check PostgreSQL
Write-Host "`n[PostgreSQL]" -ForegroundColor Yellow
$pgPaths = @(
    'C:\Program Files\PostgreSQL',
    'D:\PostgreSQL',
    'D:\tools\postgresql'
)
$pgFound = $false
foreach ($p in $pgPaths) {
    if (Test-Path $p) {
        Write-Host "  [OK] Found at: $p" -ForegroundColor Green
        $pgFound = $true
        break
    }
}
if (-not $pgFound) {
    Write-Host "  [--] Not found" -ForegroundColor Gray
}

# Check Redis
Write-Host "`n[Redis/Memurai]" -ForegroundColor Yellow
$redisPaths = @(
    'C:\Program Files\Memurai',
    'C:\Program Files\Redis',
    'D:\tools\redis'
)
$redisFound = $false
foreach ($p in $redisPaths) {
    if (Test-Path $p) {
        Write-Host "  [OK] Found at: $p" -ForegroundColor Green
        $redisFound = $true
        break
    }
}
if (-not $redisFound) {
    Write-Host "  [--] Not found" -ForegroundColor Gray
}

# Check NATS
Write-Host "`n[NATS Server]" -ForegroundColor Yellow
$natsPaths = @(
    'D:\services\nats',
    'D:\tools\nats',
    'C:\nats'
)
$natsFound = $false
foreach ($p in $natsPaths) {
    if (Test-Path $p) {
        Write-Host "  [OK] Found at: $p" -ForegroundColor Green
        $natsFound = $true
        break
    }
}
if (-not $natsFound) {
    Write-Host "  [--] Not found" -ForegroundColor Gray
}

# Check Go
Write-Host "`n[Go]" -ForegroundColor Yellow
try {
    $goVersion = & go version 2>&1
    Write-Host "  [OK] $goVersion" -ForegroundColor Green
} catch {
    Write-Host "  [--] Not found" -ForegroundColor Gray
}

# Check Caddy
Write-Host "`n[Caddy]" -ForegroundColor Yellow
$caddyPaths = @(
    'D:\services\caddy\caddy.exe',
    'D:\tools\caddy\caddy.exe',
    'C:\caddy\caddy.exe'
)
$caddyFound = $false
foreach ($p in $caddyPaths) {
    if (Test-Path $p) {
        Write-Host "  [OK] Found at: $p" -ForegroundColor Green
        $caddyFound = $true
        break
    }
}
if (-not $caddyFound) {
    Write-Host "  [--] Not found" -ForegroundColor Gray
}

# Summary
Write-Host "`n=== Summary ===" -ForegroundColor Cyan
Write-Host "PostgreSQL: $(if($pgFound){'Installed'}else{'Need to install'})"
Write-Host "Redis:      $(if($redisFound){'Installed'}else{'Need to install'})"
Write-Host "NATS:       $(if($natsFound){'Installed'}else{'Need to install'})"
Write-Host "Caddy:      $(if($caddyFound){'Installed'}else{'Need to install'})"
