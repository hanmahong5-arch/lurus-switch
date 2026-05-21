import { useEffect, useState, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Gift, Plus, Trash2, RefreshCw, AlertCircle, Download } from 'lucide-react'
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
          <Gift className="h-6 w-6 text-pink-400" />
          {t('gateway.redemptions')}
        </h2>
        <button
          onClick={() => load(page)}
          disabled={loading}
          className="flex items-center gap-1 px-3 py-1.5 rounded-md border border-border hover:bg-muted text-sm"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
        </button>
      </div>

      {/* Error banner */}
      {error && (
        <div className="text-sm text-red-400 bg-red-900/20 rounded px-3 py-2">{error}</div>
      )}

      {/* Issued-batch banner — appears after Create returns codes. CSV export
          uses the in-memory list, since Hub never returns the plaintext key
          again on subsequent reads. */}
      {lastIssued.length > 0 && (
        <div className="flex items-center justify-between gap-3 px-3 py-2 rounded-lg border border-emerald-500/30 bg-emerald-950/20 text-sm">
          <div className="text-emerald-300">
            {t('gateway.issuedBanner', '刚刚生成 {{n}} 条激活码', { n: lastIssued.length })}
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={handleExportLastIssued}
              className="flex items-center gap-1 px-3 py-1.5 rounded-md bg-emerald-700 hover:bg-emerald-600 text-white text-xs"
            >
              <Download className="h-3.5 w-3.5" />
              {t('gateway.exportCSV', '导出 CSV')}
            </button>
            <button
              onClick={() => setLastIssued([])}
              className="px-2 py-1 rounded text-xs hover:bg-muted text-muted-foreground"
            >
              {t('common.dismiss', '关闭')}
            </button>
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
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-1 px-3 py-1.5 rounded-md bg-indigo-600 hover:bg-indigo-500 text-white text-sm"
        >
          <Plus className="h-4 w-4" />
          {t('gateway.createRedemption', 'Create')}
        </button>
        <button
          onClick={() => setShowDeleteInvalid(true)}
          className="flex items-center gap-1 px-3 py-1.5 rounded-md bg-red-600 hover:bg-red-500 text-white text-sm"
        >
          <Trash2 className="h-4 w-4" />
          {t('gateway.deleteInvalidRedemptions', 'Delete Invalid')}
        </button>
      </SearchBar>

      {/* Table */}
      <div className="rounded-lg border border-border overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-muted-foreground">
            <tr>
              <th className="text-left px-4 py-2">ID</th>
              <th className="text-left px-4 py-2">{t('gateway.redemptionName', 'Name')}</th>
              <th className="text-left px-4 py-2">{t('gateway.redemptionKey', 'Key')}</th>
              <th className="text-left px-4 py-2">{t('gateway.redemptionStatus', 'Status')}</th>
              <th className="text-left px-4 py-2">{t('gateway.redemptionQuota', 'Quota')}</th>
              <th className="text-left px-4 py-2">{t('gateway.redemptionCount', 'Count / Used')}</th>
              <th className="text-left px-4 py-2">{t('gateway.redemptionCreated', 'Created')}</th>
              <th className="text-right px-4 py-2">{t('gateway.actions', 'Actions')}</th>
            </tr>
          </thead>
          <tbody>
            {redemptions.length === 0 && (
              <tr>
                <td colSpan={8} className="text-center py-8 text-muted-foreground">
                  {loading ? t('status.loading') : t('gateway.noRedemptions', 'No redemptions')}
                </td>
              </tr>
            )}
            {redemptions.map((r) => (
              <tr key={r.id} className="border-t border-border hover:bg-muted/30">
                <td className="px-4 py-2 text-muted-foreground">{r.id}</td>
                <td className="px-4 py-2 font-medium">{r.name}</td>
                <td className="px-4 py-2 font-mono text-xs">{maskKey(r.key)}</td>
                <td className="px-4 py-2">
                  <StatusBadge status={STATUS_MAP[r.status] ?? 'disabled'} />
                </td>
                <td className="px-4 py-2">{r.quota}</td>
                <td className="px-4 py-2">
                  {r.count} / {r.used_count}
                </td>
                <td className="px-4 py-2 text-muted-foreground text-xs">
                  {formatTime(r.created_time)}
                </td>
                <td className="px-4 py-2 text-right">
                  <button
                    onClick={() => setConfirmDelete(r.id)}
                    title="Delete"
                    className="p-1 hover:text-red-400"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      <Pagination
        page={page}
        total={total}
        perPage={PER_PAGE}
        onPageChange={handlePageChange}
      />

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-card border border-border rounded-lg p-6 w-96 space-y-4">
            <h3 className="font-semibold">
              {t('gateway.createRedemption', 'Create Redemption')}
            </h3>
            <div className="space-y-3">
              <label className="block text-sm">
                <span className="text-muted-foreground">
                  {t('gateway.redemptionName', 'Name')}
                </span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={formName}
                  onChange={(e) => setFormName(e.target.value)}
                  placeholder="redemption-batch-01"
                  autoFocus
                />
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground">
                  {t('gateway.redemptionQuota', 'Quota')}
                </span>
                <input
                  type="number"
                  min={1}
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={formQuota}
                  onChange={(e) => setFormQuota(Number(e.target.value) || 1)}
                />
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground">
                  {t('gateway.redemptionCount', 'Count')}
                </span>
                <input
                  type="number"
                  min={1}
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={formCount}
                  onChange={(e) => setFormCount(Number(e.target.value) || 1)}
                />
              </label>
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => { setShowCreateModal(false); resetForm() }}
                className="px-4 py-1.5 rounded border border-border text-sm hover:bg-muted"
              >
                {t('settings.data.cancel')}
              </button>
              <button
                onClick={handleCreate}
                disabled={!formName.trim()}
                className="px-4 py-1.5 rounded bg-indigo-600 hover:bg-indigo-500 text-white text-sm disabled:opacity-50"
              >
                {t('settings.save')}
              </button>
            </div>
          </div>
        </div>
      )}

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
