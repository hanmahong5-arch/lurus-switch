# Gateway Service

> Hertz-based HTTP Gateway for AI Provider Proxy

## Overview

Gateway Service is a high-performance HTTP gateway built with [Hertz](https://github.com/cloudwego/hertz) (ByteDance's HTTP framework). It provides:

- **AI Provider Proxy**: Routes requests to Claude, OpenAI/Codex, and Gemini APIs
- **NEW-API Integration**: Forwards all requests through new-api unified gateway
- **Async Logging**: Publishes request logs to NATS JetStream
- **Observability**: OpenTelemetry tracing, Prometheus metrics

## Architecture

```
                 Client Requests
                       │
                       ▼
┌──────────────────────────────────────────┐
│           Gateway Service :18100          │
├──────────────────────────────────────────┤
│  Middleware: CORS → Tracing → Logger     │
│                     → Metrics            │
├────────┬──────────┬──────────┬───────────┤
│ Claude │  Codex   │  Gemini  │  Health   │
│Handler │ Handler  │ Handler  │  /metrics │
└────────┴────┬─────┴──────────┴───────────┘
              │
              ▼
┌─────────────────────────────────────────┐
│         RelayService (Proxy Core)       │
├───────────────┬─────────────────────────┤
│ ProviderClient│ BillingClient           │
│ (new-api)     │ (quota/usage)           │
└───────────────┴─────────────────────────┘
              │
      ┌───────┴───────┐
      ▼               ▼
┌──────────┐   ┌──────────────┐
│ NATS     │   │ AI Providers │
│ (logs)   │   │ (upstream)   │
└──────────┘   └──────────────┘
```

## API Routes

### Claude API
| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/messages` | Create message |
| POST | `/v1/messages/count_tokens` | Count tokens |
| POST | `/v1/messages/batches` | Batch operations |

### OpenAI/Codex API
| Method | Path | Description |
|--------|------|-------------|
| POST | `/responses` | Codex responses API |
| POST | `/v1/chat/completions` | Chat completions |
| POST | `/chat/completions` | Chat completions (alt) |
| POST | `/v1/completions` | Text completions |
| POST | `/v1/embeddings` | Embeddings |

### Gemini API
| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1beta/models` | List models |
| POST | `/v1beta/models/*` | Model actions (generateContent, etc.) |

### System
| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/ready` | Readiness probe |
| GET | `/metrics` | Prometheus metrics |

## Configuration

Configuration file: `configs/config.yaml`

```yaml
server:
  mode: production
  http:
    addr: ":18100"
    read_timeout: 30s
    write_timeout: 300s
    idle_timeout: 120s

features:
  new_api_enabled: true
  new_api_url: "http://localhost:3000"
  async_logging: true

nats:
  url: "nats://localhost:4222"

tracing:
  enabled: false
  endpoint: "localhost:4317"
  sample_rate: 0.1
```

## Build & Run

### Development
```bash
cd gateway-service
go run ./cmd/gateway -conf configs/config.yaml
```

### Production (Windows)
```powershell
# Build
go build -o gateway-hertz.exe ./cmd/gateway

# Run
.\gateway-hertz.exe -conf configs/config.yaml
```

## Dependencies

- **Hertz**: HTTP framework (~180k QPS)
- **Zap**: Structured logging
- **NATS**: Async message publishing
- **OpenTelemetry**: Distributed tracing
- **Prometheus**: Metrics collection

## Directory Structure

```
gateway-service/
├── cmd/
│   └── gateway/
│       └── main.go           # Entry point
├── configs/
│   └── config.yaml           # Configuration
├── internal/
│   ├── client/               # External service clients
│   │   ├── billing.go        # Billing service client
│   │   └── provider.go       # Provider service client
│   ├── conf/
│   │   └── config.go         # Config loader
│   ├── handler/              # HTTP handlers
│   │   ├── claude.go         # Claude API handler
│   │   ├── codex.go          # Codex API handler
│   │   └── gemini.go         # Gemini API handler
│   ├── middleware/           # HTTP middleware
│   │   ├── cors.go           # CORS
│   │   ├── logging.go        # Request logging
│   │   ├── metrics.go        # Prometheus metrics
│   │   └── tracing.go        # OpenTelemetry tracing
│   └── proxy/
│       ├── relay.go          # Core proxy logic
│       └── relay_test.go     # Unit tests
└── pkg/
    └── nats/
        └── publisher.go      # NATS publisher
```
