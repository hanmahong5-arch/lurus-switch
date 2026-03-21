import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useBillingStore } from '../stores/billingStore'
import { useDashboardStore } from '../stores/dashboardStore'
import { useConfigStore } from '../stores/configStore'

export function AccountStatusBadge() {
  const { t } = useTranslation()
  const { userInfo, identityOverview } = useBillingStore()
  const { proxySettings } = useDashboardStore()
  const { setActiveTool } = useConfigStore()

  // No endpoint configured
  if (!proxySettings.apiEndpoint || !proxySettings.userToken) {
    return (
      <span className="text-xs text-muted-foreground/60">{t('account.notConnected')}</span>
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
        onClick={() => { setActiveTool('account'); useConfigStore.getState().setSubTab('account', 'billing') }}
        className="text-xs text-red-500 hover:underline"
      >
        {t('account.subExpired')}
      </button>
    )
  }

  // Check quota exhaustion
  if (quota != null && usedQuota != null && quota > 0 && usedQuota >= quota) {
    return (
      <button
        onClick={() => { setActiveTool('account'); useConfigStore.getState().setSubTab('account', 'billing') }}
        className={cn(
          'text-xs text-red-500 hover:underline',
          'animate-pulse'
        )}
      >
        {t('account.quotaExhausted')}
      </button>
    )
  }

  // Warn when quota >= 80%
  if (quota != null && usedQuota != null && quota > 0) {
    const pct = (usedQuota / quota) * 100
    if (pct >= 80) {
      return (
        <button
          onClick={() => { setActiveTool('account'); useConfigStore.getState().setSubTab('account', 'billing') }}
          className="text-xs text-amber-500 hover:underline"
        >
          {t('account.quotaWarning', { pct: Math.round(pct) })}
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
        onClick={() => { setActiveTool('account'); useConfigStore.getState().setSubTab('account', 'billing') }}
        className="text-xs text-primary hover:underline"
      >
        {t('account.balanceDisplay', { amount: formatted })}
      </button>
    )
  }

  // Fallback: connected but no data yet
  return (
    <span className="text-xs text-muted-foreground/60">{t('account.connected')}</span>
  )
}
