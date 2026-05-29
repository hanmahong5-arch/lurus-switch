# Lurus Switch — 真经销商 Onboarding 实跑手册

> 受众:首位真实经销商。从"装 newhub 实例"到"终端客户激活成功"的 6 步实跑。
> 每步引用 Switch 真实 Wails binding(file:line)+ `doc/test-plan-distribution.md` 节号。
> 实测前提(2026-05-28,只读 curl):`hub.lurus.cn/` 返回 200;`POST /api/v2/switch/redeem` 返回 400(路由已 wired,空 body 被校验拒绝)。

## 前置(test-plan §0)
- Windows 运行 `lurus-switch.exe`(经销商机)
- logo PNG/SVG ≤256KB + 一个 `.ico` ≤256×256
- 经销商专属 newhub 的 Hub URL + Admin Token + Tenant Slug(第 1 步产出)
- 真 redemption code(第 5 步 Hub 侧生成)

## 第 1 步:部署/接入经销商 newhub
Reseller 必须独立实例(Personal 才走自营 `hub.lurus.cn`)。Switch 内 Provider 仅 `Manual` 为 `Implemented:true`(`bindings_reseller.go:45` `resellerKindCatalog`,Sealos/Aliyun 标 false),所以经销商先自行部署 newhub,拿到 URL + Admin Token + Tenant Slug。
**验证**:`curl -s -o /dev/null -w "%{http_code}" https://<reseller-hub>/` 非 5xx。

## 第 2 步:Reseller 模式设置(Wizard)
- `ListResellerDeployKinds()`(`bindings_reseller.go:73`)— 列 Provider,选 Manual
- `TestHubConnection(hubURL, token)`(`bindings_reseller.go:86`)— 「测试连接」,内部跑 `admin.ListChannels(page=1,size=1)`,8s 超时
- `ProvisionResellerHub(kind, displayName, hubURL, adminToken, tenantSlug)`(`bindings_reseller.go:129`)— 「保存配置」,幂等

**验证(§8.1)**:返回 `连接成功 · 当前 Hub 共 N 个 channel`;`app-settings.json` 新增 `reseller.hubUrl/.adminToken/.tenantSlug`;`HasResellerConfig()`(`:185`)返回 true。

## 第 3 步:WhiteLabelPreflight 预检
`WhiteLabelPreflight(hubURL, tenantSlug)`(`bindings_whitelabel.go:464`),各 5s 超时探 4 项:
1. `hub-root` HEAD Hub 根(<500 过)
2. `redeem` POST `/api/v2/switch/redeem`(`:485`;404=未实现,其他=路由存在)
3. `heartbeat` POST `/api/v2/switch/heartbeat`(或 `/api/v2/<slug>/user/heartbeat`)
4. `reseller-cfg` 本机 HubURL+AdminToken 已填

**验证(§8.2)**:全过→绿「全部通过」;失败→黄 banner + HTTP 码。实测 redeem 返回 400 即 PASS。

## 第 4 步:BuildWhiteLabelPackage 导出
`BuildWhiteLabelPackage(WhiteLabelInputs)`(`bindings_whitelabel.go:75`)。PackagerPage 填品牌名/Hub URL/主题色/logo/`.ico`,4 步进度。产物落 `OutputDir`(空则 `<appdata>/lurus-switch/whitelabel-builds/<brand-slug>`):品牌化 `.exe` + 签名 `whitelabel.json`。HMAC key 优先 Hub `/api/v2/admin/whitelabel/hmac-key`,失败回退 baked secret(`whitelabelHMACKey()` `:356`)。

**验证(§8.3)**:输出区显示 OutputDir/Binary/Sidecar/BinarySHA256/SidecarSHA256;`OpenWhiteLabelOutputDir`(`:579`)开目录;`ZipWhiteLabelOutputDir`(`:589`)生成 zip;`ListWhiteLabelBuilds`(`:680`)加一行。

## 第 5 步:发包 + 生成激活码
- **发包**:OutputDir/zip 交付客户。客户清空 `%APPDATA%\lurus-switch\` 后双击 exe,启动 `applyWhiteLabelSidecar()`(`bindings_whitelabel.go:393`)验签 → 锁 EndUser 模式 + 锁 Hub URL,不弹模式选择器。
- **激活码**:在 newhub 侧用 Admin Token 经 V1 `/api/redemption/*` 创建(生成是 Hub 侧操作,Switch 仅消费,无对应 binding)。

**验证(§8.4 关键 / §8.5)**:§8.4 干净机启动品牌资产正确、模式锁 EndUser、Hub URL read-only;§8.5 改 `whitelabel.json` hub_url 不动 hmac → stderr `sidecar rejected`,回落模式选择器。

## 第 6 步:激活码兑换(终端客户)
`ActivateRedemption(code)`(`bindings_enduser.go:87`)。Hub URL 由本机锁定值供(`resolveEndUserHubURL` `:153`,不收前端传入),POST `/api/v2/switch/redeem`(`internal/redemption/redeem.go:29`,body `{code,fingerprint,app_version}`),成功后持久化 + 起心跳。辅助:`GetEndUserStatus()`(`:70`)、`HeartbeatNow()`(`:136`)、`GetDeviceFingerprint()`(`:63`)。

**验证(§9.1/§9.2)**:§9.1 真码→跳 EndUser 主页,重启仍激活(指纹绑定);§9.2 假码按 `[kind=...]` 映射文案,kind 常量已确认(`redeem.go:63-66`):`code_not_found`/`code_used`/`code_expired`/`code_disabled`。

## 退路(§12)
未签名→SmartScreen 弹窗(已知限制);icon 替换 base 已签名会报错,非 PE silent skip,均不阻断打包。

---
_注:激活码生成无 Switch binding(Hub 侧 `/api/redemption/*` 操作);所有 binding file:line 与 test-plan 节号经 grep 实测确认存在(2026-05-28)。手册不替代真经销商落点决策(经销商专属 newhub 落点 + Zitadel tenant 注册方仍 owner-gated)。_
