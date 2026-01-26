# Lurus Switch 重构计划：从微服务到桌面应用
# Refactoring Plan: Microservices to Desktop Application

> **状态 / Status**: Completed
> **创建日期 / Created**: 2026-01-26
> **更新日期 / Updated**: 2026-01-26

---

## 概述 / Overview

将 lurus-switch 从复杂微服务架构（7+ 服务）重构为 **Wails 桌面应用**，专注于为三个 AI CLI 工具生成可执行配置包。

---

## 核心功能 / Core Features

三个配置生成按钮：

| 工具 | GitHub | 配置格式 | 打包方式 | 状态 |
|------|--------|----------|----------|------|
| **Claude Code** | github.com/anthropics/claude-code | JSON (settings.json) | Bun compile | ✅ |
| **Codex** | github.com/openai/codex | TOML (config.toml) | Rust binary download | ✅ |
| **Gemini CLI** | github.com/google-gemini/gemini-cli | Markdown (GEMINI.md) | Node.js pkg | ✅ |

---

## 技术栈 / Tech Stack

- **后端**: Go 1.22+ + Wails v2
- **前端**: React 18 + TypeScript + Tailwind CSS
- **状态管理**: Zustand
- **UI 组件**: Radix UI + Lucide Icons
- **代码编辑器**: Monaco Editor

---

## 实现进度 / Implementation Progress

### Phase 1: 项目初始化 ✅
- [x] 清理现有微服务代码（保留 doc/ 目录）
- [x] 初始化 Wails 项目: `wails init -n lurus-switch -t react-ts`
- [x] 配置 npm (因 Wails 兼容性)
- [x] 安装前端依赖 (Tailwind, Zustand, Monaco Editor, Radix UI)

### Phase 2: 配置模型 ✅
- [x] 实现 `internal/config/claude.go` - Claude Code 配置结构
- [x] 实现 `internal/config/codex.go` - Codex 配置结构
- [x] 实现 `internal/config/gemini.go` - Gemini CLI 配置结构
- [x] 实现 `internal/config/store.go` - 配置持久化（JSON 文件存储）

### Phase 3: 配置生成器 ✅
- [x] 实现 Claude Generator (JSON)
- [x] 实现 Codex Generator (TOML，使用 BurntSushi/toml)
- [x] 实现 Gemini Generator (Markdown)

### Phase 4: 打包器 ✅
- [x] 实现 Bun Packager - 调用 `bun build --compile` 打包 Claude Code
- [x] 实现 Rust Packager - 从 GitHub Releases 下载 Codex 二进制
- [x] 实现 Node Packager - 使用 pkg 打包 Gemini CLI

### Phase 5: 前端 UI ✅
- [x] 实现 Sidebar 组件
- [x] 实现 StatusBar 组件
- [x] 实现 ConfigPreview 组件 (Monaco Editor)
- [x] 实现 ClaudePage - 模型设置、权限、沙箱、高级选项
- [x] 实现 CodexPage - 模型设置、安全、Provider、MCP
- [x] 实现 GeminiPage - 认证、模型、Markdown 编辑器
- [x] 实现 Zustand store 状态管理

### Phase 6: 集成测试 🔄
- [x] Go 代码编译通过
- [x] 前端 TypeScript 编译通过
- [x] Wails 构建成功（生成 lurus-switch.exe）
- [ ] 手动功能测试
- [ ] 跨平台构建测试 (macOS/Linux)

---

## 项目结构 / Project Structure

```
lurus-switch/
├── main.go                      # Wails 入口
├── app.go                       # 暴露给前端的 Go 方法
├── wails.json
├── go.mod
│
├── internal/
│   ├── config/                  # 配置结构定义
│   │   ├── store.go             # 配置持久化
│   │   ├── claude.go            # Claude Code schema
│   │   ├── codex.go             # Codex schema
│   │   └── gemini.go            # Gemini CLI schema
│   │
│   ├── generator/               # 配置文件生成
│   │   ├── claude_generator.go  # 生成 settings.json
│   │   ├── codex_generator.go   # 生成 config.toml
│   │   └── gemini_generator.go  # 生成 GEMINI.md
│   │
│   ├── packager/                # 可执行文件打包
│   │   ├── bun_packager.go      # Claude Code (Bun)
│   │   ├── rust_packager.go     # Codex (下载 Rust 二进制)
│   │   └── node_packager.go     # Gemini CLI (Node.js)
│   │
│   ├── downloader/              # GitHub Release / NPM 下载
│   └── validator/               # 配置验证
│
├── frontend/                    # React + TypeScript
│   ├── src/
│   │   ├── pages/
│   │   │   ├── ClaudePage.tsx
│   │   │   ├── CodexPage.tsx
│   │   │   └── GeminiPage.tsx
│   │   ├── components/
│   │   │   ├── Sidebar.tsx
│   │   │   ├── StatusBar.tsx
│   │   │   └── ConfigPreview.tsx
│   │   ├── stores/
│   │   │   └── configStore.ts
│   │   └── lib/
│   │       └── utils.ts
│   └── package.json
│
└── doc/                         # 文档
```

---

## 数据存储 / Data Storage

用户配置存储位置：
- **Windows**: `%APPDATA%\lurus-switch\configs\`
- **macOS**: `~/Library/Application Support/lurus-switch/configs/`
- **Linux**: `~/.config/lurus-switch/configs/`

---

## 构建命令 / Build Commands

```bash
# 开发模式
wails dev

# 生产构建
wails build

# 生成 Wails 绑定
wails generate module
```

---

## 清理完成 / Cleanup Completed

已删除以下微服务目录：
- `gateway-service/`
- `provider-service/`
- `billing-service/`
- `log-service/`
- `identity-service/`
- `tenant-service/`
- `subscription-service/`
- `agent-service/`
- `lurus-common/`
- `www/`
- `lurus-portal/`
- `deploy/`
- `api/`

保留：
- `doc/` - 文档
- `CLAUDE.md` - 开发指南
- `README.md` - 说明
- `.git/` - Git 历史
- `.github/` - GitHub 配置

---

*Updated by Claude Code | 2026-01-26*
