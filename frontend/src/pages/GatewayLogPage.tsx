import { useEffect, useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { FileText, RefreshCw, AlertCircle, Trash2, BarChart3 } from 'lucide-react'
import { Button, Card, KpiCard } from '../components/ui'
import { useGatewayStore } from '../stores/gatewayStore'
import { useConfigStore } from '../stores/configStore'
import { makeLogSource, type LogSource, type GatewayLog, type GatewayLogStat } from '../lib/logSource'
import { formatLocal } from '../lib/formatTime'
import { SearchBar } from '../components/gateway/SearchBar'
import { Pagination } from '../components/gateway/Pagination'
import { DateRangePicker } from '../components/gateway/DateRangePicker'
import { ConfirmModal } from '../components/gateway/ConfirmModal'
import { SimpleBarChart } from '../components/gateway/SimpleBarChart'

const PER_PAGE = 50

const LOG_TYPE_LABELS: Record<number, string> = {
  1: 'Recharge',
  2: 'Consume',
  3: 'Manage',
  4: 'System',
}

type LogTab = 'usage' | 'draw' | 'task'

export function GatewayLogPage() {
  const { t } = useTranslation()
  const { status: serverStatus, adminToken } = useGatewayStore()
  const appMode = useConfigStore((s) => s.appMode)
  const isReseller = appMode === 'reseller'

  const [tab, setTab] = useState<LogTab>('usage')
  const [logs, setLogs] = useState<GatewayLog[]>([])
  const [page, setPage] = useState(0)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Filters
  const [usernameFilter, setUsernameFilter] = useState('')
  const [modelFilter, setModelFilter] = useState('')
  const [channelFilter, setChannelFilter] = useState('')
  const [tokenNameFilter, setTokenNameFilter] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')

  // Stats — newapi /api/log/stat returns a single aggregate object
  // {quota, rpm, tpm}, not a date series. The bar-chart layout is gone;
  // we render a 3-tile summary instead.
  const [showStats, setShowStats] = useState(false)
  const [stats, setStats] = useState<GatewayLogStat | null>(null)

  // Clear history
  const [showClearConfirm, setShowClearConfirm] = useState(false)

  const source: LogSource | null = useMemo(() => {
    if (isReseller) return makeLogSource({ mode: 'hub' })
    if (serverStatus?.running && adminToken) {
      return makeLogSource({ mode: 'local', baseURL: serverStatus.url, token: adminToken })
    }
    return null
  }, [isReseller, serverStatus?.running, serverStatus?.url, adminToken])

  const caps = source?.capabilities

  const typeForTab = (t: LogTab): number | undefined => {
    switch (t) {
      case 'usage': return 2
      case 'draw': return undefined // show all types
      case 'task': return 4
    }
  }

  const load = async (p = page) => {
    if (!source) return
    setLoading(true)
    setError(null)
    try {
      const channelId = channelFilter.trim() ? parseInt(channelFilter) : undefined
      const startTs = startDate ? Math.floor(new Date(startDate).getTime() / 1000) : undefined
      const endTs = endDate ? Math.floor(new Date(endDate + 'T23:59:59').getTime() / 1000) : undefined
      const logType = typeForTab(tab)

      const res = await source.list({
        page: p,
        perPage: PER_PAGE,
        username: usernameFilter.trim() || undefined,
        model: modelFilter.trim() || undefined,
        tokenName: tokenNameFilter.trim() || undefined,
        channelId: Number.isFinite(channelId as number) ? channelId : undefined,
        startTimestamp: startTs,
        endTimestamp: endTs,
        type: logType,
      })
      setLogs(res.items)
      if (res.total > 0 && res.total !== res.items.length) {
        setTotal(res.total)
      } else {
        setTotal(res.items.length === PER_PAGE ? (p + 2) * PER_PAGE : (p * PER_PAGE) + res.items.length)
      }
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  const loadStats = async () => {
    if (!source?.stats) return
    try {
      const now = Date.now()
      const start = Math.floor((now - 13 * 86400000) / 1000)
      const end = Math.floor(now / 1000)
      const data = await source.stats(start, end)
      setStats(data)
    } catch (e) {
      setError(String(e))
    }
  }

  useEffect(() => { load() }, [source, tab])

  const handlePageChange = (p: number) => {
    setPage(p)
    load(p)
  }

  const handleSearch = () => {
    setPage(0)
    load(0)
  }

  const handleDateChange = (start: string, end: string) => {
    setStartDate(start)
    setEndDate(end)
  }

  const handleClearHistory = async () => {
    if (!source?.clearHistory) return
    try {
      await source.clearHistory()
      setShowClearConfirm(false)
      setLogs([])
      setTotal(0)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleToggleStats = () => {
    if (!showStats) loadStats()
    setShowStats(!showStats)
  }

  const formatTime = (ts: number) =>
    ts > 0 ? formatLocal(ts * 1000) : '-'

  const tabs: { key: LogTab; label: string }[] = [
    { key: 'usage', label: t('gateway.usageLogs') },
    { key: 'draw', label: t('gateway.drawLogs') },
    { key: 'task', label: t('gateway.taskLogs') },
  ]

  if (!source) {
    return (
      <div className="flex flex-col h-full items-center justify-center text-muted-foreground gap-2">
        <AlertCircle className="h-8 w-8" />
        <p>
          {isReseller
            ? t('gateway.hubNotConfigured', '请先在「设置」中配置 Reseller Hub URL 与管理员 Token')
            : t('gateway.status.stopped')
          }
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold flex items-center gap-2">
          <FileText className="h-6 w-6 text-primary" />
          {t('gateway.logs')}
        </h2>
        <div className="flex gap-2">
          {caps?.stats && (
            <Button
              variant="secondary"
              size="sm"
              onClick={handleToggleStats}
              title={t('gateway.logStats')}
              icon={<BarChart3 className="h-4 w-4" />}
            />
          )}
          {caps?.clearHistory && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowClearConfirm(true)}
              className="border border-red-500/30 text-red-400 hover:bg-red-500/10"
              icon={<Trash2 className="h-4 w-4" />}
            >
              {t('gateway.clearHistory')}
            </Button>
          )}
          <Button
            variant="secondary"
            size="sm"
            onClick={() => load()}
            disabled={loading}
            loading={loading}
            icon={!loading ? <RefreshCw className="h-4 w-4" /> : undefined}
          />
        </div>
      </div>

      {/* Tab bar */}
      <div className="flex border-b border-border">
        {tabs.map((tb) => {
          const isActive = tab === tb.key
          return (
          <button
            key={tb.key}
            onClick={() => { setTab(tb.key); setPage(0) }}
            className={`px-4 py-2 -mb-px border-b-2 transition-all duration-150 ${
              isActive
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            <span className={isActive ? 'font-mono text-[11px] tracking-[0.12em]' : 'text-sm font-medium'}>
              {isActive ? `[ ${tb.label.toUpperCase()} ]` : tb.label}
            </span>
          </button>
          )
        })}
      </div>

      {/* Stats panel — 14-day aggregate (quota / rpm / tpm) */}
      {showStats && stats && (
        <Card variant="elevated" className="p-4">
          <h3 className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground mb-3">[ {t('gateway.logStats').toUpperCase()} (14 DAYS) ]</h3>
          <div className="grid grid-cols-3 gap-3">
            <KpiCard label="Quota" value={stats.quota} />
            <KpiCard label="RPM" value={stats.rpm} />
            <KpiCard label="TPM" value={stats.tpm} />
          </div>
        </Card>
      )}

      {/* Filters */}
      <div className="space-y-2">
        <SearchBar value={usernameFilter} onChange={setUsernameFilter} onSearch={handleSearch} placeholder={t('gateway.filterUser')} />
        <div className="flex flex-wrap gap-2">
          <input
            className="rounded border border-border bg-background px-3 py-1.5 text-sm w-32"
            placeholder={t('gateway.logModel')}
            value={modelFilter}
            onChange={(e) => setModelFilter(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          />
          <input
            className="rounded border border-border bg-background px-3 py-1.5 text-sm w-32"
            placeholder={t('gateway.logChannel')}
            value={channelFilter}
            onChange={(e) => setChannelFilter(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          />
          <input
            className="rounded border border-border bg-background px-3 py-1.5 text-sm w-32"
            placeholder={t('gateway.logTokenName')}
            value={tokenNameFilter}
            onChange={(e) => setTokenNameFilter(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          />
          <DateRangePicker start={startDate} end={endDate} onChange={handleDateChange} />
        </div>
      </div>

      {error && (
        <div className="text-sm text-red-400 bg-red-500/10 border border-red-500/30 rounded px-3 py-2 font-mono">▸ {error}</div>
      )}

      <Card variant="default" className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-card-recessed">
            <tr className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
              <th className="text-left px-4 py-2">[ {t('gateway.logTime').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.logUser').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.logModel').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.logTokens').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.logChannel').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.logTokenName').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.logType').toUpperCase()} ]</th>
            </tr>
          </thead>
          <tbody>
            {logs.length === 0 && (
              <tr>
                <td colSpan={7} className="text-center py-8 text-muted-foreground font-mono">
                  ▪ {loading ? t('status.loading') : t('gateway.noLogs')}
                </td>
              </tr>
            )}
            {logs.map((log) => (
              <tr key={log.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                <td className="px-4 py-2 text-xs text-muted-foreground font-mono tabular-nums">{formatTime(log.created_at)}</td>
                <td className="px-4 py-2">{log.username}</td>
                <td className="px-4 py-2 font-mono text-xs">{log.model_name || '-'}</td>
                <td className="px-4 py-2 font-mono tabular-nums">
                  {log.prompt_tokens + log.completion_tokens > 0
                    ? `${log.prompt_tokens}+${log.completion_tokens}`
                    : '-'}
                </td>
                <td className="px-4 py-2 text-muted-foreground text-xs font-mono">{log.channel_name || log.channel}</td>
                <td className="px-4 py-2 text-xs font-mono">{log.token_name || '-'}</td>
                <td className="px-4 py-2 text-xs">
                  <span className="font-mono text-[10px] uppercase tracking-[0.08em] rounded px-1.5 py-0.5 bg-card-recessed text-muted-foreground">
                    {LOG_TYPE_LABELS[log.type] ?? `Type ${log.type}`}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>

      <Pagination page={page} total={total} perPage={PER_PAGE} onPageChange={handlePageChange} />

      {/* Clear history confirm */}
      <ConfirmModal
        open={showClearConfirm}
        title={t('gateway.clearConfirmTitle')}
        desc={t('gateway.clearConfirm')}
        danger
        onConfirm={handleClearHistory}
        onCancel={() => setShowClearConfirm(false)}
      />
    </div>
  )
}
