---
date: 2026-02-02
regenerated: 2026-02-27
author: Anita (via BMAD Gap Analysis)
framework: BMAD v6.0.0-Beta.5
scope: lurus-switch service assessment across all 4 BMAD phases
---

# BMAD Gap Analysis & Improvement Roadmap
# BMAD 差距分析与改进路线图

---

## Executive Summary / 执行摘要

对 Lurus Switch 服务基于 BMAD 4 阶段方法论进行全面审查的 **第三轮更新 (2026-02-27)**。本轮重大进展：**完成了 lurus-switch PRD v2.0**，将产品从"无需求定义的内部工具"提升为有明确市场定位和商业模式的桌面产品。同时，深度代码审查发现了多项技术债务：God Object (app.go 1337行)、死代码、缺失的 i18n、无错误边界等。

### Overall Maturity Score / 整体成熟度评分

| BMAD Phase | Initial Score | Previous Score | Current Score | Grade | Change |
|-----------|---------------|----------------|---------------|-------|--------|
| Phase 1: Analysis (分析) | 45/100 | 70/100 | 80/100 | B+ | +10 |
| Phase 2: Planning (规划) | 35/100 | 65/100 | 78/100 | B | +13 |
| Phase 3: Solutioning (方案) | 70/100 | 82/100 | 82/100 | B+ | +0 |
| Phase 4: Implementation (实施) | 55/100 | 75/100 | 60/100 | C+ | -15 |
| **Overall / 总分** | **51/100** | **73/100** | **75/100** | **B** | **+2** |

> **Note**: Phase 4 score decreased for lurus-switch specifically due to newly discovered technical debt (God Object, dead code, missing error boundaries). The PRD and analysis improvements offset this at the overall level.

---

## Phase 1: Analysis Gaps / 分析阶段差距

### 1.1 Product Brief / 产品简报

| Item | Previous | Current | Status |
|------|----------|---------|--------|
| Product vision statement | ❌ Missing | ✅ Generated | `product-brief.md` created and regenerated |
| User personas | ❌ Missing | ✅ Defined | 3 personas documented with needs |
| Success metrics | ❌ Missing | ✅ Defined | North Star + 12 KPIs with baselines |
| Competitive analysis | ❌ Missing | ✅ Generated | 4 competitors analyzed |
| Revenue model | ❌ Missing | ✅ Documented | Internal tool + 3 future options |

**Remaining Gaps**:
- 📋 Quarterly product review cadence not established
- 📋 Product brief not yet reviewed by full team

### 1.2 Market Research / 市场研究

| Item | Previous | Current | Status |
|------|----------|---------|--------|
| Domain research | ❌ Missing | ⚠️ Partial | Competitive analysis done, no deep market research |
| Technical research | ⚠️ Partial | ✅ Documented | Tech stack decisions with ADRs (11 total) |
| User research | ❌ Missing | ⚠️ Minimal | Team-only usage, no formal feedback |

**Recommendation**:
- Consider lightweight user feedback mechanism (usage analytics)
- Domain research can be deferred (2-person team, internal tool)

### 1.3 Code Quality Analysis (NEW - 2026-02-27) / 代码质量分析

深度代码审查发现以下技术债务:

| # | Finding | Severity | Category | Detail |
|---|---------|----------|----------|--------|
| CQ1 | God Object: `app.go` | Critical | Maintainability | 1337 行, 40+ 重复 CRUD 方法, 违反 SRP |
| CQ2 | Dead Code: `ClaudePage.tsx` | Medium | Dead Code | 组件存在但从未在路由中引用 |
| CQ3 | Non-functional Settings | High | UX Debt | 主题切换/语言切换/数据管理 UI 存在但无实际功能 |
| CQ4 | No i18n Framework | High | Localization | 界面中英文混用, 无 i18n 框架, 字符串硬编码在组件中 |
| CQ5 | No Onboarding Flow | Medium | UX | 新用户无引导, 直接进入空白仪表盘 |
| CQ6 | Missing Billing Features | Medium | Feature Gap | 无取消订阅 UI, 支付状态无轮询确认 |
| CQ7 | No React Error Boundaries | High | Reliability | 任何组件异常导致全应用白屏崩溃 |

**Impact Assessment**:
- CQ1 (God Object) 直接阻碍新功能开发效率, 每次修改 app.go 都有高回归风险
- CQ4 (No i18n) 阻碍中国开发者市场拓展, 这是 PRD 定义的核心用户群
- CQ7 (No Error Boundaries) 在桌面应用中是 P0 级可靠性问题

---

## Phase 2: Planning Gaps / 规划阶段差距

### 2.1 PRD (Product Requirements Document)

| Item | Previous | Current | Status |
|------|----------|---------|--------|
| Formal PRD (lucrum) | ❌ Missing | ✅ Generated | `prd-lucrum.md` with 6 user journeys |
| **Formal PRD (lurus-switch)** | **❌ Missing** | **✅ Created (2026-02-27)** | **`prd.md` v2.0 — 10 feature groups, 3 milestones, 3 user personas** |
| Functional requirements (switch) | ❌ None | ✅ F1-F10 | 50+ requirements across 10 feature groups |
| Non-functional requirements (switch) | ❌ None | ✅ 4 NFR categories | Performance, reliability, security, compatibility |
| User personas (switch) | ❌ None | ✅ 3 personas | AI-Assisted Developer, Chinese Developer, Team Lead |
| User flows (switch) | ❌ None | ✅ 4 flows | First-time setup, configure tool, generate CLAUDE.md, monitor costs |
| Monetization model (switch) | ❌ None | ✅ Freemium 3-tier | Free / Pro (¥49/月) / Team (¥149/月) |
| Distribution strategy (switch) | ❌ None | ✅ 5 channels | GitHub Releases, Scoop, WinGet, Homebrew, npm |
| Success metrics (switch) | ❌ None | ✅ 6 KPIs | Downloads, WAU, config saves, conversion, NPS, stars |

**Resolved Gaps**:
- ~~📋 No PRD for lurus-switch~~ → **RESOLVED**: `prd.md` v2.0 created 2026-02-27

**Remaining Gaps**:
- 📋 PRDs for other services (lurus-api, lurus-webmail) not yet created
- 📋 API documentation (OpenAPI spec) still missing
- 📋 lurus-switch PRD needs Epics decomposition (in progress)

### 2.2 UX Design / UX 设计

| Item | Previous | Current | Status |
|------|----------|---------|--------|
| Design system | ✅ Exists | ✅ Active | `docs/DESIGN_SYSTEM.md` for lucrum-web |
| UX specification | ❌ Missing | ⚠️ Partial | Implicit in PRD user journeys |
| Responsive design spec | ⚠️ Partial | ✅ Implemented | Mobile card view below 768px |
| Accessibility spec | ⚠️ Partial | ✅ Improved | ARIA labels, keyboard nav, WCAG targets |

**Remaining Gaps**:
- 📋 Formal UX specification still needed for complex flows
- 📋 Component library documentation

---

## Phase 3: Solutioning Gaps / 方案阶段差距

### 3.1 Architecture / 架构

| Item | Previous | Current | Status |
|------|----------|---------|--------|
| Architecture document | ⚠️ Partial | ✅ Comprehensive | `architecture.md` with 11 ADRs |
| Architecture Decision Records | ⚠️ 1 ADR | ✅ 11 ADRs | ADR-001 to ADR-011 covering all major decisions |
| System context diagram | ❌ Missing | ✅ ASCII diagram | System boundary + infrastructure topology |
| Data flow diagram | ❌ Missing | ✅ Schema map | Database, cache, and event streaming documented |
| Security architecture | ⚠️ Implicit | ✅ Documented | Auth flow, network security, data protection |
| Technology radar | ❌ Missing | ✅ Generated | 15 technologies rated (Adopt/Trial/Assess/Hold) |

**Remaining Gaps**:
- 📋 Visual architecture diagrams (Excalidraw/draw.io) for presentation
- 📋 Capacity planning spreadsheet

### 3.2 Epics & Stories / 史诗与用户故事

| Item | Previous | Current | Status |
|------|----------|---------|--------|
| Epic definition (lucrum) | ❌ Missing | ⚠️ Informal | Q2 roadmap has 5 epics in plan.md |
| **Epic definition (lurus-switch)** | **❌ Missing** | **⏳ In Progress** | **Epics being created from PRD v2.0** |
| User stories | ❌ Missing | ⚠️ Implicit | PRD FRs can be decomposed to stories |
| Sprint planning | ❌ Missing | ✅ Active | 3 sprints planned in doc/plan.md |
| Backlog grooming | ❌ Missing | ⚠️ Partial | plan.md populated but no formal backlog tool |

**Remaining Gaps**:
- 📋 Formal epics document for lurus-lucrum (`epics-lucrum.md`)
- 📋 lurus-switch epics (`epics.md`) — in progress, derived from PRD v2.0
- 📋 Story estimation and velocity tracking

---

## Phase 4: Implementation Gaps / 实施阶段差距

### 4.1 Code Quality / 代码质量

| Item | Previous | Current | Assessment |
|------|----------|---------|------------|
| Code style consistency | ✅ Good | ✅ Good | CLAUDE.md enforces standards |
| Type safety | ✅ Good | ✅ Good | TypeScript strict mode, Zod validation |
| Error handling | ✅ Good | ✅ Good | Structured error codes (BT1XX-BT9XX) |
| Financial precision | ✅ Excellent | ✅ Excellent | Decimal.js, 680+ tests verifying |
| Component architecture | ✅ Good | ✅ Good | React.memo, virtual scroll, hooks |
| New features | N/A | ✅ Added | Workflow system, strategy crawler, hybrid cache |

#### 4.1.1 lurus-switch Code Quality Issues (NEW - 2026-02-27)

| Item | Status | Assessment |
|------|--------|------------|
| Backend architecture | ❌ God Object | `app.go` is 1337 lines with 40+ repetitive CRUD methods; violates SRP, needs decomposition into service layer |
| Dead code | ⚠️ Present | `ClaudePage.tsx` exists in `frontend/src/pages/` but is never referenced in routing |
| Error boundaries | ❌ Missing | No React Error Boundaries; any component crash causes full app white-screen |
| i18n | ❌ Missing | No i18n framework; UI strings hardcoded in components; Chinese/English mixed inconsistently |
| Onboarding | ❌ Missing | No first-time user guidance; app opens to empty dashboard |
| Settings functionality | ❌ Non-functional | Theme switch, language switch, data management UI elements exist but are wired to no-ops |
| Billing completeness | ⚠️ Partial | Subscription creation works; cancel subscription and payment polling not implemented |

### 4.2 Testing / 测试

| Service | Previous Coverage | Current Coverage | Target | Gap |
|---------|------------------|-----------------|--------|-----|
| lurus-lucrum (backtest/) | ~15% | **85%+ (680 tests)** | 80% | **✅ Exceeded** |
| lurus-lucrum (components) | ~5% | ~25% | 50% | -25% |
| lurus-api | ~50% | ~50% | 70% | -20% |
| lurus-switch (Go backend) | ~40% | ~40% | 60% | -20% |
| lurus-switch (React frontend) | ~0% | ~0% | 50% | -50% |
| lurus-webmail | ~5% | ~10% | 50% | -40% |
| lurus-www | 0% | 0% | 30% | -30% |

**Major Achievement**: Backtest engine coverage went from ~15% to 85%+ (680 tests). This was the highest-risk area identified in the initial gap analysis.

**Remaining Gaps**:
- Priority 1: Component tests for lucrum-web (strategy editor, ranking)
- Priority 2: lurus-api coverage improvement
- Priority 3: lurus-webmail basic test suite

### 4.3 CI/CD Pipeline / CI/CD 流水线

| Item | Previous | Current | Status |
|------|----------|---------|--------|
| Automated build | ✅ Working | ✅ Working | GitHub Actions |
| Automated tests in CI | ⚠️ Partial | ⚠️ Partial | Backtest tests comprehensive, CI step pending |
| Docker image build | ✅ Working | ✅ Working | Multi-stage, public dir fix applied |
| ArgoCD sync | ✅ Working | ✅ Working | GitOps |
| Staging environment | ❌ Missing | ✅ Deployed | `lucrum-staging` namespace |
| Rollback procedure | ⚠️ Manual | ⚠️ Manual | ArgoCD supports it, not documented |

**Remaining Gaps**:
- 📋 CI mandatory test step for all services
- 📋 Documented rollback procedure in `doc/runbook/`
- 📋 Automated staging deployment on PR

### 4.4 Documentation / 文档

| Item | Previous | Current | Assessment |
|------|----------|---------|------------|
| Root README.md | ✅ Basic | ✅ Good | Quick start guide |
| CLAUDE.md (root) | ✅ Good | ✅ Updated | Company standards |
| CLAUDE.md (lucrum-web) | ✅ Excellent | ✅ Updated | Dev workflow |
| doc/process.md | ✅ Active | ✅ Active | 10KB development log |
| doc/plan.md | ❌ Empty | ✅ Populated | Q1-Q3 roadmap with sprints |
| doc/structure.md | ❌ Missing | ⚠️ Partial | Architecture.md serves as substitute |
| doc/develop-guide.md | ❌ Missing | ⚠️ Partial | CLAUDE.md + project-context.md serve as substitute |
| BMAD artifacts | ❌ None | ✅ 5 documents | project-context, product-brief, prd, architecture, gap-analysis |
| API documentation | ❌ None | ⚠️ Partial | API surface documented in PRD, no OpenAPI spec |

---

## Risk Assessment Matrix / 风险评估矩阵

| # | Risk | Category | Severity | Likelihood | Priority | Previous |
|---|------|----------|----------|-----------|----------|----------|
| R1 | Worker node resource exhaustion (2C/2GB) | Infrastructure | High | High | **P0** | P0 (unchanged) |
| R2 | ~~No staging environment~~ | ~~Process~~ | ~~High~~ | ~~Medium~~ | ~~P0~~ | **Resolved** |
| R3 | ~~Low test coverage on financial engine~~ | ~~Quality~~ | ~~Critical~~ | ~~Medium~~ | ~~P0~~ | **Resolved (85%+)** |
| R4 | ~~Empty planning documents~~ | ~~Process~~ | ~~Medium~~ | ~~Already true~~ | ~~P1~~ | **Resolved** |
| R5 | Single PostgreSQL instance (no HA) | Infrastructure | Critical | Low | **P1** | P1 (unchanged) |
| R6 | ~~No formal PRD (lurus-switch)~~ | ~~Process~~ | ~~Medium~~ | ~~Medium~~ | ~~P1~~ | **Resolved (2026-02-27)** |
| R7 | Component test coverage < 50% | Quality | Medium | Already true | **P1** | Unchanged |
| R8 | No CI mandatory test step | Process | Medium | Already true | **P1** | Elevated |
| R9 | Office node reliability for messaging | Infrastructure | Medium | Medium | **P2** | P2 (unchanged) |
| R10 | No API documentation (OpenAPI) | DX | Medium | Already true | **P2** | P2 (unchanged) |
| R11 | IP reputation for self-hosted mail | Operations | Medium | High | **P2** | P2 (unchanged) |
| R12 | Crawler rate limiting / GitHub API | Operations | Low | Medium | **P3** | Unchanged |
| **R13** | **God Object `app.go` (1337 lines)** | **Quality** | **High** | **Already true** | **P0** | **New (2026-02-27)** |
| **R14** | **No React Error Boundaries (lurus-switch)** | **Reliability** | **High** | **Already true** | **P1** | **New (2026-02-27)** |
| **R15** | **No i18n framework (lurus-switch)** | **UX** | **High** | **Already true** | **P1** | **New (2026-02-27)** |
| **R16** | **Non-functional settings UI (lurus-switch)** | **UX Debt** | **Medium** | **Already true** | **P1** | **New (2026-02-27)** |
| **R17** | **Dead code: `ClaudePage.tsx` never routed** | **Quality** | **Low** | **Already true** | **P2** | **New (2026-02-27)** |
| **R18** | **Missing billing features (cancel/polling)** | **Feature Gap** | **Medium** | **Already true** | **P2** | **New (2026-02-27)** |
| **R19** | **No onboarding flow (lurus-switch)** | **UX** | **Medium** | **Already true** | **P2** | **New (2026-02-27)** |

**Resolved Risks**: R2 (staging), R3 (test coverage), R4 (empty plans), R6 (no PRD) - 4 out of original risks resolved.
**New Risks**: R13-R19 — 7 new risks identified from lurus-switch deep code review (2026-02-27).

---

## Improvement Roadmap / 改进路线图

### Completed Since Initial Assessment / 已完成

1. ✅ **Product brief generated** → `product-brief.md`
2. ✅ **Architecture document generated** → `architecture.md` (11 ADRs)
3. ✅ **Project context generated** → `project-context.md`
4. ✅ **PRD created for lurus-lucrum** → `prd-lucrum.md` (6 journeys, 60+ FRs)
5. ✅ **Gap analysis generated** → `bmad-gap-analysis.md`
6. ✅ **Financial engine tests** → 680+ tests, 85%+ coverage
7. ✅ **Staging environment deployed** → `lucrum-staging` namespace
8. ✅ **doc/plan.md populated** → Q1-Q3 roadmap with sprints
9. ✅ **Workflow system launched** → Multi-step strategy development
10. ✅ **Strategy crawler launched** → GitHub discovery pipeline
11. ✅ **Hybrid cache implemented** → Redis + in-memory
12. ✅ **PRD created for lurus-switch** → `prd.md` v2.0 (2026-02-27) — 10 feature groups, 3 milestones, freemium model
13. ✅ **Deep code analysis for lurus-switch** → 7 technical debt items identified (CQ1-CQ7)

### Immediate (This Sprint) / 立即行动 — lurus-switch Focus

1. **Decompose `app.go` God Object** (R13) — Extract into service layer: `internal/app/config_service.go`, `internal/app/tool_service.go`, etc.
2. **Add React Error Boundaries** (R14) — Wrap top-level routes and critical components
3. **Remove dead code** (R17) — Delete `ClaudePage.tsx` or wire it into routing
4. **Create lurus-switch Epics** → `epics.md` from PRD v2.0 (in progress)

### Short-Term (Next Sprint) / 短期

5. **Implement i18n framework** (R15) — `react-i18next` with CN/EN locale files, extract all hardcoded strings
6. **Wire up settings** (R16) — Connect theme toggle, language selector, data management to actual functionality
7. **Add onboarding wizard** (R19) — First-time setup flow per PRD Flow 1
8. **Complete billing features** (R18) — Cancel subscription UI, payment status polling
9. **Add CI mandatory test step** to all GitHub Actions workflows

### Medium-Term (1 Month) / 中期

10. **Implement MVP features per PRD Milestone 1** — F1 (Tool Dashboard), F2 (Visual Config Editor), F3 (Proxy & Network), F4 (Config Persistence)
11. **Create PRDs** for lurus-api and lurus-webmail
12. **Generate OpenAPI specs** for lurus-api
13. **Achieve 60%+ backend test coverage** for lurus-switch
14. **Frontend component tests** for lurus-switch (target 30%+)

### Long-Term (Quarter) / 长期

15. **Implement Milestone 2 features** — F5 (CLAUDE.md Generator), F6 (MCP Manager), F7 (Cost Dashboard)
16. **Distribution pipeline** — Scoop, Homebrew, WinGet manifests + CI/CD
17. **Consider PostgreSQL HA** (CNPG failover testing)
18. **Achieve 70%+ overall test coverage**
19. **Sprint velocity tracking** and estimation

---

## Action Plan: lurus-switch Epics / 行动计划

The PRD v2.0 defines 3 milestones. The Epics document (in creation at `_bmad-output/planning-artifacts/epics.md`) will decompose these into:

| Epic | PRD Milestone | Key Features | Priority | Addresses Risks |
|------|--------------|--------------|----------|----------------|
| E0: Technical Debt Cleanup | Pre-MVP | God Object decomposition, Error Boundaries, Dead code removal, i18n setup | P0 | R13, R14, R15, R17 |
| E1: Core Configuration Manager | Milestone 1 | F1 Tool Dashboard, F2 Visual Config Editor, F4 Config Persistence | P0 | — |
| E2: Proxy & Network | Milestone 1 | F3 Proxy & Network (China developer focus) | P0 | — |
| E3: Onboarding & Settings | Milestone 1 | F1.4 Onboarding Wizard, Settings functionality, Billing completeness | P1 | R16, R18, R19 |
| E4: Smart CLAUDE.md Generator | Milestone 2 | F5 CLAUDE.md Generator (core monetization differentiator) | P1 | — |
| E5: MCP Server Manager | Milestone 2 | F6 MCP Server Manager | P1 | — |
| E6: Cost Dashboard | Milestone 2 | F7 Cost Dashboard (lurus-api integration) | P2 | — |
| E7: Ecosystem & Distribution | Milestone 3 | F8 Prompt Library, F9 Distribution, F10 Team Features | P2 | — |

**Key Dependency**: E0 (Tech Debt) must be completed before E1-E3 to avoid compounding the God Object problem with new feature code.

---

## BMAD Workflow Recommendations / BMAD 工作流建议

Based on the updated gap analysis, recommended next BMAD workflows:

| Order | Workflow | Agent | Purpose | Status |
|-------|----------|-------|---------|--------|
| 1 | `generate-project-context` | BMad Master | Project context | ✅ Done (regenerated) |
| 2 | `create-product-brief` | Mary (Analyst) | Product brief | ✅ Done (regenerated) |
| 3 | `create-prd` (lucrum) | John (PM) | Lucrum PRD | ✅ Done (regenerated) |
| 4 | `create-prd` (lurus-switch) | John (PM) | Switch PRD | ✅ Done (2026-02-27) — `prd.md` v2.0 |
| 5 | `create-architecture` | Winston (Architect) | Architecture doc | ✅ Done (regenerated) |
| 6 | `check-implementation-readiness` | Bob (SM) | Gap analysis | ✅ Done (updated 2026-02-27) |
| 7 | `create-epics-and-stories` (switch) | Bob (SM) | **In Progress** - Break PRD into epics → `epics.md` |
| 8 | `sprint-planning` (switch) | Bob (SM) | **Next** - Generate `sprint-status.yaml` |
| 9 | `create-prd` (api) | John (PM) | PRD for lurus-api |
| 10 | `create-prd` (webmail) | John (PM) | PRD for lurus-webmail |
| 11 | `code-review` | Amelia (Dev) | Adversarial review — focus on `app.go` God Object |

---

## Conclusion / 结论

Lurus 平台自首次 BMAD 评估以来取得了显著进步，整体成熟度从 C- 提升到 B:

**本轮成就 (2026-02-27) / Achievements**:
- 整体成熟度从 **B- (73/100) 提升到 B (75/100)**
- lurus-switch PRD v2.0 完成 — 产品从无需求定义提升为有完整市场定位、商业模式和 50+ 功能需求
- 深度代码审查识别 7 项技术债务 (CQ1-CQ7)，为重构提供明确路线图
- R6 (No PRD for lurus-switch) 风险已解决
- Analysis (Phase 1) 从 70 提升到 80, Planning (Phase 2) 从 65 提升到 78

**关键发现 / Key Findings**:
- `app.go` God Object (1337 lines) 是最大技术风险, 必须在新功能开发前解决
- 缺失 i18n 直接阻碍 PRD 定义的中国开发者目标用户群
- React 无 Error Boundaries 在桌面应用中是可靠性隐患
- 多处 UI 元素 (settings) 无实际功能, 损害用户信任

**下一步重点 / Next Focus**:
1. 完成 Epics 分解 (`epics.md`) 和 Sprint 规划 (`sprint-status.yaml`)
2. E0: 技术债务清理 — God Object 拆分、Error Boundaries、死代码清除
3. E1-E2: MVP 核心功能实现 — 配置管理器 + 代理网络
4. 建立 i18n 基础设施, 统一中文界面
