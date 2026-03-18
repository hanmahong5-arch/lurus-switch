---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
inputDocuments: ['lurus.yaml', 'CLAUDE.md', 'doc/decisions/0001-single-source-of-truth.md', 'product-brief.md', 'project-context.md', 'prd-gushen.md', 'epics-gushen.md', 'prd.md']
date: 2026-02-02
regenerated: 2026-02-27
author: Anita (via BMAD Architecture Review)
sectionsAdded: ['8-implementation-patterns', '9-project-structure-boundaries']
sectionsUpdated: ['1-system-context', '2-adr', '7-tech-radar', '8-implementation-patterns', '9-project-structure']
---

# Architecture Decision Document: Lurus Platform
# 架构决策文档：Lurus 平台

---

## 1. System Context / 系统上下文

### 1.1 System Boundary / 系统边界

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Lurus Platform                                   │
│                                                                         │
│  ┌─────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐  ┌───────────┐ │
│  │lurus-api│  │lurus-    │  │lurus-    │  │lurus-   │  │lurus-     │ │
│  │(Gateway)│  │lucrum    │  │webmail   │  │newapi   │  │switch     │ │
│  │         │  │(Quant)   │  │(Mail)    │  │(LLM Mgr)│  │(Desktop)  │ │
│  └────┬────┘  └────┬─────┘  └────┬─────┘  └────┬────┘  └─────┬─────┘ │
│       │            │             │              │              │        │
│  ┌────┴────────────┴─────────────┴──────────────┴────────────┐ │        │
│  │              Shared Infrastructure Layer                   │ │        │
│  │  PostgreSQL │ Redis │ NATS JetStream │ MinIO │ Stalwart │ Zitadel│ │
│  └───────────────────────────────────────────────────────────┘ │        │
│                                                                 │        │
│  ┌───────────────────────────────────────────────────────────┐ │        │
│  │              Platform Layer (K3s + Tailscale VPN)          │ │        │
│  │  5 nodes │ Traefik Ingress │ ArgoCD │ Grafana │ Prometheus │ Loki│ │
│  └───────────────────────────────────────────────────────────┘ │        │
└─────────────────────────────────────────────────────────────────┘        │
         ↕                    ↕                    ↕                       │
    LLM Providers        Market Data APIs       Email Providers            │
    (OpenAI, etc.)       (Eastmoney, Sina)      (SendCloud relay)          │
                                                                           │
┌──────────────────────────────────────────────────────────────────────────┘
│
│  lurus-switch (Desktop App — Wails)
│  ┌─────────────────────────────────────────────────────────────────────┐
│  │                                                                     │
│  │  ┌───────────────┐  ┌──────────────┐  ┌──────────────────────────┐ │
│  │  │ Config Editor │  │ CLAUDE.md    │  │ MCP Server Manager       │ │
│  │  │ (Claude/Codex │  │ Generator    │  │ (Visual config, catalog, │ │
│  │  │  /Gemini)     │  │ (Static      │  │  cross-tool sync)        │ │
│  │  │               │  │  analysis +  │  │                          │ │
│  │  │               │  │  templates)  │  │                          │ │
│  │  └───────────────┘  └──────────────┘  └──────────────────────────┘ │
│  │                                                                     │
│  │  ┌───────────────┐  ┌──────────────┐  ┌──────────────────────────┐ │
│  │  │ Cost Dashboard│  │ Proxy &      │  │ Prompt Library           │ │
│  │  │ (via lurus-   │  │ Network      │  │ (Built-in + custom       │ │
│  │  │  api gateway) │  │ (auto-detect │  │  prompt management)      │ │
│  │  │               │  │  + GFW)      │  │                          │ │
│  │  └───────────────┘  └──────────────┘  └──────────────────────────┘ │
│  │                                                                     │
│  │                    Distribution Channels                            │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌─────────────────┐       │
│  │  │  Scoop   │ │ Homebrew │ │  WinGet  │ │ GitHub Releases │       │
│  │  └──────────┘ └──────────┘ └──────────┘ └─────────────────┘       │
│  └─────────────────────────────────────────────────────────────────────┘
│          │                              │
│          ▼                              ▼
│   AI CLI Tools                    lurus-api
│   (Claude Code,                   (Cost data,
│    Codex, Gemini CLI)              LLM usage)
│
```

### 1.2 External Dependencies / 外部依赖

| External System | Protocol | Purpose | Fallback |
|----------------|----------|---------|----------|
| OpenAI API | HTTPS | LLM inference | Route to Anthropic/Google |
| Anthropic API | HTTPS | LLM inference | Route to OpenAI/Google |
| Google AI API | HTTPS | LLM inference | Route to OpenAI/Anthropic |
| AWS Bedrock | HTTPS | LLM inference (Claude) | Direct Anthropic API |
| DeepSeek API | HTTPS | Strategy code generation | Fallback to OpenAI |
| Eastmoney API | HTTPS | A-share market data | Simulated data |
| Sina Finance API | HTTPS | Real-time quotes | Eastmoney fallback |
| GitHub API | HTTPS | Strategy crawler source | Cached results |
| SendCloud SMTP | SMTP | China email relay | Direct Stalwart delivery |
| GitHub / GHCR | HTTPS | Code hosting & container registry | Manual deployment |
| Zitadel | OIDC | Authentication | Session-based fallback |

---

## 2. Key Architecture Decisions / 关键架构决策

### ADR-001: Single Source of Truth (lurus.yaml)

**Status**: ✅ Accepted (2026-01-26)

**Context**: 7 services with interconnected infrastructure, 2-person team needs minimal overhead.

**Decision**: All architecture configuration in a single `lurus.yaml` file.

**Consequences**:
- (+) One file to rule them all - no config drift
- (+) Easy to review infrastructure changes
- (-) Single point of knowledge - must be version controlled carefully
- (-) No dynamic service discovery (acceptable for small scale)

---

### ADR-002: Independent Projects, Not Microservices

**Status**: ✅ Accepted

**Context**: Small team, 7 distinct business domains.

**Decision**: Each business = independent project = independent Pod. NO microservice splitting within a single business.

**Rationale**:
- 2-person team cannot maintain microservice complexity
- Each service has clear domain boundaries
- Inter-service communication via NATS JetStream when needed
- Shared infrastructure (PostgreSQL, Redis) with logical isolation

**Consequences**:
- (+) Simple deployment (one pod per service)
- (+) Independent scaling and lifecycle
- (+) Clear ownership boundaries
- (-) Some code duplication across services (acceptable trade-off)

---

### ADR-003: Schema Isolation over Database Isolation

**Status**: ✅ Accepted

**Context**: Running 7 services, budget for one PostgreSQL instance.

**Decision**: Single PostgreSQL instance (via CNPG operator), schema-level isolation per service.

**Rationale**:
- CNPG provides automated backup, failover, monitoring
- Schema isolation provides logical separation without operational overhead
- Cross-schema queries explicitly forbidden by team convention

**Consequences**:
- (+) Lower infrastructure cost (one DB instance)
- (+) Shared backup/recovery procedures
- (-) Noisy neighbor risk (one service can impact others)
- (-) Schema name discipline required
- Mitigation: Resource monitoring, connection pooling

---

### ADR-004: Hybrid Cloud Architecture

**Status**: ✅ Accepted

**Context**: Cost optimization, some workloads need cloud, some are fine on-premise.

**Decision**: K3s cluster spanning cloud VMs (compute/database) + office machines (messaging/storage) connected via Tailscale VPN.

**Node Allocation**:

| Node | Location | Role | Rationale |
|------|----------|------|-----------|
| cloud-ubuntu-1 (16C/32G) | Cloud | Master + API Gateway + Staging | Public IP needed, high CPU for gateway |
| cloud-ubuntu-2 (4C/8G) | Cloud | Database | Low latency to compute nodes |
| cloud-ubuntu-3 (2C/2G) | Cloud | Worker | Web services |
| office-debian-2 | Office | Messaging | NATS/Redis don't need public access |
| office-win-1 | Office | Storage | MinIO on cheap local storage |

**Consequences**:
- (+) Significant cost reduction (~60% vs all-cloud)
- (+) Leverage existing office hardware
- (-) Cross-WAN latency for office nodes
- (-) Office network reliability dependency
- Mitigation: Tailscale mesh networking, monitoring alerts

---

### ADR-005: GitOps Deployment Pipeline

**Status**: ✅ Accepted

**Decision**: GitHub Actions → GHCR → ArgoCD → K3s

```
Code Push → GitHub Actions (build, test, docker build, push to GHCR)
                ↓
            GHCR Image
                ↓
            ArgoCD Sync (watches deploy/ manifests)
                ↓
            K3s Rolling Update (production or staging)
```

**Consequences**:
- (+) Fully automated, reproducible deployments
- (+) Git history = deployment history
- (+) Easy rollback (ArgoCD)
- (-) Requires ArgoCD operational health
- (-) Cold start delay (build → push → sync cycle)

---

### ADR-006: Financial Calculation with Decimal.js

**Status**: ✅ Accepted

**Context**: Quantitative trading platform requires exact decimal arithmetic.

**Decision**: ALL monetary calculations use Decimal.js via `FinancialAmount` wrapper. JavaScript native numbers FORBIDDEN for financial values.

**Rationale**:
```javascript
// The classic floating point problem
0.1 + 0.2 === 0.30000000000000004 // true in JS
// With Decimal.js
new Decimal('0.1').plus('0.2').toString() // '0.3'
```

**Validation**: 680+ unit tests (85%+ coverage) verify financial calculation correctness.

**Consequences**:
- (+) No floating point precision bugs in financial calculations
- (+) China A-share 100-lot constraint properly enforced
- (-) Performance overhead (~10x slower than native numbers)
- (-) Requires discipline (easy to accidentally use native numbers)
- Mitigation: Linting rules, code review, `FinancialAmount` wrapper API

---

### ADR-007: Multi-Agent AI Advisor Architecture

**Status**: ✅ Accepted

**Decision**: 11 specialized AI agents (4 analysts + 3 researchers + 4 master personas) + 7 investment schools + debate mode.

**Architecture**:
```
User Query
    ↓
Agent Router (selects relevant agents)
    ↓
┌─────────────────────────────────────────┐
│ Analyst Agents    │ Researcher Agents    │
│ - Technical       │ - Market             │
│ - Fundamental     │ - Industry           │
│ - Quantitative    │ - Macro              │
│ - Sentiment       │                      │
├──────────────────────────────────────────┤
│ Master Personas (Investment Philosophy)  │
│ - Buffett (Value) │ - Lynch (Growth)     │
│ - Livermore (Technical) │ - Simons (Quant)│
└──────────────────────────────────────────┘
    ↓
Token Budget Manager (context size control)
    ↓
Response Synthesis (SSE streaming)
```

**Consequences**:
- (+) Rich, multi-perspective investment analysis
- (+) Debate mode provides balanced bull/bear arguments
- (-) Token consumption can be high (mitigated by token budget manager)
- (-) Agent quality depends on prompt engineering

---

### ADR-008: Self-Hosted Email with Stalwart + SendCloud Relay

**Status**: ✅ Accepted

**Context**: Need corporate email (lurus.cn), Chinese ISPs block direct SMTP.

**Decision**: Stalwart (self-hosted, RocksDB backend) + SendCloud relay for Chinese domains.

**Mail Routing**:
```
Outbound to Chinese domains (qq.com, 163.com, etc.)
    → SendCloud SMTP relay (smtp.sendcloud.net:587)

Outbound to international domains
    → Direct Stalwart delivery

Inbound
    → MX record → Stalwart (port 25/465/993)
```

**Consequences**:
- (+) Full data sovereignty for email
- (+) No per-user SaaS fees
- (+) Chinese email delivery reliability via SendCloud
- (-) Self-managed SPF/DKIM/DMARC
- (-) IP reputation management required

---

### ADR-009: Workflow Orchestration System (NEW)

**Status**: ✅ Accepted (2026-01-24)

**Context**: Strategy development involves multiple interdependent steps (input → generate → backtest → validate). Users frequently iterate on individual steps while wanting to preserve others.

**Decision**: Implement a `WorkflowManager` with `StepExecutor` and `CacheStrategy` in `src/lib/workflow/`.

**Architecture**:
```
POST /api/workflow         → WorkflowManager.createSession()
POST /api/workflow/:id/step/:n → StepExecutor.execute()
                                      ↓
                                CacheStrategy.lookup(inputHash)
                                      ↓ miss
                                Execute step logic
                                      ↓
                                CacheStrategy.store(inputHash, result, TTL)
```

**Cache Strategy**:
- Input hash = SHA-256 of step inputs (deterministic)
- Per-step TTL configuration
- Automatic invalidation when upstream step re-executed

**Consequences**:
- (+) Users can iterate on individual steps without re-running the entire pipeline
- (+) Cache eliminates redundant AI calls and backtest computations
- (+) Clear separation of concerns (orchestration vs execution vs caching)
- (-) Additional complexity in session state management
- (-) Cache invalidation edge cases require careful handling

---

### ADR-010: Strategy Crawler & Discovery (NEW)

**Status**: ✅ Accepted (2026-01-24)

**Context**: Users benefit from discovering proven trading strategies from the open-source community rather than starting from scratch.

**Decision**: Implement a GitHub crawler (`src/lib/crawler/`) that discovers, scores, and converts open-source strategies.

**Pipeline**:
```
GitHubCrawler (search + fetch)
    ↓
PopularityScorer (stars, forks, quality, freshness)
    ↓
StrategyConverter (→ vnpy CtaTemplate format)
    ↓
CrawlerScheduler (cron-based, rate-limited)
    ↓
API endpoints (/api/strategies/popular, /trending)
```

**Consequences**:
- (+) Users discover quality strategies without manual searching
- (+) Popularity scoring surfaces the most relevant strategies
- (+) Automatic format conversion reduces friction
- (-) GitHub API rate limits require careful management
- (-) Converted strategies may need manual parameter tuning

---

### ADR-011: Staging Environment Strategy (NEW)

**Status**: ✅ Accepted (2026-02-01)

**Context**: Testing in production is risky; need isolated pre-production environment.

**Decision**: Deploy staging to `ai-qtrd-staging` namespace on master node with isolated Redis (db:3).

**K8s Configuration** (`deploy/k8s/staging/web-deployment.yaml`):
- Namespace: `ai-qtrd-staging`
- Image tag: `staging`
- Node selector: master node (cloud-ubuntu-1)
- Redis: db:3 (isolated from production db:1)
- Resources: CPU 50-250m, Memory 128-256Mi
- Health probes: liveness (30s) + readiness (10s)

**Consequences**:
- (+) Safe testing without production impact
- (+) Same K3s cluster, minimal additional cost
- (+) Isolated Redis namespace prevents data leakage
- (-) Master node hosts both staging + critical services (monitor resources)
- (-) Shared PostgreSQL instance (schema-level isolation only)

---

### ADR-012: i18n Strategy (react-i18next, Chinese Default)

**Status**: Accepted (2026-02-27)

**Context**: Lurus Switch targets Chinese developers as the primary audience (P5 in PRD), with English as a secondary language. The desktop app uses React 18 via Wails. UI text must be fully translatable with zero hardcoded strings, and Chinese must be the default locale.

**Decision**: Use `react-i18next` with namespace-based JSON resource files. Chinese (zh-CN) is the default and fallback language; English (en) is supported as a secondary locale.

**Architecture**:
```
frontend/src/i18n/
├── i18n.ts                  # i18next initialization (lng: 'zh-CN', fallbackLng: 'zh-CN')
├── locales/
│   ├── zh-CN/
│   │   ├── common.json      # Shared (nav, buttons, errors)
│   │   ├── dashboard.json   # Dashboard page
│   │   ├── config.json      # Config editor
│   │   ├── claudemd.json    # CLAUDE.md generator
│   │   ├── mcp.json         # MCP server manager
│   │   ├── cost.json        # Cost dashboard
│   │   └── onboarding.json  # Onboarding wizard
│   └── en/
│       └── (mirrors zh-CN structure)
```

**Key Decisions**:
- Namespace per feature page to enable code-splitting of translation bundles
- Language persisted in Zustand store (`settings.language`) with `persist` middleware
- No inline Chinese/English text in components; all strings via `useTranslation()` hook
- Date/number formatting via `Intl.DateTimeFormat` / `Intl.NumberFormat` with locale parameter

**Consequences**:
- (+) Clean separation of UI text from component logic
- (+) Adding new languages requires only new JSON files, no code changes
- (+) Namespace splitting keeps bundle size small per page
- (-) Initial setup overhead for extracting all strings
- (-) Translation key management requires discipline (unused keys accumulate)

---

### ADR-013: app.go Decomposition into Domain-Specific Binding Files

**Status**: Accepted (2026-02-27)

**Context**: The current `app.go` contains all Wails binding methods (Go functions exposed to the frontend). As features grow from MVP (config editing) through Milestone 2 (CLAUDE.md generator, MCP manager, cost dashboard), a single file will become unmaintainable (projected 1500+ lines).

**Decision**: Decompose `app.go` into domain-specific binding files. Each file owns one feature domain's Wails-exported methods. The `App` struct remains in `app.go` as the central composition root.

**Structure**:
```
lurus-switch/
├── app.go                   # App struct, lifecycle (OnStartup/OnShutdown), shared state
├── bindings_config.go       # Config read/write/validate/snapshot for Claude/Codex/Gemini
├── bindings_billing.go      # Billing status, subscription check, entitlement queries
├── bindings_mcp.go          # MCP server CRUD, health check, catalog browse
├── bindings_tools.go        # Tool detection, install/update/uninstall, health status
├── bindings_claudemd.go     # Project scanning, CLAUDE.md generation, quality scoring
├── bindings_cost.go         # Cost data fetch from lurus-api, budget alerts
├── bindings_proxy.go        # Proxy detection, configuration, connectivity test
├── bindings_prompt.go       # Prompt library CRUD, import/export
└── internal/
    ├── config/              # Config data models & persistence
    ├── generator/           # CLAUDE.md generation engine
    ├── analyzer/            # Project structure static analysis
    ├── mcp/                 # MCP server management logic
    ├── tools/               # CLI tool detection & management
    ├── cost/                # Cost data aggregation
    ├── proxy/               # Proxy detection & configuration
    ├── keychain/            # OS keychain integration
    ├── validator/           # Config validation
    └── packager/            # Executable packaging
```

**Rules**:
- `app.go` ONLY contains: `App` struct definition, `NewApp()`, `OnStartup()`, `OnShutdown()`, and shared helpers
- Each `bindings_*.go` file defines methods on `*App` for one domain
- Business logic lives in `internal/<domain>/`, NOT in binding files
- Binding methods are thin wrappers: validate input -> call internal -> return result
- File naming: `bindings_<domain>.go` (snake_case, `bindings_` prefix)

**Consequences**:
- (+) Each domain's API surface is isolated and easy to navigate
- (+) Multiple developers can work on different features without merge conflicts
- (+) Clear mapping from frontend feature to backend binding file
- (-) More files to manage (mitigated by consistent naming convention)
- (-) `App` struct may accumulate many internal service dependencies (mitigated by lazy initialization)

---

### ADR-014: API Key Storage (OS Keychain with Fallback)

**Status**: Accepted (2026-02-27)

**Context**: Lurus Switch manages API keys for multiple AI CLI tools (Anthropic, OpenAI, Google, DeepSeek) and the lurus-api gateway. Keys must be stored securely, survive app reinstalls, and not leak in config exports or logs.

**Decision**: Primary storage via OS keychain (Windows Credential Manager / macOS Keychain / Linux Secret Service). Fallback to AES-256-GCM encrypted file for environments without keychain access.

**Architecture**:
```
internal/keychain/
├── keychain.go              # Interface: Store/Retrieve/Delete/List
├── keychain_windows.go      # Windows Credential Manager (wincred)
├── keychain_darwin.go       # macOS Keychain (go-keyring)
├── keychain_linux.go        # Linux Secret Service D-Bus API
└── keychain_fallback.go     # AES-256-GCM encrypted JSON file

Service name: "lurus-switch"
Key naming:   "lurus-switch/<provider>"
              e.g. "lurus-switch/anthropic", "lurus-switch/openai"
```

**Key Decisions**:
- Use `github.com/zalando/go-keyring` as the cross-platform keychain abstraction
- Fallback file stored at `~/.lurus-switch/credentials.enc`
- Encryption key derived from machine-specific entropy (machine ID + username hash)
- Config export always strips API keys; import prompts user to re-enter
- API keys never appear in log output (masked as `sk-...xxxx`)

**Consequences**:
- (+) Keys protected by OS-level security (biometric unlock, secure enclave on macOS)
- (+) Keys survive app reinstall (keychain persists independently)
- (+) Fallback ensures functionality on headless/containerized environments
- (-) Cross-platform keychain behavior is inconsistent (Linux D-Bus may not be available)
- (-) Fallback encryption is only as strong as machine-specific entropy
- Mitigation: Clear warning in UI when using fallback mode

---

### ADR-015: CLAUDE.md Analysis Engine (Static Analysis of Project Structure)

**Status**: Accepted (2026-02-27)

**Context**: The Smart CLAUDE.md Generator (F5 in PRD) needs to scan a project directory and generate context-aware CLAUDE.md content. This requires understanding the project's technology stack, directory structure, key configuration files, and common patterns — without executing any project code.

**Decision**: Implement a static analysis engine in Go that reads filesystem metadata and known config files to infer project characteristics. No code execution, no AST parsing — purely file-based heuristic analysis.

**Architecture**:
```
internal/analyzer/
├── analyzer.go              # Orchestrator: scan directory -> ProjectProfile
├── detector_language.go     # Language detection (file extensions, counts)
├── detector_framework.go    # Framework detection (config file signatures)
├── detector_tooling.go      # Tooling detection (linter, formatter, CI configs)
├── detector_structure.go    # Directory structure classification
├── profile.go               # ProjectProfile struct (analysis output)
└── rules/
    ├── rules.go             # Rule engine interface
    ├── rules_go.go          # Go project best practices
    ├── rules_react.go       # React/Next.js best practices
    ├── rules_python.go      # Python best practices
    ├── rules_rust.go        # Rust best practices
    └── rules_general.go     # Cross-language best practices

internal/generator/
├── generator.go             # CLAUDE.md content assembly from ProjectProfile + rules
├── templates/               # Go text/template files for each section
├── scorer.go                # Quality scoring (rule compliance check)
└── optimizer.go             # Optimization suggestions
```

**Detection Heuristics**:

| Signal File | Inferred Stack |
|-------------|---------------|
| `go.mod` | Go (extract module path, Go version) |
| `package.json` + `next.config.*` | Next.js |
| `package.json` + `vite.config.*` | Vite + React/Vue |
| `Cargo.toml` | Rust |
| `pyproject.toml` / `requirements.txt` | Python |
| `Dockerfile` | Containerized deployment |
| `.github/workflows/` | GitHub Actions CI |
| `tsconfig.json` | TypeScript |
| `tailwind.config.*` | Tailwind CSS |
| `wails.json` | Wails desktop app |

**Quality Scoring Criteria**:
- Command section present (how to build/test/run)
- Directory structure documented
- Technology stack specified
- Coding conventions defined
- No contradictory rules
- Appropriate length (not too verbose, not too sparse)

**Consequences**:
- (+) Fast analysis (filesystem reads only, no code execution)
- (+) Safe (no risk of executing malicious code from scanned projects)
- (+) Extensible (add new detectors/rules without changing core logic)
- (+) Works offline (no AI API calls required for basic generation)
- (-) Heuristic-based detection has false positives/negatives
- (-) Cannot understand project-specific business logic or architecture intent
- Mitigation: Generated content is always presented for user review and editing

---

### ADR-016: Distribution via Package Managers (Scoop / Homebrew / WinGet)

**Status**: Accepted (2026-02-27)

**Context**: PRD F9 requires zero-cost distribution across Windows, macOS, and Linux. GitHub Releases provides the binary hosting baseline, but developers expect `scoop install` / `brew install` / `winget install` for seamless installation and updates.

**Decision**: Maintain package manager manifests in the repository, auto-generated by CI/CD on release. GitHub Releases is the single source of truth for binaries.

**Architecture**:
```
lurus-switch/
├── deploy/
│   ├── scoop/
│   │   └── lurus-switch.json       # Scoop manifest (auto-updated by CI)
│   ├── homebrew/
│   │   └── lurus-switch.rb          # Homebrew formula (auto-updated by CI)
│   └── winget/
│       └── LurusTech.LurusSwitch.yaml  # WinGet manifest (submitted to winget-pkgs)

.github/workflows/
└── release.yaml                     # Build → GitHub Release → update manifests
```

**Release Pipeline**:
```
git tag v1.0.0
    ↓
GitHub Actions (release.yaml)
    ├── Build: Wails build for windows/amd64, darwin/amd64, darwin/arm64, linux/amd64
    ├── Sign: Code signing (macOS notarization, Windows Authenticode if available)
    ├── Upload: Attach binaries to GitHub Release
    ├── Scoop: Update hash + version in deploy/scoop/lurus-switch.json, commit
    ├── Homebrew: Update hash + version in deploy/homebrew/lurus-switch.rb, commit
    └── WinGet: Generate manifest, submit PR to microsoft/winget-pkgs (manual review)
```

**Scoop Manifest Structure**:
```json
{
  "version": "1.0.0",
  "architecture": { "64bit": { "url": "https://github.com/.../lurus-switch-windows-amd64.zip", "hash": "sha256:..." } },
  "bin": "lurus-switch.exe",
  "checkver": "github",
  "autoupdate": { "architecture": { "64bit": { "url": "https://github.com/.../lurus-switch-windows-amd64.zip" } } }
}
```

**Consequences**:
- (+) Users install with one command on any platform
- (+) Auto-update via package manager mechanisms (scoop update, brew upgrade)
- (+) Zero hosting cost (GitHub Releases + community package repos)
- (+) Discoverability via package manager search
- (-) WinGet requires Microsoft review (1-2 week delay)
- (-) Homebrew tap requires separate GitHub repo (hanmahong5-arch/homebrew-tap)
- (-) Must maintain CI workflows for manifest generation

---

## 3. Data Architecture / 数据架构

### 3.1 Database Schema Map

```
PostgreSQL (CNPG Cluster)
├── lurus_api schema
│   ├── users              # User accounts
│   ├── tokens             # API tokens
│   ├── channels           # LLM provider channels
│   ├── logs               # API call logs
│   ├── tenants            # Multi-tenant (planned)
│   └── ...
│
├── lucrum schema
│   ├── users              # NextAuth.js compatible
│   ├── userPreferences    # User settings & defaults
│   ├── userDrafts         # Auto-saved draft recovery
│   ├── stocks             # Stock metadata (~5,000 A-shares)
│   ├── sectors            # Industry sector classifications
│   ├── stock_sector_mapping  # Stock-sector relationships
│   ├── kline_daily        # Historical OHLCV (indexed: symbol+date)
│   ├── strategyHistory    # User-saved strategies
│   ├── validation_cache   # Cached validation results
│   └── data_update_log    # Data refresh history
│
├── identity schema
│   └── (Zitadel managed)
│
├── billing schema
│   └── (planned)
│
└── webmail schema
    ├── accounts           # Email accounts
    ├── messages           # Email messages
    ├── folders            # Mailbox folders
    ├── contacts           # Address book
    └── ...
```

### 3.2 Caching Strategy

```
Redis
├── db:0 (lurus-api)
│   ├── session:*          # User sessions
│   ├── ratelimit:*        # API rate limiting
│   └── cache:model:*      # Model availability cache
│
├── db:1 (lucrum - production)
│   ├── kline:*            # K-line data cache (1hr TTL)
│   ├── backtest:*         # Backtest result cache (hash key)
│   ├── stock:list         # Stock list cache
│   ├── workflow:*         # Workflow step result cache (per-step TTL)
│   └── crawler:*          # Crawler result cache
│
├── db:2 (rate limiting)
│   └── rl:*               # Global rate limits
│
└── db:3 (lucrum - staging)
    └── (mirrors db:1 structure, isolated data)
```

### 3.3 Event Streaming

```
NATS JetStream
├── LLM_EVENTS stream
│   ├── llm.request.*      # API request events
│   ├── llm.response.*     # API response events
│   └── llm.error.*        # Error events
│
└── LUCRUM_EVENTS stream
    ├── lucrum.backtest.*   # Backtest execution events
    ├── lucrum.strategy.*   # Strategy CRUD events
    ├── lucrum.workflow.*   # Workflow step events
    ├── lucrum.crawler.*    # Crawler discovery events
    └── lucrum.market.*     # Market data events
```

---

## 4. Security Architecture / 安全架构

### 4.1 Authentication Flow

```
User → Zitadel (OIDC) → JWT Token → Service API
                                        ↓
                                   Middleware validates
                                   JWT signature & claims
```

Lucrum-specific: NextAuth.js with email/password + session-based auth.

### 4.2 Network Security

- **External access**: Public IP (43.226.46.164) → Traefik Ingress (TLS termination)
- **Internal communication**: Tailscale VPN (100.x.x.x mesh)
- **Secrets management**: K8s Secrets (not in Git)
- **Container security**: Non-root, read-only filesystem, scratch/alpine base

### 4.3 Data Protection

- Database credentials in K8s Secrets
- API keys stored encrypted in database
- `.env` files gitignored
- DKIM/SPF/DMARC for email authentication
- Zod schema validation on all API inputs

---

## 5. Monitoring & Observability / 监控与可观测性

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Grafana    │    │  Prometheus  │    │    Loki      │
│ (Dashboards) │←───│  (Metrics)   │    │   (Logs)     │
│              │    │              │    │              │
│ grafana.     │    │ prometheus.  │    │ loki.        │
│ lurus.cn     │    │ lurus.cn     │    │ lurus.cn     │
└──────────────┘    └──────────────┘    └──────────────┘
                           ↑                    ↑
                    ┌──────┴──────────────────┴────────┐
                    │        K3s Cluster Nodes           │
                    │  (node-exporter, promtail, etc.)   │
                    └────────────────────────────────────┘

┌──────────────┐    ┌──────────────┐
│   Jaeger     │    │   ArgoCD     │
│ (Tracing)    │    │ (Deployment) │
│              │    │              │
│ jaeger.      │    │ argocd.      │
│ lurus.cn     │    │ lurus.cn     │
└──────────────┘    └──────────────┘
```

---

## 6. Scalability Analysis / 可扩展性分析

### Current Capacity / 当前容量

| Resource | Current Load | Utilization | Bottleneck Risk |
|----------|-------------|-------------|-----------------|
| Master CPU (16C) | lurus-api + Traefik + ArgoCD + Monitoring + Staging | ~45% | Low |
| Master RAM (32GB) | All control plane + services + staging | ~65% | Medium |
| DB CPU (4C) | PostgreSQL + Zitadel | ~30% | Low |
| DB Storage | ~50GB | ~25% of available | Low |
| Worker CPU (2C) | lucrum-web + www + docs | ~70% | **High** |
| Worker RAM (2GB) | Next.js + Vue + VitePress | ~80% | **High** |

### Scaling Strategy / 扩展策略

1. **Short-term**: Staging already moved to master (has headroom); monitor worker closely
2. **Medium-term**: Add another cloud worker node (2-4C) for lucrum-web production
3. **Long-term**: Consider managed PostgreSQL if data grows significantly

---

## 7. Technology Radar / 技术雷达

| Technology | Ring | Service(s) | Rationale |
|-----------|------|------------|-----------|
| Go + Gin | **Adopt** | Platform-wide | Proven, performant, team expertise |
| Next.js 14 (App Router) | **Adopt** | lucrum | Modern React, good DX |
| Bun | **Adopt** | Platform-wide | 10-20x faster than npm |
| Drizzle ORM | **Adopt** | lucrum | Type-safe, lightweight |
| K3s | **Adopt** | Platform-wide | Lightweight K8s, perfect for small cluster |
| Vitest | **Adopt** | lucrum | ESM-native, fast, excellent DX |
| Decimal.js | **Adopt** | lucrum | Financial-grade precision, proven in 680+ tests |
| LangChain/LangGraph | **Adopt** | lucrum | Multi-agent orchestration, mature ecosystem |
| Zustand | **Adopt** | lucrum, switch | Minimal boilerplate, excellent performance |
| React Query | **Adopt** | lucrum | Server state management |
| Wails v2 | **Adopt** | switch | Desktop app framework, stable release |
| react-i18next | **Adopt** | switch | i18n with namespace-based code-splitting (ADR-012) |
| Monaco Editor | **Adopt** | switch | Config preview & editing, VS Code parity |
| go-keyring | **Adopt** | switch | Cross-platform OS keychain access (ADR-014) |
| Tailwind CSS | **Adopt** | lucrum, switch | Utility-first CSS, consistent design system |
| NATS JetStream | **Trial** | Platform-wide | Event streaming, not yet fully utilized |
| Stalwart | **Trial** | webmail | Self-hosted mail, relatively new |
| VitePress | **Adopt** | Platform-wide | Documentation, simple and effective |
| Zitadel | **Assess** | Platform-wide | OIDC provider, complex setup for 2-person team |
| vnpy | **Hold** | lucrum | Python quant framework, consider Go/TS replacement long-term |
| Scoop/Homebrew/WinGet | **Adopt** | switch | Package manager distribution (ADR-016) |

---

## 8. Implementation Patterns & Consistency Rules / 实现模式与一致性规则

> Purpose: Prevent different AI agents from producing conflicting code. Every pattern below is a binding decision — agents MUST follow these, not invent alternatives.

### 8.1 Naming Patterns / 命名模式

#### Database Naming

| Element | Convention | Example |
|---------|-----------|---------|
| Table names | `snake_case`, plural | `users`, `kline_daily`, `stock_sector_mapping` |
| Column names | `snake_case` | `created_at`, `is_st`, `market_cap` |
| Primary key | `id` (serial/uuid) | `id SERIAL PRIMARY KEY` |
| Foreign key column | `<entity>_id` | `user_id`, `sector_id` |
| Index names | `idx_<table>_<column(s)>` | `idx_kline_stock_date` |
| Unique constraint | `uq_<table>_<column>` | `uq_users_email` |
| Timestamps | Always include both | `created_at TIMESTAMP`, `updated_at TIMESTAMP` |
| Boolean columns | `is_` or `has_` prefix | `is_active`, `is_st`, `has_verified` |

#### JSON Field Naming (Cross-Service)

| Service Type | Convention | Rationale |
|-------------|-----------|-----------|
| Go API responses | `snake_case` | Go ecosystem convention, OpenAI-compatible |
| TypeScript API responses | `camelCase` | JavaScript ecosystem convention |
| NATS event payloads | `snake_case` | Infrastructure layer, Go-idiomatic |
| Frontend-to-backend request body | `camelCase` | Originated from JS, backend adapts at boundary |

**Rule**: Each service uses its language-idiomatic convention. Transformation happens at the consumer boundary, NOT at the producer.

#### API Naming

| Element | Convention | Example |
|---------|-----------|---------|
| REST route segments | `kebab-case` | `/api/agent-protocol/threads` |
| Route resource names | plural nouns | `/api/stocks/list`, `/api/strategies/popular` |
| Query parameters | `camelCase` | `?pageSize=20&excludeST=true` |
| Path parameters | `camelCase` | `/api/workflow/:sessionId/step/:stepNum` |
| Go controller functions | `PascalCase` verb+resource | `GetLoginConfig`, `UpdateChannel` |
| TS route handlers | HTTP method exports | `export async function POST(req)` |

#### Code Naming

| Language | Element | Convention | Example |
|----------|---------|-----------|---------|
| TypeScript | Files (components) | `kebab-case.tsx` | `target-selector.tsx` |
| TypeScript | Files (lib/utils) | `kebab-case.ts` | `financial-math.ts` |
| TypeScript | Exports (components) | `PascalCase` | `TargetSelector` |
| TypeScript | Functions | `camelCase` | `getKLineData()` |
| TypeScript | Types/Interfaces | `PascalCase`, no `I` prefix | `BacktestConfig` |
| TypeScript | Hooks | `use<Feature><Action>` | `useStrategyWorkspaceStore` |
| TypeScript | HOCs | `with<Behavior>` | `withErrorBoundary` |
| TypeScript | Zustand selectors | `select<Property>` | `selectWorkspace` |
| TypeScript | Constants | `UPPER_SNAKE_CASE` | `CACHE_TTL`, `MAX_RETRIES` |
| TypeScript | Branded types | `PascalCase` | `UserId`, `StockSymbol`, `Price` |
| Go | Files | `snake_case.go` | `admin_config.go` |
| Go | Exported functions | `PascalCase` verb+noun | `DisableChannel` |
| Go | Structs | `PascalCase` | `ClaudeConfig` |
| Go | Context keys | `ContextKey<Domain><Field>` | `ContextKeyTokenId` |
| Go | Error codes | `ErrorCode<Domain><Detail>` | `ErrorCodeChannelInvalidKey` |

### 8.2 Structure Patterns / 结构模式

#### Go Service Structure (Binding — HTTP Services)

```
<service>/
├── cmd/server/main.go          # Entry point, fast-fail config validation
├── internal/
│   ├── biz/                    # Business logic (interfaces defined here)
│   │   └── service/            # Service functions
│   ├── data/                   # Data access layer
│   │   └── model/              # GORM models
│   ├── server/
│   │   ├── controller/         # HTTP handlers
│   │   ├── middleware/         # Auth, logging, rate-limit
│   │   └── router/            # Route registration
│   ├── pkg/                    # Internal shared utilities
│   │   ├── common/             # Env helpers, logging
│   │   ├── constant/           # Constants & context keys
│   │   ├── dto/                # Data transfer objects
│   │   └── types/              # Domain error types
│   └── lifecycle/              # Init & shutdown hooks
├── migrations/                 # SQL migration files (NNN_description.sql)
├── deploy/k8s/                 # K8s manifests
├── Dockerfile
└── CLAUDE.md
```

**Rule**: Tests use co-located `_test.go` files (Go convention). No separate `tests/` directory.

#### Wails Desktop App Structure (Binding — lurus-switch)

```
lurus-switch/
├── main.go                      # Wails entry point
├── app.go                       # App struct, lifecycle (OnStartup/OnShutdown)
├── bindings_config.go           # Config CRUD for Claude/Codex/Gemini
├── bindings_billing.go          # Billing & subscription queries
├── bindings_mcp.go              # MCP server management
├── bindings_tools.go            # CLI tool detection & lifecycle
├── bindings_claudemd.go         # CLAUDE.md generation & scoring
├── bindings_cost.go             # Cost dashboard data (via lurus-api)
├── bindings_proxy.go            # Proxy detection & configuration
├── bindings_prompt.go           # Prompt library management
├── internal/
│   ├── config/                  # Config data models & file I/O
│   ├── generator/               # CLAUDE.md generation engine
│   ├── analyzer/                # Project structure static analysis (ADR-015)
│   ├── mcp/                     # MCP server management logic
│   ├── tools/                   # CLI tool detection & management
│   ├── cost/                    # Cost data aggregation from lurus-api
│   ├── proxy/                   # Proxy detection & configuration
│   ├── keychain/                # OS keychain integration (ADR-014)
│   ├── validator/               # Config validation
│   └── packager/                # Executable packaging
├── frontend/                    # React 18 + TypeScript
│   ├── src/
│   │   ├── i18n/                # i18n setup + locale JSON files (ADR-012)
│   │   ├── pages/               # Page components
│   │   ├── components/          # Shared UI components
│   │   │   ├── ui/              # Reusable primitives
│   │   │   ├── onboarding/      # Onboarding wizard flow
│   │   │   └── error-boundary.tsx
│   │   ├── stores/              # Zustand state management
│   │   └── lib/                 # Utilities, types, constants
│   ├── wailsjs/                 # Auto-generated Wails bindings (DO NOT EDIT)
│   └── package.json
├── deploy/
│   ├── scoop/                   # Scoop manifest (ADR-016)
│   ├── homebrew/                # Homebrew formula (ADR-016)
│   └── winget/                  # WinGet manifest (ADR-016)
├── wails.json                   # Wails project config
└── CLAUDE.md                    # Service context
```

**Rules**:
- `app.go` ONLY contains `App` struct, `NewApp()`, lifecycle methods, and shared helpers
- Each `bindings_*.go` is a thin wrapper: validate -> delegate to `internal/<domain>` -> return
- Frontend components organized by feature domain, NOT by component type
- `frontend/wailsjs/` is auto-generated by Wails; never hand-edit
- Error boundaries wrap each major page section for graceful degradation

#### TypeScript (Next.js) Structure (Binding)

```
lucrum-web/
├── src/
│   ├── app/
│   │   ├── api/<resource>/route.ts     # API routes
│   │   ├── dashboard/                   # Page routes
│   │   ├── layout.tsx
│   │   └── globals.css
│   ├── components/
│   │   ├── ui/                          # Reusable primitives (button, card, dialog)
│   │   ├── <feature>/                   # Feature-specific components
│   │   │   ├── __tests__/               # Co-located tests
│   │   │   ├── component-name.tsx
│   │   │   └── component-name.tsx
│   │   └── error-boundary.tsx
│   ├── lib/
│   │   ├── backtest/                    # Backtest engine subsystem
│   │   │   └── core/                    # Engine core (errors, financial-math)
│   │   ├── stores/                      # Zustand stores
│   │   ├── db/                          # Drizzle schema & queries
│   │   ├── redis/                       # Cache layer
│   │   ├── types/                       # Centralized type definitions
│   │   │   └── index.ts                 # Barrel export
│   │   └── utils.ts                     # General utilities
│   └── middleware.ts
├── public/
├── drizzle.config.ts
├── vitest.config.ts
└── CLAUDE.md
```

**Rules**:
- Components organized **by feature domain**, NOT by component type
- `ui/` folder ONLY for generic, reusable primitives (no business logic)
- Tests: `__tests__/` co-located in feature directory, named `<subject>.test.ts`
- Types that cross module boundaries → `lib/types/` centralized
- Types internal to one module → co-located with that module

### 8.3 Format Patterns / 格式模式

#### API Response Format (TypeScript Services)

```typescript
// Success — single resource
{ success: true, data: T, meta?: Record<string, unknown>, timestamp: number }

// Success — list with pagination
{ success: true, data: T[], pagination: { page, pageSize, total, totalPages, hasNext, hasPrev }, timestamp: string }

// Error — client error (4xx)
{ success: false, error: string }  // status 400/401/403/404

// Error — server error (5xx)
{ success: false, error: string, details?: string }  // status 500
```

#### API Response Format (Go Services)

- **lurus-api (LLM Gateway)**: OpenAI-compatible format (industry standard, 不改)
- **Other Go services**: Use `{ success, data, error }` wrapper to match platform convention

#### Date/Time Format

| Context | Format | Example |
|---------|--------|---------|
| API response timestamps | ISO 8601 string | `"2026-02-03T10:00:00Z"` |
| Database storage | `TIMESTAMP` | PostgreSQL native |
| Backtest engine internal | Unix timestamp (ms) | `1738540800000` |
| User-facing display | Localized via `Intl.DateTimeFormat` | `"2026年2月3日"` |
| API `timestamp` field | `Date.now()` (ms) or ISO string | Consistent within each endpoint |

**Rule**: API boundaries always use ISO 8601. Unix timestamps only for performance-critical internal processing.

#### Error Code Namespacing

| Service | Prefix | Format | Example |
|---------|--------|--------|---------|
| lucrum-web (backtest) | `BT` | `BT<category><detail>` | `BT100` (validation), `BT200` (data), `BT300` (calc) |
| lucrum-web (other) | `LR` | `LR<category><detail>` | `LR100`, `LR200` |
| lurus-api | `domain:code` | `<domain>:<detail>` | `channel:invalid_key` |

**Rule**: Error codes MUST be unique within their service. Each service owns its error code namespace. Error messages MUST be bilingual (zh + en) with actionable `suggestion` field.

### 8.4 Communication Patterns / 通信模式

#### NATS Event Envelope

All NATS JetStream events use this standard envelope:

```json
{
  "id": "uuid-v4",
  "type": "lucrum.backtest.completed",
  "source": "lucrum-web",
  "time": "2026-02-03T10:00:00Z",
  "data": { }
}
```

- `type`: `<service>.<domain>.<action>` (dot-separated, lowercase)
- `source`: Service name from `lurus.yaml`
- `time`: ISO 8601
- `data`: Event-specific payload, `snake_case` fields

#### State Management (TypeScript)

| State Type | Tool | Example |
|-----------|------|---------|
| Server state (API data) | React Query | Stock lists, backtest results, market data |
| Client state (UI) | Zustand + `immer` + `persist` | Workspace drafts, user preferences |
| Form state | Local `useState` or React Hook Form | Input fields, validation |
| URL state | Next.js searchParams | Page, filters, selected tab |

**Zustand Action Naming**:
- Simple replacement: `set<Property>` → `setUserId(id)`
- Partial update: `update<Property>` → `updateStrategyInput(partial)`
- Add to collection: `add<Item>` → `addFavorite(symbol)`
- Remove from collection: `remove<Item>` → `removeFavorite(symbol)`
- Reset: `reset<Scope>` → `resetWorkspace()`
- Selectors: `select<Property>` → `selectWorkspace`

### 8.5 Process Patterns / 流程模式

#### Error Handling

**Frontend (TypeScript)**:
1. ErrorBoundary wraps feature sections → catches render errors → shows fallback UI
2. API routes: `try/catch` → return `{ success: false, error }` with appropriate HTTP status
3. User-facing errors: Chinese primary, with error code for support reference
4. Console errors: English, with full stack trace for debugging

**Backend (Go)**:
1. Error wrapping: `fmt.Errorf("<context>: %w", err)` at each layer
2. Never swallow errors: `_ = fn()` is forbidden
3. Controller layer: Translate domain errors to HTTP status codes
4. Structured logging: `slog.Error(msg, "error", err, "context", ctx)`

#### Retry & Fallback Strategy

| Operation Type | Retry | Strategy |
|---------------|-------|----------|
| GET (read) | 3x | Exponential backoff: 1s → 2s → 4s |
| POST/PUT/DELETE (write) | No auto-retry | Idempotency not guaranteed for all endpoints |
| Market data fetch | Fallback chain | DB → EastMoney → Sina → Mock data |
| LLM API call | Provider fallback | Primary → Secondary → Tertiary provider |
| Redis cache miss | No retry | Proceed without cache (graceful degradation) |

#### Validation Strategy

| Boundary | Tool | Responsibility |
|----------|------|---------------|
| Frontend form submission | Zod schema | Immediate user feedback (Chinese error messages) |
| API route entry | Zod schema | Defense-in-depth, reject malformed requests |
| Go API entry | Custom validators | Request body & query param validation |
| Internal function calls | **No validation** | Trust upstream-validated data |

**Rule**: Validate at system boundaries ONLY. Internal code trusts that data has been validated upstream. Double-validation wastes CPU and creates maintenance burden.

#### Loading State Pattern

```typescript
// React Query handles loading/error states automatically
const { data, isLoading, error } = useQuery({
  queryKey: ['stocks', filters],
  queryFn: () => fetchStocks(filters),
  staleTime: 60_000,      // 1 min
  retry: 3,
  retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 8000),
});

// Zustand for async operations not covered by React Query
interface AsyncState {
  status: 'idle' | 'loading' | 'success' | 'error';
  error: string | null;
}
```

### 8.6 Enforcement Guidelines / 执行指南

**All AI Agents MUST**:

1. Follow naming conventions in §8.1 exactly — no "creative" alternatives
2. Place new files in the correct directory per §8.2 structure
3. Use the API response wrapper format in §8.3 for all new endpoints
4. Wrap NATS events in the standard envelope (§8.4)
5. Use React Query for server state, Zustand for client state — never mix
6. Validate at boundaries only (§8.5), trust internal data
7. Include bilingual error messages (zh + en) with actionable suggestions
8. Use `Decimal.js` / `FinancialAmount` for ALL monetary calculations — never native JS numbers
9. Use `@/` path alias for all TypeScript imports
10. Never introduce new state management libraries without architecture review

**Anti-Patterns (Forbidden)**:

- `camelCase` database columns or table names
- API endpoints returning raw data without `{ success, data }` wrapper (except lurus-api OpenAI-compat)
- `console.log` for error handling (use structured logging / ErrorBoundary)
- Inline magic numbers (extract to named constants)
- `any` type in TypeScript (use `unknown` + type narrowing)
- `context.Background()` in Go business code (always propagate parent context)
- `_ = fn()` to swallow errors in Go

---

## 9. Project Structure & Boundaries / 项目结构与边界

### 9.1 lucrum-web Complete Directory Structure

```
lucrum-web/
├── src/
│   ├── app/
│   │   ├── layout.tsx                          # Root layout (providers, fonts)
│   │   ├── page.tsx                            # Landing page
│   │   ├── globals.css                         # Tailwind + global styles
│   │   ├── admin/data-updates/page.tsx         # Admin data management
│   │   ├── auth/                               # Auth pages
│   │   │   ├── login/page.tsx
│   │   │   ├── register/page.tsx
│   │   │   ├── forgot-password/page.tsx
│   │   │   ├── reset-password/page.tsx
│   │   │   ├── verify-email/page.tsx
│   │   │   └── error/page.tsx
│   │   ├── dashboard/                          # Protected dashboard pages
│   │   │   ├── page.tsx                        # Main workspace (strategy editor + backtest)
│   │   │   ├── strategy-validation/page.tsx    # Multi-stock validation
│   │   │   ├── advisor/page.tsx                # AI investment advisor
│   │   │   ├── strategies/page.tsx             # Strategy library & discovery
│   │   │   ├── history/page.tsx                # Backtest history
│   │   │   ├── trading/page.tsx                # Paper trading
│   │   │   ├── insights/page.tsx               # Market insights
│   │   │   ├── account/page.tsx                # User account
│   │   │   └── settings/page.tsx               # User settings
│   │   └── api/                                # API routes (see §9.3)
│   │
│   ├── components/                             # UI components (by feature domain)
│   │   ├── ui/                                 # Generic primitives (shadcn/ui)
│   │   │   ├── button.tsx, card.tsx, dialog.tsx, input.tsx, select.tsx
│   │   │   ├── table.tsx, tabs.tsx, badge.tsx, tooltip.tsx, progress.tsx
│   │   │   └── accordion.tsx, checkbox.tsx, command.tsx, popover.tsx, ...
│   │   ├── strategy-editor/                    # Strategy creation & editing
│   │   │   ├── strategy-input.tsx              # Natural language input
│   │   │   ├── ai-strategy-assistant.tsx       # AI code generation
│   │   │   ├── parameter-editor.tsx            # Parameter tuning
│   │   │   ├── code-preview.tsx                # Generated code display
│   │   │   ├── backtest-panel.tsx              # Backtest execution & results
│   │   │   ├── backtest-basis-panel.tsx        # Backtest config details
│   │   │   ├── enhanced-trade-card.tsx         # Trade detail cards
│   │   │   ├── strategy-templates.tsx          # Template selector
│   │   │   ├── strategy-guide-card.tsx         # Step-by-step guide
│   │   │   ├── auto-save-indicator.tsx         # Save status indicator
│   │   │   ├── draft-history-panel.tsx         # Draft recovery
│   │   │   ├── parameter-boundary-panel.tsx    # Parameter constraints
│   │   │   ├── parameter-info-dialog.tsx       # Parameter help
│   │   │   └── __tests__/                      # Feature tests
│   │   ├── strategy-validation/                # Multi-stock validation
│   │   │   ├── target-selector.tsx             # Sector/stock selector
│   │   │   ├── stock-multi-selector.tsx        # Multi-stock picker
│   │   │   ├── config-panel.tsx                # Validation config
│   │   │   ├── stock-ranking.tsx               # Ranking table
│   │   │   ├── result-summary.tsx              # Aggregate metrics
│   │   │   ├── return-distribution.tsx         # Distribution chart
│   │   │   ├── signal-details.tsx              # Signal breakdown
│   │   │   ├── signal-timeline.tsx             # Timeline visualization
│   │   │   └── __tests__/
│   │   ├── advisor/                            # AI advisor feature
│   │   │   ├── advisor-chat.tsx                # Chat interface
│   │   │   ├── mode-selector.tsx               # Agent/school selection
│   │   │   ├── philosophy-selector.tsx         # Investment philosophy
│   │   │   ├── master-agent-cards.tsx          # Master persona cards
│   │   │   ├── debate-view.tsx                 # Bull/bear debate
│   │   │   └── alert-panel.tsx                 # Market alerts
│   │   ├── backtest/                           # Shared backtest components
│   │   │   ├── target-selector.tsx             # Stock/sector selector
│   │   │   ├── result-dashboard.tsx            # Results overview
│   │   │   ├── diagnostic-panel.tsx            # Debug diagnostics
│   │   │   └── sensitivity-analysis.tsx        # Parameter sensitivity
│   │   ├── dashboard/                          # Dashboard chrome
│   │   │   ├── dashboard-layout.tsx            # Main layout wrapper
│   │   │   ├── dashboard-header.tsx            # Top header
│   │   │   ├── nav-header.tsx                  # Navigation
│   │   │   └── data-status-panel.tsx           # Data freshness status
│   │   ├── charts/kline-chart.tsx              # K-line candlestick chart
│   │   ├── trading/                            # Trading components
│   │   │   ├── symbol-selector.tsx
│   │   │   ├── orderbook-panel.tsx
│   │   │   └── indicator-quick-panel.tsx
│   │   ├── landing/                            # Landing page sections
│   │   ├── settings/                           # Settings panels
│   │   ├── auth/                               # Auth UI (risk disclaimer)
│   │   ├── providers/session-provider.tsx       # NextAuth session
│   │   └── error-boundary.tsx                  # Global error boundary
│   │
│   ├── hooks/                                  # Custom React hooks
│   │   ├── use-kline-data.ts                   # K-line data fetching
│   │   ├── use-market-data.ts                  # Real-time market data
│   │   ├── use-streaming-chat.ts               # SSE chat streaming
│   │   ├── use-advisor-preferences.ts          # Advisor settings
│   │   ├── use-saved-strategies.ts             # Strategy persistence
│   │   ├── use-user-workspace.ts               # Workspace management
│   │   ├── use-broker.ts                       # Broker connection
│   │   └── use-websocket.ts                    # WebSocket connection
│   │
│   ├── lib/                                    # Core business logic
│   │   ├── backtest/                           # Backtest engine (680+ tests)
│   │   │   ├── engine.ts                       # Main backtest engine
│   │   │   ├── statistics.ts                   # 30+ metric calculations
│   │   │   ├── signal-scanner.ts               # Buy/sell signal detection
│   │   │   ├── lot-size.ts                     # 100-lot constraint
│   │   │   ├── transaction-costs.ts            # Commission & slippage
│   │   │   ├── market-status.ts                # Market calendar
│   │   │   ├── symbol-info.ts                  # Stock metadata
│   │   │   ├── diagnostics.ts                  # Engine diagnostics
│   │   │   ├── db-kline-provider.ts            # DB data source
│   │   │   ├── kline-persister.ts              # Data persistence
│   │   │   ├── types.ts                        # Domain types
│   │   │   ├── core/
│   │   │   │   ├── financial-math.ts           # Decimal.js wrapper
│   │   │   │   ├── errors.ts                   # BT error codes
│   │   │   │   ├── interfaces.ts               # Engine contracts
│   │   │   │   └── validators.ts               # Input validation
│   │   │   └── __tests__/                      # 15+ test files
│   │   ├── advisor/                            # AI advisor subsystem
│   │   │   ├── agent/                          # Agent orchestration
│   │   │   │   ├── agent-orchestrator.ts       # Multi-agent routing
│   │   │   │   ├── analyst-agents.ts           # 4 analyst agents
│   │   │   │   ├── researcher-agents.ts        # 3 researcher agents
│   │   │   │   └── master-agents.ts            # 4 master personas
│   │   │   ├── philosophies/                   # 7 investment schools
│   │   │   ├── context-builder.ts              # Context assembly
│   │   │   ├── prediction/alert-generator.ts   # Market alerts
│   │   │   └── reaction/debate-engine.ts       # Debate mode
│   │   ├── data-service/                       # Market data abstraction
│   │   │   ├── sources/
│   │   │   │   ├── eastmoney.ts                # Primary data source
│   │   │   │   ├── eastmoney-sector.ts         # Sector data
│   │   │   │   ├── eastmoney-institutional.ts  # Institutional data
│   │   │   │   └── sina.ts                     # Fallback source
│   │   │   ├── cache.ts                        # Data caching
│   │   │   ├── circuit-breaker.ts              # Resilience
│   │   │   ├── retry.ts                        # Retry logic
│   │   │   ├── validators.ts                   # Data validation
│   │   │   └── logger.ts                       # Data service logging
│   │   ├── workflow/                           # Workflow orchestration
│   │   │   ├── workflow-manager.ts             # Session lifecycle
│   │   │   ├── step-executor.ts                # Step execution
│   │   │   ├── cache-strategy.ts               # Step result caching
│   │   │   └── workflows/strategy-workflow.ts  # Strategy pipeline
│   │   ├── crawler/                            # Strategy discovery
│   │   │   ├── sources/github-crawler.ts       # GitHub search
│   │   │   ├── popularity-scorer.ts            # Scoring algorithm
│   │   │   ├── strategy-converter.ts           # vnpy format conversion
│   │   │   └── scheduler.ts                    # Cron scheduling
│   │   ├── stores/                             # Zustand state stores
│   │   │   ├── strategy-workspace-store.ts     # Main workspace state
│   │   │   ├── workflow-store.ts               # Workflow state
│   │   │   └── trading-store.ts                # Trading state
│   │   ├── db/                                 # Database layer
│   │   │   ├── schema.ts                       # Drizzle schema definition
│   │   │   ├── queries.ts                      # Query helpers
│   │   │   └── index.ts                        # DB connection
│   │   ├── redis/                              # Cache layer
│   │   │   ├── client.ts                       # Redis connection
│   │   │   └── index.ts
│   │   ├── cache/                              # Hybrid cache (Redis + memory)
│   │   │   ├── hybrid-cache.ts
│   │   │   └── cache-keys.ts
│   │   ├── auth/                               # Authentication
│   │   │   ├── auth.ts                         # NextAuth config
│   │   │   ├── with-user.ts                    # Auth middleware helper
│   │   │   ├── email-verification.ts
│   │   │   └── reset-token.ts
│   │   ├── agent/                              # LangGraph agent protocol
│   │   │   ├── graphs/advisor-graph.ts
│   │   │   ├── stores/thread-store.ts
│   │   │   └── tools/                          # Agent tools
│   │   ├── broker/                             # Broker abstraction
│   │   │   ├── interfaces.ts
│   │   │   ├── broker-factory.ts
│   │   │   └── adapters/mock-broker.ts
│   │   ├── types/                              # Centralized type defs
│   │   │   ├── index.ts                        # Barrel export
│   │   │   ├── auth.ts                         # Auth types + branded
│   │   │   └── market.ts                       # Market types + branded
│   │   ├── strategy/                           # Strategy utilities
│   │   ├── strategy-templates/                 # Built-in templates
│   │   ├── trading/                            # Trading utilities
│   │   ├── investment-context/                 # Investment framework
│   │   ├── risk/risk-manager.ts                # Risk management
│   │   ├── services/history-service.ts         # History operations
│   │   ├── cron/daily-updater.ts               # Scheduled tasks
│   │   ├── utils/trading-calendar.ts           # Trading calendar
│   │   └── utils.ts                            # General utilities
│   │
│   ├── __tests__/setup.ts                      # Global test setup
│   └── middleware.ts                           # Next.js middleware (auth)
│
├── public/                                     # Static assets
├── drizzle.config.ts                           # Drizzle ORM config
├── vitest.config.ts                            # Test config
├── next.config.js                              # Next.js config
├── tailwind.config.ts                          # Tailwind config
├── tsconfig.json                               # TypeScript config
├── package.json                                # Dependencies (bun)
├── deploy/k8s/                                 # K8s manifests
│   ├── production/web-deployment.yaml
│   └── staging/web-deployment.yaml
├── Dockerfile                                  # Multi-stage build
└── CLAUDE.md                                   # Service context
```

### 9.2 Epic → Directory Mapping / 史诗到目录的映射

| Epic | Primary Directories | Key Files |
|------|-------------------|-----------|
| **E1: Real Data Backtest** | `lib/backtest/`, `lib/data-service/`, `lib/db/` | `engine.ts`, `db-kline-provider.ts`, `sources/eastmoney.ts` |
| **E2: Quality & Reliability** | `lib/backtest/__tests__/`, `components/**/__tests__/` | All `*.test.ts` files |
| **E3: Strategy Library & Discovery** | `lib/crawler/`, `lib/strategy-templates/`, `app/api/strategies/` | `github-crawler.ts`, `popularity-scorer.ts` |
| **E4: Advanced Analysis** | `components/backtest/`, `lib/backtest/statistics.ts` | `sensitivity-analysis.tsx`, `result-dashboard.tsx` |
| **E5: Paper Trading** | `lib/broker/`, `components/trading/`, `lib/trading/` | `broker-factory.ts`, `mock-broker.ts` |
| **E6: AI Advisor Evolution** | `lib/advisor/`, `lib/agent/`, `components/advisor/` | `agent-orchestrator.ts`, `debate-engine.ts` |

### 9.3 API Route Map / API 路由映射

| Route | Methods | Epic | Purpose |
|-------|---------|------|---------|
| `/api/backtest` | POST | E1 | Single-stock backtest execution |
| `/api/backtest/sector` | POST | E1 | Sector-based batch backtest |
| `/api/backtest/multi-stocks` | POST | E1 | Custom multi-stock backtest |
| `/api/backtest/unified` | POST | E1 | Unified backtest entry point |
| `/api/stocks/list` | GET | E1 | Paginated stock list |
| `/api/stocks/search` | GET | E1 | Stock search (symbol/name) |
| `/api/stocks/favorites` | GET, POST | E1 | User favorite stocks |
| `/api/market/kline` | GET | E1 | K-line data (DB → API fallback) |
| `/api/market/quote` | GET | E1 | Real-time quotes |
| `/api/market/indices` | GET | E1 | Market indices |
| `/api/market/flow` | GET | E6 | Capital flow data |
| `/api/market/status` | GET | E1 | Market open/close status |
| `/api/strategy/generate` | POST | E1 | AI strategy code generation |
| `/api/strategy/optimize` | POST | E4 | Strategy optimization |
| `/api/strategies/popular` | GET | E3 | Popular strategies list |
| `/api/strategies/popular/[id]` | GET | E3 | Strategy detail |
| `/api/strategies/trending` | GET | E3 | Trending strategies |
| `/api/advisor/chat` | POST | E6 | AI advisor chat (SSE) |
| `/api/advisor/debate` | POST | E6 | Bull/bear debate |
| `/api/workflow` | POST, GET | E1 | Workflow session management |
| `/api/workflow/[sessionId]` | GET, DELETE | E1 | Session lifecycle |
| `/api/workflow/[sessionId]/step/[n]` | POST | E1 | Step execution |
| `/api/history` | GET, POST | E4 | Strategy save/load |
| `/api/history/backtests` | GET | E4 | Backtest history |
| `/api/data/fetch` | POST | E1 | Manual data fetch trigger |
| `/api/data/update` | POST | E1 | Data update trigger |
| `/api/data/status` | GET | E1 | Data freshness status |
| `/api/data/institutional` | GET | E6 | Institutional data |
| `/api/cron/crawl-strategies` | POST | E3 | Crawler trigger |
| `/api/cron/init` | POST | E1 | Data initialization |
| `/api/auth/[...nextauth]` | ALL | Cross | NextAuth handler |
| `/api/auth/verify-email` | POST | Cross | Email verification |
| `/api/auth/reset-password` | POST | Cross | Password reset |
| `/api/backend/[...path]` | ALL | Cross | Proxy to Go backend |
| `/api/agent-protocol/**` | ALL | E6 | LangGraph agent protocol |

### 9.4 Architectural Boundaries / 架构边界

#### Layer Boundaries (Strict)

```
┌─────────────────────────────────────────────────────────┐
│  Transport Layer (app/api/, app/dashboard/)              │
│  - Receives HTTP requests                                │
│  - Validates input (Zod)                                 │
│  - Calls business logic                                  │
│  - Returns formatted response                            │
│  ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ │
│  FORBIDDEN: Direct DB queries, Redis calls, or           │
│  external API calls from route handlers.                 │
│  Must delegate to lib/ layer.                            │
├─────────────────────────────────────────────────────────┤
│  Business Logic Layer (lib/)                             │
│  - Core algorithms (backtest engine, advisor, crawler)   │
│  - Orchestration (workflow manager, agent router)        │
│  - Domain types and validation                           │
│  ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ │
│  FORBIDDEN: Import from app/ or components/.             │
│  Must not depend on Next.js request/response types.      │
├─────────────────────────────────────────────────────────┤
│  Data Layer (lib/db/, lib/redis/, lib/data-service/)     │
│  - Database queries (Drizzle ORM)                        │
│  - Cache operations (Redis)                              │
│  - External API calls (Eastmoney, Sina, GitHub)          │
│  ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ │
│  FORBIDDEN: Business logic here. Data layer only         │
│  fetches, stores, and transforms data format.            │
├─────────────────────────────────────────────────────────┤
│  UI Layer (components/, hooks/)                          │
│  - React components, custom hooks                        │
│  - State management (Zustand, React Query)               │
│  - User interactions and display logic                   │
│  ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ │
│  FORBIDDEN: Direct DB/Redis/external API calls.          │
│  Must use API routes (/api/) for all data operations.    │
└─────────────────────────────────────────────────────────┘
```

#### Import Dependency Rules

```
app/api/        → CAN import from: lib/
app/dashboard/  → CAN import from: components/, hooks/, lib/stores/
components/     → CAN import from: hooks/, lib/stores/, lib/types/, ui/
hooks/          → CAN import from: lib/types/
lib/            → CAN import from: lib/ (internal), NOT from app/ or components/
lib/stores/     → CAN import from: lib/types/
```

#### Cross-Service Boundaries

| Boundary | Communication | Protocol |
|----------|--------------|----------|
| lucrum-web → lurus-api | HTTP proxy (`/api/backend/[...path]`) | REST, API key auth |
| lucrum-web → PostgreSQL | Direct (Drizzle ORM, `lucrum` schema ONLY) | TCP, connection pool |
| lucrum-web → Redis | Direct (`db:1` production, `db:3` staging) | TCP |
| lucrum-web → External APIs | HTTP client (data-service layer) | HTTPS, rate-limited |
| lucrum-web → NATS | Publish events (future) | NATS protocol |

**Schema Isolation Rule**: lucrum-web MUST only access `lucrum` schema. Cross-schema queries to `lurus_api`, `identity`, or `webmail` are FORBIDDEN. Use the `/api/backend/` proxy for lurus-api data.

### 9.5 Data Flow / 数据流

#### Backtest Execution Flow

```
User Input (strategy + target + config)
    ↓
API Route (/api/backtest)
    ↓ validates via Zod
Business Logic (lib/backtest/engine.ts)
    ↓ requests K-line data
Data Layer (lib/backtest/db-kline-provider.ts)
    ├─→ PostgreSQL (lucrum.kline_daily) ──→ HIT: return data
    ├─→ Eastmoney API (fallback) ──→ HIT: return + persist to DB
    ├─→ Sina API (second fallback)
    └─→ Mock generator (last resort, clearly labeled)
    ↓
Engine processes with Decimal.js
    ↓ produces
BacktestResult (30+ metrics, trades, equity curve)
    ↓
API Response ({ success, data, meta: { dataSource } })
    ↓
UI Display (components/strategy-editor/backtest-panel.tsx)
```

#### AI Advisor Flow

```
User Message + History
    ↓
API Route (/api/advisor/chat)
    ↓
Agent Orchestrator (lib/advisor/agent/agent-orchestrator.ts)
    ├─→ Selects relevant agents (analysts + researchers + master)
    ├─→ Builds context (lib/advisor/context-builder.ts)
    └─→ Manages token budget
    ↓
LLM API (via lurus-api proxy)
    ↓ SSE stream
Response streamed to client
    ↓
UI Display (components/advisor/advisor-chat.tsx)
```

### 9.6 New Feature Placement Guide (lucrum-web) / 新功能放置指南

When adding new features to lucrum-web, follow this decision tree:

| New Code Type | Place In | Example |
|--------------|----------|---------|
| New API endpoint | `src/app/api/<resource>/route.ts` | `/api/portfolio/route.ts` |
| New page | `src/app/dashboard/<feature>/page.tsx` | `dashboard/portfolio/page.tsx` |
| New feature UI | `src/components/<feature>/` | `components/portfolio/` |
| New business logic | `src/lib/<domain>/` | `lib/portfolio/` |
| New data source | `src/lib/data-service/sources/` | `sources/tushare.ts` |
| New Zustand store | `src/lib/stores/` | `stores/portfolio-store.ts` |
| New React hook | `src/hooks/` | `hooks/use-portfolio.ts` |
| New DB table | `src/lib/db/schema.ts` (add to existing) | — |
| New branded type | `src/lib/types/` (add to existing) | — |
| New error codes | Extend domain-specific error file | `lib/<domain>/errors.ts` |
| New UI primitive | `src/components/ui/` | Only if reusable across 3+ features |
| Tests | `__tests__/` co-located with source | `<feature>/__tests__/` |

### 9.7 lurus-switch Component Architecture / lurus-switch 组件架构

#### Frontend Component Tree

```
frontend/src/
├── i18n/
│   ├── i18n.ts                          # Init: lng='zh-CN', ns=['common','dashboard',...]
│   └── locales/
│       ├── zh-CN/                       # Default locale (all keys MUST exist here)
│       │   ├── common.json              # Nav, buttons, generic errors, status labels
│       │   ├── dashboard.json           # Tool dashboard, status cards
│       │   ├── config.json              # Config editor (Claude/Codex/Gemini tabs)
│       │   ├── claudemd.json            # CLAUDE.md generator & scorer
│       │   ├── mcp.json                 # MCP server manager
│       │   ├── cost.json                # Cost dashboard, charts, budget
│       │   ├── onboarding.json          # Onboarding wizard steps
│       │   └── settings.json            # App settings, proxy, theme
│       └── en/                          # Secondary locale (mirrors zh-CN keys)
│
├── pages/
│   ├── DashboardPage.tsx                # Tool cards + quick actions + quota widget
│   ├── ConfigPage.tsx                   # Visual config editor (tabs per tool)
│   ├── ClaudeMdPage.tsx                 # Smart CLAUDE.md generator
│   ├── McpPage.tsx                      # MCP server manager
│   ├── CostPage.tsx                     # Cost dashboard (data from lurus-api)
│   ├── PromptPage.tsx                   # Prompt library
│   ├── SettingsPage.tsx                 # App settings (proxy, theme, language, updates)
│   └── AccountPage.tsx                  # Billing & subscription
│
├── components/
│   ├── ui/                              # Reusable primitives (button, card, input, etc.)
│   ├── layout/
│   │   ├── Sidebar.tsx                  # Main navigation sidebar
│   │   ├── AppLayout.tsx                # Root layout with sidebar + content area
│   │   └── PageHeader.tsx               # Consistent page header
│   ├── onboarding/
│   │   ├── OnboardingWizard.tsx         # Multi-step wizard container
│   │   ├── StepToolSelect.tsx           # Step 1: Select tools to manage
│   │   ├── StepToolDetect.tsx           # Step 2: Auto-detect installed tools
│   │   ├── StepProxyConfig.tsx          # Step 3: Proxy setup (optional, GFW-aware)
│   │   ├── StepApiKeys.tsx              # Step 4: API key input (stored via keychain)
│   │   └── StepComplete.tsx             # Step 5: Summary + go to dashboard
│   ├── config/
│   │   ├── ClaudeConfigEditor.tsx       # Claude Code settings.json form editor
│   │   ├── CodexConfigEditor.tsx        # Codex config.toml form editor
│   │   ├── GeminiConfigEditor.tsx       # Gemini CLI settings form editor
│   │   ├── ConfigPreview.tsx            # Monaco Editor preview pane
│   │   ├── ConfigSnapshotManager.tsx    # Save/restore/compare snapshots
│   │   └── PresetTemplateSelector.tsx   # Quick-start / security / performance presets
│   ├── claudemd/
│   │   ├── ProjectScanner.tsx           # Directory picker + scan progress
│   │   ├── GeneratedPreview.tsx         # Generated CLAUDE.md preview + edit
│   │   ├── QualityScoreRing.tsx         # Quality score visualization
│   │   └── OptimizationSuggestions.tsx  # Actionable improvement suggestions
│   ├── mcp/
│   │   ├── McpServerList.tsx            # Server list with status badges
│   │   ├── McpServerForm.tsx            # Add/edit server (form replaces JSON)
│   │   ├── McpCatalog.tsx              # Browse community servers
│   │   └── McpHealthPanel.tsx           # Server health & logs
│   ├── cost/
│   │   ├── CostTrendChart.tsx           # Daily/weekly/monthly cost chart
│   │   ├── ModelBreakdown.tsx           # Cost by model/provider
│   │   ├── BudgetAlertConfig.tsx        # Budget threshold settings
│   │   └── SavingSuggestions.tsx         # Model downgrade / cache recommendations
│   ├── tools/
│   │   ├── ToolCard.tsx                 # Tool status card (version, health, actions)
│   │   └── ToolInstaller.tsx            # Install/update/uninstall flow
│   ├── proxy/
│   │   ├── ProxyDetector.tsx            # Auto-detect system proxy
│   │   └── ProxyConfigForm.tsx          # Manual proxy configuration
│   └── error-boundary.tsx               # Global error boundary with fallback UI
│
├── stores/
│   ├── settingsStore.ts                 # Language, theme, proxy config (persisted)
│   ├── toolStore.ts                     # Detected tools, versions, health status
│   ├── configStore.ts                   # Active config editor state
│   ├── onboardingStore.ts               # Onboarding wizard progress
│   └── costStore.ts                     # Cost data cache
│
└── lib/
    ├── wails.ts                         # Typed wrappers around Wails runtime calls
    ├── constants.ts                     # App-wide constants
    └── types.ts                         # Shared TypeScript types
```

#### i18n Integration Pattern

All components MUST use the `useTranslation` hook. No hardcoded UI strings.

```typescript
// Correct pattern
import { useTranslation } from 'react-i18next';

function ToolCard({ tool }: Props) {
  const { t } = useTranslation('dashboard');
  return (
    <Card>
      <h3>{t('toolCard.title', { name: tool.name })}</h3>
      <Badge>{t(`toolCard.status.${tool.status}`)}</Badge>
    </Card>
  );
}

// FORBIDDEN: hardcoded strings
function ToolCard({ tool }: Props) {
  return <h3>{tool.name} 配置</h3>;  // BAD: untranslatable
}
```

#### Error Boundary Strategy

```
AppLayout
├── ErrorBoundary (page-level, catches render errors in entire page)
│   ├── DashboardPage
│   │   ├── ErrorBoundary (section-level, per tool card)
│   │   │   └── ToolCard
│   │   └── ErrorBoundary (section-level, quota widget)
│   │       └── QuotaWidget
│   ├── ConfigPage
│   │   ├── ErrorBoundary (editor section)
│   │   │   └── ConfigEditor
│   │   └── ErrorBoundary (preview section)
│   │       └── ConfigPreview
│   └── CostPage
│       └── ErrorBoundary (chart section, API may fail)
│           └── CostTrendChart
```

**Rules**:
- Page-level error boundary: shows "page load failed" with retry button
- Section-level error boundary: shows inline error message, rest of page remains functional
- Error fallback UI always shows error code + user-actionable suggestion (in current locale)
- Wails binding call failures caught at the store/hook level, NOT in components

#### Onboarding Flow State Machine

```
[start] → StepToolSelect → StepToolDetect → StepProxyConfig → StepApiKeys → StepComplete → [dashboard]
               │                                    │
               │ (skip if no tools selected)        │ (skip if not in China / no proxy needed)
               └────────────────────────────────────┘
```

- Progress persisted in `onboardingStore` (Zustand + persist middleware)
- User can quit and resume at any step
- Completed onboarding sets `settingsStore.onboardingCompleted = true` (never show again)

#### lurus-switch Data Flow / 数据流

```
Frontend Component
    ↓ calls Wails binding
bindings_*.go (thin wrapper)
    ↓ delegates to
internal/<domain>/ (business logic)
    ↓ reads/writes
Config files on disk (Claude settings.json, Codex config.toml, etc.)
    OR
OS Keychain (API keys via internal/keychain)
    OR
lurus-api HTTP (cost data, billing status)
    OR
Local filesystem (project scanning for CLAUDE.md generator)
```

#### lurus-switch New Feature Placement Guide

| New Code Type | Backend (Go) | Frontend (React) |
|--------------|-------------|-----------------|
| New Wails binding | `bindings_<domain>.go` | Auto-generated in `wailsjs/` |
| New business logic | `internal/<domain>/` | N/A |
| New page | N/A | `pages/<Feature>Page.tsx` |
| New feature UI | N/A | `components/<feature>/` |
| New Zustand store | N/A | `stores/<feature>Store.ts` |
| New translations | N/A | `i18n/locales/{zh-CN,en}/<ns>.json` |
| New UI primitive | N/A | `components/ui/` (reusable across 3+ features) |
| New config model | `internal/config/` | Types in `lib/types.ts` |
| Tests (Go) | Co-located `_test.go` | N/A |
| Tests (React) | N/A | `components/<feature>/__tests__/` |
