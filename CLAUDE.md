# Switch (2c-gui-switch)

桌面 AI CLI 网关 + 渠道分销基础设施。三模式：
- **Personal**：个人 CLI 管家（Claude Code / Codex / Gemini / PicoClaw / NullClaw / OpenClaw 配置 + 代理 + MCP + 快照 + 账单），调 Lurus 自营 Hub。
- **Reseller**：经销商运营台。部署专属 newhub 实例 → 生成激活码 → 导出白标 EndUser 安装包。
- **EndUser**：白标 C 端客户端（Hub URL 锁死，激活码兑换 token，心跳验证）。

Desktop 产品组 (P2)。详细路线图见 `_bmad-output/planning-artifacts/transformation-roadmap-v0.4.md`。

- Delivery: Desktop (Wails v2)，无 K8s
- Repo: `hanmahong5-arch/lurus-switch` (GitHub releases 自更新)
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

- **2b-svc-newhub (Lurus Hub)**: 三模式后端。Personal → Lurus 自营 `hub.lurus.cn`；Reseller → 部署经销商专属实例；EndUser → 白标包内嵌的经销商 hub URL。
  - V2 多租户: `/api/v2/:tenant_slug/{auth/login, user/me, ...}` (Zitadel OIDC)
  - V2 Switch 专用: `/api/v2/switch/{tools/versions, presets}`, `/api/v2/admin/switch/presets`
  - V2 计费: `/api/v2/user/billing/{summary, payment-methods, checkout}`
  - V2 经销商管理: `/api/v2/admin/{tenants, mappings, governance, audit/events}`
  - V1 兼容（Reseller 控制台用）: `/api/{token, channel, redemption, log, data, wallet, openrouter-sync}/*`
  - 激活码兑换: `POST /api/user/topup` (newapi 原生 RedeemCodeV2)
- **2l-svc-platform**: Zitadel OIDC (PKCE, port 31416), 钱包/计费 — newhub 通过 gRPC 转发，Switch 不直连 platform
- **Env**: `LURUS_SWITCH_INTERNAL_KEY` (Personal mode 自营 hub provisioning)
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

---
_BMAD artifacts last review: 2026-05-18 — governance: `lurus/doc/audit/2026-05-18-bmad-output-stale.md`._
