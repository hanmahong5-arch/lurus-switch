import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Shield, RefreshCw, Undo2, Filter as FilterIcon, ChevronDown, ChevronRight,
  CheckCircle2, CircleSlash, AlertTriangle,
} from 'lucide-react'
import { useAuditStore, type AuditEntry } from '../../stores/auditStore'
import { cn } from '../../lib/utils'

// Colored chip per outcome — auditors should see at a glance which
// rows succeeded vs which got denied by capability gate vs which the
// upstream errored on.
function OutcomeBadge({ outcome }: { outcome: string }) {
  const cfg: Record<string, { Icon: typeof CheckCircle2; cls: string; label: string }> = {
    ok:     { Icon: CheckCircle2,   cls: 'text-emerald-500 bg-emerald-500/10 border-emerald-500/30', label: 'ok' },
    denied: { Icon: CircleSlash,   cls: 'text-amber-500 bg-amber-500/10 border-amber-500/30',       label: 'denied' },
    error:  { Icon: AlertTriangle, cls: 'text-red-500 bg-red-500/10 border-red-500/30',             label: 'error' },
  }
  const c = cfg[outcome] ?? cfg.error
  const Icon = c.Icon
  return (
    <span className={cn('inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] border', c.cls)}>
      <Icon className="h-3 w-3" />
      {c.label}
    </span>
  )
}

function EntryRow({ entry }: { entry: AuditEntry }) {
  const { t } = useTranslation()
  const undo = useAuditStore((s) => s.undo)
  const undoingId = useAuditStore((s) => s.undoingId)
  const capabilities = useAuditStore((s) => s.capabilities)
  const [expanded, setExpanded] = useState(false)

  const isUndone = !!entry.undoneAt
  const canUndo = entry.reversible && entry.outcome === 'ok' && !isUndone
  const isUndoing = undoingId === entry.id

  return (
    <div className="border-b border-border/40 last:border-0">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-2 px-3 py-2 hover:bg-muted/30 text-left"
      >
        {expanded
          ? <ChevronDown className="h-3 w-3 text-muted-foreground shrink-0" />
          : <ChevronRight className="h-3 w-3 text-muted-foreground shrink-0" />}
        <span className="text-[10px] text-muted-foreground tabular-nums w-24 shrink-0 font-mono">
          {new Date(entry.timestamp).toLocaleTimeString()}
        </span>
        <span className="text-xs font-mono shrink-0 max-w-[12rem] truncate">{entry.operation}</span>
        <OutcomeBadge outcome={entry.outcome} />
        <span className="text-xs text-muted-foreground truncate flex-1">{entry.principal}</span>
        {entry.target && (
          <span className="text-[10px] text-muted-foreground tabular-nums shrink-0">→ {entry.target}</span>
        )}
        {isUndone && (
          <span className="text-[10px] text-muted-foreground/60 shrink-0">
            {t('audit.undoneBy', '已撤销 by {{by}}', { by: entry.undoneBy ?? '—' })}
          </span>
        )}
        {canUndo && (
          <button
            onClick={(e) => { e.stopPropagation(); undo(entry.id) }}
            disabled={isUndoing}
            className="flex items-center gap-1 px-2 py-0.5 text-[10px] rounded border border-border hover:bg-muted disabled:opacity-50 shrink-0"
            title={t('audit.undo', '撤销')}
          >
            <Undo2 className="h-3 w-3" />
            {isUndoing ? '…' : t('audit.undo', '撤销')}
          </button>
        )}
      </button>
      {expanded && (
        <div className="px-9 pb-3 text-[11px] space-y-1">
          {entry.error && (
            <div className="text-red-500 font-mono break-all">{entry.error}</div>
          )}
          {entry.capsHeld?.length > 0 && (
            <div className="text-muted-foreground">
              <span className="opacity-60">caps:</span>{' '}
              {entry.capsHeld.map((c) => (
                <span key={c} className="inline-block px-1 mr-1 my-0.5 bg-muted rounded font-mono" title={capabilities[c] ?? c}>
                  {c}
                </span>
              ))}
            </div>
          )}
          {entry.before !== undefined && entry.before !== null && (
            <details>
              <summary className="cursor-pointer text-muted-foreground">before</summary>
              <pre className="bg-muted/30 rounded p-2 mt-1 overflow-x-auto font-mono text-[10px]">
                {JSON.stringify(entry.before, null, 2)}
              </pre>
            </details>
          )}
          {entry.after !== undefined && entry.after !== null && (
            <details>
              <summary className="cursor-pointer text-muted-foreground">after / payload</summary>
              <pre className="bg-muted/30 rounded p-2 mt-1 overflow-x-auto font-mono text-[10px]">
                {JSON.stringify(entry.after, null, 2)}
              </pre>
            </details>
          )}
        </div>
      )}
    </div>
  )
}

export function AuditLogPanel() {
  const { t } = useTranslation()
  const {
    entries, stats, principal, filter,
    loading, error, load, loadCapabilities, setFilter, resetFilter,
  } = useAuditStore()

  useEffect(() => {
    loadCapabilities()
    load()
  }, [load, loadCapabilities])

  const topPrincipals = useMemo(() => {
    if (!stats) return []
    return Object.entries(stats.byPrincipal)
      .sort(([, a], [, b]) => b - a)
      .slice(0, 4)
  }, [stats])

  const topOps = useMemo(() => {
    if (!stats) return []
    return Object.entries(stats.byOperation)
      .sort(([, a], [, b]) => b - a)
      .slice(0, 6)
  }, [stats])

  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-border bg-card p-4">
        <div className="flex items-center justify-between gap-2 mb-3">
          <h3 className="text-sm font-semibold flex items-center gap-2">
            <Shield className="h-4 w-4 text-amber-400" />
            {t('audit.title', '审计日志')}
          </h3>
          <div className="flex items-center gap-2">
            <span className="text-[10px] text-muted-foreground">
              {t('audit.youAre', '当前身份')}: <span className="font-mono">{principal || '—'}</span>
            </span>
            <button
              onClick={load}
              disabled={loading}
              className="p-1 rounded hover:bg-muted disabled:opacity-50"
              title={t('ui.refresh', 'Refresh')}
            >
              <RefreshCw className={cn('h-3.5 w-3.5', loading && 'animate-spin')} />
            </button>
          </div>
        </div>

        {/* Stats strip */}
        {stats && (
          <div className="grid grid-cols-4 gap-2 mb-3">
            <Tile label={t('audit.stats.total', '总条数')} value={stats.total} />
            <Tile label={t('audit.stats.ok', '成功')} value={stats.ok} cls="text-emerald-500" />
            <Tile label={t('audit.stats.denied', '已拒')} value={stats.denied} cls="text-amber-500" />
            <Tile label={t('audit.stats.error', '错误')} value={stats.error} cls="text-red-500" />
          </div>
        )}

        {/* Top principals + ops */}
        {(topPrincipals.length > 0 || topOps.length > 0) && (
          <div className="grid grid-cols-2 gap-3 mb-3 text-[11px]">
            {topPrincipals.length > 0 && (
              <div>
                <div className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">
                  {t('audit.topPrincipals', '操作最多的身份')}
                </div>
                {topPrincipals.map(([p, n]) => (
                  <div key={p} className="flex justify-between">
                    <span className="font-mono truncate">{p}</span>
                    <span className="tabular-nums text-muted-foreground">{n}</span>
                  </div>
                ))}
              </div>
            )}
            {topOps.length > 0 && (
              <div>
                <div className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">
                  {t('audit.topOps', '出现最多的操作')}
                </div>
                {topOps.map(([op, n]) => (
                  <div key={op} className="flex justify-between">
                    <span className="font-mono truncate">{op}</span>
                    <span className="tabular-nums text-muted-foreground">{n}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Filter row */}
        <div className="flex items-center gap-2 flex-wrap">
          <FilterIcon className="h-3 w-3 text-muted-foreground" />
          <input
            type="text"
            value={filter.principal}
            onChange={(e) => setFilter({ principal: e.target.value })}
            placeholder={t('audit.filter.principal', 'Principal…')}
            className="px-2 py-1 text-xs bg-muted/30 border border-border rounded w-32"
          />
          <input
            type="text"
            value={filter.operation}
            onChange={(e) => setFilter({ operation: e.target.value })}
            placeholder={t('audit.filter.operation', 'Operation…')}
            className="px-2 py-1 text-xs bg-muted/30 border border-border rounded w-32"
          />
          <select
            value={filter.outcome}
            onChange={(e) => setFilter({ outcome: e.target.value })}
            className="px-2 py-1 text-xs bg-muted/30 border border-border rounded"
          >
            <option value="">{t('audit.filter.outcomeAny', 'Outcome (any)')}</option>
            <option value="ok">ok</option>
            <option value="denied">denied</option>
            <option value="error">error</option>
          </select>
          <label className="flex items-center gap-1 text-xs text-muted-foreground">
            <input
              type="checkbox"
              checked={filter.onlyReversible}
              onChange={(e) => setFilter({ onlyReversible: e.target.checked })}
            />
            {t('audit.filter.onlyReversible', '仅可撤销')}
          </label>
          <button
            onClick={resetFilter}
            className="text-xs text-muted-foreground hover:text-foreground underline-offset-2 hover:underline ml-auto"
          >
            {t('audit.filter.reset', '重置')}
          </button>
        </div>
      </div>

      {error && (
        <div className="text-xs text-red-500 bg-red-500/10 border border-red-500/20 rounded-md px-3 py-2">{error}</div>
      )}

      <div className="rounded-lg border border-border bg-card overflow-hidden">
        {entries.length === 0 ? (
          <div className="text-xs text-muted-foreground p-6 text-center">
            {t('audit.empty', '尚无审计条目 — 改写敏感设置时会自动记录到这里。')}
          </div>
        ) : (
          <div className="max-h-[60vh] overflow-y-auto">
            {entries.map((e) => <EntryRow key={e.id} entry={e} />)}
          </div>
        )}
      </div>
    </div>
  )
}

function Tile({ label, value, cls }: { label: string; value: number; cls?: string }) {
  return (
    <div className="rounded-md border border-border bg-background/50 p-2">
      <div className="text-[10px] text-muted-foreground">{label}</div>
      <div className={cn('text-lg font-semibold tabular-nums', cls)}>{value}</div>
    </div>
  )
}
