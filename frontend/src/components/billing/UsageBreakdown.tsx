import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
import { GetAppSummaries, GetModelSummaries } from '../../../wailsjs/go/main/App'
import type { metering } from '../../../wailsjs/go/models'
import { TOOL_DISPLAY } from '../../lib/toolMeta'

// Donut palette — chosen for contrast against bg-card (dark slate) and the
// existing primary/amber/red used by QuotaCard so the page reads as one.
const PALETTE = ['#3b82f6', '#10b981', '#f59e0b', '#a855f7', '#ec4899', '#06b6d4', '#84cc16', '#f97316']

// Empty slice colour (used when data is null/short).
const EMPTY_COLOUR = 'rgba(120,120,120,0.18)'

// Window key — kept in sync with the upstream Wails binding's expected
// values. "30d" maps to the last 30 day-buckets in the metering store.
const WINDOW = '30d'

interface Slice {
  label: string
  display: string
  value: number
  colour: string
  pct: number // 0..100
}

interface DonutProps {
  title: string
  slices: Slice[]
  total: number
  empty: string
}

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return String(n)
}

// SVG donut: a single ring composed of arc segments. We use one SVG circle
// per slice and animate its stroke-dasharray. Simpler than a path-based
// renderer and good enough for 5-8 slices, which is the realistic ceiling
// for tool/model counts.
function Donut({ title, slices, total, empty }: DonutProps) {
  const size = 140
  const stroke = 18
  const radius = (size - stroke) / 2
  const circumference = 2 * Math.PI * radius
  const hasData = total > 0 && slices.length > 0

  let offset = 0

  return (
    <div
      data-testid={`donut-${title.toLowerCase().replace(/\s+/g, '-')}`}
      className="border border-border rounded-lg p-4 bg-card"
    >
      <div className="text-sm font-medium mb-3">{title}</div>
      <div className="flex items-center gap-4">
        <svg width={size} height={size} className="shrink-0">
          {/* Background ring */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            stroke={EMPTY_COLOUR}
            strokeWidth={stroke}
          />
          {hasData &&
            slices.map((s, i) => {
              const len = (s.pct / 100) * circumference
              const dash = `${len} ${circumference - len}`
              const rotation = (offset / circumference) * 360 - 90
              offset += len
              return (
                <circle
                  key={i}
                  cx={size / 2}
                  cy={size / 2}
                  r={radius}
                  fill="none"
                  stroke={s.colour}
                  strokeWidth={stroke}
                  strokeDasharray={dash}
                  transform={`rotate(${rotation} ${size / 2} ${size / 2})`}
                  strokeLinecap="butt"
                />
              )
            })}
          <text
            x={size / 2}
            y={size / 2 - 4}
            textAnchor="middle"
            className="fill-foreground"
            fontSize="14"
            fontWeight={600}
          >
            {hasData ? formatTokens(total) : '—'}
          </text>
          <text
            x={size / 2}
            y={size / 2 + 14}
            textAnchor="middle"
            className="fill-muted-foreground"
            fontSize="10"
          >
            tokens
          </text>
        </svg>
        <div className="flex-1 min-w-0">
          {hasData ? (
            <ul className="space-y-1.5">
              {slices.slice(0, 6).map((s, i) => (
                <li key={i} className="flex items-center gap-2 text-xs">
                  <span
                    className="inline-block h-2.5 w-2.5 rounded-sm shrink-0"
                    style={{ background: s.colour }}
                  />
                  <span className="truncate flex-1" title={s.display}>
                    {s.display}
                  </span>
                  <span className="font-mono text-muted-foreground shrink-0">
                    {s.pct.toFixed(1)}%
                  </span>
                </li>
              ))}
              {slices.length > 6 && (
                <li className="text-[11px] text-muted-foreground italic">
                  +{slices.length - 6} more
                </li>
              )}
            </ul>
          ) : (
            <div className="text-xs text-muted-foreground italic">{empty}</div>
          )}
        </div>
      </div>
    </div>
  )
}

// Pure helper — sums tokensIn + tokensOut, sorts desc, computes pct.
// Exported so the test can drive it directly without the Wails round-trip.
export function buildSlices<T extends { totalCalls: number; tokensIn: number; tokensOut: number }>(
  rows: T[] | null | undefined,
  labelOf: (row: T) => { key: string; display: string },
): { slices: Slice[]; total: number } {
  if (!rows || rows.length === 0) return { slices: [], total: 0 }
  const grouped = new Map<string, { display: string; value: number }>()
  for (const row of rows) {
    const value = (row.tokensIn ?? 0) + (row.tokensOut ?? 0)
    if (value <= 0) continue
    const { key, display } = labelOf(row)
    const existing = grouped.get(key)
    if (existing) existing.value += value
    else grouped.set(key, { display, value })
  }
  const arr = Array.from(grouped, ([key, { display, value }]) => ({ key, display, value }))
  arr.sort((a, b) => b.value - a.value)
  const total = arr.reduce((s, x) => s + x.value, 0)
  if (total === 0) return { slices: [], total: 0 }
  const slices: Slice[] = arr.map((x, i) => ({
    label: x.key,
    display: x.display,
    value: x.value,
    colour: PALETTE[i % PALETTE.length],
    pct: (x.value / total) * 100,
  }))
  return { slices, total }
}

export function UsageBreakdown() {
  const { t } = useTranslation()
  const [models, setModels] = useState<metering.ModelSummary[] | null>(null)
  const [apps, setApps] = useState<metering.AppSummary[] | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    Promise.all([
      GetModelSummaries(WINDOW).catch(() => [] as metering.ModelSummary[]),
      GetAppSummaries(WINDOW).catch(() => [] as metering.AppSummary[]),
    ])
      .then(([ms, as]) => {
        if (cancelled) return
        setModels(ms || [])
        setApps(as || [])
      })
      .catch((e) => {
        if (cancelled) return
        setError(String(e))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const modelChart = useMemo(
    () =>
      buildSlices(models, (m) => ({
        key: m.model,
        display: m.model || t('billing.breakdown.unknownModel', 'Unknown'),
      })),
    [models, t],
  )

  const appChart = useMemo(
    () =>
      buildSlices(apps, (a) => ({
        key: a.appId,
        display: TOOL_DISPLAY[a.appId] ?? a.appId,
      })),
    [apps],
  )

  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="border border-border rounded-lg p-6 bg-card flex items-center justify-center h-[180px]">
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        </div>
        <div className="border border-border rounded-lg p-6 bg-card flex items-center justify-center h-[180px]">
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="border border-border rounded-lg p-4 bg-card text-xs text-amber-300">
        {t('billing.breakdown.error', '用量明细暂不可用：{{err}}', { err: error })}
      </div>
    )
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      <Donut
        title={t('billing.breakdown.byModel', 'Tokens by model (30d)')}
        slices={modelChart.slices}
        total={modelChart.total}
        empty={t('billing.breakdown.empty', '过去 30 天暂无调用记录')}
      />
      <Donut
        title={t('billing.breakdown.byTool', 'Tokens by tool (30d)')}
        slices={appChart.slices}
        total={appChart.total}
        empty={t('billing.breakdown.empty', '过去 30 天暂无调用记录')}
      />
    </div>
  )
}
