// Notify API helpers. Mirrors liveSessionApi.ts's `window.go.main.App.*`
// direct-call pattern so we don't gate this feature on `wails generate
// module` succeeding for the new bindings (the generator no-ops silently
// on some Go ASTs in this repo and we've lost half a day to it before).
//
// Backend source of truth:
//   bindings_notify.go       — Wails surface
//   internal/notify/types.go — Event shape
//   internal/notify/store/   — AppConfig persistence

// NotifyBridge is the subset of window.go.main.App this module reaches
// for. We intentionally don't extend the global Window type here — the
// liveSessionApi sibling already declares an incompatible inline shape
// for Window.go.main.App, and TS won't merge two literal-typed
// declarations. A local interface + runtime cast keeps both files
// independent without touching the other module.
interface NotifyBridge {
  GetNotifyConfig?: () => Promise<NotifyConfig>
  SaveNotifyConfig?: (cfg: NotifyConfig) => Promise<void>
  TestNotify?: () => Promise<void>
  GetRecentNotifications?: () => Promise<NotifyEvent[]>
}

function getBridge(): NotifyBridge | undefined {
  if (typeof window === 'undefined') return undefined
  const w = window as unknown as { go?: { main?: { App?: NotifyBridge } } }
  return w.go?.main?.App
}

// NotifyConfig mirrors store.AppConfig in Go. Field casing follows the
// `json` tags on the Go struct — webhookUrl / secret are lowercase camel.
export interface NotifyConfig {
  enabled: boolean
  feishu: FeishuConfig
  rules: NotifyRulesConfig
}

export interface FeishuConfig {
  webhookUrl: string
  secret?: string
}

// NotifyRulesConfig mirrors store.RulesPersist. Durations live as integer
// seconds so the form can edit them as plain numbers (no parsing of
// Go's "1m0s" string form).
export interface NotifyRulesConfig {
  stuckAfterSec: number
  stuckEscalateSec: number
  idleAfterSec: number
  notifyStuck: boolean
  notifyDone: boolean
}

export type NotifyKind =
  | 'tool_stuck'
  | 'session_done'
  | 'budget_alert'
  | 'bashguard_approval'
  | 'test'

export type NotifySeverity = 'info' | 'success' | 'warning' | 'error'

// NotifyEvent mirrors notify.Event in Go. `approval` is omitted because
// the bus's `json:"-"` tag drops it on serialisation; if/when interactive
// approval lands, that field will be surfaced via a separate channel.
export interface NotifyEvent {
  id: string
  time: string
  kind: NotifyKind
  severity: NotifySeverity
  title: string
  body: string
  project?: string
  tool?: string
}

// DEFAULT_CONFIG matches store.DefaultAppConfig in Go. Used by the UI as
// a starting point when the backend isn't available (dev / test) so the
// form doesn't render with undefined fields.
export const DEFAULT_NOTIFY_CONFIG: NotifyConfig = {
  enabled: false,
  feishu: { webhookUrl: '', secret: '' },
  rules: {
    stuckAfterSec: 60,
    stuckEscalateSec: 300,
    idleAfterSec: 300,
    notifyStuck: true,
    notifyDone: true,
  },
}

export async function getNotifyConfig(): Promise<NotifyConfig> {
  const b = getBridge()
  if (!b?.GetNotifyConfig) return DEFAULT_NOTIFY_CONFIG
  return b.GetNotifyConfig()
}

export async function saveNotifyConfig(cfg: NotifyConfig): Promise<void> {
  const b = getBridge()
  if (!b?.SaveNotifyConfig) throw new Error('Wails 桥接不可用 — 请在桌面应用内打开')
  return b.SaveNotifyConfig(cfg)
}

export async function testNotify(): Promise<void> {
  const b = getBridge()
  if (!b?.TestNotify) throw new Error('Wails 桥接不可用 — 请在桌面应用内打开')
  return b.TestNotify()
}

export async function getRecentNotifications(): Promise<NotifyEvent[]> {
  const b = getBridge()
  if (!b?.GetRecentNotifications) return []
  return b.GetRecentNotifications()
}
