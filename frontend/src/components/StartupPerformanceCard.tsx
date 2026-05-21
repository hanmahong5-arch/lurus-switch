import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Gauge, RefreshCw, ChevronDown, ChevronRight } from 'lucide-react'
import { GetStartupTrace, GetStartupHistory } from '../../wailsjs/go/main/App'
import { cn } from '../lib/utils'
import { formatLocal } from '../lib/formatTime'

interface Phase {
  name: string
  startedAt: string
  durationMs: number
}

interface Trace {
  coldStartMs: number
  guiReadyMs: number
  phases: Phase[]
  capturedAt: string
}

// Map internal phase names to human labels. Unknown phases fall back to
// the raw name so a newly-added marker still renders.
const PHASE_LABEL: Record<string, { zh: string; en: string }> = {
  'services-init': { zh: '服务初始化', en: 'Services init' },
  'activity-bus': { zh: '活动总线', en: 'Activity bus' },
  'audit-undo-handlers': { zh: '审计撤销注册', en: 'Audit undo handlers' },
  'whitelabel-sidecar': { zh: '白标边车', en: 'White-label sidecar' },
  'gateway-autostart': { zh: '网关自启动', en: 'Gateway autostart' },
  'heartbeat-init': { zh: '心跳初始化', en: 'Heartbeat init' },
  'live-watcher': { zh: '会话监视器', en: 'Live watcher' },
  'notify-subsystem': { zh: '通知子系统', en: 'Notify subsystem' },
  'tray-start': { zh: '托盘启动', en: 'Tray start' },
  'hotkey-start': { zh: '全局热键', en: 'Hotkeys' },
}

function phaseLabel(name: string, isZh: boolean): string {
  const l = PHASE_LABEL[name]
  if (!l) return name
  return isZh ? l.zh : l.en
}

export function StartupPerformanceCard() {
  const { t, i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh') ?? true
  const [trace, setTrace] = useState<Trace | null>(null)
  const [history, setHistory] = useState<Trace[]>([])
  const [loading, setLoading] = useState(false)
  const [expanded, setExpanded] = useState(false)

  const load = async () => {
    setLoading(true)
    try {
      const [cur, hist] = await Promise.all([
        GetStartupTrace() as Promise<Trace>,
        GetStartupHistory() as Promise<Trace[]>,
      ])
      setTrace(cur)
      setHistory(hist || [])
    } catch {
      // best-effort — diagnostics are non-critical
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  // Previous launch = most recent persisted trace that isn't the current
  // process. History is newest-first; the current process's trace is only
  // persisted on the *next* launch, so history[0] is the previous launch.
  const previous = history.length > 0 ? history[0] : null
  const coldDelta = trace && previous ? trace.coldStartMs - previous.coldStartMs : null

  return (
    <div className="border-t border-border pt-4">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <Gauge className="h-4 w-4 text-primary" />
          <h3 className="text-sm font-medium">{t('settings.startup.title', '启动性能')}</h3>
        </div>
        <button
          onClick={load}
          disabled={loading}
          className="p-1 rounded hover:bg-muted disabled:opacity-50"
          title={t('ui.refresh', '刷新')}
        >
          <RefreshCw className={cn('h-3.5 w-3.5', loading && 'animate-spin')} />
        </button>
      </div>

      {!trace ? (
        <p className="text-xs text-muted-foreground">{t('settings.startup.empty', '暂无启动数据')}</p>
      ) : (
        <>
          {/* Milestones */}
          <div className="grid grid-cols-2 gap-2 mb-3">
            <Milestone
              label={t('settings.startup.guiReady', 'GUI 就绪')}
              valueMs={trace.guiReadyMs}
            />
            <Milestone
              label={t('settings.startup.allReady', '全部就绪')}
              valueMs={trace.coldStartMs}
              deltaMs={coldDelta}
            />
          </div>

          {/* Phase breakdown (collapsible) */}
          <button
            onClick={() => setExpanded(!expanded)}
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
          >
            {expanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
            {t('settings.startup.phases', '阶段明细')} ({trace.phases.length})
          </button>
          {expanded && (
            <div className="mt-2 space-y-1">
              {trace.phases.map((p, i) => (
                <div key={`${p.name}-${i}`} className="flex items-center gap-2 text-[11px]">
                  <span className="flex-1 truncate text-muted-foreground">{phaseLabel(p.name, isZh)}</span>
                  <PhaseBar durationMs={p.durationMs} maxMs={maxPhase(trace.phases)} />
                  <span className="tabular-nums w-14 text-right font-mono">{p.durationMs} ms</span>
                </div>
              ))}
              {trace.capturedAt && (
                <p className="text-[10px] text-muted-foreground/60 pt-1">
                  {t('settings.startup.capturedAt', '采集于')} {formatLocal(trace.capturedAt)}
                </p>
              )}
            </div>
          )}
        </>
      )}
    </div>
  )
}

function maxPhase(phases: Phase[]): number {
  return phases.reduce((m, p) => Math.max(m, p.durationMs), 1)
}

function Milestone({ label, valueMs, deltaMs }: { label: string; valueMs: number; deltaMs?: number | null }) {
  return (
    <div className="rounded-md border border-border bg-background/50 p-2">
      <div className="text-[10px] text-muted-foreground">{label}</div>
      <div className="flex items-baseline gap-1.5">
        <span className="text-lg font-semibold tabular-nums">{valueMs}</span>
        <span className="text-[10px] text-muted-foreground">ms</span>
        {deltaMs != null && deltaMs !== 0 && (
          <span
            className={cn(
              'text-[10px] tabular-nums ml-auto',
              deltaMs < 0 ? 'text-emerald-500' : 'text-red-500',
            )}
          >
            {deltaMs < 0 ? '▼' : '▲'} {Math.abs(deltaMs)} ms
          </span>
        )}
      </div>
    </div>
  )
}

function PhaseBar({ durationMs, maxMs }: { durationMs: number; maxMs: number }) {
  const pct = Math.max(2, Math.round((durationMs / maxMs) * 100))
  return (
    <div className="w-20 h-1.5 bg-muted rounded-full overflow-hidden">
      <div
        className={cn('h-full rounded-full', durationMs > 500 ? 'bg-amber-500' : 'bg-primary')}
        style={{ width: `${pct}%` }}
      />
    </div>
  )
}
