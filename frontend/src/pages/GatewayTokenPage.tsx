import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Key, Plus, Trash2, RefreshCw, AlertCircle, Edit2, Copy, Check } from 'lucide-react'
import { useGatewayStore } from '../stores/gatewayStore'
import { createGatewayClient, type GatewayToken } from '../lib/gateway-api'
import { SearchBar } from '../components/gateway/SearchBar'
import { Pagination } from '../components/gateway/Pagination'
import { StatusBadge } from '../components/gateway/StatusBadge'
import { ConfirmModal } from '../components/gateway/ConfirmModal'

const PER_PAGE = 50

export function GatewayTokenPage() {
  const { t } = useTranslation()
  const { status: serverStatus, adminToken } = useGatewayStore()

  const [tokens, setTokens] = useState<GatewayToken[]>([])
  const [page, setPage] = useState(0)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [copiedId, setCopiedId] = useState<number | null>(null)

  // Modal state
  const [showModal, setShowModal] = useState(false)
  const [editing, setEditing] = useState<Partial<GatewayToken> | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)

  // Batch
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())

  const client = serverStatus?.running && adminToken
    ? createGatewayClient(serverStatus.url, adminToken)
    : null

  const load = async (p = page) => {
    if (!client) return
    setLoading(true)
    setError(null)
    try {
      const res = keyword.trim()
        ? await client.searchTokens(keyword.trim(), p, PER_PAGE)
        : await client.getTokens(p, PER_PAGE)
      setTokens(res.data ?? [])
      setTotal(res.data?.length === PER_PAGE ? (p + 2) * PER_PAGE : (p * PER_PAGE) + (res.data?.length ?? 0))
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [serverStatus?.running, adminToken])

  const handlePageChange = (p: number) => {
    setPage(p)
    load(p)
  }

  const handleSearch = () => {
    setPage(0)
    load(0)
  }

  const handleDelete = async () => {
    if (!client || confirmDelete === null) return
    try {
      await client.deleteToken(confirmDelete)
      setTokens((prev) => prev.filter((tk) => tk.id !== confirmDelete))
      setConfirmDelete(null)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleBatchDelete = async () => {
    if (!client || selectedIds.size === 0) return
    try {
      await client.batchDeleteTokens(Array.from(selectedIds))
      setSelectedIds(new Set())
      await load()
    } catch (e) {
      setError(String(e))
    }
  }

  const handleSave = async () => {
    if (!client || !editing) return
    try {
      if (editing.id) {
        await client.updateToken(editing as GatewayToken)
      } else {
        await client.createToken(editing)
      }
      setShowModal(false)
      setEditing(null)
      await load()
    } catch (e) {
      setError(String(e))
    }
  }

  const handleToggleStatus = async (tk: GatewayToken) => {
    if (!client) return
    try {
      await client.updateToken({ id: tk.id, status: tk.status === 1 ? 2 : 1 })
      setTokens((prev) => prev.map((t) => t.id === tk.id ? { ...t, status: t.status === 1 ? 2 : 1 } : t))
    } catch (e) {
      setError(String(e))
    }
  }

  const handleCopyKey = (tk: GatewayToken) => {
    navigator.clipboard.writeText(tk.key).then(() => {
      setCopiedId(tk.id)
      setTimeout(() => setCopiedId(null), 2000)
    })
  }

  const maskKey = (key: string) => key ? key.slice(0, 8) + '••••••••' : '-'
  const formatDate = (ts: number) =>
    ts > 0 ? new Date(ts * 1000).toLocaleDateString() : t('gateway.tokenNeverExpires')

  const toggleSelect = (id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const toggleSelectAll = () => {
    if (selectedIds.size === tokens.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(tokens.map((t) => t.id)))
    }
  }

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
          <Key className="h-6 w-6 text-yellow-400" />
          {t('gateway.tokens')}
        </h2>
        <div className="flex gap-2">
          <button
            onClick={() => load()}
            disabled={loading}
            className="flex items-center gap-1 px-3 py-1.5 rounded-md border border-border hover:bg-muted text-sm"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
          <button
            onClick={() => { setEditing({ status: 1, unlimited_quota: true }); setShowModal(true) }}
            className="flex items-center gap-1 px-3 py-1.5 rounded-md bg-indigo-600 hover:bg-indigo-500 text-white text-sm"
          >
            <Plus className="h-4 w-4" />
            {t('gateway.createToken')}
          </button>
        </div>
      </div>

      {/* Search + Batch bar */}
      <SearchBar value={keyword} onChange={setKeyword} onSearch={handleSearch} placeholder={t('gateway.search')}>
        {selectedIds.size > 0 && (
          <button
            onClick={handleBatchDelete}
            className="px-3 py-1.5 rounded-md bg-red-700 hover:bg-red-600 text-white text-sm"
          >
            <Trash2 className="h-4 w-4 inline mr-1" />
            {t('gateway.batchDelete')} ({selectedIds.size})
          </button>
        )}
      </SearchBar>

      {error && (
        <div className="text-sm text-red-400 bg-red-900/20 rounded px-3 py-2">{error}</div>
      )}

      <div className="rounded-lg border border-border overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-muted-foreground">
            <tr>
              <th className="px-3 py-2 w-8">
                <input type="checkbox" checked={selectedIds.size === tokens.length && tokens.length > 0} onChange={toggleSelectAll} />
              </th>
              <th className="text-left px-4 py-2">ID</th>
              <th className="text-left px-4 py-2">{t('gateway.tokenName')}</th>
              <th className="text-left px-4 py-2">{t('gateway.tokenKey')}</th>
              <th className="text-left px-4 py-2">{t('gateway.channelStatus')}</th>
              <th className="text-left px-4 py-2">{t('gateway.tokenQuota')}</th>
              <th className="text-left px-4 py-2">{t('gateway.tokenExpires')}</th>
              <th className="text-left px-4 py-2">{t('gateway.tokenGroup')}</th>
              <th className="text-right px-4 py-2">{t('gateway.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {tokens.length === 0 && (
              <tr>
                <td colSpan={9} className="text-center py-8 text-muted-foreground">
                  {loading ? t('status.loading') : t('gateway.noTokens')}
                </td>
              </tr>
            )}
            {tokens.map((tk) => (
              <tr key={tk.id} className="border-t border-border hover:bg-muted/30">
                <td className="px-3 py-2">
                  <input type="checkbox" checked={selectedIds.has(tk.id)} onChange={() => toggleSelect(tk.id)} />
                </td>
                <td className="px-4 py-2 text-muted-foreground">{tk.id}</td>
                <td className="px-4 py-2 font-medium">{tk.name}</td>
                <td className="px-4 py-2 font-mono text-xs">
                  <span className="mr-1">{maskKey(tk.key)}</span>
                  <button onClick={() => handleCopyKey(tk)} className="inline-flex p-0.5 hover:text-indigo-400" title={t('gateway.copyKey')}>
                    {copiedId === tk.id ? <Check className="h-3 w-3 text-green-400" /> : <Copy className="h-3 w-3" />}
                  </button>
                </td>
                <td className="px-4 py-2">
                  <button onClick={() => handleToggleStatus(tk)}>
                    <StatusBadge status={tk.status === 1 ? 'enabled' : 'disabled'} />
                  </button>
                </td>
                <td className="px-4 py-2">
                  {tk.unlimited_quota ? '∞' : `${tk.used_quota} / ${tk.quota}`}
                </td>
                <td className="px-4 py-2 text-muted-foreground text-xs">{formatDate(tk.expired_time)}</td>
                <td className="px-4 py-2 text-xs">{tk.group || '-'}</td>
                <td className="px-4 py-2 text-right">
                  <div className="flex justify-end gap-1">
                    <button
                      onClick={() => { setEditing(tk); setShowModal(true) }}
                      title={t('gateway.edit')}
                      className="p-1 hover:text-indigo-400"
                    >
                      <Edit2 className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => setConfirmDelete(tk.id)}
                      title={t('gateway.delete')}
                      className="p-1 hover:text-red-400"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} total={total} perPage={PER_PAGE} onPageChange={handlePageChange} />

      {/* Delete confirm */}
      <ConfirmModal
        open={confirmDelete !== null}
        title={t('gateway.deleteConfirmTitle')}
        desc={t('gateway.deleteConfirm')}
        danger
        onConfirm={handleDelete}
        onCancel={() => setConfirmDelete(null)}
      />

      {/* Create/Edit Modal */}
      {showModal && editing && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-card border border-border rounded-lg p-6 w-[28rem] max-h-[80vh] overflow-y-auto space-y-4">
            <h3 className="font-semibold">{editing.id ? t('gateway.editToken') : t('gateway.createToken')}</h3>
            <div className="space-y-3">
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.tokenName')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.name ?? ''}
                  onChange={(e) => setEditing({ ...editing, name: e.target.value })}
                />
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.tokenQuota')}</span>
                <input
                  type="number"
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.quota ?? 0}
                  onChange={(e) => setEditing({ ...editing, quota: parseInt(e.target.value) || 0 })}
                />
              </label>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={editing.unlimited_quota ?? false}
                  onChange={(e) => setEditing({ ...editing, unlimited_quota: e.target.checked })}
                />
                <span className="text-muted-foreground">{t('gateway.tokenUnlimited')}</span>
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.tokenExpires')}</span>
                <input
                  type="date"
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.expired_time && editing.expired_time > 0 ? new Date(editing.expired_time * 1000).toISOString().slice(0, 10) : ''}
                  onChange={(e) => setEditing({ ...editing, expired_time: e.target.value ? Math.floor(new Date(e.target.value).getTime() / 1000) : -1 })}
                />
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.tokenModelLimits')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono"
                  value={editing.model_limits ?? ''}
                  onChange={(e) => setEditing({ ...editing, model_limits: e.target.value })}
                  placeholder="model1,model2"
                />
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.tokenSubnet')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono"
                  value={editing.subnet ?? ''}
                  onChange={(e) => setEditing({ ...editing, subnet: e.target.value })}
                  placeholder="0.0.0.0/0"
                />
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.tokenGroup')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.group ?? ''}
                  onChange={(e) => setEditing({ ...editing, group: e.target.value })}
                />
              </label>
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => { setShowModal(false); setEditing(null) }}
                className="px-4 py-1.5 rounded border border-border text-sm hover:bg-muted"
              >
                {t('gateway.cancel')}
              </button>
              <button
                onClick={handleSave}
                className="px-4 py-1.5 rounded bg-indigo-600 hover:bg-indigo-500 text-white text-sm"
              >
                {t('gateway.save')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
