# Switch (2c-gui-switch)

桌面 AI CLI 网关——统一管理 Claude Code / Codex / Gemini CLI / PicoClaw / NullClaw 配置、安装、代理、MCP、快照、账单。Desktop 产品组 (P2)。

- Delivery: Desktop (Wails v2)，无 K8s
- Repo: `lurus-dev/lurus-switch` (GitHub releases 自更新)
- Local SQLite: `%APPDATA%\lurus-switch\`

## Tech Stack

| Layer | Stack |
|-------|-------|
| Backend | Go 1.25 + Wails v2.11 |
| Frontend | React 18 + TypeScript + Vite + Tailwind + Zustand + Monaco + Bun |

## Directory

```
main.go · app.go          # Wails bootstrap + God Object (S1.2 目标拆分)
internal/                 # config, appconfig, generator, validator, installer, packager,
                          # proxy, billing, updater, mcp, snapshot, promptlib, process,
                          # toolconfig, docmgr, envmgr, analytics, downloader, auth
frontend/src/             # pages/, stores/, components/
```

## Commands

```bash
wails dev                              # hot reload
wails build                            # → build/bin/lurus-switch.exe

go test -v ./...                       # backend
cd frontend && bun install && bun run test:run
cd frontend && bun run build

./tests/run-tests.ps1                  # all
```

## Paths

| Purpose | Path |
|---------|------|
| Tool configs | `~/.claude/settings.json`, `~/.codex/config.toml`, `~/.gemini/settings.json`, `~/.picoclaw/config.json`, `~/.nullclaw/config.json` |
| App data | `%APPDATA%\lurus-switch\{app-settings.json,proxy.json,snapshots,prompts,mcp-presets,analytics,auth.enc}` |

## Cross-service Dependencies

- **2b-svc-api Hub (`lurus-api /api/v2/*`)**: user info, quota, plans, subscriptions, top-up — configured via Proxy Settings (APIEndpoint + UserToken)
- **Zitadel OIDC (PKCE, port 31416)**: encrypted token (AES-GCM) at `auth.enc` → auto gateway token provisioning
- **Env**: `LURUS_SWITCH_INTERNAL_KEY` (gateway provisioning)
- **npm registry**: tool version checks · **bun/pip**: tool install

## Gotchas

- Token priority: OIDC session gateway token > manual proxy UserToken
- `app.go` 是 God Object，新增 Wails 绑定时注意拆分（S1.2 计划）
- 无 DB / NATS，所有状态走本地 JSON 或远端 API

## BMAD

| Resource | Path |
|----------|------|
| PRD | `./_bmad-output/planning-artifacts/prd.md` |
| Epics | `./_bmad-output/planning-artifacts/epics.md` |
| Architecture | `./_bmad-output/planning-artifacts/architecture.md` |
| Gap Analysis | `./_bmad-output/planning-artifacts/bmad-gap-analysis.md` |
| Sprint Status | `./_bmad-output/sprint-status.yaml` |
