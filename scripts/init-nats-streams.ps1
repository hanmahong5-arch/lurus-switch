# NATS JetStream Initialization Script for CodeSwitch
# This script initializes NATS streams for message synchronization

param(
    [string]$NatsUrl = "nats://localhost:4222",
    [int]$MaxRetries = 30
)

$ErrorActionPreference = "Stop"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  NATS JetStream Initialization" -ForegroundColor Cyan
Write-Host "  URL: $NatsUrl" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Wait for NATS to be ready
Write-Host "Waiting for NATS to be ready..." -ForegroundColor Yellow
$ready = $false
for ($i = 1; $i -le $MaxRetries; $i++) {
    try {
        $null = nats server ping -s $NatsUrl 2>&1
        Write-Host "NATS is ready!" -ForegroundColor Green
        $ready = $true
        break
    } catch {
        Write-Host "  Attempt $i/$MaxRetries..." -ForegroundColor Gray
        Start-Sleep -Seconds 1
    }
}

if (-not $ready) {
    Write-Host "ERROR: NATS not available after $MaxRetries attempts" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Define streams to create
$streams = @(
    @{
        Name = "CHAT_MESSAGES"
        Subjects = "chat.*.*.msg"
        Storage = "file"
        MaxBytes = "10GB"
        Retention = "limits"
        Replicas = 1
        Discard = "old"
    },
    @{
        Name = "SESSION_STATUS"
        Subjects = "chat.*.*.status,chat.*.*.typing"
        Storage = "memory"
        MaxAge = "1d"
        Retention = "limits"
        Replicas = 1
        Discard = "old"
    },
    @{
        Name = "USER_EVENTS"
        Subjects = "user.*.auth,user.*.presence,user.*.notification,user.*.quota"
        Storage = "file"
        MaxAge = "7d"
        Retention = "limits"
        Replicas = 1
        Discard = "old"
    },
    @{
        Name = "LLM_REQUESTS"
        Subjects = "llm.request.*,llm.response.*"
        Storage = "memory"
        MaxAge = "1h"
        Retention = "limits"
        Replicas = 1
        Discard = "old"
    },
    @{
        Name = "AUDIT_LOG"
        Subjects = "admin.audit"
        Storage = "file"
        MaxAge = "365d"
        Retention = "limits"
        Replicas = 1
        Discard = "old"
    },
    @{
        Name = "SYSTEM_BROADCAST"
        Subjects = "admin.broadcast,admin.metrics"
        Storage = "memory"
        MaxAge = "1d"
        Retention = "limits"
        Replicas = 1
        Discard = "old"
    }
)

# Create each stream
$created = 0
$existed = 0
$failed = 0

foreach ($stream in $streams) {
    Write-Host "Creating stream: $($stream.Name)..." -ForegroundColor Cyan

    # Build arguments
    $args = @(
        "stream", "add", $stream.Name,
        "--subjects", $stream.Subjects,
        "--retention", $stream.Retention,
        "--storage", $stream.Storage,
        "--replicas", $stream.Replicas,
        "--discard", $stream.Discard,
        "-s", $NatsUrl,
        "--defaults"
    )

    # Add optional parameters
    if ($stream.ContainsKey("MaxBytes")) {
        $args += "--max-bytes"
        $args += $stream.MaxBytes
    }

    if ($stream.ContainsKey("MaxAge")) {
        $args += "--max-age"
        $args += $stream.MaxAge
    }

    # Execute NATS CLI command
    try {
        $output = & nats @args 2>&1
        $outputStr = $output | Out-String

        if ($outputStr -match "already exists") {
            Write-Host "  Stream already exists" -ForegroundColor Yellow
            $existed++
        } else {
            Write-Host "  Created successfully" -ForegroundColor Green
            $created++
        }
    } catch {
        Write-Host "  Failed to create: $_" -ForegroundColor Red
        $failed++
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Created: $created" -ForegroundColor Green
Write-Host "  Already Existed: $existed" -ForegroundColor Yellow
Write-Host "  Failed: $failed" -ForegroundColor $(if ($failed -gt 0) { "Red" } else { "Gray" })
Write-Host ""

# List all streams
Write-Host "Current JetStream Streams:" -ForegroundColor Cyan
try {
    nats stream list -s $NatsUrl
} catch {
    Write-Host "  Failed to list streams: $_" -ForegroundColor Red
}

Write-Host ""

if ($failed -gt 0) {
    Write-Host "Initialization completed with errors" -ForegroundColor Red
    exit 1
} else {
    Write-Host "Initialization completed successfully!" -ForegroundColor Green
    exit 0
}
