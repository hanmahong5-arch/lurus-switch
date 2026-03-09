import { useEffect, useState } from 'react'
import { Loader2, ExternalLink, WifiOff } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useDashboardStore } from '../stores/dashboardStore'
import { useConfigStore } from '../stores/configStore'
import { BillingGetQuotaSummary } from '../../wailsjs/go/main/App'

interface QuotaData {
  quota: number
  used_quota: number
  remaining_quota: number
  daily_quota: number
  daily_used: number
}

export function DashboardQuotaWidget() {
  const { t } = useTranslation()
  const { proxySettings } = useDashboardStore()
  const { setActiveTool } = useConfigStore()
  const [data, setData] = useState<QuotaData | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(false)

  const hasToken = !!proxySettings.userToken

  useEffect(() => {
    if (!hasToken) return
    setLoading(true)
    setError(false)
    BillingGetQuotaSummary()
      .then((r) => {
        if (r && typeof r === 'object' && 'quota' in r && 'used_quota' in r) {
          setData(r as QuotaData)
        } else {
          setError(true)
        }
      })
      .catch(() => setError(true))
      .finally(() => setLoading(false))
  }, [hasToken])

  // Not connected state
  if (!hasToken) {
    return (
      <div className="border border-border rounded-lg p-4 bg-card flex items-center justify-between">
        <div>
          <h3 className="text-sm font-medium">{t('dashboard.quota.title')}</h3>
          <p className="text-xs text-muted-foreground mt-0.5">{t('dashboard.quota.connectDesc')}</p>
        </div>
        <button
          onClick={() => setActiveTool('billing')}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
            'bg-primary text-primary-foreground hover:bg-primary/90'
          )}
        >
          <ExternalLink className="h-3.5 w-3.5" />
          {t('dashboard.quota.connectAccount')}
        </button>
      </div>
    )
  }

  // Loading / error state
  if (loading) {
    return (
      <div className="border border-border rounded-lg p-4 bg-card flex items-center gap-2">
        <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        <span className="text-sm text-muted-foreground">{t('dashboard.quota.title')}</span>
      </div>
    )
  }

  if (error || !data) {
    return (
      <div className="border border-border rounded-lg p-4 bg-card flex items-center gap-2 text-muted-foreground">
        <WifiOff className="h-4 w-4" />
        <span className="text-sm">{t('dashboard.quota.offline')}</span>
      </div>
    )
  }

  // Connected + data
  const totalPercent = data.quota > 0 ? Math.min(100, (data.used_quota / data.quota) * 100) : 0
  const dailyPercent = data.daily_quota > 0 ? Math.min(100, (data.daily_used / data.daily_quota) * 100) : 0

  const formatQuota = (n: number) => {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
    return n.toString()
  }

  return (
    <div className="border border-border rounded-lg p-4 bg-card space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">{t('dashboard.quota.title')}</h3>
        <button
          onClick={() => setActiveTool('billing')}
          className="text-xs text-primary hover:underline"
        >
          {t('dashboard.quota.details')}
        </button>
      </div>

      {/* Total usage */}
      <div className="space-y-1">
        <div className="flex justify-between text-xs text-muted-foreground">
          <span>{t('dashboard.quota.totalUsage')}</span>
          <span>
            {formatQuota(data.used_quota)} / {data.quota > 0 ? formatQuota(data.quota) : t('dashboard.quota.unlimited')}
          </span>
        </div>
        <div className="h-2 bg-muted rounded-full overflow-hidden">
          <div
            className={cn(
              'h-full rounded-full transition-all',
              totalPercent > 90 ? 'bg-red-500' : totalPercent > 70 ? 'bg-amber-500' : 'bg-primary'
            )}
            style={{ width: `${totalPercent}%` }}
          />
        </div>
      </div>

      {/* Daily usage */}
      <div className="space-y-1">
        <div className="flex justify-between text-xs text-muted-foreground">
          <span>{t('dashboard.quota.dailyUsage')}</span>
          <span>
            {formatQuota(data.daily_used)} / {data.daily_quota > 0 ? formatQuota(data.daily_quota) : t('dashboard.quota.unlimited')}
          </span>
        </div>
        <div className="h-1.5 bg-muted rounded-full overflow-hidden">
          <div
            className={cn(
              'h-full rounded-full transition-all',
              dailyPercent > 90 ? 'bg-red-500' : dailyPercent > 70 ? 'bg-amber-500' : 'bg-blue-500'
            )}
            style={{ width: `${dailyPercent}%` }}
          />
        </div>
      </div>
    </div>
  )
}
