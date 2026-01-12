# CodeSwitch Environment Health Check Script
# Verifies all services are running and accessible

param(
    [string]$PostgresPassword = "CodeSwitch_Dev_2026!",
    [string]$RedisPassword = "Redis_Dev_2026!",
    [switch]$Detailed,
    [switch]$ContinuousMonitor,
    [int]$Interval = 10
)

$ErrorActionPreference = "SilentlyContinue"

function Write-ServiceStatus {
    param(
        [string]$ServiceName,
        [string]$Status,  # OK, WARN, FAIL, SKIP
        [string]$Message = ""
    )

    $padding = 25 - $ServiceName.Length
    if ($padding -lt 1) { $padding = 1 }

    Write-Host "  $ServiceName" -NoNewline
    Write-Host (" " * $padding) -NoNewline

    switch ($Status) {
        "OK" { Write-Host "[ OK ]" -ForegroundColor Green -NoNewline }
        "WARN" { Write-Host "[WARN]" -ForegroundColor Yellow -NoNewline }
        "FAIL" { Write-Host "[FAIL]" -ForegroundColor Red -NoNewline }
        "SKIP" { Write-Host "[SKIP]" -ForegroundColor Gray -NoNewline }
    }

    if ($Message) {
        Write-Host " $Message" -ForegroundColor Gray
    } else {
        Write-Host ""
    }
}

function Test-AllServices {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "  CodeSwitch Health Check" -ForegroundColor Cyan
    Write-Host "  Time: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host ""

    # Infrastructure Layer
    Write-Host "Infrastructure Layer:" -ForegroundColor Yellow
    Write-Host ""

    # PostgreSQL
    try {
        $env:PGPASSWORD = $PostgresPassword
        $result = & psql -h localhost -U codeswitch -d codeswitch -c "SELECT 1;" 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-ServiceStatus "PostgreSQL" "OK" ":5432"

            if ($Detailed) {
                $dbCount = & psql -h localhost -U codeswitch -d postgres -t -c "SELECT count(*) FROM pg_database WHERE datname IN ('codeswitch','casdoor','lago','new_api');" 2>&1
                if ($dbCount) {
                    Write-Host "    Databases: $($dbCount.Trim())/4" -ForegroundColor Gray
                }
            }
        } else {
            Write-ServiceStatus "PostgreSQL" "FAIL" "Connection failed"
        }
    } catch {
        Write-ServiceStatus "PostgreSQL" "FAIL" "psql not found or connection failed"
    }

    # Redis
    try {
        $result = & redis-cli -a $RedisPassword ping 2>&1
        if ($result -eq "PONG") {
            Write-ServiceStatus "Redis" "OK" ":6379"

            if ($Detailed) {
                $dbsize = & redis-cli -a $RedisPassword dbsize 2>&1
                if ($dbsize) {
                    Write-Host "    Keys: $dbsize" -ForegroundColor Gray
                }
            }
        } else {
            Write-ServiceStatus "Redis" "FAIL"
        }
    } catch {
        Write-ServiceStatus "Redis" "FAIL" "redis-cli not found"
    }

    # NATS
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8222/healthz" -TimeoutSec 3
        Write-ServiceStatus "NATS" "OK" ":4222 (Monitor :8222)"

        if ($Detailed) {
            $varz = Invoke-RestMethod -Uri "http://localhost:8222/varz" -TimeoutSec 3
            if ($varz) {
                Write-Host "    Version: $($varz.version)" -ForegroundColor Gray
                Write-Host "    Connections: $($varz.connections)" -ForegroundColor Gray
            }
        }
    } catch {
        Write-ServiceStatus "NATS" "FAIL" "Not running"
    }

    # ClickHouse (Optional)
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8123/ping" -TimeoutSec 3
        Write-ServiceStatus "ClickHouse" "OK" ":8123/:9000"
    } catch {
        Write-ServiceStatus "ClickHouse" "SKIP" "Optional service"
    }

    Write-Host ""

    # Application Layer
    Write-Host "Application Layer:" -ForegroundColor Yellow
    Write-Host ""

    # Casdoor
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8000" -TimeoutSec 3
        if ($response.StatusCode -eq 200) {
            Write-ServiceStatus "Casdoor" "OK" ":8000"
        } else {
            Write-ServiceStatus "Casdoor" "WARN" "Unexpected status: $($response.StatusCode)"
        }
    } catch {
        Write-ServiceStatus "Casdoor" "SKIP" "Not running or optional"
    }

    # Lago API
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:3001/health" -TimeoutSec 3
        Write-ServiceStatus "Lago API" "OK" ":3001"
    } catch {
        Write-ServiceStatus "Lago API" "SKIP" "Not running or optional"
    }

    # Lago Frontend
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8080" -TimeoutSec 3
        if ($response.StatusCode -eq 200) {
            Write-ServiceStatus "Lago Frontend" "OK" ":8080"
        }
    } catch {
        Write-ServiceStatus "Lago Frontend" "SKIP" "Not running or optional"
    }

    # NEW-API
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:3000" -TimeoutSec 3
        if ($response.StatusCode -eq 200) {
            Write-ServiceStatus "NEW-API" "OK" ":3000"

            if ($Detailed) {
                try {
                    $status = Invoke-RestMethod -Uri "http://localhost:3000/api/status" -TimeoutSec 3
                    if ($status -and $status.version) {
                        Write-Host "    Version: $($status.version)" -ForegroundColor Gray
                    }
                } catch {}
            }
        }
    } catch {
        Write-ServiceStatus "NEW-API" "SKIP" "Not running or optional"
    }

    # Sync Service
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8081/api/v1/admin/system/status" -TimeoutSec 3
        Write-ServiceStatus "Sync Service" "OK" ":8081"

        if ($Detailed -and $response) {
            Write-Host "    Uptime: $($response.uptime)" -ForegroundColor Gray
            Write-Host "    NATS: $($response.nats_status)" -ForegroundColor Gray
        }
    } catch {
        Write-ServiceStatus "Sync Service" "SKIP" "Not running or optional"
    }

    # CodeSwitch Gateway
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:18100/health" -TimeoutSec 3
        if ($response.StatusCode -eq 200) {
            Write-ServiceStatus "CodeSwitch Gateway" "OK" ":18100"
        }
    } catch {
        Write-ServiceStatus "CodeSwitch Gateway" "SKIP" "Desktop app not running"
    }

    Write-Host ""

    # Docker Containers (if Docker is available)
    Write-Host "Docker Containers:" -ForegroundColor Yellow
    Write-Host ""
    try {
        $containers = docker ps --filter "name=lurus" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>&1
        if ($LASTEXITCODE -eq 0 -and $containers) {
            $containers | Out-String | ForEach-Object {
                $_.Split("`n") | Where-Object { $_.Trim() -and $_ -notmatch "^NAMES" } | ForEach-Object {
                    Write-Host "  $_" -ForegroundColor Gray
                }
            }
        } else {
            Write-Host "  No containers running" -ForegroundColor Gray
        }
    } catch {
        Write-Host "  Docker not accessible" -ForegroundColor Yellow
    }

    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "Health check complete!" -ForegroundColor Green
    Write-Host ""
}

# Main execution
if ($ContinuousMonitor) {
    Write-Host "Continuous monitoring mode (Ctrl+C to stop)" -ForegroundColor Yellow
    Write-Host "Refresh interval: $Interval seconds" -ForegroundColor Yellow
    while ($true) {
        Clear-Host
        Test-AllServices
        Start-Sleep -Seconds $Interval
    }
} else {
    Test-AllServices
}
