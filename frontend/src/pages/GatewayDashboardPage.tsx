import { useEffect, useState, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  BarChart3,
  Users,
  Layers,
  Key,
  Activity,
  Cpu,
  HardDrive,
  Clock,
  AlertCircle,
  RefreshCw,
} from 'lucide-react'
import { useGatewayStore } from '../stores/gatewayStore'
import { useConfigStore } from '../stores/configStore'
import { Button, Card, KpiCard } from '../components/ui'
import {
  makeDashboardSource,
  type DashboardSource,
  type GatewayDashboardData,
  type GatewayQuotaDate,
  type GatewayPerformanceStats,
} from '../lib/dashboardSource'
import { SimpleBarChart } from '../components/gateway/SimpleBarChart'

const BYTES_PER_MB = 1024 * 1024
const SECONDS_PER_HOUR = 3600
const DAYS_RANGE = 14
const MS_PER_DAY = 86400000

function formatMemoryMB(bytes: number): string {
  return `${(bytes / BYTES_PER_MB).toFixed(1)} MB`
}

function formatUptimeHours(seconds: number): string {
  return `${(seconds / SECONDS_PER_HOUR).toFixed(1)} h`
}

function getDateRange(): { start: string; end: string } {
  const end = new Date().toISOString().slice(0, 10)
  const start = new Date(Date.now() - (DAYS_RANGE - 1) * MS_PER_DAY).toISOString().slice(0, 10)
  return { start, end }
}

export function GatewayDashboardPage() {
  const { t } = useTranslation()
  const { status: serverStatus, adminToken } = useGatewayStore()
  const appMode = useConfigStore((s) => s.appMode)
  const isReseller = appMode === 'reseller'

  const [dashboardData, setDashboardData] = useState<GatewayDashboardData | null>(null)
  const [quotaDates, setQuotaDates] = useState<GatewayQuotaDate[]>([])
  const [performanceStats, setPerformanceStats] = useState<GatewayPerformanceStats | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const source: DashboardSource | null = useMemo(() => {
    if (isReseller) return makeDashboardSource({ mode: 'hub' })
    if (serverStatus?.running && adminToken) {
      return makeDashboardSource({ mode: 'local', baseURL: serverStatus.url, token: adminToken })
    }
    return null
  }, [isReseller, serverStatus?.running, serverStatus?.url, adminToken])

  const load = useCallback(async () => {
    if (!source) return
    setLoading(true)
    setError(null)
    try {
      const { start, end } = getDateRange()
      const bundle = await source.fetch(start, end)
      setDashboardData(bundle.summary)
      setQuotaDates(bundle.quota)
      setPerformanceStats(bundle.performance)
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }, [source])

  useEffect(() => {
    load()
  }, [load])

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

  const statCards = [
    {
      icon: Users,
      color: 'text-blue-400',
      label: t('gateway.dashboard.userCount', 'Users'),
      value: dashboardData?.user_count ?? 0,
    },
    {
      icon: Layers,
      color: 'text-emerald-400',
      label: t('gateway.dashboard.channelCount', 'Channels'),
      value: dashboardData?.channel_count ?? 0,
    },
    {
      icon: Key,
      color: 'text-amber-400',
      label: t('gateway.dashboard.tokenCount', 'Tokens'),
      value: dashboardData?.token_count ?? 0,
    },
    {
      icon: Activity,
      color: 'text-rose-400',
      label: t('gateway.dashboard.todayRequest', 'Today Requests'),
      value: dashboardData?.today_request ?? 0,
    },
  ]

  const perfRows = performanceStats
    ? [
        {
          icon: Cpu,
          label: t('gateway.dashboard.goroutines', 'Goroutines'),
          value: String(performanceStats.goroutines),
        },
        {
          icon: HardDrive,
          label: t('gateway.dashboard.memoryAlloc', 'Memory'),
          value: formatMemoryMB(performanceStats.memory_alloc),
        },
        {
          icon: Clock,
          label: t('gateway.dashboard.uptime', 'Uptime'),
          value: formatUptimeHours(performanceStats.uptime),
        },
        {
          icon: Activity,
          label: t('gateway.dashboard.requestsTotal', 'Total Requests'),
          value: String(performanceStats.requests_total),
        },
        {
          icon: BarChart3,
          label: t('gateway.dashboard.requestsPerSec', 'Req/s'),
          value: performanceStats.requests_per_sec.toFixed(2),
        },
      ]
    : []

  return (
    <div className="flex flex-col h-full overflow-y-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold flex items-center gap-2">
          <BarChart3 className="h-6 w-6 text-primary" />
          {t('gateway.dashboard', 'Dashboard')}
        </h2>
        <Button
          variant="secondary"
          size="sm"
          onClick={load}
          disabled={loading}
          loading={loading}
          icon={!loading ? <RefreshCw className="h-4 w-4" /> : undefined}
        />
      </div>

      {/* Error */}
      {error && (
        <Card variant="default" className="text-sm text-red-400 bg-red-500/10 border-red-500/30 px-3 py-2 font-mono">▸ {error}</Card>
      )}

      {/* Stat Cards 2x2 */}
      <div className="grid grid-cols-2 gap-4">
        {statCards.map((card) => (
          <KpiCard
            key={card.label}
            icon={card.icon}
            label={card.label}
            value={card.value.toLocaleString()}
          />
        ))}
      </div>

      {/* Quota Trend Chart */}
      <Card variant="elevated" className="p-5 space-y-3">
        <h3 className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground">
          [ {t('gateway.dashboard.quotaTrend', 'Quota Trend (14 days)').toUpperCase()} ]
        </h3>
        <SimpleBarChart data={quotaDates as unknown as Record<string, unknown>[]} labelKey="date" valueKey="quota" />
      </Card>

      {/* Performance Panel */}
      {performanceStats && (
        <Card variant="elevated" className="p-5 space-y-3">
          <h3 className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground">
            [ {t('gateway.dashboard.performance', 'Performance').toUpperCase()} ]
          </h3>
          <div className="space-y-2">
            {perfRows.map((row) => {
              const Icon = row.icon
              return (
                <div
                  key={row.label}
                  className="flex items-center justify-between text-sm py-1.5 border-b border-border last:border-0"
                >
                  <span className="flex items-center gap-2 text-muted-foreground">
                    <Icon className="h-4 w-4" />
                    {row.label}
                  </span>
                  <span className="font-mono tabular-nums">{row.value}</span>
                </div>
              )
            })}
          </div>
        </Card>
      )}
    </div>
  )
}
