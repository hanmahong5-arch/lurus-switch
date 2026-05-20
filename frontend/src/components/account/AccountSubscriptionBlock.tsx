import { useTranslation } from 'react-i18next'
import { Ticket } from 'lucide-react'

interface Props {
  planName?: string
  planCode?: string
  status?: string
  expiresAt?: string
  autoRenew?: boolean
  loading?: boolean
  onRenew?: () => void
  onUpgrade?: () => void
  compact?: boolean
}

function fmtDate(s?: string): string {
  if (!s) return '—'
  // ISO 8601 (e.g. 2026-08-15T00:00:00Z) → YYYY-MM-DD
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return s
  return d.toISOString().slice(0, 10)
}

export function AccountSubscriptionBlock({
  planName,
  planCode,
  status,
  expiresAt,
  autoRenew,
  loading = false,
  onRenew,
  onUpgrade,
  compact = false,
}: Props) {
  const { t } = useTranslation()

  if (loading) {
    return (
      <div className="space-y-2">
        <div className="h-3 w-16 bg-muted rounded animate-pulse" />
        <div className="h-4 w-28 bg-muted rounded animate-pulse" />
      </div>
    )
  }

  const display = planName || planCode || t('account.detail.subscription.free', '免费')
  const statusLabel = status
    ? t(`account.subStatus.${status}`, status)
    : t('account.detail.subscription.free', '免费')

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5">
        <Ticket className="h-3.5 w-3.5 text-violet-500" />
        <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground/70">
          {t('account.detail.subscription.title', '订阅')}
        </span>
      </div>
      <div className="grid grid-cols-2 gap-2 text-xs">
        <div>
          <p className="text-muted-foreground/70">{t('account.detail.subscription.plan', '计划')}</p>
          <p className="font-medium">{display}</p>
        </div>
        <div>
          <p className="text-muted-foreground/70">
            {status === 'expired'
              ? t('account.detail.subscription.expired', '已到期')
              : t('account.detail.subscription.expires', '到期')}
          </p>
          <p className="font-medium tabular-nums">{fmtDate(expiresAt)}</p>
        </div>
      </div>
      {autoRenew && (
        <p className="text-[10px] text-emerald-500">
          {t('account.detail.subscription.autoRenew', '已开启自动续费')}
        </p>
      )}
      {!compact && (
        <div className="flex gap-2">
          {onRenew && (
            <button
              onClick={onRenew}
              className="text-xs px-2 py-1 rounded-sm bg-primary/10 text-primary hover:bg-primary/15"
            >
              {t('account.detail.subscription.renew', '续费')}
            </button>
          )}
          {onUpgrade && (
            <button
              onClick={onUpgrade}
              className="text-xs px-2 py-1 rounded-sm bg-muted hover:bg-muted/70 text-muted-foreground"
            >
              {t('account.detail.subscription.upgrade', '升级')}
            </button>
          )}
        </div>
      )}
      {!compact && status === 'free' && !onUpgrade && (
        <p className="text-[10px] text-muted-foreground/60">{statusLabel}</p>
      )}
    </div>
  )
}
