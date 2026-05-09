// Metadata registry for newapi (`/api/option/`) keys. Drives the rich
// settings UI in GatewaySettingsPage: each entry knows its widget type,
// human label, description, and which tab + sub-group it belongs to.
//
// Keys not present in this registry fall back to the generic OptionEditor
// (auto-detect boolean/number/json/text). Add entries here to upgrade an
// option to the labeled-form treatment.

export type OptionWidget = 'text' | 'number' | 'boolean' | 'textarea' | 'select' | 'pricing-map'

export type SettingsTab =
  | 'operations'
  | 'dashboard'
  | 'chat'
  | 'drawing'
  | 'payment'
  | 'pricing'
  | 'rateLimit'
  | 'modelConfig'
  | 'modelDeploy'
  | 'performance'
  | 'system'
  | 'other'

export interface SelectChoice {
  value: string
  labelKey: string
  labelFallback: string
}

export interface OptionMeta {
  key: string
  tab: SettingsTab
  /** Sub-section identifier within the tab (used to group rows visually). */
  group: string
  widget: OptionWidget
  /** i18n keys + fallback strings — labels are always shown to the user. */
  labelKey: string
  labelFallback: string
  descKey?: string
  descFallback?: string
  placeholder?: string
  /** For 'select' widget. */
  choices?: SelectChoice[]
  /** For 'number' widget — soft validation hints. */
  min?: number
  max?: number
  /** For 'pricing-map' — what does each ratio mean (input price multiplier, etc). */
  unitHint?: string
}

// Group identifiers per tab — used to render sub-section headers.
export const SETTINGS_GROUPS: Record<SettingsTab, Array<{ id: string; labelKey: string; labelFallback: string }>> = {
  operations: [
    { id: 'general', labelKey: 'gateway.settings.group.general', labelFallback: '通用设置' },
    { id: 'quota', labelKey: 'gateway.settings.group.quota', labelFallback: '配额设置' },
    { id: 'auth', labelKey: 'gateway.settings.group.auth', labelFallback: '注册与登录' },
    { id: 'branding', labelKey: 'gateway.settings.group.branding', labelFallback: '品牌与文案' },
  ],
  dashboard: [{ id: 'general', labelKey: 'gateway.settings.group.general', labelFallback: '通用设置' }],
  chat: [{ id: 'general', labelKey: 'gateway.settings.group.general', labelFallback: '通用设置' }],
  drawing: [{ id: 'general', labelKey: 'gateway.settings.group.general', labelFallback: '通用设置' }],
  payment: [
    { id: 'general', labelKey: 'gateway.settings.group.general', labelFallback: '通用设置' },
    { id: 'epay', labelKey: 'gateway.settings.group.epay', labelFallback: 'Epay 支付' },
    { id: 'stripe', labelKey: 'gateway.settings.group.stripe', labelFallback: 'Stripe 支付' },
    { id: 'creem', labelKey: 'gateway.settings.group.creem', labelFallback: 'Creem 支付' },
    { id: 'waffo', labelKey: 'gateway.settings.group.waffo', labelFallback: 'Waffo 支付' },
  ],
  pricing: [{ id: 'general', labelKey: 'gateway.settings.group.general', labelFallback: '价格设置' }],
  rateLimit: [{ id: 'general', labelKey: 'gateway.settings.group.general', labelFallback: '速率限制' }],
  modelConfig: [{ id: 'general', labelKey: 'gateway.settings.group.general', labelFallback: '模型行为' }],
  modelDeploy: [{ id: 'general', labelKey: 'gateway.settings.group.general', labelFallback: '部署设置' }],
  performance: [{ id: 'general', labelKey: 'gateway.settings.group.general', labelFallback: '通用设置' }],
  system: [
    { id: 'email', labelKey: 'gateway.settings.group.email', labelFallback: '邮件 / SMTP' },
    { id: 'oauth', labelKey: 'gateway.settings.group.oauth', labelFallback: '第三方登录' },
    { id: 'turnstile', labelKey: 'gateway.settings.group.turnstile', labelFallback: '人机验证' },
    { id: 'storage', labelKey: 'gateway.settings.group.storage', labelFallback: '文件与图片权限' },
    { id: 'general', labelKey: 'gateway.settings.group.general', labelFallback: '通用设置' },
  ],
  other: [{ id: 'general', labelKey: 'gateway.settings.group.general', labelFallback: '其他' }],
}

// Per-key metadata. Order within a group is preserved for rendering.
export const OPTION_META: OptionMeta[] = [
  // === Operations · General ===
  {
    key: 'TopUpLink',
    tab: 'operations', group: 'general', widget: 'text',
    labelKey: 'gateway.settings.opt.TopUpLink', labelFallback: '充值链接',
    descKey: 'gateway.settings.opt.TopUpLink.desc',
    descFallback: '用户在 Web 控制台点击"充值"时跳转的外部链接，例如发卡网站。',
    placeholder: 'https://…',
  },
  {
    key: 'ChatLink',
    tab: 'operations', group: 'general', widget: 'text',
    labelKey: 'gateway.settings.opt.ChatLink', labelFallback: '聊天链接',
    descKey: 'gateway.settings.opt.ChatLink.desc',
    descFallback: '主聊天入口 URL（例如自部署的 NextChat / LobeChat）。',
    placeholder: 'https://chat.example.com',
  },
  {
    key: 'ChatLink2',
    tab: 'operations', group: 'general', widget: 'text',
    labelKey: 'gateway.settings.opt.ChatLink2', labelFallback: '聊天链接（备用）',
    descFallback: '第二个聊天入口，可选。',
  },
  {
    key: 'RetryTimes',
    tab: 'operations', group: 'general', widget: 'number',
    labelKey: 'gateway.settings.opt.RetryTimes', labelFallback: '失败重试次数',
    descFallback: '上游请求失败时的最大重试次数（不含首次请求）。',
    min: 0, max: 10,
  },
  {
    key: 'DisplayInCurrencyEnabled',
    tab: 'operations', group: 'general', widget: 'boolean',
    labelKey: 'gateway.settings.opt.DisplayInCurrencyEnabled',
    labelFallback: '以美元 ($) 展示额度',
    descFallback: '关闭则显示原始 token 配额数字。',
  },
  {
    key: 'DisplayTokenStatEnabled',
    tab: 'operations', group: 'general', widget: 'boolean',
    labelKey: 'gateway.settings.opt.DisplayTokenStatEnabled',
    labelFallback: '额度查询接口返回令牌额度而非用户额度',
    descFallback: '影响 /api/user/self 等接口的 quota 字段语义。',
  },
  {
    key: 'DefaultCollapseSidebar',
    tab: 'operations', group: 'general', widget: 'boolean',
    labelKey: 'gateway.settings.opt.DefaultCollapseSidebar',
    labelFallback: '默认折叠侧边栏',
    descFallback: '新用户首次访问时左侧栏默认状态。',
  },
  {
    key: 'DemoSiteEnabled',
    tab: 'operations', group: 'general', widget: 'boolean',
    labelKey: 'gateway.settings.opt.DemoSiteEnabled',
    labelFallback: '演示站点模式',
    descFallback: '开启后写操作（删除、修改密码等）会被拒绝，适合公开演示部署。',
  },
  {
    key: 'SelfUseModeEnabled',
    tab: 'operations', group: 'general', widget: 'boolean',
    labelKey: 'gateway.settings.opt.SelfUseModeEnabled',
    labelFallback: '自用模式',
    descFallback: '开启后跳过模型倍率计算，所有请求直通 — 必须先配置好渠道倍率。',
  },
  {
    key: 'ApproximateTokenEnabled',
    tab: 'operations', group: 'general', widget: 'boolean',
    labelKey: 'gateway.settings.opt.ApproximateTokenEnabled',
    labelFallback: '允许使用近似 Token 估算',
    descFallback: '部分上游不返回精确 token 数时使用启发式估算。',
  },
  {
    key: 'DefaultUseAutoGroup',
    tab: 'operations', group: 'general', widget: 'boolean',
    labelKey: 'gateway.settings.opt.DefaultUseAutoGroup',
    labelFallback: '默认自动分组',
    descFallback: '新用户默认走自动分组（按价格/质量动态选择渠道）。',
  },

  // === Operations · Quota ===
  {
    key: 'QuotaForNewUser',
    tab: 'operations', group: 'quota', widget: 'number',
    labelKey: 'gateway.settings.opt.QuotaForNewUser', labelFallback: '新用户初始额度',
    descFallback: '注册时赠送给新用户的额度（Token 数）。',
    min: 0,
  },
  {
    key: 'QuotaForInviter',
    tab: 'operations', group: 'quota', widget: 'number',
    labelKey: 'gateway.settings.opt.QuotaForInviter', labelFallback: '邀请人奖励',
    descFallback: '被邀请的用户完成注册后给邀请人的奖励额度。',
    min: 0,
  },
  {
    key: 'QuotaForInvitee',
    tab: 'operations', group: 'quota', widget: 'number',
    labelKey: 'gateway.settings.opt.QuotaForInvitee', labelFallback: '受邀人奖励',
    descFallback: '使用邀请码注册的用户额外获得的奖励额度。',
    min: 0,
  },
  {
    key: 'QuotaRemindThreshold',
    tab: 'operations', group: 'quota', widget: 'number',
    labelKey: 'gateway.settings.opt.QuotaRemindThreshold', labelFallback: '余额预警阈值',
    descFallback: '用户余额低于此值时邮件提醒（Token 数）。',
    min: 0,
  },
  {
    key: 'PreConsumedQuota',
    tab: 'operations', group: 'quota', widget: 'number',
    labelKey: 'gateway.settings.opt.PreConsumedQuota', labelFallback: '预扣额度',
    descFallback: '请求开始时先扣除的额度，结算后多退少补。',
    min: 0,
  },

  // === Operations · Auth ===
  {
    key: 'RegisterEnabled',
    tab: 'operations', group: 'auth', widget: 'boolean',
    labelKey: 'gateway.settings.opt.RegisterEnabled', labelFallback: '允许注册',
    descFallback: '关闭后只有管理员可以创建用户。',
  },
  {
    key: 'PasswordLoginEnabled',
    tab: 'operations', group: 'auth', widget: 'boolean',
    labelKey: 'gateway.settings.opt.PasswordLoginEnabled', labelFallback: '允许密码登录',
  },
  {
    key: 'PasswordRegisterEnabled',
    tab: 'operations', group: 'auth', widget: 'boolean',
    labelKey: 'gateway.settings.opt.PasswordRegisterEnabled', labelFallback: '允许密码注册',
  },

  // === Operations · Branding ===
  {
    key: 'SystemName',
    tab: 'operations', group: 'branding', widget: 'text',
    labelKey: 'gateway.settings.opt.SystemName', labelFallback: '站点名称',
    placeholder: 'My API',
  },
  {
    key: 'Logo',
    tab: 'operations', group: 'branding', widget: 'text',
    labelKey: 'gateway.settings.opt.Logo', labelFallback: 'Logo URL',
    placeholder: 'https://…/logo.png',
  },
  {
    key: 'Footer',
    tab: 'operations', group: 'branding', widget: 'textarea',
    labelKey: 'gateway.settings.opt.Footer', labelFallback: '页脚 HTML',
    descFallback: '原始 HTML 片段，渲染在站点底部。',
  },
  {
    key: 'Notice',
    tab: 'operations', group: 'branding', widget: 'textarea',
    labelKey: 'gateway.settings.opt.Notice', labelFallback: '公告',
    descFallback: '首页顶部公告，支持 Markdown。',
  },
  {
    key: 'About',
    tab: 'operations', group: 'branding', widget: 'textarea',
    labelKey: 'gateway.settings.opt.About', labelFallback: '关于页面',
    descFallback: '"关于"页面内容，支持 Markdown / HTML。',
  },
  {
    key: 'HomePageContent',
    tab: 'operations', group: 'branding', widget: 'textarea',
    labelKey: 'gateway.settings.opt.HomePageContent', labelFallback: '首页内容',
    descFallback: '首页主体内容，支持 Markdown。',
  },

  // === Pricing · General — special pricing-map widget per ratio key ===
  {
    key: 'ModelRatio',
    tab: 'pricing', group: 'general', widget: 'pricing-map',
    labelKey: 'gateway.settings.opt.ModelRatio', labelFallback: '模型倍率',
    descFallback: 'model_id → 倍率（相对 USD $0.002/1K tokens 的 input 价格）。',
    unitHint: '相对默认价（USD $0.002/1K tokens 输入）的乘数',
  },
  {
    key: 'CompletionRatio',
    tab: 'pricing', group: 'general', widget: 'pricing-map',
    labelKey: 'gateway.settings.opt.CompletionRatio', labelFallback: '补全倍率',
    descFallback: 'model_id → 输出价格相对输入价格的倍数。',
    unitHint: '输出 / 输入价格比（多数模型 = 2 或 3）',
  },
  {
    key: 'GroupRatio',
    tab: 'pricing', group: 'general', widget: 'pricing-map',
    labelKey: 'gateway.settings.opt.GroupRatio', labelFallback: '分组倍率',
    descFallback: 'group_name → 折扣倍数（用户实际付价 = 倍率 × 分组倍率）。',
    unitHint: '分组折扣（< 1 是优惠，> 1 是溢价）',
  },
  {
    key: 'ModelPrice',
    tab: 'pricing', group: 'general', widget: 'pricing-map',
    labelKey: 'gateway.settings.opt.ModelPrice', labelFallback: '按次定价',
    descFallback: 'model_id → 每次调用价格（USD），用于不按 token 计费的模型如 MJ。',
    unitHint: '每次请求收费（USD）',
  },
  {
    key: 'CacheRatio',
    tab: 'pricing', group: 'general', widget: 'pricing-map',
    labelKey: 'gateway.settings.opt.CacheRatio', labelFallback: '缓存读取倍率',
    descFallback: 'model_id → cache hit 时 input 价格的倍数（通常 < 1）。',
    unitHint: '缓存命中价 / 普通输入价（OpenAI 默认 0.5）',
  },
]

export function getOptionMeta(key: string): OptionMeta | undefined {
  return OPTION_META.find((m) => m.key === key)
}

export function metaForTab(tab: SettingsTab): OptionMeta[] {
  return OPTION_META.filter((m) => m.tab === tab)
}

/**
 * Returns the set of all tab IDs each known key falls under. Used by the
 * settings page to route an option to its tab without doing string-prefix
 * matching guesses.
 */
export function buildKeyToTabIndex(): Record<string, SettingsTab> {
  const out: Record<string, SettingsTab> = {}
  for (const m of OPTION_META) out[m.key] = m.tab
  return out
}
