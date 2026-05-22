import { useCallback, useEffect, useState } from 'react'
import {
  Stethoscope,
  Loader2,
  CheckCircle2,
  XCircle,
  ArrowRight,
  Zap,
  Globe,
  Wand2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import {
  RunConnectivityDiagnostic,
  GetProxySettings,
  SaveProxySettings,
} from '../../wailsjs/go/main/App'
import { connectivity, proxy, netproxy } from '../../wailsjs/go/models'

/**
 * 360-style network "Doctor" panel.
 *
 * Probes Anthropic / OpenAI / Gemini / Lurus / GitHub / npm in parallel,
 * compares direct vs through-upstream-proxy reachability, scans common
 * loopback ports for an already-running V2Ray/Clash/MasterDnsVPN, and
 * surfaces one-click remedies. The diagnostic does no harm — every action
 * the user can take is explicit.
 */
export function ConnectivityDoctor({
  onSwitchToProxyTab,
  onApplyProxy,
}: {
  onSwitchToProxyTab?: () => void
  onApplyProxy?: (url: string) => void
}) {
  const { t } = useTranslation()
  const [running, setRunning] = useState(false)
  const [report, setReport] = useState<connectivity.Report | null>(null)
  const [applyBusy, setApplyBusy] = useState(false)

  const run = useCallback(async () => {
    setRunning(true)
    try {
      const r = await RunConnectivityDiagnostic()
      setReport(r)
    } catch (err) {
      console.error('connectivity diagnostic failed:', err)
    } finally {
      setRunning(false)
    }
  }, [])

  useEffect(() => {
    void run()
  }, [run])

  const applyProxyURL = async (url: string) => {
    setApplyBusy(true)
    try {
      const current = await GetProxySettings()
      const next = proxy.ProxySettings.createFrom({
        ...(current ?? {}),
        upstreamProxy: netproxy.Settings.createFrom({
          enabled: true,
          url,
          noProxy: current?.upstreamProxy?.noProxy ?? '',
          testUrl: current?.upstreamProxy?.testUrl ?? '',
        }),
      })
      await SaveProxySettings(next)
      if (onApplyProxy) onApplyProxy(url)
      await run()
    } catch (err) {
      console.error('apply proxy failed:', err)
    } finally {
      setApplyBusy(false)
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-2">
          <Stethoscope className="h-4 w-4 mt-0.5 text-muted-foreground" />
          <div>
            <h3 className="text-sm font-semibold">
              {t('doctor.title', '连通性诊断')}
            </h3>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t(
                'doctor.subtitle',
                '同时检测主流 AI 服务商、Lurus Hub 和基础设施服务的可达性，对比直连与经上游代理的结果，并给出一键解决方案。',
              )}
            </p>
          </div>
        </div>
        <button
          onClick={run}
          disabled={running}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md border border-border hover:bg-muted shrink-0',
            'disabled:opacity-50 disabled:cursor-not-allowed',
          )}
        >
          {running ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Stethoscope className="h-3.5 w-3.5" />
          )}
          {t('doctor.rerun', '重新诊断')}
        </button>
      </div>

      {report && (
        <>
          {report.suggestions && report.suggestions.length > 0 && (
            <div className="space-y-2">
              {report.suggestions.map((s, i) => (
                <SuggestionCard
                  key={`${s.kind}-${i}`}
                  suggestion={s}
                  busy={applyBusy}
                  onApply={() => {
                    if (s.kind === 'auto-fill-proxy' && s.payload) {
                      void applyProxyURL(s.payload)
                    } else if (s.kind === 'use-upstream' && onSwitchToProxyTab) {
                      onSwitchToProxyTab()
                    }
                  }}
                />
              ))}
            </div>
          )}

          <ProviderMatrix report={report} />

          {report.localProxies && report.localProxies.length > 0 && (
            <LocalProxyList
              proxies={report.localProxies}
              onPick={(url) => void applyProxyURL(url)}
              busy={applyBusy}
            />
          )}

          {(report.systemProxy?.httpProxy || report.systemProxy?.httpsProxy || report.systemProxy?.allProxy) && (
            <SystemProxyHint sys={report.systemProxy} onPick={(url) => void applyProxyURL(url)} busy={applyBusy} />
          )}
        </>
      )}
    </div>
  )
}

function SuggestionCard({
  suggestion,
  busy,
  onApply,
}: {
  suggestion: connectivity.Suggestion
  busy: boolean
  onApply: () => void
}) {
  const { t } = useTranslation()

  const tone =
    suggestion.kind === 'all-ok'
      ? 'border-green-500/30 bg-green-500/5 text-green-700 dark:text-green-300'
      : suggestion.kind === 'auto-fill-proxy'
        ? 'border-primary/30 bg-primary/5'
        : 'border-amber-500/30 bg-amber-500/5'
  const Icon =
    suggestion.kind === 'all-ok'
      ? CheckCircle2
      : suggestion.kind === 'auto-fill-proxy'
        ? Wand2
        : Zap

  const actionable =
    suggestion.kind === 'auto-fill-proxy' || suggestion.kind === 'use-upstream'

  return (
    <div className={cn('rounded-md border p-3 flex items-start gap-2', tone)}>
      <Icon className="h-4 w-4 mt-0.5 shrink-0" />
      <div className="min-w-0 flex-1">
        <div className="text-sm font-medium">{suggestion.title}</div>
        <div className="text-xs text-muted-foreground mt-0.5 leading-relaxed">
          {suggestion.detail}
        </div>
        {suggestion.payload && (
          <div className="text-[11px] font-mono mt-1 opacity-70 break-all">
            → {suggestion.payload}
          </div>
        )}
      </div>
      {actionable && (
        <button
          onClick={onApply}
          disabled={busy}
          className={cn(
            'shrink-0 px-2.5 py-1 text-xs font-medium rounded border border-current hover:bg-muted/40 transition-colors',
            'disabled:opacity-50 disabled:cursor-not-allowed inline-flex items-center gap-1',
          )}
        >
          {busy ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <ArrowRight className="h-3 w-3" />
          )}
          {suggestion.kind === 'auto-fill-proxy'
            ? t('doctor.applyProxy', '一键应用')
            : t('doctor.goConfigure', '去配置')}
        </button>
      )}
    </div>
  )
}

function ProviderMatrix({ report }: { report: connectivity.Report }) {
  const { t } = useTranslation()
  const anyUpstream = report.providers?.some((p) => p.upstreamTried)
  return (
    <div className="rounded-md border border-border overflow-hidden">
      <table className="w-full text-xs">
        <thead className="bg-muted/40 text-muted-foreground">
          <tr>
            <th className="text-left px-3 py-2 font-medium">
              {t('doctor.col.provider', '服务')}
            </th>
            <th className="text-left px-3 py-2 font-medium">DNS</th>
            <th className="text-left px-3 py-2 font-medium">
              {t('doctor.col.direct', '直连')}
            </th>
            {anyUpstream && (
              <th className="text-left px-3 py-2 font-medium">
                {t('doctor.col.upstream', '经代理')}
              </th>
            )}
          </tr>
        </thead>
        <tbody>
          {report.providers?.map((p) => (
            <tr key={p.provider.id} className="border-t border-border">
              <td className="px-3 py-2">
                <div className="font-medium">{p.provider.label}</div>
                <div className="text-[10px] text-muted-foreground font-mono truncate max-w-[200px]">
                  {p.provider.url}
                </div>
              </td>
              <td className="px-3 py-2">
                <StateDot ok={p.dnsOK} err={p.dnsError} />
              </td>
              <td className="px-3 py-2">
                <StateDot ok={p.directOK} ms={p.directMs} err={p.directError} />
              </td>
              {anyUpstream && (
                <td className="px-3 py-2">
                  {p.upstreamTried ? (
                    <StateDot
                      ok={p.upstreamOK}
                      ms={p.upstreamMs}
                      err={p.upstreamError}
                    />
                  ) : (
                    <span className="text-muted-foreground/50">—</span>
                  )}
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function StateDot({
  ok,
  ms,
  err,
}: {
  ok?: boolean
  ms?: number
  err?: string
}) {
  if (ok) {
    return (
      <span
        className="inline-flex items-center gap-1 text-green-600 dark:text-green-400"
        title={ms ? `${ms}ms` : undefined}
      >
        <CheckCircle2 className="h-3 w-3" />
        {typeof ms === 'number' && ms > 0 && (
          <span className="opacity-70">{ms}ms</span>
        )}
      </span>
    )
  }
  return (
    <span className="inline-flex items-center gap-1 text-red-500" title={err}>
      <XCircle className="h-3 w-3" />
    </span>
  )
}

function LocalProxyList({
  proxies,
  onPick,
  busy,
}: {
  proxies: connectivity.LocalProxy[]
  onPick: (url: string) => void
  busy: boolean
}) {
  const { t } = useTranslation()
  return (
    <details className="rounded-md border border-border p-3" open>
      <summary className="cursor-pointer text-sm font-medium flex items-center gap-1.5">
        <Globe className="h-3.5 w-3.5" />
        {t('doctor.localProxies', '检测到的本地代理')} ({proxies.length})
      </summary>
      <div className="mt-2 space-y-1">
        {proxies.map((p) => (
          <div
            key={`${p.host}:${p.port}`}
            className="flex items-center justify-between text-xs p-2 bg-muted/30 rounded"
          >
            <div>
              <span className="font-mono">{p.url}</span>
              {p.guessedName && (
                <span className="ml-2 text-muted-foreground">({p.guessedName})</span>
              )}
            </div>
            <button
              onClick={() => onPick(p.url)}
              disabled={busy}
              className="px-2 py-0.5 text-[11px] border border-border rounded hover:bg-background disabled:opacity-50"
            >
              {t('doctor.useThis', '使用')}
            </button>
          </div>
        ))}
      </div>
    </details>
  )
}

function SystemProxyHint({
  sys,
  onPick,
  busy,
}: {
  sys: connectivity.SystemProxy
  onPick: (url: string) => void
  busy: boolean
}) {
  const { t } = useTranslation()
  const pick = sys.allProxy || sys.httpsProxy || sys.httpProxy
  if (!pick) return null
  return (
    <div className="rounded-md border border-border p-3 bg-muted/30">
      <div className="text-xs font-medium mb-1">
        {t('doctor.systemProxyTitle', '系统环境变量里发现代理')}
      </div>
      <div className="text-[11px] font-mono break-all opacity-80">{pick}</div>
      <button
        onClick={() => onPick(pick)}
        disabled={busy}
        className="mt-2 px-2 py-0.5 text-[11px] border border-border rounded hover:bg-background disabled:opacity-50"
      >
        {t('doctor.useThis', '使用')}
      </button>
    </div>
  )
}
