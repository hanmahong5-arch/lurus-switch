# Lurus Switch 架构文档 / Architecture Document

## 架构概览 / Overview

本项目是 **Ailurus PaaS** 的核心组件，采用**本地网关 + 云端服务**混合架构，为 AI 开发者提供统一的模型访问、配额管理和多端同步能力。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              客户端层 / Client Layer                         │
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
│                 │    │                     │    │   - Stats/Alerts    │
│                 │    │                     │    │   - Users/Sessions  │
│                 │    │                     │    │   - Audit Logs      │
└─────────────────┘    └──────────┬──────────┘    └─────────────────────┘
                                  │
                   ┌──────────────▼──────────────┐
                   │          NEW-API            │
                   │      (LLM Unified Gateway)  │
                   │ OpenAI / Claude / Gemini    │
                   └─────────────────────────────┘
```

---

## 一、Monorepo 目录结构 / Directory Structure

```
lurus-switch/
├── codeswitch/           # 主桌面应用 (Go + Vue 3 + Wails 3)
│   ├── services/         # 后端服务层
│   ├── frontend/         # Vue 3 前端
│   ├── sync-service/     # 独立同步服务
│   └── deploy/           # 部署配置
│
├── gemini-cli/           # Gemini CLI 工具链 (Node.js + Bun)
│   ├── packages/core/    # 核心业务逻辑
│   ├── packages/cli/     # 终端 UI (Ink/React)
│   ├── packages/electron/# 桌面 GUI (Electron)
│   └── packages/nats-proxy/ # NATS 代理
│
├── new-api/              # LLM 统一网关服务 (Go)
│
├── nats/                 # NATS 服务器配置
│
└── Academic-De-AIGC/     # 学术工具
```

---

## 二、核心数据流 / Core Data Flow

### CodeSwitch 代理模式

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
         ├─── Local Provider ─── Direct API Call
         │
         └─── NEW-API Mode ─── NEW-API (:3000) ─── AI Providers
                                                        │
                                                        ▼
┌───────────────────────────────────────────────────────────────┐
│                       AI Providers                             │
│  OpenAI / Anthropic / Google Gemini / DeepSeek / ...          │
└───────────────────────────────────────────────────────────────┘
         │
         ▼ Response + Logging
┌───────────────────┐
│  SQLite (app.db)  │ ◄─── Write Queue (Async Batch)
│  Request Logs     │
└───────────────────┘
```

---

## 三、服务层详解 / Service Layer

### 1. CodeSwitch (本地网关) - Port :18100

主要职责：
- **多平台代理**：支持 Claude Code / Codex / Gemini CLI
- **供应商管理**：Provider 配置、优先级、故障转移
- **请求日志**：本地 SQLite 存储，费用预计算
- **NATS 集成**：可选的消息同步

| 路由 | 平台 | API 格式 |
|------|------|----------|
| `POST /v1/messages` | Claude Code | Anthropic API |
| `POST /responses` | Codex | OpenAI Responses API |
| `POST /v1/chat/completions` | Generic | OpenAI-compatible |
| `POST /v1beta/models/*` | Gemini CLI | Gemini Native → OpenAI 转换 |

### 2. NEW-API (统一网关) - Port :3000

主要职责：
- **40+ 供应商支持**：OpenAI, Anthropic, Google, DeepSeek, 国产模型等
- **配额与计费**：用户余额、Token 计量
- **支付集成**：Stripe 等支付渠道

### 3. Sync Service (同步服务) - Port :8081

主要职责：
- **NATS 消息处理**：多端同步、事件广播
- **Admin API**：运维监控、用户管理、审计日志
- **告警系统**：错误率、延迟、费用监控

| 路由 | 说明 |
|------|------|
| `GET /api/v1/admin/system/status` | 系统状态 |
| `GET /api/v1/admin/stats/overview` | 统计概览 |
| `GET /api/v1/admin/users` | 用户列表 |
| `GET /api/v1/admin/sessions` | 会话列表 |
| `GET /api/v1/admin/audit-logs` | 审计日志 |
| `GET /api/v1/admin/alert-rules` | 告警规则 |

### 4. NATS Server - Port :4222 / :8222 (WS)

主要职责：
- **消息队列**：JetStream 持久化
- **主题订阅**：
  - `chat.{user_id}.{session}.msg` - 多端消息同步
  - `llm.request.*` / `llm.response.*` - LLM 事件
  - `user.{user_id}.quota` - 配额变更广播

---

## 四、客户端产品 / Client Products

### 三大件 CLI (Core CLI Tools)

| 工具 | 说明 | 集成方式 |
|------|------|----------|
| **Claude Code** | Anthropic CLI 客户端 | 通过 CodeSwitch 代理 |
| **Codex** | OpenAI CLI 客户端 | 通过 CodeSwitch 代理 |
| **Gemini CLI** | Google Gemini CLI | 原生 NATS / CodeSwitch 代理 |

### GUI 客户端

| 客户端 | 技术栈 | 说明 |
|--------|--------|------|
| **CodeSwitch** | Wails 3 + Vue 3 | 主桌面应用，供应商管理 |
| **Gemini Electron** | Electron + React + rough.js | 手绘风格 Gemini GUI |
| **Flutter App** | Flutter | 移动端 (开发中) |
| **Admin Console** | Vue 3 | 运维监控后台 |

---

## 五、技术栈 / Technology Stack

| 层 | 技术 |
|-----|------|
| 桌面 GUI | Wails 3 + Vue 3 + Vite + Tailwind CSS 4 |
| Electron GUI | Electron + React + Vite + rough.js |
| 后端服务 | Go + Gin |
| CLI | Node.js + Bun (gemini-cli) |
| 消息总线 | NATS + JetStream |
| 数据库 | SQLite (本地), PostgreSQL (云端) |
| 缓存 | Redis |
| 移动端 | Flutter |

---

## 六、配置文件路径 / Configuration Paths

CodeSwitch 存储配置于 `~/.code-switch/`:

```
~/.code-switch/
├── claude-code.json    # Claude Code 供应商配置
├── codex.json          # Codex 供应商配置
├── mcp.json            # MCP 服务器配置
├── app.json            # 应用设置 (NEW-API, NATS 等)
├── sync-settings.json  # NATS 同步设置
└── app.db              # SQLite 数据库 (日志)
```

Gemini CLI 配置于 `~/.gemini/`:

```
~/.gemini/
├── settings.json       # 用户设置
└── .gemini/settings.json # 项目级设置
```

---

## 七、部署架构 / Deployment Architecture

### 跨境双机房模式

```
┌────────────────────────────────────────────────────────────────────────────┐
│                        外网服务商 / External Providers                       │
│               (OpenAI / Anthropic / Google / Cohere / ...)                 │
└────────────────────────────────────┬───────────────────────────────────────┘
                                     │
                                 V2RAY 代理
                                     │
┌────────────────────────────────────┼───────────────────────────────────────┐
│                                    ↓                                        │
│  ┌──────────────────────────────────────────────────────┐                  │
│  │           中国外 (ailurus.top)                         │                  │
│  │           123.56.80.174:31                            │                  │
│  │                                                       │                  │
│  │   NEW-API (:3000)  ←  V2RAY  →  外网 LLM              │                  │
│  │   NATS Server (:4222)                                 │                  │
│  │   CodeSwitch 分发服务                                  │                  │
│  └──────────────────────────────────────────────────────┘                  │
│                                                                             │
│                           无直接数据同步                                     │
│                                                                             │
│  ┌──────────────────────────────────────────────────────┐                  │
│  │           中国内 (lurus.cn)                            │                  │
│  │           47.104.234.209                              │                  │
│  │                                                       │                  │
│  │   NEW-API (:3000)  →  国内 LLM (DeepSeek 等)          │                  │
│  │   NATS Server (:4222)                                 │                  │
│  │   Academic-De-AIGC 服务                                │                  │
│  └──────────────────────────────────────────────────────┘                  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 数据隔离原则

- 中国外与中国内数据库**完全独立**
- 无跨境数据同步
- 符合数据合规要求

| 环境 | 域名 | IP 地址 | 位置 |
|------|------|---------|------|
| 海外生产 | ailurus.top | 123.56.80.174:31 | 海外服务器 |
| 国内生产 | lurus.cn | 47.104.234.209 | 阿里云 |

---

## 八、性能优化 / Performance Optimizations

### 1. 数据库写入队列

**问题**：SQLite 写锁竞争导致并发流式请求死锁

**方案**：
- 单线程写入队列 (`logWriteQueue chan`, buffer 1000)
- 批处理：10 条记录或 100ms 超时
- 非阻塞插入，溢出处理

### 2. 费用预计算

**问题**：统计查询重复计算价格

**方案**：
- 日志插入时预计算 7 个费用字段
- 统计查询直接读取存储的费用

### 3. HTTP 客户端超时

**问题**：请求无限挂起

**方案**：
- 非流式：60s 超时
- 流式：300s (5分钟) 超时
- 标准库 `http.Client` + 连接池

---

## 九、开发命令速查 / Quick Reference

### CodeSwitch

```bash
cd codeswitch
wails3 task dev        # 开发模式
wails3 task build      # 构建
wails3 task package    # 打包
go test ./...          # 测试
```

### Gemini CLI

```bash
cd gemini-cli
bun install            # 安装依赖
bun run start          # CLI 开发
bun run build          # 构建

cd packages/electron
bun run dev            # Electron 开发
bun run package:win    # Windows 打包
```

### NEW-API

```bash
cd new-api
go build -o new-api .  # 构建
./new-api              # 运行
```

### NATS

```bash
cd nats
./nats-server.exe      # 运行 NATS 服务器
```
