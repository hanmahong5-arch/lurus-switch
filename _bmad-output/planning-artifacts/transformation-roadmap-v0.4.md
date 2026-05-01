# Switch 改造路线图 v0.4 — 渠道分销基础设施 Pivot

**Version**: 0.4
**Date**: 2026-05-01
**Owner**: Hanmahong + AI 协作整理
**Aligned PRD**: `Switch PRD v0.1 (B2B2C 渠道分销基础设施)`
**Supersedes**: 龙虾管理员 v3 (Agent Fleet 方向)，但保留 Agent Fleet 工作（已落库 `internal/agent/`、`AgentsPage.tsx`）作为 Personal 模式的高级特性

---

## 0. TL;DR

Switch 从"个人 CLI 管家"扩展为**三模式渠道分销基础设施**：

| Mode | 用户 | 核心动作 | Hub 后端 |
|------|------|---------|---------|
| **Personal** | 个人开发者 | 装 CLI、配代理、用 Lurus 自营 LLM | `hub.lurus.cn` (Lurus 运营) |
| **Reseller** | 技术博主 / 培训机构 / 跨境团队 | 部署专属 Hub、生成激活码、导出白标客户端 | 经销商自部署 newhub |
| **EndUser** | 经销商的客户 | 输激活码 → 装 CLI → 用 | 白标包内嵌的经销商 Hub URL |

**所有三个模式共用同一个 Wails 二进制，启动时选模式 / 白标包预锁模式。**

后端不自研：直接用 **`lurus-newhub` (`2b-svc-newhub`)** 作为 Hub。它已实现多租户 (Zitadel + tenant_slug)、Platform gRPC 计费集成、激活码 (newapi 原生 redemption)、Switch 专用 API (`/api/v2/switch/*`)。

---

## 1. 锁定决策（Architecture Decisions）

### D1: 不集成开源 new-api，使用自研 lurus-newhub

**背景**：PRD v0.1 ADR-006 倾向集成 QuantumNous/new-api。
**新决策**：使用 `lurus-newhub` (在 newapi 之上叠加多租户 + Platform 集成 + Switch 专用 endpoint)。
**理由**：
- newhub 已经为 Switch 留好对接位 (`/api/v2/switch/{tools/versions, presets}` 公共接口 + `POST /api/v2/admin/switch/presets` 经销商创建 preset)
- AGPL 风险消除（自研代码，自家 license）
- 与 Lurus Platform 计费链路打通（`IDENTITY_GRPC_ADDR` → 钱包扣款）
- 多租户隔离已实现（GORM 插件自动注入 `tenant_id`，无需 Switch 关心）

### D2: 保留现有 `internal/{gateway,relay,optimizer,modelcatalog}` 作为客户端层

**背景**：之前讨论过是否砍掉自研网关层。
**新决策**：保留。它们是**本地代理 + 端点选择器 + 诊断器**，与远端 Hub 不冲突，反而是 Hub 的客户端互补层：
- `internal/gateway/` → 本地 OpenAI 兼容代理（监听 `:19090`），把 CLI 请求转给远端 Hub
- `internal/relay/` → 端点列表 + 健康状态（Hub 推荐的 relay 列表）
- `internal/optimizer/` → 配置医生（不依赖 Hub，本地诊断）
- `internal/modelcatalog/` → 模型清单缓存（从 Hub 拉，本地缓存兜底）

### D3: 一个二进制三模式，启动时选

**背景**：PRD 说"经销商版 + 用户版"两个产品。
**新决策**：单 Wails 二进制 + `AppMode = { Personal, Reseller, EndUser }`。
- 首次启动：模式选择向导（除非白标包预锁）
- Reseller / EndUser 仅显示对应页面（路由守卫）
- 白标打包：经销商通过 Reseller 模式导出 EndUser 包（嵌入 `hub_url` + 模式锁死）

### D4: 11 个现有 GatewayXxx 页面 → Reseller 控制台

**背景**：之前未确定是砍还是留这 11 个无后端的 page。
**新决策**：留下，作为 **Reseller 模式**的运营控制台，对接 newhub V1 API。
- `GatewayChannelPage` → newhub `/api/channel/*` (28 endpoints, channel CRUD + test + tag)
- `GatewayTokenPage` → newhub `/api/token/*` (CRUD + 批量删除)
- `GatewayRedemptionPage` → newhub `/api/redemption/*` (激活码 CRUD)
- `GatewayLogPage` → newhub `/api/log/*` (用户日志，含 Meilisearch 全文检索)
- `GatewayUserPage` → newhub `/api/v2/admin/mappings/*` (跨租户用户映射)
- `GatewaySubscriptionPage` → newhub `/api/v2/user/billing/*`
- `GatewayModelPage` → newhub `/api/models` (admin) + `/api/openrouter-sync/*`
- 其余: `GatewayDashboardPage` → newhub governance + audit endpoints

### D5: 激活码 = newapi 原生 RedemptionCode + 设备指纹绑定

**背景**：PRD 要求激活码绑设备 + 心跳。
**新决策**：
- 复用 newhub 自带的 redemption code 系统 (`POST /api/redemption/` 创建批次, `POST /api/user/topup` 兑换)
- Switch 客户端额外做：本地保存 device fingerprint (CPU+MAC+OS hash) + 周期心跳到 Hub `/api/v2/:tenant_slug/user/me`
- newhub 侧扩展：tenant 配置允许"设备绑定模式"，首次兑换时 Hub 记录 fingerprint，之后 token 验证带 fingerprint header

### D6: Reseller 部署初版只支持 Sealos + 阿里云 ECS

**背景**：PRD 列了阿里云/腾讯云/Cloudflare/Sealos/Vercel 五个目标。
**新决策**：MVP 阶段只做两个，覆盖 80% 经销商：
- **Sealos**：K8s 模板最成熟（newhub 现成 K8s manifest 在 `2b-svc-newhub/deploy/k8s/`），一键拉起
- **阿里云 ECS**：SSH + docker-compose 起 newhub + Postgres + Redis 套装

腾讯云、Cloudflare Workers 留 Phase 2。

### D7: Agent Fleet 工作（E4-E10）保留为 Personal 模式高级特性

**背景**：v3 路线图把 agent fleet 做成主线，已落库部分代码（`internal/agent/`、`AgentsPage.tsx`）。
**新决策**：不删除、不优先。Agent fleet 在 Personal 模式下仍有价值（个人开发者管理多 CLI agent），但市场切入点是 Reseller 模式。完成 E12-E15 后再回来推进 Agent Fleet。

---

## 2. 三模式形态定义

### 2.1 Personal Mode（保持现状 + 增强）

**目标用户**：自己用 Switch 的开发者。
**核心 UX**：
- 启动 → 登录 Lurus 账号 (Zitadel PKCE) → 看 dashboard
- 装 CLI、配代理、查账单全部用现有页面
- Hub 配置默认 `hub.lurus.cn`（Lurus 自营），用户不可见也不修改

**新增能力**：无（现有特性即覆盖）。

### 2.2 Reseller Mode（新模式，工作量最大）

**目标用户**：开 AI 编程小卖部的技术博主、培训机构、跨境团队。
**核心 UX**：
1. **首次设置**：选云厂商 (Sealos/阿里云) → 输 API Key → 选套餐 → Switch 拉起 newhub → 拿到 `hub_url` + admin token
2. **路由配置**：在 GatewayChannelPage 配 channel（上游 Anthropic/OpenRouter/智谱/...）+ 路由策略
3. **激活码批次**：在 GatewayRedemptionPage 生成 N 个 code（CSV 导出）
4. **白标客户端**：在新 PackagerPage 上传 logo + 主题色 + 品牌名 → 生成嵌入了 `hub_url` 的 EndUser .exe（每个 code 一包，提升防破解）
5. **日常运营**：看 Dashboard 用量曲线 + 一键封禁滥用用户 + 调路由策略

**新增模块**：
- `internal/hub/deploy/` — 云部署器（Sealos client + Aliyun ECS SSH）
- `internal/hub/admin/` — newhub V1+V2 admin API client
- `internal/whitelabel/` — 白标二进制打包器（参考现有 `internal/packager/`）
- `frontend/src/pages/ResellerSetupWizard.tsx` — 引导向导
- `frontend/src/pages/PackagerPage.tsx` — 白标导出（替换/重命名 PromoterHubPage）

### 2.3 EndUser Mode（新模式，逻辑较简单）

**目标用户**：经销商的 C 端客户（学生、初级开发者）。
**核心 UX**：
1. **首次启动**：白标包已嵌入 `hub_url` + 经销商品牌 → 显示"输入激活码"页
2. **激活**：输入 code → POST `/api/user/topup` → 拿到 user_token + quota → 持久化 + 绑定设备指纹
3. **使用**：选 CLI → 自动装 + 配 env → 在终端用
4. **续费**：用量耗尽时弹激活码输入页（或经销商续费链接）

**新增模块**：
- `internal/redemption/` — 激活码兑换 + 设备指纹 + 心跳
- `internal/lockedhub/` — Hub URL 锁定 (EndUser 模式不可改 Hub URL)
- `frontend/src/pages/EndUserActivationPage.tsx` — 激活码输入向导
- `frontend/src/pages/EndUserMainPage.tsx` — 简化版 dashboard（仅余额 + CLI 列表 + 用量）

### 2.4 模式守卫与切换

| 行为 | Personal | Reseller | EndUser |
|------|----------|----------|---------|
| 首次启动选模式 | ✅ | ✅ | ❌（白标锁定） |
| 切换模式 | ✅（设置中） | ✅ | ❌（重装可改回 Personal） |
| 修改 Hub URL | ✅（设置中） | ✅（向导） | ❌ |
| 看 GatewayXxx 管理页 | ❌（隐藏） | ✅ | ❌（隐藏） |
| 看 PackagerPage | ❌ | ✅ | ❌ |
| 看 EndUser 激活页 | ❌ | ❌ | ✅ |
| 装 CLI | ✅ | ✅（测试用） | ✅ |

---

## 3. 三阶段实施路径

### **Phase A: 形态分离基础（Sprint 4b, 2 周）**

> 目标：AppMode 三态可切换，路由守卫到位。**为后续两阶段奠基**。

**Stories**:
- S-Xa.1: AppMode 三态 + 持久化 + Wails bindings (5pt)
- S-Xa.2: 首次启动模式选择向导 (3pt)
- S-Xa.3: 路由守卫（11 GatewayXxx 页限 Reseller）(3pt)
- S-Xa.4: Sidebar 模式徽章 + 模式切换设置 (2pt)
- S-Xa.5: newhub V1+V2 client SDK (`internal/hub/admin/client.go`) (8pt)

**总计**: 21pt

**Definition of Done**: 用户能在三模式间切换，对应页面正确显示/隐藏；newhub client 单元测试通过（mock server）。

### **Phase B: Reseller 核心（Sprint 4c-4d, 4 周）**

> 目标：Reseller 模式从向导到日常运营全链路打通。

**Stories**:
- S-Xb.1: ResellerSetupWizard 框架 (5pt)
- S-Xb.2: Sealos deploy adapter (`internal/hub/deploy/sealos.go`) (8pt)
- S-Xb.3: 阿里云 ECS deploy adapter (8pt)
- S-Xb.4: GatewayChannelPage 对接 newhub V1 channel API (5pt)
- S-Xb.5: GatewayTokenPage 对接 + 批量操作 (3pt)
- S-Xb.6: GatewayRedemptionPage 对接 + CSV 导出 (5pt)
- S-Xb.7: GatewayLogPage 对接 + Meilisearch 检索 (5pt)
- S-Xb.8: GatewayDashboardPage 对接 governance + audit (5pt)

**总计**: 44pt（拆 2 sprint）

**Definition of Done**: 经销商能在 30 分钟内：部署 Hub → 配 channel → 生成激活码 → 看到日志。E2E 测试用 docker-compose 起 newhub + 全流程 happy path。

### **Phase C: EndUser + 白标 + 心跳（Sprint 4e, 2 周）**

> 目标：白标 EndUser 安装包能交付到 C 端用户手上并完整使用。

**Stories**:
- S-Xc.1: `internal/whitelabel/` 白标打包器 + 资源替换 (8pt)
- S-Xc.2: PackagerPage UI（上传 logo + 颜色 + 嵌入 hub_url + 批量生成）(5pt)
- S-Xc.3: EndUserActivationPage + 激活码兑换 (5pt)
- S-Xc.4: `internal/redemption/` 设备指纹 + token 持久化 (5pt)
- S-Xc.5: 心跳客户端 + 失效自动锁定 (3pt)
- S-Xc.6: EndUserMainPage 简化 dashboard (5pt)

**总计**: 31pt

**Definition of Done**: 经销商打包 EndUser .exe → 双击安装 → 输码激活 → 装 Claude Code → 跑通一次请求。心跳断开 5 分钟后 token 标记 stale，10 分钟后封禁。

### **Phase D（远期，可选）**: 自动更新、代码签名、多平台 EndUser、Cloudflare Workers Hub

---

## 4. 模块映射表（现状 → 新形态）

| 现状模块 | 用途变化 | Phase |
|---------|---------|-------|
| `internal/auth/` | Personal Zitadel 登录 + EndUser 激活码兑换 (复用 PKCE 不同 code path) | A→C |
| `internal/deeplink/` | 不变（Reseller 经 deeplink 推 hub_url 给 EndUser 是备用方案） | — |
| `internal/installer/` | 不变（三模式都用） | — |
| `internal/toolmanifest/` | EndUser 模式从锁定 hub 拉 manifest，Reseller 可上传自定义 | C |
| `internal/gateway/` | 本地代理保留，目标 URL 由 AppMode 决定 | A |
| `internal/relay/`, `optimizer/`, `modelcatalog/` | 不变（Personal/EndUser 都用） | — |
| `internal/promoter/` | **重定位** → `internal/hub/admin/affiliate.go`（仅 Reseller 模式查 affiliate 数据） | B |
| `internal/packager/` | 保留作"CLI 配置打包"原功能，**不**升级为白标打包 | — |
| `internal/serverctl/` | **废弃**（legacy gateway binary manager），Phase A 末删除 | A |
| `internal/agent/` | 保留，Agent Fleet 路线后续推进，仅 Personal 模式可用 | — (later) |
| `internal/db/` | Phase B 加 schema：reseller_codes, device_bindings, hub_deployments, white_label_profiles | B |
| `internal/billing/` | 不变（Personal/EndUser 看自己余额；Reseller 经 newhub admin） | — |

| 现状页面 | 新归属 | Phase |
|---------|-------|-------|
| HomePage / DashboardPage | Personal | — |
| AccountPage / BillingPage | Personal + EndUser（简化） | C |
| AdminPage | **删除** (替换为 GatewayXxx 集合) | A |
| GatewayChannelPage 等 11 个 | Reseller only | B |
| PromoterHubPage | **重写** → PackagerPage (Reseller) | C |
| AgentsPage | Personal only（Agent Fleet 路线） | — (later) |
| SwitchHubPage | Personal main dashboard | — |
| ToolConfigPage / ProcessPage / SettingsPage | 三模式共用（按权限裁剪） | A |

---

## 5. 风险登记

| ID | 风险 | 概率 | 影响 | 缓解 |
|----|------|------|------|------|
| R1 | newhub V2 API 不稳定（创建于今天，0 stars 实战检验少） | 高 | 高 | Switch 接入前先 PR 补 OpenAPI spec + integration test，与 newhub 团队约定契约稳定后再 ship |
| R2 | Sealos / 阿里云 deploy adapter 边缘场景多 | 高 | 中 | 只支持 1 region 1 instance type；失败回退手动安装文档 |
| R3 | 设备指纹绕过（虚拟机克隆、网卡欺骗） | 中 | 中 | 多因子指纹（CPU+MAC+OS+硬盘 ID）；监控异常激活速率，发现后 Hub 端封禁 code |
| R4 | 上游 LLM 提供商封号波及经销商账号 | 中 | 高 | newhub 多 channel + 自动故障转移；经销商账号严格隔离，Switch 不持有上游 key |
| R5 | 11 GatewayXxx 页面对接工作量超估 | 高 | 中 | Sprint B 留 buffer；优先做 channel + token + redemption 三个核心 page，其余留 Phase B+1 |
| R6 | 白标打包文件名冲突 / 资源替换失败（参考 `gh-release` skill 的 `#name` 陷阱） | 中 | 低 | Phase C 起手就写集成测试：打包 → 反向解析嵌入资源是否正确 |
| R7 | EndUser 心跳断网误封 | 中 | 中 | 离线宽限期 72h（PRD M7 已规划），断网期间 token 不立即失效 |
| R8 | Agent Fleet 工作（已部分入库）冷启动后回来时合并困难 | 低 | 中 | 不删 `internal/agent/`、不删 `AgentsPage.tsx`；标 Personal-only 隐藏即可，未来回归时只需打开路由守卫 |

---

## 6. 与现有 BMAD 的关系

| Epic | 状态 | 决策 |
|------|------|------|
| E1-E3 (Foundation/Onboarding/Editor) | ✅ 完成 | 不动 |
| E4 (Agent Foundation) | 🟡 部分入库 (`internal/agent/`, `AgentsPage.tsx`) | **暂停**，标 Personal-only |
| E5-E10 (Agent Lifecycle/Resource/Context/Monitoring/Templates) | 📋 规划 | **冻结**，Phase A-C 完成后回归 |
| **E12 (新)** AppMode 三态 | — | Phase A |
| **E13 (新)** Hub Integration | — | Phase A 末 + Phase B |
| **E14 (新)** Reseller Mode | — | Phase B |
| **E15 (新)** EndUser Mode | — | Phase C |
| E11 (远期编排) | 🔮 远期 | 不变 |

详见 `epics.md` 增量更新。

---

## 7. 时间表（粗估）

| Phase | Sprint | 周次 | 起止 (建议) | 总点数 |
|-------|--------|------|------------|-------|
| A | 4b | 2 周 | 2026-05-04 → 2026-05-17 | 21pt |
| B (part 1) | 4c | 2 周 | 2026-05-18 → 2026-05-31 | 22pt |
| B (part 2) | 4d | 2 周 | 2026-06-01 → 2026-06-14 | 22pt |
| C | 4e | 2 周 | 2026-06-15 → 2026-06-28 | 31pt |
| **合计** | — | **8 周** | — | **96pt** |

8 周后：Reseller MVP 上线，可启动 5 个种子经销商内测（PRD Phase 1 里程碑）。

---

## 8. 立即行动项（本会话）

- [x] 调研 newhub V2 API 完整性
- [x] 把 lurus-newhub clone 到 `2b-svc-newhub/`
- [x] 注册 newhub 到 `lurus.yaml` + 根 CLAUDE.md
- [x] Switch CLAUDE.md 更新 cross-service deps
- [ ] 写本路线图（本文件）— **进行中**
- [ ] 写 ADR-020
- [ ] 更新 epics.md (E12-E15)
- [ ] 更新 sprint-status.yaml (Sprint 4b 启动)
- [ ] 启动 S-Xa.1: AppMode 三态实现

---

## 附录 A: newhub API 对接速查

| Switch 页面 | newhub Endpoint | Auth |
|------------|----------------|------|
| GatewayChannelPage | `/api/channel/*` (28 endpoints) | AdminAuth |
| GatewayTokenPage | `/api/token/*` | AdminAuth |
| GatewayRedemptionPage | `/api/redemption/*` | AdminAuth |
| GatewayLogPage | `/api/log/*`（含 Meilisearch） | AdminAuth or UserAuth (self) |
| GatewayUserPage | `/api/v2/admin/mappings/*` | RootJWTAuth |
| GatewayModelPage | `/api/models` + `/api/openrouter-sync/*` | AdminAuth |
| GatewaySubscriptionPage | `/api/v2/user/billing/*` | ZitadelAuth |
| GatewayDashboardPage | `/api/v2/admin/{stats, governance/*, audit/events}` | RootJWTAuth |
| EndUserActivationPage | `POST /api/user/topup` | UserAuth (token from redeemed code) |
| Switch presets dropdown | `GET /api/v2/switch/presets` | none (public) |
| Tool version manifest | `GET /api/v2/switch/tools/versions` + `GET /api/v2/tools/download-manifest` | none (public) |

---

## 附录 B: 命名规范决策

- **Repo**: `hanmahong5-arch/lurus-newhub` （已存在）
- **Monorepo dir**: `2b-svc-newhub` （遵循 audience-delivery-name 规范）
- **lurus.yaml service key**: `lurus-newhub`
- **K8s deployment**: `lurus-api` （runtime resource name 保留，与镜像 `ghcr.io/LurusTech/lurus-api:main` 一致）
- **Module path**: `github.com/LurusTech/lurus-hub` （Go module 不变）
- **Future domain**: `hub.lurus.cn` （目前仅规划，未签证书）

> 历史：CLAUDE.md 旧版自称 `2b-svc-api`，是被废弃的命名；此次落库统一为 `2b-svc-newhub` + `lurus-newhub`。
