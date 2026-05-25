import { useEffect, useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { ShieldCheck, ShieldAlert, ShieldQuestion, Shield, RefreshCw, AlertTriangle } from 'lucide-react'
import { cn } from '../lib/utils'
import { formatLocalTime } from '../lib/formatTime'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { RunModelAuthCheck, GetLastModelAuthResults } from '../../wailsjs/go/main/App'

type Verdict =
  | 'match'
  | 'mismatch'
  | 'inconclusive'
  | 'auth'
  | 'unreachable'
  | 'timeout'
  | 'error'

interface AuthResult {
  providerId: string
  providerName: string
  requestedModel: string
  reportedModel: string
  verdict: Verdict
  latencyMs: number
  note?: string
  testedAt: string
}

const VERDICT_META: Record<Verdict, { icon: typeof ShieldCheck; cls: string; zh: string; en: string }> = {
  match:        { icon: ShieldCheck,   cls: 'text-emerald-500', zh: '一致',    en: 'Match' },
  mismatch:     { icon: ShieldAlert,   cls: 'text-red-500',     zh: '不一致',  en: 'Mismatch' },
  inconclusive: { icon: ShieldQuestion,cls: 'text-amber-500',   zh: '不确定',  en: 'Inconclusive' },
  auth:         { icon: Shield,        cls: 'text-amber-500',   zh: '鉴权失败', en: 'Auth' },
  unreachable:  { icon: ShieldQuestion,cls: 'text-red-500',     zh: '不可达',  en: 'Unreachable' },
  timeout:      { icon: ShieldQuestion,cls: 'text-orange-500',  zh: '超时',    en: 'Timeout' },
  error:        { icon: ShieldAlert,   cls: 'text-red-500',     zh: '错误',    en: 'Error' },
}

interface Props {
  // When true, also probe user-defined custom providers (Settings has
  // them visible). On the gateway page we only show probe-configured
  // relay endpoints; flip to false there.
  includeCustom?: boolean
}

// ModelAuthenticityPanel mounts next to ModelHealthMatrix as a Wave3
// W3.4 surface. It detects upstream model SWAPS at the declaration
// layer — does the response's `model` field match the one we asked
// for? — and is explicit that it cannot detect deeper impersonation.
export function ModelAuthenticityPanel({ includeCustom = true }: Props) {
  const { t, i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh') ?? true
  const [results, setResults] = useState<AuthResult[]>([])
  const [running, setRunning] = useState(false)
  const [confirming, setConfirming] = useState(false)

  useEffect(() => {
    GetLastModelAuthResults()
      .then((r) => setResults((r as AuthResult[]) || []))
      .catch(() => {})
  }, [])

  useEffect(() => {
    const offProgress = EventsOn('model:auth:progress', (raw: unknown) => {
      const r = raw as AuthResult
      setResults((prev) => {
        const next = prev.filter(
          (x) => !(x.providerId === r.providerId && x.requestedModel === r.requestedModel),
        )
        return [...next, r]
      })
    })
    const offDone = EventsOn('model:auth:done', (raw: unknown) => {
      const list = (raw as AuthResult[]) || []
      if (list.length > 0) setResults(list)
      setRunning(false)
      setConfirming(false)
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
      await RunModelAuthCheck(includeCustom)
    } catch {
      setRunning(false)
      setConfirming(false)
    }
  }, [includeCustom])

  const sorted = [...results].sort((a, b) => {
    if (a.providerName !== b.providerName) return a.providerName.localeCompare(b.providerName)
    return a.requestedModel.localeCompare(b.requestedModel)
  })
  const mismatchCount = sorted.filter((r) => r.verdict === 'mismatch').length

  return (
    <div data-testid="model-authenticity-panel" className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <ShieldCheck className="h-4 w-4 text-primary" />
          <h3 className="text-sm font-medium">{t('modelAuth.title', '模型真伪检测')}</h3>
          {mismatchCount > 0 && (
            <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-red-500/15 text-red-400">
              {t('modelAuth.mismatchCount', '{{n}} 不一致', { n: mismatchCount })}
            </span>
          )}
        </div>
        {!confirming ? (
          <button
            onClick={() => setConfirming(true)}
            disabled={running}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-md border border-border hover:bg-muted disabled:opacity-50"
          >
            {running ? <RefreshCw className="h-3.5 w-3.5 animate-spin" /> : <ShieldCheck className="h-3.5 w-3.5" />}
            {t('modelAuth.run', '检测真伪')}
          </button>
        ) : (
          <div className="flex items-center gap-1.5">
            <button
              onClick={() => setConfirming(false)}
              className="px-2.5 py-1.5 text-[11px] rounded-md border border-border hover:bg-muted"
            >
              {t('modelAuth.cancel', '取消')}
            </button>
            <button
              onClick={run}
              className="px-2.5 py-1.5 text-[11px] rounded-md bg-amber-500/15 text-amber-500 hover:bg-amber-500/25"
            >
              {t('modelAuth.confirmCost', '确认烧 token')}
            </button>
          </div>
        )}
      </div>

      <div className="rounded border border-amber-500/30 bg-amber-500/5 p-2.5 flex gap-2 items-start">
        <AlertTriangle className="h-3.5 w-3.5 text-amber-500 mt-0.5 shrink-0" />
        <div className="text-[11px] text-muted-foreground space-y-1">
          <p>
            {t(
              'modelAuth.disclaimer1',
              '此检测只能识别「上游声明的 model 字段是否被替换」（最常见的偷换手法），无法识别更深层的模型指纹冒充。',
            )}
          </p>
          <p>
            {t(
              'modelAuth.disclaimer2',
              '每个 model 会发一次真实 chat 请求 (max_tokens=1)，会消耗少量上游 token。建议手动触发，不要轮询。',
            )}
          </p>
        </div>
      </div>

      {sorted.length === 0 ? (
        <div className="text-center py-8 text-xs text-muted-foreground border border-dashed border-border rounded-lg">
          {running
            ? t('modelAuth.running', '检测中…')
            : t('modelAuth.empty', '尚未检测。建议在购买新中转 / 怀疑被偷换时手动跑一次。')}
        </div>
      ) : (
        <div className="space-y-1.5">
          {sorted.map((r) => {
            const meta = VERDICT_META[r.verdict] ?? VERDICT_META.error
            const Icon = meta.icon
            return (
              <div
                key={r.providerId + r.requestedModel}
                className="flex items-center gap-3 p-2 rounded-lg border border-border bg-card"
              >
                <Icon className={cn('h-4 w-4 shrink-0', meta.cls)} />
                <div className="flex-1 min-w-0">
                  <div className="text-xs font-medium truncate">{r.providerName}</div>
                  <div className="text-[10px] text-muted-foreground font-mono tabular-nums truncate">
                    {t('modelAuth.requested', '请求')}: {r.requestedModel}
                    {r.reportedModel && r.reportedModel !== r.requestedModel && (
                      <>
                        {' '}→ {t('modelAuth.reported', '返回')}: {r.reportedModel}
                      </>
                    )}
                  </div>
                  {r.note && r.verdict !== 'match' && (
                    <div className="text-[10px] text-muted-foreground italic truncate">{r.note}</div>
                  )}
                </div>
                <div className="text-right shrink-0">
                  <div className={cn('text-[11px] font-medium', meta.cls)}>{isZh ? meta.zh : meta.en}</div>
                  {r.latencyMs > 0 && (
                    <div className="text-[10px] text-muted-foreground tabular-nums">{r.latencyMs} ms</div>
                  )}
                </div>
              </div>
            )
          })}
          {sorted[0]?.testedAt && (
            <p className="text-[10px] text-muted-foreground/60 pt-1">
              {t('modelAuth.lastRun', '上次检测')} {formatLocalTime(sorted[0].testedAt)}
            </p>
          )}
        </div>
      )}
    </div>
  )
}
