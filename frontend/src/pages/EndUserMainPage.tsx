import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Activity, AlertTriangle, Check, Clock, KeyRound, LogOut, RefreshCw, Wrench } from 'lucide-react'
import {
  ClearActivation,
  GetEndUserStatus,
  HeartbeatNow,
} from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'
import { TOOL_ORDER, TOOL_DISPLAY } from '../lib/toolMeta'
import { formatLocal } from '../lib/formatTime'
import { useDashboardStore } from '../stores/dashboardStore'
import { useConfigStore } from '../stores/configStore'

interface Props {
  onDeactivated: () => void
}

// EndUserMainPage — the simplified dashboard a white-label customer sees
// after activation. Shows: brand context (Hub URL + quota), the CLI tools
// they have access to (install/launch only — no advanced config), and a
// "reset activation" affordance for support cases.
//
// Hidden from this page on purpose: gateway settings, channel/token mgmt,
// agent fleet, model catalog editing, anything that implies multi-tenant
// admin. Those live behind RESELLER_ONLY_PAGES.
export function EndUserMainPage({ onDeactivated }: Props) {
  const { t } = useTranslation()
  const [status, setStatus] = useState<main.ActivationStatus | null>(null)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const tools = useDashboardStore((s) => s.tools)
  const setActiveTool = useConfigStore((s) => s.setActiveTool)

  const refresh = async () => {
    setRefreshing(true)
    try {
      const s = await GetEndUserStatus()
      setStatus(s)
    } catch (e) {
      setError(String(e))
    } finally {
      setRefreshing(false)
    }
  }

  useEffect(() => {
    refresh()
  }, [])

  const handleHeartbeat = async () => {
    try {
      await HeartbeatNow()
      await refresh()
    } catch (e) {
      setError(String(e))
    }
  }

  const handleReset = async () => {
    if (!confirm(t('enduser.main.resetConfirm', '确定要重置激活状态吗？此操作会清除本地激活信息，下次启动需要重新输入激活码。'))) {
      return
    }
    try {
      await ClearActivation()
      onDeactivated()
    } catch (e) {
      setError(String(e))
    }
  }

  const isStale = status?.state === 'stale'
  const formatQuota = (n?: number) => {
    if (!n) return '—'
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
    return String(n)
  }
  const formatDate = (iso?: string) => {
    if (!iso) return '—'
    const d = new Date(iso)
    if (isNaN(d.getTime()) || d.getFullYear() < 2000) return '—'
    return formatLocal(d)
  }

  // Days remaining until expiry. null = no valid expiresAt (no countdown).
  // < 0 means already expired (status.state should be revoked, but we still
  // compute it for the banner so a stale UI flash before the next heartbeat
  // doesn't silently hide the expiry).
  const daysUntilExpiry = useMemo(() => {
    const iso = status?.expiresAt
    if (!iso) return null
    const d = new Date(iso)
    if (isNaN(d.getTime()) || d.getFullYear() < 2000) return null
    const diffMs = d.getTime() - Date.now()
    return Math.ceil(diffMs / 86_400_000)
  }, [status?.expiresAt])

  // Banner severity is driven by days remaining + activation present.
  // Unactivated/revoked states are handled elsewhere — this only triggers
  // when the user IS active but the clock is running out.
  const expirySeverity: 'critical' | 'warning' | null = useMemo(() => {
    if (daysUntilExpiry === null) return null
    if (!status?.activated) return null
    if (status.state === 'revoked' || status.state === 'unactivated') return null
    if (daysUntilExpiry <= 7) return 'critical'
    if (daysUntilExpiry <= 30) return 'warning'
    return null
  }, [daysUntilExpiry, status?.activated, status?.state])

  // Quota severity uses absolute thresholds — there's no "used vs total"
  // signal from the Hub yet (heartbeat returns a quota but the store does
  // not persist it), so we colour the remaining number rather than render
  // a fake progress bar. Numbers chosen to align with typical reseller
  // grants (10K = bottom of the cheapest tier, 100K = comfortable).
  const quotaSeverity: 'critical' | 'warning' | 'ok' | null = useMemo(() => {
    const q = status?.quota
    if (q === undefined || q === null) return null
    if (q <= 0) return 'critical'
    if (q < 10_000) return 'critical'
    if (q < 100_000) return 'warning'
    return 'ok'
  }, [status?.quota])

  return (
    <div className="h-full overflow-auto bg-background text-foreground">
      <div className="max-w-4xl mx-auto p-6 space-y-6">
        <header className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold flex items-center gap-2">
              <KeyRound className="h-5 w-5 text-emerald-400" />
              {t('enduser.main.title', '我的服务')}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t('enduser.main.subtitle', '查看额度、连接状态，启动 CLI 工具。')}
            </p>
          </div>
          <button
            onClick={refresh}
            disabled={refreshing}
            className="px-3 py-1.5 rounded border border-border text-sm hover:bg-muted inline-flex items-center gap-1.5 disabled:opacity-50"
          >
            <RefreshCw className={'h-4 w-4 ' + (refreshing ? 'animate-spin' : '')} />
            {t('common.refresh', '刷新')}
          </button>
        </header>

        {expirySeverity === 'critical' && (
          <div
            role="alert"
            data-testid="expiry-banner-critical"
            className="rounded-md border border-red-500/50 bg-red-950/30 px-3 py-2.5 text-xs text-red-100 flex items-start gap-2"
          >
            <Clock className="h-4 w-4 shrink-0 mt-0.5 text-red-300" />
            <div className="flex-1">
              <div className="font-semibold">
                {daysUntilExpiry !== null && daysUntilExpiry <= 0
                  ? t('enduser.main.expiredNow', '激活已到期')
                  : t('enduser.main.expiringCritical', '剩余 {{days}} 天到期', { days: daysUntilExpiry })}
              </div>
              <div className="text-red-100/80 mt-0.5">
                {t(
                  'enduser.main.expiringCriticalHint',
                  '请尽快联系经销商续期，否则服务将停止。',
                )}
              </div>
            </div>
          </div>
        )}

        {expirySeverity === 'warning' && (
          <div
            role="alert"
            data-testid="expiry-banner-warning"
            className="rounded-md border border-amber-500/40 bg-amber-950/20 px-3 py-2 text-xs text-amber-200 flex items-start gap-2"
          >
            <Clock className="h-4 w-4 shrink-0 mt-0.5" />
            <div className="flex-1">
              <div className="font-medium">
                {t('enduser.main.expiringSoon', '剩余 {{days}} 天到期', { days: daysUntilExpiry })}
              </div>
              <div className="text-amber-200/80 mt-0.5">
                {t('enduser.main.expiringSoonHint', '提前与经销商沟通续期，避免服务中断。')}
              </div>
            </div>
          </div>
        )}

        {isStale && (
          <div className="rounded-md border border-amber-500/40 bg-amber-950/20 px-3 py-2 text-xs text-amber-200 flex items-start gap-2">
            <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
            <div className="flex-1">
              <div className="font-medium">{t('enduser.main.stale', 'Hub 连接异常')}</div>
              <div className="text-amber-200/80 mt-0.5">
                {status?.stateReason ?? t('enduser.main.staleHint', '已经一段时间没有联系到 Hub，服务暂时仍可用，但请检查网络。')}
              </div>
            </div>
            <button
              onClick={handleHeartbeat}
              className="text-xs underline hover:no-underline shrink-0"
            >
              {t('enduser.main.retry', '立即重试')}
            </button>
          </div>
        )}

        {error && (
          <div className="rounded-md border border-red-500/30 bg-red-950/20 px-3 py-2 text-xs text-red-200">
            {error}
          </div>
        )}

        {/* Status card */}
        <div className="rounded-lg border border-border bg-card p-5">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <Stat
              icon={<Activity className="h-4 w-4 text-emerald-400" />}
              label={t('enduser.main.status', '服务状态')}
              value={status?.state === 'active'
                ? t('enduser.main.statusActive', '正常')
                : status?.state === 'stale'
                ? t('enduser.main.statusStale', '降级')
                : '—'}
            />
            <Stat
              label={t('enduser.main.quota', '剩余额度')}
              value={formatQuota(status?.quota)}
              severity={quotaSeverity}
            />
            <Stat
              label={t('enduser.main.expires', '到期时间')}
              value={formatDate(status?.expiresAt)}
              hint={
                daysUntilExpiry !== null
                  ? daysUntilExpiry <= 0
                    ? t('enduser.main.expiredNow', '激活已到期')
                    : t('enduser.main.daysRemaining', '{{days}} 天后', { days: daysUntilExpiry })
                  : undefined
              }
              severity={expirySeverity}
            />
            <Stat
              label={t('enduser.main.userId', '账号 ID')}
              value={status?.userId ? `#${status.userId}` : '—'}
            />
          </div>
          {status?.hubUrl && (
            <div className="mt-4 pt-4 border-t border-border/50 text-xs text-muted-foreground font-mono break-all">
              <span className="text-foreground/70">Hub:</span> {status.hubUrl}
              {status.tenantSlug && <> · <span className="text-foreground/70">Tenant:</span> {status.tenantSlug}</>}
            </div>
          )}
        </div>

        {/* Tools */}
        <section>
          <header className="flex items-center gap-2 mb-3">
            <Wrench className="h-4 w-4 text-blue-400" />
            <h2 className="font-medium">{t('enduser.main.tools', 'CLI 工具')}</h2>
          </header>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {TOOL_ORDER.filter((id) => tools?.[id]?.installed).map((id) => {
              const label = TOOL_DISPLAY[id] ?? id
              const tool = tools?.[id]
              return (
                <div key={id} className="border border-border rounded-lg p-4 bg-card">
                  <div className="flex items-center justify-between">
                    <div className="font-medium">{label}</div>
                    {tool?.version && (
                      <div className="text-xs text-muted-foreground font-mono">v{tool.version}</div>
                    )}
                  </div>
                  <button
                    onClick={() => setActiveTool('tools')}
                    className="mt-3 px-3 py-1 rounded border border-border text-xs hover:bg-muted"
                  >
                    {tool?.installed ? t('enduser.main.configure', '配置') : t('enduser.main.install', '安装')}
                  </button>
                </div>
              )
            })}
            {TOOL_ORDER.filter((id) => tools?.[id]?.installed).length === 0 && (
              <div className="col-span-full text-sm text-muted-foreground italic px-2 py-4">
                {t('enduser.main.noTools', '尚未安装任何 CLI 工具，点击「工具」页面安装。')}
                <button
                  onClick={() => setActiveTool('tools')}
                  className="ml-2 underline hover:no-underline"
                >
                  {t('enduser.main.openTools', '打开工具页')}
                </button>
              </div>
            )}
          </div>
        </section>

        {/* Footer actions */}
        <div className="pt-2 border-t border-border/50 flex items-center justify-between text-xs">
          <div className="text-muted-foreground/70">
            {t('enduser.main.activatedAt', '激活时间')}: {formatDate(status?.activatedAt)}
          </div>
          <button
            onClick={handleReset}
            className="px-2 py-1 rounded border border-red-500/40 text-red-300 hover:bg-red-950/20 inline-flex items-center gap-1.5"
          >
            <LogOut className="h-3.5 w-3.5" />
            {t('enduser.main.reset', '重置激活')}
          </button>
        </div>
      </div>
    </div>
  )
}

function Stat({
  icon,
  label,
  value,
  hint,
  severity,
}: {
  icon?: React.ReactNode
  label: string
  value: string
  hint?: string
  severity?: 'critical' | 'warning' | 'ok' | null
}) {
  const valueColour =
    severity === 'critical'
      ? 'text-red-300'
      : severity === 'warning'
      ? 'text-amber-300'
      : severity === 'ok'
      ? 'text-emerald-200'
      : ''
  const hintColour =
    severity === 'critical'
      ? 'text-red-300/80'
      : severity === 'warning'
      ? 'text-amber-300/80'
      : 'text-muted-foreground'
  return (
    <div>
      <div className="text-[11px] text-muted-foreground uppercase tracking-wide mb-1 flex items-center gap-1">
        {icon}
        {label}
      </div>
      <div className={'text-lg font-semibold flex items-center gap-1 ' + valueColour}>
        <Check className="h-4 w-4 text-emerald-400/0" />
        {value}
      </div>
      {hint && <div className={'text-xs mt-0.5 ' + hintColour}>{hint}</div>}
    </div>
  )
}
