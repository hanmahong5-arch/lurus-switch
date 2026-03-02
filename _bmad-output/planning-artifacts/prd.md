# Lurus Switch - Product Requirements Document (PRD)

**Version**: 2.0
**Date**: 2026-02-27
**Status**: Active

---

## 1. Product Vision / 产品愿景

**Lurus Switch** is the unified configuration & optimization hub for AI CLI tools.

**一句话定位**: 让开发者用一个桌面应用管理所有 AI 编码工具的配置、成本和最佳实践。

**核心价值主张**:
- **统一管理**: Claude Code / Codex / Gemini CLI 一站式配置
- **信息差变现**: 将隐藏的最佳实践、成本优化策略打包为可操作的功能
- **零摩擦分发**: 10MB 桌面应用，GitHub Releases + 包管理器零成本覆盖全平台

---

## 2. Problem Statement / 问题陈述

### 2.1 Core Pain Points

| # | Pain Point | Severity | Current Solutions |
|---|-----------|----------|-------------------|
| P1 | AI CLI 工具配置分散（3+ 格式: JSON/TOML/MD），互不兼容 | High | 手动编辑，无统一工具 |
| P2 | CLAUDE.md 编写缺少指导，质量差异导致 5-11% 性能差距 | High | 社区博客、试错 |
| P3 | MCP 服务器配置复杂（手写 JSON，路径/版本易错） | High | 手动 JSON 编辑 |
| P4 | 跨工具 API 成本不可见，月均超支 $50-120 | Medium | 各平台独立查看 |
| P5 | 中国开发者代理配置困难（GFW + OAuth 认证） | High | 手动设置 HTTPS_PROXY |
| P6 | 工具安装/更新需要不同包管理器（npm/cargo/bun） | Medium | 手动执行命令 |

### 2.2 Market Gap (信息差)

现有工具生态存在明确空白:

1. **无统一配置管理器** -- AionUi (开源早期) 最接近，但仅管理 AI Agent session，不管配置文件
2. **CLAUDE.md 优化是蓝海** -- CodeWithClaude.net 提供基础生成器，但缺少代码库分析和性能量化
3. **成本可观测性缺失** -- 开发者日均 AI 工具花费 $6+，无统一仪表盘
4. **MCP GUI 配置为零** -- 所有工具仍要求手写 JSON
5. **中国开发者被忽略** -- 主流工具无内置代理支持，国内模型 (DeepSeek/Kimi K2) 无缝切换需求大

---

## 3. Target Users / 目标用户

### 3.1 Primary: AI-Assisted Developer (AI 辅助开发者)

**画像**: 同时使用 2-3 个 AI CLI 工具的全栈开发者
- 日常使用 Claude Code + Codex 或 Gemini CLI
- 关心成本但没时间逐个平台查看用量
- 知道 CLAUDE.md 重要但不知道怎么写好
- 在 macOS/Windows/Linux 上工作

**Jobs to be Done**:
1. 快速安装和配置 AI CLI 工具
2. 管理多个工具的 API key 和代理设置
3. 优化 AI 工具配置以获得最佳编码体验
4. 控制 AI 工具使用成本

### 3.2 Secondary: Chinese Developer (中国开发者)

**画像**: 需要翻墙使用 AI CLI 工具的国内开发者
- 依赖 Clash/V2Ray 代理
- 对 OAuth 认证流程中的代理配置感到困惑
- 愿意在国内模型 (DeepSeek) 和海外模型间切换
- 对中文界面和文档有强需求

**Jobs to be Done**:
1. 一键配置代理以访问 AI 服务
2. 在不同 AI 模型间无缝切换
3. 使用中文界面管理所有配置

### 3.3 Tertiary: Team Lead (团队负责人)

**画像**: 管理 3-10 人开发团队的技术负责人
- 需要统一团队的 AI 工具配置
- 关心团队级别的 AI 使用成本
- 需要标准化 CLAUDE.md 和编码规范

**Jobs to be Done**:
1. 分发标准化配置给团队成员
2. 监控团队 AI 使用成本
3. 维护团队级 prompt 库和最佳实践

---

## 4. Feature Requirements / 功能需求

### 4.1 MVP (Milestone 1) - Core Configuration Manager

> Goal: Replace manual config editing for 3 major AI CLI tools

#### F1: Tool Dashboard (工具仪表盘)
- **F1.1** 自动检测已安装的 AI CLI 工具及版本
- **F1.2** 一键安装/更新/卸载 (Claude Code via bun, Codex via cargo, Gemini via npm)
- **F1.3** 工具健康状态指示（版本、配置有效性、连接状态）
- **F1.4** 首次使用引导向导 (Onboarding Wizard)

#### F2: Visual Config Editor (可视化配置编辑器)
- **F2.1** Claude Code: settings.json 可视化编辑（模型选择、权限、沙箱、MCP）
- **F2.2** Codex: config.toml 可视化编辑（模型、安全级别、审批策略）
- **F2.3** Gemini CLI: GEMINI.md 可视化编辑（扩展、安全设置）
- **F2.4** 实时配置预览 (Monaco Editor)
- **F2.5** 配置验证与错误提示
- **F2.6** 预设模板（快速开始、安全优先、性能优先、省钱模式）

#### F3: Proxy & Network (代理与网络)
- **F3.1** 系统代理自动检测 (HTTP_PROXY/HTTPS_PROXY/系统代理)
- **F3.2** 代理配置向导（Clash/V2Ray/Shadowsocks 一键配置）
- **F3.3** API Endpoint 自定义（支持 lurus-api 等自建网关）
- **F3.4** 连接测试与诊断

#### F4: Config Persistence (配置持久化)
- **F4.1** 命名配置快照（保存/恢复/比较）
- **F4.2** 配置导入/导出（JSON/ZIP）
- **F4.3** 配置历史版本浏览

### 4.2 Milestone 2 - Smart Optimization (信息差核心)

> Goal: Deliver quantifiable value through hidden best practices

#### F5: Smart CLAUDE.md Generator (智能 CLAUDE.md 生成器)
- **F5.1** 扫描项目目录结构和技术栈，自动生成 CLAUDE.md
- **F5.2** 内置最佳实践模板库（按框架: React/Go/Python/Rust）
- **F5.3** CLAUDE.md 质量评分（基于已知的优化规则）
- **F5.4** 优化建议（"用正面指令替换负面指令"、"减少冗余规则"等）
- **F5.5** 支持 GEMINI.md / Codex instructions 生成

#### F6: MCP Server Manager (MCP 服务器管理器)
- **F6.1** 可视化 MCP 服务器配置（表单替代 JSON 编辑）
- **F6.2** MCP 服务器目录（浏览社区热门服务器）
- **F6.3** 一键安装/配置 MCP 服务器
- **F6.4** MCP 服务器健康检测与日志查看
- **F6.5** 跨工具 MCP 配置同步（Claude 和 Gemini 都支持 MCP）

#### F7: Cost Dashboard (成本仪表盘)
- **F7.1** 连接 lurus-api 网关显示实时用量
- **F7.2** 日/周/月用量趋势图
- **F7.3** 按工具/模型/项目分类的费用明细
- **F7.4** 预算预警（日预算/月预算上限通知）
- **F7.5** 省钱建议（模型降级推荐、Prompt 缓存分析）

### 4.3 Milestone 3 - Ecosystem & Distribution

> Goal: Build distribution and retention

#### F8: Prompt Library (提示词库)
- **F8.1** 内置高质量 prompt 模板分类（编码/调试/重构/文档）
- **F8.2** 自定义 prompt 保存与管理
- **F8.3** 社区 prompt 共享（上传/下载/评分）
- **F8.4** Prompt 版本控制

#### F9: Distribution & Update (分发与更新)
- **F9.1** 自动更新检测与下载
- **F9.2** Scoop manifest (Windows)
- **F9.3** Homebrew formula (macOS/Linux)
- **F9.4** WinGet manifest (Windows)
- **F9.5** GitHub Releases 自动发布 (CI/CD)

#### F10: Team Features (团队功能) [Pro]
- **F10.1** 配置模板导出为可分发包
- **F10.2** 团队级 CLAUDE.md 标准
- **F10.3** 成本汇总报告

---

## 5. UI/UX Requirements / 界面体验需求

### 5.1 Design Principles

1. **一致的中文界面** -- 默认中文，支持英文切换，不混用
2. **渐进式展示** -- 新手看到简化视图，高级用户可展开详细配置
3. **零学习成本** -- 每个功能入口附带简短说明，关键操作有引导
4. **即时反馈** -- 所有操作 <200ms 响应，长操作显示进度
5. **深色主题优先** -- 开发者偏好，支持浅色切换

### 5.2 Navigation Structure

```
[Sidebar]
├── 🏠 仪表盘           ← Dashboard with tool cards + quota widget
├── ⚙️ 工具配置
│   ├── Claude Code
│   ├── Codex CLI
│   └── Gemini CLI
├── 🔌 MCP 服务器       ← Visual MCP manager
├── 📝 CLAUDE.md 助手   ← Smart generator
├── 📊 成本监控          ← Cost dashboard
├── 📚 提示词库          ← Prompt library
├── ──────────
├── ⚡ 设置              ← App settings (proxy, theme, language, updates)
└── 💰 账户              ← Billing & subscription
```

### 5.3 Key User Flows

**Flow 1: First-Time Setup (首次使用)**
```
Welcome Screen → Select tools to manage → Auto-detect installed tools
→ Proxy configuration (if in China) → API key setup → Dashboard
```

**Flow 2: Configure a Tool (配置工具)**
```
Dashboard → Click tool card → Visual editor (tabs: Basic/Advanced/MCP)
→ Edit settings → Live preview → Validate → Save/Export
```

**Flow 3: Generate CLAUDE.md (生成 CLAUDE.md)**
```
CLAUDE.md Assistant → Select project directory → Auto-scan
→ Review generated content → Edit → Quality score → Export to project
```

**Flow 4: Monitor Costs (监控成本)**
```
Cost Dashboard → View daily/weekly trend → Drill into model breakdown
→ See optimization suggestions → Apply recommended settings
```

### 5.4 Key UI Components

| Component | Description | Priority |
|-----------|------------|----------|
| OnboardingWizard | 首次使用引导，3-5步 | P0 (MVP) |
| ToolCard | 工具状态卡片（安装状态、版本、健康度） | P0 (MVP) |
| VisualConfigEditor | 表单式配置编辑器 + Monaco 预览 | P0 (MVP) |
| ProxyWizard | 代理配置向导（自动检测 + 手动配置） | P0 (MVP) |
| MCPServerCard | MCP 服务器卡片（状态、配置、日志） | P1 |
| CostChart | 费用趋势图 (日/周/月) | P1 |
| QualityScoreRing | CLAUDE.md 质量评分环形图 | P1 |
| PromptCard | 提示词卡片（预览、复制、收藏） | P2 |

---

## 6. Non-Functional Requirements / 非功能需求

### 6.1 Performance
- 应用启动 < 2s
- 配置加载 < 500ms
- 工具检测 < 3s (并行检测)
- 二进制大小 < 15MB (Wails)

### 6.2 Reliability
- 配置写入前自动备份
- 异常退出后配置不丢失
- 网络不可用时优雅降级（离线模式）

### 6.3 Security
- API key 加密存储（OS keychain 优先，fallback 加密文件）
- 不在日志中打印敏感信息
- 配置导出时脱敏 API key

### 6.4 Compatibility
- Windows 10/11 (x64)
- macOS 12+ (x64, arm64)
- Linux (x64, Wayland/X11)

---

## 7. Distribution Strategy / 分发策略

### 7.1 Zero-Cost Channels

| Channel | Platform | Priority | Setup Cost |
|---------|----------|----------|-----------|
| GitHub Releases | All | P0 | CI/CD already exists |
| Scoop | Windows | P0 | JSON manifest |
| WinGet | Windows | P1 | YAML manifest + review |
| Homebrew | macOS/Linux | P1 | Formula + tap repo |
| npm (optional) | All | P2 | package.json wrapper |

### 7.2 Growth Strategy

1. **Phase 1**: GitHub + 中文技术社区（掘金/V2EX/知乎）发布
2. **Phase 2**: Scoop/Homebrew 上架，Product Hunt launch
3. **Phase 3**: 中国开发者社区口碑传播（代理配置 + 国内模型支持作为差异化卖点）

---

## 8. Monetization / 商业模式

### 8.1 Freemium Tiers

| Tier | Price | Features |
|------|-------|----------|
| **Free** | $0 | 3 工具配置管理, 基础 CLAUDE.md 模板, 手动 MCP 配置, 无成本监控 |
| **Pro** | ¥49/月 (~$7) | 智能 CLAUDE.md 生成+优化, MCP 服务器目录, 成本仪表盘, 无限快照, 优先更新 |
| **Team** | ¥149/月/5人 (~$21) | 团队配置分发, 成本汇总报告, 共享 prompt 库 |

### 8.2 Revenue Integration
- 通过 lurus-api 网关的 LLM API 调用抽成（现有架构）
- Pro/Team 订阅通过 lurus-identity 账户系统

---

## 9. Success Metrics / 成功指标

| Metric | Target (3 months) | Target (6 months) |
|--------|-------------------|-------------------|
| Downloads | 500 | 3,000 |
| WAU (Weekly Active Users) | 50 | 300 |
| Config saves/week | 200 | 1,500 |
| Pro conversion rate | 3% | 5% |
| NPS | > 30 | > 50 |
| GitHub Stars | 100 | 500 |

---

## 10. Competitive Positioning / 竞品定位

```
                    Configuration Depth
                         ↑
                         |
           Manual        |     Lurus Switch
           Editing    ---|-------- ★
                         |        /
            AionUi  ●   |       /
                         |      /
                    ─────┼─────/──────→ Tool Coverage
                         |    /
           Cursor    ●   |   /  (single-tool)
           (built-in)    |  /
                         | /
                         |/
```

**Lurus Switch 的独特位置**: 高配置深度 + 多工具覆盖 + 信息差变现

---

## Appendix: Glossary

| Term | Definition |
|------|-----------|
| MCP | Model Context Protocol - AI 工具与外部服务的通信协议 |
| CLAUDE.md | Claude Code 的项目级配置文件 |
| Wails | Go + WebView 桌面框架 |
| lurus-api | Lurus 平台的 LLM API 网关 |
| 信息差 | Information asymmetry - 专业知识与普通用户之间的认知差距 |
