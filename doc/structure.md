# Lurus Switch Architecture Document
# Lurus Switch 架构设计文档

> **Version**: v2.0 | **Created**: 2026-01-12
> **Status**: Production Ready | **Framework**: Hertz + Kratos + K3s

---

## 1. Infrastructure Overview / 基础设施总览

### 1.1 Server Mapping / 服务器角色映射

Based on the actual hardware resources, we define the following node roles:
根据实际硬件资源，定义以下节点角色：

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         LURUS SWITCH INFRASTRUCTURE                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    BRAIN NODE (大脑节点)                               │  │
│  │                    16C32G | 43.226.46.164                             │  │
│  │                    50Mbps Unlimited Traffic                           │  │
│  │                                                                       │  │
│  │    Role: K3s Master + API Gateway + NATS Server                      │  │
│  │    ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │  │
│  │    │ K3s Master  │  │   Gateway   │  │    NATS     │                 │  │
│  │    │  (Control)  │  │   :18100    │  │  :4222/8222 │                 │  │
│  │    └─────────────┘  └─────────────┘  └─────────────┘                 │  │
│  │    ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │  │
│  │    │  Provider   │  │   Billing   │  │   NEW-API   │                 │  │
│  │    │   :18101    │  │   :18103    │  │    :3000    │                 │  │
│  │    └─────────────┘  └─────────────┘  └─────────────┘                 │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                    │                                        │
│                              Tailscale VPN                                  │
│                            (100.x.x.x/24)                                   │
│                                    │                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    HEART NODE (心脏节点)                               │  │
│  │                    4C8G | 115.190.239.146                             │  │
│  │                    Fixed IP | 5Mbps                                   │  │
│  │                                                                       │  │
│  │    Role: K3s Worker + PostgreSQL Primary + Auth Center               │  │
│  │    ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │  │
│  │    │ PostgreSQL  │  │    Redis    │  │  Casdoor    │                 │  │
│  │    │    :5432    │  │    :6379    │  │   (Auth)    │                 │  │
│  │    └─────────────┘  └─────────────┘  └─────────────┘                 │  │
│  │    ┌─────────────┐                                                   │  │
│  │    │   K3s       │  (Worker Node for stateful services)              │  │
│  │    │  Worker     │                                                   │  │
│  │    └─────────────┘                                                   │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                    │                                        │
│                              Tailscale VPN                                  │
│                                    │                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    MONITOR NODE (监控节点)                             │  │
│  │                    2C2G | 82.156.7.13                                 │  │
│  │                    Lightweight | 1Mbps                                │  │
│  │                                                                       │  │
│  │    Role: Prometheus + Grafana + Alertmanager                         │  │
│  │    ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │  │
│  │    │ Prometheus  │  │   Grafana   │  │ Alertmgr    │                 │  │
│  │    │   :9090     │  │    :3000    │  │   :9093     │                 │  │
│  │    └─────────────┘  └─────────────┘  └─────────────┘                 │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                    │                                        │
│                              Tailscale VPN                                  │
│                                    │                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    LAB NODE (实验室节点)                               │  │
│  │                    4C8G Local | 192.168.88.252                        │  │
│  │                    No Public IP | 200Mbps LAN                         │  │
│  │                                                                       │  │
│  │    Role: Log Service + ClickHouse + Agent Dev Environment            │  │
│  │    ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │  │
│  │    │ ClickHouse  │  │ Log Service │  │   Jaeger    │                 │  │
│  │    │ :8123/9000  │  │   :18102    │  │   :16686    │                 │  │
│  │    └─────────────┘  └─────────────┘  └─────────────┘                 │  │
│  │    ┌─────────────┐                                                   │  │
│  │    │   MinIO     │  (F: Drive, 2TB SSD Backup)                       │  │
│  │    │    :9000    │                                                   │  │
│  │    └─────────────┘                                                   │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 Network Topology / 网络拓扑

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          NETWORK TOPOLOGY                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Public Internet                                                           │
│        │                                                                    │
│        ▼ HTTPS (:443)                                                       │
│   ┌─────────────────┐                                                       │
│   │  Cloudflare CDN │  ← TLS Termination, DDoS Protection                  │
│   │   (Optional)    │                                                       │
│   └────────┬────────┘                                                       │
│            │                                                                │
│            ▼ HTTP (:80/443)                                                 │
│   ┌─────────────────┐                                                       │
│   │   Brain Node    │  43.226.46.164 (Public IP)                           │
│   │  K3s Ingress    │                                                       │
│   └────────┬────────┘                                                       │
│            │                                                                │
│   ╔════════╧════════════════════════════════════════════════════════╗       │
│   ║              TAILSCALE MESH VPN (100.x.x.x/24)                  ║       │
│   ║                                                                 ║       │
│   ║   Brain: 100.x.x.1    Heart: 100.x.x.2    Monitor: 100.x.x.3   ║       │
│   ║                        Lab: 100.x.x.4                           ║       │
│   ║                                                                 ║       │
│   ╚═════════════════════════════════════════════════════════════════╝       │
│                                                                             │
│   Internal Communication:                                                   │
│   ├─ NATS: 100.x.x.1:4222 (Message Bus)                                    │
│   ├─ PostgreSQL: 100.x.x.2:5432 (via Tailscale only)                       │
│   ├─ Redis: 100.x.x.2:6379 (via Tailscale only)                            │
│   └─ ClickHouse: 100.x.x.4:8123 (via Tailscale only)                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Microservice Architecture / 微服务架构

### 2.1 Service Overview / 服务总览

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       MICROSERVICE ARCHITECTURE                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   CLIENT LAYER (客户端层)                                                    │
│   ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐          │
│   │   Claude    │ │   Codex     │ │  Gemini     │ │  Admin      │          │
│   │    Code     │ │   CLI       │ │    CLI      │ │  Console    │          │
│   └──────┬──────┘ └──────┬──────┘ └──────┬──────┘ └──────┬──────┘          │
│          │               │               │               │                  │
│          └───────────────┴───────────────┴───────────────┘                  │
│                                   │                                         │
│                          HTTP / WebSocket                                   │
│                                   │                                         │
│   ╔═══════════════════════════════╧═══════════════════════════════════╗     │
│   ║                    GATEWAY LAYER (网关层)                          ║     │
│   ║                                                                    ║     │
│   ║   ┌─────────────────────────────────────────────────────────┐     ║     │
│   ║   │              Gateway Service (Hertz)                     │     ║     │
│   ║   │              Port: 18100                                 │     ║     │
│   ║   │                                                          │     ║     │
│   ║   │   Routes:                                                │     ║     │
│   ║   │   ├─ POST /v1/messages         → Claude API             │     ║     │
│   ║   │   ├─ POST /responses           → Codex API              │     ║     │
│   ║   │   ├─ POST /v1/chat/completions → OpenAI Compatible      │     ║     │
│   ║   │   └─ POST /v1beta/models/*     → Gemini API             │     ║     │
│   ║   │                                                          │     ║     │
│   ║   │   Features:                                              │     ║     │
│   ║   │   ├─ SSE Streaming (native Hertz StreamBody)            │     ║     │
│   ║   │   ├─ Provider Matching & Failover                       │     ║     │
│   ║   │   ├─ Request/Response Logging (async NATS)              │     ║     │
│   ║   │   └─ Balance Checking (via Billing Service)             │     ║     │
│   ║   └─────────────────────────────────────────────────────────┘     ║     │
│   ╚═══════════════════════════════════════════════════════════════════╝     │
│                                   │                                         │
│                    HTTP (sync) / NATS (async)                               │
│                                   │                                         │
│   ╔═══════════════════════════════╧═══════════════════════════════════╗     │
│   ║                    SERVICE LAYER (服务层)                          ║     │
│   ║                                                                    ║     │
│   ║   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐               ║     │
│   ║   │  Provider   │  │  Billing    │  │    Log      │               ║     │
│   ║   │  Service    │  │  Service    │  │  Service    │               ║     │
│   ║   │  (Kratos)   │  │  (Kratos)   │  │  (Kratos)   │               ║     │
│   ║   │  :18101     │  │  :18103     │  │  :18102     │               ║     │
│   ║   │             │  │             │  │             │               ║     │
│   ║   │ - Config    │  │ - Auth      │  │ - OLAP      │               ║     │
│   ║   │ - Cache     │  │ - Quota     │  │ - Stats     │               ║     │
│   ║   │ - Health    │  │ - Payment   │  │ - Cost      │               ║     │
│   ║   └──────┬──────┘  └──────┬──────┘  └──────┬──────┘               ║     │
│   ╚══════════╧════════════════╧════════════════╧══════════════════════╝     │
│              │                │                │                             │
│              │                │                │                             │
│   ╔══════════╧════════════════╧════════════════╧══════════════════════╗     │
│   ║                    DATA LAYER (数据层)                             ║     │
│   ║                                                                    ║     │
│   ║   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐               ║     │
│   ║   │ PostgreSQL  │  │   Redis     │  │ ClickHouse  │               ║     │
│   ║   │  (Primary)  │  │  (Cache)    │  │   (OLAP)    │               ║     │
│   ║   │  :5432      │  │  :6379      │  │ :8123/9000  │               ║     │
│   ║   │             │  │             │  │             │               ║     │
│   ║   │ - providers │  │ - sessions  │  │ - logs      │               ║     │
│   ║   │ - users     │  │ - quota     │  │ - stats     │               ║     │
│   ║   │ - wallets   │  │ - rate_lim  │  │ - cost      │               ║     │
│   ║   └─────────────┘  └─────────────┘  └─────────────┘               ║     │
│   ╚═══════════════════════════════════════════════════════════════════╝     │
│                                                                             │
│   ╔═══════════════════════════════════════════════════════════════════╗     │
│   ║                    MESSAGE BUS (消息总线)                          ║     │
│   ║                                                                    ║     │
│   ║   ┌─────────────────────────────────────────────────────────┐     ║     │
│   ║   │              NATS JetStream                              │     ║     │
│   ║   │              Port: 4222 (client) / 8222 (ws)             │     ║     │
│   ║   │                                                          │     ║     │
│   ║   │   Streams:                                               │     ║     │
│   ║   │   ├─ LLM_EVENTS    (llm.>) - 7 days retention           │     ║     │
│   ║   │   ├─ LOG_EVENTS    (log.write) - 1 day retention        │     ║     │
│   ║   │   └─ BILLING_EVENTS(billing.>) - 30 days retention      │     ║     │
│   ║   └─────────────────────────────────────────────────────────┘     ║     │
│   ╚═══════════════════════════════════════════════════════════════════╝     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Service Responsibilities / 服务职责

| Service | Framework | Port | Responsibilities | Data Ownership |
|---------|-----------|------|------------------|----------------|
| **Gateway** | Hertz | :18100 | HTTP Proxy, SSE Streaming, Model Matching, Failover | Stateless |
| **Provider** | Kratos | :18101 | Provider CRUD, Model Validation, Health Check | `providers`, `model_pricing` |
| **Billing** | Kratos | :18103 | Auth, Quota, Payment, Subscription | `users`, `wallets`, `transactions` |
| **Log** | Kratos | :18102 | Log Storage, Statistics, Cost Analysis | `request_log`, `stats` |
| **Sync** | Gin | :8081 | Session Sync, Online Status, Admin API | `sessions`, `messages` |
| **NEW-API** | Go | :3000 | LLM Unified Gateway, 40+ Providers | External |

---

## 3. Directory Structure / 目录结构

```
lurus-switch/
├── api/                          # Protobuf API definitions
│   └── proto/
│       ├── provider/v1/
│       ├── billing/v1/
│       └── log/v1/
│
├── gateway-service/              # Hertz - API Gateway
│   ├── cmd/gateway/main.go
│   ├── internal/
│   │   ├── handler/              # HTTP handlers (claude, codex, gemini)
│   │   ├── middleware/           # CORS, Auth, Metrics, Tracing
│   │   ├── proxy/                # Core relay logic
│   │   └── client/               # Provider/Billing HTTP clients
│   ├── pkg/nats/                 # NATS publisher
│   └── configs/config.yaml
│
├── provider-service/             # Kratos - Provider Management
│   ├── cmd/provider/main.go
│   ├── internal/
│   │   ├── biz/provider.go       # Business logic
│   │   ├── data/provider.go      # PostgreSQL + Redis
│   │   ├── server/http.go        # HTTP server
│   │   └── service/provider.go   # Service implementation
│   └── configs/config.yaml
│
├── billing-service/              # Kratos - Billing & Auth
│   ├── cmd/billing/main.go
│   ├── internal/
│   │   ├── biz/                  # Billing, Auth logic
│   │   ├── data/                 # PostgreSQL, Casdoor, Lago
│   │   ├── server/               # HTTP + NATS consumer
│   │   └── service/
│   └── configs/config.yaml
│
├── log-service/                  # Kratos - Log Analytics
│   ├── cmd/log/main.go
│   ├── internal/
│   │   ├── biz/log.go            # Log processing
│   │   ├── data/clickhouse.go    # ClickHouse OLAP
│   │   ├── consumer/nats.go      # NATS consumer
│   │   └── server/http.go        # Query API
│   └── configs/config.yaml
│
├── subscription-service/         # Kratos - Subscription Management
│   └── ...
│
├── lurus-common/                 # Shared libraries
│   ├── models/                   # Common data models
│   ├── nats/                     # NATS client wrapper
│   ├── observability/            # Tracing, Metrics, Logging
│   └── errors/                   # Error handling
│
├── lurus-portal/                 # Vue 3 Web Portal
│   └── ...
│
├── deploy/                       # Deployment configurations
│   ├── docker-compose.yml        # Development
│   ├── docker-compose.production.yml
│   ├── k3s/                      # Kubernetes manifests
│   │   ├── namespace.yaml
│   │   ├── deployments/
│   │   ├── statefulsets/
│   │   ├── configmaps/
│   │   └── ingress.yaml
│   ├── prometheus/
│   ├── grafana/
│   └── alertmanager/
│
├── scripts/                      # Utility scripts
│   ├── db_sync.go
│   └── monitor_traffic.sh
│
└── doc/                          # Documentation
    ├── structure.md              # THIS FILE
    ├── develop-guide.md          # Development guide
    ├── plan.md                   # Roadmap
    └── process.md                # Work log
```

---

## 4. Data Flow / 数据流

### 4.1 Request Flow (Online Mode)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         REQUEST DATA FLOW                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   1. Client Request                                                         │
│      │                                                                      │
│      ▼                                                                      │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │  Gateway Service (:18100)                                           │   │
│   │  ├─ 1.1 Extract API Key                                            │   │
│   │  ├─ 1.2 Match Platform (Claude/Codex/Gemini)                       │   │
│   │  └─ 1.3 Extract Model Name                                         │   │
│   └────────────────────────────────┬────────────────────────────────────┘   │
│                                    │                                        │
│      2. Balance Check (HTTP)       │      3. Provider Lookup (HTTP)         │
│   ┌────────────────────────────────┼────────────────────────────────────┐   │
│   │                                │                                    │   │
│   ▼                                ▼                                    │   │
│   ┌─────────────┐              ┌─────────────┐                          │   │
│   │  Billing    │              │  Provider   │                          │   │
│   │  Service    │              │  Service    │                          │   │
│   │  :18103     │              │  :18101     │                          │   │
│   │             │              │             │                          │   │
│   │ CheckQuota()│              │ MatchModel()│ ← Redis Cache (5min TTL) │   │
│   └─────────────┘              └─────────────┘                          │   │
│         │                            │                                  │   │
│         │ OK                         │ Provider Config                  │   │
│         └─────────────┬──────────────┘                                  │   │
│                       │                                                 │   │
│                       ▼                                                 │   │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │  Gateway Service                                                    │   │
│   │  4. Forward Request to AI Provider                                 │   │
│   │     ├─ OpenAI / Anthropic / Google / DeepSeek / ...               │   │
│   │     └─ Or via NEW-API (:3000) for unified access                   │   │
│   └────────────────────────────────┬────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │  5. Stream Response to Client (SSE)                                │   │
│   │     │                                                              │   │
│   │     └─► Async: Publish to NATS                                     │   │
│   └────────────────────────────────┬────────────────────────────────────┘   │
│                                    │                                        │
│   ┌────────────────────────────────┼────────────────────────────────────┐   │
│   │                                │                                    │   │
│   ▼                                ▼                                    │   │
│   ┌─────────────┐              ┌─────────────┐                          │   │
│   │    NATS     │              │    NATS     │                          │   │
│   │ log.write   │              │ billing.    │                          │   │
│   │             │              │ usage.*     │                          │   │
│   └──────┬──────┘              └──────┬──────┘                          │   │
│          │                            │                                 │   │
│          ▼                            ▼                                 │   │
│   ┌─────────────┐              ┌─────────────┐                          │   │
│   │ Log Service │              │  Billing    │                          │   │
│   │ → ClickHouse│              │  Service    │                          │   │
│   └─────────────┘              └─────────────┘                          │   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.2 Offline Fallback Mode / 离线降级模式

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         OFFLINE FALLBACK FLOW                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   When services are unavailable:                                            │
│                                                                             │
│   Gateway Service                                                           │
│   ├─ Provider Lookup Failed → Read from local JSON file                    │
│   │   (~/.code-switch/claude-code.json, codex.json)                        │
│   │                                                                         │
│   ├─ Billing Service Failed → Skip balance check (Feature Flag)            │
│   │   (billing_skip_check: true)                                           │
│   │                                                                         │
│   ├─ NATS Unavailable → Write to local SQLite queue                        │
│   │   (~/.code-switch/app.db)                                              │
│   │                                                                         │
│   └─ Reconnect → Sync local queue to services                              │
│                                                                             │
│   Feature Flags:                                                            │
│   ┌───────────────────────────────────────────────────────────────────┐     │
│   │  provider_fallback_local: true    # Use local JSON if service down│     │
│   │  billing_skip_check: false        # Emergency skip (admin only)   │     │
│   │  log_fallback_sqlite: true        # Local SQLite if NATS down     │     │
│   │  sync_offline_queue: true         # Queue events when offline     │     │
│   └───────────────────────────────────────────────────────────────────┘     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 5. K3s Cluster Design / K3s 集群设计

### 5.1 Node Labels & Taints

```yaml
# Brain Node (43.226.46.164)
labels:
  lurus.cn/role: gateway
  lurus.cn/bandwidth: high
  node.kubernetes.io/instance-type: 16c32g
taints: []

# Heart Node (115.190.239.146)
labels:
  lurus.cn/role: database
  lurus.cn/stateful: "true"
taints:
  - key: "lurus.cn/database"
    value: "postgres"
    effect: "NoSchedule"

# Lab Node (192.168.88.252)
labels:
  lurus.cn/role: analytics
  lurus.cn/local: "true"
taints:
  - key: "lurus.cn/local"
    value: "true"
    effect: "PreferNoSchedule"
```

### 5.2 Pod Distribution / Pod 分布

| Node | Services | Resources |
|------|----------|-----------|
| **Brain** | Gateway (x2), Provider, Billing, NATS, NEW-API | 16C32G |
| **Heart** | PostgreSQL, Redis, Casdoor | 4C8G |
| **Monitor** | Prometheus, Grafana, Alertmanager | 2C2G |
| **Lab** | Log Service, ClickHouse, Jaeger, MinIO | 4C8G |

---

## 6. Technology Stack / 技术栈

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         TECHNOLOGY STACK                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Service Frameworks:                                                       │
│   ├─ Gateway Service: Hertz 0.9+ (ByteDance, 180k QPS, native SSE)         │
│   ├─ Business Services: Kratos 2.8+ (Bilibili, Wire DI, Protobuf)          │
│   └─ Sync Service: Gin 1.10 (preserved, stable)                            │
│                                                                             │
│   Data Layer:                                                               │
│   ├─ PostgreSQL 16 (Provider, Billing, Sync data)                          │
│   ├─ ClickHouse 24+ (Log OLAP, analytics)                                  │
│   ├─ Redis 7 (Session cache, rate limiting, quota cache)                   │
│   └─ SQLite (Local fallback, offline mode)                                 │
│                                                                             │
│   Message Bus:                                                              │
│   └─ NATS 2.10 + JetStream (Event-driven async communication)              │
│                                                                             │
│   Observability:                                                            │
│   ├─ Metrics: Prometheus + Grafana                                         │
│   ├─ Tracing: OpenTelemetry → Jaeger                                       │
│   ├─ Logging: Zap → Loki (optional)                                        │
│   └─ Alerting: Alertmanager                                                │
│                                                                             │
│   Infrastructure:                                                           │
│   ├─ Container: Docker                                                     │
│   ├─ Orchestration: K3s (Lightweight Kubernetes)                           │
│   ├─ VPN: Tailscale (Mesh networking)                                      │
│   └─ Storage: MinIO (S3-compatible, local 2TB SSD)                         │
│                                                                             │
│   Development:                                                              │
│   ├─ Language: Go 1.25 (Latest Stable)                                     │
│   ├─ Code Gen: protoc, Wire, hz CLI                                        │
│   └─ CI/CD: GitHub Actions                                                 │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Port Reference / 端口参考

| Service | Port | Protocol | Node | Description |
|---------|------|----------|------|-------------|
| Gateway | 18100 | HTTP | Brain | API Gateway |
| Provider | 18101 | HTTP | Brain | Provider Management |
| Log | 18102 | HTTP | Lab | Log Analytics |
| Billing | 18103 | HTTP | Brain | Billing & Auth |
| Sync | 8081 | HTTP | Brain | Session Sync |
| NEW-API | 3000 | HTTP | Brain | LLM Unified Gateway |
| NATS Client | 4222 | TCP | Brain | Message Bus |
| NATS WebSocket | 8222 | WS | Brain | Web clients |
| PostgreSQL | 5432 | TCP | Heart | Database |
| Redis | 6379 | TCP | Heart | Cache |
| ClickHouse HTTP | 8123 | HTTP | Lab | OLAP HTTP |
| ClickHouse Native | 9000 | TCP | Lab | OLAP Native |
| Prometheus | 9090 | HTTP | Monitor | Metrics |
| Grafana | 3000 | HTTP | Monitor | Dashboards |
| Alertmanager | 9093 | HTTP | Monitor | Alerts |
| Jaeger | 16686 | HTTP | Lab | Tracing UI |
| MinIO | 9000 | HTTP | Lab | Object Storage |

---

## 8. Security Considerations / 安全考量

1. **Network Isolation**: All internal services communicate via Tailscale VPN only
2. **TLS Everywhere**: External traffic through Cloudflare or direct TLS termination
3. **Secrets Management**: K8s Secrets for credentials, env vars from ConfigMaps
4. **API Authentication**: API keys validated at Gateway, forwarded as Bearer tokens
5. **Database Access**: PostgreSQL/Redis only accessible via Tailscale IPs

---

*Generated by Claude Code | 2026-01-12*
*Tech Stack: Hertz + Kratos + K3s + NATS + ClickHouse*
