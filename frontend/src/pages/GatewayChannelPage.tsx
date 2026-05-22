import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Button, Card, Modal } from '../components/ui'
import {
  Layers, Plus, Trash2, TestTube, RefreshCw, AlertCircle,
  Edit2, Copy, Download, ChevronDown, Check, Tag,
} from 'lucide-react'
import { useGatewayStore } from '../stores/gatewayStore'
import { useConfigStore } from '../stores/configStore'
import { CHANNEL_TYPES } from '../lib/gateway-api'
import { makeChannelSource, type ChannelSource, type GatewayChannel } from '../lib/channelSource'
import { SearchBar } from '../components/gateway/SearchBar'
import { Pagination } from '../components/gateway/Pagination'
import { StatusBadge } from '../components/gateway/StatusBadge'
import { ConfirmModal } from '../components/gateway/ConfirmModal'

const PER_PAGE = 50

export function GatewayChannelPage() {
  const { t } = useTranslation()
  const { status: serverStatus, adminToken } = useGatewayStore()
  const appMode = useConfigStore((s) => s.appMode)
  const isReseller = appMode === 'reseller'

  // --- Data state ---
  const [channels, setChannels] = useState<GatewayChannel[]>([])
  const [page, setPage] = useState(0)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [typeFilter, setTypeFilter] = useState<number | null>(null)
  const [tagFilter, setTagFilter] = useState('')

  // --- Selection ---
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())

  // --- UI state ---
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [testResults, setTestResults] = useState<Record<number, string>>({})
  const [showModal, setShowModal] = useState(false)
  const [editing, setEditing] = useState<Partial<GatewayChannel> | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<null | number | 'batch'>(null)

  // --- Tag operations ---
  const [showTagOps, setShowTagOps] = useState(false)
  const [tagOpInput, setTagOpInput] = useState('')
  const [tagOpOldTag, setTagOpOldTag] = useState('')
  const [tagOpNewTag, setTagOpNewTag] = useState('')

  // --- Batch set-tag popover ---
  const [showBatchTagInput, setShowBatchTagInput] = useState(false)
  const [batchTagValue, setBatchTagValue] = useState('')

  // Pick the data source based on mode. Reseller talks to a remote Hub via
  // Wails bindings; Personal talks to the in-process gateway directly. The
  // memo'd source is stable across renders so callbacks don't churn.
  const source: ChannelSource | null = useMemo(() => {
    if (isReseller) return makeChannelSource({ mode: 'hub' })
    if (serverStatus?.running && adminToken) {
      return makeChannelSource({ mode: 'local', baseURL: serverStatus.url, token: adminToken })
    }
    return null
  }, [isReseller, serverStatus?.running, serverStatus?.url, adminToken])

  const caps = source?.capabilities

  const load = useCallback(async (p = page) => {
    if (!source) return
    setLoading(true)
    setError(null)
    try {
      const res = await source.list(p, PER_PAGE, { keyword: keyword.trim() })
      let data = res.items

      if (typeFilter !== null) {
        data = data.filter((ch) => ch.type === typeFilter)
      }
      if (tagFilter.trim()) {
        const tf = tagFilter.trim().toLowerCase()
        data = data.filter((ch) => (ch.tag ?? '').toLowerCase().includes(tf))
      }

      setChannels(data)
      // Hub returns an authoritative total; local source falls back to length
      // and we estimate the next-page existence as before.
      if (res.total > 0 && res.total !== data.length) {
        setTotal(res.total)
      } else {
        setTotal(data.length < PER_PAGE && p === 0
          ? data.length
          : Math.max((p + 1) * PER_PAGE + (data.length === PER_PAGE ? PER_PAGE : 0), data.length)
        )
      }
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }, [source, keyword, typeFilter, tagFilter, page])

  useEffect(() => {
    load(page)
  }, [source, page])

  const handleSearch = useCallback(() => {
    setPage(0)
    setSelectedIds(new Set())
    load(0)
  }, [load])

  // ==================== Single row actions ====================

  const handleDelete = async (id: number) => {
    if (!source) return
    try {
      await source.delete(id)
      setChannels((prev) => prev.filter((c) => c.id !== id))
      setSelectedIds((prev) => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
      setTotal((prev) => Math.max(0, prev - 1))
    } catch (e) {
      setError(String(e))
    }
  }

  const handleTest = async (id: number) => {
    if (!source) return
    setTestResults((prev) => ({ ...prev, [id]: 'testing...' }))
    try {
      const msg = await source.test(id)
      setTestResults((prev) => ({ ...prev, [id]: msg }))
    } catch (e) {
      setTestResults((prev) => ({ ...prev, [id]: String(e) }))
    }
  }

  const handleCopy = async (id: number) => {
    if (!source?.copy) return
    try {
      await source.copy(id)
      await load(page)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleFetchModels = async (id: number) => {
    if (!source?.fetchModels) return
    try {
      const models = await source.fetchModels(id)
      if (models.length > 0) {
        setTestResults((prev) => ({
          ...prev,
          [id]: `Fetched ${models.length} models: ${models.slice(0, 5).join(', ')}${models.length > 5 ? '...' : ''}`,
        }))
      } else {
        setTestResults((prev) => ({ ...prev, [id]: 'No models found' }))
      }
    } catch (e) {
      setTestResults((prev) => ({ ...prev, [id]: String(e) }))
    }
  }

  const handleToggleStatus = async (ch: GatewayChannel) => {
    if (!source) return
    const newStatus = ch.status === 1 ? 2 : 1
    try {
      await source.update({ id: ch.id, status: newStatus })
      setChannels((prev) =>
        prev.map((c) => (c.id === ch.id ? { ...c, status: newStatus } : c))
      )
    } catch (e) {
      setError(String(e))
    }
  }

  // ==================== Batch operations ====================

  const handleBatchDelete = async () => {
    if (!source || selectedIds.size === 0) return
    try {
      await source.batchDelete(Array.from(selectedIds))
      setSelectedIds(new Set())
      await load(page)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleBatchEnable = async () => {
    if (!source?.batchEnable || selectedIds.size === 0) return
    try {
      await source.batchEnable(Array.from(selectedIds))
      setChannels((prev) =>
        prev.map((c) => (selectedIds.has(c.id) ? { ...c, status: 1 } : c))
      )
    } catch (e) {
      setError(String(e))
    }
  }

  const handleBatchDisable = async () => {
    if (!source?.batchDisable || selectedIds.size === 0) return
    try {
      await source.batchDisable(Array.from(selectedIds))
      setChannels((prev) =>
        prev.map((c) => (selectedIds.has(c.id) ? { ...c, status: 2 } : c))
      )
    } catch (e) {
      setError(String(e))
    }
  }

  const handleBatchSetTag = async () => {
    if (!source?.batchSetTag || selectedIds.size === 0 || !batchTagValue.trim()) return
    try {
      await source.batchSetTag(Array.from(selectedIds), batchTagValue.trim())
      setChannels((prev) =>
        prev.map((c) => (selectedIds.has(c.id) ? { ...c, tag: batchTagValue.trim() } : c))
      )
      setShowBatchTagInput(false)
      setBatchTagValue('')
    } catch (e) {
      setError(String(e))
    }
  }

  // ==================== Tag operations ====================

  const handleEnableByTag = async () => {
    if (!source?.enableByTag || !tagOpInput.trim()) return
    try {
      await source.enableByTag(tagOpInput.trim())
      await load(page)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleDisableByTag = async () => {
    if (!source?.disableByTag || !tagOpInput.trim()) return
    try {
      await source.disableByTag(tagOpInput.trim())
      await load(page)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleEditTag = async () => {
    if (!source?.editTag || !tagOpOldTag.trim() || !tagOpNewTag.trim()) return
    try {
      await source.editTag(tagOpOldTag.trim(), tagOpNewTag.trim())
      await load(page)
      setTagOpOldTag('')
      setTagOpNewTag('')
    } catch (e) {
      setError(String(e))
    }
  }

  // ==================== Save (create/edit) ====================

  const handleSave = async () => {
    if (!source || !editing) return
    try {
      if (editing.id) {
        await source.update(editing as GatewayChannel)
      } else {
        await source.create(editing)
      }
      setShowModal(false)
      setEditing(null)
      await load(page)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleFixAbilities = async () => {
    if (!source?.fixAbilities) return
    try {
      await source.fixAbilities()
      await load(page)
    } catch (e) {
      setError(String(e))
    }
  }

  // ==================== Selection helpers ====================

  const allSelected = channels.length > 0 && channels.every((ch) => selectedIds.has(ch.id))

  const toggleSelectAll = () => {
    if (allSelected) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(channels.map((ch) => ch.id)))
    }
  }

  const toggleSelect = (id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  // ==================== Confirm delete handler ====================

  const handleConfirmDelete = () => {
    if (confirmDelete === 'batch') {
      handleBatchDelete()
    } else if (typeof confirmDelete === 'number') {
      handleDelete(confirmDelete)
    }
    setConfirmDelete(null)
  }

  // ==================== Render: source unavailable ====================

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

  // ==================== Render ====================

  return (
    <div className="flex flex-col h-full overflow-y-auto p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold flex items-center gap-2">
          <Layers className="h-6 w-6 text-primary" />
          {t('gateway.channels')}
        </h2>
        <div className="flex gap-2">
          {caps?.fixAbilities && (
            <Button variant="secondary" size="sm" onClick={handleFixAbilities} title="Fix Abilities">
              Fix Abilities
            </Button>
          )}
          <Button
            variant="secondary"
            size="sm"
            onClick={() => load(page)}
            disabled={loading}
            loading={loading}
            icon={!loading ? <RefreshCw className="h-4 w-4" /> : undefined}
          />
          <Button
            size="sm"
            onClick={() => { setEditing({ status: 1, type: 1 }); setShowModal(true) }}
            icon={<Plus className="h-4 w-4" />}
          >
            {t('gateway.addChannel', 'Add Channel')}
          </Button>
        </div>
      </div>

      {error && (
        <div className="text-sm text-red-400 bg-red-500/10 border border-red-500/30 rounded px-3 py-2 font-mono">▸ {error}</div>
      )}

      {/* Search + Filters */}
      <SearchBar
        value={keyword}
        onChange={setKeyword}
        onSearch={handleSearch}
        placeholder={t('gateway.searchChannels', 'Search channels...')}
      >
        {/* Type filter dropdown */}
        <div className="relative">
          <select
            value={typeFilter ?? ''}
            onChange={(e) => {
              setTypeFilter(e.target.value === '' ? null : Number(e.target.value))
              setPage(0)
            }}
            className="appearance-none pl-3 pr-8 py-1.5 rounded-md border border-border bg-background text-sm focus:outline-none focus:ring-1 focus:ring-primary cursor-pointer"
          >
            <option value="">{t('gateway.allTypes', 'All Types')}</option>
            {Object.entries(CHANNEL_TYPES).map(([value, label]) => (
              <option key={value} value={value}>{label}</option>
            ))}
          </select>
          <ChevronDown className="absolute right-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
        </div>

        {/* Tag filter input */}
        <input
          type="text"
          value={tagFilter}
          onChange={(e) => { setTagFilter(e.target.value); setPage(0) }}
          placeholder={t('gateway.filterByTag', 'Filter by tag...')}
          className="px-3 py-1.5 rounded-md border border-border bg-background text-sm focus:outline-none focus:ring-1 focus:ring-primary min-w-[140px]"
        />
      </SearchBar>

      {/* Batch operations toolbar */}
      {selectedIds.size > 0 && (
        <Card variant="recessed" className="flex items-center gap-2 px-3 py-2 border-primary/30 bg-primary/10">
          <span className="font-mono text-[11px] uppercase tracking-[0.12em] text-muted-foreground">
            ▸ {selectedIds.size} {t('gateway.selected', 'selected')}
          </span>
          <div className="h-4 w-px bg-border" />
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setConfirmDelete('batch')}
            icon={<Trash2 className="h-3.5 w-3.5" />}
            className="hover:text-red-400 hover:bg-red-500/10"
          >
            {t('gateway.batchDelete', 'Batch Delete')}
          </Button>
          {caps?.batchEnableDisable && (
            <>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleBatchEnable}
                icon={<Check className="h-3.5 w-3.5" />}
                className="hover:text-emerald-400 hover:bg-emerald-500/10"
              >
                {t('gateway.batchEnable', 'Batch Enable')}
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleBatchDisable}
              >
                {t('gateway.batchDisable', 'Batch Disable')}
              </Button>
            </>
          )}
          {caps?.batchSetTag && (
          <div className="relative">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowBatchTagInput((prev) => !prev)}
              icon={<Tag className="h-3.5 w-3.5" />}
              className="hover:text-primary"
            >
              {t('gateway.setTag', 'Set Tag')}
            </Button>
            {showBatchTagInput && (
              <div className="absolute top-full left-0 mt-1 z-20 flex items-center gap-1 bg-card-elevated border border-rule-strong rounded-md p-2 shadow-card-elevated">
                <input
                  type="text"
                  value={batchTagValue}
                  onChange={(e) => setBatchTagValue(e.target.value)}
                  placeholder="tag"
                  className="px-2 py-1 rounded border border-border bg-background text-xs font-mono w-28 focus:outline-none focus:ring-1 focus:ring-primary"
                  autoFocus
                  onKeyDown={(e) => e.key === 'Enter' && handleBatchSetTag()}
                />
                <Button
                  variant="primary"
                  size="sm"
                  onClick={handleBatchSetTag}
                  disabled={!batchTagValue.trim()}
                  icon={<Check className="h-3.5 w-3.5" />}
                />
              </div>
            )}
          </div>
          )}
        </Card>
      )}

      {/* Table */}
      <Card variant="default" className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-card-recessed">
            <tr className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
              <th className="w-10 px-3 py-2">
                <input
                  type="checkbox"
                  checked={allSelected}
                  onChange={toggleSelectAll}
                  className="rounded border-border cursor-pointer accent-primary"
                />
              </th>
              <th className="text-left px-4 py-2">[ ID ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.channelName', 'Name').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.channelType', 'Type').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.channelTag', 'Tag').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.channelStatus', 'Status').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.channelBalance', 'Balance').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.channelTest', 'Test').toUpperCase()} ]</th>
              <th className="text-right px-4 py-2">[ {t('gateway.actions', 'Actions').toUpperCase()} ]</th>
            </tr>
          </thead>
          <tbody>
            {channels.length === 0 && (
              <tr>
                <td colSpan={9} className="text-center py-8 text-muted-foreground font-mono">
                  ▪ {loading ? t('status.loading') : t('gateway.noChannels', 'No channels')}
                </td>
              </tr>
            )}
            {channels.map((ch) => (
              <tr key={ch.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                <td className="w-10 px-3 py-2">
                  <input
                    type="checkbox"
                    checked={selectedIds.has(ch.id)}
                    onChange={() => toggleSelect(ch.id)}
                    className="rounded border-border cursor-pointer accent-primary"
                  />
                </td>
                <td className="px-4 py-2 text-muted-foreground font-mono tabular-nums">{ch.id}</td>
                <td className="px-4 py-2 font-medium">{ch.name}</td>
                <td className="px-4 py-2 text-xs font-mono">
                  {CHANNEL_TYPES[ch.type] ?? `Type ${ch.type}`}
                </td>
                <td className="px-4 py-2">
                  {ch.tag ? (
                    <span className="font-mono text-[10px] uppercase tracking-[0.08em] rounded px-1.5 py-0.5 bg-primary/10 text-primary border border-primary/30">
                      {ch.tag}
                    </span>
                  ) : (
                    <span className="text-muted-foreground font-mono">-</span>
                  )}
                </td>
                <td className="px-4 py-2">
                  <button
                    onClick={() => handleToggleStatus(ch)}
                    title={ch.status === 1 ? 'Click to disable' : 'Click to enable'}
                    className="transition-opacity hover:opacity-80"
                  >
                    <StatusBadge status={ch.status === 1 ? 'enabled' : 'disabled'} />
                  </button>
                </td>
                <td className="px-4 py-2 font-mono text-xs tabular-nums">{ch.balance ?? '-'}</td>
                <td className="px-4 py-2 text-xs text-muted-foreground max-w-[200px] truncate font-mono" title={testResults[ch.id]}>
                  {testResults[ch.id] ?? '-'}
                </td>
                <td className="px-4 py-2 text-right">
                  <div className="flex justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        setEditing({ ...ch })
                        setShowModal(true)
                      }}
                      title="Edit"
                      icon={<Edit2 className="h-3.5 w-3.5" />}
                      className="hover:text-primary"
                    />
                    {caps?.copy && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleCopy(ch.id)}
                        title="Copy"
                        icon={<Copy className="h-3.5 w-3.5" />}
                        className="hover:text-primary"
                      />
                    )}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleTest(ch.id)}
                      title="Test"
                      icon={<TestTube className="h-3.5 w-3.5" />}
                      className="hover:text-emerald-400"
                    />
                    {caps?.fetchModels && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleFetchModels(ch.id)}
                        title="Fetch Models"
                        icon={<Download className="h-3.5 w-3.5" />}
                        className="hover:text-amber-400"
                      />
                    )}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setConfirmDelete(ch.id)}
                      title="Delete"
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

      {/* Pagination */}
      <Pagination
        page={page}
        total={total}
        perPage={PER_PAGE}
        onPageChange={(p) => { setPage(p); setSelectedIds(new Set()) }}
      />

      {/* Tag Operations Panel — hidden when the data source doesn't expose
          tag bulk endpoints (e.g. Reseller Hub doesn't bind these yet). */}
      {caps?.tagOperations && (
      <Card variant="default" className="overflow-hidden">
        <button
          onClick={() => setShowTagOps((prev) => !prev)}
          className="w-full flex items-center justify-between px-4 py-2.5 bg-card-recessed hover:bg-muted/50 transition-colors"
        >
          <span className="font-mono text-[11px] uppercase tracking-[0.12em] text-muted-foreground">
            [ {t('gateway.tagOperations', 'Tag Operations').toUpperCase()} ]
          </span>
          <ChevronDown className={`h-4 w-4 text-muted-foreground transition-transform ${showTagOps ? 'rotate-180' : ''}`} />
        </button>
        {showTagOps && (
          <div className="px-4 py-4 space-y-4">
            {/* Enable / Disable by tag */}
            <div className="flex items-end gap-3 flex-wrap">
              <label className="block text-sm">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.tagName', 'Tag')}</span>
                <input
                  type="text"
                  value={tagOpInput}
                  onChange={(e) => setTagOpInput(e.target.value)}
                  placeholder="tag-name"
                  className="mt-1 block w-48 rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </label>
              <Button
                variant="primary"
                size="sm"
                onClick={handleEnableByTag}
                disabled={!tagOpInput.trim()}
                className="bg-emerald-600 hover:bg-emerald-500 ring-emerald-500/40"
              >
                {t('gateway.enableByTag', 'Enable by Tag')}
              </Button>
              <Button
                variant="secondary"
                size="sm"
                onClick={handleDisableByTag}
                disabled={!tagOpInput.trim()}
              >
                {t('gateway.disableByTag', 'Disable by Tag')}
              </Button>
            </div>

            {/* Edit tag (old -> new) */}
            <div className="flex items-end gap-3 flex-wrap border-t border-border pt-4">
              <label className="block text-sm">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.oldTag', 'Old Tag')}</span>
                <input
                  type="text"
                  value={tagOpOldTag}
                  onChange={(e) => setTagOpOldTag(e.target.value)}
                  placeholder="old-tag"
                  className="mt-1 block w-40 rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.newTag', 'New Tag')}</span>
                <input
                  type="text"
                  value={tagOpNewTag}
                  onChange={(e) => setTagOpNewTag(e.target.value)}
                  placeholder="new-tag"
                  className="mt-1 block w-40 rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </label>
              <Button
                variant="primary"
                size="sm"
                onClick={handleEditTag}
                disabled={!tagOpOldTag.trim() || !tagOpNewTag.trim()}
              >
                {t('gateway.editTag', 'Edit Tag')}
              </Button>
            </div>
          </div>
        )}
      </Card>
      )}

      {/* Delete Confirm Modal */}
      <ConfirmModal
        open={confirmDelete !== null}
        title={confirmDelete === 'batch'
          ? t('gateway.confirmBatchDelete', 'Confirm Batch Delete')
          : t('gateway.confirmDelete', 'Confirm Delete')
        }
        desc={confirmDelete === 'batch'
          ? t('gateway.confirmBatchDeleteDesc', `Delete ${selectedIds.size} selected channel(s)? This cannot be undone.`)
          : t('gateway.confirmDeleteDesc', 'Delete this channel? This cannot be undone.')
        }
        danger
        onConfirm={handleConfirmDelete}
        onCancel={() => setConfirmDelete(null)}
      />

      {/* Create/Edit Modal */}
      <Modal
        open={showModal && !!editing}
        onClose={() => { setShowModal(false); setEditing(null) }}
        title={editing?.id ? t('gateway.editChannel', 'Edit Channel') : t('gateway.addChannel', 'Add Channel')}
        icon={editing?.id ? Edit2 : Plus}
        size="xl"
        footer={
          <>
            <Button variant="secondary" size="sm" onClick={() => { setShowModal(false); setEditing(null) }}>
              {t('settings.data.cancel')}
            </Button>
            <Button size="sm" onClick={handleSave}>
              {t('settings.save')}
            </Button>
          </>
        }
      >
        {editing && (
          <div className="grid grid-cols-2 gap-4">
              {/* Name */}
              <label className="block text-sm col-span-2">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.channelName', 'Name')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                  value={editing.name ?? ''}
                  onChange={(e) => setEditing({ ...editing, name: e.target.value })}
                />
              </label>

              {/* Type */}
              <label className="block text-sm">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.channelType', 'Type')}</span>
                <div className="relative mt-1">
                  <select
                    value={editing.type ?? 1}
                    onChange={(e) => setEditing({ ...editing, type: Number(e.target.value) })}
                    className="w-full appearance-none rounded border border-border bg-background px-3 py-1.5 text-sm pr-8 cursor-pointer focus:outline-none focus:ring-1 focus:ring-primary"
                  >
                    {Object.entries(CHANNEL_TYPES).map(([value, label]) => (
                      <option key={value} value={value}>{label}</option>
                    ))}
                  </select>
                  <ChevronDown className="absolute right-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
                </div>
              </label>

              {/* Tag */}
              <label className="block text-sm">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.channelTag', 'Tag')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                  value={editing.tag ?? ''}
                  onChange={(e) => setEditing({ ...editing, tag: e.target.value })}
                />
              </label>

              {/* Key (multi-key textarea) */}
              <label className="block text-sm col-span-2">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.channelKey', 'API Key')} (one per line for multi-key)</span>
                <textarea
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono h-20 resize-y focus:outline-none focus:ring-1 focus:ring-primary"
                  value={editing.key ?? ''}
                  onChange={(e) => setEditing({ ...editing, key: e.target.value })}
                  placeholder="sk-..."
                />
              </label>

              {/* Base URL */}
              <label className="block text-sm col-span-2">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.channelBaseURL', 'Base URL')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                  value={editing.base_url ?? ''}
                  onChange={(e) => setEditing({ ...editing, base_url: e.target.value })}
                  placeholder="https://api.openai.com"
                />
              </label>

              {/* Models (textarea) */}
              <label className="block text-sm col-span-2">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.channelModels', 'Models')} (comma-separated)</span>
                <textarea
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono h-16 resize-y focus:outline-none focus:ring-1 focus:ring-primary"
                  value={editing.models ?? ''}
                  onChange={(e) => setEditing({ ...editing, models: e.target.value })}
                  placeholder="gpt-4o,claude-3-5-sonnet"
                />
              </label>

              {/* Group */}
              <label className="block text-sm">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.channelGroup', 'Group')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                  value={editing.group ?? ''}
                  onChange={(e) => setEditing({ ...editing, group: e.target.value })}
                  placeholder="default"
                />
              </label>

              {/* Priority */}
              <label className="block text-sm">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.channelPriority', 'Priority')}</span>
                <input
                  type="number"
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono tabular-nums focus:outline-none focus:ring-1 focus:ring-primary"
                  value={editing.priority ?? 0}
                  onChange={(e) => setEditing({ ...editing, priority: Number(e.target.value) })}
                />
              </label>

              {/* Weight */}
              <label className="block text-sm">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.channelWeight', 'Weight')}</span>
                <input
                  type="number"
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono tabular-nums focus:outline-none focus:ring-1 focus:ring-primary"
                  value={editing.weight ?? 0}
                  onChange={(e) => setEditing({ ...editing, weight: Number(e.target.value) })}
                />
              </label>

              {/* Auto Ban toggle */}
              <label className="flex items-center gap-2 text-sm col-span-1">
                <input
                  type="checkbox"
                  checked={(editing.auto_ban ?? 0) === 1}
                  onChange={(e) => setEditing({ ...editing, auto_ban: e.target.checked ? 1 : 0 })}
                  className="rounded border-border accent-primary"
                />
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.channelAutoBan', 'Auto Ban')}</span>
              </label>

              {/* Model Mapping (JSON textarea) */}
              <label className="block text-sm col-span-2">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.channelModelMapping', 'Model Mapping')} (JSON)</span>
                <textarea
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono h-20 resize-y focus:outline-none focus:ring-1 focus:ring-primary"
                  value={editing.model_mapping ?? ''}
                  onChange={(e) => setEditing({ ...editing, model_mapping: e.target.value })}
                  placeholder='{"gpt-4": "gpt-4-turbo"}'
                />
              </label>

              {/* Other (textarea) */}
              <label className="block text-sm col-span-2">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.channelOther', 'Other')}</span>
                <textarea
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono h-16 resize-y focus:outline-none focus:ring-1 focus:ring-primary"
                  value={editing.other ?? ''}
                  onChange={(e) => setEditing({ ...editing, other: e.target.value })}
                />
              </label>
            </div>
        )}
      </Modal>
    </div>
  )
}
