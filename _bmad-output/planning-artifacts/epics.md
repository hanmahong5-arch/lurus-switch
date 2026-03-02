# Lurus Switch - Epics & Sprint Planning

**Version**: 1.0
**Date**: 2026-02-27
**Aligned PRD**: prd.md v2.0
**Sprint Duration**: 2 weeks

---

## Overview / 概览

本文档将 PRD 中的 3 个 Milestone 分解为 8 个 Epic，每个 Epic 包含用户故事和验收标准。
Sprint 1-3 构成 MVP (Milestone 1)，Sprint 4-6 构成 Milestone 2，Sprint 7-8 构成 Milestone 3。

### Milestone-to-Epic Mapping

| Milestone | Epics | Sprints |
|-----------|-------|---------|
| M1: Core Config Manager | Epic 1, Epic 2, Epic 3 | Sprint 1-3 |
| M2: Smart Optimization | Epic 4, Epic 5, Epic 6 | Sprint 4-6 |
| M3: Ecosystem & Distribution | Epic 7, Epic 8 | Sprint 7-8+ |

### Epic Dependency Graph

```
Epic 1: Foundation ──┬──→ Epic 2: Onboarding & Dashboard
                     │
                     ├──→ Epic 3: Visual Config Editor V2
                     │         │
                     │         ├──→ Epic 4: CLAUDE.md Generator
                     │         │
                     │         └──→ Epic 5: MCP Server Manager
                     │
                     └──→ Epic 6: Cost Dashboard (also depends on lurus-api)
                              │
Epic 7: Distribution ←───────┤  (independent, can start after Epic 3)
                              │
Epic 8: Team & Ecosystem ←───┘  (depends on Epic 4, 5, 6)
```

---

## Epic 1: Foundation Cleanup & Quality / 基础整治与质量提升

**Sprint**: 1 (Week 1-2)
**PRD Refs**: F1 (partial), NFR 6.1-6.3
**Goal**: Establish a clean, stable codebase and working app settings as the foundation for all subsequent features.

### Current State (技术债务清单)

- `app.go` is a God Object (~67 fields, imports 15+ packages, exposes all Wails bindings directly)
- `ClaudePage.tsx` contains legacy UI code that is superseded by `ToolConfigPage.tsx`
- `configStore` has unused types and methods from early prototyping
- Settings page (theme, language, data management) is rendered but non-functional
- No i18n system; Chinese and English strings are hardcoded and mixed
- No error boundaries; unhandled frontend errors crash the entire app

### User Stories

#### S1.1: Remove Dead Code

**As a** developer,
**I want** unused code removed from the codebase,
**So that** the project is easier to understand and maintain.

**Acceptance Criteria**:
- [ ] `ClaudePage.tsx` is deleted; no imports reference it
- [ ] Unused types in `internal/config/store.go` are removed (verified via `go vet` + IDE unused analysis)
- [ ] Unused generator methods (if any) are removed
- [ ] `picoclaw` and `nullclaw` related files are reviewed: delete if no longer in PRD scope, or document why they exist
- [ ] `go build ./...` and `cd frontend && bun run build` both pass with zero warnings related to removed code
- [ ] All tests still pass: `go test ./...`

#### S1.2: Break Up app.go God Object

**As a** developer,
**I want** `app.go` split into focused service facades,
**So that** each Wails binding group has a single responsibility and is testable independently.

**Acceptance Criteria**:
- [ ] `app.go` retains only lifecycle methods (`startup`, `shutdown`, `domReady`) and `GetSystemInfo()`
- [ ] Tool-related bindings extracted to `internal/facade/tool_facade.go` (detect, install, uninstall, update)
- [ ] Config-related bindings extracted to `internal/facade/config_facade.go` (load, save, validate, preview)
- [ ] Billing bindings extracted to `internal/facade/billing_facade.go`
- [ ] MCP, Snapshot, Prompt, Doc, Env, Analytics bindings extracted to respective facade files
- [ ] All facades are registered in `main.go` via Wails `Bind()` option
- [ ] Existing frontend calls updated to use new binding paths (if Wails namespace changes)
- [ ] All tests pass; no behavior regression

#### S1.3: Implement i18n System

**As a** Chinese developer (primary user),
**I want** the entire UI in consistent Chinese with the option to switch to English,
**So that** I have a professional, native-language experience.

**Acceptance Criteria**:
- [ ] i18n library integrated (e.g., `react-i18next` or lightweight alternative)
- [ ] All hardcoded UI strings extracted to locale files: `frontend/src/locales/zh.json`, `frontend/src/locales/en.json`
- [ ] Chinese (`zh`) is the default locale
- [ ] Language switch in Settings page persists choice to app config (via Go backend `appconfig`)
- [ ] No mixed-language UI: every visible string comes from locale files
- [ ] At least 95% of existing UI strings are translated (remaining 5% tracked as TODOs in locale file)

#### S1.4: Fix Non-Functional Settings

**As a** user,
**I want** Settings page controls to actually work,
**So that** I can customize my app experience.

**Acceptance Criteria**:
- [ ] **Theme toggle**: Dark/Light mode switch persists to `appconfig` and applies immediately via CSS class on `<html>`
- [ ] **Language selector**: Switches locale and persists (see S1.3)
- [ ] **Data management**: "Clear cache" button clears snapshot store and temp files; "Export data" exports app config as JSON; "Reset to defaults" resets appconfig with confirmation dialog
- [ ] **Proxy settings**: Reads current proxy from `proxy.ProxyManager` and allows manual override (linked to F3 in later Epic, but basic display works now)
- [ ] **Auto-update toggle**: Enables/disables update check on startup
- [ ] All settings persist across app restart

#### S1.5: Add Error Boundaries

**As a** user,
**I want** individual page errors to be caught gracefully,
**So that** one broken feature does not crash the entire app.

**Acceptance Criteria**:
- [ ] React Error Boundary component created at `frontend/src/components/ErrorBoundary.tsx`
- [ ] Each route/page wrapped in an Error Boundary
- [ ] Error UI shows: what happened, a "retry" button, and an option to copy error details
- [ ] Go backend panics caught by Wails runtime (verify: intentional panic in a binding does not crash the app)
- [ ] Frontend `window.onerror` and `unhandledrejection` caught and logged

---

## Epic 2: Onboarding & Dashboard / 引导流程与仪表盘

**Sprint**: 2 (Week 3-4)
**PRD Refs**: F1.1-F1.4, F3.1, UI 5.2-5.3
**Dependencies**: Epic 1 (clean codebase, working settings, i18n)
**Goal**: Deliver a polished first-run experience and a dashboard that shows tool status at a glance.

### User Stories

#### S2.1: First-Time Setup Wizard (Onboarding)

**As a** new user,
**I want** a guided setup wizard on first launch,
**So that** I can configure my tools and proxy without reading documentation.

**Acceptance Criteria**:
- [ ] Wizard appears on first launch (detected via `appconfig.onboarding_completed` flag)
- [ ] Step 1: Welcome screen with app intro (i18n)
- [ ] Step 2: Auto-detect installed AI CLI tools (Claude Code, Codex, Gemini CLI) and display results
- [ ] Step 3: Proxy configuration — auto-detect system proxy (HTTP_PROXY, HTTPS_PROXY, system settings); if in China region (detected by locale or timezone), prominently offer Clash/V2Ray presets
- [ ] Step 4: Optional API key setup (text inputs with show/hide toggle; stored via OS keychain if available, fallback encrypted file)
- [ ] Step 5: Summary + "Go to Dashboard" button
- [ ] User can skip the wizard at any step; skipped state saved
- [ ] Wizard can be re-triggered from Settings page
- [ ] All steps respect current locale (zh/en)

#### S2.2: Dashboard Redesign with Tool Cards

**As a** user,
**I want** a dashboard showing the status of all my AI CLI tools,
**So that** I can see version, health, and config validity at a glance.

**Acceptance Criteria**:
- [ ] Dashboard displays one `ToolCard` per detected tool (Claude Code, Codex CLI, Gemini CLI)
- [ ] Each `ToolCard` shows: tool name, installed version (or "Not installed"), config validity indicator (green/yellow/red), last config modification time
- [ ] Uninstalled tools show an "Install" button; installed tools show "Configure" and "Update" buttons
- [ ] Tool detection runs in parallel and completes within 3 seconds (NFR 6.1)
- [ ] Dashboard layout is responsive: 3 columns on wide screens, 1 column on narrow
- [ ] Empty state: if no tools detected, show "Get Started" prompt linking to wizard

#### S2.3: Quota Widget on Dashboard

**As a** lurus-api user,
**I want** a quota/balance widget on the dashboard,
**So that** I can see my remaining API credits without navigating elsewhere.

**Acceptance Criteria**:
- [ ] Widget displays current balance and usage percentage if lurus-api credentials are configured
- [ ] If credentials are not configured, widget shows "Connect your account" CTA linking to billing page
- [ ] Widget refreshes on dashboard load and has a manual refresh button
- [ ] Graceful degradation: if lurus-api is unreachable, show "Offline" state with last known value (if cached)
- [ ] Visual: compact card fitting in dashboard sidebar or top bar

#### S2.4: Tool Health Indicators

**As a** user,
**I want** to see if my tool configuration is valid and if the tool can connect to its API,
**So that** I can fix issues before they interrupt my workflow.

**Acceptance Criteria**:
- [ ] Health check runs config validation (format correctness, required fields present)
- [ ] Health check pings the tool's API endpoint if API key is configured (timeout: 5 seconds)
- [ ] Results displayed as status icons on ToolCard: green (all good), yellow (config warning), red (error/unreachable)
- [ ] Clicking the status icon expands a detail panel showing specific issues
- [ ] Health check results cached for 5 minutes; manual refresh available

#### S2.5: Proxy Auto-Detection

**As a** Chinese developer,
**I want** the app to auto-detect my proxy settings,
**So that** I don't need to manually configure them every time.

**Acceptance Criteria**:
- [ ] On startup, detect: `HTTP_PROXY` / `HTTPS_PROXY` env vars, system proxy settings (Windows registry / macOS networksetup / Linux gsettings)
- [ ] Detect common proxy software: Clash (port 7890), V2Ray (port 10808/10809)
- [ ] Display detected proxy in Settings and Dashboard
- [ ] If proxy detected but not configured in tool configs, suggest applying it (non-intrusive notification)
- [ ] User can override auto-detected proxy with manual settings

---

## Epic 3: Visual Config Editor V2 / 可视化配置编辑器 V2

**Sprint**: 3 (Week 5-6)
**PRD Refs**: F2.1-F2.6, F4.1-F4.3
**Dependencies**: Epic 1 (facades, i18n), Epic 2 (dashboard navigation)
**Goal**: Replace raw Monaco editing with form-based editors for common settings while retaining Monaco for advanced/raw mode.

### User Stories

#### S3.1: Form-Based Editor for Claude Code

**As a** user,
**I want** a visual form to edit Claude Code settings,
**So that** I don't need to know the JSON schema to configure it correctly.

**Acceptance Criteria**:
- [ ] Form editor covers: model selection (dropdown), permission flags (checkboxes), sandbox mode (toggle), API key (masked input), custom instructions path
- [ ] Tabs: "Basic" (form) / "Advanced" (Monaco editor showing the raw JSON)
- [ ] Changes in form mode are reflected in Monaco preview in real-time
- [ ] Changes in Monaco mode are reflected in form mode (bidirectional sync)
- [ ] Invalid JSON in Monaco mode shows inline error; form mode prevents invalid states
- [ ] Save button writes to the actual Claude Code settings file path
- [ ] Config backup created automatically before save (F4 prerequisite)

#### S3.2: Form-Based Editor for Codex CLI

**As a** user,
**I want** a visual form to edit Codex CLI TOML config,
**So that** I can configure model, safety level, and approval policies without TOML syntax knowledge.

**Acceptance Criteria**:
- [ ] Form editor covers: model selection, safety level (dropdown: low/medium/high), approval policy (auto/manual/ask), API base URL
- [ ] Tabs: "Basic" (form) / "Advanced" (Monaco with TOML syntax highlighting)
- [ ] Bidirectional sync between form and Monaco
- [ ] TOML validation before save
- [ ] Config backup before save

#### S3.3: Form-Based Editor for Gemini CLI

**As a** user,
**I want** a visual form to edit Gemini CLI settings,
**So that** I can configure extensions and safety settings easily.

**Acceptance Criteria**:
- [ ] Form editor covers: model selection, extension toggles, safety settings, API key
- [ ] Tabs: "Basic" (form) / "Advanced" (Monaco with Markdown highlighting for GEMINI.md)
- [ ] Bidirectional sync between form and Monaco
- [ ] Config backup before save

#### S3.4: Preset Templates

**As a** new user,
**I want** to apply preset configurations for common use cases,
**So that** I get a good starting point without manual tuning.

**Acceptance Criteria**:
- [ ] 4 presets available per tool: "Quick Start" (defaults), "Security First" (restricted permissions), "Performance" (best model, aggressive caching), "Budget" (cheapest model, minimal features)
- [ ] Preset selection shows a diff preview: what will change from current config
- [ ] Applying a preset auto-backs up the current config as a named snapshot
- [ ] Presets defined in `frontend/src/data/presets/` as typed JSON objects (not hardcoded in components)
- [ ] Presets are extensible: adding a new preset requires only a new JSON entry, no code changes

#### S3.5: Config Validation with Actionable Errors

**As a** user,
**I want** clear, actionable error messages when my config is invalid,
**So that** I know exactly what to fix.

**Acceptance Criteria**:
- [ ] Validation runs on every form change (debounced 300ms) and before save
- [ ] Errors displayed inline next to the offending field (form mode) or as gutter markers (Monaco mode)
- [ ] Each error message contains: what is wrong, what is expected, a suggested fix (e.g., "Model 'gpt-4o' is not valid for Claude Code. Did you mean 'claude-sonnet-4-20250514'?")
- [ ] Warnings (non-blocking) are visually distinct from errors (blocking)
- [ ] Validation rules defined in Go backend (`internal/validator/`) to ensure single source of truth

#### S3.6: Config Snapshot Management

**As a** user,
**I want** to save, restore, and compare configuration snapshots,
**So that** I can experiment with settings and roll back if needed.

**Acceptance Criteria**:
- [ ] "Save snapshot" button in config editor; user provides a name (auto-suggested: `<tool>-<date>-<time>`)
- [ ] Snapshot list page shows: name, tool, timestamp, size
- [ ] "Restore" action loads snapshot into editor with a diff preview showing what will change
- [ ] "Compare" action opens side-by-side diff view of two snapshots
- [ ] "Export" downloads snapshot as JSON; "Import" loads from JSON file
- [ ] Maximum 50 snapshots per tool; oldest auto-deleted with warning
- [ ] Snapshots stored in app data directory (`internal/snapshot/store.go`)

---

## Epic 4: Smart CLAUDE.md Generator / 智能 CLAUDE.md 生成器

**Sprint**: 4 (Week 7-8)
**PRD Refs**: F5.1-F5.5
**Dependencies**: Epic 1 (facades, i18n), Epic 3 (editor infrastructure)
**Goal**: Deliver the core "information asymmetry" feature -- automatically generate and optimize project-level instruction files.

### User Stories

#### S4.1: Project Scanner

**As a** developer,
**I want** the app to scan my project directory and detect its tech stack,
**So that** a CLAUDE.md can be generated with accurate, project-specific content.

**Acceptance Criteria**:
- [ ] User selects a project directory via native OS file picker (Wails dialog)
- [ ] Scanner detects: programming languages (by file extensions and config files), frameworks (package.json, go.mod, Cargo.toml, requirements.txt, etc.), directory structure patterns (monorepo, standard layouts), existing CLAUDE.md / GEMINI.md / .codex files
- [ ] Scan completes within 5 seconds for projects up to 10,000 files
- [ ] Scan excludes: `node_modules/`, `.git/`, `vendor/`, `target/`, `dist/`, `build/`
- [ ] Results displayed as a summary card: detected languages, frameworks, project type
- [ ] Go implementation in `internal/docmgr/scanner.go`

#### S4.2: Template Library

**As a** developer,
**I want** curated CLAUDE.md templates organized by framework,
**So that** I can start with a high-quality base and customize from there.

**Acceptance Criteria**:
- [ ] Templates available for: Go (Gin/Echo/Wails), React (Next.js/Vite), Python (FastAPI/Django), Rust, Vue, TypeScript (general)
- [ ] Each template includes: project structure guidance, coding standards, testing requirements, error handling patterns, forbidden patterns
- [ ] Templates stored in `internal/docmgr/templates/` as Go embedded files
- [ ] Template selection UI shows: template name, target framework, preview of content, user rating (local-only for now)
- [ ] User can fork a template and save as custom template

#### S4.3: Quality Scoring

**As a** developer,
**I want** a quality score for my CLAUDE.md file,
**So that** I know how effective it is and where to improve.

**Acceptance Criteria**:
- [ ] Scoring criteria (0-100 scale): completeness (are key sections present?), specificity (does it reference actual project paths/patterns?), clarity (positive instructions vs negative "don't" patterns), length (penalize too short <50 lines or too long >500 lines), structure (proper heading hierarchy, consistent formatting)
- [ ] Score displayed as a ring/gauge chart (`QualityScoreRing` component)
- [ ] Score breakdown shows: category scores with explanations
- [ ] Score updates in real-time as user edits

#### S4.4: Optimization Suggestions

**As a** developer,
**I want** actionable suggestions to improve my CLAUDE.md,
**So that** I can optimize my AI coding assistant's performance.

**Acceptance Criteria**:
- [ ] Suggestions generated based on quality scoring gaps
- [ ] Examples of suggestions: "Replace 'Don't use console.log' with 'Use structured logging via pino'" (positive framing), "Add a testing section specifying your test runner and coverage targets", "Your CLAUDE.md is 600 lines -- consider splitting into root + service-level files", "Add framework-specific patterns (detected: Next.js App Router)"
- [ ] Suggestions ranked by impact (high/medium/low)
- [ ] One-click "Apply" inserts suggested text at the appropriate location
- [ ] Suggestion engine implemented in Go (`internal/docmgr/optimizer.go`) using rule-based analysis (no LLM dependency)

#### S4.5: Multi-Tool Instruction Support

**As a** developer using multiple AI tools,
**I want** to generate GEMINI.md and Codex instructions alongside CLAUDE.md,
**So that** all my tools get optimized project context.

**Acceptance Criteria**:
- [ ] After generating a CLAUDE.md, user can click "Also generate for Gemini/Codex"
- [ ] Adapter layer translates CLAUDE.md patterns to GEMINI.md format and Codex instruction format
- [ ] User can review and edit each generated file independently
- [ ] Export writes all files to the project directory (with backup of existing files)

---

## Epic 5: MCP Server Manager / MCP 服务器管理器

**Sprint**: 5 (Week 9-10)
**PRD Refs**: F6.1-F6.5
**Dependencies**: Epic 1 (facades), Epic 3 (config editor infrastructure)
**Goal**: Replace manual JSON editing of MCP server configurations with a visual manager.

### User Stories

#### S5.1: Visual MCP Configuration

**As a** user,
**I want** a form-based UI to add and edit MCP server configurations,
**So that** I don't need to hand-write JSON paths, arguments, and environment variables.

**Acceptance Criteria**:
- [ ] Form fields: server name, transport type (stdio/sse), command path (with OS file picker for stdio), arguments (tag input), environment variables (key-value editor)
- [ ] Form validates: command exists on PATH or at absolute path, required fields present, no duplicate server names
- [ ] Preview pane shows the resulting JSON that will be written to the tool's config
- [ ] Save writes to Claude Code `settings.json` MCP section and/or Gemini CLI MCP config
- [ ] Edit existing MCP server: loads current config into form
- [ ] Delete MCP server with confirmation

#### S5.2: MCP Server Directory / Catalog

**As a** user,
**I want** to browse popular MCP servers from a catalog,
**So that** I can discover and install useful MCP integrations without searching the web.

**Acceptance Criteria**:
- [ ] Built-in catalog with 20+ curated MCP servers (filesystem, GitHub, database, web search, etc.)
- [ ] Catalog data stored locally in `internal/mcp/catalog.json` (embedded at build time)
- [ ] Each entry shows: name, description, category, install command, popularity indicator
- [ ] "Install" button runs the install command and auto-configures the MCP server entry
- [ ] Search and filter by category
- [ ] Catalog is updatable: app checks for catalog updates from GitHub (optional, with user consent)

#### S5.3: MCP Health Monitoring

**As a** user,
**I want** to see if my configured MCP servers are running and responsive,
**So that** I can troubleshoot connection issues.

**Acceptance Criteria**:
- [ ] For stdio servers: verify the command binary exists and is executable
- [ ] For SSE servers: ping the endpoint URL (HTTP HEAD, timeout 5s)
- [ ] Status indicators on each MCP server card: green (OK), yellow (slow/warning), red (unreachable/missing)
- [ ] "View logs" button shows the last 100 lines of MCP server output (if available from tool's log)
- [ ] Health check runs on page load and is manually refreshable

#### S5.4: Cross-Tool MCP Sync

**As a** user managing both Claude Code and Gemini CLI,
**I want** to sync MCP server configurations between tools,
**So that** I don't need to configure the same server twice.

**Acceptance Criteria**:
- [ ] Sync UI shows: MCP servers configured in Claude Code only, Gemini only, and both
- [ ] User can select servers and click "Sync to Gemini" or "Sync to Claude"
- [ ] Sync translates config format between tools (Claude JSON format vs Gemini format)
- [ ] Conflict resolution: if both tools have the same server name with different configs, show a diff and let user choose
- [ ] Sync is manual (not automatic) to prevent surprises

---

## Epic 6: Cost Dashboard / 成本仪表盘

**Sprint**: 6 (Week 11-12)
**PRD Refs**: F7.1-F7.5
**Dependencies**: Epic 1 (facades, i18n), lurus-api gateway (external dependency)
**Goal**: Provide visibility into AI tool spending through the lurus-api gateway.

### User Stories

#### S6.1: lurus-api Integration

**As a** lurus-api user,
**I want** the desktop app to connect to my lurus-api account,
**So that** I can view my usage data locally.

**Acceptance Criteria**:
- [ ] Settings page has "Connect lurus-api" section: API URL input (default: `https://api.lurus.cn`), API key input
- [ ] Connection test button verifies credentials and displays account info
- [ ] Credentials stored securely (OS keychain preferred, encrypted file fallback)
- [ ] Billing client (`internal/billing/client.go`) handles authentication, request signing, and retry logic
- [ ] Graceful handling of: invalid credentials (clear error message), network errors (offline mode), rate limiting (backoff)

#### S6.2: Usage Charts

**As a** user,
**I want** to see my AI API usage as visual charts,
**So that** I can understand my spending patterns.

**Acceptance Criteria**:
- [ ] Daily usage bar chart (last 30 days)
- [ ] Weekly summary line chart (last 12 weeks)
- [ ] Monthly totals (last 6 months)
- [ ] Breakdown by: model (pie chart), tool (stacked bar)
- [ ] Charts use a lightweight library (e.g., Recharts or Chart.js) -- not a heavy BI framework
- [ ] Data cached locally with 1-hour TTL; manual refresh available
- [ ] Date range selector for custom periods

#### S6.3: Budget Alerts

**As a** user,
**I want** to set daily and monthly budget limits,
**So that** I get warned before overspending.

**Acceptance Criteria**:
- [ ] Settings for: daily budget (USD), monthly budget (USD)
- [ ] Dashboard widget shows: current spend vs budget as progress bar
- [ ] Notification when 80% of budget reached (in-app toast notification)
- [ ] Notification when 100% of budget reached (prominent warning banner)
- [ ] Budget settings persisted in appconfig
- [ ] Historical budget adherence: monthly report showing days over budget

#### S6.4: Optimization Recommendations

**As a** user,
**I want** cost-saving suggestions based on my usage patterns,
**So that** I can reduce spending without sacrificing productivity.

**Acceptance Criteria**:
- [ ] Analyze usage data and generate suggestions: model downgrade (e.g., "Use Haiku for 60% of your queries to save ~$30/month"), prompt caching opportunities, idle tool detection ("Codex CLI has 0 usage this month -- consider disabling")
- [ ] Each suggestion shows: estimated monthly savings, effort to implement (one-click vs manual), impact on quality (low/medium/high)
- [ ] "Apply" button for one-click suggestions (e.g., switch default model in tool config)
- [ ] Suggestions refresh weekly; dismissed suggestions don't reappear for 30 days

---

## Epic 7: Distribution / 分发渠道

**Sprint**: 7 (Week 13-14)
**PRD Refs**: F9.1-F9.5
**Dependencies**: Epic 3 (stable MVP release)
**Goal**: Make Lurus Switch installable through package managers and auto-updatable.

### User Stories

#### S7.1: Scoop Manifest (Windows)

**As a** Windows developer,
**I want** to install Lurus Switch via `scoop install lurus-switch`,
**So that** I can manage it alongside my other dev tools.

**Acceptance Criteria**:
- [ ] Scoop manifest JSON file created at `deploy/scoop/lurus-switch.json`
- [ ] Manifest includes: version, download URL (GitHub Releases), SHA256 hash, bin path, shortcuts
- [ ] `scoop install` successfully installs the app (tested locally)
- [ ] `scoop update lurus-switch` updates to the latest version
- [ ] Manifest auto-generated by CI on release tag push

#### S7.2: Homebrew Formula (macOS/Linux)

**As a** macOS/Linux developer,
**I want** to install Lurus Switch via `brew install lurus-switch`,
**So that** installation and updates are seamless.

**Acceptance Criteria**:
- [ ] Homebrew formula created at `deploy/homebrew/lurus-switch.rb`
- [ ] Formula includes: version, SHA256, dependencies, caveats
- [ ] Tap repository set up: `homebrew-tap` repo in the org
- [ ] `brew install hanmahong5-arch/tap/lurus-switch` works end-to-end
- [ ] Formula auto-updated by CI on release tag push

#### S7.3: Auto-Update Improvements

**As a** user,
**I want** the app to notify me of updates and update itself seamlessly,
**So that** I always have the latest features and fixes.

**Acceptance Criteria**:
- [ ] Update check on startup (respects "disable auto-update" setting from S1.4)
- [ ] Notification shows: current version, new version, changelog summary
- [ ] "Update now" button downloads and applies update (Wails self-update or platform-appropriate method)
- [ ] Update progress bar during download
- [ ] Fallback: if self-update fails, provide direct download link to GitHub Releases
- [ ] Update checker uses `internal/updater/github_checker.go` (already exists, improve reliability)

#### S7.4: GitHub Releases Automation

**As a** maintainer,
**I want** GitHub Releases created automatically on version tag push,
**So that** I don't need to manually build and upload binaries.

**Acceptance Criteria**:
- [ ] GitHub Actions workflow: `.github/workflows/release-switch.yml`
- [ ] Triggered by: push tag `lurus-switch/v*`
- [ ] Builds: Windows x64, macOS x64, macOS arm64, Linux x64
- [ ] Artifacts: `.exe` (Windows), `.app` bundle in `.dmg` (macOS), `.AppImage` or tarball (Linux)
- [ ] Creates GitHub Release with: version tag, auto-generated changelog (from commits since last release), attached binaries
- [ ] Updates Scoop manifest and Homebrew formula with new version + hash

---

## Epic 8: Team & Ecosystem / 团队与生态

**Sprint**: 8+ (Week 15+)
**PRD Refs**: F8.1-F8.4, F10.1-F10.3
**Dependencies**: Epic 4, Epic 5, Epic 6
**Goal**: Build retention through sharing, community content, and team features.

### User Stories

#### S8.1: Config Sharing (Export/Import Packages)

**As a** team lead,
**I want** to export my tool configurations as a shareable package,
**So that** I can distribute standardized settings to my team.

**Acceptance Criteria**:
- [ ] "Export package" creates a ZIP containing: tool configs (API keys stripped), CLAUDE.md template, MCP server configs, preset selections
- [ ] "Import package" applies all configs with a review step (show diff of what will change)
- [ ] Package format is versioned (header includes format version for forward compatibility)
- [ ] Import handles conflicts: if user has existing configs, show merge/replace/skip options per file

#### S8.2: Community Prompt Library

**As a** developer,
**I want** a library of high-quality prompts organized by category,
**So that** I can find effective prompts for common tasks.

**Acceptance Criteria**:
- [ ] Built-in prompt library with 50+ curated prompts in categories: coding, debugging, refactoring, documentation, testing, code review
- [ ] Each prompt shows: title, description, category, word count, preview
- [ ] "Copy" button copies prompt to clipboard
- [ ] "Save to favorites" for quick access
- [ ] User can create custom prompts and organize them in personal collections
- [ ] Search by keyword and filter by category
- [ ] Prompt data stored locally (`internal/promptlib/store.go` already exists)

#### S8.3: Team Cost Reports

**As a** team lead,
**I want** aggregated cost reports across team members,
**So that** I can manage my team's AI tool budget.

**Acceptance Criteria**:
- [ ] Requires lurus-api Team tier subscription
- [ ] Dashboard shows: total team spend, per-member breakdown, per-model breakdown
- [ ] Monthly report exportable as CSV
- [ ] Budget alerts at team level (separate from individual budgets)
- [ ] Data fetched via lurus-api team endpoints

#### S8.4: Prompt Version Control

**As a** developer,
**I want** my custom prompts to have version history,
**So that** I can track changes and revert to previous versions.

**Acceptance Criteria**:
- [ ] Each prompt save creates a version entry (timestamp, content hash)
- [ ] Version history view shows: list of versions with timestamps and diffs
- [ ] "Restore version" replaces current content with historical version
- [ ] Maximum 20 versions per prompt; oldest auto-pruned

---

## Sprint Planning Summary / Sprint 规划总览

### MVP Scope (Sprint 1-3)

| Sprint | Epic | Key Deliverables | Success Criteria |
|--------|------|-----------------|-----------------|
| Sprint 1 | Epic 1: Foundation | Dead code removed, app.go refactored, i18n working, settings functional, error boundaries | `go test ./...` pass, `bun run build` clean, all settings persist across restart |
| Sprint 2 | Epic 2: Onboarding | Setup wizard, dashboard with ToolCards, quota widget, proxy auto-detection | New user can complete setup in <2 minutes; dashboard loads in <2s |
| Sprint 3 | Epic 3: Config Editor V2 | Form editors for 3 tools, presets, validation, snapshots | User can configure any tool without touching raw config files |

### Post-MVP Scope (Sprint 4-8)

| Sprint | Epic | Key Deliverables | External Dependencies |
|--------|------|------------------|-----------------------|
| Sprint 4 | Epic 4: CLAUDE.md Generator | Project scanner, template library, quality scoring, optimization suggestions | None |
| Sprint 5 | Epic 5: MCP Manager | Visual MCP config, server catalog, health monitoring, cross-tool sync | None |
| Sprint 6 | Epic 6: Cost Dashboard | lurus-api integration, usage charts, budget alerts, cost optimization | lurus-api gateway endpoints |
| Sprint 7 | Epic 7: Distribution | Scoop manifest, Homebrew formula, auto-update, CI/CD release | GitHub Actions, Scoop/Homebrew infra |
| Sprint 8+ | Epic 8: Team & Ecosystem | Config sharing, prompt library, team reports | lurus-api Team tier, lurus-identity |

### Risk Register / 风险登记

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Wails v2 → v3 migration during development | High | Pin to Wails v2 for MVP; evaluate v3 after Sprint 3 |
| lurus-api team endpoints not ready for Epic 6/8 | Medium | Implement with mock data first; integrate when API is ready |
| MCP protocol changes (still evolving) | Medium | Abstract MCP config behind an adapter layer; catalog is data-driven |
| i18n coverage incomplete | Low | Track untranslated strings as TODOs; community contributions post-launch |
| Scoop/Homebrew review process delays | Low | Ship GitHub Releases first (Sprint 7 week 1); package managers are secondary channel |

---

## Definition of Done (per Story) / 完成定义

A story is **Done** when ALL of the following are satisfied:

1. **Code complete**: Implementation merged to development branch
2. **Tests pass**: `go test ./...` and `bun run build` pass with no regressions
3. **i18n complete**: All new UI strings have zh and en translations
4. **No hardcoded values**: URLs, ports, timeouts, magic strings extracted to config/constants
5. **Error handling**: All new external calls have error handling with user-friendly messages
6. **Accessibility**: New interactive elements have `aria-label` or equivalent
7. **Verified**: Manual smoke test or automated test proves the feature works end-to-end
8. **Documented**: CLAUDE.md updated if new patterns or conventions introduced

---

## Appendix: Story Estimation Reference

| Size | Points | Typical Scope |
|------|--------|---------------|
| XS | 1 | Config change, single-file fix |
| S | 2 | Single component or function, <100 LOC |
| M | 3 | Feature spanning 2-3 files, ~200-500 LOC |
| L | 5 | Feature spanning multiple packages, ~500-1000 LOC |
| XL | 8 | Major feature or refactor, >1000 LOC, cross-cutting concerns |

| Story | Estimate | Notes |
|-------|----------|-------|
| S1.1 Remove Dead Code | S (2) | Mostly deletion |
| S1.2 Break Up app.go | L (5) | Touches all bindings + frontend calls |
| S1.3 Implement i18n | L (5) | Need to extract all strings |
| S1.4 Fix Settings | M (3) | Settings exist, just need wiring |
| S1.5 Error Boundaries | S (2) | Standard React pattern |
| S2.1 Onboarding Wizard | L (5) | Multi-step UI + proxy detection |
| S2.2 Dashboard Redesign | M (3) | Redesign existing page |
| S2.3 Quota Widget | S (2) | Billing client exists |
| S2.4 Health Indicators | M (3) | New validation + network checks |
| S2.5 Proxy Auto-Detection | M (3) | OS-specific detection logic |
| S3.1 Claude Editor | L (5) | Form + Monaco bidirectional sync |
| S3.2 Codex Editor | M (3) | Similar pattern to S3.1 |
| S3.3 Gemini Editor | M (3) | Similar pattern to S3.1 |
| S3.4 Presets | M (3) | Data-driven templates |
| S3.5 Validation | M (3) | Extend existing validator |
| S3.6 Snapshots | M (3) | Snapshot store exists, add UI |
| S4.1 Project Scanner | L (5) | File system traversal + pattern matching |
| S4.2 Template Library | M (3) | Embedded Go templates |
| S4.3 Quality Scoring | M (3) | Rule-based scoring engine |
| S4.4 Optimization Suggestions | L (5) | Complex rule engine |
| S4.5 Multi-Tool Instructions | M (3) | Adapter/translation layer |
| S5.1 Visual MCP Config | L (5) | Complex form + file system access |
| S5.2 MCP Catalog | M (3) | Curated data + search UI |
| S5.3 MCP Health | M (3) | Process/network checks |
| S5.4 Cross-Tool Sync | M (3) | Config format translation |
| S6.1 lurus-api Integration | M (3) | Billing client exists, add auth flow |
| S6.2 Usage Charts | L (5) | Charting library + data transformation |
| S6.3 Budget Alerts | M (3) | Threshold logic + notifications |
| S6.4 Cost Optimization | L (5) | Analysis engine + apply actions |
| S7.1 Scoop Manifest | S (2) | JSON template + CI step |
| S7.2 Homebrew Formula | S (2) | Ruby template + tap repo |
| S7.3 Auto-Update | M (3) | Improve existing updater |
| S7.4 Release CI | M (3) | GitHub Actions workflow |
| S8.1 Config Sharing | M (3) | ZIP packaging + merge UI |
| S8.2 Prompt Library | M (3) | Store exists, add browsing UI |
| S8.3 Team Reports | L (5) | API integration + charts |
| S8.4 Prompt Versioning | S (2) | Simple version list |

**Total Points**: ~117 across 8 sprints
**Average per Sprint**: ~15 points (feasible for 1-2 developer team)
