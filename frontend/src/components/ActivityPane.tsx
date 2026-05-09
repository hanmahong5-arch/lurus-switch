import { useEffect, useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Loader2, CheckCircle2, AlertTriangle, ChevronUp, ChevronDown,
  Activity as ActivityIcon, X,
} from 'lucide-react'
import { cn } from '../lib/utils'
import { EventsOn } from '../../wailsjs/runtime/runtime'

// Wire-format mirrors internal/activity.Event 1:1 — keep in sync if the
// Go struct grows fields.
interface ActivityEvent {
  id: string
  phase: 'start' | 'progress' | 'done' | 'error'
  titleZh: string
  titleEn: string
  detailZh?: string
  detailEn?: string
  progress?: number
  total?: number
  step?: number
  error?: string
  startedAt: string
  updatedAt: string
}

const EVENT_NAME = 'activity:event'
const RECENT_LIMIT = 12
// Auto-dismiss completed/errored entries after this delay so the pane
// doesn't grow forever. Active ops are never dismissed.
const SETTLED_TTL_MS = 8_000

export function ActivityPane() {
  const { i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh') ?? true
  const [events, setEvents] = useState<Map<string, ActivityEvent>>(new Map())
  const [collapsed, setCollapsed] = useState(true)

  // Subscribe once on mount.
  useEffect(() => {
    const unsub = EventsOn(EVENT_NAME, (raw: unknown) => {
      const ev = raw as ActivityEvent
      if (!ev?.id) return
      setEvents((prev) => {
        const next = new Map(prev)
        next.set(ev.id, ev)
        // Cap map size FIFO style — drop oldest done/error entries first.
        if (next.size > RECENT_LIMIT) {
          for (const [k, v] of next) {
            if (v.phase === 'done' || v.phase === 'error') {
              next.delete(k)
              if (next.size <= RECENT_LIMIT) break
            }
          }
        }
        return next
      })
    })
    return () => { if (typeof unsub === 'function') unsub() }
  }, [])

  // Auto-dismiss settled entries.
  useEffect(() => {
    const t = setInterval(() => {
      setEvents((prev) => {
        const cutoff = Date.now() - SETTLED_TTL_MS
        let changed = false
        const next = new Map(prev)
        for (const [k, v] of next) {
          if (v.phase !== 'done' && v.phase !== 'error') continue
          if (new Date(v.updatedAt).getTime() < cutoff) {
            next.delete(k)
            changed = true
          }
        }
        return changed ? next : prev
      })
    }, 1000)
    return () => clearInterval(t)
  }, [])

  const dismissAll = useCallback(() => setEvents(new Map()), [])

  const list = [...events.values()].sort((a, b) =>
    new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime(),
  )
  const active = list.filter((e) => e.phase === 'start' || e.phase === 'progress')
  const settled = list.filter((e) => e.phase === 'done' || e.phase === 'error')

  if (list.length === 0) return null

  return (
    <div className="fixed bottom-7 right-3 z-40 w-80 max-w-[calc(100vw-1.5rem)]">
      <div className="rounded-lg border border-border bg-card/95 backdrop-blur-sm shadow-lg overflow-hidden">
        {/* Header */}
        <button
          onClick={() => setCollapsed((c) => !c)}
          className="w-full flex items-center justify-between px-3 py-2 hover:bg-muted/50 transition-colors"
          title={isZh ? '点击折叠/展开' : 'Click to collapse/expand'}
        >
          <div className="flex items-center gap-2 min-w-0">
            {active.length > 0
              ? <Loader2 className="h-3.5 w-3.5 animate-spin text-primary" />
              : <ActivityIcon className="h-3.5 w-3.5 text-muted-foreground" />}
            <span className="text-xs font-medium truncate">
              {active.length > 0
                ? (isZh ? `进行中 ${active.length} 项` : `${active.length} active`)
                : (isZh ? '最近活动' : 'Recent activity')}
            </span>
            {settled.length > 0 && active.length === 0 && (
              <span className="text-[10px] text-muted-foreground tabular-nums">{settled.length}</span>
            )}
          </div>
          <div className="flex items-center gap-1">
            {settled.length > 0 && active.length === 0 && (
              <span
                onClick={(e) => { e.stopPropagation(); dismissAll() }}
                className="h-5 w-5 inline-flex items-center justify-center rounded hover:bg-muted text-muted-foreground"
                title={isZh ? '清空' : 'Clear'}
                role="button"
              >
                <X className="h-3 w-3" />
              </span>
            )}
            {collapsed
              ? <ChevronUp className="h-3.5 w-3.5 text-muted-foreground" />
              : <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />}
          </div>
        </button>

        {/* Body */}
        {!collapsed && (
          <div className="max-h-72 overflow-y-auto border-t border-border">
            {list.map((ev) => <Row key={ev.id} ev={ev} isZh={isZh} />)}
          </div>
        )}

        {/* Always-visible mini summary when collapsed */}
        {collapsed && active.length > 0 && (
          <div className="px-3 py-1.5 border-t border-border text-[11px] text-muted-foreground truncate">
            {active[0] && (isZh ? active[0].titleZh : active[0].titleEn)}
            {active[0]?.progress != null && active[0].progress > 0 && (
              <span className="ml-1 tabular-nums opacity-70">{active[0].progress}%</span>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function Row({ ev, isZh }: { ev: ActivityEvent; isZh: boolean }) {
  const title = isZh ? ev.titleZh : ev.titleEn
  const detail = isZh ? ev.detailZh : ev.detailEn
  const inProgress = ev.phase === 'start' || ev.phase === 'progress'
  const isError = ev.phase === 'error'
  const isDone = ev.phase === 'done'

  return (
    <div className="px-3 py-2 border-b border-border/40 last:border-0 text-xs">
      <div className="flex items-start gap-2">
        <div className="shrink-0 mt-0.5">
          {inProgress && <Loader2 className="h-3.5 w-3.5 animate-spin text-primary" />}
          {isDone && <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />}
          {isError && <AlertTriangle className="h-3.5 w-3.5 text-red-400" />}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1.5 flex-wrap">
            <span className="font-medium text-foreground truncate">{title}</span>
            {ev.total && ev.total > 1 && (
              <span className="text-[10px] text-muted-foreground tabular-nums">
                {ev.step}/{ev.total}
              </span>
            )}
          </div>
          {detail && <div className="text-[11px] text-muted-foreground mt-0.5 truncate">{detail}</div>}
          {ev.error && <div className="text-[11px] text-red-400 mt-0.5 break-words">{ev.error}</div>}
          {inProgress && ev.progress != null && ev.progress > 0 && (
            <div className="mt-1 h-1 bg-muted/50 rounded-full overflow-hidden">
              <div
                className="h-full bg-primary transition-all"
                style={{ width: `${Math.min(100, ev.progress)}%` }}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
