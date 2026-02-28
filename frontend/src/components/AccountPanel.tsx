/**
 * Account Panel Component
 *
 * Displays Lurus identity overview: LurusID, VIP badge, Lubell wallet balance,
 * subscription status, and a top-up button that opens identity.lurus.cn in browser.
 */

import { useEffect } from 'react'
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

// Subscription status display
const SUB_STATUS: Record<string, { color: string; label: string }> = {
  active:  { color: 'text-green-400', label: '订阅中' },
  grace:   { color: 'text-orange-400', label: '宽限期' },
  expired: { color: 'text-red-400', label: '已到期' },
}

function vipColor(level: number): string {
  return VIP_COLORS[level] ?? VIP_COLORS[0]!
}

function subStatus(s: IdentityOverview['subscription']): { color: string; label: string } {
  if (!s) return { color: 'text-gray-400', label: '免费' }
  return SUB_STATUS[s.status] ?? { color: 'text-gray-400', label: s.status }
}

export function AccountPanel() {
  const { identityOverview: ov, setIdentityOverview } = useBillingStore()

  useEffect(() => {
    BillingGetIdentityOverview('lurus-switch')
      .then((data) => setIdentityOverview(data))
      .catch(() => { /* silently degrade if not configured */ })
  }, [setIdentityOverview])

  if (!ov) return null

  const sub = subStatus(ov.subscription ?? null)
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
        <span className="text-xs text-white/50">鹿贝余额</span>
        <span className="font-mono tabular-nums text-sm text-amber-400">
          🦌 {(ov.wallet?.balance ?? 0).toFixed(2)}{' '}
          <span className="text-xs text-white/40">LB</span>
        </span>
      </div>

      {/* Subscription status */}
      <div className="flex items-center justify-between">
        <span className="text-xs text-white/50">订阅状态</span>
        <span className={cn('text-xs font-medium', sub.color)}>{sub.label}</span>
      </div>

      {/* Top-up button */}
      <button
        onClick={handleTopup}
        className="w-full py-1.5 rounded text-xs font-medium bg-amber-500/10 text-amber-400 hover:bg-amber-500/20 active:scale-95 transition-all"
      >
        充值鹿贝
      </button>
    </div>
  )
}
