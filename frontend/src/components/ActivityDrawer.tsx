import { useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  X, Activity as ActivityIcon, AlertTriangle, CheckCircle2, Loader2,
  Trash2,
} from 'lucide-react'
import { cn } from '../lib/utils'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import {
  useActivityStore,
  type ActivityEvent,
  type ActivityFilter,
} from '../stores/activityStore'

const EVENT_NAME = 'activity:event'

// Filter buttons rendered at the top of the drawer. Order matters — "all"
// first so the default appears highlighted on cold open.
const FILTERS: { id: ActivityFilter; labelKey: string; fallback: string }[] = [
  { id: 'all', labelKey: 'activityDrawer.filter.all', fallback: '全部' },
  { id: 'active', labelKey: 'activityDrawer.filter.active', fallback: '进行中' },
  { id: 'mutation', labelKey: 'activityDrawer.filter.mutation', fallback: '配置' },
  { id: 'error', labelKey: 'activityDrawer.filter.error', fallback: '错误' },
  { id: 'auth', labelKey: 'activityDrawer.filter.auth', fallback: '认证' },
  { id: 'system', labelKey: 'activityDrawer.filter.system', fallback: '系统' },
]

// Subscriber hook: lazily mounts the activity:event subscription exactly
// once at app startup, regardless of whether the drawer is currently open.
// Without this the drawer would only collect events while open and the
// "persistent history" promise would break.
export function useActivityIngest() {
  const ingest = useActivityStore((s) => s.ingest)
  useEffect(() => {
    const unsub = EventsOn(EVENT_NAME, (raw: unknown) => {
      const ev = raw as ActivityEvent
      if (!ev?.id) return
      ingest(ev)
    })
    return () => {
      if (typeof unsub === 'function') unsub()
    }
  }, [ingest])
}

export function ActivityDrawer() {
  const { t, i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh') ?? true
  const open = useActivityStore((s) => s.drawerOpen)
  const setOpen = useActivityStore((s) => s.setDrawerOpen)
  const events = useActivityStore((s) => s.events)
  const filter = useActivityStore((s) => s.filter)
  const setFilter = useActivityStore((s) => s.setFilter)
  const clear = useActivityStore((s) => s.clear)
  const markAllSeen = useActivityStore((s) => s.markAllSeen)

  // Mark seen whenever the drawer opens so the sidebar badge clears.
  useEffect(() => {
    if (open) markAllSeen()
  }, [open, markAllSeen])

  // Esc to close.
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [open, setOpen])

  const sorted = useMemo(
    () =>
      [...events].sort(
        (a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime(),
      ),
    [events],
  )

  const filtered = useMemo(() => {
    if (filter === 'all') return sorted
    return sorted.filter((ev) => {
      if (filter === 'active') return ev.phase === 'start' || ev.phase === 'progress'
      if (filter === 'error') return ev.phase === 'error' || ev.tags?.includes('error') === true
      if (filter === 'mutation') return ev.tags?.includes('mutation') === true
      if (filter === 'auth') return ev.tags?.includes('auth') === true
      if (filter === 'system') {
        const tg = ev.tags ?? []
        return !tg.includes('mutation') && !tg.includes('auth') && !tg.includes('error')
      }
      return true
    })
  }, [sorted, filter])

  return (
    <>
      {/* Scrim */}
      <div
        data-testid="activity-drawer-scrim"
        className={cn(
          'fixed inset-0 bg-black/40 z-40 transition-opacity',
          open ? 'opacity-100 pointer-events-auto' : 'opacity-0 pointer-events-none',
        )}
        onClick={() => setOpen(false)}
      />

      {/* Drawer */}
      <aside
        data-testid="activity-drawer"
        aria-hidden={!open}
        className={cn(
          'fixed top-0 right-0 bottom-0 w-[420px] max-w-[95vw] bg-card border-l border-border z-50',
          'shadow-2xl transition-transform flex flex-col',
          open ? 'translate-x-0' : 'translate-x-full',
        )}
      >
        {/* Header */}
        <header className="flex items-center justify-between px-4 py-3 border-b border-border">
          <div className="flex items-center gap-2">
            <ActivityIcon className="h-4 w-4 text-primary" />
            <h2 className="text-sm font-semibold">
              {t('activityDrawer.title', '活动流')}
            </h2>
            <span className="text-[11px] text-muted-foreground tabular-nums">
              {filtered.length}/{events.length}
            </span>
          </div>
          <div className="flex items-center gap-1">
            <button
              onClick={clear}
              className="h-7 w-7 inline-flex items-center justify-center rounded text-muted-foreground hover:bg-muted"
              title={t('activityDrawer.clear', '清空历史')}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
            <button
              onClick={() => setOpen(false)}
              className="h-7 w-7 inline-flex items-center justify-center rounded text-muted-foreground hover:bg-muted"
              title={t('common.close', '关闭')}
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </header>

        {/* Filter bar */}
        <div className="flex items-center gap-1 px-4 py-2 border-b border-border overflow-x-auto">
          {FILTERS.map((f) => (
            <button
              key={f.id}
              onClick={() => setFilter(f.id)}
              className={cn(
                'px-2.5 py-1 rounded text-[11px] font-medium border transition-colors whitespace-nowrap',
                filter === f.id
                  ? 'bg-primary/10 border-primary/40 text-primary'
                  : 'border-border text-muted-foreground hover:bg-muted',
              )}
            >
              {t(f.labelKey, f.fallback)}
            </button>
          ))}
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto">
          {filtered.length === 0 ? (
            <div className="px-4 py-10 text-center text-xs text-muted-foreground">
              {events.length === 0
                ? t('activityDrawer.empty', '暂无活动记录')
                : t('activityDrawer.emptyFiltered', '当前过滤无匹配条目')}
            </div>
          ) : (
            <ul className="divide-y divide-border/40">
              {filtered.map((ev) => (
                <DrawerRow key={ev.id + ev.updatedAt} ev={ev} isZh={isZh} />
              ))}
            </ul>
          )}
        </div>
      </aside>
    </>
  )
}

function DrawerRow({ ev, isZh }: { ev: ActivityEvent; isZh: boolean }) {
  const title = isZh ? ev.titleZh : ev.titleEn
  const detail = isZh ? ev.detailZh : ev.detailEn
  const inProgress = ev.phase === 'start' || ev.phase === 'progress'
  const isError = ev.phase === 'error'
  const isDone = ev.phase === 'done'

  const ts = new Date(ev.updatedAt)
  const tsLabel = isNaN(ts.getTime())
    ? ev.updatedAt
    : ts.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' })

  return (
    <li className="px-4 py-2.5 text-xs">
      <div className="flex items-start gap-2">
        <div className="shrink-0 mt-0.5">
          {inProgress && <Loader2 className="h-3.5 w-3.5 animate-spin text-primary" />}
          {isDone && <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />}
          {isError && <AlertTriangle className="h-3.5 w-3.5 text-red-400" />}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-baseline gap-2">
            <span className="font-medium text-foreground truncate">{title}</span>
            <span className="text-[10px] text-muted-foreground tabular-nums ml-auto shrink-0">
              {tsLabel}
            </span>
          </div>
          {detail && (
            <div className="text-[11px] text-muted-foreground mt-0.5 break-words">{detail}</div>
          )}
          {ev.error && (
            <div className="text-[11px] text-red-400 mt-0.5 break-words">{ev.error}</div>
          )}
          {ev.tags && ev.tags.length > 0 && (
            <div className="mt-1 flex flex-wrap gap-1">
              {ev.tags.map((tag) => (
                <span
                  key={tag}
                  className="px-1.5 py-0.5 rounded text-[9px] uppercase tracking-wide bg-muted text-muted-foreground"
                >
                  {tag}
                </span>
              ))}
            </div>
          )}
        </div>
      </div>
    </li>
  )
}
