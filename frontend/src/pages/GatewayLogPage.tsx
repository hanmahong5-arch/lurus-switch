import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FileText, RefreshCw, AlertCircle, Trash2, BarChart3 } from 'lucide-react'
import { useGatewayStore } from '../stores/gatewayStore'
import { createGatewayClient, type GatewayLog, type GatewayLogStat } from '../lib/gateway-api'
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

  // Stats
  const [showStats, setShowStats] = useState(false)
  const [stats, setStats] = useState<GatewayLogStat[]>([])

  // Clear history
  const [showClearConfirm, setShowClearConfirm] = useState(false)

  const client = serverStatus?.running && adminToken
    ? createGatewayClient(serverStatus.url, adminToken)
    : null

  const typeForTab = (t: LogTab): number | undefined => {
    switch (t) {
      case 'usage': return 2
      case 'draw': return undefined // show all types
      case 'task': return 4
    }
  }

  const load = async (p = page) => {
    if (!client) return
    setLoading(true)
    setError(null)
    try {
      const params: Record<string, unknown> = {}
      if (usernameFilter.trim()) params.username = usernameFilter.trim()
      if (modelFilter.trim()) params.model = modelFilter.trim()
      if (tokenNameFilter.trim()) params.token_name = tokenNameFilter.trim()
      if (channelFilter.trim()) params.channel_id = parseInt(channelFilter) || undefined
      if (startDate) params.start_timestamp = Math.floor(new Date(startDate).getTime() / 1000)
      if (endDate) params.end_timestamp = Math.floor(new Date(endDate + 'T23:59:59').getTime() / 1000)
      const logType = typeForTab(tab)
      if (logType !== undefined) params.type = logType

      const res = await client.getLogs(p, PER_PAGE, params as Parameters<typeof client.getLogs>[2])
      setLogs(res.data ?? [])
      setTotal(res.data?.length === PER_PAGE ? (p + 2) * PER_PAGE : (p * PER_PAGE) + (res.data?.length ?? 0))
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  const loadStats = async () => {
    if (!client) return
    try {
      const now = Date.now()
      const start = Math.floor((now - 13 * 86400000) / 1000)
      const end = Math.floor(now / 1000)
      const res = await client.getLogStats(start, end)
      setStats(res.data ?? [])
    } catch (e) {
      setError(String(e))
    }
  }

  useEffect(() => { load() }, [serverStatus?.running, adminToken, tab])

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
    if (!client) return
    try {
      await client.clearLogs()
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
    ts > 0 ? new Date(ts * 1000).toLocaleString() : '-'

  const tabs: { key: LogTab; label: string }[] = [
    { key: 'usage', label: t('gateway.usageLogs') },
    { key: 'draw', label: t('gateway.drawLogs') },
    { key: 'task', label: t('gateway.taskLogs') },
  ]

  if (!serverStatus?.running) {
    return (
      <div className="flex flex-col h-full items-center justify-center text-muted-foreground gap-2">
        <AlertCircle className="h-8 w-8" />
        <p>{t('gateway.status.stopped')}</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold flex items-center gap-2">
          <FileText className="h-6 w-6 text-teal-400" />
          {t('gateway.logs')}
        </h2>
        <div className="flex gap-2">
          <button
            onClick={handleToggleStats}
            className="flex items-center gap-1 px-3 py-1.5 rounded-md border border-border hover:bg-muted text-sm"
            title={t('gateway.logStats')}
          >
            <BarChart3 className="h-4 w-4" />
          </button>
          <button
            onClick={() => setShowClearConfirm(true)}
            className="flex items-center gap-1 px-3 py-1.5 rounded-md border border-red-800 hover:bg-red-900/30 text-red-400 text-sm"
          >
            <Trash2 className="h-4 w-4" />
            {t('gateway.clearHistory')}
          </button>
          <button
            onClick={() => load()}
            disabled={loading}
            className="flex items-center gap-1 px-3 py-1.5 rounded-md border border-border hover:bg-muted text-sm"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      {/* Tab bar */}
      <div className="flex border-b border-border">
        {tabs.map((tb) => (
          <button
            key={tb.key}
            onClick={() => { setTab(tb.key); setPage(0) }}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              tab === tb.key
                ? 'border-indigo-500 text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {tb.label}
          </button>
        ))}
      </div>

      {/* Stats panel */}
      {showStats && stats.length > 0 && (
        <div className="rounded-lg border border-border bg-card p-4">
          <h3 className="text-sm font-medium mb-3">{t('gateway.logStats')} (14 days)</h3>
          <SimpleBarChart data={stats as unknown as Record<string, unknown>[]} labelKey="date" valueKey="request_count" height={100} />
        </div>
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
        <div className="text-sm text-red-400 bg-red-900/20 rounded px-3 py-2">{error}</div>
      )}

      <div className="rounded-lg border border-border overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-muted-foreground">
            <tr>
              <th className="text-left px-4 py-2">{t('gateway.logTime')}</th>
              <th className="text-left px-4 py-2">{t('gateway.logUser')}</th>
              <th className="text-left px-4 py-2">{t('gateway.logModel')}</th>
              <th className="text-left px-4 py-2">{t('gateway.logTokens')}</th>
              <th className="text-left px-4 py-2">{t('gateway.logChannel')}</th>
              <th className="text-left px-4 py-2">{t('gateway.logTokenName')}</th>
              <th className="text-left px-4 py-2">{t('gateway.logType')}</th>
            </tr>
          </thead>
          <tbody>
            {logs.length === 0 && (
              <tr>
                <td colSpan={7} className="text-center py-8 text-muted-foreground">
                  {loading ? t('status.loading') : t('gateway.noLogs')}
                </td>
              </tr>
            )}
            {logs.map((log) => (
              <tr key={log.id} className="border-t border-border hover:bg-muted/30">
                <td className="px-4 py-2 text-xs text-muted-foreground">{formatTime(log.created_at)}</td>
                <td className="px-4 py-2">{log.username}</td>
                <td className="px-4 py-2 font-mono text-xs">{log.model || '-'}</td>
                <td className="px-4 py-2">
                  {log.prompt_tokens + log.completion_tokens > 0
                    ? `${log.prompt_tokens}+${log.completion_tokens}`
                    : '-'}
                </td>
                <td className="px-4 py-2 text-muted-foreground text-xs">{log.channel_name || log.channel_id}</td>
                <td className="px-4 py-2 text-xs">{log.token_name || '-'}</td>
                <td className="px-4 py-2 text-xs">
                  <span className="rounded px-1.5 py-0.5 bg-muted/60">
                    {LOG_TYPE_LABELS[log.type] ?? `Type ${log.type}`}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

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
