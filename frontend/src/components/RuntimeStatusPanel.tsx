import { useEffect, useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  RefreshCw, Loader2, Wifi, WifiOff, AlertTriangle, CheckCircle2,
  Cpu, Activity, KeyRound, MapPin,
} from 'lucide-react'
import { cn } from '../lib/utils'
import { GetToolRuntimes } from '../../wailsjs/go/main/App'
import type { toolruntime } from '../../wailsjs/go/models'

const CONN_TREATMENT: Record<string, { Icon: typeof Wifi; color: string; bg: string; border: string; labelZh: string; labelEn: string }> = {
  reachable: {
    Icon: CheckCircle2,
    color: 'text-emerald-400',
    bg: 'bg-emerald-500/10',
    border: 'border-emerald-500/30',
    labelZh: '在线',
    labelEn: 'Online',
  },
  degraded: {
    Icon: AlertTriangle,
    color: 'text-amber-400',
    bg: 'bg-amber-500/10',
    border: 'border-amber-500/30',
    labelZh: '降级',
    labelEn: 'Degraded',
  },
  down: {
    Icon: WifiOff,
    color: 'text-red-400',
    bg: 'bg-red-500/10',
    border: 'border-red-500/30',
    labelZh: '不通',
    labelEn: 'Down',
  },
  unknown: {
    Icon: Wifi,
    color: 'text-zinc-400',
    bg: 'bg-zinc-500/10',
    border: 'border-zinc-500/30',
    labelZh: '未知',
    labelEn: 'Unknown',
  },
}

const TOOL_DISPLAY: Record<string, string> = {
  claude: 'Claude Code',
  codex: 'Codex',
  gemini: 'Gemini CLI',
  picoclaw: 'PicoClaw',
  nullclaw: 'NullClaw',
  zeroclaw: 'ZeroClaw',
  openclaw: 'OpenClaw',
}

const ENDPOINT_KIND_LABEL: Record<string, { zh: string; en: string; color: string }> = {
  official: { zh: '官方', en: 'Official', color: 'text-cyan-400' },
  'lurus-gateway': { zh: 'Switch 网关', en: 'Switch Gateway', color: 'text-violet-400' },
  'third-party': { zh: '第三方', en: 'Third-party', color: 'text-orange-400' },
  unknown: { zh: '未配置', en: 'Unset', color: 'text-zinc-500' },
}

// Auto-refresh cadence — 30s is enough for "live" feel without
// hammering vendor endpoints. User can also click Refresh.
const AUTO_REFRESH_MS = 30_000

export function RuntimeStatusPanel() {
  const { t, i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh') ?? true
  const [runtimes, setRuntimes] = useState<toolruntime.ToolRuntime[]>([])
  const [loading, setLoading] = useState(false)
  const [lastRefreshed, setLastRefreshed] = useState<Date | null>(null)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const rs = await GetToolRuntimes()
      setRuntimes(rs ?? [])
      setLastRefreshed(new Date())
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
    const id = setInterval(refresh, AUTO_REFRESH_MS)
    return () => clearInterval(id)
  }, [refresh])

  // Filter to only installed tools — uninstalled rows are noise on the
  // dashboard. User can install from Tools page if they want one.
  const installed = runtimes.filter((r) => r.installed)

  return (
    <section className="rounded-lg border border-border bg-card/40 p-4">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-foreground">
            {t('home.runtime.title', '实时连接状态')}
          </h3>
          <span className="text-[10px] text-muted-foreground/70 font-normal">
            {t('home.runtime.titleEn', 'Live runtime status')}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {lastRefreshed && (
            <span className="text-[10px] text-muted-foreground/60 tabular-nums">
              {isZh ? '刷新于' : 'Refreshed'} {lastRefreshed.toLocaleTimeString()}
            </span>
          )}
          <button
            type="button"
            onClick={refresh}
            disabled={loading}
            className="h-6 w-6 inline-flex items-center justify-center rounded hover:bg-muted text-muted-foreground"
            title={isZh ? '立即刷新' : 'Refresh now'}
          >
            {loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
          </button>
        </div>
      </div>

      {error && (
        <div className="rounded-md border border-red-500/30 bg-red-950/20 text-red-200 text-xs px-3 py-2 mb-2">
          {error}
        </div>
      )}

      {installed.length === 0 && !loading && (
        <div className="rounded-md border border-dashed border-border/60 p-4 text-center text-xs text-muted-foreground">
          {isZh ? '暂无安装的 CLI 工具——去工具页装一个吧。' : 'No CLI tools installed yet — head to the Tools page.'}
        </div>
      )}

      <div className="space-y-2">
        {installed.map((rt) => (
          <RuntimeRow key={rt.tool} rt={rt} isZh={isZh} />
        ))}
      </div>
    </section>
  )
}

function RuntimeRow({ rt, isZh }: { rt: toolruntime.ToolRuntime; isZh: boolean }) {
  const conn = CONN_TREATMENT[rt.connState ?? 'unknown'] ?? CONN_TREATMENT.unknown
  const ConnIcon = conn.Icon
  const kind = ENDPOINT_KIND_LABEL[rt.endpointKind ?? 'unknown'] ?? ENDPOINT_KIND_LABEL.unknown

  return (
    <div className={cn('rounded-md border p-3', conn.border, conn.bg)}>
      <div className="flex items-start gap-3">
        <ConnIcon className={cn('h-5 w-5 shrink-0 mt-0.5', conn.color)} />
        <div className="flex-1 min-w-0">
          {/* Header row */}
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm font-medium text-foreground">{TOOL_DISPLAY[rt.tool] ?? rt.tool}</span>
            <span className={cn('text-[10px] font-medium uppercase tracking-wider', conn.color)}>
              {isZh ? conn.labelZh : conn.labelEn}
            </span>
            {rt.latencyMs != null && rt.latencyMs > 0 && (
              <span className="text-[10px] text-muted-foreground tabular-nums">
                · {rt.latencyMs}ms
              </span>
            )}
            {rt.processRunning && (
              <span className="inline-flex items-center gap-1 text-[10px] text-emerald-400 bg-emerald-500/15 border border-emerald-500/30 rounded px-1.5 py-0.5">
                <Activity className="h-2.5 w-2.5" />
                {isZh ? '进程运行中' : 'Running'}
                {(rt.processPID ?? 0) > 0 && <span className="opacity-70 tabular-nums">PID {rt.processPID}</span>}
              </span>
            )}
          </div>

          {/* Metadata grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-x-4 gap-y-1 mt-1.5 text-[11px]">
            <Field icon={MapPin} label={isZh ? '端点' : 'Endpoint'}>
              <span className="font-mono text-muted-foreground/90 break-all">{rt.endpoint || '—'}</span>
              {rt.endpointKind && rt.endpointKind !== 'unknown' && (
                <span className={cn('ml-1.5 text-[9px] font-medium uppercase', kind.color)}>
                  {isZh ? kind.zh : kind.en}
                </span>
              )}
            </Field>
            <Field icon={Cpu} label={isZh ? '模型' : 'Model'}>
              <span className="font-mono text-muted-foreground/90 truncate">
                {rt.model || (isZh ? '默认' : 'default')}
              </span>
            </Field>
            <Field icon={KeyRound} label={isZh ? 'API Key' : 'API key'}>
              <span className={rt.hasApiKey ? 'text-emerald-400' : 'text-amber-400'}>
                {rt.hasApiKey
                  ? (isZh ? '已配置' : 'Set')
                  : (isZh ? '未配置（用环境变量？）' : 'Unset (using env var?)')}
              </span>
            </Field>
            <Field icon={Activity} label={isZh ? '配置文件' : 'Config'}>
              <span className="font-mono text-muted-foreground/70 truncate text-[10px]">
                {rt.configPath || '—'}
              </span>
            </Field>
          </div>

          {rt.probeError && (
            <div className="mt-1.5 text-[10px] text-red-300/80 font-mono break-all">
              {rt.probeError}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function Field({ icon: Icon, label, children }: { icon: typeof Wifi; label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-1.5 min-w-0">
      <Icon className="h-3 w-3 shrink-0 text-muted-foreground/60" />
      <span className="text-muted-foreground/60 shrink-0">{label}:</span>
      <span className="min-w-0 truncate flex-1">{children}</span>
    </div>
  )
}
