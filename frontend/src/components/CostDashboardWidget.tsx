import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { DollarSign, Loader2, AlertTriangle, Gauge } from 'lucide-react'
import { GetCostDashboard } from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'
import { Card } from './ui'

// CostDashboardWidget surfaces today's spend, top-cost models, the
// local budget-wall progress, and (when the billing client is wired)
// the remote quota. Refreshes itself every 5s so the user sees the
// counter tick during an active session.
const REFRESH_MS = 5000

type Dash = main.CostDashboard | null

function fmtUSD(v: number): string {
  if (!isFinite(v) || v <= 0) return '$0.00'
  if (v < 0.01) return '< $0.01'
  return '$' + v.toFixed(v < 1 ? 4 : 2)
}

function fmtTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'k'
  return String(n)
}

export function CostDashboardWidget() {
  const { t } = useTranslation()
  const [data, setData] = useState<Dash>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let active = true
    const refresh = () => {
      GetCostDashboard('today')
        .then((r) => {
          if (active) setData(r)
        })
        .catch(() => {
          /* surfaced via empty state */
        })
        .finally(() => {
          if (active) setLoading(false)
        })
    }
    refresh()
    const id = setInterval(refresh, REFRESH_MS)
    return () => {
      active = false
      clearInterval(id)
    }
  }, [])

  if (loading && !data) {
    return (
      <Card className="p-4 flex items-center gap-2 text-xs text-muted-foreground">
        <Loader2 className="h-3 w-3 animate-spin" />
        {t('cost.loading', '加载今日成本…')}
      </Card>
    )
  }
  if (!data) return null

  const byModelTop = [...(data.byModel ?? [])]
    .sort((a, b) => (b.costUSD ?? 0) - (a.costUSD ?? 0))
    .slice(0, 5)
  const maxCost = byModelTop[0]?.costUSD ?? 0

  return (
    <Card data-testid="cost-dashboard" variant="elevated" className="p-4 space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <DollarSign className="h-4 w-4 text-emerald-400" />
          <h3 className="text-sm font-semibold">
            {t('cost.todayTitle', '今日成本（本地估算）')}
          </h3>
        </div>
        <span className="font-mono text-lg tabular-nums text-emerald-400">
          {fmtUSD(data.todayUSD ?? 0)}
        </span>
      </div>
      <p className="text-[10px] text-muted-foreground">
        {t('cost.note', '按公开 USD/Mtok 价格本地估算，非账单。')}
      </p>

      <div className="grid grid-cols-3 gap-2 text-center">
        <div className="rounded bg-card-recessed p-2">
          <p className="text-[10px] uppercase text-muted-foreground">{t('cost.calls', 'calls')}</p>
          <p className="font-mono text-sm tabular-nums">{data.todayCalls ?? 0}</p>
        </div>
        <div className="rounded bg-card-recessed p-2">
          <p className="text-[10px] uppercase text-muted-foreground">tokens in</p>
          <p className="font-mono text-sm tabular-nums">{fmtTokens(data.todayTokensIn ?? 0)}</p>
        </div>
        <div className="rounded bg-card-recessed p-2">
          <p className="text-[10px] uppercase text-muted-foreground">tokens out</p>
          <p className="font-mono text-sm tabular-nums">{fmtTokens(data.todayTokensOut ?? 0)}</p>
        </div>
      </div>

      {byModelTop.length > 0 && (
        <div className="space-y-1.5">
          <p className="text-[10px] uppercase text-muted-foreground font-mono tracking-wide">
            {t('cost.byModel', '按模型成本')}
          </p>
          {byModelTop.map((m) => {
            const cost = m.costUSD ?? 0
            const pct = maxCost > 0 ? Math.round((cost / maxCost) * 100) : 0
            return (
              <div key={m.model} className="space-y-0.5">
                <div className="flex items-baseline gap-2 text-xs">
                  <span className="font-mono tabular-nums truncate flex-1">{m.model || '(unknown)'}</span>
                  <span className="font-mono text-[10px] tabular-nums text-emerald-400">
                    {fmtUSD(cost)}
                  </span>
                </div>
                <div className="h-1 rounded bg-card-recessed overflow-hidden">
                  <div
                    className="h-full bg-emerald-500/60"
                    style={{ width: pct + '%' }}
                  />
                </div>
              </div>
            )
          })}
        </div>
      )}

      {data.budgetEnabled && (data.budgetDailyTokens ?? 0) > 0 && (
        <div className="space-y-1.5 rounded border border-border p-2">
          <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
            <Gauge className="h-3 w-3" />
            <span>{t('cost.budgetWall', '预算墙（日 tokens）')}</span>
            <span className="ml-auto font-mono tabular-nums">
              {fmtTokens(data.budgetDailyUsed ?? 0)} / {fmtTokens(data.budgetDailyTokens ?? 0)}
            </span>
          </div>
          <div className="h-1.5 rounded bg-card-recessed overflow-hidden">
            <div
              className={`h-full ${data.budgetHitDaily ? 'bg-red-500' : (data.budgetDailyPct ?? 0) >= 80 ? 'bg-amber-500' : 'bg-emerald-500'}`}
              style={{ width: Math.min(100, data.budgetDailyPct ?? 0) + '%' }}
            />
          </div>
          {data.budgetHitDaily && (
            <p className="text-[10px] text-red-400 flex items-center gap-1">
              <AlertTriangle className="h-3 w-3" />
              {t('cost.budgetHit', '已触顶 — 进一步请求会被网关拒绝')}
            </p>
          )}
        </div>
      )}

      {data.quota && (
        <div className="space-y-1 rounded border border-border p-2 text-[11px]">
          <div className="flex justify-between text-muted-foreground">
            <span>{t('cost.remoteQuota', '远端配额（hub）')}</span>
            <span className="font-mono tabular-nums">
              {data.quota.used_quota ?? 0} / {data.quota.quota ?? 0}
            </span>
          </div>
        </div>
      )}
      {data.quotaErr && (
        <p className="text-[10px] text-muted-foreground italic">
          {t('cost.quotaUnavailable', '远端配额暂不可用')}: {data.quotaErr}
        </p>
      )}
    </Card>
  )
}
