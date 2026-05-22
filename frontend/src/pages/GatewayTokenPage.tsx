import { useEffect, useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Key, Plus, Trash2, RefreshCw, AlertCircle, Edit2, Copy, Check } from 'lucide-react'
import { Button, Card, Modal } from '../components/ui'
import { useGatewayStore } from '../stores/gatewayStore'
import { useConfigStore } from '../stores/configStore'
import { makeTokenSource, type TokenSource, type GatewayToken } from '../lib/tokenSource'
import { formatLocalDate } from '../lib/formatTime'
import { SearchBar } from '../components/gateway/SearchBar'
import { Pagination } from '../components/gateway/Pagination'
import { StatusBadge } from '../components/gateway/StatusBadge'
import { ConfirmModal } from '../components/gateway/ConfirmModal'

const PER_PAGE = 50

export function GatewayTokenPage() {
  const { t } = useTranslation()
  const { status: serverStatus, adminToken } = useGatewayStore()
  const appMode = useConfigStore((s) => s.appMode)
  const isReseller = appMode === 'reseller'

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

  const source: TokenSource | null = useMemo(() => {
    if (isReseller) return makeTokenSource({ mode: 'hub' })
    if (serverStatus?.running && adminToken) {
      return makeTokenSource({ mode: 'local', baseURL: serverStatus.url, token: adminToken })
    }
    return null
  }, [isReseller, serverStatus?.running, serverStatus?.url, adminToken])

  const load = async (p = page) => {
    if (!source) return
    setLoading(true)
    setError(null)
    try {
      const res = await source.list(p, PER_PAGE, { keyword: keyword.trim() })
      setTokens(res.items)
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

  useEffect(() => { load() }, [source])

  const handlePageChange = (p: number) => {
    setPage(p)
    load(p)
  }

  const handleSearch = () => {
    setPage(0)
    load(0)
  }

  const handleDelete = async () => {
    if (!source || confirmDelete === null) return
    try {
      await source.delete(confirmDelete)
      setTokens((prev) => prev.filter((tk) => tk.id !== confirmDelete))
      setConfirmDelete(null)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleBatchDelete = async () => {
    if (!source || selectedIds.size === 0) return
    try {
      await source.batchDelete(Array.from(selectedIds))
      setSelectedIds(new Set())
      await load()
    } catch (e) {
      setError(String(e))
    }
  }

  const handleSave = async () => {
    if (!source || !editing) return
    try {
      if (editing.id) {
        await source.update(editing as GatewayToken)
      } else {
        await source.create(editing)
      }
      setShowModal(false)
      setEditing(null)
      await load()
    } catch (e) {
      setError(String(e))
    }
  }

  const handleToggleStatus = async (tk: GatewayToken) => {
    if (!source) return
    try {
      await source.update({ id: tk.id, status: tk.status === 1 ? 2 : 1 })
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
    ts > 0 ? formatLocalDate(ts * 1000) : t('gateway.tokenNeverExpires')

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
          <Key className="h-6 w-6 text-primary" />
          {t('gateway.tokens')}
        </h2>
        <div className="flex gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => load()}
            disabled={loading}
            loading={loading}
            icon={!loading ? <RefreshCw className="h-4 w-4" /> : undefined}
          />
          <Button
            size="sm"
            onClick={() => { setEditing({ status: 1, unlimited_quota: true }); setShowModal(true) }}
            icon={<Plus className="h-4 w-4" />}
          >
            {t('gateway.createToken')}
          </Button>
        </div>
      </div>

      {/* Search + Batch bar */}
      <SearchBar value={keyword} onChange={setKeyword} onSearch={handleSearch} placeholder={t('gateway.search')}>
        {selectedIds.size > 0 && (
          <Button variant="danger" size="sm" onClick={handleBatchDelete} icon={<Trash2 className="h-4 w-4" />}>
            {t('gateway.batchDelete')} ({selectedIds.size})
          </Button>
        )}
      </SearchBar>

      {error && (
        <div className="text-sm text-red-400 bg-red-500/10 border border-red-500/30 rounded px-3 py-2 font-mono">▸ {error}</div>
      )}

      <Card variant="default" className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-card-recessed">
            <tr className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
              <th className="px-3 py-2 w-8">
                <input type="checkbox" checked={selectedIds.size === tokens.length && tokens.length > 0} onChange={toggleSelectAll} />
              </th>
              <th className="text-left px-4 py-2">[ ID ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.tokenName').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.tokenKey').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.channelStatus').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.tokenQuota').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.tokenExpires').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.tokenGroup').toUpperCase()} ]</th>
              <th className="text-right px-4 py-2">[ {t('gateway.actions').toUpperCase()} ]</th>
            </tr>
          </thead>
          <tbody>
            {tokens.length === 0 && (
              <tr>
                <td colSpan={9} className="text-center py-8 text-muted-foreground font-mono">
                  ▪ {loading ? t('status.loading') : t('gateway.noTokens')}
                </td>
              </tr>
            )}
            {tokens.map((tk) => (
              <tr key={tk.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                <td className="px-3 py-2">
                  <input type="checkbox" checked={selectedIds.has(tk.id)} onChange={() => toggleSelect(tk.id)} />
                </td>
                <td className="px-4 py-2 text-muted-foreground font-mono tabular-nums">{tk.id}</td>
                <td className="px-4 py-2 font-medium">{tk.name}</td>
                <td className="px-4 py-2 font-mono text-xs tabular-nums">
                  <span className="mr-1">{maskKey(tk.key)}</span>
                  <button onClick={() => handleCopyKey(tk)} className="inline-flex p-0.5 hover:text-primary transition-colors" title={t('gateway.copyKey')}>
                    {copiedId === tk.id ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
                  </button>
                </td>
                <td className="px-4 py-2">
                  <button onClick={() => handleToggleStatus(tk)} className="transition-opacity hover:opacity-80">
                    <StatusBadge status={tk.status === 1 ? 'enabled' : 'disabled'} />
                  </button>
                </td>
                <td className="px-4 py-2 font-mono tabular-nums">
                  {tk.unlimited_quota ? '∞' : `${tk.used_quota} / ${tk.quota}`}
                </td>
                <td className="px-4 py-2 text-muted-foreground text-xs font-mono tabular-nums">{formatDate(tk.expired_time)}</td>
                <td className="px-4 py-2 text-xs font-mono">{tk.group || '-'}</td>
                <td className="px-4 py-2 text-right">
                  <div className="flex justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => { setEditing(tk); setShowModal(true) }}
                      title={t('gateway.edit')}
                      icon={<Edit2 className="h-3.5 w-3.5" />}
                      className="hover:text-primary"
                    />
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setConfirmDelete(tk.id)}
                      title={t('gateway.delete')}
                      icon={<Trash2 className="h-3.5 w-3.5" />}
                      className="hover:text-red-400 hover:bg-red-500/10"
                    />
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>

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
      <Modal
        open={showModal && !!editing}
        onClose={() => { setShowModal(false); setEditing(null) }}
        title={editing?.id ? t('gateway.editToken') : t('gateway.createToken')}
        icon={editing?.id ? Edit2 : Plus}
        size="md"
        footer={
          <>
            <Button variant="secondary" size="sm" onClick={() => { setShowModal(false); setEditing(null) }}>
              {t('gateway.cancel')}
            </Button>
            <Button size="sm" onClick={handleSave}>
              {t('gateway.save')}
            </Button>
          </>
        }
      >
        {editing && (
          <div className="space-y-3">
            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.tokenName')}</span>
              <input
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                value={editing.name ?? ''}
                onChange={(e) => setEditing({ ...editing, name: e.target.value })}
              />
            </label>
            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.tokenQuota')}</span>
              <input
                type="number"
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono tabular-nums focus:outline-none focus:ring-1 focus:ring-primary"
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
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.tokenExpires')}</span>
              <input
                type="date"
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                value={editing.expired_time && editing.expired_time > 0 ? new Date(editing.expired_time * 1000).toISOString().slice(0, 10) : ''}
                onChange={(e) => setEditing({ ...editing, expired_time: e.target.value ? Math.floor(new Date(e.target.value).getTime() / 1000) : -1 })}
              />
            </label>
            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.tokenModelLimits')}</span>
              <input
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                value={editing.model_limits ?? ''}
                onChange={(e) => setEditing({ ...editing, model_limits: e.target.value })}
                placeholder="model1,model2"
              />
            </label>
            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.tokenSubnet')}</span>
              <input
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                value={editing.subnet ?? ''}
                onChange={(e) => setEditing({ ...editing, subnet: e.target.value })}
                placeholder="0.0.0.0/0"
              />
            </label>
            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.tokenGroup')}</span>
              <input
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                value={editing.group ?? ''}
                onChange={(e) => setEditing({ ...editing, group: e.target.value })}
              />
            </label>
          </div>
        )}
      </Modal>
    </div>
  )
}
