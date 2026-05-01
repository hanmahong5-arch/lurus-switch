# ADR-020: Switch 转向渠道分销基础设施 (B2B2C Pivot)

**Date**: 2026-05-01
**Status**: Accepted
**Decision Maker**: Hanmahong (CEO) + AI 协作整理
**Supersedes**: 龙虾管理员 v3 主线（保留为 Personal 模式高级特性）
**Aligned PRD**: Switch PRD v0.1
**Aligned Roadmap**: `transformation-roadmap-v0.4.md`

---

## Context

Switch v0.1.0 已发布，定位为"个人 AI CLI 管家"。现状代码 35 个 internal 模块、29 个前端页面，能力远超个人工具范围（已有 7 个 GatewayXxx 管理页、`internal/promoter/`、`internal/packager/`、`internal/serverctl/` 等强烈暗示已经在朝"分销/Hub"方向暗中演化），但**形态停留在单租户单二进制**。

CEO 提出 PRD v0.1（B2B2C 渠道分销基础设施），要求：
1. 经销商部署专属 Hub（云端，多租户）+ 生成激活码 + 导出白标客户端
2. C 端用户用激活码激活白标客户端，享受丝滑装 CLI 体验
3. Switch 收经销商 SaaS 订阅费，不碰 C 端钱、不碰流量

**关键约束**：
- 不可放弃现有 Personal 模式用户（已发 v0.1.0）
- 公司已存在自研 Hub 后端 `lurus-newhub`（2026-05-01 创建），实现了多租户、Platform 计费、Switch 专用 endpoint
- 时间窗口紧（PRD Phase 1 内测 5 个种子经销商目标）

---

## Decision

### 整体方向

Switch 演化为**单二进制三模式（Personal / Reseller / EndUser）渠道分销客户端**，远端 Hub 后端复用 `lurus-newhub`。

### 关键子决策

1. **不集成开源 new-api**（推翻 PRD ADR-006）→ 使用自研 `lurus-newhub`
   - 消除 AGPL 协议风险
   - 复用 Platform gRPC 计费链路
   - newhub 已为 Switch 留对接位（`/api/v2/switch/*`）

2. **保留 `internal/{gateway,relay,optimizer,modelcatalog}`** → 作为客户端互补层
   - 与远端 Hub 不冲突（远端做计费，本地做代理 + 端点选择 + 诊断）

3. **一个二进制三模式**（推翻 PRD"经销商版 + 用户版"双产品提议）
   - 启动时模式选择 + 白标包预锁
   - 路由守卫按模式裁剪可见页

4. **11 个现有 GatewayXxx 页面**作为 Reseller 控制台留下，对接 newhub V1 admin API
   - 现成 UI + 现成后端，开发成本最低

5. **激活码** = newhub 原生 redemption code + Switch 客户端追加设备指纹绑定 + 心跳验证
   - 不重新发明，扩展 newhub 端 tenant 配置即可

6. **云部署 MVP** 只做 Sealos + 阿里云 ECS（推翻 PRD 五选目标）
   - 覆盖 80% 经销商；腾讯云 / Cloudflare Workers 留 Phase 2

7. **Agent Fleet 路线（E4-E10）** 暂停，保留代码 + 标 Personal-only 隐藏
   - 已部分入库（`internal/agent/`、`AgentsPage.tsx`）的工作不浪费
   - Phase A-C 完成后回归

---

## Alternatives Considered

### A1: 集成开源 QuantumNous/new-api

**Pros**: 零自研后端、社区成熟、上游持续更新
**Cons**:
- AGPL 协议传染风险（Switch 客户端如分发 new-api 镜像，可能被解释为衍生作品）
- 多租户、Platform 计费集成需自行扩展
- 失去与 Lurus Platform (Zitadel + 钱包) 一体化优势

**Rejected**: 自研 newhub 已经实现这些扩展，且公司已投入；继续走自研更优。

### A2: 桌面端 + 独立 Web 控制台分发（Reseller 用 Web，EndUser 用 Desktop）

**Pros**: Web 控制台容易集中升级；权限隔离更清晰
**Cons**:
- 需新建 Web 项目（前端 + 部署 + 域名）
- 经销商部署专属 Hub 时，Web 控制台从哪个域名访问？需为每个经销商分配子域名 → 复杂度爆炸
- 用户需在两个产品间切换（管理用 Web，使用用 Desktop）

**Rejected**: 单二进制三模式更简洁，经销商也只需维护一个客户端。

### A3: 双产品分发（经销商专属 .exe + EndUser .exe 各打各包）

**Pros**: 二进制小、权限严格不可越权
**Cons**:
- 升级要发两套 release，运维成本翻倍
- 经销商想测试 EndUser 体验时要装两个 app
- 代码 split 工作量大

**Rejected**: 启动时模式 + 路由守卫已能达到等价效果，不需要物理隔离。

### A4: 不动 Switch，新建 `2b-gui-switch-reseller` 单独项目

**Pros**: Switch 现有用户不受影响、风险隔离
**Cons**:
- 现有 7 个 GatewayXxx 页面 + `promoter/` 等代码白写
- Reseller 也要装 CLI 测试，与 Switch 现有 installer 重复造轮
- 长期维护两个 Wails 项目成本太高

**Rejected**: 现有代码 60% 三模式可复用，分项目反而割裂。

---

## Consequences

### 正面影响

- **8 周内**可达成 PRD Phase 1 里程碑（5 个种子经销商）
- 已写代码（GatewayXxx, promoter, packager 框架）找到归属，不浪费
- 与 Lurus Platform 计费链路天然打通（newhub 已经集成）
- 单二进制单升级渠道，运维简单
- Personal 模式用户**无感知**（默认进 Personal 模式，体验不变）

### 负面影响

- **复杂度增加**：路由守卫 + 模式持久化 + 白标资源替换 = 新认知负担
- **测试矩阵扩大 3 倍**：每个特性要在 3 个模式下验证
- **newhub 实战检验少**（创建于今天）：可能踩坑要回头补
- **Agent Fleet 延迟**（已部分代码冷冻 8 周）：部分 sprint 4 工作短期不交付

### 中性影响

- BMAD epic 编号跳跃（E4 部分完成 → E5-E10 冻结 → E12-E15 启用），需在 epics.md 显式标注

---

## Implementation Plan

详见 `transformation-roadmap-v0.4.md` § 3：
- **Phase A** (Sprint 4b, 2 周, 21pt): AppMode 三态 + Hub V1+V2 client SDK
- **Phase B** (Sprint 4c-4d, 4 周, 44pt): Reseller 部署 + 控制台对接
- **Phase C** (Sprint 4e, 2 周, 31pt): EndUser 激活 + 白标打包 + 心跳
- **Phase D** (远期): 自动更新、代码签名、多平台、Cloudflare Workers Hub

**总计**: 96pt / 8 周

---

## Validation / Success Criteria

- [ ] **Phase A 末**: 用户能在三模式间切换；newhub client SDK 单元测试通过
- [ ] **Phase B 末**: 经销商能在 30 分钟内完成"部署 Hub → 配 channel → 生成激活码 → 看到日志"
- [ ] **Phase C 末**: 经销商打包 EndUser .exe → 双击安装 → 输码激活 → 装 Claude Code → 跑通一次请求
- [ ] **8 周后**: 5 个种子经销商签约、内测、给反馈

---

## Open Questions

1. newhub 多租户 → Switch 客户端的 `tenant_slug` 分配机制？
   - **当前假设**：经销商首次部署 Hub 时由 newhub 端创建初始 tenant（admin），slug 由经销商指定（如 `acmecorp`）
   - **待确认**：newhub `/api/v2/admin/tenants` POST endpoint 的 input schema（看 newhub 团队文档）
2. EndUser 心跳频率与离线宽限期默认值？
   - **当前假设**：心跳 5 分钟一次（PRD M7 默认），离线宽限 72 小时
   - **待验证**：内测后调参
3. 白标二进制是 Wails build 时 inject 还是运行时读 sidecar config？
   - **倾向**：运行时读 `whitelabel.json` sidecar（不需重新 build，一次构建多次打包）
   - **风险**：sidecar 易被替换 → 攻击者用经销商品牌打包恶意客户端
   - **缓解**：白标包追加 HMAC 签名（hub_url + brand）+ 启动时 Hub 端验签

---

## References

- PRD: `_bmad-output/planning-artifacts/prd.md`（待写入 v0.1 PRD 终稿）
- Roadmap: `_bmad-output/planning-artifacts/transformation-roadmap-v0.4.md`
- newhub: `https://github.com/hanmahong5-arch/lurus-newhub` ↔ `2b-svc-newhub/`
- newhub V2 API spec: `2b-svc-newhub/internal/adapter/handler/router/api-v2-router.go`
- 关联 ADR: ADR-006 (open-source new-api integration) — **本 ADR 推翻**
- 关联 ADR: ADR-001 (white-label binary form) — **本 ADR 强化**
- 关联 ADR: ADR-002 (cloud Hub deployment) — **本 ADR 实现**
- 关联 ADR: ADR-003 (B2B SaaS pricing model) — **本 ADR 不变**
- 关联 ADR: ADR-004 (plugin agent description) — **本 ADR 推迟到 Phase C+**
- 关联 ADR: ADR-005 (claude-code-router for routing) — **本 ADR 推迟到 newhub 集成 CCR 后再决策**
