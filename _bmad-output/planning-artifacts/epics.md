# Lurus Switch - Epics & Sprint Planning

**Version**: 2.0
**Date**: 2026-04-10
**Aligned PRD**: product-brief v3 (龙虾管理员 — AI 助理舰队管理中心)
**Sprint Duration**: 2 weeks
**Supersedes**: epics v1.0 (Sprint 4-8 replaced; Sprint 1-3 retained as completed)

---

## Overview / 概览

Sprint 1-3 (M1 MVP: Foundation + Onboarding + Config Editor) 已于 2026-02-28 完成。
本文档重新规划 Sprint 4+，对齐龙虾管理员 v3 愿景。

### Milestone-to-Phase Mapping

| Milestone | Phase | Epics | Sprints | 目标 |
|-----------|-------|-------|---------|------|
| M1 (已完成) | — | E1-E3 | S1-S3 | 基础清理 + 入门 + 配置编辑器 |
| M2: Agent MVP | P0-P2 | E4-E7 | S4-S7 | Agent 管理 + 预算控制 = 可用的舰队管理器 |
| M3: Agent Pro | P3-P5 | E8-E10 | S8-S10 | 上下文 + 监控 + 模板 = 完整产品 |
| M4: Ecosystem (远期) | P6 | E11 | S11+ | 编排 + 多机 + 社区 |

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

## Phase 0: Agent 基础 (Sprint 4, 2 weeks)

### Epic 4: Agent Foundation / Agent 基础数据层

**Goal**: 建立 Agent 数据模型和多实例配置能力，为后续所有 Agent 功能奠基。

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
| S4 | P0 | E4: Agent Foundation | 26 | 📋 计划 |
| S5 | P1 | E5: Agent Lifecycle (part 1) | 21 | 📋 计划 |
| S6 | P1 | E5: Agent Lifecycle (part 2) | 13 | 📋 计划 |
| S7 | P2 | E7: Resource Management | 23 | 📋 计划 |
| S8 | P3 | E8: Context & Knowledge | 23 | 📋 计划 |
| S9 | P4 | E9: Monitoring & Observability | 18 | 📋 计划 |
| S10 | P5 | E10: Template Ecosystem | 13 | 📋 计划 |
| S11+ | P6 | E11: Advanced Orchestration | ~40 | 🔮 远期 |

**M2 MVP (P0-P2): Sprint 4-7, ~83 points, ~8 weeks**
**M3 Pro (P3-P5): Sprint 8-10, ~54 points, ~6 weeks**
**Total planned: ~202 points (including M1 65pt completed)**
