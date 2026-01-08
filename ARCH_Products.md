# Ailurus PaaS - Product Architecture

> Version: 3.0.0 | Updated: 2026-01-05

## Overview

Ailurus PaaS is a unified AI gateway platform that provides centralized management for multiple AI CLI tools (Claude Code, Codex, Gemini CLI). The platform offers:

- **Unified LLM Routing**: All AI requests through NEW-API gateway
- **Unified Payment System**: Centralized quota management
- **Unified Message Bus**: NATS for real-time sync and events

---

## Product Components

### 1. CodeSwitch (Desktop Gateway)

**Technology**: Go + Wails 3 + Vue 3

**Location**: `codeswitch/`

**Function**: Local HTTP proxy that intercepts CLI requests and routes them through the unified gateway.

```
┌─────────────────────────────────────────────────────────────────┐
│  CodeSwitch Gateway (:18100)                                    │
├─────────────────────────────────────────────────────────────────┤
│  Endpoints:                                                     │
│  ├─ POST /v1/messages      → Claude (Anthropic format)          │
│  ├─ POST /responses        → Codex (OpenAI Responses API)       │
│  ├─ POST /v1/chat/completions → Generic (OpenAI format)         │
│  └─ POST /v1beta/models/*  → Gemini (Native, auto-converted)    │
├─────────────────────────────────────────────────────────────────┤
│  Features:                                                      │
│  ├─ NEW-API unified forwarding                                  │
│  ├─ Gemini ↔ OpenAI format conversion                          │
│  ├─ NATS event publishing                                       │
│  ├─ Request logging & statistics                                │
│  └─ Provider failover                                           │
└─────────────────────────────────────────────────────────────────┘
```

**Key Files**:
| File | Purpose |
|------|---------|
| `services/providerrelay.go` | HTTP proxy, NEW-API forwarding, format conversion |
| `services/sync_integration.go` | NATS event hooks |
| `services/sync/sync_service.go` | NATS client, LLM consumer |
| `services/sync/nats_client.go` | NATS connection management |
| `services/appsettings.go` | App settings + NEW-API config |

---

### 2. Gemini CLI (Forked)

**Technology**: TypeScript + Node.js

**Location**: `gemini-cli/`

**Function**: Google's official Gemini CLI, modified to integrate with Ailurus PaaS.

**Modifications**:
- CodeSwitch integration via `USE_CODESWITCH` environment variable
- NATS client for direct message bus communication
- Unified quota event subscription

**Key Files**:
| File | Purpose |
|------|---------|
| `packages/core/src/core/contentGenerator.ts` | LLM API entry point, CodeSwitch integration |
| `packages/core/src/nats/nats-client.ts` | NATS WebSocket client |
| `packages/core/src/nats/llm-client.ts` | LLM client with NATS + HTTP fallback |
| `packages/core/src/nats/types.ts` | NATS message types |

**Environment Variables**:
```bash
# CodeSwitch integration (default: enabled)
USE_CODESWITCH=true
CODESWITCH_BASE_URL=http://127.0.0.1:18100

# NATS integration (optional)
NATS_ENABLED=true
NATS_URL=nats://localhost:4222

# NEW-API direct access (for NATS mode)
NEW_API_URL=http://localhost:3000
NEW_API_TOKEN=sk-xxx

# User identification (for NATS events)
AILURUS_USER_ID=user_123
AILURUS_SESSION_ID=session_456
```

---

### 3. NEW-API (LLM Gateway)

**Technology**: Go

**Location**: `new-api/`

**Production URL**: `http://api.lurus.cn` (管理台 + API 端点)

**Function**: Unified LLM gateway supporting 40+ AI providers with quota management.

**Features**:
- Multi-provider routing (OpenAI, Claude, Gemini, DeepSeek, etc.)
- Token-based quota system
- Request logging and cost calculation
- API format conversion
- Web management console (用户管理、配额充值、渠道配置)

**API Endpoints**:
```
# LLM API
POST /v1/chat/completions  → OpenAI compatible
POST /v1/messages          → Anthropic compatible

# User API
GET  /api/user/self        → Get current user info + quota
GET  /api/user/token       → List API tokens

# Management Console
GET  /                     → Web dashboard (http://api.lurus.cn)
```

---

### 4. NATS Server (Message Bus)

**Technology**: NATS + JetStream

**Location**: `codeswitch/deploy/nats/`

**Function**: Real-time message bus for event synchronization.

**Subjects**:
```
# LLM Requests
llm.request.{platform}      → LLM request queue (claude/codex/gemini)
llm.response.{trace_id}     → LLM response delivery

# User Events
user.{user_id}.quota        → Quota change notifications
user.{user_id}.auth         → Authentication events
user.{user_id}.presence     → Online status

# Chat Sync
chat.{user_id}.{session}.msg    → Message sync
chat.{user_id}.{session}.status → Session status
```

**JetStream Streams**:
| Stream | Subjects | Retention |
|--------|----------|-----------|
| CHAT_MESSAGES | `chat.*.*.msg` | Permanent (10GB) |
| SESSION_STATUS | `chat.*.*.status` | 1 day (memory) |
| USER_EVENTS | `user.*.auth,quota,...` | 7 days |
| LLM_REQUESTS | `llm.request.*,llm.response.*` | 1 hour (memory) |

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Client Layer                                 │
│  (Claude Code / Codex CLI / Gemini CLI / Mobile App)                │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         │ HTTP            │ NATS            │
         ▼                 ▼                 │
┌─────────────────┐ ┌─────────────────────────────────────────────────┐
│  CodeSwitch     │ │              NATS Server (:4222)                │
│  Gateway        │ │          JetStream + Message Bus                 │
│  (:18100)       │ │                                                 │
│                 │ │  Subjects:                                       │
│  - /v1/messages │ │  ├─ llm.request.{platform}  → LLM requests      │
│  - /responses   │ │  ├─ llm.response.{trace_id} → LLM responses     │
│  - /v1beta/*    │ │  ├─ user.{user_id}.quota    → Quota changes     │
│                 │ │  └─ chat.{user_id}.{session} → Message sync     │
└────────┬────────┘ └──────────────────────────┬──────────────────────┘
         │                                     │
         └─────────────────┬───────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      NEW-API Gateway (:3000)                        │
│  ┌────────────────────────────────────────────────────────────┐    │
│  │  • Unified Authentication (Token + APIKey)                  │    │
│  │  • Quota Management (Charge / Deduct / Query)               │    │
│  │  • Multi-Provider Routing (40+ Providers)                   │    │
│  │  • Format Conversion (OpenAI ↔ Claude ↔ Gemini)            │    │
│  │  • Request Logging + Cost Calculation                       │    │
│  └────────────────────────────────────────────────────────────┘    │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       AI Providers                                   │
│  (OpenAI / Anthropic / Google Gemini / DeepSeek / Qwen / ...)       │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Request Flow

### A. HTTP Mode (Default)

```
Client CLI ──HTTP──> CodeSwitch :18100
                          │
                          ▼ [Check NEW-API Mode]
                          │
           ┌──────────────┴──────────────┐
           │ NEW-API Enabled             │ Fallback
           ▼                             ▼
    NEW-API :3000              Local Provider List
    /v1/chat/completions       (Priority + Round-Robin)
           │                             │
           └──────────────┬──────────────┘
                          │
                          ▼
                    AI Provider API
```

### B. NATS Mode (Optional)

```
Client ──NATS──> llm.request.{platform}
                          │
                          ▼
              [LLM Request Consumer]
                          │
                          ▼
                   NEW-API :3000
                          │
                          ▼
              ──NATS──> llm.response.{trace_id}
```

### C. Gemini Format Conversion

```
Gemini CLI ──HTTP──> :18100/v1beta/models/*
                          │
                          ▼ [Gemini → OpenAI Conversion]
                          │
                          ▼
                   NEW-API :3000/v1/chat/completions
                          │
                          ▼ [OpenAI → Gemini Conversion]
                          │
                          ▼
                   Response to CLI
```

---

## Data Types

### LLM Request (NATS)

```typescript
interface LLMNATSRequest {
  trace_id: string;
  user_id: string;
  session_id?: string;
  platform: 'claude' | 'codex' | 'gemini';
  model: string;
  messages: LLMMessage[];
  stream: boolean;
  metadata?: Record<string, string>;
}
```

### LLM Response (NATS)

```typescript
interface LLMNATSResponse {
  trace_id: string;
  success: boolean;
  model: string;
  content: string;
  tokens_input: number;
  tokens_output: number;
  cost: number;
  duration_ms: number;
  error?: string;
}
```

### Quota Change Event

```typescript
interface QuotaChangeEvent {
  user_id: string;
  quota_total: number;
  quota_used: number;
  quota_remain: number;
  last_cost: number;
  model?: string;
  trace_id?: string;
  timestamp: string;
}
```

---

## Configuration

### CodeSwitch (`~/.code-switch/app.json`)

```json
{
  "new_api_enabled": true,
  "new_api_url": "http://api.lurus.cn",
  "new_api_token": "sk-xxx",
  "show_heatmap": true,
  "auto_start": false
}
```

> **Note**: 开发环境可使用 `http://localhost:3000`，生产环境推荐 `http://api.lurus.cn`

### NATS Sync (`~/.code-switch/sync-settings.json`)

```json
{
  "enabled": true,
  "url": "nats://localhost:4222"
}
```

### Gemini CLI Environment

```bash
# CodeSwitch (HTTP mode)
USE_CODESWITCH=true
CODESWITCH_BASE_URL=http://127.0.0.1:18100

# NATS (direct message bus)
NATS_ENABLED=true
NATS_URL=nats://localhost:4222

# NEW-API (production)
NEW_API_URL=http://api.lurus.cn
NEW_API_TOKEN=sk-xxx

# NEW-API (development)
# NEW_API_URL=http://localhost:3000
```

---

## Implementation Status

### Phase 1: Unified LLM Calls ✅

- [x] NEW-API unified gateway mode
- [x] Claude/Codex/Gemini multi-endpoint support
- [x] Gemini Native ↔ OpenAI format conversion

### Phase 2: Unified Payment System ✅

- [x] NEW-API quota API integration
- [x] Quota change event broadcasting
- [x] Sync integration hooks

### Phase 3: NATS Message Bus ✅

- [x] NATS client (Go) - `sync/nats_client.go`
- [x] LLM request consumer (Go) - `sync/sync_service.go`
- [x] NATS client (TypeScript) - `nats/nats-client.ts`
- [x] LLM client with fallback (TypeScript) - `nats/llm-client.ts`

### Phase 4: Future Enhancements

- [ ] Admin dashboard
- [ ] Mobile app integration
- [ ] Multi-tenant management
- [ ] Advanced analytics

---

## Directory Structure

```
lurus-switch/
├── codeswitch/                 # Desktop App (Wails 3 + Vue 3)
│   ├── main.go                 # Entry point
│   ├── services/
│   │   ├── providerrelay.go    # HTTP Proxy + NEW-API
│   │   ├── sync_integration.go # NATS hooks
│   │   └── sync/               # NATS client
│   ├── frontend/               # Vue 3 UI
│   └── deploy/nats/            # NATS configuration
│
├── gemini-cli/                 # Gemini CLI (forked)
│   ├── packages/core/src/
│   │   ├── core/               # LLM core
│   │   │   └── contentGenerator.ts  # CodeSwitch integration
│   │   └── nats/               # NATS integration
│   │       ├── nats-client.ts  # WebSocket NATS client
│   │       ├── llm-client.ts   # LLM client + HTTP fallback
│   │       └── types.ts        # Type definitions
│   └── Ailurus_Architecture.md
│
├── new-api/                    # LLM Gateway (forked)
│   └── ...
│
├── ARCH_Products.md            # This file
└── nats/                       # NATS Server (reference)
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 3.0.0 | 2026-01-05 | NATS TypeScript client, unified architecture |
| 2.0.0 | 2026-01-04 | NEW-API integration, quota system |
| 1.0.0 | 2026-01-03 | Initial CodeSwitch gateway |
