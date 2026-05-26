import { useEffect, useState, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Gift, Plus, Trash2, RefreshCw, AlertCircle, Download, TrendingUp, CheckCircle2, Percent, Timer } from 'lucide-react'
import { Button, Card, KpiCard, Modal } from '../components/ui'
import { useGatewayStore } from '../stores/gatewayStore'
import { useConfigStore } from '../stores/configStore'
import {
  makeRedemptionSource,
  downloadRedemptionsCSV,
  type RedemptionSource,
  type GatewayRedemption,
} from '../lib/redemptionSource'
import { formatLocal } from '../lib/formatTime'
import { SearchBar } from '../components/gateway/SearchBar'
import { Pagination } from '../components/gateway/Pagination'
import { StatusBadge } from '../components/gateway/StatusBadge'
import { ConfirmModal } from '../components/gateway/ConfirmModal'

const PER_PAGE = 20

const STATUS_MAP: Record<number, 'enabled' | 'disabled' | 'used'> = {
  1: 'enabled',
  2: 'disabled',
  3: 'used',
}

// Wave 5 W5.2 — page-scoped funnel summary. We don't paginate the full set
// (could be 10K+ for active resellers) so the KPI strip is explicitly
// "current page" stats. The caption underneath says so to keep the user
// from mistaking page-level numbers for fleet-wide ones.
const STATUS_USED = 3
const DAY_MS = 86400000

interface RedemptionFunnel {
  total: number
  used: number
  ratePct: number  // 0..100
  avgDaysToRedeem: number | null
}

function summarize(rows: GatewayRedemption[]): RedemptionFunnel {
  if (rows.length === 0) {
    return { total: 0, used: 0, ratePct: 0, avgDaysToRedeem: null }
  }
  const used = rows.filter((r) => r.status === STATUS_USED).length
  const ratePct = (used / rows.length) * 100
  const usedRows = rows.filter((r) => r.status === STATUS_USED && r.redeemed_time > 0 && r.created_time > 0)
  const avg =
    usedRows.length === 0
      ? null
      : usedRows.reduce((sum, r) => sum + (r.redeemed_time - r.created_time) * 1000, 0) /
        usedRows.length /
        DAY_MS
  return { total: rows.length, used, ratePct, avgDaysToRedeem: avg }
}

export function GatewayRedemptionPage() {
  const { t } = useTranslation()
  const { status: serverStatus, adminToken } = useGatewayStore()
  const appMode = useConfigStore((s) => s.appMode)
  const isReseller = appMode === 'reseller'

  const [redemptions, setRedemptions] = useState<GatewayRedemption[]>([])
  const [page, setPage] = useState(0)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)
  const [showDeleteInvalid, setShowDeleteInvalid] = useState(false)

  // Last issued batch \u2014 surfaces a banner with CSV export after creation.
  const [lastIssued, setLastIssued] = useState<GatewayRedemption[]>([])

  // Create modal form state
  const [formName, setFormName] = useState('')
  const [formQuota, setFormQuota] = useState(100)
  const [formCount, setFormCount] = useState(1)

  const source: RedemptionSource | null = useMemo(() => {
    if (isReseller) return makeRedemptionSource({ mode: 'hub' })
    if (serverStatus?.running && adminToken) {
      return makeRedemptionSource({ mode: 'local', baseURL: serverStatus.url, token: adminToken })
    }
    return null
  }, [isReseller, serverStatus?.running, serverStatus?.url, adminToken])

  const load = useCallback(async (p = page) => {
    if (!source) return
    setLoading(true)
    setError(null)
    try {
      const res = await source.list(p, PER_PAGE, { keyword: keyword.trim() })
      setRedemptions(res.items)
      if (res.total > 0 && res.total !== res.items.length) {
        setTotal(res.total)
      } else {
        setTotal(res.items.length < PER_PAGE ? p * PER_PAGE + res.items.length : (p + 2) * PER_PAGE)
      }
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }, [source, keyword, page])

  useEffect(() => { load(page) }, [source, page])

  const handleSearch = () => {
    setPage(0)
    load(0)
  }

  const handlePageChange = (newPage: number) => {
    setPage(newPage)
  }

  const handleCreate = async () => {
    if (!source || !formName.trim()) return
    try {
      const created = await source.create({
        name: formName.trim(),
        quota: formQuota,
        count: formCount,
      })
      setShowCreateModal(false)
      resetForm()
      setLastIssued(created)
      await load(0)
      setPage(0)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleDelete = async () => {
    if (!source || confirmDelete === null) return
    try {
      await source.delete(confirmDelete)
      setRedemptions((prev) => prev.filter((r) => r.id !== confirmDelete))
      setTotal((prev) => Math.max(0, prev - 1))
      setConfirmDelete(null)
    } catch (e) {
      setError(String(e))
      setConfirmDelete(null)
    }
  }

  const handleDeleteInvalid = async () => {
    if (!source) return
    try {
      await source.deleteInvalid()
      setShowDeleteInvalid(false)
      setPage(0)
      await load(0)
    } catch (e) {
      setError(String(e))
      setShowDeleteInvalid(false)
    }
  }

  const handleExportLastIssued = () => {
    if (lastIssued.length === 0) return
    const stamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
    downloadRedemptionsCSV(lastIssued, `redemptions-${stamp}.csv`)
  }

  const resetForm = () => {
    setFormName('')
    setFormQuota(100)
    setFormCount(1)
  }

  const maskKey = (key: string) =>
    key ? key.slice(0, 6) + '\u2022\u2022\u2022\u2022' : '-'

  const formatTime = (ts: number) =>
    ts > 0 ? formatLocal(ts * 1000) : '-'

  if (!source) {
    return (
      <div className="flex flex-col h-full items-center justify-center text-muted-foreground gap-2">
        <AlertCircle className="h-8 w-8" />
        <p>
          {isReseller
            ? t('gateway.hubNotConfigured', '\u8bf7\u5148\u5728\u300c\u8bbe\u7f6e\u300d\u4e2d\u914d\u7f6e Reseller Hub URL \u4e0e\u7ba1\u7406\u5458 Token')
            : t('gateway.status.stopped')
          }
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold flex items-center gap-2">
          <Gift className="h-6 w-6 text-primary" />
          {t('gateway.redemptions')}
        </h2>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => load(page)}
          disabled={loading}
          loading={loading}
          icon={!loading ? <RefreshCw className="h-4 w-4" /> : undefined}
        />
      </div>

      {/* Error banner */}
      {error && (
        <div className="text-sm text-red-400 bg-red-500/10 border border-red-500/30 rounded px-3 py-2 font-mono">▸ {error}</div>
      )}

      {/* Funnel KPI strip — page-scoped (Wave 5 W5.2) */}
      <FunnelStrip rows={redemptions} total={total} />

      {/* Issued-batch banner — appears after Create returns codes. CSV export
          uses the in-memory list, since Hub never returns the plaintext key
          again on subsequent reads. */}
      {lastIssued.length > 0 && (
        <div className="flex items-center justify-between gap-3 px-3 py-2 rounded-lg border border-emerald-500/30 bg-emerald-500/10 text-sm">
          <div className="text-emerald-400 font-mono">
            ▸ {t('gateway.issuedBanner', '刚刚生成 {{n}} 条激活码', { n: lastIssued.length })}
          </div>
          <div className="flex items-center gap-2">
            <Button
              size="sm"
              onClick={handleExportLastIssued}
              className="bg-emerald-600 hover:bg-emerald-500 ring-emerald-500/40"
              icon={<Download className="h-3.5 w-3.5" />}
            >
              {t('gateway.exportCSV', '导出 CSV')}
            </Button>
            <Button variant="ghost" size="sm" onClick={() => setLastIssued([])}>
              {t('common.dismiss', '关闭')}
            </Button>
          </div>
        </div>
      )}

      {/* Action bar */}
      <SearchBar
        value={keyword}
        onChange={setKeyword}
        onSearch={handleSearch}
        placeholder={t('gateway.searchRedemptions', 'Search redemptions...')}
      >
        <Button
          size="sm"
          onClick={() => setShowCreateModal(true)}
          icon={<Plus className="h-4 w-4" />}
        >
          {t('gateway.createRedemption', 'Create')}
        </Button>
        <Button
          variant="danger"
          size="sm"
          onClick={() => setShowDeleteInvalid(true)}
          icon={<Trash2 className="h-4 w-4" />}
        >
          {t('gateway.deleteInvalidRedemptions', 'Delete Invalid')}
        </Button>
      </SearchBar>

      {/* Table */}
      <Card variant="default" className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-card-recessed">
            <tr className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
              <th className="text-left px-4 py-2">[ ID ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.redemptionName', 'Name').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.redemptionKey', 'Key').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.redemptionStatus', 'Status').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.redemptionQuota', 'Quota').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.redemptionCount', 'Count / Used').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.redemptionCreated', 'Created').toUpperCase()} ]</th>
              <th className="text-right px-4 py-2">[ {t('gateway.actions', 'Actions').toUpperCase()} ]</th>
            </tr>
          </thead>
          <tbody>
            {redemptions.length === 0 && (
              <tr>
                <td colSpan={8} className="text-center py-8 text-muted-foreground font-mono">
                  ▪ {loading ? t('status.loading') : t('gateway.noRedemptions', 'No redemptions')}
                </td>
              </tr>
            )}
            {redemptions.map((r) => (
              <tr key={r.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                <td className="px-4 py-2 text-muted-foreground font-mono tabular-nums">{r.id}</td>
                <td className="px-4 py-2 font-medium">{r.name}</td>
                <td className="px-4 py-2 font-mono text-xs tabular-nums">{maskKey(r.key)}</td>
                <td className="px-4 py-2">
                  <StatusBadge status={STATUS_MAP[r.status] ?? 'disabled'} />
                </td>
                <td className="px-4 py-2 font-mono tabular-nums">{r.quota}</td>
                <td className="px-4 py-2 font-mono tabular-nums">
                  {r.count} / {r.used_count}
                </td>
                <td className="px-4 py-2 text-muted-foreground text-xs font-mono tabular-nums">
                  {formatTime(r.created_time)}
                </td>
                <td className="px-4 py-2 text-right">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setConfirmDelete(r.id)}
                    title="Delete"
                    icon={<Trash2 className="h-3.5 w-3.5" />}
                    className="hover:text-red-400 hover:bg-red-500/10"
                  />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>

      {/* Pagination */}
      <Pagination
        page={page}
        total={total}
        perPage={PER_PAGE}
        onPageChange={handlePageChange}
      />

      {/* Create Modal */}
      <Modal
        open={showCreateModal}
        onClose={() => { setShowCreateModal(false); resetForm() }}
        title={t('gateway.createRedemption', 'Create Redemption')}
        icon={Plus}
        size="md"
        footer={
          <>
            <Button variant="secondary" size="sm" onClick={() => { setShowCreateModal(false); resetForm() }}>
              {t('settings.data.cancel')}
            </Button>
            <Button size="sm" onClick={handleCreate} disabled={!formName.trim()}>
              {t('settings.save')}
            </Button>
          </>
        }
      >
        <div className="space-y-3">
          <label className="block text-sm">
            <span className="text-muted-foreground font-mono text-xs">
              {t('gateway.redemptionName', 'Name')}
            </span>
            <input
              className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
              value={formName}
              onChange={(e) => setFormName(e.target.value)}
              placeholder="redemption-batch-01"
              autoFocus
            />
          </label>
          <label className="block text-sm">
            <span className="text-muted-foreground font-mono text-xs">
              {t('gateway.redemptionQuota', 'Quota')}
            </span>
            <input
              type="number"
              min={1}
              className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono tabular-nums focus:outline-none focus:ring-1 focus:ring-primary"
              value={formQuota}
              onChange={(e) => setFormQuota(Number(e.target.value) || 1)}
            />
          </label>
          <label className="block text-sm">
            <span className="text-muted-foreground font-mono text-xs">
              {t('gateway.redemptionCount', 'Count')}
            </span>
            <input
              type="number"
              min={1}
              className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono tabular-nums focus:outline-none focus:ring-1 focus:ring-primary"
              value={formCount}
              onChange={(e) => setFormCount(Number(e.target.value) || 1)}
            />
          </label>
        </div>
      </Modal>

      {/* Delete single confirmation */}
      <ConfirmModal
        open={confirmDelete !== null}
        title={t('gateway.deleteRedemption', 'Delete Redemption')}
        desc={t('gateway.deleteRedemptionDesc', 'Are you sure you want to delete this redemption code? This action cannot be undone.')}
        danger
        onConfirm={handleDelete}
        onCancel={() => setConfirmDelete(null)}
      />

      {/* Delete invalid confirmation */}
      <ConfirmModal
        open={showDeleteInvalid}
        title={t('gateway.deleteInvalidRedemptions', 'Delete Invalid Redemptions')}
        desc={t('gateway.deleteInvalidRedemptionsDesc', 'This will permanently delete all used and disabled redemption codes. Continue?')}
        danger
        onConfirm={handleDeleteInvalid}
        onCancel={() => setShowDeleteInvalid(false)}
      />
    </div>
  )
}

interface FunnelStripProps {
  rows: GatewayRedemption[]
  total: number
}

export function FunnelStrip({ rows, total }: FunnelStripProps) {
  const { t } = useTranslation()
  const funnel = useMemo(() => summarize(rows), [rows])

  // Display the Hub-reported total when present (more honest about the
  // fleet size) but fall back to page rows when total isn't populated.
  const issued = total > 0 ? total : funnel.total
  const avgLabel = funnel.avgDaysToRedeem == null
    ? '—'
    : `${funnel.avgDaysToRedeem.toFixed(1)} ${t('gateway.redemption.kpi.days', '天')}`

  return (
    <div>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <KpiCard
          label={t('gateway.redemption.kpi.issued', '生成总数')}
          value={issued.toLocaleString()}
          icon={TrendingUp}
        />
        <KpiCard
          label={t('gateway.redemption.kpi.used', '本页已兑换')}
          value={funnel.used.toLocaleString()}
          icon={CheckCircle2}
        />
        <KpiCard
          label={t('gateway.redemption.kpi.rate', '本页兑换率')}
          value={`${funnel.ratePct.toFixed(0)}%`}
          icon={Percent}
        />
        <KpiCard
          label={t('gateway.redemption.kpi.avgDays', '本页平均兑换天数')}
          value={avgLabel}
          icon={Timer}
        />
      </div>
      <p className="mt-1 text-[10px] text-muted-foreground/70 font-mono uppercase tracking-wider">
        {t('gateway.redemption.kpi.scopeHint', '* 兑换率与天数基于当前页统计 (issued 来自全量)')}
      </p>
    </div>
  )
}
