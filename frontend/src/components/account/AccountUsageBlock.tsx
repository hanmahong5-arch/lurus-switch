import { useTranslation } from 'react-i18next'
import { Activity, AlertTriangle } from 'lucide-react'
import { cn } from '../../lib/utils'

interface Props {
  /** Total quota (sum granted). Newapi units: quota / 500000 = $1. */
  quota?: number
  /** Used quota (within total). Same units as quota. */
  usedQuota?: number
  /** Daily used (today). Same units. */
  dailyUsed?: number
  /** Estimated month cost in display currency (¥). Marked with ~ in UI. */
  estimatedMonthCost?: number
  /** True backend value for month cost; if set, overrides estimate and drops ~ prefix. */
  monthCost?: number
  loading?: boolean
  onByModel?: () => void
  onByChannel?: () => void
  compact?: boolean
}

const QUOTA_PER_DOLLAR = 500000

// Newapi convention: quota / 500000 ≈ $1. Convert to ¥ at ~7.2 for display.
function quotaToCNY(q?: number): number | undefined {
  if (q == null) return undefined
  return (q / QUOTA_PER_DOLLAR) * 7.2
}

function fmtCNY(v: number | undefined, prefix = ''): string {
  if (v == null || Number.isNaN(v)) return '—'
  return `${prefix}¥${v.toFixed(2)}`
}

export function AccountUsageBlock({
  quota,
  usedQuota,
  dailyUsed,
  estimatedMonthCost,
  monthCost,
  loading = false,
  onByModel,
  onByChannel,
  compact = false,
}: Props) {
  const { t } = useTranslation()

  if (loading) {
    return (
      <div className="space-y-2">
        <div className="h-3 w-16 bg-muted rounded animate-pulse" />
        <div className="h-4 w-32 bg-muted rounded animate-pulse" />
        <div className="h-1.5 w-full bg-muted rounded animate-pulse" />
      </div>
    )
  }

  const pct = (quota != null && quota > 0 && usedQuota != null)
    ? Math.min(100, (usedQuota / quota) * 100)
    : null

  const warn = pct != null && pct >= 80
  const exhausted = pct != null && pct >= 100

  // Prefer real backend month cost; fall back to frontend estimate (marked ~).
  const displayMonth = monthCost != null
    ? fmtCNY(monthCost)
    : fmtCNY(estimatedMonthCost, '~')
  const monthTooltip = monthCost == null
    ? t('account.usage.monthCostEstimated', '本月费用估算 — 基于今日消耗外推,Hub 字段就绪后切换')
    : t('account.usage.monthCost', '本月费用')

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5">
        <Activity className="h-3.5 w-3.5 text-cyan-500" />
        <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground/70">
          {t('account.detail.usage.title', '用量(本月)')}
        </span>
      </div>
      <div className="grid grid-cols-2 gap-2 text-xs">
        <div>
          <p className="text-muted-foreground/70">{t('account.detail.usage.month', '本月')}</p>
          <p className="font-medium tabular-nums" title={monthTooltip}>{displayMonth}</p>
        </div>
        <div>
          <p className="text-muted-foreground/70">{t('account.detail.usage.today', '今日')}</p>
          <p className="font-medium tabular-nums">{fmtCNY(quotaToCNY(dailyUsed))}</p>
        </div>
      </div>
      {pct != null && (
        <div className="space-y-1">
          <div className="flex justify-between text-[10px]">
            <span className="text-muted-foreground/70">
              {t('account.detail.usage.quota', '配额')}
            </span>
            <span className={cn(
              'tabular-nums font-mono',
              exhausted ? 'text-red-500' : warn ? 'text-amber-500' : 'text-muted-foreground',
            )}>
              {pct.toFixed(0)}%
              {warn && <AlertTriangle className="inline h-3 w-3 ml-0.5 -mt-0.5" />}
            </span>
          </div>
          <div className="h-1.5 bg-muted rounded-full overflow-hidden">
            <div
              className={cn(
                'h-full transition-all duration-300',
                exhausted ? 'bg-red-500' : warn ? 'bg-amber-500' : 'bg-primary',
              )}
              style={{ width: `${pct}%` }}
            />
          </div>
        </div>
      )}
      {!compact && (onByModel || onByChannel) && (
        <div className="flex gap-2 pt-1">
          {onByModel && (
            <button
              onClick={onByModel}
              className="text-xs px-2 py-1 rounded-sm bg-muted hover:bg-muted/70 text-muted-foreground"
            >
              {t('account.detail.usage.byModel', '按模型')}
            </button>
          )}
          {onByChannel && (
            <button
              onClick={onByChannel}
              className="text-xs px-2 py-1 rounded-sm bg-muted hover:bg-muted/70 text-muted-foreground"
            >
              {t('account.detail.usage.byChannel', '按渠道')}
            </button>
          )}
        </div>
      )}
    </div>
  )
}
