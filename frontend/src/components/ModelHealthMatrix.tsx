import { useEffect, useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Activity, RefreshCw, CheckCircle2, AlertTriangle, Clock, WifiOff, Ban } from 'lucide-react'
import { cn } from '../lib/utils'
import { formatLocalTime } from '../lib/formatTime'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { RunModelHealthCheck, GetLastHealthCheckResults } from '../../wailsjs/go/main/App'

type Status = 'ok' | 'auth' | 'unreachable' | 'timeout' | 'error'

interface TestResult {
  providerId: string
  providerName: string
  status: Status
  latencyMs: number
  models: string[]
  error?: string
  testedAt: string
}

const STATUS_META: Record<Status, { icon: typeof CheckCircle2; cls: string; zh: string; en: string }> = {
  ok:          { icon: CheckCircle2,  cls: 'text-emerald-500', zh: '正常',   en: 'OK' },
  auth:        { icon: Ban,           cls: 'text-amber-500',   zh: '鉴权失败', en: 'Auth' },
  timeout:     { icon: Clock,         cls: 'text-orange-500',  zh: '超时',   en: 'Timeout' },
  unreachable: { icon: WifiOff,       cls: 'text-red-500',     zh: '不可达',  en: 'Unreachable' },
  error:       { icon: AlertTriangle, cls: 'text-red-500',     zh: '错误',   en: 'Error' },
}

interface Props {
  // When true (Settings providers tab), probe custom providers too.
  includeCustom?: boolean
}

export function ModelHealthMatrix({ includeCustom = true }: Props) {
  const { t, i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh') ?? true
  const [results, setResults] = useState<TestResult[]>([])
  const [running, setRunning] = useState(false)

  // Load cached results on mount.
  useEffect(() => {
    GetLastHealthCheckResults()
      .then((r) => setResults((r as TestResult[]) || []))
      .catch(() => {})
  }, [])

  // Subscribe to streaming progress + done.
  useEffect(() => {
    const offProgress = EventsOn('model:test:progress', (raw: unknown) => {
      const r = raw as TestResult
      setResults((prev) => {
        const next = prev.filter((x) => x.providerId !== r.providerId)
        return [...next, r]
      })
    })
    const offDone = EventsOn('model:test:done', (raw: unknown) => {
      const list = (raw as TestResult[]) || []
      if (list.length > 0) setResults(list)
      setRunning(false)
    })
    return () => {
      if (typeof offProgress === 'function') offProgress()
      if (typeof offDone === 'function') offDone()
    }
  }, [])

  const run = useCallback(async () => {
    setRunning(true)
    setResults([])
    try {
      await RunModelHealthCheck(includeCustom)
    } catch {
      setRunning(false)
    }
  }, [includeCustom])

  const sorted = [...results].sort((a, b) => a.providerName.localeCompare(b.providerName))

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Activity className="h-4 w-4 text-primary" />
          <h3 className="text-sm font-medium">{t('modelHealth.title', '模型可用性')}</h3>
        </div>
        <button
          onClick={run}
          disabled={running}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-md bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          {running ? <RefreshCw className="h-3.5 w-3.5 animate-spin" /> : <Activity className="h-3.5 w-3.5" />}
          {t('modelHealth.run', '检测全部供应商')}
        </button>
      </div>

      <p className="text-[11px] text-muted-foreground">
        {t('modelHealth.disclaimer', '检测的是「端点是否在线 + 是否返回模型列表」，并非逐个模型可对话。每次会请求每个供应商的 /v1/models。')}
      </p>

      {sorted.length === 0 ? (
        <div className="text-center py-8 text-xs text-muted-foreground border border-dashed border-border rounded-lg">
          {running
            ? t('modelHealth.running', '检测中…')
            : t('modelHealth.empty', '尚未检测。点击上方按钮开始。')}
        </div>
      ) : (
        <div className="space-y-1.5">
          {sorted.map((r) => {
            const meta = STATUS_META[r.status] ?? STATUS_META.error
            const Icon = meta.icon
            return (
              <div key={r.providerId} className="flex items-center gap-3 p-2.5 rounded-lg border border-border bg-card">
                <Icon className={cn('h-4 w-4 shrink-0', meta.cls)} />
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium truncate">{r.providerName || r.providerId}</div>
                  {r.status === 'ok' ? (
                    <div className="text-[11px] text-muted-foreground">
                      {t('modelHealth.modelsListed', '{{count}} 个模型', { count: r.models?.length ?? 0 })}
                    </div>
                  ) : (
                    r.error && <div className="text-[11px] text-red-400 truncate">{r.error}</div>
                  )}
                </div>
                <div className="text-right shrink-0">
                  <div className={cn('text-xs font-medium', meta.cls)}>{isZh ? meta.zh : meta.en}</div>
                  {r.latencyMs > 0 && (
                    <div className="text-[10px] text-muted-foreground tabular-nums">{r.latencyMs} ms</div>
                  )}
                </div>
              </div>
            )
          })}
          {sorted[0]?.testedAt && (
            <p className="text-[10px] text-muted-foreground/60 pt-1">
              {t('modelHealth.lastRun', '上次检测')} {formatLocalTime(sorted[0].testedAt)}
            </p>
          )}
        </div>
      )}
    </div>
  )
}
