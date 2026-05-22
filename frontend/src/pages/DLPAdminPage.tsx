import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Shield, AlertTriangle, Eye, EyeOff, RefreshCw, FlaskConical, Trash2 } from 'lucide-react'
import { useDLPStore, type DLPPolicy, type DLPHitRecord } from '../stores/dlpStore'
import { Button, Card, KpiCard } from '../components/ui'

const POLICIES: DLPPolicy[] = ['allow', 'warn', 'redact', 'block']

const POLICY_BADGE: Record<DLPPolicy, string> = {
  allow:  'bg-emerald-500/15 text-emerald-400 ring-emerald-500/30',
  warn:   'bg-amber-500/15  text-amber-400  ring-amber-500/30',
  redact: 'bg-blue-500/15   text-blue-400   ring-blue-500/30',
  block:  'bg-red-500/15    text-red-400    ring-red-500/30',
}

const SEVERITY_DOT: Record<string, string> = {
  info:     'bg-slate-400',
  warning:  'bg-amber-400',
  critical: 'bg-red-500',
}

export function DLPAdminPage() {
  const { t } = useTranslation()
  const {
    patterns, hits, stats, loading, error, scanResult, scanInput,
    load, setPolicy, removePattern, scan, setScanInput,
  } = useDLPStore()

  const [pollOn, setPollOn] = useState(true)

  useEffect(() => {
    void load()
  }, [load])

  // Live polling (only while page is visible). 5s cadence is enough to
  // see hits without hammering the gateway thread.
  useEffect(() => {
    if (!pollOn) return
    const h = setInterval(() => {
      if (document.visibilityState === 'visible') void load()
    }, 5000)
    return () => clearInterval(h)
  }, [pollOn, load])

  const blockedCount = stats?.byPolicy?.block ?? 0
  const redactedCount = stats?.byPolicy?.redact ?? 0
  const warnedCount = stats?.byPolicy?.warn ?? 0
  const totalHits = stats?.total ?? 0

  return (
    <div className="h-full overflow-auto p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-xl font-semibold flex items-center gap-2">
            <Shield className="h-5 w-5 text-primary" />
            {t('dlp.title', 'DLP — Data Loss Prevention')}
          </h1>
          <p className="text-xs text-muted-foreground mt-1">
            {t('dlp.subtitle', '扫描经过本机网关的每条请求，按规则阻断 / 脱敏 / 记录敏感数据。修改即时生效。')}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => setPollOn(!pollOn)}
            title={pollOn ? t('dlp.pausePolling', 'pause polling') : t('dlp.resumePolling', 'resume polling')}
            icon={pollOn ? <Eye className="h-3.5 w-3.5" /> : <EyeOff className="h-3.5 w-3.5" />}
            className={pollOn ? 'border-emerald-500/40 text-emerald-400 bg-emerald-500/10' : ''}
          >
            {pollOn ? t('dlp.live', '实时') : t('dlp.paused', '已暂停')}
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => void load()}
            disabled={loading}
            loading={loading}
            icon={!loading ? <RefreshCw className="h-3.5 w-3.5" /> : undefined}
          >
            {t('common.refresh', '刷新')}
          </Button>
        </div>
      </div>

      {error && (
        <Card variant="default" className="mb-3 p-2 border-red-500/30 bg-red-500/10 text-red-400 text-xs flex items-center gap-2 font-mono">
          <AlertTriangle className="h-3.5 w-3.5" />
          ▸ {error}
        </Card>
      )}

      {/* Stats tiles */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
        <KpiCard label={t('dlp.stats.total', '总命中')} value={totalHits.toLocaleString()} />
        <KpiCard label={t('dlp.stats.blocked', '已阻断')} value={blockedCount.toLocaleString()} />
        <KpiCard label={t('dlp.stats.redacted', '已脱敏')} value={redactedCount.toLocaleString()} />
        <KpiCard label={t('dlp.stats.warned', '仅告警')} value={warnedCount.toLocaleString()} />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Pattern table — 2 columns of the 3-column grid */}
        <section className="lg:col-span-2 rounded-lg border border-border bg-card">
          <header className="p-3 border-b border-border">
            <h2 className="text-sm font-medium">{t('dlp.patterns.title', '规则集')}</h2>
            <p className="text-[11px] text-muted-foreground mt-0.5">
              {t('dlp.patterns.hint', 'Block 直接拒绝；Redact 把命中部分替换为占位符；Warn / Allow 不改请求只记录。')}
            </p>
          </header>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead className="text-[10px] uppercase tracking-wider text-muted-foreground bg-muted/30">
                <tr>
                  <th className="text-left px-3 py-2">{t('dlp.col.severity', 'sev')}</th>
                  <th className="text-left px-3 py-2">{t('dlp.col.name', '规则')}</th>
                  <th className="text-left px-3 py-2">{t('dlp.col.tags', 'tags')}</th>
                  <th className="text-left px-3 py-2">{t('dlp.col.policy', '策略')}</th>
                  <th className="text-right px-3 py-2"></th>
                </tr>
              </thead>
              <tbody>
                {patterns.length === 0 && !loading && (
                  <tr><td colSpan={5} className="px-3 py-6 text-center text-muted-foreground">{t('common.empty', '暂无数据')}</td></tr>
                )}
                {patterns.map((p) => (
                  <tr key={p.name} className="border-t border-border/50 hover:bg-muted/30">
                    <td className="px-3 py-2">
                      <span className={`inline-block h-2 w-2 rounded-full ${SEVERITY_DOT[p.severity] ?? 'bg-slate-400'}`} title={p.severity} />
                    </td>
                    <td className="px-3 py-2 font-mono">
                      <div className="font-medium">{p.name}</div>
                      <div className="text-[10px] text-muted-foreground mt-0.5 truncate max-w-[26ch]" title={p.description}>{p.description}</div>
                    </td>
                    <td className="px-3 py-2">
                      <div className="flex flex-wrap gap-1">
                        {(p.tags ?? []).map(tg => (
                          <span key={tg} className="px-1.5 py-0.5 rounded bg-muted text-[10px] font-mono">{tg}</span>
                        ))}
                      </div>
                    </td>
                    <td className="px-3 py-2">
                      <select
                        value={p.policy}
                        onChange={(e) => void setPolicy(p.name, e.target.value as DLPPolicy)}
                        className={`px-2 py-1 rounded border ring-1 text-[11px] font-medium ${POLICY_BADGE[p.policy]}`}
                      >
                        {POLICIES.map(pp => <option key={pp} value={pp}>{pp}</option>)}
                      </select>
                    </td>
                    <td className="px-3 py-2 text-right">
                      <button
                        onClick={() => void removePattern(p.name)}
                        title={t('dlp.removePattern', '删除规则')}
                        className="text-muted-foreground hover:text-red-400 p-1"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Right column: scan tester + recent hits */}
        <aside className="space-y-4">
          <ScanTester
            input={scanInput}
            setInput={setScanInput}
            onRun={() => void scan(scanInput)}
            result={scanResult}
          />
          <RecentHits hits={hits} />
        </aside>
      </div>
    </div>
  )
}

function ScanTester({
  input, setInput, onRun, result,
}: {
  input: string
  setInput: (s: string) => void
  onRun: () => void
  result: ReturnType<typeof useDLPStore.getState>['scanResult']
}) {
  const { t } = useTranslation()
  return (
    <Card as="section" variant="default">
      <header className="p-3 border-b border-border">
        <h2 className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground flex items-center gap-2">
          <FlaskConical className="h-3.5 w-3.5 text-primary" />
          [ {t('dlp.tester.title', '即时扫描测试').toUpperCase()} ]
        </h2>
      </header>
      <div className="p-3 space-y-2">
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder={t('dlp.tester.placeholder', '在此粘贴一段要测试的 prompt，点「扫描」查看会被哪些规则触发。')}
          className="w-full h-24 px-2 py-1.5 rounded border border-border bg-background text-xs font-mono resize-y focus:outline-none focus:ring-1 focus:ring-primary"
        />
        <Button size="sm" onClick={onRun} className="w-full justify-center">
          {t('dlp.tester.run', '扫描')}
        </Button>
        {result && (
          <div className="space-y-1.5 pt-2 border-t border-border">
            <div className="flex items-center justify-between text-xs">
              <span className="text-muted-foreground">{t('dlp.tester.result', '结果')}:</span>
              <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${result.blocked ? POLICY_BADGE.block : POLICY_BADGE[result.highestPolicy] ?? POLICY_BADGE.allow}`}>
                {result.blocked ? t('dlp.blocked', 'BLOCKED') : result.highestPolicy.toUpperCase()}
              </span>
            </div>
            <div className="text-[11px] text-muted-foreground">
              {t('dlp.tester.hits', '命中')}: {result.hits.length}
            </div>
            {result.hits.length > 0 && (
              <ul className="text-[11px] space-y-0.5">
                {result.hits.slice(0, 8).map((h, i) => (
                  <li key={i} className="flex items-center gap-1.5">
                    <span className={`inline-block h-1.5 w-1.5 rounded-full ${SEVERITY_DOT[h.severity] ?? 'bg-slate-400'}`} />
                    <span className="font-mono">{h.patternName}</span>
                    <span className="text-muted-foreground">@ {h.start}-{h.end}</span>
                  </li>
                ))}
              </ul>
            )}
            {result.redacted && result.redacted !== input && (
              <details className="text-[11px]">
                <summary className="cursor-pointer text-muted-foreground hover:text-foreground font-mono">▾ {t('dlp.tester.redacted', '查看脱敏后版本')}</summary>
                <pre className="mt-1 p-2 rounded bg-card-recessed text-[10px] font-mono whitespace-pre-wrap break-all border border-border">{result.redacted}</pre>
              </details>
            )}
          </div>
        )}
      </div>
    </Card>
  )
}

function RecentHits({ hits }: { hits: DLPHitRecord[] }) {
  const { t } = useTranslation()
  const recent = useMemo(() => hits.slice(0, 10), [hits])
  return (
    <Card as="section" variant="default">
      <header className="p-3 border-b border-border">
        <h2 className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground">[ {t('dlp.recent.title', '最近命中').toUpperCase()} ]</h2>
        <p className="text-[10px] text-muted-foreground mt-0.5">
          {t('dlp.recent.hint', '环形缓冲（最多 200 条），由网关请求路径写入。')}
        </p>
      </header>
      <ul className="divide-y divide-border">
        {recent.length === 0 && (
          <li className="p-3 text-xs text-muted-foreground">{t('dlp.recent.empty', '暂无命中。')}</li>
        )}
        {recent.map((r, i) => (
          <li key={i} className="p-2.5 text-[11px]">
            <div className="flex items-center gap-2">
              <span className={`inline-block h-1.5 w-1.5 rounded-full ${SEVERITY_DOT[r.hit.severity] ?? 'bg-slate-400'}`} />
              <span className="font-mono font-medium">{r.hit.patternName}</span>
              <span className={`ml-auto px-1.5 py-0.5 rounded text-[10px] ${POLICY_BADGE[r.hit.policy]}`}>{r.hit.policy}</span>
            </div>
            <div className="mt-1 flex items-center gap-2 text-muted-foreground">
              <span className="font-mono">{r.source}</span>
              {r.path && <span className="font-mono truncate max-w-[14ch]" title={r.path}>{r.path}</span>}
              <span className="ml-auto">{new Date(r.timestamp).toLocaleTimeString()}</span>
            </div>
            <div className="mt-1 font-mono text-muted-foreground/70 truncate" title={r.hit.snippet}>{r.hit.snippet}</div>
          </li>
        ))}
      </ul>
    </Card>
  )
}
