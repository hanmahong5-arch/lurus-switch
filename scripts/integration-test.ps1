# Lurus Switch Integration Test Script
# Tests all microservices and observability stack
# Usage: .\scripts\integration-test.ps1

$ErrorActionPreference = "Continue"

Write-Host ""
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host "      Lurus Switch Integration Test Suite" -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Started at: $(Get-Date)"
Write-Host ""

# Configuration - Services
$GATEWAY_URL = "http://localhost:18100"
$PROVIDER_URL = "http://localhost:18101"
$LOG_URL = "http://localhost:18102"
$BILLING_URL = "http://localhost:18103"

# Configuration - Infrastructure
$NATS_URL = "http://localhost:8222"
$CLICKHOUSE_URL = "http://localhost:8123"
$CONSUL_URL = "http://localhost:8500"

# Configuration - Observability
$PROMETHEUS_URL = "http://localhost:9090"
$GRAFANA_URL = "http://localhost:3000"
$JAEGER_URL = "http://localhost:16686"
$ALERTMANAGER_URL = "http://localhost:9093"

# Alias for backward compatibility
$PROVIDER_SERVICE_URL = $PROVIDER_URL
$BILLING_SERVICE_URL = $BILLING_URL
$GATEWAY_SERVICE_URL = $GATEWAY_URL
$LOG_SERVICE_URL = $LOG_URL

# Counters
$script:PASSED = 0
$script:FAILED = 0
$script:SKIPPED = 0

function Write-Pass {
    param([string]$Message = "")
    Write-Host " OK" -ForegroundColor Green -NoNewline
    if ($Message) { Write-Host " $Message" -ForegroundColor Gray } else { Write-Host "" }
    $script:PASSED++
}

function Write-Fail {
    param([string]$Message = "")
    Write-Host " FAILED" -ForegroundColor Red -NoNewline
    if ($Message) { Write-Host " $Message" -ForegroundColor Gray } else { Write-Host "" }
    $script:FAILED++
}

function Write-Skip {
    param([string]$Message = "")
    Write-Host " SKIPPED" -ForegroundColor Yellow -NoNewline
    if ($Message) { Write-Host " $Message" -ForegroundColor Gray } else { Write-Host "" }
    $script:SKIPPED++
}

function Test-ServiceHealth {
    param(
        [string]$ServiceName,
        [string]$Url
    )

    Write-Host "Testing $ServiceName health..." -NoNewline
    try {
        $response = Invoke-RestMethod -Uri "$Url/health" -Method Get -TimeoutSec 5
        if ($response.status -eq "healthy") {
            Write-Host " OK" -ForegroundColor Green
            return $true
        } else {
            Write-Host " UNHEALTHY" -ForegroundColor Red
            return $false
        }
    } catch {
        Write-Host " FAILED (not running)" -ForegroundColor Red
        return $false
    }
}

function Test-ProviderAPI {
    Write-Host ""
    Write-Host "=== Provider Service API Tests ===" -ForegroundColor Yellow

    # Test: List providers
    Write-Host "  GET /api/v1/providers?platform=claude..." -NoNewline
    try {
        $response = Invoke-RestMethod -Uri "$PROVIDER_SERVICE_URL/api/v1/providers?platform=claude" -Method Get
        Write-Host " OK (found $($response.providers.Count) providers)" -ForegroundColor Green
    } catch {
        Write-Host " FAILED: $_" -ForegroundColor Red
    }

    # Test: Create provider
    Write-Host "  POST /api/v1/providers (create)..." -NoNewline
    try {
        $body = @{
            name = "Test Provider"
            platform = "claude"
            api_url = "https://api.test.com"
            api_key = "test-key"
            enabled = $true
            supported_models = @{"claude-*" = $true}
        } | ConvertTo-Json

        $response = Invoke-RestMethod -Uri "$PROVIDER_SERVICE_URL/api/v1/providers" -Method Post -Body $body -ContentType "application/json"
        $createdId = $response.provider.id
        Write-Host " OK (id=$createdId)" -ForegroundColor Green

        # Test: Delete provider
        Write-Host "  DELETE /api/v1/providers/$createdId..." -NoNewline
        Invoke-RestMethod -Uri "$PROVIDER_SERVICE_URL/api/v1/providers/$createdId" -Method Delete | Out-Null
        Write-Host " OK" -ForegroundColor Green
    } catch {
        Write-Host " FAILED: $_" -ForegroundColor Red
    }
}

function Test-BillingAPI {
    Write-Host ""
    Write-Host "=== Billing Service API Tests ===" -ForegroundColor Yellow

    $testUser = "test-user-$(Get-Random)"

    # Test: Check balance (new user)
    Write-Host "  GET /api/v1/billing/check/$testUser..." -NoNewline
    try {
        $response = Invoke-RestMethod -Uri "$BILLING_SERVICE_URL/api/v1/billing/check/$testUser" -Method Get
        if ($response.allowed) {
            Write-Host " OK (allowed=$($response.allowed))" -ForegroundColor Green
        } else {
            Write-Host " OK (not allowed: $($response.message))" -ForegroundColor Yellow
        }
    } catch {
        Write-Host " FAILED: $_" -ForegroundColor Red
    }

    # Test: Get quota
    Write-Host "  GET /api/v1/billing/quota/$testUser..." -NoNewline
    try {
        $response = Invoke-RestMethod -Uri "$BILLING_SERVICE_URL/api/v1/billing/quota/$testUser" -Method Get
        Write-Host " OK (quota_limit=$($response.quota_limit))" -ForegroundColor Green
    } catch {
        Write-Host " FAILED: $_" -ForegroundColor Red
    }

    # Test: Record usage
    Write-Host "  POST /api/v1/billing/usage..." -NoNewline
    try {
        $body = @{
            user_id = $testUser
            platform = "claude"
            model = "claude-3-opus"
            input_tokens = 1000
            output_tokens = 500
            total_cost = 0.05
        } | ConvertTo-Json

        $response = Invoke-RestMethod -Uri "$BILLING_SERVICE_URL/api/v1/billing/usage" -Method Post -Body $body -ContentType "application/json"
        Write-Host " OK" -ForegroundColor Green
    } catch {
        Write-Host " FAILED: $_" -ForegroundColor Red
    }
}

function Test-GatewayAPI {
    Write-Host ""
    Write-Host "=== Gateway Service API Tests ===" -ForegroundColor Yellow

    # Test: Health check
    Write-Host "  GET /health..." -NoNewline
    try {
        $response = Invoke-RestMethod -Uri "$GATEWAY_SERVICE_URL/health" -Method Get -TimeoutSec 5
        Write-Pass
    } catch {
        Write-Fail
    }

    # Test: Ready check
    Write-Host "  GET /ready..." -NoNewline
    try {
        $response = Invoke-RestMethod -Uri "$GATEWAY_SERVICE_URL/ready" -Method Get -TimeoutSec 5
        Write-Pass
    } catch {
        Write-Fail
    }

    # Test: Metrics endpoint
    Write-Host "  GET /metrics..." -NoNewline
    try {
        $response = Invoke-WebRequest -Uri "$GATEWAY_SERVICE_URL/metrics" -Method Get -TimeoutSec 5
        if ($response.Content -match "go_") {
            Write-Pass "(Prometheus format)"
        } else {
            Write-Pass
        }
    } catch {
        Write-Fail
    }
}

function Test-Infrastructure {
    Write-Host ""
    Write-Host "=== Infrastructure Tests ===" -ForegroundColor Yellow

    # PostgreSQL
    Write-Host "  PostgreSQL..." -NoNewline
    try {
        $result = docker exec lurus-postgres pg_isready -U lurus 2>&1
        if ($LASTEXITCODE -eq 0) { Write-Pass } else { Write-Fail "(not ready)" }
    } catch {
        Write-Fail "(container not running)"
    }

    # Redis
    Write-Host "  Redis..." -NoNewline
    try {
        $result = docker exec lurus-redis redis-cli ping 2>&1
        if ($result -match "PONG") { Write-Pass } else { Write-Fail }
    } catch {
        Write-Fail "(container not running)"
    }

    # NATS
    Write-Host "  NATS Server..." -NoNewline
    try {
        $response = Invoke-RestMethod -Uri "$NATS_URL/varz" -Method Get -TimeoutSec 5
        Write-Pass
    } catch {
        Write-Fail
    }

    # ClickHouse
    Write-Host "  ClickHouse..." -NoNewline
    try {
        $response = Invoke-WebRequest -Uri "$CLICKHOUSE_URL/ping" -Method Get -TimeoutSec 5
        Write-Pass
    } catch {
        Write-Fail
    }

    # Consul
    Write-Host "  Consul..." -NoNewline
    try {
        $response = Invoke-RestMethod -Uri "$CONSUL_URL/v1/status/leader" -Method Get -TimeoutSec 5
        Write-Pass
    } catch {
        Write-Skip
    }
}

function Test-Observability {
    Write-Host ""
    Write-Host "=== Observability Stack Tests ===" -ForegroundColor Yellow

    # Prometheus
    Write-Host "  Prometheus..." -NoNewline
    try {
        $response = Invoke-WebRequest -Uri "$PROMETHEUS_URL/-/healthy" -Method Get -TimeoutSec 5
        Write-Pass
    } catch {
        Write-Fail
    }

    # Prometheus targets
    Write-Host "  Prometheus targets..." -NoNewline
    try {
        $response = Invoke-RestMethod -Uri "$PROMETHEUS_URL/api/v1/targets" -Method Get -TimeoutSec 5
        if ($response.data.activeTargets.Count -gt 0) { Write-Pass } else { Write-Skip }
    } catch {
        Write-Skip
    }

    # Grafana
    Write-Host "  Grafana..." -NoNewline
    try {
        $response = Invoke-RestMethod -Uri "$GRAFANA_URL/api/health" -Method Get -TimeoutSec 5
        if ($response.database -eq "ok") { Write-Pass } else { Write-Fail }
    } catch {
        Write-Fail
    }

    # Jaeger
    Write-Host "  Jaeger..." -NoNewline
    try {
        $response = Invoke-WebRequest -Uri "$JAEGER_URL" -Method Get -TimeoutSec 5
        Write-Pass
    } catch {
        Write-Fail
    }

    # Alertmanager
    Write-Host "  Alertmanager..." -NoNewline
    try {
        $response = Invoke-WebRequest -Uri "$ALERTMANAGER_URL/-/healthy" -Method Get -TimeoutSec 5
        Write-Pass
    } catch {
        Write-Fail
    }
}

# Main execution
Write-Host "1. Checking service health..." -ForegroundColor Cyan
Write-Host ""

$gatewayOk = Test-ServiceHealth "Gateway Service (:18100)" $GATEWAY_SERVICE_URL
$providerOk = Test-ServiceHealth "Provider Service (:18101)" $PROVIDER_SERVICE_URL
$logOk = Test-ServiceHealth "Log Service (:18102)" $LOG_SERVICE_URL
$billingOk = Test-ServiceHealth "Billing Service (:18103)" $BILLING_SERVICE_URL

Write-Host ""
Write-Host "2. Running API tests..." -ForegroundColor Cyan

if ($providerOk) {
    Test-ProviderAPI
} else {
    Write-Host "`nSkipping Provider API tests (service not running)" -ForegroundColor Yellow
}

if ($billingOk) {
    Test-BillingAPI
} else {
    Write-Host "`nSkipping Billing API tests (service not running)" -ForegroundColor Yellow
}

if ($gatewayOk) {
    Test-GatewayAPI
} else {
    Write-Host "`nSkipping Gateway API tests (service not running)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "3. Testing Infrastructure..." -ForegroundColor Cyan
Test-Infrastructure

Write-Host ""
Write-Host "4. Testing Observability Stack..." -ForegroundColor Cyan
Test-Observability

Write-Host ""
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host "  Integration Test Complete" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Completed at: $(Get-Date)"
Write-Host ""

# Summary
$runningCount = @($gatewayOk, $providerOk, $logOk, $billingOk) | Where-Object { $_ } | Measure-Object | Select-Object -ExpandProperty Count

Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Blue
Write-Host "  Passed:  $($script:PASSED)" -ForegroundColor Green
Write-Host "  Failed:  $($script:FAILED)" -ForegroundColor Red
Write-Host "  Skipped: $($script:SKIPPED)" -ForegroundColor Yellow
Write-Host ""
Write-Host "Microservices: $runningCount/4 running" -ForegroundColor $(if ($runningCount -eq 4) { "Green" } else { "Yellow" })

if ($runningCount -lt 4) {
    Write-Host ""
    Write-Host "To start all services, run:" -ForegroundColor Cyan
    Write-Host "  docker-compose -f docker-compose.dev.yaml up -d" -ForegroundColor White
}

if ($script:FAILED -gt 0) {
    Write-Host ""
    Write-Host "Some tests failed. Check logs above for details." -ForegroundColor Red
    exit 1
}
