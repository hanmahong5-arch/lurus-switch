---
date: 2026-04-10
author: Anita
status: draft-v3
supersedes: product-brief-v2 (local AI gateway platform positioning)
---

# Product Brief: Lurus Switch — 龙虾管理员

# 产品简报：Lurus Switch — AI 助理舰队管理中心

---

## 0. 一句话定位

**Switch 是安装在用户电脑上的"AI 助理管理中心"：创建、调度、监控、预算控制数十个 AI 助理实例，统一计费，一个面板掌控全局。**

类比演进：
- v1: **设置面板** — 管理 AI 工具的配置文件
- v2: **Wi-Fi 路由器** — 所有 AI 应用走统一网关和计费
- **v3: 舰队指挥中心** — 管理 N 个运行中的 AI 助理实例的生命周期、资源、协作

核心隐喻：
- Switch 之于 AI 助理 = **Docker Desktop 之于容器** — 创建、启动、停止、监控、资源限制
- Switch 之于 LLM API = **路由器之于互联网** — 统一入口、流量计量、访问控制（v2 能力保留）
- Switch 之于 AI 工具生态 = **Homebrew 之于命令行工具** — 安装、更新、依赖管理

---

## 1. Vision / 愿景

### Problem Statement / 问题陈述

**v2 已识别的问题（保留）：**

1. **多应用多账号** — 每个 AI 工具单独 Key、单独充值、单独管理
2. **费用不透明** — 不知道哪个工具花了多少钱
3. **配置重复** — 每个工具独立配置 Provider/模型/代理

**v3 新增识别的问题：**

4. **多实例管理困难** — 用户可能同时运行 5-20 个 AI 助理实例（不同项目的 Claude Code、多个 ZeroClaw bot、多个 OpenClaw agent），没有统一视图
5. **实例无身份** — 20 个进程在跑，分不清哪个在做什么。PID 和进程名没有业务含义
6. **成本失控** — 多个助理并行运行，token 消耗速率不可预测，月底账单远超预期
7. **上下文碎片化** — 每个助理各自维护上下文，项目知识无法共享，助理崩溃后上下文丢失
8. **运维空白** — 助理崩了不知道、卡住不知道、重复消耗不知道。没有健康检查、自动重启、告警
9. **从零配置门槛高** — 每次创建新助理都要从头配置工具+模型+提示词+MCP+权限，新用户望而却步

### Vision Statement / 愿景声明

**Lurus Switch 是 AI 助理舰队管理中心：用户在一个面板里创建、调度、监控数十个 AI 助理，每个助理有独立身份、预算和权限。Switch 让管理 AI 助理像管理 Docker 容器一样简单 — 从模板一键创建，实时监控资源消耗，崩溃自动重启，成本全程可控。**

底层继承 v2 的 Gateway + 统一计费能力，作为不可见的基础设施层。

### Unique Value Proposition / 独特价值主张

| 角色 | v2 价值（保留） | v3 新增价值 |
|------|----------------|------------|
| **对用户** | 一份余额 → 所有 AI 应用通用 | 一个面板 → 管理所有助理的生命周期、预算、健康 |
| **对重度用户** | 费用透明 | 20 个助理一目了然，成本自动控制，崩溃自动恢复 |
| **对开发者** | 零成本接入 LLM | 助理模板生态 — 发布模板 → 用户一键创建 |
| **对 Lurus** | Token 销售渠道 | 多实例 = 更多 token 消耗 = 更高 GMV |

---

## 2. 核心商业逻辑

### 2.1 基础层（v2 保留）

```
用户充值 → Lurus 账户余额
任何助理调用 → Switch 本地 Gateway → Lurus Cloud → 扣余额
Lurus 赚批量采购差价
```

### 2.2 管理层（v3 新增）

```
用户创建 Agent Profile → 选工具+模型+模板 → 分配预算
    → Switch 管理实例生命周期 (start/stop/restart/health-check)
    → 实时监控 token 消耗和运行状态
    → 预算耗尽 → 自动暂停/降级
    → 崩溃检测 → 自动重启
```

**更多助理 = 更多 token 消耗 = 更高收入。管理层直接驱动计费层增长。**

### 2.3 模板层（v3 新增）

```
内置模板 ("代码审查员", "文档写手", "数据分析师"...)
    → 用户一键创建 → 零配置启动
社区模板 → 用户分享/导入 → 生态扩展
```

---

## 3. Target Users / 目标用户

### Persona 1: AI Power Developer / AI 重度开发者（核心用户）

- **画像**: 同时推进 3-5 个项目，每个项目配 1-3 个 AI 助理（coding、review、doc），月消费 ¥500-3000
- **核心需求**: 一个面板管理所有项目的所有助理；切换项目不用手动改配置；每个项目独立预算
- **当前方案**: 手动开关不同的 CLI 实例，终端标签页混乱，记不清哪个在哪
- **Switch 价值**: Dashboard 一眼看清所有助理状态，按项目分组，一键启停

### Persona 2: Agent Farmer / Agent 农场主（高价值用户）

- **画像**: 运行 10-30 个 ZeroClaw/OpenClaw 实例，各自服务不同 Telegram 群/Discord 频道/业务流程
- **核心需求**: 批量管理、健康监控、自动重启、成本控制
- **当前方案**: 手写 systemd service + 自己写监控脚本，运维负担重
- **Switch 价值**: 图形化的"容器管理器"，批量操作，自动健康检查，预算自动断

### Persona 3: AI Power User / AI 重度用户（付费用户，v2 保留）

- **画像**: 每天使用 3+ AI 工具，月 API 消费 ¥200-2000
- **核心需求**: 统一计费，费用透明
- **Switch 价值**: 充一次钱 → 所有工具通用

### Persona 4: AI 探索者 / AI Explorer（增长层）

- **画像**: 想试各种 AI 助理但不知道从何开始
- **核心需求**: 低门槛上手，预设配置，不用学每个工具的配置格式
- **Switch 价值**: 从模板库选一个 → 一键创建 → 立刻可用

---

## 4. Core Features / 核心功能

### 4.0 Agent Fleet Management（助理舰队管理）— **v3 核心新增**

| 能力 | 说明 |
|------|------|
| **Agent Profile** | 每个助理有独立身份：名称、图标、标签、绑定工具、绑定模型、权限等级 |
| **多实例** | 同一工具可创建多个助理实例，各有独立配置和上下文 |
| **生命周期管理** | 创建 → 启动 → 运行 → 暂停 → 恢复 → 终止。完整状态机 |
| **Agent 仪表盘** | 卡片式总览：状态灯、当前任务、token 消耗、运行时长、预算进度 |
| **批量操作** | 全选/按标签/按项目，批量启停/重启/删除 |
| **创建向导** | 选工具 → 选模型 → 填名称 → 设预算 → 选模板（可选）→ 完成 |
| **Agent 克隆** | 从已有助理快速创建副本 |
| **自动重启** | 健康检查 → 崩溃检测 → 自动重启（带重启次数上限 + 退避）|
| **日志流** | 实时查看每个助理的 stdout/stderr |

### 4.1 Resource Management（资源管控）

| 能力 | 说明 |
|------|------|
| **每 Agent 预算** | 独立 token 预算，到期策略：暂停/降级/通知 |
| **全局预算** | 日/月硬上限，触及后暂停所有非关键助理 |
| **Burn Rate 仪表盘** | 实时消耗速率 + 趋势预测 + 月末预估 |
| **智能降级** | 接近预算时自动切换到更便宜的模型 |
| **成本报表** | 按 agent / 模型 / 项目 / 时间 的多维成本分析，可导出 |

### 4.2 Context & Knowledge（上下文与知识管理）

| 能力 | 说明 |
|------|------|
| **项目空间** | 项目下绑定多个 agent，共享项目级上下文 |
| **上下文模板** | 系统提示词 + CLAUDE.md + MCP 配置的组合模板 |
| **Agent 快照** | 导出 agent 上下文（配置+提示词+工作进度摘要）|
| **快照恢复** | 从快照创建新 agent，继承前一个 agent 的工作上下文 |
| **共享知识库** | 跨 agent 共享文档/FAQ/代码规范，启动时自动注入 |

### 4.3 Monitoring & Observability（监控与可观测性）

| 能力 | 说明 |
|------|------|
| **统一日志** | 所有 agent 日志汇聚，按 agent/级别/时间筛选 |
| **性能对比** | 横向对比 agent 效率：任务完成率、平均 token、出错率 |
| **告警规则** | agent 挂了、预算超标、连续失败 N 次 → 桌面通知 |
| **审计日志** | 完整操作记录 |

### 4.4 Template Ecosystem（模板生态）

| 能力 | 说明 |
|------|------|
| **内置模板** | 15+ 预制 agent 模板：代码审查员、文档写手、测试工程师、数据分析师… |
| **自定义模板** | 将现有 agent 保存为模板 |
| **导入/导出** | JSON 格式，支持社区分享 |
| **一键部署** | 从模板创建 + 自动配置 + 启动 |

### 4.5 Local Gateway（本地网关）— v2 保留

（内容同 v2 product-brief，此处不重复。核心：OpenAI-compatible API Server on localhost:19090，per-app token，透传 Lurus Cloud，本地计量。）

### 4.6 Billing Hub（计费中心）— v2 保留

（内容同 v2。核心：统一余额、per-app 账单、充值/订阅、费用预警。）

### 4.7 保留的 v2 功能

Tool Config Management, MCP Presets, Config Snapshots, Prompt Library, Self Updater, Promoter, Process Monitor, DocMgr (CLAUDE.md)。

---

## 5. Architecture / 架构

### 5.1 分层架构

```
┌──────────────────────────────────────────────────────────┐
│  Layer 3: Orchestration (远期)                             │
│  工作流定义 · 多 Agent 协作 · 任务路由 · Agent 间通信        │
├──────────────────────────────────────────────────────────┤
│  Layer 2: Fleet Management (v3 核心)                      │
│  Agent Profile · 生命周期 · 预算控制 · 监控告警              │
│  项目空间 · 上下文管理 · 模板 · 日志                        │
├──────────────────────────────────────────────────────────┤
│  Layer 1: Infrastructure (v2 已建)                        │
│  Local Gateway · 计费 · 安装检测 · 配置生成                  │
│  MCP · Prompt · Snapshot · Relay · Auth                   │
├──────────────────────────────────────────────────────────┤
│  Layer 0: Platform (Wails + React)                        │
│  Go backend · React frontend · SQLite · IPC               │
└──────────────────────────────────────────────────────────┘
```

### 5.2 新增 Go 包

| 包名 | 职责 |
|------|------|
| `internal/agent/` | Agent Profile CRUD + 实例生命周期管理 (start/stop/restart/health) |
| `internal/agent/budget.go` | Per-agent 预算控制 + 降级策略 |
| `internal/project/` | 项目空间 — 多 agent 绑定 + 共享上下文 |
| `internal/template/` | Agent 模板管理 (builtin + custom + import/export) |
| `internal/monitor/` | 指标采集 + 告警引擎 + 报表生成 |
| `internal/logstream/` | Agent 日志聚合 + 实时流 + 搜索 |

### 5.3 数据存储

| 数据 | 存储 | 位置 |
|------|------|------|
| Agent Profiles | SQLite | `%APPDATA%/lurus-switch/switch.db` |
| Agent 运行日志 | SQLite + 文件轮转 | `%APPDATA%/lurus-switch/logs/` |
| 用量计量 | SQLite (已有 metering) | `%APPDATA%/lurus-switch/metering/` |
| 模板 | JSON 文件 | `%APPDATA%/lurus-switch/templates/` |
| 项目空间 | SQLite | `switch.db` |
| 快照 | JSON 文件 (已有) | `%APPDATA%/lurus-switch/snapshots/` |

### 5.4 UI 导航重构

```
当前 (v2):
  Home → Tools → Gateway → Workspace → Account → Settings

目标 (v3):
  Dashboard (舰队总览)
    └─ 全局指标条 + Agent 卡片网格

  Agents (助理管理)
    ├─ 列表/网格视图 + 创建向导
    ├─ 单个 agent 详情 (配置/日志/指标/快照)
    └─ 批量操作

  Projects (项目空间)
    ├─ 项目列表 + 绑定的 agent
    └─ 共享上下文管理

  Templates (模板库)
    ├─ 内置模板 + 我的模板
    └─ 导入/导出

  Gateway (网关) — v2 保留
    ├─ 状态控制 + 流量监控
    └─ Relay 管理

  Analytics (分析)
    ├─ 成本报表 (per-agent/model/project)
    ├─ 性能对比
    └─ 审计日志

  Settings (设置)
    ├─ 账户 & 计费
    ├─ 外观 & 语言
    └─ 安全策略 & 全局预算
```

---

## 6. Revenue Model / 收入模型

（同 v2，核心不变。管理层直接放大计费层收入。）

| 来源 | 占比 | v3 增长驱动 |
|------|------|------------|
| **API Token 销售** | 70% | 多实例 = 更多 token 消耗 |
| **订阅计划** | 20% | Agent 数量限制推动 Pro 升级 |
| **模板市场** | 5% (新) | 开发者发布付费模板 |
| **增值服务** | 5% | 高级监控、优先通道 |

### 免费 vs Pro 边界

| 能力 | 免费 | Pro (¥39/月) |
|------|------|-------------|
| Agent 数量 | ≤ 5 | 无限 |
| 并发运行 | ≤ 3 | 无限 |
| 预算控制 | 全局预算 | Per-agent 预算 + 智能降级 |
| 模板 | 内置 | 内置 + 自定义 + 社区 |
| 日志保留 | 7 天 | 90 天 |
| 项目空间 | 1 个 | 无限 |

---

## 7. Success Metrics / 成功指标

### North Star Metric

**月活 Agent 实例数 (MAI) × 月均 Token 消耗**

### Key Metrics

| Category | Metric | Phase 1 | Phase 3 |
|----------|--------|---------|---------|
| **Agent** | 月活 Agent 实例数 | 2,000 | 20,000 |
| **Agent** | 平均每用户 Agent 数 | 3 | 8 |
| **Revenue** | 月 API GMV | ¥10,000 | ¥150,000 |
| **Users** | MAU | 500 | 5,000 |
| **Activation** | 首次创建 Agent 转化率 | 60% | 80% |
| **Engagement** | 日均活跃 Agent 数/用户 | 2 | 5 |
| **Template** | 模板使用率 | 40% | 70% |
| **Retention** | 30 日留存 | 45% | 65% |

---

## 8. Competitive Landscape / 竞争格局

| 竞品 | 模式 | Switch v3 差异 |
|------|------|---------------|
| **Docker Desktop** | 容器管理 | Switch 专注 AI 助理，内置模板+计费+上下文管理 |
| **OpenRouter** | Web API 聚合 | Switch 是本地管理器，管的是助理实例不是 API |
| **LangChain/CrewAI** | 多 Agent 编排框架 | Switch 是产品不是框架，零代码操作 |
| **Cursor/Windsurf** | AI IDE | Switch 管理多个工具实例，不限于 IDE |
| 各工具自带管理 | 原生 | Switch 跨工具统一管理 |

### 护城河

1. **多实例锁定** — 用户的 20 个 agent 配置、预算、日志都在 Switch 里，迁移成本极高
2. **模板网络效应** — 模板越多 → 新用户上手越快 → 用户越多 → 模板越多
3. **计费粘性** — 余额在 Lurus，agent 越多消耗越快，充值越频繁
4. **上下文资产** — 项目空间、共享知识库、快照，这些是用户积累的智力资产

---

## 9. Scope & Boundaries / 范围与边界

### In Scope

- Agent 多实例生命周期管理
- Per-agent 预算和资源控制
- 项目空间和上下文共享
- Agent 模板库 (builtin + custom)
- 统一日志和监控
- 本地 Gateway + 统一计费（v2 保留）
- 系统托盘常驻
- 跨平台 (Windows / macOS / Linux)

### Out of Scope

- Chat UI（不做聊天界面）
- 本地模型推理（不嵌入 Ollama）
- 多 Agent 编排（Phase 6 远期，不在 MVP）
- Agent 间实时通信（Phase 6 远期）
- 远程机器管理（Phase 6 远期）
- 移动端

---

## 10. Roadmap / 分阶段实施

| Phase | 名称 | 时长 | 核心目标 |
|-------|------|------|---------|
| **0** | Agent 基础 | 2 周 | 数据模型、多实例配置、SQLite、进程关联 |
| **1** | 舰队管理 | 3 周 | 创建向导、仪表盘、启停、健康检查、日志流 |
| **2** | 资源管控 | 2 周 | Per-agent 预算、Burn Rate、降级、成本报表 |
| **3** | 上下文 | 2 周 | 项目空间、模板库、快照继承、共享知识 |
| **4** | 监控 | 2 周 | 统一日志、性能对比、告警、桌面通知、审计 |
| **5** | 模板生态 | 1 周 | 内置模板、自定义、导入导出、一键部署 |
| **6** | 高级编排 | 3 周 (远期) | 任务队列、Agent 间通信、工作流、多机管理 |

**MVP = Phase 0 + Phase 1 + Phase 2 (7 周)**

---

## 11. Technical Risks / 技术风险

| Risk | Impact | Likelihood | Mitigation |
|------|--------|-----------|------------|
| 进程管理跨平台差异 | 高 | 高 | Windows/macOS/Linux 各有 `exec_*.go`，已有模式 |
| SQLite 并发写冲突 | 中 | 中 | WAL 模式 + 写锁超时 + 读写分离 |
| 20+ 实例内存压力 | 中 | 中 | Switch 本身 < 50MB，管理的是外部进程不是嵌入进程 |
| Agent 崩溃级联 | 高 | 低 | 独立进程模型，各实例隔离，崩溃不传染 |
| 用户可能不需要这么多助理 | 高 | 中 | 模板降低门槛；即使只管 3 个也比手动好 |
| 现有 Process Monitor 能力不足 | 中 | 高 | Phase 0 重构，从 PID 监控升级为 Agent 生命周期管理 |

---

## 12. Key Decisions / 关键决策

### D4: Agent 是进程还是配置？

**决策: Agent = 配置 Profile + 可选的运行实例。**

Agent 可以存在但不运行（已配置但未启动）。启动时 Switch 生成配置文件 → 启动对应工具的子进程 → 关联 PID → 监控。

这意味着 Switch 不嵌入 AI 工具的逻辑，而是**编排外部工具进程**。

### D5: 全新 UI 还是渐进式改造？

**决策: 渐进式改造。**

在现有导航中插入 "Agents" 和 "Projects" 页面，Dashboard 改版为 Agent 总览。保留 Gateway / Settings / Account 等已有页面。分 Phase 逐步替换。

### D6: SQLite 还是纯文件？

**决策: SQLite（WAL 模式）。**

Agent 元数据、运行日志、用量统计需要结构化查询（按时间、按 agent、按 model 聚合）。JSON 文件无法高效支持。已有 metering 也可迁移到 SQLite。
