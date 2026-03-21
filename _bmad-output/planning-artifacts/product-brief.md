---
date: 2026-03-20
author: Anita
status: draft-v2
supersedes: product-brief-v1 (tool config manager positioning)
---

# Product Brief: Lurus Switch

# 产品简报：Lurus Switch — 本地 AI 接入平台

---

## 0. 一句话定位

**Switch 是安装在用户电脑上的"AI 接入层"：任何需要调用大模型的桌面应用、CLI 工具或脚本，接入 Switch 就能用，费用走 Lurus 统一计费。**

类比：
- Switch 之于 LLM API = **Wi-Fi 路由器之于互联网** — 你不管每个设备怎么联网，接上路由器就行
- Switch 之于 AI 应用 = **Steam 之于游戏** — 开发者不用自己做支付/DRM，接入平台就有用户和收入

---

## 1. Vision / 愿景

### Problem Statement / 问题陈述

**对应用开发者（供给侧）：**

1. **接入成本高** — 每个想用 LLM 的桌面应用都要自己处理：API Key 管理、多 Provider 适配、计费、用量追踪。重复造轮子。
2. **用户流失** — 用户嫌配 API Key 麻烦，或者不想在又一个 App 里绑信用卡。注册→配置→充值的漏斗每一步都在流失。
3. **没有分发渠道** — 做了个 AI 应用，怎么让用户发现？App Store 不收桌面应用，独立开发者缺乏曝光。

**对终端用户（需求侧）：**

1. **多应用多账号** — 用 Claude Code 要一个 Key，用 Cursor 要一个 Key，用某个 AI 工具又要一个 Key。每个应用单独充值、单独管理。
2. **费用不透明** — 不知道哪个应用花了多少钱，月底各平台账单加起来才知道总花费。
3. **配置重复** — 每个应用都要配一遍 Provider/模型/代理，换个模型要改 N 个地方。

### Vision Statement / 愿景声明

**Lurus Switch 是本地 AI 接入平台：用户充一次钱，所有应用都能用；开发者接一个 SDK，用户和计费都有了。Switch 让 AI 能力像电一样，插上就用。**

### Unique Value Proposition / 独特价值主张

| 角色 | 价值 |
|------|------|
| **对用户** | 一份余额 → 所有 AI 应用通用；一个面板 → 看清每个应用花了多少 |
| **对 AI 应用开发者** | 零成本接入 LLM 能力 — 不用管 Key/计费/Provider，指向 `localhost` 就行 |
| **对 Lurus** | 每一次 AI 调用都走平台计费 — Switch 是 token 销售的本地分发渠道 |

---

## 2. 核心商业逻辑

```
                    ┌──────────────────────────────────┐
                    │         Lurus Cloud (NewAPI)       │
                    │   多 Provider 聚合 / 计费 / 路由    │
                    └──────────────┬───────────────────┘
                                   │ API (OpenAI-compatible)
                    ┌──────────────▼───────────────────┐
                    │       Lurus Switch (本地)          │
                    │  ┌─────────────────────────────┐  │
                    │  │   Local Gateway (localhost)  │  │
                    │  │   • 鉴权 (per-app token)     │  │
                    │  │   • 用量计量                  │  │
                    │  │   • 智能路由                  │  │
                    │  │   • 缓存 / 重试              │  │
                    │  └──────────┬──────────────────┘  │
                    │             │                      │
                    │  ┌──────── │ ────────────────┐    │
                    │  │         │                  │    │
                    └──┼─────────┼──────────────────┼────┘
                       │         │                  │
                ┌──────▼──┐ ┌───▼────┐ ┌───────────▼──────┐
                │Claude   │ │Cursor  │ │ 第三方 AI App     │
                │Code     │ │/Codex  │ │ (任何需要 LLM 的   │
                │         │ │        │ │  桌面应用/脚本)    │
                └─────────┘ └────────┘ └──────────────────┘
```

**钱怎么流：**
1. 用户在 Switch 里充值（支付宝/微信 → Lurus Identity 账户余额）
2. 任何应用调用 `localhost:PORT` → Switch 转发到 Lurus Cloud → 扣余额
3. Lurus Cloud 用批量采购价调用上游 Provider（OpenAI/Anthropic/DeepSeek...）
4. **利润 = 用户支付 - 上游成本** （聚合差价 + 增值服务费）

**为什么用户愿意走 Switch 而不是自己配 Key：**
- 比自己买便宜（聚合批量折扣）
- 一次充值全部应用通用（不用每个 App 单独绑卡）
- 有统一的费用视图和预算控制
- 新应用零配置，安装即用

---

## 3. Target Users / 目标用户

### Persona 1: AI Power User / AI 重度用户（核心付费用户）

- **画像**: 每天使用 3+ AI 工具，月 API 消费 ¥200-2000
- **核心需求**: 统一管理费用，一份余额多工具通用，费用透明
- **当前方案**: 各平台分别充值，手动追踪费用
- **Switch 价值**: 充一次钱 → 所有工具通用，Dashboard 看清每分钱花在哪
- **付费意愿**: 高 — 愿意为便利性和折扣付费

### Persona 2: AI App Developer / AI 应用开发者（供给侧）

- **画像**: 独立开发者/小团队，正在做需要 LLM 的桌面应用
- **核心需求**: 不想自己处理 Key 管理和计费，希望有现成用户群
- **当前方案**: 要么让用户自带 Key（体验差），要么自己搭计费系统（成本高）
- **Switch 价值**: 应用只需 `baseURL = "http://localhost:19090"` → 用户已有余额，直接能用
- **付费意愿**: 免费接入，Lurus 从 API 调用中抽成

### Persona 3: Casual AI User / AI 普通用户（增长层）

- **画像**: 偶尔用 AI 工具，月消费 ¥20-100
- **核心需求**: 简单方便，不想折腾配置
- **当前方案**: 直接用 ChatGPT/Claude Web
- **Switch 价值**: 安装 Switch → 从"应用商店"装个 AI 工具 → 充点钱就能用
- **付费意愿**: 低但稳定 — 小额高频

---

## 4. Core Features / 核心功能

### 4.1 Local Gateway（本地网关）— **核心中的核心**

Switch 启动后，在本地运行一个 OpenAI-compatible API Server：

```
http://localhost:19090/v1/chat/completions
http://localhost:19090/v1/embeddings
http://localhost:19090/v1/models
...
```

| 能力 | 说明 |
|------|------|
| **OpenAI 兼容** | 任何支持 OpenAI SDK 的应用，改一行 `base_url` 就能接入 |
| **Per-App Token** | 每个注册的应用分配独立 token，用量独立计量 |
| **透明代理** | 请求 → Switch 本地 → Lurus Cloud NewAPI → 上游 Provider |
| **智能缓存** | 相同请求短时间内命中本地缓存，省钱省延迟 |
| **自动重试 + Fallback** | 上游 503 自动切备用 Provider，用户无感 |
| **离线队列** | 网络断开时请求入队，恢复后自动重发（非流式） |

**现有基础**: `serverctl` 已能管理嵌入式 NewAPI binary 的生命周期（start/stop/config/auto-start）。需要增强为 always-on 模式 + per-app metering。

### 4.2 App Connector（应用连接器）— 让接入极简

**对已知工具（Claude Code、Codex、Gemini CLI 等）：**
- 自动检测已安装工具 → 一键注入 Switch 网关地址 + App Token
- 现有 `installer` + `envmgr` + `toolconfig` 能力完全复用

**对任意第三方应用：**

```
接入只需 3 步：
1. 开发者在 Switch "App Registry" 注册（填名称 + 图标）
2. 获得 per-app token（或让用户在 Switch 里手动添加）
3. 应用代码里: base_url = "http://localhost:19090", api_key = "{app_token}"
```

| 接入方式 | 复杂度 | 适用场景 |
|---------|--------|---------|
| **Zero-config** | 零 | 已知工具（Claude/Codex/Gemini），Switch 自动检测+注入 |
| **Standard** | 改一行代码 | 任何用 OpenAI SDK 的应用，改 `base_url` |
| **Deep Integration** | 集成 SDK | 想在应用内显示余额/用量/模型选择器的应用 |

**Deep Integration SDK（未来）：**
```typescript
// 应用内嵌入 Switch SDK
import { LurusSwitch } from '@lurus/switch-sdk'

const sw = new LurusSwitch({ appId: 'my-app' })
const balance = await sw.getBalance()       // 查余额
const models = await sw.listModels()        // 可用模型
sw.openTopup()                              // 打开 Switch 充值界面
```

### 4.3 Billing Hub（计费中心）

| 功能 | 说明 |
|------|------|
| **统一余额** | 一个 Lurus 账户余额，所有应用共享 |
| **Per-App 账单** | 按应用维度的消费明细（今日/本周/本月） |
| **Per-Model 账单** | 按模型维度（claude-sonnet 花了多少，gpt-4o 花了多少） |
| **预算控制** | 给每个应用设月度预算上限，超了自动断 |
| **充值** | 支付宝/微信扫码充值，兑换码充值 |
| **订阅计划** | 月付包（含基础额度 + 折扣倍率） |
| **费用预警** | 余额不足 / 某应用异常消费 → 桌面通知 |

**现有基础**: `billing` 包已有 GetUserInfo/GetQuotaSummary/TopUp/Subscribe/RedeemCode 等完整接口。需要增加 per-app 维度。

### 4.4 App Store / 应用发现（Phase 3）

- 展示"可搭配 Switch 使用的 AI 应用"列表
- 一键安装 + 自动配置网关连接
- 应用评分、用量排名、推荐
- 开发者入驻 → 应用上架 → 从用户消费中分成

### 4.5 保留并重新定位的现有功能

| 现有功能 | 新定位 | 变化 |
|---------|--------|------|
| Tool Config Management | "已知应用连接器" — App Connector 的子集 | 从"核心功能"降为"便利功能"，服务于一键配置流 |
| MCP Presets | "跨应用 MCP 共享" | 不变，MCP 配好后多个工具共用 |
| Relay Management | 合并到 Gateway 路由配置 | Relay 概念升级为 Gateway 的 upstream provider 配置 |
| Config Snapshot + Diff | 保留 | 配置安全网，不变 |
| Prompt Library | 保留 | 共享 Prompt 资产 |
| Self Updater | 保留 | 不变 |
| GY Products | 升级为 App Store 的种子内容 | 从硬编码 3 个产品 → 动态 App Registry |
| Promoter | 保留并强化 | 推广返利，核心增长引擎 |
| Process Monitor | 保留 | 监控本地 AI 工具进程 |
| Analytics | 强化 | 增加 per-app 用量追踪 |
| DocMgr (CLAUDE.md) | 保留 | 开发者工具的增值功能 |

---

## 5. Architecture / 架构

### 5.1 组件图

```
┌─────────────────────────────────────────────────────┐
│                   Lurus Switch                       │
│                                                      │
│  ┌─────────────┐  ┌───────────────┐  ┌───────────┐  │
│  │  Wails GUI   │  │ Local Gateway │  │  Tray Icon │  │
│  │  (React)     │  │ (HTTP Server) │  │  (systray) │  │
│  │              │  │               │  │            │  │
│  │ • Dashboard  │  │ • /v1/chat/*  │  │ • 状态显示 │  │
│  │ • Billing    │  │ • /v1/models  │  │ • 快捷操作 │  │
│  │ • App Mgmt   │  │ • /v1/embed   │  │ • 余额预览 │  │
│  │ • Settings   │  │ • /health     │  │            │  │
│  │ • App Store  │  │ • /metrics    │  │            │  │
│  └──────┬───────┘  └──────┬────────┘  └─────┬─────┘  │
│         │                 │                  │        │
│  ┌──────▼─────────────────▼──────────────────▼─────┐  │
│  │              Core Services Layer                 │  │
│  │                                                  │  │
│  │  appRegistry   gateway     billing   analytics   │  │
│  │  appConnector  routing     metering  promoter    │  │
│  │  installer     cache       quota     updater     │  │
│  │  config        upstream    budget    snapshot    │  │
│  └──────────────────────┬──────────────────────────┘  │
│                         │                             │
└─────────────────────────┼─────────────────────────────┘
                          │ HTTPS
               ┌──────────▼──────────┐
               │   Lurus Cloud       │
               │                     │
               │  NewAPI (路由/计费)   │
               │  Identity (账户)     │
               │  多上游 Provider     │
               └─────────────────────┘
```

### 5.2 新增 internal 包规划

| 包名 | 职责 | 依赖关系 |
|------|------|---------|
| `gateway/` | 本地 HTTP 网关核心，OpenAI-compatible API 路由 | 替代 `serverctl`（不再嵌入外部 binary，自建） |
| `gateway/middleware/` | 鉴权（per-app token）、计量、限流、缓存 | |
| `gateway/upstream/` | 上游 Provider 管理（Lurus Cloud / 直连 / 自定义） | 整合 `relay/` |
| `appreg/` | App Registry — 应用注册、token 分发、元数据 | |
| `appconn/` | App Connector — 自动检测+注入配置到已知工具 | 复用 `installer/` + `toolconfig/` |
| `metering/` | Per-app / per-model 本地用量统计 | SQLite (嵌入式) |
| `budget/` | 预算控制 — per-app 配额管理 | 依赖 `metering/` |
| `tray/` | 系统托盘图标 + 快捷菜单 | |

### 5.3 保留不变的包

`config/`, `appconfig/`, `billing/`, `snapshot/`, `promptlib/`, `mcp/`,
`docmgr/`, `envmgr/`, `updater/`, `validator/`, `process/`, `analytics/`,
`promoter/`, `modelcatalog/`, `downloader/`, `proxydetect/`

### 5.4 重构/合并的包

| 现有包 | → 变化 |
|--------|--------|
| `serverctl/` | → 废弃，Gateway 不再嵌入外部 binary，而是 Switch 内建 HTTP Server |
| `relay/` | → 合并到 `gateway/upstream/`，Relay 概念升级为 upstream provider |
| `proxy/` | → 简化为存储用户 Lurus 账户凭证（endpoint + token），网关配置移到 `gateway/` |
| `installer/` | → 拆分：工具安装保留，配置注入移到 `appconn/` |
| `gy/` | → 升级为 `appreg/` 的种子数据，不再硬编码 |
| `toolconfig/` | → 合并到 `appconn/` |
| `generator/` | → 合并到 `appconn/` |

### 5.5 本地网关 vs 嵌入 NewAPI Binary

**关键架构决策：自建网关，不再嵌入 NewAPI binary。**

| 方案 | 优点 | 缺点 |
|------|------|------|
| 嵌入 NewAPI binary（现状） | 功能完整，有 admin UI | 体积大 (~50MB)，更新慢，配置复杂，难以定制 per-app metering |
| **自建轻量网关（新方案）** | 体积小，完全可控，per-app metering 原生支持，启动快 | 需要开发，仅做透明代理+计量（不需要完整的 NewAPI 功能） |

自建网关只需要：
1. OpenAI-compatible API 路由（透传到 Lurus Cloud）
2. Per-app token 鉴权
3. 本地用量计量（SQLite）
4. 缓存 + 重试 + fallback
5. `/health` + `/metrics` 端点

**不需要**：用户管理、渠道管理、日志管理（这些都在 Lurus Cloud 的 NewAPI 里）。

---

## 6. Revenue Model / 收入模型

### 核心模型：API 中间层抽成

```
用户在 Switch 充值 → Lurus 账户余额增加
应用调用 Switch 网关 → 转发到 Lurus Cloud → 按 token 数计费扣余额
Lurus 用批量价调用上游 → 赚差价

毛利率 = (用户支付价 - 上游成本) / 用户支付价
目标毛利率: 30-50%（通过批量折扣 + 智能路由 + 缓存实现）
```

### 收入来源

| 来源 | 占比（预期） | 说明 |
|------|-------------|------|
| **API Token 销售** | 70% | 核心：用户充值购买 API 额度 |
| **订阅计划** | 20% | Pro 月付包：含基础额度 + 更低单价 |
| **应用分成** | 5% | 第三方应用通过 Switch 产生的调用，平台抽成 |
| **增值服务** | 5% | 高级路由策略、优先通道、SLA 保障 |

### 定价层级

| 层级 | 价格 | 权益 |
|------|------|------|
| **免费** | ¥0 | 网关功能 + 自带 Key 模式（不走 Lurus 计费）|
| **按量付费** | 充值即用 | Lurus 定价（比官方便宜 10-30%）|
| **Switch Pro** | ¥39/月 | 含 ¥50 额度 + 额外调用 8 折 + 高级路由 + 费用分析 |
| **Switch Team** | ¥29/人/月（≥3 人）| 团队余额池 + 管理后台 + 审计 |

### 增长飞轮

```
更多应用接入 Switch
    → 用户装 Switch 的理由更多
        → 用户量增长
            → 充值量增长
                → 吸引更多应用接入
                    → 循环 ↑
```

---

## 7. Success Metrics / 成功指标

### North Star Metric / 北极星指标

**月 API 调用 GMV（通过 Switch 产生的 API 消费总额）**

### Key Metrics

| Category | Metric | Phase 1 Target | Phase 3 Target |
|----------|--------|----------------|----------------|
| **Revenue** | 月 API GMV | ¥10,000 | ¥100,000 |
| **Users** | 月活安装量 (MAU) | 500 | 5,000 |
| **Activation** | 首次充值转化率 | 15% | 25% |
| **Engagement** | 日均 API 调用次数/用户 | 50 | 200 |
| **Apps** | 接入 Switch 的应用数 | 5（内置） | 30+ |
| **Retention** | 30 日留存率 | 40% | 60% |
| **Gateway** | 网关可用性 | 99.5% | 99.9% |
| **Gateway** | 本地延迟开销 (p99) | < 50ms | < 20ms |

---

## 8. Competitive Landscape / 竞争格局

| 竞品 | 模式 | 优势 | Switch 差异 |
|------|------|------|------------|
| **OpenRouter** | Web API 聚合 | 200+ 模型，成熟 | Switch 是本地网关，Key 不出机器；深度集成桌面工具 |
| **TypingMind** | Web Chat UI + Key 管理 | 好用的 Chat | Switch 不做 Chat，做基础设施层 |
| **LiteLLM Proxy** | 开源 LLM 代理 | 免费开源 | Switch 有计费+App Store+GUI，不只是代理 |
| **各工具自带 Key** | 原生 | 无额外依赖 | Switch 提供统一账单+折扣+跨工具管理 |

### 护城河

1. **双边网络效应** — 应用越多 → 用户越多 → 开发者越愿意接入 → 应用越多
2. **余额锁定** — 用户充了钱在 Lurus 账户里，自然继续用
3. **习惯形成** — 所有工具都指向 `localhost:19090`，换掉 Switch 要改所有工具配置
4. **数据资产** — 用量分析、Prompt 库、MCP 配置、配置快照都在 Switch 里
5. **本地优先** — 中国市场对"数据不出境"的硬需求，云端竞品天然劣势

---

## 9. Scope & Boundaries / 范围与边界

### In Scope

- 本地 OpenAI-compatible API 网关（always-on）
- Per-app token 管理和用量计量
- 统一计费（充值/订阅/费用分析）
- 已知工具自动检测+一键注入网关
- App Registry（应用注册和发现）
- 系统托盘常驻（后台运行，不占前台）
- 预算控制（per-app 配额）
- 跨平台（Windows / macOS / Linux）

### Out of Scope

- Chat UI（不做聊天界面）
- 本地模型推理（不嵌入 Ollama）
- 云端 API 网关（那是 Lurus Cloud NewAPI 的事）
- 移动端
- IDE 插件（但可以被 IDE 插件连接）

### Constraints

1. **架构**: Wails v2 单二进制，网关内建，不嵌入外部 binary
2. **数据**: 用量统计用本地 SQLite，账户数据从 Lurus Cloud 拉取
3. **网络**: 必须能连 Lurus Cloud 才能用付费功能；自带 Key 模式可离线
4. **兼容**: 网关 API 严格 OpenAI-compatible，不发明私有协议

---

## 10. Roadmap / 分阶段实施

### Phase 1: Gateway Foundation（4 周）

**目标**: Switch 变成一个可用的本地 LLM 网关，用户充值后任何工具指向 localhost 即可调用。

| 任务 | 说明 | 涉及包 |
|------|------|--------|
| **P1.1 自建本地网关** | 内建 HTTP Server，OpenAI-compatible /v1/* 路由，透传到 Lurus Cloud | 新建 `gateway/` 替代 `serverctl/` |
| **P1.2 Per-App Token** | App Registry：注册应用 → 分配 token → 网关鉴权 | 新建 `appreg/` |
| **P1.3 本地计量** | SQLite 记录每次 API 调用（app_id, model, tokens_in, tokens_out, cost, timestamp） | 新建 `metering/` |
| **P1.4 系统托盘** | Switch 启动后常驻系统托盘，网关后台运行，GUI 按需打开 | 新建 `tray/` |
| **P1.5 Dashboard 改版** | 首页展示：网关状态、余额、今日调用量、per-app 消费排行 | 前端 DashboardPage 重写 |
| **P1.6 已知工具一键接入** | 检测 Claude/Codex/Gemini → 一键写入 Switch 网关 endpoint + app token | 重构 `installer/` → `appconn/` |

**Phase 1 验收标准**:
- 启动 Switch → 网关自动运行在 localhost:19090
- Claude Code 的 settings.json 被注入 Switch 网关地址
- 用 Claude Code 发一条消息 → Switch 计量到这次调用 → Dashboard 显示消费

### Phase 2: Billing & Control（3 周）

**目标**: 完整的费用管理闭环，用户能充值、能看账单、能控预算。

| 任务 | 说明 | 涉及包 |
|------|------|--------|
| **P2.1 Billing 页面升级** | 余额、充值、订阅、消费明细（per-app + per-model 维度） | 前端 BillingPage + `billing/` |
| **P2.2 预算控制** | 给每个应用设月预算上限，超额自动断 | 新建 `budget/` |
| **P2.3 费用预警** | 桌面通知：余额不足、某应用异常消费、预算即将耗尽 | `tray/` + OS notification |
| **P2.4 自带 Key 模式** | 网关支持 BYO Key（不走 Lurus 计费），用户可选 | `gateway/upstream/` |
| **P2.5 Relay → Upstream 迁移** | 将现有 Relay 概念升级为 Gateway 的上游路由配置 | 合并 `relay/` → `gateway/upstream/` |

**Phase 2 验收标准**:
- Dashboard 显示"本月 Claude Code 花了 ¥47，Codex 花了 ¥23"
- 给 Codex 设 ¥50/月预算 → 超额后 Codex 的请求被拒绝，Claude Code 不受影响
- 余额低于 ¥10 时弹出桌面通知

### Phase 3: App Ecosystem（4 周）

**目标**: 从"管理 5 个已知工具"扩展为"任意应用都能接入"。

| 任务 | 说明 | 涉及包 |
|------|------|--------|
| **P3.1 App Registry UI** | 用户可手动添加应用（名称+图标）→ 获得独立 app token | 前端 AppRegistryPage + `appreg/` |
| **P3.2 Cloud App Directory** | 从 Lurus Cloud 拉取推荐应用列表（名称/描述/接入指南） | `appreg/` + Cloud API |
| **P3.3 一键安装 + 自动配置** | 从 App Directory 安装应用 → 自动注入 Switch 网关 | `appconn/` + `installer/` |
| **P3.4 智能缓存** | 相同请求的 LLM 响应本地缓存（TTL + LRU），显著降低重复调用成本 | `gateway/middleware/` |
| **P3.5 GY 迁移** | 将 GY 硬编码产品迁移为 App Registry 的种子数据 | 废弃 `gy/` → `appreg/` |
| **P3.6 开发者文档** | "如何让你的应用接入 Switch" 接入指南 | docs |

**Phase 3 验收标准**:
- 用户手动注册一个 Python 脚本作为"应用" → 获得 token → 脚本用 OpenAI SDK 调用成功
- App Directory 展示 10+ 可接入应用
- 缓存命中时延迟 < 5ms

### Phase 4: Growth & Moat（持续）

| 任务 | 说明 |
|------|------|
| **P4.1 智能路由** | 按任务复杂度/成本偏好自动选 Provider+Model |
| **P4.2 推广返利强化** | 分享链接 → 被推荐用户消费 → 推荐人返利 |
| **P4.3 Switch SDK** | npm/pip 包，应用内嵌入余额查询/模型选择/充值跳转 |
| **P4.4 团队版** | 团队余额池 + 管理后台 + 审计日志 |
| **P4.5 开发者分成** | 第三方应用通过 Switch 产生的消费，开发者获得分成 |

---

## 11. Technical Risks / 技术风险

| Risk | Impact | Likelihood | Mitigation |
|------|--------|-----------|------------|
| 网关延迟影响用户体验 | 高 | 中 | 本地纯转发开销 < 5ms；性能基准测试+持续监控 |
| 自建网关 vs NewAPI 功能差距 | 中 | 低 | Switch 网关只做透传+计量，不需要完整 NewAPI 功能 |
| 用户不信任本地代理安全性 | 高 | 中 | 开源网关代码；安全审计；本地数据加密 |
| 上游 Provider ToS 风险 | 高 | 低 | 法务审查；作为"增值服务"定位而非纯转售 |
| 双边市场冷启动 | 高 | 高 | 自有工具（Claude/Codex/Gemini 连接器）作为种子供给；充值折扣作为种子需求 |
| Wails 系统托盘支持不完善 | 中 | 中 | 使用 `getlantern/systray` 或 `fyne.io/systray` 独立库 |

---

## 12. Key Decisions / 关键决策

### D1: 自建网关 vs 继续嵌入 NewAPI

**决策: 自建轻量网关。**

理由：
- NewAPI binary ~50MB，Switch 目前 < 20MB，嵌入后体积翻倍
- Per-app metering 是核心需求，嵌入 NewAPI 要 fork 修改，维护成本高
- Switch 网关只需透传+计量，90% 的 NewAPI 功能（用户管理/渠道管理/admin UI）不需要
- 自建可完全控制启动速度、内存占用、API 兼容性

### D2: 始终运行 vs 按需启动

**决策: 系统托盘常驻，网关始终运行。**

理由：
- 应用随时可能调 API，网关必须始终可用
- 类似 Docker Desktop 的体验 — 后台运行，不打扰
- 内存占用目标 < 30MB（纯 Go HTTP Server，无 VM）

### D3: 产品目录名称

**现状**: 代码在 `2c-gui-switch`，问题文件在 `2c-app-switch`
**建议**: 统一为 `2c-gui-switch`（desktop GUI 准确反映产品形态），`2c-app-switch` 仅保留规划文档或归档。

---

## 13. 现有代码改造清单

### 保留（无需改动）

| 文件/包 | 原因 |
|---------|------|
| `main.go` | Wails bootstrap，结构不变 |
| `internal/appconfig/` | App 设置，不变 |
| `internal/billing/` | 计费 client 完全复用 |
| `internal/config/` | 工具配置 model，复用 |
| `internal/docmgr/` | CLAUDE.md 管理，保留 |
| `internal/envmgr/` | Key 管理，复用 |
| `internal/mcp/` | MCP 预设，保留 |
| `internal/modelcatalog/` | 模型目录，保留 |
| `internal/packager/` | 打包工具，保留 |
| `internal/process/` | 进程监控，保留 |
| `internal/promoter/` | 推广系统，保留并强化 |
| `internal/promptlib/` | Prompt 库，保留 |
| `internal/snapshot/` | 快照，保留 |
| `internal/updater/` | 自更新，保留 |
| `internal/validator/` | 校验，保留 |
| `internal/downloader/` | 下载工具，保留 |
| `internal/proxydetect/` | 代理检测，保留 |
| `internal/analytics/` | 追踪，保留 |

### 重构

| 文件/包 | 改动 |
|---------|------|
| `app.go` | God Object 拆分（已在 Sprint 1 计划）；增加网关生命周期 |
| `services.go` | 替换 `serverMgr` → `gatewayMgr`；增加 `appRegistry`, `metering`, `budget` |
| `internal/proxy/` | 简化为 Lurus 账户凭证存储（endpoint + token），移除网关相关逻辑 |
| `internal/installer/` | 保留工具安装能力，配置注入拆分到 `appconn/` |
| `bindings_proxy.go` | QuickSetup/SwitchModel/ConfigureAllToolsRelay 移到 appconn bindings |
| `bindings_server.go` | Start/StopServer → Start/StopGateway |
| `bindings_relay.go` | Relay 概念升级为 upstream provider |
| `bindings_gy.go` | GY 产品移到 App Registry |

### 新增

| 文件/包 | 内容 |
|---------|------|
| `internal/gateway/` | 本地 HTTP 网关：路由、中间件、upstream 管理 |
| `internal/gateway/middleware/` | auth（per-app token）、metering、ratelimit、cache |
| `internal/gateway/upstream/` | 上游 provider 管理（Lurus Cloud / BYO Key / custom） |
| `internal/appreg/` | App Registry：注册、token 管理、元数据、cloud sync |
| `internal/appconn/` | App Connector：已知工具检测 + 配置注入 |
| `internal/metering/` | 本地 SQLite 计量：per-app / per-model 用量 |
| `internal/budget/` | 预算控制：per-app 配额、超额策略 |
| `internal/tray/` | 系统托盘：状态、余额、快捷操作 |
| `bindings_gateway.go` | 网关 Wails bindings |
| `bindings_appreg.go` | App Registry Wails bindings |
| `bindings_metering.go` | 用量统计 Wails bindings |
| `frontend/src/pages/AppRegistryPage.tsx` | 应用管理页面 |
| `frontend/src/pages/MeteringPage.tsx` | 用量分析页面 |
| `frontend/src/stores/gatewayStore.ts` | 网关状态 store（现有，需重构） |
| `frontend/src/stores/appStore.ts` | 应用列表 store |
| `frontend/src/stores/meteringStore.ts` | 用量数据 store |

### 废弃

| 文件/包 | 原因 |
|---------|------|
| `internal/serverctl/` | 不再嵌入外部 NewAPI binary |
| `internal/relay/` | 合并到 `gateway/upstream/` |
| `internal/toolconfig/` | 合并到 `appconn/` |
| `internal/generator/` | 合并到 `appconn/` |
| `internal/gy/` | 升级为 `appreg/` 种子数据 |
| Gateway* 前端页面（10 个） | 不再需要 NewAPI admin UI，替换为自建网关 Dashboard |
