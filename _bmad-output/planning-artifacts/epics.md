# Lurus Switch - Epics & Sprint Planning

**Version**: 3.0
**Date**: 2026-05-01
**Aligned PRD**: Switch PRD v0.1 (B2B2C 渠道分销基础设施)
**Aligned Roadmap**: `transformation-roadmap-v0.4.md`
**Aligned ADR**: ADR-020 (channel-distribution pivot)
**Sprint Duration**: 2 weeks
**Supersedes**: epics v2.0 龙虾管理员路线（E4 部分入库后冻结，E5-E10 冻结到 Phase A-C 完成）

---

## Overview / 概览

Sprint 1-3 (M1 MVP: Foundation + Onboarding + Config Editor) 已于 2026-02-28 完成。
**Sprint 4 (E4 Agent Foundation) 部分入库后转向 Pivot**：因 PRD v0.1 把 Switch 重定位为渠道分销基础设施，Sprint 4 改为 Sprint 4b-4e 推进 E12-E15。Agent Fleet (E4-E10) 工作产出物保留 (`internal/agent/`、`AgentsPage.tsx`)，Personal 模式可见，Phase A-C 完成后回归。

### Milestone-to-Phase Mapping

| Milestone | Phase | Epics | Sprints | 目标 | 状态 |
|-----------|-------|-------|---------|------|------|
| M1 (已完成) | — | E1-E3 | S1-S3 | 基础清理 + 入门 + 配置编辑器 | ✅ |
| M2-old: Agent Fleet | — | E4-E10 | S4-S10 | （旧路线，冻结） | 🟡 Hold |
| **M5: Channel Pivot Foundation** | A | **E12** AppMode + Hub Client | S4b | 三模式分离 + newhub V1+V2 SDK | 📋 |
| **M6: Reseller Mode** | B | **E13-E14** Hub Integration + Reseller | S4c-4d | 经销商部署 + 控制台对接 + 激活码批次 | 📋 |
| **M7: EndUser + White Label** | C | **E15** EndUser Mode | S4e | 白标打包 + 激活码 + 心跳 | 📋 |
| M2-resume: Agent Fleet | — | E4-E10 | S5+ (after C) | Agent 管理（个人模式高级特性） | 🔮 |
| M4: Ecosystem (远期) | P6 | E11 | S11+ | 编排 + 多机 + 社区 | 🔮 |

### Epic Dependency Graph

```
[已完成] E1 Foundation → E2 Onboarding → E3 Config Editor
                                               │
    ┌──────────────────────────────────────────┘
    │
    ▼
E4 Agent Foundation ──→ E5 Agent Lifecycle ──→ E6 Agent Dashboard
    │                        │                       │
    ▼                        ▼                       ▼
E7 Resource Mgmt ←──── E5 (budget needs lifecycle) E8 Context & Knowledge
    │                                                │
    ▼                                                ▼
E9 Monitoring ←─────────────────────────────── E10 Templates
    │
    ▼
E11 Orchestration (远期)
```

---

## 已完成 Sprints (M1 — 保留，不修改)

### Sprint 1: Foundation Cleanup (E1) ✅
- S1.1: Remove dead code — **done** (3pt)
- S1.2: Decompose app.go God Object — **done** (8pt)
- S1.3: Implement i18n — **done** (5pt)
- S1.4: Fix Settings page — **done** (5pt)
- S1.5: Add React error boundaries — **done** (2pt)

### Sprint 2: Onboarding & Dashboard (E2) ✅
- S2.1: First-time setup wizard — **done** (5pt)
- S2.2: Dashboard redesign with ToolCard — **done** (5pt)
- S2.3: Quota widget — **done** (3pt)
- S2.4: Tool health indicators — **done** (3pt)
- S2.5: Proxy auto-detection — **done** (5pt)

### Sprint 3: Visual Config Editor V2 (E3) ✅
- S3.1-S3.3: Form editors (Claude/Codex/Gemini) — **done** (11pt)
- S3.4: Presets — **done** (5pt)
- S3.5: Validation — **done** (3pt)
- S3.6: Snapshots — **done** (2pt)

**已完成总计: 65 points**

---

---

## ⚡ Channel Distribution Pivot (新增 Epics: E12-E15)

> 详见 `transformation-roadmap-v0.4.md` § 3 + `decisions/ADR-020-channel-distribution-pivot.md`。
>
> **优先级高于 E4-E10**。E4-E10 标 Hold，已入库代码保留为 Personal 模式特性，路由守卫隐藏。

---

## Phase A: AppMode 与 Hub Client 基础 (Sprint 4b, 2 weeks)

### Epic 12: AppMode Tri-State / 三模式形态分离

**Goal**: 建立 Personal / Reseller / EndUser 三模式分离机制，路由守卫到位，为后续 Reseller / EndUser 模式奠基。

#### S-Xa.1: AppMode 数据模型 + 持久化 + Wails Bindings

**As a** Switch user,
**I want** the app to remember which mode I selected,
**So that** I don't have to re-pick on every launch.

**Acceptance Criteria**:
- [ ] `internal/appconfig/mode.go`: `AppMode` 枚举（`personal` / `reseller` / `enduser`）
- [ ] `app-settings.json` 增加 `mode: <string>` 字段，默认 `personal`
- [ ] EndUser 模式追加 `lockedHubURL: <string>` 字段（白标包写入后不可改）
- [ ] Wails bindings: `GetAppMode() AppMode`, `SetAppMode(mode AppMode) error`, `IsModeLocked() bool`
- [ ] 模式切换前置校验：EndUser 锁定模式不可切回 Personal/Reseller（除非用户手动删除 settings 文件）
- [ ] 单元测试覆盖：默认值、合法切换、非法切换、锁定状态
- [ ] `go test ./internal/appconfig/... → PASS`

**Points**: 5

#### S-Xa.2: 首次启动模式选择向导

**As a** new user,
**I want** to choose my mode on first launch,
**So that** I see the right UI for my use case.

**Acceptance Criteria**:
- [ ] `frontend/src/pages/AppModeSelectPage.tsx`: 三选一卡片（Personal/Reseller/EndUser）+ 描述 + 推荐场景
- [ ] 路由：`mode === ""` (未选) 强制进入 select 页
- [ ] EndUser 选项检测白标包 marker（`whitelabel.json` sidecar 存在）→ 跳过选择直接进入 EndUser 流程
- [ ] 选完后写入 `app-settings.json`，跳转到对应模式主页
- [ ] i18n 完整（zh + en）
- [ ] Personal 默认推荐（已有 v0.1.0 用户升级体验：默认进 Personal 模式）

**Points**: 3

#### S-Xa.3: 路由守卫 + 11 GatewayXxx 页面限 Reseller

**As a** Personal/EndUser mode user,
**I want** to NOT see Reseller-only management pages,
**So that** my UI is clean and focused.

**Acceptance Criteria**:
- [ ] `frontend/src/router/guards.ts`: `requireMode(mode)` HOC
- [ ] 11 个 GatewayXxx page 套 `requireMode('reseller')`
- [ ] PromoterHubPage 套 `requireMode('reseller')`
- [ ] AgentsPage 套 `requireMode('personal')`（暂不开放给 Reseller/EndUser）
- [ ] EndUser 模式仅显示：HomePage（简化版）、AccountPage（余额）、ToolConfigPage、SettingsPage（仅一个 tab：基础）
- [ ] Sidebar 按模式动态裁剪
- [ ] 直接访问被守卫页面 → 重定向到当前模式主页 + toast 提示

**Points**: 3

#### S-Xa.4: Sidebar 模式徽章 + 设置页模式切换

**As a** user,
**I want** to see my current mode and switch it without restarting,
**So that** I can experiment with different modes.

**Acceptance Criteria**:
- [ ] Sidebar 顶部显示模式徽章（彩色标签：蓝色 Personal / 紫色 Reseller / 绿色 EndUser）
- [ ] SettingsPage 增加"应用模式"卡片（EndUser 锁定时禁用切换 + 显示原因）
- [ ] 切换模式触发 SetAppMode → 路由器立即切换 + toast 通知
- [ ] EndUser 模式徽章追加经销商品牌名（从 whitelabel.json 读）

**Points**: 2

#### S-Xa.5: newhub V1+V2 Client SDK

**As a** Switch developer,
**I want** a typed Go client for newhub APIs,
**So that** all 11 GatewayXxx pages can call the Hub uniformly.

**Acceptance Criteria**:
- [ ] `internal/hub/admin/client.go`: 接口 + 实现，配置 `BaseURL` + `AdminToken`
- [ ] V1 endpoints 实现：channel CRUD（28 endpoints）, token CRUD, redemption CRUD, log query, user/wallet
- [ ] V2 endpoints 实现：tenants CRUD, governance, audit, billing summary, switch presets
- [ ] 错误处理：HTTP 401 → 弹回 Reseller 设置页提示重配 token；500/网络错 → 重试 3 次指数退避
- [ ] Token 来源：`app-settings.json` `reseller.adminToken` 字段（向导阶段保存）
- [ ] 单元测试用 `httptest.NewServer` mock 全部 endpoint
- [ ] 集成测试：起 newhub docker-compose → 客户端真实调用 → 校验 happy path
- [ ] Wails bindings: 暴露所有 admin client 方法给前端

**Points**: 8

**Sprint 4b 总计**: 21 points

---

## Phase B: Reseller 部署与控制台 (Sprint 4c-4d, 4 weeks)

### Epic 13: Hub Integration / Hub 集成

**Goal**: 11 个 GatewayXxx page 全部接通 newhub 后端，Reseller 能通过控制台管理 Hub。

#### S-Xb.4: GatewayChannelPage 对接 newhub V1 channel API

**Acceptance Criteria**:
- [ ] List/Search/Create/Edit/Delete channel
- [ ] Channel test (single + all)
- [ ] Tag 操作（disable/enable tag, edit tag, batch set tag）
- [ ] Channel key 查看（root only）
- [ ] Multi-key manage
- [ ] Copy channel
- [ ] OpenRouter sync jobs（Phase 2 可延后到 S-Xb.4b）
- [ ] 错误状态友好提示（中文）

**Points**: 5

#### S-Xb.5: GatewayTokenPage + 批量操作

**Acceptance Criteria**:
- [ ] List/Search/Create/Edit/Delete token
- [ ] Batch delete
- [ ] Token usage 查询
- [ ] Quota / 模型白名单 / IP 白名单 配置 UI

**Points**: 3

#### S-Xb.6: GatewayRedemptionPage + CSV 导出

**Acceptance Criteria**:
- [ ] List/Search/Create redemption code
- [ ] 批量生成（指定数量 + quota + 过期时间）
- [ ] CSV 导出（code 字符串 + quota + 备注）
- [ ] Delete invalid/single
- [ ] UI 显示已使用 / 未使用状态

**Points**: 5

#### S-Xb.7: GatewayLogPage + Meilisearch 检索

**Acceptance Criteria**:
- [ ] 日志列表（时间倒序）+ 分页
- [ ] 过滤：日期范围、模型、状态、用户、token
- [ ] 关键词全文检索（newhub Meilisearch）
- [ ] 单条详情 modal（请求/响应 JSON）
- [ ] 导出选中条目为 CSV

**Points**: 5

#### S-Xb.8: GatewayDashboardPage 对接 governance + audit

**Acceptance Criteria**:
- [ ] 系统统计：QPS、平均延迟、错误率、在线 channel 数
- [ ] Governance：channel 分布、fingerprint stats、latency stats、efficiency stats
- [ ] Audit log: 最近 100 条 admin 操作
- [ ] 实时刷新（10s polling 或 SSE）

**Points**: 5

**Epic 13 小计**: 23 points

---

### Epic 14: Reseller Mode / 经销商部署 + 运营

**Goal**: 经销商从零部署 Hub 到日常运营完整流程。

#### S-Xb.1: ResellerSetupWizard 框架

**Acceptance Criteria**:
- [ ] 多步向导：1) 选云厂商 2) 输 API Key 3) 选实例规格 4) 配域名（可选） 5) 拉起 Hub 6) 拿 admin token
- [ ] 步骤进度可视化
- [ ] 失败回滚（已部署资源能清理）
- [ ] 完成后写 `app-settings.json` `reseller.{cloudProvider, hubURL, adminToken, tenantSlug}`

**Points**: 5

#### S-Xb.2: Sealos Deploy Adapter

**Acceptance Criteria**:
- [ ] `internal/hub/deploy/sealos.go`：Sealos OpenAPI client
- [ ] 创建 namespace + 应用 K8s manifest（基于 `2b-svc-newhub/deploy/k8s/`）
- [ ] 等待 pod ready + readiness probe
- [ ] 拿 ingress URL + 自动配 TLS
- [ ] 失败原因映射到中文错误信息（quota 不足、API key 错、镜像拉取失败等）

**Points**: 8

#### S-Xb.3: Aliyun ECS Deploy Adapter

**Acceptance Criteria**:
- [ ] `internal/hub/deploy/aliyun.go`：阿里云 SDK client
- [ ] 创建 ECS 实例（默认 ecs.c6.large 4G + 系统盘 40G）
- [ ] SSH 进入 + 安装 Docker + 拉 newhub docker-compose
- [ ] 等待健康检查通过
- [ ] 配安全组（仅开 80/443）+ 域名 A 记录提示（用户自行操作 DNS）

**Points**: 8

**Epic 14 小计**: 21 points (deploy adapters) + 部分由 E13 承接 (控制台对接)

**Sprint 4c-4d 总计**: 44 points

---

## Phase C: EndUser + 白标 + 心跳 (Sprint 4e, 2 weeks)

### Epic 15: EndUser Mode / 终端用户激活与白标

**Goal**: 经销商能导出白标 EndUser 安装包，C 端用户输码即用。

#### S-Xc.1: 白标打包器

**As a** reseller,
**I want** to export EndUser .exe with my logo and Hub URL embedded,
**So that** my customers see my brand.

**Acceptance Criteria**:
- [ ] `internal/whitelabel/packer.go`：
  - 读取 base Switch .exe (从 latest GitHub release 或本地 build)
  - 替换 icon resource (Windows: rcedit-style PE editing)
  - 写入 `whitelabel.json` sidecar（hub_url, brand_name, primary_color, logo_data, hmac_signature）
  - 输出 `<brand>-Switch-windows-amd64.exe` + sha256 验证
- [ ] HMAC 签名密钥来自 newhub /api/v2/admin/whitelabel/hmac-key (新增 endpoint TBD)
- [ ] EndUser 模式启动时校验 sidecar HMAC，校验失败拒绝启动
- [ ] 单元测试：打包 → 反向解析 → 校验 hub_url 正确

**Points**: 8

#### S-Xc.2: PackagerPage UI

**Acceptance Criteria**:
- [ ] 上传 logo（PNG, 256x256）
- [ ] 选主题色（color picker）+ 输品牌名
- [ ] 关联激活码批次（可选）
- [ ] "导出 .exe" 按钮 → 调用 whitelabel packer → 下载文件
- [ ] 历史记录：保留最近 10 个 white_label_profiles（SQLite）

**Points**: 5

#### S-Xc.3: EndUserActivationPage + 激活码兑换

**Acceptance Criteria**:
- [ ] 启动检测 EndUser 模式 + token 未持久化 → 显示激活码输入页
- [ ] 输入框校验：长度、字符集
- [ ] POST 到 `<lockedHubURL>/api/user/topup` (newapi RedeemCodeV2)
- [ ] 成功 → 写 token 到 `auth.enc` + 跳转 EndUserMainPage
- [ ] 失败：码已用、码错误、码过期 → 中文友好提示

**Points**: 5

#### S-Xc.4: 设备指纹 + Token 持久化

**Acceptance Criteria**:
- [ ] `internal/redemption/fingerprint.go`：CPU ID + MAC + OS + 硬盘 SN → SHA256 → 16 字符指纹
- [ ] 兑换码请求带 `X-Device-Fingerprint` header（newhub 端记录）
- [ ] Token 加密写入 `auth.enc` (AES-GCM, key 派生自 fingerprint)
- [ ] 重启 → 解密 → 自动续 session（指纹不变前提下）
- [ ] 指纹改变检测：尝试解密失败 → 提示重新激活

**Points**: 5

#### S-Xc.5: 心跳客户端 + 失效自动锁定

**Acceptance Criteria**:
- [ ] `internal/redemption/heartbeat.go`：每 5 分钟 POST `<hub>/api/v2/:tenant_slug/user/heartbeat`（新增 endpoint TBD）+ fingerprint
- [ ] 心跳响应：`active` / `revoked` / `expired`
- [ ] 连续 3 次失败 → 进入"待重连"状态（72h 宽限期内仍可使用）
- [ ] 收到 `revoked` 立即清空 token + 跳激活页 + toast 通知
- [ ] 离线宽限期超时 → 同样跳激活页

**Points**: 3

#### S-Xc.6: EndUserMainPage 简化 Dashboard

**Acceptance Criteria**:
- [ ] 顶部：经销商品牌 logo + 当前余额（从 newhub `/api/v2/:tenant_slug/user/me` 拉）
- [ ] 中间：CLI 工具卡片（仅显示已安装 + 一键启动）
- [ ] 底部：本月用量 + 联系经销商按钮（mailto 或 webURL，由白标配置）
- [ ] 隐藏 Personal / Reseller 模式所有特性（高级配置、Agent、网关等）

**Points**: 5

**Sprint 4e 总计**: 31 points

---

## 📦 已冻结 Epics: E4-E10 (Agent Fleet 路线，Phase A-C 完成后回归)

> **状态**：S4.1 (`internal/agent/` 已写)、S4.4 (`AgentsPage.tsx`、`agentStore.ts`、`bindings_agent.go` 已写) 部分入库。
> **决策 (ADR-020)**：渠道分销 Pivot 期间冻结，Personal 模式仍可见已实现部分（路由守卫限 Personal）。
> 下面 E4-E10 内容**保留供回归参考，当前 Sprint 不执行**。

---

## Phase 0: Agent 基础 (Sprint 4 Old, 🟡 Hold)

### Epic 4: Agent Foundation / Agent 基础数据层 — 🟡 Hold

**Goal**: 建立 Agent 数据模型和多实例配置能力。

#### S4.1: Agent Profile 数据模型 + SQLite

**As a** developer,
**I want** a structured Agent data model persisted in SQLite,
**So that** each agent has a unique identity and can be managed independently.

**Acceptance Criteria**:
- [ ] 新建 `internal/agent/` 包
- [ ] `AgentProfile` struct: ID (UUID), Name, Icon (emoji/path), Tags[], ToolType (claude/codex/gemini/openclaw/zeroclaw/picoclaw/nullclaw), ModelID, SystemPrompt, MCPServers[], Permissions, BudgetLimit, Status (created/running/stopped/error), CreatedAt, UpdatedAt
- [ ] SQLite store (WAL mode): Create/Read/Update/Delete/List agents
- [ ] `%APPDATA%/lurus-switch/switch.db` as single DB file
- [ ] Migration support (version table, auto-migrate on startup)
- [ ] Unit tests for CRUD operations
- [ ] `go test ./internal/agent/... → PASS`

**Points**: 8

#### S4.2: Multi-Instance Config Support

**As a** user,
**I want** to create multiple named configurations for the same tool,
**So that** "Claude-Frontend" and "Claude-Backend" can coexist with different settings.

**Acceptance Criteria**:
- [ ] Each Agent Profile links to a **named config variant** (not the global tool config)
- [ ] Config variants stored in `%APPDATA%/lurus-switch/agent-configs/<agent-id>/`
- [ ] Config generation uses agent's model, prompt, MCP, permissions (not global defaults)
- [ ] Existing `config.Store` extended with `SaveAgentConfig(agentID, toolType, config)` / `LoadAgentConfig(agentID)`
- [ ] Tests: create 3 Claude agents with different models, verify each generates correct config

**Points**: 5

#### S4.3: Process-Agent Binding

**As a** system,
**I want** each running process linked to its Agent Profile,
**So that** I can show which agent is running vs just showing PIDs.

**Acceptance Criteria**:
- [ ] `internal/process/monitor.go` extended: `LaunchAgent(ctx, agentID) (sessionID, error)` — reads AgentProfile → generates temp config → launches tool process
- [ ] Running processes tracked with `agentID` in session metadata
- [ ] `ListAgents()` returns agents with their current process state (running/stopped)
- [ ] When process exits, agent status updated to "stopped" (or "error" if non-zero exit)
- [ ] `StopAgent(agentID)` → gracefully stops the associated process
- [ ] Tests: launch agent, verify PID tracking, stop agent, verify cleanup

**Points**: 5

#### S4.4: Agent Wails Bindings + Basic Frontend

**As a** user,
**I want** to see a list of my agents in the UI,
**So that** I can begin managing them visually.

**Acceptance Criteria**:
- [ ] `bindings_agent.go`: CreateAgent, ListAgents, GetAgent, UpdateAgent, DeleteAgent, LaunchAgent, StopAgent
- [ ] New Zustand store: `agentStore.ts` (agents[], selectedAgent, CRUD actions)
- [ ] New page: `AgentsPage.tsx` — simple list view with agent name, tool icon, status badge, actions (start/stop/delete)
- [ ] Navigation: add "Agents" tab in sidebar (between Home and Tools)
- [ ] i18n keys for all new strings (zh + en)
- [ ] `cd frontend && bun run build → success`

**Points**: 8

**Sprint 4 Total: 26 points** (larger sprint due to foundational nature)

---

## Phase 1: Agent 生命周期管理 (Sprint 5-6, 4 weeks)

### Epic 5: Agent Lifecycle / Agent 生命周期

**Goal**: 完整的 agent 创建→运行→监控→恢复流程。

#### S5.1: Agent Creation Wizard

**As a** user,
**I want** a step-by-step wizard to create a new agent,
**So that** I don't have to manually configure every field.

**Acceptance Criteria**:
- [ ] Multi-step wizard component:
  1. 选工具 (tool type selector with icons)
  2. 选模型 (model picker, filtered by tool)
  3. 设名称 + 图标 + 标签
  4. 设系统提示词 (optional, from template or custom)
  5. 设 MCP servers (optional, from presets)
  6. 设预算 (optional, default unlimited)
  7. 确认 + 创建
- [ ] "从模板创建" shortcut bypasses steps 2-6
- [ ] Created agent appears in agent list immediately
- [ ] i18n complete

**Points**: 8

#### S5.2: Agent Dashboard (Home Page Redesign)

**As a** user,
**I want** the home page to show all my agents at a glance,
**So that** I know what's running, what's idle, and what's broken.

**Acceptance Criteria**:
- [ ] Global metrics bar: 🟢 N running · 🟡 N idle · 🔴 N error · 💰 today $X / $Y · ⏱️ Nk tokens/h
- [ ] Agent card grid (responsive, 2-4 columns):
  - Status LED (green/yellow/red/gray)
  - Agent name + tool icon
  - Current task (if available via stdout parsing)
  - Runtime + token consumption + budget progress bar
  - Quick actions: [Start/Stop] [Logs] [Config]
- [ ] Filter: by status, by tool, by project, by tag
- [ ] Sort: by name, by status, by token consumption, by creation date
- [ ] Empty state: "No agents yet" → link to creation wizard
- [ ] Replaces current HomePage (preserve health score as a collapsible section)

**Points**: 8

#### S5.3: Health Check + Auto-Restart

**As a** user,
**I want** agents to be automatically restarted when they crash,
**So that** my work isn't interrupted by transient failures.

**Acceptance Criteria**:
- [ ] `internal/agent/health.go`: periodic health check (process alive? responsive?)
- [ ] Check interval: configurable per agent, default 30 seconds
- [ ] On crash detection: auto-restart with exponential backoff (5s → 10s → 30s → 60s → give up)
- [ ] Max restart attempts: configurable, default 5
- [ ] After max restarts: set status to "error", stop retrying, emit alert event
- [ ] Wails event: `agent:health:change` (agentID, oldStatus, newStatus)
- [ ] Frontend: real-time status update on agent cards
- [ ] Tests: simulate process exit, verify restart behavior

**Points**: 5

#### S5.4: Agent Log Stream

**As a** user,
**I want** to see real-time logs from each agent,
**So that** I can debug issues without opening a separate terminal.

**Acceptance Criteria**:
- [ ] `internal/logstream/` package: captures stdout+stderr per agent
- [ ] Ring buffer per agent (last 5000 lines, configurable)
- [ ] Wails binding: `GetAgentLogs(agentID, lastN)` + `StreamAgentLogs(agentID)` via events
- [ ] Frontend: log viewer panel (slide-out or full page)
  - Auto-scroll to bottom
  - Pause/resume auto-scroll
  - Search within logs
  - Clear logs
  - Copy selected
- [ ] Log level coloring (stderr = red, stdout = normal)

**Points**: 5

#### S5.5: Agent Clone

**As a** user,
**I want** to duplicate an existing agent,
**So that** I can quickly create variants without reconfiguring from scratch.

**Acceptance Criteria**:
- [ ] `CloneAgent(sourceID, newName)` → creates new AgentProfile with same config, new ID/name
- [ ] Clone preserves: tool type, model, system prompt, MCP, permissions, budget settings
- [ ] Clone does NOT copy: runtime state, logs, metering history
- [ ] UI: "Clone" button in agent card dropdown menu
- [ ] Cloned agent starts in "stopped" state

**Points**: 3

#### S5.6: Batch Operations

**As a** user managing 10+ agents,
**I want** to select multiple agents and perform bulk actions,
**So that** I don't have to click each one individually.

**Acceptance Criteria**:
- [ ] Multi-select in agent list (checkboxes)
- [ ] "Select all" / "Select by tag" / "Select by status"
- [ ] Batch actions: Start All, Stop All, Restart All, Delete All (with confirmation)
- [ ] Batch action progress indicator
- [ ] Keyboard shortcuts: Ctrl+A (select all), Delete (with confirmation)

**Points**: 5

**Sprint 5 Total: 21 points**
**Sprint 6 Total: 13 points** (S5.4-S5.6 overflow to Sprint 6)

---

## Phase 2: 资源管控 (Sprint 7, 2 weeks)

### Epic 7: Resource Management / 资源管控

**Goal**: 控制多 agent 的成本爆炸。

#### S7.1: Per-Agent Budget

**As a** user,
**I want** to set a token budget for each agent,
**So that** no single agent can drain my balance unexpectedly.

**Acceptance Criteria**:
- [ ] `internal/agent/budget.go`:
  - `SetBudget(agentID, limit TokenBudget)` — daily/monthly/total limit in tokens or currency
  - `CheckBudget(agentID) (remaining, exceeded bool)`
  - Budget enforcement in gateway middleware: if agent's app token exceeds budget → reject with 429
- [ ] Overage policy per agent: `pause` (default) | `degrade` (switch to cheaper model) | `notify_only`
- [ ] Budget usage stored in SQLite (daily granularity)
- [ ] UI: budget setting in agent creation wizard + agent detail page
- [ ] Budget progress bar on agent card

**Points**: 8

#### S7.2: Burn Rate Dashboard

**As a** user,
**I want** to see real-time token consumption rates,
**So that** I can predict my monthly spend and take action before it's too late.

**Acceptance Criteria**:
- [ ] Metrics computed from metering data:
  - Current hour burn rate (tokens/hour, $/hour)
  - Today's total spend
  - This month's total + projected month-end
  - Per-agent breakdown (top N consumers)
- [ ] Visualization: line chart (7-day trend) + bar chart (per-agent ranking)
- [ ] Warning thresholds: 50% / 80% / 100% of global budget → color change
- [ ] Accessible from Dashboard page and Analytics page

**Points**: 5

#### S7.3: Global Budget + Smart Degradation

**As a** user,
**I want** a global monthly spending cap,
**So that** I never exceed my budget even if I forget to set per-agent limits.

**Acceptance Criteria**:
- [ ] Global budget in Settings: monthly cap (currency)
- [ ] When 80% reached: desktop notification warning
- [ ] When 95% reached: auto-degrade all agents to cheapest available model
- [ ] When 100% reached: pause all non-essential agents (user marks "essential" per agent)
- [ ] Override: user can manually resume paused agents (one-time override for current day)
- [ ] Budget reset: monthly on billing cycle date

**Points**: 5

#### S7.4: Cost Report & Export

**As a** user,
**I want** detailed cost reports by agent, model, project, and time period,
**So that** I can optimize my spending.

**Acceptance Criteria**:
- [ ] Report page with filters: date range, agent, model, project
- [ ] Pivot table: rows = agents, columns = models, values = tokens & cost
- [ ] Time series: daily cost trend with agent breakdown
- [ ] Export: CSV download
- [ ] Aggregation: total, average, max, min per dimension

**Points**: 5

**Sprint 7 Total: 23 points**

---

## Phase 3: 上下文与知识管理 (Sprint 8, 2 weeks)

### Epic 8: Context & Knowledge / 上下文与知识

**Goal**: 解决多 agent 的知识碎片化问题。

#### S8.1: Project Workspace

**As a** user working on multiple projects,
**I want** to group agents by project and share context within a project,
**So that** all agents on the same project have access to the same knowledge.

**Acceptance Criteria**:
- [ ] `internal/project/` package: Project (ID, Name, Description, ContextFiles[], AgentIDs[])
- [ ] SQLite store: CRUD projects
- [ ] Assign agents to projects (many-to-one: agent belongs to one project)
- [ ] Project context files: markdown/txt files that get injected into agent system prompts on launch
- [ ] UI: Projects page with project list + bound agents + context file editor
- [ ] Navigation: add "Projects" tab in sidebar

**Points**: 8

#### S8.2: Context Template Library

**As a** user,
**I want** to save and reuse combinations of system prompt + CLAUDE.md + MCP config,
**So that** I can quickly apply the same context setup to new agents.

**Acceptance Criteria**:
- [ ] `internal/template/context.go`: ContextTemplate (name, systemPrompt, claudeMdContent, mcpServers[], tags[])
- [ ] Builtin templates: "Code Reviewer", "Technical Writer", "Test Engineer", "Data Analyst" (with appropriate prompts)
- [ ] User can save current agent's context as a new template
- [ ] Apply template to agent (overwrite or merge)
- [ ] Stored in `%APPDATA%/lurus-switch/templates/`

**Points**: 5

#### S8.3: Agent Snapshot & Resume

**As a** user,
**I want** to save an agent's working state and resume it later,
**So that** I don't lose progress when I need to stop and restart work.

**Acceptance Criteria**:
- [ ] Agent snapshot captures: config + system prompt + working directory + last 100 log lines + metering summary
- [ ] Create snapshot: manual or auto-on-stop
- [ ] Resume from snapshot: create new agent from snapshot → same config → same working dir
- [ ] Snapshot list in agent detail page with timestamps
- [ ] Stored in `%APPDATA%/lurus-switch/snapshots/agents/<agent-id>/`

**Points**: 5

#### S8.4: Shared Knowledge Base

**As a** user,
**I want** a set of documents that all agents in a project can access,
**So that** shared knowledge (API specs, coding standards, FAQs) doesn't need to be duplicated.

**Acceptance Criteria**:
- [ ] Knowledge base = folder of markdown/txt files per project
- [ ] Auto-injected into system prompt header on agent launch (configurable: inject all / inject by tag)
- [ ] UI: knowledge base editor in project detail page (add/edit/delete documents)
- [ ] Size limit: warn if total injection > 10,000 tokens

**Points**: 5

**Sprint 8 Total: 23 points**

---

## Phase 4: 监控与可观测性 (Sprint 9, 2 weeks)

### Epic 9: Monitoring & Observability / 监控与可观测性

**Goal**: 完整的运维可见性。

#### S9.1: Unified Log Center

**As a** user with 10+ agents,
**I want** a single place to see logs from all agents,
**So that** I can investigate issues without switching between individual agent logs.

**Acceptance Criteria**:
- [ ] Log aggregation page: combined view of all agent logs
- [ ] Filters: by agent, by log level (stdout/stderr), by time range, by keyword
- [ ] Color-coded by agent (each agent a different color)
- [ ] Timestamp alignment across agents
- [ ] Real-time streaming (new logs appear live)

**Points**: 5

#### S9.2: Performance Comparison Panel

**As a** user,
**I want** to compare the efficiency of different agents/models,
**So that** I can optimize which model to use for which task type.

**Acceptance Criteria**:
- [ ] Comparison table: agent name, tool, model, avg tokens/request, error rate, total cost, runtime
- [ ] Sortable by any column
- [ ] Time period selector
- [ ] "Best value" highlight (lowest cost per successful request)

**Points**: 5

#### S9.3: Alert Rules + Desktop Notifications

**As a** user,
**I want** to be notified when something goes wrong,
**So that** I can take action before problems escalate.

**Acceptance Criteria**:
- [ ] Alert rules engine: condition → action
  - Agent crashed (N times in M minutes) → notification
  - Budget > X% used → notification
  - Agent idle > N hours → notification (potential waste)
  - Global spend rate > $X/hour → notification
- [ ] Desktop notification via OS native API (Wails notification API or direct syscall)
- [ ] Alert history page (last 100 alerts with timestamps)
- [ ] Configurable: enable/disable per rule

**Points**: 5

#### S9.4: Audit Log

**As a** user (especially enterprise),
**I want** a complete record of all agent operations,
**So that** I have accountability and can troubleshoot.

**Acceptance Criteria**:
- [ ] Audit events: agent created/started/stopped/deleted, budget changed, config changed, alert fired
- [ ] Stored in SQLite (audit_log table)
- [ ] UI: audit log page with filters (event type, agent, date range)
- [ ] Export: CSV
- [ ] Auto-purge: keep last 90 days (configurable)

**Points**: 3

**Sprint 9 Total: 18 points**

---

## Phase 5: 模板生态 (Sprint 10, 1-2 weeks)

### Epic 10: Template Ecosystem / 模板生态

**Goal**: 降低新用户门槛，形成生态飞轮。

#### S10.1: Built-in Agent Templates

**As a** new user,
**I want** pre-made agent templates I can use immediately,
**So that** I don't have to figure out the optimal configuration myself.

**Acceptance Criteria**:
- [ ] 15+ built-in templates across categories:
  - **Development**: Code Reviewer, Backend Developer, Frontend Developer, Test Engineer, DevOps Assistant
  - **Writing**: Technical Writer, Blog Writer, Translator
  - **Analysis**: Data Analyst, Code Auditor, Security Reviewer
  - **Productivity**: Meeting Summarizer, Email Drafter, Research Assistant
  - **Automation**: Bot Manager (ZeroClaw), Task Automator (OpenClaw)
- [ ] Each template includes: name, description, icon, tool type, recommended model, system prompt, MCP suggestions
- [ ] Template browser UI with category filters and search
- [ ] "Use this template" → opens creation wizard with pre-filled fields

**Points**: 5

#### S10.2: Custom Templates + Import/Export

**As a** power user,
**I want** to save my agent configurations as templates and share them,
**So that** I can reuse my setups and help others.

**Acceptance Criteria**:
- [ ] "Save as template" from any agent's detail page
- [ ] Template editor: edit name, description, icon, and all config fields
- [ ] Export template as `.json` file
- [ ] Import template from `.json` file
- [ ] Template versioning (auto-increment on save)
- [ ] Template stored in `%APPDATA%/lurus-switch/templates/`

**Points**: 5

#### S10.3: One-Click Deploy from Template

**As a** user,
**I want** to create a running agent from a template with one click,
**So that** the setup is instant.

**Acceptance Criteria**:
- [ ] "Deploy" button on template card → creates agent + auto-starts
- [ ] If required tool not installed → prompt to install first
- [ ] Auto-name: template name + counter ("Code Reviewer #3")
- [ ] Default budget from template (if specified) or user's global default

**Points**: 3

**Sprint 10 Total: 13 points**

---

## Phase 6: 高级编排 (Sprint 11+, 远期, 此处仅规划不分解)

### Epic 11: Advanced Orchestration / 高级编排

**Scope**: Agent 分组标签、任务队列、Agent 间通信、工作流编辑器、多机管理、安全策略引擎。

**Not decomposed into stories yet** — will be planned after Phase 5 delivery based on user feedback and market validation.

**Key capabilities**:
- Task queue with priority scheduling
- Agent-to-agent output routing (A's output → B's input)
- Visual workflow editor (multi-agent pipeline)
- Remote machine management (via SSH/Tailscale tunnel)
- Fine-grained security policies (per-agent filesystem/network sandboxing)

**Estimated**: 40+ points, 3-4 sprints

---

## Summary / 总览

| Sprint | Phase | Epic | Points | 状态 |
|--------|-------|------|--------|------|
| S1 | — | E1: Foundation Cleanup | 23 | ✅ 完成 |
| S2 | — | E2: Onboarding & Dashboard | 21 | ✅ 完成 |
| S3 | — | E3: Visual Config Editor V2 | 21 | ✅ 完成 |
| S4 (old) | P0 | E4: Agent Foundation | 26 | 🟡 部分入库后冻结 |
| **S4b** | **A** | **E12: AppMode Tri-State** | **21** | **📋 当前 Sprint** |
| S4c | B | E13: Hub Integration (part 1) | 22 | 📋 计划 |
| S4d | B | E13: Hub Integration (part 2) + E14: Reseller Deploy | 22 | 📋 计划 |
| S4e | C | E15: EndUser Mode + 白标 | 31 | 📋 计划 |
| S5+ | (resume) | E5-E10 Agent Fleet | ~91 | 🟡 Hold (Phase A-C 后回归) |
| S11+ | P6 | E11: Advanced Orchestration | ~40 | 🔮 远期 |

**Channel Distribution Pivot (Phase A-C)**: Sprint 4b-4e, 96 points, 8 weeks
**Agent Fleet Resume (M2-resume)**: Sprint 5+, 91 points, ~9 weeks (Phase A-C 完成后)
**Total active planned: ~96 points (Pivot)**, Total roadmap incl. resume: ~187 + 65 (M1 done)**
