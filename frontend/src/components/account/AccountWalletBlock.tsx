import { useTranslation } from 'react-i18next'
import { Wallet, ArrowUpRight } from 'lucide-react'
import { cn } from '../../lib/utils'

interface Props {
  balance?: number
  frozen?: number
  currency?: string
  loading?: boolean
  onTopUp?: () => void
  onHistory?: () => void
  /** Compact: single-row layout (no action buttons). */
  compact?: boolean
}

function fmt(v: number | undefined, currency: string): string {
  if (v == null || Number.isNaN(v)) return '—'
  const sym = currency === 'CNY' ? '¥' : (currency || '¥')
  return `${sym}${v.toFixed(2)}`
}

export function AccountWalletBlock({
  balance,
  frozen,
  currency = 'CNY',
  loading = false,
  onTopUp,
  onHistory,
  compact = false,
}: Props) {
  const { t } = useTranslation()

  if (loading) {
    return (
      <div className={cn('space-y-2', compact && 'space-y-1')}>
        <div className="h-3 w-16 bg-muted rounded animate-pulse" />
        <div className="h-5 w-24 bg-muted rounded animate-pulse" />
      </div>
    )
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5">
        <Wallet className="h-3.5 w-3.5 text-emerald-500" />
        <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground/70">
          {t('account.detail.wallet.title', '钱包')}
        </span>
      </div>
      <div className="flex items-baseline gap-2">
        <span className="text-lg font-semibold tabular-nums">{fmt(balance, currency)}</span>
        {frozen != null && frozen > 0 && (
          <span className="text-[10px] text-muted-foreground">
            {t('account.detail.wallet.frozen', '冻结 {{amount}}', { amount: fmt(frozen, currency) })}
          </span>
        )}
      </div>
      {!compact && (
        <div className="flex gap-2">
          {onTopUp && (
            <button
              onClick={onTopUp}
              className="text-xs px-2 py-1 rounded-sm bg-primary/10 text-primary hover:bg-primary/15 inline-flex items-center gap-1"
            >
              <ArrowUpRight className="h-3 w-3" />
              {t('account.detail.wallet.topUp', '充值')}
            </button>
          )}
          {onHistory && (
            <button
              onClick={onHistory}
              className="text-xs px-2 py-1 rounded-sm bg-muted hover:bg-muted/70 text-muted-foreground"
            >
              {t('account.detail.wallet.history', '交易明细')}
            </button>
          )}
        </div>
      )}
    </div>
  )
}
