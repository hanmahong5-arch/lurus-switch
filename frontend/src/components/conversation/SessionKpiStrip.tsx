import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { MessageSquare, Hash, Clock, Cpu } from 'lucide-react'
import type { conversation } from '../../../wailsjs/go/models'
import { KpiCard } from '../ui'
import { parseSaneDate, formatRelative } from '../../lib/conversationUtils'
import { roleKindOf, type RoleKind } from './MessageCard'
import { cn } from '../../lib/utils'

interface Props {
  events: conversation.Event[]
}

const ROLE_ORDER: RoleKind[] = ['user', 'assistant', 'tool_use', 'tool_result', 'system', 'meta']

const ROLE_BAR_COLOR: Record<RoleKind, string> = {
  user: 'bg-blue-500',
  assistant: 'bg-amber-500',
  tool_use: 'bg-purple-500',
  tool_result: 'bg-emerald-500',
  system: 'bg-slate-500',
  meta: 'bg-zinc-500',
}

export function SessionKpiStrip({ events }: Props) {
  const { t, i18n } = useTranslation()
  const isZh = (i18n.language || '').startsWith('zh')

  const stats = useMemo(() => {
    const roles: Record<RoleKind, number> = {
      user: 0, assistant: 0, tool_use: 0, tool_result: 0, system: 0, meta: 0,
    }
    let inT = 0, outT = 0, cacheT = 0
    const modelCounts: Record<string, number> = {}
    let first: Date | null = null
    let last: Date | null = null
    for (const ev of events) {
      roles[roleKindOf(ev.type)]++
      inT += ev.inputTokens || 0
      outT += ev.outputTokens || 0
      cacheT += ev.cacheReadTokens || 0
      if (ev.model) modelCounts[ev.model] = (modelCounts[ev.model] || 0) + 1
      const d = parseSaneDate(ev.timestamp)
      if (d) {
        if (!first || d < first) first = d
        if (!last || d > last) last = d
      }
    }
    const sortedModels = Object.entries(modelCounts).sort((a, b) => b[1] - a[1])
    return {
      roles, inT, outT, cacheT, first, last,
      totalTokens: inT + outT + cacheT,
      models: sortedModels,
    }
  }, [events])

  const total = events.length
  const roleEntries = ROLE_ORDER
    .map((r) => ({ role: r, count: stats.roles[r] }))
    .filter((e) => e.count > 0)

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-2 px-4 py-3 border-b border-border bg-card-recessed/40">
      <KpiCard
        label={t('conversations.kpi.messages', 'Messages')}
        value={total}
        icon={MessageSquare}
      />
      <div>
        <KpiCard
          label={t('conversations.kpi.tokens', 'Tokens')}
          value={formatNumber(stats.totalTokens)}
          icon={Hash}
        />
        <div className="mt-1 text-[10px] font-mono text-muted-foreground tabular-nums">
          in {formatNumber(stats.inT)} · out {formatNumber(stats.outT)}
          {stats.cacheT > 0 && <> · cache {formatNumber(stats.cacheT)}</>}
        </div>
      </div>
      <div>
        <KpiCard
          label={t('conversations.kpi.timeSpan', 'Time span')}
          value={timeSpan(stats.first, stats.last, isZh)}
          icon={Clock}
        />
        <div className="mt-1 text-[10px] font-mono text-muted-foreground tabular-nums">
          {stats.first && stats.last
            ? `${stats.first.toLocaleTimeString()} → ${stats.last.toLocaleTimeString()}`
            : '—'}
        </div>
      </div>
      <KpiCard
        label={t('conversations.kpi.models', 'Models')}
        value={modelLabel(stats.models, isZh)}
        icon={Cpu}
      />
      {roleEntries.length > 0 && (
        <div className="col-span-2 md:col-span-4 -mt-1">
          <div className="flex h-1.5 rounded-full overflow-hidden bg-muted/30" title={roleTooltip(roleEntries, total, t)}>
            {roleEntries.map((e) => (
              <div
                key={e.role}
                className={cn(ROLE_BAR_COLOR[e.role])}
                style={{ width: `${(e.count / total) * 100}%` }}
              />
            ))}
          </div>
          <div className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-[10px] font-mono text-muted-foreground tabular-nums">
            {roleEntries.map((e) => (
              <span key={e.role} className="inline-flex items-center gap-1">
                <span className={cn('h-1.5 w-1.5 rounded-full', ROLE_BAR_COLOR[e.role])} />
                {t(`conversations.role.${e.role}`, e.role)} · {e.count}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function formatNumber(n: number): string {
  if (n < 1000) return String(n)
  if (n < 1_000_000) return `${(n / 1000).toFixed(n < 10_000 ? 1 : 0)}k`
  return `${(n / 1_000_000).toFixed(n < 10_000_000 ? 1 : 0)}M`
}

function timeSpan(first: Date | null, last: Date | null, isZh: boolean): string {
  if (!first || !last) return '—'
  const ms = last.getTime() - first.getTime()
  if (ms < 1000) return isZh ? '< 1 秒' : '< 1s'
  const sec = Math.round(ms / 1000)
  if (sec < 60) return `${sec}s`
  const min = Math.round(sec / 60)
  if (min < 60) return `${min}m`
  const hr = (min / 60).toFixed(min < 600 ? 1 : 0)
  if (Number(hr) < 24) return `${hr}h`
  return formatRelative(first, last, isZh)
}

function modelLabel(models: Array<[string, number]>, _isZh: boolean): string {
  if (models.length === 0) return '—'
  const primary = models[0][0]
  if (models.length === 1) return primary
  const short = primary.length > 18 ? `${primary.slice(0, 18)}…` : primary
  return `${short} +${models.length - 1}`
}

function roleTooltip(
  entries: Array<{ role: RoleKind; count: number }>,
  total: number,
  t: TFunction,
): string {
  return entries
    .map((e) => `${t(`conversations.role.${e.role}`, e.role)}: ${e.count} (${Math.round((e.count / total) * 100)}%)`)
    .join(' · ')
}
