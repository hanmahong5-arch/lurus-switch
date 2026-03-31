# lurus-switch

统一 AI CLI 工具配置管理桌面应用（Wails v2）。管理 Claude Code / Codex / Gemini CLI / PicoClaw / NullClaw 的配置、安装、代理、MCP、快照、账单等。

**Tech Stack**: Go 1.25 + Wails v2.11 / React 18 + TypeScript + Vite + Tailwind CSS + Zustand + Monaco Editor / Bun (frontend package manager)

## Structure

```
main.go                      # Wails bootstrap (embed frontend/dist, window 1024x768)
app.go                       # App struct — all Wails-bound methods (God Object, S1.2 targets decomposition)
internal/
  config/                    # Domain models: ClaudeConfig, CodexConfig, GeminiConfig, PicoClawConfig, NullClawConfig + Store
  appconfig/                 # App-level UI settings (theme/language/autoUpdate); stored in %APPDATA%/lurus-switch/app-settings.json
  generator/                 # Config file generation: Claude (JSON), Codex (TOML), Gemini (Markdown), PicoClaw/NullClaw (JSON)
  validator/                 # Config validation, returns ValidationResult with field-level errors
  installer/                 # Tool install/detect/update/uninstall via bun/pip; tools: claude, codex, gemini, picoclaw, nullclaw
  packager/                  # Bun packager (standalone exe) + Rust packager (Codex binary download)
  proxy/                     # NewAPI proxy settings; stored in %APPDATA%/lurus-switch/proxy.json
  billing/                   # HTTP client → lurus-api /api/v2/* (user info, quota, plans, subscriptions, top-up)
  updater/                   # Self-update via GitHub releases; npm registry version checker for tools
  mcp/                       # MCP server presets (builtin + user-saved); apply to ~/.claude/settings.json or ~/.gemini/settings.json
  snapshot/                  # Config file snapshots with diff support; stored in %APPDATA%/lurus-switch/snapshots/
  promptlib/                 # Prompt library (builtin + user); export/import JSON
  process/                   # CLI process monitor (list/kill/launch/output)
  toolconfig/                # Read/write real tool config files on disk (see paths below)
  docmgr/                    # Context file manager (CLAUDE.md scan/read/write)
  envmgr/                    # API key listing and update across all tool configs
  analytics/                 # Local usage tracking (tool actions, daily active events)
  downloader/                # Generic file download utility
frontend/src/
  pages/                     # DashboardPage, ToolConfigPage, BillingPage, SettingsPage, AdminPage, DocumentPage, ProcessPage, PromptLibraryPage, ClaudePage, CodexPage, GeminiPage
  stores/                    # configStore, dashboardStore, billingStore, promptStore (Zustand)
  components/                # Reusable UI components (Radix UI primitives + Tailwind)
```

## Tool Config File Paths

| Tool | Config File | Format |
|------|------------|--------|
| claude | `~/.claude/settings.json` | JSON |
| codex | `~/.codex/config.toml` | TOML |
| gemini | `~/.gemini/settings.json` | JSON |
| picoclaw | `~/.picoclaw/config.json` | JSON |
| nullclaw | `~/.nullclaw/config.json` | JSON |

## App Data Paths (Windows)

| File | Path |
|------|------|
| App settings | `%APPDATA%\lurus-switch\app-settings.json` |
| Proxy settings | `%APPDATA%\lurus-switch\proxy.json` |
| Snapshots | `%APPDATA%\lurus-switch\snapshots\` |
| Prompt library | `%APPDATA%\lurus-switch\prompts\` |
| MCP presets | `%APPDATA%\lurus-switch\mcp-presets\` |
| Analytics | `%APPDATA%\lurus-switch\analytics\` |

## Commands

```bash
# Development
wails dev                             # Hot reload dev mode (launches Vite + Go)
wails build                           # Production build → build/bin/lurus-switch.exe

# Backend tests
go test -v ./...                      # All Go tests
go test -v ./internal/config/...      # Config model tests only
go test -v ./internal/proxy/...       # Proxy tests

# Frontend
cd frontend && bun install            # Install dependencies
cd frontend && bun run dev            # Vite dev server only (no Wails)
cd frontend && bun run build          # Production frontend build
cd frontend && bun run test           # Vitest (watch)
cd frontend && bun run test:run       # Vitest (single run)
cd frontend && bun run test:coverage  # Coverage report

# All tests (from lurus root)
./tests/run-tests.ps1
```

## Key Runtime Dependencies (no DB, no NATS)

- **Billing API**: lurus-api `/api/v2/*` — configured via Proxy Settings (APIEndpoint + UserToken)
- **Self-update**: GitHub Releases `lurus-dev/lurus-switch`
- **Tool version checks**: npm registry `https://registry.npmjs.org`
- **Tool installation**: `bun` (npm tools) / `pip` (PicoClaw, NullClaw)

## BMAD

| Resource | Path |
|----------|------|
| PRD | `./_bmad-output/planning-artifacts/prd.md` |
| Epics | `./_bmad-output/planning-artifacts/epics.md` |
| Architecture | `./_bmad-output/planning-artifacts/architecture.md` |
| Gap Analysis | `./_bmad-output/planning-artifacts/bmad-gap-analysis.md` |
| Sprint Status | `./_bmad-output/sprint-status.yaml` |

Current: Sprint 0 (Planning). Sprint 1 starts 2026-03-13 — key items: decompose app.go God Object (S1.2), implement i18n (S1.3), fix Settings page (S1.4).

## Zitadel OIDC Login (2026-03-21)

Auth flow: Zitadel PKCE (port 31416) → encrypted token storage (AES-GCM) → auto gateway provisioning.
Fallback: Manual token entry in Proxy Settings (preserved for advanced users).

| File | Purpose |
|------|---------|
| `internal/auth/session.go` | Encrypted token storage (`%APPDATA%/lurus-switch/auth.enc`) |
| `internal/auth/pkce.go` | OIDC authorization code + PKCE flow |
| `internal/auth/provisioner.go` | Gateway token auto-provisioning |
| `bindings_auth.go` | Wails bindings: Login/Logout/RefreshAuth/GetAuthState |
| `frontend/src/stores/authStore.ts` | Zustand auth state |
| `frontend/src/components/AuthLoginPanel.tsx` | Login UI (3 states: logged-out/logging-in/logged-in) |

Token priority: OIDC session gateway token > manual proxy UserToken
Env: `LURUS_SWITCH_INTERNAL_KEY` (for gateway provisioning)
