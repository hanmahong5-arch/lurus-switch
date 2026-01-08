# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is a monorepo called **Lurus Switch** (Ailurus PaaS), containing multiple projects for AI provider management and multi-device synchronization. The system is evolving from a hybrid monolith to an event-driven microservices architecture.

```
lurus-switch/
├── codeswitch/           # Desktop GUI app (Go + Vue 3 + Wails 3) - main local gateway
├── gemini-cli/           # Gemini CLI toolchain (Node.js + Bun, forked from Google)
├── gateway-service/      # API Gateway microservice (Go + Hertz)
├── provider-service/     # Provider config microservice (Go + Kratos)
├── log-service/          # Log analytics microservice (Go + Kratos + ClickHouse)
├── billing-service/      # Billing & auth microservice (Go + Kratos)
├── subscription-service/ # Subscription management (Go + Kratos)
├── lurus-common/         # Shared libraries (tracing, metrics, NATS)
├── lurus-portal/         # Web portal frontend (Vue 3)
├── new-api/              # LLM unified gateway service (Go)
├── nats/                 # NATS server configuration
└── deploy/               # Deployment configs (Prometheus, Grafana, Alertmanager)
```

## Project-Specific Instructions

Each major project has its own `CLAUDE.md` with detailed guidance:
- **codeswitch**: See `codeswitch/CLAUDE.md` for architecture, commands, and troubleshooting

## Build & Development Commands

### CodeSwitch (Main Desktop App)

```bash
cd codeswitch

# Development (hot reload)
wails3 task dev

# Build for current platform
wails3 task build

# Build production package
wails3 task common:update:build-assets
wails3 task package

# Windows cross-compile from macOS
env ARCH=amd64 wails3 task windows:build

# Run Go tests
go test ./...
go test ./services/providerservice_test.go  # Single file

# Frontend only
cd frontend && npm run dev
```

### Gemini CLI

> **Note**: gemini-cli uses **bun** instead of npm.

```bash
cd gemini-cli

bun install                    # Install dependencies
bun run start                  # CLI development
bun run build                  # Build CLI

# Electron GUI
cd packages/electron
bun run dev                    # Development mode
bun run package:win            # Package for Windows

# Tests
bun run test                   # Unit tests
bun run test:integration:sandbox:none  # Integration tests
```

### Microservices (Kratos-based)

```bash
# Gateway Service (Hertz)
cd gateway-service
make build                     # Build binary
./gateway.exe                  # Run (Windows)
make run                       # Dev run with hot reload

# Provider/Log/Billing Services (Kratos)
cd provider-service  # or log-service, billing-service
make build
make run

# Shared library
cd lurus-common
go test ./...
```

### Docker Development Environment

```bash
# Start all infrastructure (PostgreSQL, Redis, NATS, ClickHouse, etc.)
docker-compose -f docker-compose.dev.yaml up -d

# Observability stack
# - Prometheus: http://localhost:9090
# - Grafana: http://localhost:3000 (admin/admin)
# - Jaeger: http://localhost:16686
# - Alertmanager: http://localhost:9093
```

## Architecture Overview

### Current Hybrid Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Client Layer                                    │
├─────────────────┬─────────────────┬─────────────────┬───────────────────────┤
│   Mobile App    │   GUI Client    │   TUI Client    │    Admin Console      │
│   (Flutter)     │   (Wails 3)     │  (gemini-cli)   │    (Vue 3)            │
│                 │   CodeSwitch    │   claude-code   │    /admin/*           │
│                 │                 │   codex         │                       │
└────────┬────────┴────────┬────────┴────────┬────────┴───────────┬───────────┘
         │                 │                 │                    │
         │     WebSocket   │    NATS/WS      │    NATS/WS         │  HTTP/WS
         └─────────────────┴─────────────────┴────────────────────┘
                                   │
                   ┌───────────────▼───────────────┐
                   │         NATS Server           │
                   │    (Message Bus + JetStream)  │
                   │    Port: 4222 / 8222 (WS)     │
                   └───────────────┬───────────────┘
                                   │
        ┌──────────────────────────┼──────────────────────────┐
        │                          │                          │
        ▼                          ▼                          ▼
┌─────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│   Sync Service  │    │   Gateway Service   │    │   Admin API         │
│   (Go + NATS)   │    │   (Go + Gin)        │    │   (Sync Service)    │
│   :8081         │    │   :18100            │    │   :8081/admin/*     │
└─────────────────┘    └──────────┬──────────┘    └─────────────────────┘
                                  │
                   ┌──────────────▼──────────────┐
                   │          NEW-API            │
                   │      (LLM Unified Gateway)  │
                   │ OpenAI / Claude / Gemini    │
                   └─────────────────────────────┘
```

### Target Microservices Architecture (In Progress)

```
Gateway Service (:18100, Hertz) ──► Provider Service (:18101, Kratos)
         │                                   │
         ├──► Log Service (:18102, Kratos) ──► ClickHouse
         │
         └──► Billing Service (:18103, Kratos) ──► PostgreSQL
                        │
         All services ──┴──► NATS JetStream (events)
```

### Core Data Flow (CodeSwitch Proxy)

```
Claude Code / Codex / Gemini CLI
        │
        ▼ HTTP Request
┌───────────────────┐
│  :18100 Proxy     │ ◄─── Provider Config (JSON)
│  ProviderRelay    │
└────────┬──────────┘
         │
         ▼ Model Matching
┌───────────────────┐
│  Provider Select  │ ◄─── Round-Robin / Priority / Failover
│  + NEW-API Mode   │
└────────┬──────────┘
         │
         ▼ Forward Request
┌───────────────────┐
│  AI Provider API  │
└────────┬──────────┘
         │
         ▼ Response + Logging
┌───────────────────┐
│  SQLite (app.db)  │ ◄─── Write Queue (Async Batch)
│  Request Logs     │
└───────────────────┘
```

## Technology Stack

| Layer | Technology |
|-------|------------|
| Desktop GUI | Wails 3 + Vue 3 + Vite + Tailwind CSS 4 |
| Electron GUI | Electron + React + rough.js (hand-drawn style) |
| API Gateway | Hertz (ByteDance) - native SSE streaming |
| Microservices | Kratos (Bilibili) - Wire DI, service governance |
| Backend Common | Go + Gin |
| CLI | Node.js + Bun (gemini-cli) |
| Message Bus | NATS + JetStream |
| Database | SQLite (local), PostgreSQL (cloud), ClickHouse (logs) |
| Cache | Redis |
| Observability | Prometheus + Grafana + Jaeger |
| Mobile | Flutter |

## Configuration Paths

CodeSwitch stores configuration in `~/.code-switch/`:
```
~/.code-switch/
├── claude-code.json    # Claude Code providers
├── codex.json          # Codex providers
├── mcp.json            # MCP servers
├── app.json            # App settings (NEW-API, NATS config)
├── sync-settings.json  # NATS sync settings
└── app.db              # SQLite database (logs)
```

## Proxy Routes (CodeSwitch Gateway)

| Route | Platform | Format |
|-------|----------|--------|
| `POST /v1/messages` | Claude Code | Anthropic API |
| `POST /responses` | Codex | OpenAI Responses API |
| `POST /v1/chat/completions` | Generic | OpenAI-compatible |
| `POST /chat/completions` | Generic | OpenAI-compatible |
| `POST /v1beta/models/*` | Gemini CLI | Gemini Native (auto-converted) |

## Infrastructure Ports

| Service | Port | Description |
|---------|------|-------------|
| CodeSwitch Gateway | 18100 | Main proxy endpoint |
| Provider Service | 18101 | Provider CRUD API |
| Log Service | 18102 | Log analytics API |
| Billing Service | 18103 | Billing & auth API |
| NEW-API | 3000 | LLM unified gateway |
| Sync Service | 8081 | NATS sync + Admin API |
| NATS | 4222 / 8222 | Message bus (client / WebSocket) |
| PostgreSQL | 5432 | Primary database |
| Redis | 6379 | Cache |
| ClickHouse | 8123 / 9000 | Log OLAP (HTTP / Native) |
| Prometheus | 9090 | Metrics |
| Grafana | 3000 | Dashboards |
| Jaeger | 16686 | Distributed tracing |

## Admin API Routes (Sync Service :8081)

| Route | Description |
|-------|-------------|
| `GET /api/v1/admin/system/status` | System status |
| `GET /api/v1/admin/stats/overview` | Statistics overview |
| `GET /api/v1/admin/users` | User list |
| `GET /api/v1/admin/sessions` | Session list |
| `GET /api/v1/admin/audit-logs` | Audit logs |
| `GET /api/v1/admin/alert-rules` | Alert rules |
| `GET /api/v1/admin/alert-history` | Alert history |

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Gateway Framework | Hertz (ByteDance) | Best SSE streaming performance |
| Microservice Framework | Kratos (Bilibili) | Wire DI + full service governance |
| Log Database | ClickHouse | Optimal OLAP for large-scale log analytics |
| Service Discovery | Consul / K8s | No Etcd, coexists with NATS |
| NEW-API Compatibility | Full | No changes to NEW-API, only CodeSwitch side |
