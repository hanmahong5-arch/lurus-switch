/**
 * Account Panel Component
 *
 * Displays Lurus identity overview: LurusID, VIP badge, Lubell wallet balance,
 * subscription status, and a top-up button that opens identity.lurus.cn in browser.
 */

import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { BillingGetIdentityOverview, BillingOpenTopup } from '../../wailsjs/go/main/App'
import { useBillingStore, type IdentityOverview } from '../stores/billingStore'
import { cn } from '../lib/utils'

// VIP level → display color class (dark Tailwind theme)
const VIP_COLORS: Record<number, string> = {
  0: 'text-gray-400',
  1: 'text-slate-300',   // Silver
  2: 'text-amber-400',   // Gold
  3: 'text-indigo-300',  // Platinum
  4: 'text-cyan-300',    // Diamond
}

// Subscription status → color class + i18n key
const SUB_STATUS: Record<string, { color: string; key: string }> = {
  active:  { color: 'text-green-400', key: 'account.subStatus.active' },
  grace:   { color: 'text-orange-400', key: 'account.subStatus.grace' },
  expired: { color: 'text-red-400', key: 'account.subStatus.expired' },
}

function vipColor(level: number): string {
  return VIP_COLORS[level] ?? VIP_COLORS[0]!
}

export function AccountPanel() {
  const { t } = useTranslation()
  const { identityOverview: ov, setIdentityOverview } = useBillingStore()

  useEffect(() => {
    BillingGetIdentityOverview('lurus-switch')
      .then((data) => setIdentityOverview(data))
      .catch(() => { /* silently degrade if not configured */ })
  }, [setIdentityOverview])

  if (!ov) return null

  const subEntry = ov.subscription ? SUB_STATUS[ov.subscription.status] : null
  const subColor = subEntry?.color ?? 'text-gray-400'
  const subLabel = subEntry ? t(subEntry.key) : t('account.subStatus.free')
  const vipCls = vipColor(ov.vip?.level ?? 0)

  function handleTopup() {
    BillingOpenTopup(ov!.topup_url).catch(() => {/* ignore */})
  }

  return (
    <div className="rounded-lg border border-white/10 bg-white/5 p-3 space-y-2">
      {/* Header: LurusID + VIP */}
      <div className="flex items-center justify-between">
        <span className="text-xs text-white/50 font-mono">{ov.account?.lurus_id ?? '—'}</span>
        <span className={cn('text-xs font-semibold', vipCls)}>
          {ov.vip?.level_name ?? 'Standard'}
        </span>
      </div>

      {/* Lubell wallet balance */}
      <div className="flex items-center justify-between">
        <span className="text-xs text-white/50">{t('account.balance')}</span>
        <span className="font-mono tabular-nums text-sm text-amber-400">
          🦌 {(ov.wallet?.balance ?? 0).toFixed(2)}{' '}
          <span className="text-xs text-white/40">LB</span>
        </span>
      </div>

      {/* Subscription status */}
      <div className="flex items-center justify-between">
        <span className="text-xs text-white/50">{t('account.subscriptionStatus')}</span>
        <span className={cn('text-xs font-medium', subColor)}>{subLabel}</span>
      </div>

      {/* Top-up button */}
      <button
        onClick={handleTopup}
        className="w-full py-1.5 rounded text-xs font-medium bg-amber-500/10 text-amber-400 hover:bg-amber-500/20 active:scale-95 transition-all"
      >
        {t('account.topUp')}
      </button>
    </div>
  )
}
