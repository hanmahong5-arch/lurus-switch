# lurus-switch Architecture

> **Current architecture** — Wails v2 single-app desktop application (Go backend + React frontend).
> The previous microservice architecture docs are archived at `doc/archive/structure-microservices-2024.md`.

## Overview

`lurus-switch` is a Wails v2 desktop application that serves as a unified AI CLI tool manager.
It bundles 7 AI CLI tools, a gateway server manager, relay station routing, and the GY product suite.

## Process Model

```
wails-switch.exe
  ├── Go backend (App struct, bindings_*.go)
  │    ├── internal/config        — Claude/Codex/Gemini/Claw config models + store
  │    ├── internal/installer     — Tool install/detect/update (bun/pip)
  │    ├── internal/proxy         — Legacy proxy settings
  │    ├── internal/relay         — Relay endpoint store + health checks
  │    ├── internal/serverctl     — Embedded lurus-newapi gateway manager
  │    ├── internal/billing       — HTTP client → lurus-api /api/v2/*
  │    ├── internal/updater       — Self-update + npm version checker
  │    ├── internal/mcp           — MCP server presets
  │    ├── internal/snapshot      — Config snapshots + diff
  │    ├── internal/promptlib     — Prompt library
  │    ├── internal/process       — CLI process monitor
  │    ├── internal/analytics     — Local usage tracking
  │    ├── internal/gy            — GY product suite (lucrum/creator/memorus)
  │    └── ...
  └── React frontend (Vite + TypeScript)
       ├── src/pages/             — Full-page views
       ├── src/components/        — Reusable components
       ├── src/stores/            — Zustand state (config/dashboard/billing/relay/gy/gateway)
       └── src/i18n/              — i18n strings (en/zh)
```

## Key Data Flows

| Action | Frontend → Go | Storage |
|--------|--------------|---------|
| Load tool config | `ReadToolConfig(tool)` | `~/.{tool}/config.*` |
| Save tool config | `SaveToolConfig(tool, content)` | `~/.{tool}/config.*` |
| Install tool | `InstallTool(tool)` | bun global / pip |
| Relay endpoints | `GetRelayEndpoints()` | `%APPDATA%/lurus-switch/relay.json` |
| Tool→relay map | `GetToolRelayMapping()` | `%APPDATA%/lurus-switch/relay-mapping.json` |
| Apply relays | `ApplyAllToolRelays()` | Writes apiEndpoint to each tool config |
| GY products | `GetGYProducts()`, `LaunchGYProduct()` | In-memory (builtin) |
| Billing | `BillingGetUserInfo()`, `BillingGetQuotaSummary()` | HTTP → lurus-api |

## Routing (ActiveTool)

| ActiveTool value | Page rendered |
|-----------------|---------------|
| `dashboard` | DashboardPage |
| `claude`/`codex`/`gemini`/`picoclaw`/`nullclaw`/`zeroclaw`/`openclaw` | ToolConfigPage |
| `relay` | RelayPage |
| `gy-products` | GYProductsPage |
| `cli-runner` | CLIRunner component |
| `billing` | BillingPage |
| `gateway` | GatewayPage |
| `gateway-*` | Gateway sub-pages (wrapped with GatewayRequiredGuard) |
| `settings` | SettingsPage |
| `process` | ProcessPage |
| `prompts` | PromptLibraryPage |
| `documents` | DocumentPage |
| `admin` | AdminPage |
