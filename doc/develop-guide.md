# Lurus Switch Development Guide
# Lurus Switch 开发指南

> **Version**: v2.0 | **Updated**: 2026-01-12
> **Language**: Go 1.25 | **Framework**: Hertz + Kratos

---

## 1. Environment Setup / 环境搭建

### 1.1 Prerequisites / 前置条件

```bash
# Required tools / 必需工具
go version                    # >= 1.25
docker --version              # >= 24.0
docker-compose --version      # >= 2.20
kubectl version --client      # >= 1.28 (for K3s)
```

### 1.2 Install Development Tools / 安装开发工具

```powershell
# Kratos CLI (Bilibili microservice framework)
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest

# Hertz CLI (ByteDance HTTP framework)
go install github.com/cloudwego/hertz/cmd/hz@latest

# Wire (Google DI)
go install github.com/google/wire/cmd/wire@latest

# Protocol Buffers
# Windows: download from https://github.com/protocolbuffers/protobuf/releases
# Or use scoop:
scoop install protobuf

# Protoc plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
```

### 1.3 Clone Repository / 克隆仓库

```bash
git clone https://github.com/hanmahong5-arch/lurus-switch.git
cd lurus-switch
```

### 1.4 Start Development Environment / 启动开发环境

```bash
# Start all infrastructure (PostgreSQL, Redis, NATS, ClickHouse)
docker-compose -f docker-compose.dev.yaml up -d

# Verify services are running
docker ps

# Access points:
# - Prometheus: http://localhost:9090
# - Grafana: http://localhost:3000 (admin/admin)
# - Jaeger: http://localhost:16686
# - NATS Monitor: http://localhost:8222
```

---

## 2. Project Structure / 项目结构

### 2.1 Service Hierarchy / 服务层级

```
lurus-switch/
├── gateway-service/      # Hertz - API Gateway (Port: 18100)
├── provider-service/     # Kratos - Provider Management (Port: 18101)
├── log-service/          # Kratos - Log Analytics (Port: 18102)
├── billing-service/      # Kratos - Billing & Auth (Port: 18103)
├── subscription-service/ # Kratos - Subscriptions
├── lurus-common/         # Shared libraries
├── lurus-portal/         # Vue 3 Frontend
├── api/                  # Protobuf definitions
└── deploy/               # Deployment configs
```

### 2.2 Kratos Service Layout / Kratos 服务结构

```
<service-name>/
├── cmd/
│   └── <service>/
│       ├── main.go           # Entry point
│       ├── wire.go           # DI definitions
│       └── wire_gen.go       # Generated DI code
├── internal/
│   ├── biz/                  # Business logic layer
│   │   ├── <entity>.go       # Domain logic
│   │   └── <entity>_test.go  # Unit tests
│   ├── data/                 # Data access layer
│   │   ├── data.go           # DB connections
│   │   └── <entity>.go       # Repository impl
│   ├── server/               # Server layer
│   │   ├── http.go           # HTTP server (Gin/Hertz)
│   │   └── grpc.go           # gRPC server (optional)
│   ├── service/              # Service layer
│   │   └── <entity>.go       # Service impl
│   └── conf/                 # Configuration structs
│       └── conf.go
├── configs/
│   └── config.yaml           # Configuration file
├── Dockerfile
├── Makefile
└── go.mod
```

---

## 3. Build Commands / 构建命令

### 3.1 Gateway Service (Hertz)

```bash
cd gateway-service

# Build binary
go build -o gateway.exe ./cmd/gateway

# Run with config
./gateway.exe -conf configs/config.yaml

# Development with hot reload (if using air)
air

# Run tests
go test ./... -v
```

### 3.2 Kratos Services (Provider/Log/Billing)

```bash
cd provider-service  # or log-service, billing-service

# Generate Wire DI code
wire ./cmd/<service>/

# Build binary
make build

# Run service
make run

# Run tests
go test ./... -v

# Generate API from proto
make api
```

### 3.3 Shared Library (lurus-common)

```bash
cd lurus-common

# Run tests
go test ./... -v

# Update dependencies
go mod tidy
```

---

## 4. Coding Standards / 编码规范

### 4.1 Go Code Style

```go
// GOOD: English comments, clear naming
// processRequest handles incoming LLM requests and routes to appropriate provider
func (s *RelayService) processRequest(ctx context.Context, req *Request) (*Response, error) {
    // Get provider configuration with cache
    provider, err := s.providerClient.MatchModel(ctx, req.Platform, req.Model)
    if err != nil {
        return nil, errors.Wrap(err, "failed to match provider")
    }

    // Forward request with tracing
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(attribute.String("provider", provider.Name))

    return s.forwardToProvider(ctx, provider, req)
}

// BAD: Chinese comments, unclear naming
// 处理请求
func (s *RelayService) pr(ctx context.Context, r *Request) (*Response, error) {
    p, e := s.pc.mm(ctx, r.p, r.m)
    // ...
}
```

### 4.2 Error Handling (Kratos Style)

```go
import (
    "github.com/go-kratos/kratos/v2/errors"
)

// Define error codes in errors package
var (
    ErrProviderNotFound = errors.NotFound("PROVIDER_NOT_FOUND", "provider not found")
    ErrQuotaExceeded    = errors.ResourceExhausted("QUOTA_EXCEEDED", "quota exceeded")
    ErrInvalidToken     = errors.Unauthenticated("INVALID_TOKEN", "invalid api token")
)

// Use in service
func (uc *ProviderUsecase) MatchModel(ctx context.Context, platform, model string) (*Provider, error) {
    provider, err := uc.repo.FindByModel(ctx, platform, model)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrProviderNotFound.WithMetadata(map[string]string{
                "platform": platform,
                "model":    model,
            })
        }
        return nil, errors.Wrapf(err, "query provider failed")
    }
    return provider, nil
}
```

### 4.3 Configuration Management

```yaml
# configs/config.yaml
server:
  http:
    addr: 0.0.0.0:18100
    timeout: 60s
  mode: development

database:
  driver: postgres
  source: ${DATABASE_DSN:postgresql://lurus:password@localhost:5432/lurus?sslmode=disable}

redis:
  addr: ${REDIS_ADDR:localhost:6379}
  password: ${REDIS_PASSWORD:}

nats:
  url: ${NATS_URL:nats://localhost:4222}
  stream: LLM_EVENTS

tracing:
  enabled: true
  endpoint: ${TRACING_ENDPOINT:localhost:4317}
  sample_rate: 1.0
```

```go
// internal/conf/conf.go
type Bootstrap struct {
    Server      ServerConfig      `yaml:"server"`
    Database    DatabaseConfig    `yaml:"database"`
    Redis       RedisConfig       `yaml:"redis"`
    NATS        NATSConfig        `yaml:"nats"`
    Tracing     TracingConfig     `yaml:"tracing"`
}
```

### 4.4 Logging Standards

```go
import "go.uber.org/zap"

// Create logger with request context
func (h *ClaudeHandler) Messages(ctx context.Context, c *app.RequestContext) {
    logger := h.logger.With(
        zap.String("trace_id", getTraceID(ctx)),
        zap.String("platform", "claude"),
        zap.String("model", getModel(c)),
    )

    logger.Info("Processing request",
        zap.Int("body_size", len(c.Request.Body())),
    )

    // ... handle request

    logger.Info("Request completed",
        zap.Duration("latency", time.Since(start)),
        zap.Int("status", c.Response.StatusCode()),
    )
}
```

---

## 5. Testing / 测试

### 5.1 Unit Tests

```go
// internal/biz/billing_test.go
func TestBillingUsecase_CheckQuota(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    repo := mocks.NewMockBillingRepo(ctrl)
    repo.EXPECT().GetUserQuota(gomock.Any(), "user123").Return(&Quota{
        Remaining: 1000,
    }, nil)

    uc := NewBillingUsecase(repo, zap.NewNop())

    err := uc.CheckQuota(context.Background(), "user123", 500)
    assert.NoError(t, err)
}
```

### 5.2 Integration Tests

```go
// internal/server/http_test.go
func TestGatewayIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Start test server
    srv := setupTestServer(t)
    defer srv.Close()

    // Send request
    resp, err := http.Post(srv.URL+"/v1/messages", "application/json", strings.NewReader(`{
        "model": "claude-3-opus",
        "messages": [{"role": "user", "content": "Hello"}]
    }`))

    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

### 5.3 Run Tests

```bash
# All tests
go test ./... -v

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Integration tests only
go test ./... -v -run Integration

# Specific package
go test ./internal/biz/... -v
```

---

## 6. Git Workflow / Git 工作流

### 6.1 Branch Naming

```
main              # Production-ready code
develop           # Integration branch
feature/<name>    # New features
fix/<issue>       # Bug fixes
refactor/<name>   # Code refactoring
docs/<name>       # Documentation updates
```

### 6.2 Commit Message Format

```
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Maintenance

**Examples:**
```
feat(gateway): add SSE streaming for Claude API
fix(billing): correct quota calculation overflow
docs(readme): update deployment instructions
refactor(provider): extract cache logic to separate layer
```

### 6.3 Pull Request Template

```markdown
## Summary
Brief description of changes

## Changes
- [ ] Change 1
- [ ] Change 2

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests pass
- [ ] Manual testing done

## Checklist
- [ ] Code follows project style guide
- [ ] Self-review completed
- [ ] No secrets committed
```

---

## 7. Deployment / 部署

### 7.1 Local Development

```bash
# Start infrastructure
docker-compose -f docker-compose.dev.yaml up -d

# Run services (in separate terminals)
cd gateway-service && make run
cd provider-service && make run
cd billing-service && make run
cd log-service && make run
```

### 7.2 Docker Build

```bash
# Build all images
docker-compose build

# Build specific service
docker build -t lurus-gateway:latest -f gateway-service/Dockerfile .
```

### 7.3 K3s Deployment

```bash
# Apply namespace
kubectl apply -f deploy/k3s/namespace.yaml

# Apply all resources
kubectl apply -k deploy/k3s/

# Check status
kubectl get pods -n lurus-system -o wide
```

---

## 8. Observability / 可观测性

### 8.1 Metrics (Prometheus)

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    requestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "gateway_requests_total",
            Help: "Total number of requests",
        },
        []string{"platform", "model", "status"},
    )

    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "gateway_request_duration_seconds",
            Help:    "Request duration in seconds",
            Buckets: []float64{.1, .5, 1, 2.5, 5, 10, 30, 60},
        },
        []string{"platform", "is_stream"},
    )
)

func init() {
    prometheus.MustRegister(requestsTotal, requestDuration)
}
```

### 8.2 Tracing (OpenTelemetry)

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func (h *Handler) HandleRequest(ctx context.Context, c *app.RequestContext) {
    tracer := otel.Tracer("gateway-service")
    ctx, span := tracer.Start(ctx, "HandleRequest")
    defer span.End()

    span.SetAttributes(
        attribute.String("platform", "claude"),
        attribute.String("model", model),
    )

    // ... process request

    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }
}
```

### 8.3 Structured Logging

```go
// All logs must include trace_id for correlation
logger.Info("Request processed",
    zap.String("trace_id", span.SpanContext().TraceID().String()),
    zap.String("platform", platform),
    zap.String("model", model),
    zap.Int("input_tokens", inputTokens),
    zap.Int("output_tokens", outputTokens),
    zap.Duration("latency", latency),
)
```

---

## 9. Security / 安全

### 9.1 Secret Management

```bash
# Create K8s secret
kubectl create secret generic lurus-secrets \
    --from-literal=DATABASE_PASSWORD=xxx \
    --from-literal=REDIS_PASSWORD=xxx \
    --from-literal=NEW_API_TOKEN=xxx \
    -n lurus-system
```

### 9.2 API Authentication

```go
// Middleware for API key validation
func AuthMiddleware() app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        apiKey := c.GetHeader("Authorization")
        if !strings.HasPrefix(apiKey, "Bearer ") {
            c.AbortWithStatusJSON(401, map[string]string{
                "error": "Missing or invalid Authorization header",
            })
            return
        }

        token := strings.TrimPrefix(apiKey, "Bearer ")
        if !validateToken(token) {
            c.AbortWithStatusJSON(401, map[string]string{
                "error": "Invalid API token",
            })
            return
        }

        c.Next(ctx)
    }
}
```

---

## 10. Common Issues / 常见问题

### 10.1 Database Connection

```go
// Always use connection pooling
db, err := sql.Open("postgres", dsn)
if err != nil {
    return err
}
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### 10.2 NATS Reconnection

```go
// Configure auto-reconnect
nc, err := nats.Connect(url,
    nats.MaxReconnects(-1),           // Infinite reconnects
    nats.ReconnectWait(2*time.Second),
    nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
        logger.Warn("NATS disconnected", zap.Error(err))
    }),
    nats.ReconnectHandler(func(nc *nats.Conn) {
        logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
    }),
)
```

### 10.3 Windows File Handles

```go
// Always close files in defer or finally blocks
file, err := os.Open(path)
if err != nil {
    return err
}
defer file.Close() // Critical on Windows to avoid lock issues
```

---

## 11. Reference / 参考资料

- [Kratos Documentation](https://go-kratos.dev/)
- [Hertz Documentation](https://www.cloudwego.io/docs/hertz/)
- [NATS Documentation](https://docs.nats.io/)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)
- [ClickHouse Documentation](https://clickhouse.com/docs/)

---

*Generated by Claude Code | 2026-01-12*
*Framework: Hertz + Kratos + NATS + ClickHouse*
