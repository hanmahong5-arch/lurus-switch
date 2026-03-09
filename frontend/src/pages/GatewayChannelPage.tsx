import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Layers, Plus, Trash2, TestTube, RefreshCw, AlertCircle,
  Edit2, Copy, Download, ChevronDown, Check,
} from 'lucide-react'
import { useGatewayStore } from '../stores/gatewayStore'
import { createGatewayClient, CHANNEL_TYPES, type GatewayChannel } from '../lib/gateway-api'
import { SearchBar } from '../components/gateway/SearchBar'
import { Pagination } from '../components/gateway/Pagination'
import { StatusBadge } from '../components/gateway/StatusBadge'
import { ConfirmModal } from '../components/gateway/ConfirmModal'

const PER_PAGE = 50

export function GatewayChannelPage() {
  const { t } = useTranslation()
  const { status: serverStatus, adminToken } = useGatewayStore()

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

  const client = serverStatus?.running && adminToken
    ? createGatewayClient(serverStatus.url, adminToken)
    : null

  const load = useCallback(async (p = page) => {
    if (!client) return
    setLoading(true)
    setError(null)
    try {
      const res = keyword.trim()
        ? await client.searchChannels(keyword.trim(), p, PER_PAGE)
        : await client.getChannels(p, PER_PAGE)
      let data = res.data ?? []

      // Client-side type filter
      if (typeFilter !== null) {
        data = data.filter((ch) => ch.type === typeFilter)
      }
      // Client-side tag filter
      if (tagFilter.trim()) {
        const tf = tagFilter.trim().toLowerCase()
        data = data.filter((ch) => (ch.tag ?? '').toLowerCase().includes(tf))
      }

      setChannels(data)
      // The API may not return total; estimate from response length
      setTotal(data.length < PER_PAGE && p === 0
        ? data.length
        : Math.max((p + 1) * PER_PAGE + (data.length === PER_PAGE ? PER_PAGE : 0), data.length)
      )
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }, [client, keyword, typeFilter, tagFilter, page])

  useEffect(() => {
    load(page)
  }, [serverStatus?.running, adminToken, page])

  const handleSearch = useCallback(() => {
    setPage(0)
    setSelectedIds(new Set())
    load(0)
  }, [load])

  // ==================== Single row actions ====================

  const handleDelete = async (id: number) => {
    if (!client) return
    try {
      await client.deleteChannel(id)
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
    if (!client) return
    setTestResults((prev) => ({ ...prev, [id]: 'testing...' }))
    try {
      const res = await client.testChannel(id)
      setTestResults((prev) => ({ ...prev, [id]: res.data ?? res.message ?? 'OK' }))
    } catch (e) {
      setTestResults((prev) => ({ ...prev, [id]: String(e) }))
    }
  }

  const handleCopy = async (id: number) => {
    if (!client) return
    try {
      await client.copyChannel(id)
      await load(page)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleFetchModels = async (id: number) => {
    if (!client) return
    try {
      const res = await client.fetchChannelModels(id)
      const models = res.data
      if (models && models.length > 0) {
        setTestResults((prev) => ({
          ...prev,
          [id]: `Fetched ${models.length} models: ${models.slice(0, 5).join(', ')}${models.length > 5 ? '...' : ''}`,
        }))
      } else {
        setTestResults((prev) => ({ ...prev, [id]: res.message ?? 'No models found' }))
      }
    } catch (e) {
      setTestResults((prev) => ({ ...prev, [id]: String(e) }))
    }
  }

  const handleToggleStatus = async (ch: GatewayChannel) => {
    if (!client) return
    const newStatus = ch.status === 1 ? 2 : 1
    try {
      await client.updateChannel({ id: ch.id, status: newStatus })
      setChannels((prev) =>
        prev.map((c) => (c.id === ch.id ? { ...c, status: newStatus } : c))
      )
    } catch (e) {
      setError(String(e))
    }
  }

  // ==================== Batch operations ====================

  const handleBatchDelete = async () => {
    if (!client || selectedIds.size === 0) return
    try {
      await client.batchDeleteChannels(Array.from(selectedIds))
      setSelectedIds(new Set())
      await load(page)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleBatchEnable = async () => {
    if (!client || selectedIds.size === 0) return
    try {
      await client.batchEnableChannels(Array.from(selectedIds))
      setChannels((prev) =>
        prev.map((c) => (selectedIds.has(c.id) ? { ...c, status: 1 } : c))
      )
    } catch (e) {
      setError(String(e))
    }
  }

  const handleBatchDisable = async () => {
    if (!client || selectedIds.size === 0) return
    try {
      await client.batchDisableChannels(Array.from(selectedIds))
      setChannels((prev) =>
        prev.map((c) => (selectedIds.has(c.id) ? { ...c, status: 2 } : c))
      )
    } catch (e) {
      setError(String(e))
    }
  }

  const handleBatchSetTag = async () => {
    if (!client || selectedIds.size === 0 || !batchTagValue.trim()) return
    try {
      await client.batchSetChannelTag(Array.from(selectedIds), batchTagValue.trim())
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
    if (!client || !tagOpInput.trim()) return
    try {
      await client.enableChannelsByTag(tagOpInput.trim())
      await load(page)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleDisableByTag = async () => {
    if (!client || !tagOpInput.trim()) return
    try {
      await client.disableChannelsByTag(tagOpInput.trim())
      await load(page)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleEditTag = async () => {
    if (!client || !tagOpOldTag.trim() || !tagOpNewTag.trim()) return
    try {
      await client.editChannelTag(tagOpOldTag.trim(), tagOpNewTag.trim())
      await load(page)
      setTagOpOldTag('')
      setTagOpNewTag('')
    } catch (e) {
      setError(String(e))
    }
  }

  // ==================== Save (create/edit) ====================

  const handleSave = async () => {
    if (!client || !editing) return
    try {
      if (editing.id) {
        await client.updateChannel(editing as GatewayChannel)
      } else {
        await client.createChannel(editing)
      }
      setShowModal(false)
      setEditing(null)
      await load(page)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleFixAbilities = async () => {
    if (!client) return
    try {
      await client.fixChannelAbilities()
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

  // ==================== Render: server not running ====================

  if (!serverStatus?.running) {
    return (
      <div className="flex flex-col h-full items-center justify-center text-muted-foreground gap-2">
        <AlertCircle className="h-8 w-8" />
        <p>{t('gateway.status.stopped')}</p>
      </div>
    )
  }

  // ==================== Render ====================

  return (
    <div className="flex flex-col h-full overflow-y-auto p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold flex items-center gap-2">
          <Layers className="h-6 w-6 text-blue-400" />
          {t('gateway.channels')}
        </h2>
        <div className="flex gap-2">
          <button
            onClick={handleFixAbilities}
            title="Fix Abilities"
            className="flex items-center gap-1 px-3 py-1.5 rounded-md border border-border hover:bg-muted text-sm text-muted-foreground"
          >
            Fix Abilities
          </button>
          <button
            onClick={() => load(page)}
            disabled={loading}
            className="flex items-center gap-1 px-3 py-1.5 rounded-md border border-border hover:bg-muted text-sm"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
          <button
            onClick={() => { setEditing({ status: 1, type: 1 }); setShowModal(true) }}
            className="flex items-center gap-1 px-3 py-1.5 rounded-md bg-indigo-600 hover:bg-indigo-500 text-white text-sm"
          >
            <Plus className="h-4 w-4" />
            {t('gateway.addChannel', 'Add Channel')}
          </button>
        </div>
      </div>

      {error && (
        <div className="text-sm text-red-400 bg-red-900/20 rounded px-3 py-2">{error}</div>
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
        <div className="flex items-center gap-2 px-3 py-2 rounded-lg border border-indigo-500/30 bg-indigo-950/20 text-sm">
          <span className="text-muted-foreground">
            {selectedIds.size} {t('gateway.selected', 'selected')}
          </span>
          <div className="h-4 w-px bg-border" />
          <button
            onClick={() => setConfirmDelete('batch')}
            className="px-2.5 py-1 rounded hover:bg-red-900/30 text-red-400 text-xs font-medium"
          >
            {t('gateway.batchDelete', 'Batch Delete')}
          </button>
          <button
            onClick={handleBatchEnable}
            className="px-2.5 py-1 rounded hover:bg-green-900/30 text-green-400 text-xs font-medium"
          >
            {t('gateway.batchEnable', 'Batch Enable')}
          </button>
          <button
            onClick={handleBatchDisable}
            className="px-2.5 py-1 rounded hover:bg-muted text-muted-foreground text-xs font-medium"
          >
            {t('gateway.batchDisable', 'Batch Disable')}
          </button>
          <div className="relative">
            <button
              onClick={() => setShowBatchTagInput((prev) => !prev)}
              className="px-2.5 py-1 rounded hover:bg-muted text-blue-400 text-xs font-medium"
            >
              {t('gateway.setTag', 'Set Tag')}
            </button>
            {showBatchTagInput && (
              <div className="absolute top-full left-0 mt-1 z-20 flex items-center gap-1 bg-card border border-border rounded-md p-2 shadow-lg">
                <input
                  type="text"
                  value={batchTagValue}
                  onChange={(e) => setBatchTagValue(e.target.value)}
                  placeholder="tag"
                  className="px-2 py-1 rounded border border-border bg-background text-xs w-28"
                  autoFocus
                  onKeyDown={(e) => e.key === 'Enter' && handleBatchSetTag()}
                />
                <button
                  onClick={handleBatchSetTag}
                  disabled={!batchTagValue.trim()}
                  className="p-1 rounded bg-indigo-600 hover:bg-indigo-500 text-white disabled:opacity-50"
                >
                  <Check className="h-3.5 w-3.5" />
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Table */}
      <div className="rounded-lg border border-border overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-muted-foreground">
            <tr>
              <th className="w-10 px-3 py-2">
                <input
                  type="checkbox"
                  checked={allSelected}
                  onChange={toggleSelectAll}
                  className="rounded border-border cursor-pointer accent-indigo-500"
                />
              </th>
              <th className="text-left px-4 py-2">ID</th>
              <th className="text-left px-4 py-2">{t('gateway.channelName', 'Name')}</th>
              <th className="text-left px-4 py-2">{t('gateway.channelType', 'Type')}</th>
              <th className="text-left px-4 py-2">{t('gateway.channelTag', 'Tag')}</th>
              <th className="text-left px-4 py-2">{t('gateway.channelStatus', 'Status')}</th>
              <th className="text-left px-4 py-2">{t('gateway.channelBalance', 'Balance')}</th>
              <th className="text-left px-4 py-2">{t('gateway.channelTest', 'Test')}</th>
              <th className="text-right px-4 py-2">{t('gateway.actions', 'Actions')}</th>
            </tr>
          </thead>
          <tbody>
            {channels.length === 0 && (
              <tr>
                <td colSpan={9} className="text-center py-8 text-muted-foreground">
                  {loading ? t('status.loading') : t('gateway.noChannels', 'No channels')}
                </td>
              </tr>
            )}
            {channels.map((ch) => (
              <tr key={ch.id} className="border-t border-border hover:bg-muted/30">
                <td className="w-10 px-3 py-2">
                  <input
                    type="checkbox"
                    checked={selectedIds.has(ch.id)}
                    onChange={() => toggleSelect(ch.id)}
                    className="rounded border-border cursor-pointer accent-indigo-500"
                  />
                </td>
                <td className="px-4 py-2 text-muted-foreground">{ch.id}</td>
                <td className="px-4 py-2 font-medium">{ch.name}</td>
                <td className="px-4 py-2 text-xs">
                  {CHANNEL_TYPES[ch.type] ?? `Type ${ch.type}`}
                </td>
                <td className="px-4 py-2">
                  {ch.tag ? (
                    <span className="text-xs rounded px-1.5 py-0.5 bg-blue-900/30 text-blue-400">
                      {ch.tag}
                    </span>
                  ) : (
                    <span className="text-muted-foreground">-</span>
                  )}
                </td>
                <td className="px-4 py-2">
                  <button
                    onClick={() => handleToggleStatus(ch)}
                    title={ch.status === 1 ? 'Click to disable' : 'Click to enable'}
                    className="cursor-pointer"
                  >
                    <StatusBadge status={ch.status === 1 ? 'enabled' : 'disabled'} />
                  </button>
                </td>
                <td className="px-4 py-2 font-mono text-xs">{ch.balance ?? '-'}</td>
                <td className="px-4 py-2 text-xs text-muted-foreground max-w-[200px] truncate" title={testResults[ch.id]}>
                  {testResults[ch.id] ?? '-'}
                </td>
                <td className="px-4 py-2 text-right">
                  <div className="flex justify-end gap-1">
                    <button
                      onClick={() => {
                        setEditing({ ...ch })
                        setShowModal(true)
                      }}
                      title="Edit"
                      className="p-1 hover:text-blue-400"
                    >
                      <Edit2 className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => handleCopy(ch.id)}
                      title="Copy"
                      className="p-1 hover:text-indigo-400"
                    >
                      <Copy className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => handleTest(ch.id)}
                      title="Test"
                      className="p-1 hover:text-green-400"
                    >
                      <TestTube className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => handleFetchModels(ch.id)}
                      title="Fetch Models"
                      className="p-1 hover:text-yellow-400"
                    >
                      <Download className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => setConfirmDelete(ch.id)}
                      title="Delete"
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

      {/* Pagination */}
      <Pagination
        page={page}
        total={total}
        perPage={PER_PAGE}
        onPageChange={(p) => { setPage(p); setSelectedIds(new Set()) }}
      />

      {/* Tag Operations Panel */}
      <div className="rounded-lg border border-border overflow-hidden">
        <button
          onClick={() => setShowTagOps((prev) => !prev)}
          className="w-full flex items-center justify-between px-4 py-2.5 bg-muted/30 hover:bg-muted/50 text-sm font-medium"
        >
          <span>{t('gateway.tagOperations', 'Tag Operations')}</span>
          <ChevronDown className={`h-4 w-4 transition-transform ${showTagOps ? 'rotate-180' : ''}`} />
        </button>
        {showTagOps && (
          <div className="px-4 py-4 space-y-4">
            {/* Enable / Disable by tag */}
            <div className="flex items-end gap-3 flex-wrap">
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.tagName', 'Tag')}</span>
                <input
                  type="text"
                  value={tagOpInput}
                  onChange={(e) => setTagOpInput(e.target.value)}
                  placeholder="tag-name"
                  className="mt-1 block w-48 rounded border border-border bg-background px-3 py-1.5 text-sm"
                />
              </label>
              <button
                onClick={handleEnableByTag}
                disabled={!tagOpInput.trim()}
                className="px-3 py-1.5 rounded bg-green-700 hover:bg-green-600 text-white text-sm disabled:opacity-50"
              >
                {t('gateway.enableByTag', 'Enable by Tag')}
              </button>
              <button
                onClick={handleDisableByTag}
                disabled={!tagOpInput.trim()}
                className="px-3 py-1.5 rounded bg-muted hover:bg-muted/80 text-sm border border-border disabled:opacity-50"
              >
                {t('gateway.disableByTag', 'Disable by Tag')}
              </button>
            </div>

            {/* Edit tag (old -> new) */}
            <div className="flex items-end gap-3 flex-wrap border-t border-border pt-4">
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.oldTag', 'Old Tag')}</span>
                <input
                  type="text"
                  value={tagOpOldTag}
                  onChange={(e) => setTagOpOldTag(e.target.value)}
                  placeholder="old-tag"
                  className="mt-1 block w-40 rounded border border-border bg-background px-3 py-1.5 text-sm"
                />
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.newTag', 'New Tag')}</span>
                <input
                  type="text"
                  value={tagOpNewTag}
                  onChange={(e) => setTagOpNewTag(e.target.value)}
                  placeholder="new-tag"
                  className="mt-1 block w-40 rounded border border-border bg-background px-3 py-1.5 text-sm"
                />
              </label>
              <button
                onClick={handleEditTag}
                disabled={!tagOpOldTag.trim() || !tagOpNewTag.trim()}
                className="px-3 py-1.5 rounded bg-indigo-600 hover:bg-indigo-500 text-white text-sm disabled:opacity-50"
              >
                {t('gateway.editTag', 'Edit Tag')}
              </button>
            </div>
          </div>
        )}
      </div>

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
      {showModal && editing && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-card border border-border rounded-lg p-6 w-[560px] max-h-[85vh] overflow-y-auto space-y-4">
            <h3 className="font-semibold text-lg">
              {editing.id
                ? t('gateway.editChannel', 'Edit Channel')
                : t('gateway.addChannel', 'Add Channel')
              }
            </h3>

            <div className="grid grid-cols-2 gap-4">
              {/* Name */}
              <label className="block text-sm col-span-2">
                <span className="text-muted-foreground">{t('gateway.channelName', 'Name')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.name ?? ''}
                  onChange={(e) => setEditing({ ...editing, name: e.target.value })}
                />
              </label>

              {/* Type */}
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.channelType', 'Type')}</span>
                <div className="relative mt-1">
                  <select
                    value={editing.type ?? 1}
                    onChange={(e) => setEditing({ ...editing, type: Number(e.target.value) })}
                    className="w-full appearance-none rounded border border-border bg-background px-3 py-1.5 text-sm pr-8 cursor-pointer"
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
                <span className="text-muted-foreground">{t('gateway.channelTag', 'Tag')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.tag ?? ''}
                  onChange={(e) => setEditing({ ...editing, tag: e.target.value })}
                />
              </label>

              {/* Key (multi-key textarea) */}
              <label className="block text-sm col-span-2">
                <span className="text-muted-foreground">{t('gateway.channelKey', 'API Key')} (one per line for multi-key)</span>
                <textarea
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono h-20 resize-y"
                  value={editing.key ?? ''}
                  onChange={(e) => setEditing({ ...editing, key: e.target.value })}
                  placeholder="sk-..."
                />
              </label>

              {/* Base URL */}
              <label className="block text-sm col-span-2">
                <span className="text-muted-foreground">{t('gateway.channelBaseURL', 'Base URL')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono"
                  value={editing.base_url ?? ''}
                  onChange={(e) => setEditing({ ...editing, base_url: e.target.value })}
                  placeholder="https://api.openai.com"
                />
              </label>

              {/* Models (textarea) */}
              <label className="block text-sm col-span-2">
                <span className="text-muted-foreground">{t('gateway.channelModels', 'Models')} (comma-separated)</span>
                <textarea
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm h-16 resize-y"
                  value={editing.models ?? ''}
                  onChange={(e) => setEditing({ ...editing, models: e.target.value })}
                  placeholder="gpt-4o,claude-3-5-sonnet"
                />
              </label>

              {/* Group */}
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.channelGroup', 'Group')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.group ?? ''}
                  onChange={(e) => setEditing({ ...editing, group: e.target.value })}
                  placeholder="default"
                />
              </label>

              {/* Priority */}
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.channelPriority', 'Priority')}</span>
                <input
                  type="number"
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.priority ?? 0}
                  onChange={(e) => setEditing({ ...editing, priority: Number(e.target.value) })}
                />
              </label>

              {/* Weight */}
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.channelWeight', 'Weight')}</span>
                <input
                  type="number"
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
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
                  className="rounded border-border accent-indigo-500"
                />
                <span className="text-muted-foreground">{t('gateway.channelAutoBan', 'Auto Ban')}</span>
              </label>

              {/* Model Mapping (JSON textarea) */}
              <label className="block text-sm col-span-2">
                <span className="text-muted-foreground">{t('gateway.channelModelMapping', 'Model Mapping')} (JSON)</span>
                <textarea
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono h-20 resize-y"
                  value={editing.model_mapping ?? ''}
                  onChange={(e) => setEditing({ ...editing, model_mapping: e.target.value })}
                  placeholder='{"gpt-4": "gpt-4-turbo"}'
                />
              </label>

              {/* Other (textarea) */}
              <label className="block text-sm col-span-2">
                <span className="text-muted-foreground">{t('gateway.channelOther', 'Other')}</span>
                <textarea
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm h-16 resize-y"
                  value={editing.other ?? ''}
                  onChange={(e) => setEditing({ ...editing, other: e.target.value })}
                />
              </label>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => { setShowModal(false); setEditing(null) }}
                className="px-4 py-1.5 rounded border border-border text-sm hover:bg-muted"
              >
                {t('settings.data.cancel')}
              </button>
              <button
                onClick={handleSave}
                className="px-4 py-1.5 rounded bg-indigo-600 hover:bg-indigo-500 text-white text-sm"
              >
                {t('settings.save')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
