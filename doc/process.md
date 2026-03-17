# Process Log — Lurus Switch

## 2026-03-17: 全面重设计 Phase 0–6
Phase 0 Bug 修复：parseIntSafe break fix、serverctl 硬编码凭证→生成随机密钥、MCP ID 碰撞→crypto/rand、CI/CD 完整重写、toolhealth 检查所有条目。
Phase 1 安全：verify.go SHA-256 校验+自更新签名验证。
Phase 2 UX：SetupWizard 错误显示、工具未安装 Banner、代理连通性 Ping、Billing 3 状态 UI、GatewayRequiredGuard、健康红灯修复入口、安装成功 Toast、CLI Runner 路由。
Phase 3 后端：relay 包（types/store/health/cloud）+ bindings_relay.go。前端：RelayPage + relayStore。
Phase 4 后端：gy 包（types/products/launcher）+ bindings_gy.go。前端：GYProductsPage + gyStore。
Phase 5：AccountStatusBadge + StatusBar 集成 + 5min billing 轮询。
Phase 6：doc/structure.md + doc/develop-guide.md（Wails 架构），旧文档加 [ARCHIVED] 头。
Verification: `go build ./... → PASS` | `bun run build → PASS (1587 modules)`

## 2026-03-02: ZeroClaw & OpenClaw Full Support
新增 7 工具支持：ZeroClaw（GitHub Releases 二进制，TOML 配置）+ OpenClaw（npm/bun，JSON 配置）。
新建：config/zeroclaw.go, config/openclaw.go, installer/zeroclaw_installer.go, installer/openclaw_installer.go, generator/zeroclaw_generator.go, generator/openclaw_generator.go + 6 个测试文件。
修改：constants.go（+6 常量）、store.go（+8 方法）、installer.go（7 工具，3 处错误消息）、validator.go（+2 validate 方法）、toolconfig.go（+2 dir 函数+默认模板）、envmgr/manager.go（+zeroclaw/openclaw case）、bindings_config.go（+16 绑定方法）。
前端：configStore.ts（ActiveTool union +2）、ToolCard.tsx（toolMeta +2）、DashboardPage/SetupWizard/ToolConfigPage（TOOL_ORDER +2，QUICK_REF +2）、i18n en/zh（+2 key）。
Verification: `go build ./... -> PASS` | `go vet ./... -> PASS` | `go test -run "TestNew*/TestZeroClaw*/TestOpenClaw*" ./... -> PASS` | `bun run build -> PASS` | `bun run test:run -> 19 passed`

## 2026-02-28: Sprint 3 — Visual Config Editor V2 (21pts)
S3.6: snapshot auto-prune (max 20) + ClearTool. S3.5: Claude URL/timeout, Codex sk- prefix/history bounds/baseUrl, Gemini AIza prefix. S3.4: preset package (4 presets × 3 tools) + 6 bindings. S3.1/S3.2/S3.3: ClaudeConfigForm / CodexConfigForm / GeminiConfigForm + PresetSelector + ValidationPanel + shared SwitchField/SelectField/TagInput. ToolConfigPage: Form/Text toggle button.
Verification: `go build ./... -> PASS` | `go test ./internal/... -> 17 packages PASS` | `npx tsc --noEmit -> PASS`


## 2026-02-28: Scenario-Based Test Redesign
新增 4 个场景测试文件（scenarios_test.go）：appconfig(12 tests)、analytics(10 tests)、proxy(10 tests)、toolhealth(9 tests)，共 41 个场景。
场景覆盖：首次启动向导流程、Wizard 完成/重置、并发快速操作、配置文件损坏恢复、设置多次修改只保留最后值、日期窗口过滤、API Key 轮换/清除、所有健康状态迁移。
Verification: `go test ./... -> 15 packages PASS`

## 2026-02-28: Test Coverage Boost
新增 4 个测试文件：appconfig(11)、analytics(11)、updater/npm_checker(18)、github_checker+self_updater(17)、toolhealth 补充(14)。
覆盖率：analytics 0%→85%，appconfig 0%→77%，updater 32.5%→60.8%，toolhealth 57.4%→91.2%。
Verification: `go test ./... -> 15 packages PASS`

## 2026-02-27: Sprint 2 — Onboarding & Dashboard (21pts)
Implemented 5 stories: S2.5 Proxy Auto-Detection, S2.4 Tool Health Indicators, S2.3 Quota Widget, S2.1 Setup Wizard, S2.2 Dashboard Redesign.
Code review found 25 issues (2 Critical, 6 High, 9 Medium, 8 Low). Fixed 15 issues including XSS, missing types, stale closures, i18n gaps, concurrent port probing.
Verification: `go test ./... -> PASS` | `npx tsc --noEmit -> PASS` | `bun run test:run -> 19/19 PASS`
Remaining: 8 Low severity issues (cosmetic) deferred to backlog.
