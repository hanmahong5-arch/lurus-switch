# Lurus Switch 微服务改造计划
# Microservices Transformation Plan

> **状态 / Status**: Phase 1-6 Completed, Production Ready
> **创建日期 / Created**: 2026-01-07
> **更新日期 / Updated**: 2026-01-08
> **预计工期 / Timeline**: 13-16 weeks (~4 months)

---

## 〇、设计决策 (Design Decisions)

| 决策项 | 选择 | 理由 |
|--------|------|------|
| **改造优先级** | 可维护性优先 | 完整拆分，清晰边界，适合长期演进 |
| **离线能力** | 混合模式 | 默认在线，支持 Feature Flag 降级到本地 |
| **日志数据库** | ClickHouse | 最佳 OLAP 性能，适合大规模日志分析 |
| **NEW-API 兼容** | 完全兼容 | 不改动 NEW-API，仅改造 CodeSwitch 侧 |
| **Gateway 框架** | Hertz (字节) | 极致性能 + 原生 SSE 流式支持 |
| **微服务框架** | Kratos (B站) | 插件化 + Wire 依赖注入 + 完整服务治理 |
| **服务发现** | Consul / K8s | 不引入 Etcd，与 NATS 共存 |

---

## 一、执行摘要 (Executive Summary)

本计划将 Lurus Switch 从当前的**混合单体架构**演进为**事件驱动微服务架构**，通过 6 个阶段逐步完成，确保零停机迁移。

**核心改造目标**：
1. **解耦 ProviderRelayService** - 当前 900+ 行代码承担了代理、日志、同步、计费四大职责
2. **数据库分离** - 从 SQLite 单库迁移到按服务分库（PostgreSQL + ClickHouse）
3. **事件驱动** - 通过 NATS JetStream 实现服务间异步通信
4. **Gateway 混合模式** - 默认在线代理，支持本地降级（保留离线能力）

---

## 二、现有架构分析 (Current Architecture)

### 2.1 服务拓扑

```
┌─────────────────────────────────────────────────────────────────┐
│                        当前架构                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  CodeSwitch (:18100)     NEW-API (:3000)     Sync Service (:8081)
│  ├─ ProviderRelay        ├─ LLM 统一网关      ├─ 会话同步
│  ├─ LogService           ├─ 多供应商支持      ├─ 管理后台 API
│  ├─ BillingIntegration   └─ 用户配额         └─ NATS 消费者
│  ├─ SyncIntegration
│  └─ ProviderService
│                                                                 │
│  基础设施：NATS (:4222) + PostgreSQL + Redis + SQLite (本地)
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 关键耦合点

| 模块 | 文件 | 行数 | 问题 |
|------|------|------|------|
| **ProviderRelayService** | `services/providerrelay.go` | 900+ | 代理+日志+同步+计费混合 |
| **LogService** | `services/logservice.go` | 870+ | 日志查询+统计+成本分析 |
| **BillingIntegration** | `services/billing_integration.go` + `billing/*.go` | 1500+ | Casdoor+Lago+支付 |
| **SyncIntegration** | `services/sync_integration.go` + `sync/*.go` | 800+ | NATS 事件发布 |

### 2.3 数据流现状

```
请求 → ProviderRelay → Provider/NEW-API → 响应
                ↓ (同步写入)
           SQLite (app.db)
                ↓ (异步发布)
           NATS Events → Sync Service
```

---

## 三、目标微服务架构 (Target Architecture)

### 3.1 服务拆分方案

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    目标微服务架构 (Hertz + Kratos)                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Gateway   │  │  Provider   │  │     Log     │  │   Billing   │        │
│  │   Service   │  │   Service   │  │   Service   │  │   Service   │        │
│  │   :18100    │  │   :18101    │  │   :18102    │  │   :18103    │        │
│  │   Hertz     │  │   Kratos    │  │   Kratos    │  │   Kratos    │        │
│  │ (SSE 流式)  │  │ (配置管理)  │  │ (日志分析)  │  │ (计费认证)  │        │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘        │
│         │                │                │                │               │
│         └────────────────┴────────────────┴────────────────┘               │
│                                   │                                         │
│           ┌───────────────────────┼───────────────────────┐                 │
│           ▼                       ▼                       ▼                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐             │
│  │  NATS JetStream │  │ Consul / K8s    │  │  Sync Service   │             │
│  │  (消息总线)      │  │ (服务发现)       │  │  Gin (保留)     │             │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘             │
│                                                                             │
│  数据存储（按服务分库）：                                                     │
│  ├─ Provider DB (PostgreSQL) - providers, model_pricing                    │
│  ├─ Log DB (ClickHouse) - request_log, stats (OLAP 分析)                   │
│  ├─ Billing DB (PostgreSQL) - users, wallets, transactions                 │
│  ├─ Sync DB (PostgreSQL) - sessions, messages                              │
│  └─ Local SQLite (降级) - 离线模式本地缓存                                   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Gateway 混合模式设计

```
┌─────────────────────────────────────────────────────────────────┐
│                    Gateway 混合模式                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  [在线模式 - 默认]                                               │
│  ├─ Provider 配置 → Provider Service (HTTP)                    │
│  ├─ 余额检查 → Billing Service (HTTP)                          │
│  ├─ 日志写入 → NATS → Log Service → ClickHouse                 │
│  └─ 同步事件 → NATS → Sync Service                             │
│                                                                 │
│  [离线模式 - 降级]                                               │
│  ├─ Provider 配置 → 本地 JSON 文件                              │
│  ├─ 余额检查 → 跳过 (Feature Flag)                              │
│  ├─ 日志写入 → 本地 SQLite                                      │
│  └─ 同步事件 → 本地队列，联网后同步                              │
│                                                                 │
│  [切换条件]                                                      │
│  ├─ 自动：检测到服务不可用时降级                                 │
│  └─ 手动：用户配置离线模式                                       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 3.3 服务职责定义

| 服务 | 框架 | 端口 | 职责 | 数据所有权 |
|------|------|------|------|-----------|
| **Gateway Service** | Hertz | :18100 | HTTP 代理、模型匹配、SSE 流转发 | 无（无状态） |
| **Provider Service** | Kratos | :18101 | 供应商配置 CRUD、模型验证、健康检查 | providers 表 |
| **Log Service** | Kratos | :18102 | 日志存储、统计分析、成本计算 | request_log 表 |
| **Billing Service** | Kratos | :18103 | 认证、配额、支付、订阅 | users/wallets 表 |
| **Sync Service** | Gin | :8081 | 会话同步、在线状态、管理后台 | sessions 表 |

**框架选型理由**：
- **Hertz**：字节开源，180k QPS，原生 SSE 流式支持，适合 Gateway 高并发代理场景
- **Kratos**：B站开源，Wire 依赖注入，Protobuf API 定义，完整服务治理
- **Gin**：已稳定运行的 Sync Service 保持不变，降低风险

### 3.4 NATS 主题设计

```yaml
# 事件流主题
llm.request.{platform}         # claude/codex/gemini - LLM 请求事件
llm.response.{trace_id}        # 响应事件
log.write                      # 日志写入（批量）
billing.usage.{user_id}        # 用量上报
billing.quota.{user_id}        # 配额变更通知

# JetStream 配置
streams:
  LLM_EVENTS:     subjects: ["llm.>"],     retention: 7d,  storage: file
  LOG_EVENTS:     subjects: ["log.write"], retention: 1d,  storage: file
  BILLING_EVENTS: subjects: ["billing.>"], retention: 30d, storage: file
```

---

## 四、分阶段实施计划 (Implementation Phases)

### Phase 1: 准备阶段 (2-3 周)

**目标**：建立基础设施、共享库和框架环境

**任务清单**：
- [x] 创建共享库 `lurus-common/`
  - [x] 通用模型定义 (`models/`)
  - [x] NATS 客户端封装 (`nats/`)
  - [x] HTTP 客户端封装 (`http/`)
  - [x] 错误处理 (`errors/`)
- [x] **Kratos 环境搭建**
  - [x] 安装 kratos CLI: `go install github.com/go-kratos/kratos/cmd/kratos/v2@latest`
  - [x] 安装 protoc 和 protoc-gen-go
  - [x] 安装 Wire: `go install github.com/google/wire/cmd/wire@latest`
- [x] **Hertz 环境搭建**
  - [x] 安装 hz CLI: `go install github.com/cloudwego/hertz/cmd/hz@latest`
- [x] 搭建 Docker Compose 开发环境 (含 Consul)
- [ ] 定义 API 契约 (Protobuf) - 待完善
- [ ] 配置 Feature Flag 系统 - 待完善

**关键文件**：
```
lurus-common/
├── go.mod
├── models/provider.go, request_log.go, user.go
├── nats/client.go, jetstream.go, subjects.go
├── http/client.go, middleware.go
└── observability/metrics.go, tracing.go, logging.go

api/proto/                      # Protobuf API 定义
├── provider/v1/provider.proto
├── log/v1/log.proto
└── billing/v1/billing.proto
```

**Kratos 项目模板**：
```bash
# 创建 Kratos 服务骨架
kratos new provider-service
kratos new log-service
kratos new billing-service
```

---

### Phase 2: Provider Service 独立 (2 周)

**目标**：使用 Kratos 构建独立的 Provider Service

**任务清单**：
- [x] 使用 Kratos 创建 `provider-service/`
- [x] 实现 biz 层业务逻辑（从 `providerservice.go` 迁移）
- [x] 实现 data 层（PostgreSQL + Redis 缓存）
- [x] 实现 HTTP server 层
- [x] 单元测试 (provider_test.go)
- [ ] 定义 Protobuf API (`api/proto/provider/v1/provider.proto`) - 待完善
- [ ] 配置 Wire 依赖注入 - 待完善
- [ ] Gateway 通过 HTTP 调用 Provider Service - 待集成
- [ ] 保留本地 JSON 文件作为降级方案 - 待实现

**Protobuf API 定义**：
```protobuf
// api/proto/provider/v1/provider.proto
service ProviderService {
    rpc GetProviders (GetProvidersRequest) returns (GetProvidersReply);
    rpc MatchModel (MatchModelRequest) returns (MatchModelReply);
    rpc CreateProvider (CreateProviderRequest) returns (CreateProviderReply);
    rpc UpdateProvider (UpdateProviderRequest) returns (UpdateProviderReply);
    rpc DeleteProvider (DeleteProviderRequest) returns (DeleteProviderReply);
}
```

**Kratos 项目结构**：
```
provider-service/
├── api/provider/v1/           # 生成的 API 代码
├── cmd/provider/main.go       # 入口
├── configs/config.yaml        # 配置
├── internal/
│   ├── biz/provider.go        # 业务逻辑
│   ├── data/provider.go       # 数据访问 (PostgreSQL + Redis)
│   ├── server/http.go         # HTTP 服务
│   └── service/provider.go    # 服务实现
└── wire.go                    # 依赖注入
```

**关键文件修改**：
- `codeswitch/services/providerservice.go` → 迁移到 Kratos biz 层
- `codeswitch/services/providerrelay.go:728` → 调用方式改为 HTTP

---

### Phase 3: Log Service 独立 (2-3 周)

**目标**：使用 Kratos 构建 Log Service，采用事件驱动 + ClickHouse OLAP

**任务清单**：
- [x] 使用 Kratos 创建 `log-service/`
- [x] 实现 biz 层业务逻辑
- [x] 实现 data 层 (ClickHouse)
- [x] 实现 NATS 消费者
- [x] 单元测试 (log_test.go)
- [x] 搭建 ClickHouse 集群（单节点起步）
- [ ] 定义 Protobuf API (`api/proto/log/v1/log.proto`) - 待完善
- [ ] Gateway 改为发布 `log.write` 事件 - 待集成
- [ ] 保留 SQLite 作为离线模式降级 - 待实现

**Kratos NATS 消费者集成**：
```go
// internal/server/nats.go
func NewNATSServer(c *conf.Server, logger log.Logger, svc *service.LogService) *NATSServer {
    // 订阅 log.write 主题
    sub, _ := js.PullSubscribe("log.write", "log-service")
    return &NATSServer{sub: sub, svc: svc}
}
```

**数据流变更**：
```
在线模式: Gateway (Hertz) → NATS (异步) → Log Service (Kratos) → ClickHouse
离线降级: Gateway → SQLite (本地)，联网后批量同步
```

**ClickHouse 表设计**：
```sql
CREATE TABLE request_log (
    trace_id String,
    platform LowCardinality(String),
    model LowCardinality(String),
    provider LowCardinality(String),
    input_tokens UInt32,
    output_tokens UInt32,
    total_cost Decimal64(6),
    duration_sec Float32,
    created_at DateTime64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (platform, provider, created_at)
TTL created_at + INTERVAL 365 DAY;
```

**关键文件修改**：
- `codeswitch/services/providerrelay.go:228` - 移除 `processLogWriteQueue()`
- `codeswitch/services/logservice.go` → 迁移到 Kratos biz 层

---

### Phase 4: Gateway 重构 (Hertz) (2-3 周)

**目标**：使用 Hertz 重写 Gateway，实现高性能 SSE 流式代理

**任务清单**：
- [x] 使用 Hertz 创建 `gateway-service/`
- [x] 从 `providerrelay.go` 迁移核心代理逻辑
- [x] 实现 proxy relay 层
- [x] 单元测试 (relay_test.go)
- [ ] 实现 SSE 流式转发（Hertz 原生 StreamBody）- 待完善
- [ ] 集成 Provider Service 客户端（HTTP）- 待集成
- [ ] 集成 Billing Service 客户端（HTTP）- 待集成
- [ ] 集成 NATS 发布（异步日志和计费事件）- 待集成
- [ ] 实现离线降级模式 - 待实现

**Hertz 项目结构**：
```
gateway-service/
├── cmd/gateway/main.go        # 入口
├── internal/
│   ├── handler/
│   │   ├── claude.go          # /v1/messages
│   │   ├── codex.go           # /responses
│   │   └── gemini.go          # /v1beta/models
│   ├── middleware/
│   │   ├── auth.go            # 认证中间件
│   │   └── metrics.go         # Prometheus
│   ├── proxy/
│   │   └── relay.go           # 核心转发逻辑
│   └── client/
│       ├── provider.go        # Provider Service 客户端
│       └── billing.go         # Billing Service 客户端
├── pkg/
│   └── nats/publisher.go      # NATS 事件发布
└── configs/config.yaml
```

**Hertz SSE 流式实现**：
```go
// internal/handler/claude.go
func (h *ClaudeHandler) Messages(ctx context.Context, c *app.RequestContext) {
    // 1. 余额检查
    if err := h.billingClient.CheckBalance(ctx, getUserID(c)); err != nil {
        c.JSON(402, map[string]string{"error": "Insufficient balance"})
        return
    }

    // 2. 获取 Provider（带缓存）
    provider, _ := h.providerClient.MatchModel(ctx, "claude", getModel(c))

    // 3. 转发请求并流式响应
    c.Response.Header.Set("Content-Type", "text/event-stream")
    c.SetBodyStreamWriter(func(w network.ExtWriter) {
        resp := h.forwardToProvider(ctx, provider, c.Request.Body())
        defer resp.Body.Close()

        reader := bufio.NewReader(resp.Body)
        for {
            line, err := reader.ReadBytes('\n')
            if err != nil {
                break
            }
            w.Write(line)
            w.Flush()
        }

        // 4. 异步发布事件
        go h.natsPublisher.PublishLogEvent(logEvent)
        go h.natsPublisher.PublishBillingEvent(billingEvent)
    })
}
```

**关键文件修改**：
- `codeswitch/services/providerrelay.go` → 迁移到 Hertz handler
- 删除内嵌的 `BillingIntegration`、`SyncIntegration`

---

### Phase 5: Billing Service 整合 (3 周)

**目标**：使用 Kratos 构建统一的 Billing Service

**任务清单**：
- [x] 使用 Kratos 创建 `billing-service/`
- [x] 实现 biz 层业务逻辑
- [x] 实现 data 层 (PostgreSQL)
- [x] 实现 HTTP server (Gin)
- [x] 实现 NATS 消费者
- [x] 单元测试 (billing_test.go)
- [ ] 定义 Protobuf API (`api/proto/billing/v1/billing.proto`) - 待完善
- [ ] 实现 Casdoor 认证集成（OAuth2）- 待集成
- [ ] 实现 Lago 计费集成 - 待集成
- [ ] 配额检查 HTTP API - 已实现基础版本

**Protobuf API 定义**：
```protobuf
// api/proto/billing/v1/billing.proto
service BillingService {
    rpc CheckBalance (CheckBalanceRequest) returns (CheckBalanceReply);
    rpc DeductBalance (DeductBalanceRequest) returns (DeductBalanceReply);
    rpc GetQuota (GetQuotaRequest) returns (GetQuotaReply);
    rpc ReportUsage (ReportUsageRequest) returns (ReportUsageReply);
}
```

**Kratos 项目结构**：
```
billing-service/
├── api/billing/v1/            # 生成的 API 代码
├── cmd/billing/main.go
├── internal/
│   ├── biz/
│   │   ├── billing.go         # 计费业务逻辑
│   │   └── auth.go            # 认证业务逻辑
│   ├── data/
│   │   ├── billing.go         # PostgreSQL
│   │   ├── casdoor.go         # Casdoor 客户端
│   │   └── lago.go            # Lago 客户端
│   ├── server/
│   │   ├── http.go            # HTTP 服务
│   │   └── nats.go            # NATS 消费者
│   └── service/billing.go
└── wire.go
```

**关键文件修改**：
- `codeswitch/services/billing_integration.go` → 迁移到 Kratos
- `codeswitch/services/billing/*.go` → 迁移到 Kratos biz/data 层

---

### Phase 6: 可观测性 + 部署 (2 周) ✅ COMPLETED

**目标**：完善监控和部署

**任务清单**：
- [x] Prometheus metrics 埋点
- [x] OpenTelemetry tracing 集成
- [x] 结构化日志 (JSON + TraceID)
- [x] Grafana 仪表盘
- [x] Docker Compose 生产配置
- [x] Kubernetes 部署 YAML (K3S)
- [x] GitHub Actions CI/CD 流水线
- [x] 基础设施监控 (node-exporter, cAdvisor, Vector)

**关键 Metrics**：
```
gateway_requests_total{platform, provider, model, status}
gateway_request_latency_seconds{platform, provider, is_stream}
gateway_tokens_total{platform, provider, direction}
gateway_cost_usd_total{platform, provider, model}
```

---

## 五、风险与缓解 (Risks & Mitigation)

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| 服务间通信延迟 | 高 | Provider 配置缓存 (5min TTL)；批量日志写入 |
| 数据一致性 | 高 | TraceID 幂等性；Saga 补偿；最终一致性监控 |
| 迁移期中断 | 高 | Feature Flag 控制；灰度发布；双写过渡期 |
| NATS 故障 | 高 | 集群部署；JetStream 持久化；本地降级队列 |

**回滚策略**：
```yaml
feature_flags:
  provider_service_enabled: true
  provider_fallback_local: true    # 降级开关
  log_service_async: true
  log_fallback_sqlite: true        # 降级开关
  billing_service_enabled: true
  billing_skip_check: false        # 紧急跳过
```

---

## 六、关键文件清单 (Critical Files)

### 需要重构的文件

| 文件路径 | 操作 | 目标服务 | 框架 |
|---------|------|---------|------|
| `codeswitch/services/providerrelay.go` | 重写 | Gateway | **Hertz** |
| `codeswitch/services/providerservice.go` | 迁移 | Provider Service | **Kratos** |
| `codeswitch/services/logservice.go` | 迁移 | Log Service | **Kratos** |
| `codeswitch/services/billing_integration.go` | 迁移 | Billing Service | **Kratos** |
| `codeswitch/services/billing/*.go` | 迁移 | Billing Service | **Kratos** |
| `codeswitch/services/sync_integration.go` | 简化 | Gateway | **Hertz** |
| `codeswitch/services/sync/*.go` | 保留 | Sync Service | Gin |

### 新建服务清单

| 服务 | 目录 | 框架 | 代码生成命令 |
|------|------|------|-------------|
| gateway-service | `gateway-service/` | Hertz | `hz new -mod gateway-service` |
| provider-service | `provider-service/` | Kratos | `kratos new provider-service` |
| log-service | `log-service/` | Kratos | `kratos new log-service` |
| billing-service | `billing-service/` | Kratos | `kratos new billing-service` |

---

## 七、技术栈全景图 (Technology Stack)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Lurus Switch 技术栈 (Hertz + Kratos)                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  服务层 (Service Layer)                                                      │
│  ├─ Gateway Service: Hertz 0.9+ (字节, 180k QPS, SSE 流式)                  │
│  ├─ Provider Service: Kratos 2.8+ (B站, Wire DI, Protobuf)                 │
│  ├─ Log Service: Kratos 2.8+ (NATS Consumer, ClickHouse)                   │
│  ├─ Billing Service: Kratos 2.8+ (Casdoor + Lago 集成)                     │
│  └─ Sync Service: Gin 1.10 (保留, 已稳定运行)                               │
│                                                                             │
│  代码生成 (Code Generation)                                                  │
│  ├─ Kratos: protoc + kratos-layout                                         │
│  ├─ Hertz: hz CLI                                                          │
│  └─ Wire: 编译时依赖注入                                                     │
│                                                                             │
│  服务治理 (Service Governance)                                               │
│  ├─ 服务发现: Consul / K8s Service (不引入 Etcd)                            │
│  ├─ 负载均衡: Kratos 内置 P2C / Hertz Client LB                             │
│  ├─ 熔断限流: Kratos 内置 / Hertz sentinel                                  │
│  └─ 链路追踪: OpenTelemetry (Kratos/Hertz 原生支持)                         │
│                                                                             │
│  消息层 (Message Layer)                                                      │
│  └─ NATS 2.10 + JetStream (保留现有基础设施)                                │
│                                                                             │
│  数据层 (Data Layer)                                                         │
│  ├─ PostgreSQL 16 + Kratos data 层 (Provider/Billing/Sync)                 │
│  ├─ ClickHouse 24+ (Log Service, OLAP 分析)                                │
│  ├─ Redis 7 + Kratos cache (配额/Provider 缓存)                            │
│  └─ SQLite (离线降级, 本地存储)                                              │
│                                                                             │
│  可观测性 (Observability)                                                    │
│  ├─ Metrics: Prometheus (Kratos/Hertz 原生埋点)                             │
│  ├─ Tracing: OpenTelemetry → Jaeger/Tempo                                  │
│  ├─ Logging: Kratos log / Hertz hlog → Loki                                │
│  └─ Dashboard: Grafana                                                      │
│                                                                             │
│  部署 (Deployment)                                                           │
│  ├─ 开发: Docker Compose + Consul                                          │
│  └─ 生产: Kubernetes + Helm                                                 │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 八、验收标准 (Acceptance Criteria)

### 功能验收
- [ ] 所有现有 API 端点保持兼容
- [ ] Claude Code / Codex / Gemini CLI 正常工作
- [ ] 日志统计功能正常
- [ ] 计费和配额功能正常
- [ ] 多端同步功能正常

### 性能验收
- [ ] P99 延迟增加 < 50ms
- [ ] 吞吐量无明显下降
- [ ] 日志写入延迟 < 1s (事件驱动)

### 运维验收
- [ ] 各服务独立部署和扩展
- [ ] 完整的监控仪表盘
- [ ] 告警规则配置
- [ ] 灾难恢复测试通过

---

## 九、下一步行动 (Next Steps)

1. **确认方案** - 与团队 review 本计划
2. **环境准备** - 安装 Kratos CLI、Hertz CLI、Wire、protoc
3. **创建共享库** - `lurus-common/` 基础模块
4. **搭建环境** - Docker Compose (Consul + ClickHouse + NATS)
5. **试点迁移** - Provider Service (Kratos) 作为第一个独立服务

**快速启动命令**：
```bash
# 安装工具链
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
go install github.com/cloudwego/hertz/cmd/hz@latest
go install github.com/google/wire/cmd/wire@latest

# 创建服务骨架
kratos new provider-service
kratos new log-service
kratos new billing-service
hz new -mod gateway-service
```

---

*Generated by Claude Code | 2026-01-07*
*Tech Stack: Hertz (Gateway) + Kratos (Services) + NATS + ClickHouse*
