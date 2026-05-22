import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Activity, Terminal, FileEdit, Clock, Cpu,
  DollarSign, Hash, Eye, EyeOff,
  Wrench, AlertCircle, AlertTriangle,
} from 'lucide-react'
import { EventsOn } from '../../wailsjs/runtime'
import { useLiveSessionStore } from '../stores/liveSessionStore'
import type { LiveSession, LiveSessionStatus, EventSummary } from '../lib/liveSessionApi'
import { cn } from '../lib/utils'
import { useTickingNow } from '../lib/useTickingNow'
import { iconForSummaryKind } from '../components/livesession/eventIcons'
import { SessionDetailDrawer } from '../components/livesession/SessionDetailDrawer'
import { Button, Card } from '../components/ui'

// Live-session inspector — the "what is Claude doing right now" page.
//
// The card layout deliberately surfaces the highest-anxiety questions
// first: status badge ("是不是卡了") and pending tool preview ("它正在
//干什么"). Cost and counts come second. Recent activity, file touches,
// and bash history fall below the fold by default — present but not
// dominant. This mirrors how a power user reads top-tier of an Activity
// Monitor app: glance at health, then drill down.

const STATUS_META: Record<LiveSessionStatus, { label: string; color: string; ring: string; dot: string }> = {
  running: {
    label: '运行中',
    color: 'text-emerald-400',
    ring: 'ring-emerald-400/30',
    dot: 'bg-emerald-400',
  },
  tool_call: {
    label: '工具调用中',
    color: 'text-amber-400',
    ring: 'ring-amber-400/40',
    dot: 'bg-amber-400 animate-pulse',
  },
  awaiting_user: {
    label: '等待用户',
    color: 'text-blue-400',
    ring: 'ring-blue-400/30',
    dot: 'bg-blue-400',
  },
  idle: {
    label: '空闲',
    color: 'text-muted-foreground',
    ring: 'ring-border',
    dot: 'bg-muted-foreground/40',
  },
}

export function LiveSessionsPage() {
  const { t } = useTranslation()
  const sessions = useLiveSessionStore((s) => s.sessions)
  const showIdle = useLiveSessionStore((s) => s.showIdle)
  const setShowIdle = useLiveSessionStore((s) => s.setShowIdle)
  const refresh = useLiveSessionStore((s) => s.refresh)
  const error = useLiveSessionStore((s) => s.error)
  // One page-level 1Hz ticker drives every pending-tool stopwatch below.
  // Per-card timers would scale linearly with active sessions; this stays
  // O(1) regardless of how many sessions are visible.
  const now = useTickingNow(1000)

  // Selected session for the slide-in transcript drawer. Stored by
  // `transcriptPath` (the stable identity across re-renders — the store
  // replaces the LiveSession object on every poll cycle) and re-resolved
  // out of `sessions` on render so the drawer always sees the latest
  // aggregates without re-mounting.
  const [openPath, setOpenPath] = useState<string | null>(null)
  const openSession = useMemo(
    () => (openPath ? sessions.find((s) => s.transcriptPath === openPath) ?? null : null),
    [openPath, sessions],
  )

  // Initial fetch + subscribe to push events. EventsOn returns an
  // unsubscribe function but it isn't always available on older Wails
  // builds, so we guard the cleanup.
  useEffect(() => {
    void refresh()
    const off = EventsOn('livesession:update', () => {
      void refresh()
    })
    return () => {
      if (typeof off === 'function') off()
    }
  }, [refresh])

  // Roll-up: total spend across visible sessions + tool-call activity.
  const rollup = useMemo(() => {
    let usd = 0
    let toolCalls = 0
    let tokens = 0
    let active = 0
    for (const s of sessions) {
      usd += s.estimatedUsd
      toolCalls += s.toolCallCount
      // Include cache fields so the headline token figure reflects every
      // billable stream — same definition that drives the cost estimate.
      tokens += s.inputTokens + s.outputTokens +
        (s.cacheCreateTokens || 0) + (s.cacheReadTokens || 0)
      if (s.status === 'running' || s.status === 'tool_call') active++
    }
    return { usd, toolCalls, tokens, active }
  }, [sessions])

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* Header with summary metrics + toggle */}
      <div className="px-6 pt-5 pb-3 border-b border-border">
        <div className="flex items-center justify-between gap-4 mb-3">
          <div>
            <h2 className="text-lg font-semibold flex items-center gap-2">
              <Activity className="h-5 w-5 text-primary" />
              {t('live.title', '实时观察 · Live Inspector')}
            </h2>
            <p className="text-xs text-muted-foreground mt-0.5 font-mono">
              {t('live.subtitle', '看 CLI 工具此刻在干什么 — 每 2 秒推送')}
            </p>
          </div>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => setShowIdle(!showIdle)}
            title={t('live.toggleIdle', '显示空闲会话')}
            icon={showIdle ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
          >
            {showIdle ? t('live.hideIdle', '隐藏空闲') : t('live.showIdle', '显示空闲')}
          </Button>
        </div>

        {/* Roll-up chips */}
        <div className="flex flex-wrap items-center gap-2 text-xs">
          <Chip label="活动会话" value={String(rollup.active)} icon={Activity} />
          <Chip
            label="本批 估算 $"
            value={'$' + rollup.usd.toFixed(2)}
            icon={DollarSign}
            title={
              '估算口径: input + output + cache_create×1.25 + cache_read×0.10\n' +
              '按每条消息当时的模型分段计价 · 真实账单以 Anthropic 控制台为准'
            }
          />
          <Chip label="工具调用" value={String(rollup.toolCalls)} icon={Wrench} />
          <Chip label="总 token (含缓存)" value={fmtTok(rollup.tokens)} icon={Hash} />
        </div>

        {error && (
          <div className="mt-2 text-xs text-red-400 flex items-center gap-1.5 font-mono">
            <AlertCircle className="h-3.5 w-3.5" />
            {error}
          </div>
        )}
      </div>

      {/* Session list */}
      <div className="flex-1 overflow-y-auto px-6 py-4">
        {sessions.length === 0 ? (
          <EmptyState showIdle={showIdle} />
        ) : (
          <div className="grid gap-3">
            {sessions.map((s) => (
              <SessionCard
                key={s.sessionId + ':' + s.transcriptPath}
                session={s}
                now={now}
                onOpen={() => setOpenPath(s.transcriptPath)}
              />
            ))}
          </div>
        )}
      </div>

      {/* Transcript drawer — rendered at page root so it overlays the list */}
      <SessionDetailDrawer session={openSession} onClose={() => setOpenPath(null)} />
    </div>
  )
}

function Chip({ label, value, icon: Icon, title }: {
  label: string
  value: string
  icon: React.ComponentType<{ className?: string }>
  title?: string
}) {
  return (
    <div
      className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-border bg-card-recessed transition-colors hover:border-rule-strong"
      title={title}
    >
      <Icon className="h-3.5 w-3.5 text-muted-foreground" />
      <span className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">{label}</span>
      <span className="font-mono font-medium text-foreground tabular-nums">{value}</span>
    </div>
  )
}

function EmptyState({ showIdle }: { showIdle: boolean }) {
  return (
    <div className="flex flex-col items-center justify-center text-center py-16 text-sm text-muted-foreground">
      <Activity className="h-10 w-10 mb-3 opacity-40" />
      <p className="font-medium text-foreground/80 mb-1">
        {showIdle ? '没有任何会话记录' : '当前无活动会话'}
      </p>
      <p className="text-xs max-w-md">
        在终端启动 <code className="px-1 py-0.5 rounded bg-muted text-[11px]">claude</code> /
        <code className="px-1 py-0.5 rounded bg-muted text-[11px] mx-1">codex</code> /
        <code className="px-1 py-0.5 rounded bg-muted text-[11px]">gemini</code>，
        Switch 会在 2 秒内识别到，自动出现在这里。
      </p>
    </div>
  )
}

function SessionCard({ session, now, onOpen }: { session: LiveSession; now: number; onOpen: () => void }) {
  const { t } = useTranslation()
  const meta = STATUS_META[session.status] ?? STATUS_META.idle
  const lastActivityRel = useMemo(() => relTime(session.lastActivity), [session.lastActivity])
  const startedRel = useMemo(() => relTime(session.startedAt), [session.startedAt])
  const filesTouched = session.filesTouched ?? []
  const bashes = session.bashCommands ?? []

  // Pending-tool stopwatch. We derive seconds from the live `now` so the
  // label re-renders every tick. `Math.max(0, …)` guards against clock
  // skew between backend startedAt and the renderer clock.
  const pending = session.pendingTool
  const pendingStartMs = pending ? new Date(pending.startedAt).getTime() : 0
  const elapsedSec = pending && Number.isFinite(pendingStartMs) && pendingStartMs > 0
    ? Math.max(0, Math.floor((now - pendingStartMs) / 1000))
    : 0
  const escalation: 'normal' | 'hot' | 'stuck' =
    elapsedSec > 120 ? 'stuck' : elapsedSec > 30 ? 'hot' : 'normal'

  // Card is clickable — opens the transcript drawer. We render as a
  // <div role="button"> rather than an actual <button> because the card
  // contains nested interactive bits (filenames with tooltips, bash list,
  // etc.) and nesting <button>s is invalid HTML. Keyboard support comes
  // from explicit Enter/Space handlers.
  const handleKey = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      onOpen()
    }
  }

  const isActive = session.status === 'running' || session.status === 'tool_call'

  return (
    <Card
      variant="elevated"
      glow={isActive}
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={handleKey}
      className={cn(
        'p-4 ring-1 cursor-pointer',
        'hover:bg-card-elevated/80 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary',
        !isActive && meta.ring,
      )}
    >
      {/* Top row: project + status + cost */}
      <div className="flex items-start gap-3 mb-3">
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <span className={cn('h-2.5 w-2.5 rounded-full flex-shrink-0', meta.dot)} />
          <div className="min-w-0 flex-1">
            <div className="flex items-baseline gap-2">
              <h3 className="text-sm font-semibold truncate">{session.projectName}</h3>
              <span className="text-[10px] uppercase tracking-wider text-muted-foreground/70 flex-shrink-0">
                {session.tool}
              </span>
            </div>
            <p className="text-[11px] text-muted-foreground truncate font-mono mt-0.5">{session.cwd}</p>
          </div>
        </div>
        <div className="flex flex-col items-end gap-0.5 flex-shrink-0">
          <span className={cn('text-xs font-medium', meta.color)}>{meta.label}</span>
          <span className="text-[10px] text-muted-foreground">{lastActivityRel}</span>
        </div>
      </div>

      {/* Pending tool — the headline answer to "what is it doing right now".
          Visual escalates at 30s (hot) / 120s (likely stuck) so a glance is
          enough to spot a hung tool call without opening the terminal. */}
      {pending && (
        <div className={cn(
          'mb-3 rounded-md border px-3 py-2 transition-colors',
          escalation === 'stuck'
            ? 'border-red-500/50 bg-red-500/10'
            : escalation === 'hot'
              ? 'border-orange-500/40 bg-orange-500/10'
              : 'border-amber-500/30 bg-amber-500/5',
        )}>
          <div className="flex items-center gap-2 text-xs mb-1">
            <Wrench className={cn(
              'h-3.5 w-3.5',
              escalation === 'stuck' ? 'text-red-500'
                : escalation === 'hot' ? 'text-orange-500'
                : 'text-amber-500',
            )} />
            <span className={cn(
              'font-medium',
              escalation === 'stuck' ? 'text-red-600 dark:text-red-400'
                : escalation === 'hot' ? 'text-orange-600 dark:text-orange-400'
                : 'text-amber-600 dark:text-amber-400',
            )}>
              {t('live.pendingTool', '正在调用')} {pending.name}
            </span>
            <span className={cn(
              'ml-auto tabular-nums',
              escalation === 'stuck'
                ? 'text-[11px] font-semibold text-red-600 dark:text-red-400'
                : escalation === 'hot'
                  ? 'text-[11px] font-semibold text-orange-600 dark:text-orange-400'
                  : 'text-[10px] text-muted-foreground',
            )}>
              {fmtElapsed(elapsedSec, t)}
            </span>
            {escalation === 'stuck' && (
              <span className="flex items-center gap-1 text-[10px] font-medium text-red-600 dark:text-red-400">
                <AlertTriangle className="h-3 w-3" />
                {t('live.maybeStuck', '可能卡住 — 检查终端')}
              </span>
            )}
          </div>
          {pending.preview && (
            <div className="text-[11px] font-mono text-foreground/80 break-all line-clamp-2">
              {pending.preview}
            </div>
          )}
        </div>
      )}

      {/* Stats row. Tokens displayed as billed total (input + output +
          cache_create + cache_read) — the four streams compose the cost
          equation, so a glance at "total tokens that touched the bill"
          is more meaningful than just input+output. */}
      <div className="grid grid-cols-4 gap-2 text-[11px] mb-3">
        <Stat
          icon={DollarSign}
          label={`本会话 估算${(session.modelsSeen?.length ?? 0) > 1 ? ' · 混合模型' : ''}`}
          value={'$' + session.estimatedUsd.toFixed(2)}
          title={costTooltip(session)}
        />
        <Stat
          icon={Hash}
          label="Token"
          value={fmtTok(
            session.inputTokens + session.outputTokens +
            (session.cacheCreateTokens || 0) + (session.cacheReadTokens || 0)
          )}
          title={tokenTooltip(session)}
        />
        <Stat icon={Wrench} label="工具" value={String(session.toolCallCount)} />
        <Stat icon={Cpu} label="模型" value={shortModel(session.model)} />
      </div>

      {/* Two-column lower fold: recent events | files + bash */}
      <div className="grid grid-cols-2 gap-4 text-[11px]">
        <div>
          <SectionLabel icon={Clock}>最近事件</SectionLabel>
          {session.recent.length === 0 ? (
            <p className="text-muted-foreground/60 italic">尚无活动</p>
          ) : (
            <ul className="space-y-1">
              {[...session.recent].reverse().slice(0, 6).map((ev, i) => (
                <EventRow key={i} event={ev} />
              ))}
            </ul>
          )}
        </div>
        <div className="space-y-3">
          {filesTouched.length > 0 && (
            <div>
              <SectionLabel icon={FileEdit}>文件 ({filesTouched.length})</SectionLabel>
              <ul className="space-y-0.5">
                {filesTouched.slice(0, 5).map((f) => (
                  <li key={f.path} className="flex items-center gap-1.5">
                    <span className={cn(
                      'inline-block w-1.5 h-1.5 rounded-full',
                      f.kind === 'write' ? 'bg-red-500' : f.kind === 'edit' ? 'bg-amber-500' : 'bg-blue-400',
                    )} />
                    <span className="font-mono truncate flex-1 text-foreground/80" title={f.path}>{shortPath(f.path)}</span>
                    <span className="text-muted-foreground tabular-nums">×{f.count}</span>
                  </li>
                ))}
                {filesTouched.length > 5 && (
                  <li className="text-muted-foreground/60">+ {filesTouched.length - 5} 个文件…</li>
                )}
              </ul>
            </div>
          )}
          {bashes.length > 0 && (
            <div>
              <SectionLabel icon={Terminal}>最近 Bash</SectionLabel>
              <ul className="space-y-0.5">
                {bashes.slice(-4).reverse().map((c, i) => (
                  <li key={i} className="font-mono text-foreground/70 truncate" title={c}>
                    {c}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>

      {/* Footer: started + transcript */}
      <div className="flex items-center justify-between text-[10px] text-muted-foreground mt-3 pt-2 border-t border-border/50">
        <span className="font-mono">开始于 {startedRel}</span>
        <span className="font-mono truncate ml-2 max-w-[60%] tabular-nums" title={session.transcriptPath}>
          {shortPath(session.transcriptPath)}
        </span>
      </div>
    </Card>
  )
}

function Stat({ icon: Icon, label, value, title }: {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: string
  // `title` becomes the hover tooltip on the value — used to expose the
  // cost / token breakdown without crowding the visible cell.
  title?: string
}) {
  return (
    <Card variant="recessed" className="px-2 py-1.5" title={title}>
      <div className="flex items-center gap-1 text-muted-foreground mb-0.5">
        <Icon className="h-3 w-3" />
        <span className="font-mono text-[10px] uppercase tracking-[0.12em] truncate" title={label}>{label}</span>
      </div>
      <p className="font-mono font-medium text-foreground tabular-nums truncate" title={title ?? value}>{value}</p>
    </Card>
  )
}

// costTooltip — hover detail on the $-stat cell. Spell out the four
// pricing streams + the per-model note so an over- or under-estimate is
// debuggable from inside the GUI without re-reading the source.
function costTooltip(s: LiveSession): string {
  const parts: string[] = []
  parts.push(`$${s.estimatedUsd.toFixed(4)} 估算`)
  parts.push('')
  parts.push(`input ${fmtTok(s.inputTokens)} · output ${fmtTok(s.outputTokens)}`)
  parts.push(`cache_create ${fmtTok(s.cacheCreateTokens || 0)} · cache_read ${fmtTok(s.cacheReadTokens || 0)}`)
  if ((s.modelsSeen?.length ?? 0) > 1) {
    parts.push('')
    parts.push(`混合模型按各自定价分段累加: ${s.modelsSeen?.join(', ')}`)
  } else if (s.model) {
    parts.push('')
    parts.push(`按 ${s.model} 定价`)
  }
  parts.push('')
  parts.push('注: input/output 仅含未缓存部分;')
  parts.push('cache_create = 输入入缓存(1.25×全价), cache_read = 缓存命中(0.10×全价)')
  return parts.join('\n')
}

function tokenTooltip(s: LiveSession): string {
  const lines = [
    `input  ${fmtTok(s.inputTokens)}`,
    `output ${fmtTok(s.outputTokens)}`,
    `cache_create ${fmtTok(s.cacheCreateTokens || 0)}`,
    `cache_read   ${fmtTok(s.cacheReadTokens || 0)}`,
  ]
  return lines.join('\n')
}

function SectionLabel({ icon: Icon, children }: {
  icon: React.ComponentType<{ className?: string }>
  children: React.ReactNode
}) {
  return (
    <p className="flex items-center gap-1 font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground mb-1.5">
      <Icon className="h-3 w-3" />
      {children}
    </p>
  )
}

function EventRow({ event }: { event: EventSummary }) {
  const Icon = iconForSummaryKind(event.kind)
  return (
    <li className="flex items-start gap-1.5">
      <Icon className="h-3 w-3 mt-0.5 flex-shrink-0 text-muted-foreground" />
      <div className="min-w-0 flex-1">
        <p className="truncate text-foreground/85">{event.label}</p>
        {event.details && (
          <p className="truncate text-[10px] text-muted-foreground font-mono" title={event.details}>
            {event.details}
          </p>
        )}
      </div>
    </li>
  )
}

// --- Formatters ---

function fmtTok(n: number): string {
  if (!n) return '0'
  if (n < 1000) return String(n)
  if (n < 1_000_000) return (n / 1000).toFixed(1) + 'k'
  return (n / 1_000_000).toFixed(2) + 'M'
}

function shortModel(model?: string): string {
  if (!model) return '—'
  // claude-sonnet-4-6-20260301 → sonnet-4-6
  const m = model.toLowerCase()
  const stripped = m.replace(/^claude-/, '').replace(/-2\d{7}.*$/, '')
  return stripped || model
}

function shortPath(p: string): string {
  if (!p) return ''
  // Collapse home dir, take last 2 segments for visual brevity.
  const parts = p.split(/[/\\]/).filter(Boolean)
  if (parts.length <= 2) return p
  return '…/' + parts.slice(-2).join('/')
}

// fmtElapsed — stopwatch label for an in-flight tool call. Splits into
// "已运行 Ns" under a minute and "已运行 Nm Ms" once we cross 60s so the
// minute count stays prominent. The translation hook is passed in rather
// than imported here to keep this a plain function.
function fmtElapsed(sec: number, t: (key: string, fallback: string) => string): string {
  const prefix = t('live.elapsed', '已运行')
  if (sec < 60) return `${prefix} ${sec}s`
  const m = Math.floor(sec / 60)
  const s = sec % 60
  const minLabel = t('live.minutes', '分')
  const secLabel = t('live.seconds', '秒')
  return `${prefix} ${m} ${minLabel} ${s} ${secLabel}`
}

function relTime(iso: string): string {
  if (!iso) return ''
  const t = new Date(iso).getTime()
  if (!Number.isFinite(t) || t === 0) return ''
  const diff = Date.now() - t
  if (diff < 0) return '刚刚'
  const s = Math.floor(diff / 1000)
  if (s < 5) return '刚刚'
  if (s < 60) return `${s} 秒前`
  if (s < 3600) return `${Math.floor(s / 60)} 分钟前`
  if (s < 86400) return `${Math.floor(s / 3600)} 小时前`
  return `${Math.floor(s / 86400)} 天前`
}
