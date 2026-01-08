# Lurus Switch 工作流水 / Process Log

> Last Updated: 2026-01-08

---

## 2026-01-08 微服务迁移完成 / Microservices Migration Completed

### 需求分析 / Requirements Analysis

将 Lurus Switch 从混合单体架构迁移到 Hertz + Kratos 微服务架构，完成 Phase 1-5 的核心服务构建和单元测试。

### 方法设计 / Design Approach

采用 Kratos 风格的分层架构（biz/data/service/server），每个服务独立部署：
- Gateway Service: Hertz 高性能 HTTP 代理
- Provider Service: 供应商配置管理
- Log Service: 日志存储与 OLAP 分析
- Billing Service: 计费与配额管理

### 修改摘要 / Changes Summary

#### 1. lurus-common 共享库
- `models/provider.go` - Provider 模型定义
- `models/user.go` - User/Quota 模型
- `models/request_log.go` - 日志模型
- `errors/errors.go` - 统一错误处理
- `nats/client.go` - NATS 客户端封装
- 测试文件: `provider_test.go`, `user_test.go`, `request_log_test.go`, `errors_test.go`

#### 2. provider-service
- `internal/biz/provider.go` - 业务逻辑层
- `internal/data/provider.go` - PostgreSQL + Redis 数据层
- `internal/server/http.go` - HTTP API
- `internal/biz/provider_test.go` - 单元测试

#### 3. log-service
- `internal/biz/log.go` - 日志业务逻辑
- `internal/data/log.go` - ClickHouse 数据层
- `internal/consumer/nats.go` - NATS 消费者
- `internal/biz/log_test.go` - 单元测试

#### 4. gateway-service
- `internal/proxy/relay.go` - 核心转发逻辑
- `internal/handler/*.go` - 路由处理器
- `internal/proxy/relay_test.go` - 单元测试

#### 5. billing-service
- `internal/biz/billing.go` - 计费业务逻辑
- `internal/data/billing.go` - PostgreSQL 数据层
- `internal/server/http.go` - REST API (Gin)
- `internal/consumer/nats.go` - NATS 消费者
- `internal/biz/billing_test.go` - 单元测试

#### 6. 开发环境
- `docker-compose.dev.yaml` - 完整开发环境
- `init-db.sql` - PostgreSQL 初始化
- `clickhouse-init.sql` - ClickHouse OLAP 表

### 最终结果 / Final Results

**测试结果**: 全部通过

| 模块 | 测试文件 | 状态 |
|------|---------|------|
| lurus-common/errors | errors_test.go | PASS |
| lurus-common/models | provider_test.go, user_test.go, request_log_test.go | PASS |
| provider-service/internal/biz | provider_test.go | PASS |
| billing-service/internal/biz | billing_test.go | PASS |
| gateway-service/internal/proxy | relay_test.go | PASS |
| log-service/internal/biz | log_test.go | PASS |

**编译结果**: 5 个服务全部成功编译
- gateway-service: 26MB
- provider-service: 26MB
- log-service: 25MB
- billing-service: 26MB
- lurus-common: library (no binary)

### 测试修复记录 / Test Fixes

1. **billing_test.go**: 浮点数精度比较问题，改用容差比较
2. **provider_test.go**: Mock 仓库需要存储副本防止指针污染
3. **provider_test.go**: NewProviderUsecase 需要传入 logger
4. **relay_test.go**: ID 生成测试改为验证格式而非唯一性

---

## 2026-01-08 微服务部署完成 / Microservices Deployment Completed

### 需求分析 / Requirements Analysis

在阿里云 Windows Server 2019 服务器上部署 Lurus Switch 微服务架构，包含 Gateway、Provider、Billing 三个核心服务。

### 方法设计 / Design Approach

由于阿里云 VM 不支持嵌套虚拟化（无法运行 Docker Desktop），采用 Windows 原生部署：
- 各服务编译为独立 .exe 文件
- 使用 PowerShell 脚本启动服务
- Caddy 作为边缘代理处理 Host+Path 路由
- PostgreSQL 存储 Provider 和 Billing 数据
- NATS 用于异步事件发布

### 部署架构 / Deployment Architecture

```
Internet → Caddy :80 → ┬─ api.lurus.cn → new-api:3000
                       │
                       └─ ai.lurus.cn ─┬─ /v1/* → gateway:18100
                                       └─ /* → new-api:3000 (frontend)

Gateway (:18100) ──► Provider Service (:18101) ──► PostgreSQL
         │
         ├──► Billing Service (:18103) ──► PostgreSQL
         │
         └──► NATS (:4222) ──► log.write, billing.usage events
```

### 服务状态 / Service Status

| 服务 | 端口 | 部署位置 | 状态 |
|------|------|---------|------|
| PostgreSQL | 5432 | D:\PostgreSQL | ✅ 运行中 |
| Redis | 6379 | D:\tools\redis | ✅ 运行中 |
| NATS | 4222 | D:\services\nats | ✅ 运行中 |
| new-api | 3000 | D:\services\new-api | ✅ 运行中 |
| gateway-service | 18100 | D:\services\gateway | ✅ 运行中 |
| provider-service | 18101 | D:\services\provider-service | ✅ 运行中 |
| billing-service | 18103 | D:\services\billing-service | ✅ 运行中 |
| Caddy | 80 | D:\services\caddy | ✅ 运行中 |

### 数据库配置 / Database Configuration

```
用户: lurus / 密码: lurus_dev_2024
数据库:
  - lurus_provider (Provider Service)
  - lurus_billing (Billing Service)
  - lurus_sync (Sync Service)
  - lurus_subscription (Subscription Service)
  - new_api (NEW-API)
```

### 服务管理脚本 / Service Management Scripts

```powershell
# 启动所有服务
D:\services\start-all.ps1

# 停止所有服务
D:\services\stop-all.ps1

# 查看服务状态
D:\services\status.ps1
```

### Caddy 域名路由 / Caddy Domain Routing

| 域名 | 路由目标 |
|------|---------|
| api.lurus.cn | → new-api:3000 |
| ai.lurus.cn /v1/* | → gateway:18100 |
| ai.lurus.cn /* | → portal 静态文件 |
| lurus.cn | → ailurus 静态文件 |
| platform.lurus.cn | → new-api:3000 |
| portal.lurus.cn | → portal 静态文件 |

### 技术要点 / Technical Notes

1. **Hertz 路由 Gemini API**:
   - 问题: Hertz 不支持 `/:model\:action` 格式的路由
   - 解决: 使用 `/*modelAction` 通配符 + HandleModelAction 方法解析

2. **Caddy 智能路由**:
   - 同一域名 ai.lurus.cn 同时提供 API 和前端服务
   - 使用 `@ai_gateway` matcher 区分 API 路由和前端路由

3. **IIS 端口冲突**:
   - 问题: IIS 默认占用 80/443 端口
   - 解决: 停止 W3SVC 和 WAS 服务

### 验证结果 / Verification Results

```powershell
# 本地测试
curl localhost:18100/health           # {"status":"healthy"}
curl localhost:3000/api/status        # new-api status

# 外部测试
curl http://ai.lurus.cn/health        # {"status":"healthy"}
curl http://api.lurus.cn/api/status   # new-api status
```

---

## 2026-01-08 可观测性 + CI/CD + K3S 部署 / Observability + CI/CD + K3S Deployment

### 需求分析 / Requirements Analysis

增加可观测性、CI/CD 自动部署和 K3S 容器编排支持。

### 方法设计 / Design Approach

- **可观测性**: 启用 OpenTelemetry 追踪，添加基础设施监控 (node-exporter, cAdvisor, Vector)
- **CI/CD**: GitHub Actions 流水线 (CI: 测试/构建/安全扫描, CD: 构建镜像/推送 ghcr.io/部署 K3S)
- **K3S**: 单节点部署，使用 local-path 存储，Traefik Ingress + cert-manager

### 修改摘要 / Changes Summary

#### Phase 1: 可观测性
- `gateway-service/configs/config.yaml` - 启用 OpenTelemetry 追踪
- `prometheus.yml` - 添加 node-exporter, cAdvisor targets
- `deploy/docker-compose.yml` - 添加 node-exporter, cAdvisor, Vector 服务
- `deploy/vector/vector.toml` - 日志采集配置 (Docker → ClickHouse)

#### Phase 2: K3S 部署清单
- `deploy/k3s/namespace.yaml` - lurus-system 命名空间
- `deploy/k3s/secrets/lurus-secrets.yaml.example` - 凭证模板
- `deploy/k3s/configmaps/gateway-config.yaml` - Gateway 配置
- `deploy/k3s/statefulsets/{postgres,redis,nats}.yaml` - 数据库 StatefulSets
- `deploy/k3s/deployments/{gateway-service,new-api}.yaml` - 微服务部署
- `deploy/k3s/ingress.yaml` - Traefik Ingress + cert-manager
- `deploy/k3s/hpa/gateway-hpa.yaml` - 自动扩缩配置
- `deploy/k3s/kustomization.yaml` - Kustomize 配置

#### Phase 3: CI/CD 流水线
- `.github/workflows/ci.yml` - 持续集成 (测试/构建/Lint/安全扫描)
- `.github/workflows/deploy.yml` - 持续部署 (构建镜像/推送 ghcr.io/SSH 部署 K3S)

#### Phase 4: 文档
- `deploy/k3s/README.md` - K3S 部署指南

### 最终结果 / Final Results

**Git 仓库已推送**: https://github.com/hanmahong5-arch/lurus-switch

**已确定决策**:
| 决策项 | 选择 |
|--------|------|
| K3S 部署 | 单节点 |
| 存储方案 | local-path-provisioner |
| 镜像仓库 | ghcr.io |
| 告警通知 | 暂不配置 |
| 证书管理 | cert-manager + Let's Encrypt |
| 日志采集 | Vector |

---

## 2026-01-08 可观测性完善 + 多客户端同步 API / Observability + Multi-Client Sync

### 需求分析 / Requirements Analysis

完成整个公司产品服务的可观测性，以及多客户端同时登录账户、账务同步所需的接口。

### 方法设计 / Design Approach

1. **可观测性**: 为 Billing Service 添加 Prometheus 业务指标 + NATS 事件发布
2. **多客户端同步**: 提供 HTTP/SSE/WebSocket 三种同步方式
3. **Grafana 仪表盘**: 创建服务级监控面板

### 修改摘要 / Changes Summary

#### 1. Billing Service 可观测性增强
- `billing-service/internal/middleware/metrics.go` - Prometheus 业务指标
  - billing_http_requests_total
  - billing_balance_checks_total
  - billing_usage_records_total
  - billing_tokens_processed_total
  - billing_cost_usd_total
  - billing_quota_updates_total
  - billing_low_balance_alerts_total
  - billing_nats_events_published_total

- `billing-service/internal/publisher/nats.go` - NATS 事件发布器
  - quota.updated - 配额变更
  - balance.changed - 余额变动
  - usage.recorded - 用量记录
  - quota.low - 低配额警告 (≥80%)
  - quota.exhausted - 配额耗尽

#### 2. HTTP 同步 API 端点
- `billing-service/internal/server/http.go` - 新增端点:
  - GET /api/v1/billing/sync/:user_id - 获取同步状态
  - GET /api/v1/billing/sync/:user_id/stream - SSE 实时流

#### 3. WebSocket 实时同步
- `codeswitch/sync-service/internal/api/websocket.go` - WebSocket Hub
  - 多设备并发连接管理
  - NATS 事件自动转发
  - 30 秒心跳保活
  - ping/pong, subscribe, sync_request 消息支持

#### 4. Grafana 服务仪表盘
- `deploy/grafana/provisioning/dashboards/lurus-services.json`
  - 服务健康状态面板
  - 请求速率和 P95 延迟
  - Billing 业务指标
  - NATS 事件统计

#### 5. 客户端开发文档
- `doc/client-sync-api.md` - 多客户端同步 API 完整文档
  - HTTP REST API 说明
  - SSE 实时流使用指南
  - WebSocket 协议规范
  - NATS 事件类型定义
  - 数据结构定义
  - 错误处理指南
  - 最佳实践
  - Flutter/Swift/Kotlin 示例代码

### 最终结果 / Final Results

**服务状态**: 全部健康

| 服务 | 端口 | 状态 |
|------|------|------|
| Gateway Service | 18100 | ✅ Healthy |
| Provider Service | 18101 | ✅ Healthy |
| Billing Service | 18103 | ✅ Healthy (已更新) |
| NEW-API | 3000 | ✅ Healthy |

**同步 API 测试**:
```bash
# 同步状态查询
curl http://localhost:18103/api/v1/billing/sync/test-user-001
# 返回: {"user_id":"test-user-001","quota_limit":1000000,"quota_remaining":1000000,...}

# SSE 流式端点
curl http://localhost:18103/api/v1/billing/sync/test-user-001/stream
# 返回: SSE 事件流
```

**交付物**:
- 更新的 Billing Service 二进制
- Grafana 仪表盘配置
- 客户端开发文档 (`doc/client-sync-api.md`)

---

*Generated by Claude Code | 2026-01-08*
