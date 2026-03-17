import { cn } from '../lib/utils'
import { useBillingStore } from '../stores/billingStore'
import { useDashboardStore } from '../stores/dashboardStore'
import { useConfigStore } from '../stores/configStore'

export function AccountStatusBadge() {
  const { userInfo, identityOverview } = useBillingStore()
  const { proxySettings } = useDashboardStore()
  const { setActiveTool } = useConfigStore()

  // No endpoint configured → gray "未连接账号"
  if (!proxySettings.apiEndpoint || !proxySettings.userToken) {
    return (
      <span className="text-xs text-muted-foreground/60">未连接账号</span>
    )
  }

  // Quota info from identityOverview (if available)
  const quota = (identityOverview as any)?.quota
  const usedQuota = (identityOverview as any)?.used_quota
  const balance = (userInfo as any)?.balance ?? (userInfo as any)?.quota ?? null

  // Check subscription status
  const subStatus = (identityOverview as any)?.subscription_status

  if (subStatus === 'expired') {
    return (
      <button
        onClick={() => setActiveTool('billing')}
        className="text-xs text-red-500 hover:underline"
      >
        订阅已过期
      </button>
    )
  }

  // Check quota exhaustion
  if (quota != null && usedQuota != null && quota > 0 && usedQuota >= quota) {
    return (
      <button
        onClick={() => setActiveTool('billing')}
        className={cn(
          'text-xs text-red-500 hover:underline',
          'animate-pulse'
        )}
      >
        配额耗尽 !
      </button>
    )
  }

  // Warn when quota ≥ 80%
  if (quota != null && usedQuota != null && quota > 0) {
    const pct = (usedQuota / quota) * 100
    if (pct >= 80) {
      return (
        <button
          onClick={() => setActiveTool('billing')}
          className="text-xs text-amber-500 hover:underline"
        >
          配额 ⚠ {Math.round(pct)}%
        </button>
      )
    }
  }

  // Normal: show balance if available
  if (balance != null) {
    const formatted = typeof balance === 'number'
      ? `¥${(balance / 100).toFixed(2)}`
      : String(balance)
    return (
      <button
        onClick={() => setActiveTool('billing')}
        className="text-xs text-primary hover:underline"
      >
        余额 {formatted}
      </button>
    )
  }

  // Fallback: connected but no data yet
  return (
    <span className="text-xs text-muted-foreground/60">已连接账号</span>
  )
}
