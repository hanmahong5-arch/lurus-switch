# lurus-switch Development Guide

> **Current stack** — Wails v2 (Go 1.25 + React 18 + TypeScript + Vite).
> The previous microservice development guide is archived at `doc/archive/develop-guide-microservices-2024.md`.

## Quick Start

```bash
# Prerequisites
# - Go 1.25+
# - Node.js / Bun
# - Wails CLI: go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Development (hot reload)
wails dev

# Production build
wails build
# Output: build/bin/lurus-switch.exe
```

## Backend (Go)

### Adding a new Wails binding
1. Add method to `App` struct (e.g., in a new `bindings_xxx.go`)
2. Run `wails dev` — it auto-regenerates `frontend/wailsjs/go/main/App.js` and `App.d.ts`
3. If not running `wails dev`, manually update both files AND `frontend/wailsjs/go/models.ts`

### Tests

```bash
go test -v -race ./...                   # All Go tests with race detector
go test -v ./internal/updater/...        # Single package
go test -v ./internal/relay/...
```

### Internal package layout

```
internal/
  config/       — Config domain models (ClaudeConfig etc.) + JSON/TOML store
  installer/    — Detect, install, update, uninstall tools via bun/pip
  relay/        — Relay endpoint CRUD, health checks, cloud fetch
  serverctl/    — Embedded lurus-newapi binary lifecycle (start/stop/config)
  billing/      — Lurus API HTTP client (quota, subscriptions, top-up)
  updater/      — GitHub release self-updater (+ SHA-256 verification)
  gy/           — GY product suite definitions, status checks, launcher
  mcp/          — MCP preset store
  snapshot/     — Config file snapshot + diff
  promptlib/    — Prompt library
  process/      — CLI process monitor (launch/stream output/kill)
  analytics/    — Local event tracking
  envmgr/       — API key management across all tool configs
  docmgr/       — CLAUDE.md context file manager
```

## Frontend (React + TypeScript)

### Dev (Vite only, no Go backend)

```bash
cd frontend
bun install
bun run dev        # Vite dev server on :5173
```

### Tests

```bash
cd frontend
bun run test:run   # Vitest single run
bun run test       # Vitest watch mode
bun run build      # Production build (tsc + vite)
```

### State management

All global state is in `src/stores/` (Zustand):

| Store | Purpose |
|-------|---------|
| `configStore` | Active page/tool, section, highlightField |
| `dashboardStore` | Tool statuses, proxy settings, health |
| `billingStore` | User info, subscriptions, quota |
| `gatewayStore` | Gateway server status, admin token, polling |
| `relayStore` | Relay endpoints + tool mapping |
| `gyStore` | GY product list + availability status |

### Adding a new page

1. Create `src/pages/NewPage.tsx`
2. Add `'new-page'` to `ActiveTool` type in `src/stores/configStore.ts`
3. Add `case 'new-page': return <NewPage />` in `src/App.tsx`
4. Add nav entry in `src/components/Sidebar.tsx`
5. Add i18n key to `src/i18n/en.json` and `src/i18n/zh.json`

## Wails Binding Notes

- Wails auto-generates `frontend/wailsjs/go/main/App.js` and `App.d.ts` when running `wails dev`
- If adding bindings manually (without `wails dev`), also update `frontend/wailsjs/go/models.ts`
  for any new Go types used as return values or parameters
- Go types map to TypeScript classes with `createFrom()` static method in `models.ts`
- Field names: Go `json:"my_field"` → TypeScript camelCase `myField`

## CI/CD

See `.github/workflows/ci.yml` for test pipeline and `.github/workflows/deploy.yml` for release builds.

Release artifacts include `.sha256` checksum sidecars for binary integrity verification.
