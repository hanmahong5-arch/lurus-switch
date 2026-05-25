import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Box, Plus, Trash2, RefreshCw, Edit2, Download, AlertCircle } from 'lucide-react'
import { Button, Card, Modal } from '../components/ui'
import { useGatewayStore } from '../stores/gatewayStore'
import { createGatewayClient, type GatewayModelMeta, type GatewayVendor } from '../lib/gateway-api'
import { SearchBar } from '../components/gateway/SearchBar'
import { Pagination } from '../components/gateway/Pagination'
import { ConfirmModal } from '../components/gateway/ConfirmModal'
import { ModelHealthMatrix } from '../components/ModelHealthMatrix'
import { ModelAuthenticityPanel } from '../components/ModelAuthenticityPanel'

type TabKey = 'models' | 'vendors' | 'sync' | 'health'

const MODEL_TYPES = ['chat', 'embedding', 'image', 'audio'] as const
const STATUS_OPTIONS = [
  { value: 1, label: 'Enabled' },
  { value: 0, label: 'Disabled' },
] as const

const PER_PAGE = 50

interface ModelFormData {
  id?: number   // present in edit mode (newapi rc.4 keys models by integer id)
  model_name: string
  developer: string
  type: string
  context_length: number
  input_price: number
  output_price: number
  tags: string
  status: number
}

interface VendorFormData {
  id?: number
  name: string
  description: string
  icon_url: string
  website: string
  status: number
}

function emptyModelForm(): ModelFormData {
  return {
    model_name: '',
    developer: '',
    type: 'chat',
    context_length: 0,
    input_price: 0,
    output_price: 0,
    tags: '',
    status: 1,
  }
}

function emptyVendorForm(): VendorFormData {
  return {
    name: '',
    description: '',
    icon_url: '',
    website: '',
    status: 1,
  }
}

function modelToForm(m: GatewayModelMeta): ModelFormData {
  return {
    id: m.id,
    model_name: m.model_name,
    developer: m.developer,
    type: m.type,
    context_length: m.context_length,
    input_price: m.input_price,
    output_price: m.output_price,
    tags: (m.tags ?? []).join(', '),
    status: m.status,
  }
}

function vendorToForm(v: GatewayVendor): VendorFormData {
  return {
    id: v.id,
    name: v.name,
    description: v.description,
    icon_url: v.icon_url,
    website: v.website,
    status: v.status,
  }
}

export function GatewayModelPage() {
  const { t } = useTranslation()
  const { status: serverStatus, adminToken } = useGatewayStore()

  // Tab
  const [tab, setTab] = useState<TabKey>('models')

  // Models state
  const [models, setModels] = useState<GatewayModelMeta[]>([])
  const [page, setPage] = useState(0)
  const [total, setTotal] = useState(0)
  const [search, setSearch] = useState('')
  const [filteredModels, setFilteredModels] = useState<GatewayModelMeta[]>([])

  // Vendors state
  const [vendors, setVendors] = useState<GatewayVendor[]>([])

  // Sync state
  const [previewModels, setPreviewModels] = useState<GatewayModelMeta[]>([])
  const [missingModels, setMissingModels] = useState<string[]>([])

  // Shared UI state
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Model modal
  const [showModelModal, setShowModelModal] = useState(false)
  const [editingModel, setEditingModel] = useState<ModelFormData | null>(null)
  const [editingModelOriginalName, setEditingModelOriginalName] = useState<string | null>(null)

  // Vendor modal
  const [showVendorModal, setShowVendorModal] = useState(false)
  const [editingVendor, setEditingVendor] = useState<VendorFormData | null>(null)

  // Confirm delete — both model and vendor are now keyed by integer id.
  const [confirmDelete, setConfirmDelete] = useState<{
    type: 'model' | 'vendor'
    id: number
    name: string
  } | null>(null)

  const client = serverStatus?.running && adminToken
    ? createGatewayClient(serverStatus.url, adminToken)
    : null

  // ========== Data Loading ==========

  const loadModels = async () => {
    if (!client) return
    setLoading(true)
    setError(null)
    try {
      const res = await client.getModels(page, PER_PAGE)
      const data = res.data ?? []
      setModels(data)
      setTotal(data.length < PER_PAGE && page === 0 ? data.length : (page + 1) * PER_PAGE + 1)
      applySearch(data, search)
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  const loadVendors = async () => {
    if (!client) return
    setLoading(true)
    setError(null)
    try {
      const res = await client.getVendors()
      setVendors(res.data ?? [])
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  const applySearch = (data: GatewayModelMeta[], keyword: string) => {
    if (!keyword.trim()) {
      setFilteredModels(data)
      return
    }
    const kw = keyword.toLowerCase()
    setFilteredModels(
      data.filter(
        (m) =>
          m.model_name.toLowerCase().includes(kw) ||
          m.developer.toLowerCase().includes(kw) ||
          m.type.toLowerCase().includes(kw)
      )
    )
  }

  const handleSearch = () => {
    applySearch(models, search)
  }

  useEffect(() => {
    if (tab === 'models') loadModels()
    if (tab === 'vendors') loadVendors()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [serverStatus?.running, adminToken, tab, page])

  // ========== Model CRUD ==========

  const handleSaveModel = async () => {
    if (!client || !editingModel) return
    setError(null)
    try {
      const tags = editingModel.tags
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean)

      const payload = {
        model_name: editingModel.model_name,
        developer: editingModel.developer,
        type: editingModel.type,
        context_length: Number(editingModel.context_length),
        input_price: Number(editingModel.input_price),
        output_price: Number(editingModel.output_price),
        tags,
        status: editingModel.status,
      }

      if (editingModel.id !== undefined) {
        await client.updateModel({ ...payload, id: editingModel.id })
      } else {
        await client.createModel(payload)
      }
      setShowModelModal(false)
      setEditingModel(null)
      setEditingModelOriginalName(null)
      await loadModels()
    } catch (e) {
      setError(String(e))
    }
  }

  const handleDeleteModel = async (id: number) => {
    if (!client) return
    try {
      await client.deleteModel(id)
      await loadModels()
    } catch (e) {
      setError(String(e))
    }
  }

  // ========== Vendor CRUD ==========

  const handleSaveVendor = async () => {
    if (!client || !editingVendor) return
    setError(null)
    try {
      if (editingVendor.id) {
        await client.updateVendor({
          id: editingVendor.id,
          name: editingVendor.name,
          description: editingVendor.description,
          icon_url: editingVendor.icon_url,
          website: editingVendor.website,
          status: editingVendor.status,
        })
      } else {
        await client.createVendor({
          name: editingVendor.name,
          description: editingVendor.description,
          icon_url: editingVendor.icon_url,
          website: editingVendor.website,
          status: editingVendor.status,
        })
      }
      setShowVendorModal(false)
      setEditingVendor(null)
      await loadVendors()
    } catch (e) {
      setError(String(e))
    }
  }

  const handleDeleteVendor = async (id: number) => {
    if (!client) return
    try {
      await client.deleteVendor(id)
      await loadVendors()
    } catch (e) {
      setError(String(e))
    }
  }

  // ========== Sync ==========

  const handleSyncPreview = async () => {
    if (!client) return
    setLoading(true)
    setError(null)
    setPreviewModels([])
    try {
      const res = await client.syncUpstreamPreview()
      setPreviewModels(res.data ?? [])
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  const handleSyncNow = async () => {
    if (!client) return
    setLoading(true)
    setError(null)
    try {
      await client.syncUpstream()
      setPreviewModels([])
      await loadModels()
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  const handleMissingModels = async () => {
    if (!client) return
    setLoading(true)
    setError(null)
    setMissingModels([])
    try {
      const res = await client.getMissingModels()
      setMissingModels(res.data ?? [])
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  // ========== Confirm Delete Handler ==========

  const handleConfirmDelete = async () => {
    if (!confirmDelete) return
    if (confirmDelete.type === 'model') {
      await handleDeleteModel(confirmDelete.id)
    } else {
      await handleDeleteVendor(confirmDelete.id)
    }
    setConfirmDelete(null)
  }

  // ========== Render: Server Stopped ==========

  if (!serverStatus?.running) {
    return (
      <div className="flex flex-col h-full items-center justify-center text-muted-foreground gap-2">
        <AlertCircle className="h-8 w-8" />
        <p>{t('gateway.status.stopped')}</p>
      </div>
    )
  }

  // ========== Tab definitions ==========

  const tabs: { key: TabKey; label: string }[] = [
    { key: 'models', label: t('gateway.models', 'Models') },
    { key: 'vendors', label: t('gateway.vendors', 'Vendors') },
    { key: 'sync', label: t('gateway.sync', 'Sync') },
    { key: 'health', label: t('gateway.health', '可用性自检') },
  ]

  return (
    <div className="flex flex-col h-full overflow-y-auto p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold flex items-center gap-2">
          <Box className="h-6 w-6 text-primary" />
          {t('gateway.models', 'Models')}
        </h2>
      </div>

      {/* Tab Bar */}
      <div className="flex border-b border-border">
        {tabs.map((tb) => {
          const isActive = tab === tb.key
          return (
            <button
              key={tb.key}
              onClick={() => setTab(tb.key)}
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

      {/* Error Banner */}
      {error && (
        <div className="text-sm text-red-400 bg-red-500/10 border border-red-500/30 rounded px-3 py-2 font-mono">▸ {error}</div>
      )}

      {/* ===== Models Tab ===== */}
      {tab === 'models' && (
        <>
          {/* Toolbar */}
          <div className="flex items-center gap-2">
            <SearchBar
              value={search}
              onChange={setSearch}
              onSearch={handleSearch}
              placeholder={t('gateway.searchModels', 'Search models...')}
            >
              <Button
                variant="secondary"
                size="sm"
                onClick={loadModels}
                disabled={loading}
                loading={loading}
                icon={!loading ? <RefreshCw className="h-4 w-4" /> : undefined}
              />
              <Button
                size="sm"
                onClick={() => {
                  setEditingModel(emptyModelForm())
                  setEditingModelOriginalName(null)
                  setShowModelModal(true)
                }}
                icon={<Plus className="h-4 w-4" />}
              >
                {t('gateway.addModel', 'Add Model')}
              </Button>
            </SearchBar>
          </div>

          {/* Models Table */}
          <Card variant="default" className="overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-card-recessed">
                <tr className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
                  <th className="text-left px-4 py-2">[ {t('gateway.modelName', 'Model Name').toUpperCase()} ]</th>
                  <th className="text-left px-4 py-2">[ {t('gateway.modelDeveloper', 'Developer').toUpperCase()} ]</th>
                  <th className="text-left px-4 py-2">[ {t('gateway.modelType', 'Type').toUpperCase()} ]</th>
                  <th className="text-left px-4 py-2">[ {t('gateway.modelContext', 'Context').toUpperCase()} ]</th>
                  <th className="text-left px-4 py-2">[ {t('gateway.modelInputPrice', 'Input Price').toUpperCase()} ]</th>
                  <th className="text-left px-4 py-2">[ {t('gateway.modelOutputPrice', 'Output Price').toUpperCase()} ]</th>
                  <th className="text-left px-4 py-2">[ {t('gateway.modelStatus', 'Status').toUpperCase()} ]</th>
                  <th className="text-right px-4 py-2">[ {t('gateway.actions', 'Actions').toUpperCase()} ]</th>
                </tr>
              </thead>
              <tbody>
                {filteredModels.length === 0 && (
                  <tr>
                    <td colSpan={8} className="text-center py-8 text-muted-foreground font-mono">
                      ▪ {loading ? t('status.loading') : t('gateway.noModels', 'No models')}
                    </td>
                  </tr>
                )}
                {filteredModels.map((m) => (
                  <tr key={m.model_name} className="border-t border-border hover:bg-muted/30 transition-colors">
                    <td className="px-4 py-2 font-medium font-mono text-xs">{m.model_name}</td>
                    <td className="px-4 py-2">{m.developer}</td>
                    <td className="px-4 py-2">
                      <span className="font-mono text-[10px] uppercase tracking-[0.08em] rounded px-1.5 py-0.5 bg-card-recessed text-muted-foreground border border-border">{m.type}</span>
                    </td>
                    <td className="px-4 py-2 font-mono text-xs tabular-nums">
                      {m.context_length > 0 ? m.context_length.toLocaleString() : '-'}
                    </td>
                    <td className="px-4 py-2 font-mono text-xs tabular-nums">{m.input_price}</td>
                    <td className="px-4 py-2 font-mono text-xs tabular-nums">{m.output_price}</td>
                    <td className="px-4 py-2">
                      <span
                        className={`font-mono text-[10px] uppercase tracking-[0.08em] rounded px-1.5 py-0.5 border ${
                          m.status === 1
                            ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                            : 'bg-card-recessed text-muted-foreground border-border'
                        }`}
                      >
                        {m.status === 1 ? '▸ Enabled' : '▪ Disabled'}
                      </span>
                    </td>
                    <td className="px-4 py-2 text-right">
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setEditingModel(modelToForm(m))
                            setEditingModelOriginalName(m.model_name)
                            setShowModelModal(true)
                          }}
                          title="Edit"
                          icon={<Edit2 className="h-3.5 w-3.5" />}
                          className="hover:text-primary"
                        />
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() =>
                            setConfirmDelete({
                              type: 'model',
                              id: m.id,
                              name: m.model_name,
                            })
                          }
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
            onPageChange={setPage}
          />
        </>
      )}

      {/* ===== Vendors Tab ===== */}
      {tab === 'vendors' && (
        <>
          <div className="flex items-center justify-end gap-2">
            <Button
              variant="secondary"
              size="sm"
              onClick={loadVendors}
              disabled={loading}
              loading={loading}
              icon={!loading ? <RefreshCw className="h-4 w-4" /> : undefined}
            />
            <Button
              size="sm"
              onClick={() => {
                setEditingVendor(emptyVendorForm())
                setShowVendorModal(true)
              }}
              icon={<Plus className="h-4 w-4" />}
            >
              {t('gateway.addVendor', 'Add Vendor')}
            </Button>
          </div>

          <Card variant="default" className="overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-card-recessed">
                <tr className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
                  <th className="text-left px-4 py-2">[ {t('gateway.vendorName', 'Name').toUpperCase()} ]</th>
                  <th className="text-left px-4 py-2">[ {t('gateway.vendorDescription', 'Description').toUpperCase()} ]</th>
                  <th className="text-left px-4 py-2">[ {t('gateway.vendorWebsite', 'Website').toUpperCase()} ]</th>
                  <th className="text-left px-4 py-2">[ {t('gateway.vendorStatus', 'Status').toUpperCase()} ]</th>
                  <th className="text-right px-4 py-2">[ {t('gateway.actions', 'Actions').toUpperCase()} ]</th>
                </tr>
              </thead>
              <tbody>
                {vendors.length === 0 && (
                  <tr>
                    <td colSpan={5} className="text-center py-8 text-muted-foreground font-mono">
                      ▪ {loading ? t('status.loading') : t('gateway.noVendors', 'No vendors')}
                    </td>
                  </tr>
                )}
                {vendors.map((v) => (
                  <tr key={v.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                    <td className="px-4 py-2 font-medium">{v.name}</td>
                    <td className="px-4 py-2 text-muted-foreground text-xs">{v.description || '-'}</td>
                    <td className="px-4 py-2 text-xs font-mono">
                      {v.website ? (
                        <a
                          href={v.website}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-primary hover:underline"
                        >
                          {v.website}
                        </a>
                      ) : (
                        '-'
                      )}
                    </td>
                    <td className="px-4 py-2">
                      <span
                        className={`font-mono text-[10px] uppercase tracking-[0.08em] rounded px-1.5 py-0.5 border ${
                          v.status === 1
                            ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                            : 'bg-card-recessed text-muted-foreground border-border'
                        }`}
                      >
                        {v.status === 1 ? '▸ Enabled' : '▪ Disabled'}
                      </span>
                    </td>
                    <td className="px-4 py-2 text-right">
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setEditingVendor(vendorToForm(v))
                            setShowVendorModal(true)
                          }}
                          title="Edit"
                          icon={<Edit2 className="h-3.5 w-3.5" />}
                          className="hover:text-primary"
                        />
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() =>
                            setConfirmDelete({
                              type: 'vendor',
                              id: v.id,
                              name: v.name,
                            })
                          }
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
        </>
      )}

      {/* ===== Sync Tab ===== */}
      {tab === 'sync' && (
        <>
          <div className="flex items-center gap-3 flex-wrap">
            <Button
              variant="secondary"
              size="md"
              onClick={handleSyncPreview}
              disabled={loading}
              loading={loading}
              icon={!loading ? <Download className="h-4 w-4" /> : undefined}
            >
              {t('gateway.syncPreview', 'Preview Upstream')}
            </Button>
            <Button
              variant="primary"
              size="md"
              onClick={handleSyncNow}
              disabled={loading}
              loading={loading}
              icon={!loading ? <RefreshCw className="h-4 w-4" /> : undefined}
            >
              {t('gateway.syncNow', 'Sync Now')}
            </Button>
            <Button
              variant="secondary"
              size="md"
              onClick={handleMissingModels}
              disabled={loading}
              icon={<AlertCircle className="h-4 w-4" />}
            >
              {t('gateway.showMissing', 'Show Missing Models')}
            </Button>
          </div>

          {/* Preview Results */}
          {previewModels.length > 0 && (
            <div className="space-y-2">
              <h3 className="font-mono text-[11px] uppercase tracking-[0.12em] text-muted-foreground">
                [ {t('gateway.syncPreviewTitle', 'Models to be added').toUpperCase()} ] ({previewModels.length})
              </h3>
              <Card variant="default" className="overflow-hidden max-h-80 overflow-y-auto">
                <ul className="divide-y divide-border">
                  {previewModels.map((m) => (
                    <li
                      key={m.model_name}
                      className="px-4 py-2 text-sm font-mono hover:bg-muted/30 flex items-center justify-between transition-colors"
                    >
                      <span>{m.model_name}</span>
                      <span className="text-xs text-muted-foreground">{m.developer}</span>
                    </li>
                  ))}
                </ul>
              </Card>
            </div>
          )}

          {/* Missing Models */}
          {missingModels.length > 0 && (
            <div className="space-y-2">
              <h3 className="font-mono text-[11px] uppercase tracking-[0.12em] text-muted-foreground">
                [ {t('gateway.missingModels', 'Missing Models').toUpperCase()} ] ({missingModels.length})
              </h3>
              <Card variant="default" className="overflow-hidden max-h-80 overflow-y-auto">
                <ul className="divide-y divide-border">
                  {missingModels.map((name) => (
                    <li
                      key={name}
                      className="px-4 py-2 text-sm font-mono hover:bg-muted/30 transition-colors"
                    >
                      {name}
                    </li>
                  ))}
                </ul>
              </Card>
            </div>
          )}

          {previewModels.length === 0 && missingModels.length === 0 && !loading && (
            <div className="text-center py-12 text-muted-foreground font-mono text-sm">
              ▪ {t('gateway.syncHint', 'Click a button above to check upstream or missing models.')}
            </div>
          )}
        </>
      )}

      {tab === 'health' && (
        <div className="space-y-6">
          <ModelHealthMatrix includeCustom />
          <div className="border-t border-border pt-6">
            <ModelAuthenticityPanel includeCustom />
          </div>
        </div>
      )}

      {/* ===== Model Create/Edit Modal ===== */}
      <Modal
        open={showModelModal && !!editingModel}
        onClose={() => {
          setShowModelModal(false)
          setEditingModel(null)
          setEditingModelOriginalName(null)
        }}
        title={editingModelOriginalName ? t('gateway.editModel', 'Edit Model') : t('gateway.addModel', 'Add Model')}
        icon={editingModelOriginalName ? Edit2 : Plus}
        size="lg"
        footer={
          <>
            <Button
              variant="secondary"
              size="sm"
              onClick={() => {
                setShowModelModal(false)
                setEditingModel(null)
                setEditingModelOriginalName(null)
              }}
            >
              {t('settings.data.cancel')}
            </Button>
            <Button size="sm" onClick={handleSaveModel}>
              {t('settings.save')}
            </Button>
          </>
        }
      >
        {editingModel && (
          <div className="space-y-3">
            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.modelName', 'Model Name')}</span>
              <input
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary disabled:opacity-60"
                value={editingModel.model_name}
                onChange={(e) => setEditingModel({ ...editingModel, model_name: e.target.value })}
                disabled={!!editingModelOriginalName}
              />
            </label>

            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.modelDeveloper', 'Developer')}</span>
              <input
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                value={editingModel.developer}
                onChange={(e) => setEditingModel({ ...editingModel, developer: e.target.value })}
              />
            </label>

            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.modelType', 'Type')}</span>
              <select
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                value={editingModel.type}
                onChange={(e) => setEditingModel({ ...editingModel, type: e.target.value })}
              >
                {MODEL_TYPES.map((mt) => (
                  <option key={mt} value={mt}>
                    {mt}
                  </option>
                ))}
              </select>
            </label>

            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.modelContext', 'Context Length')}</span>
              <input
                type="number"
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono tabular-nums focus:outline-none focus:ring-1 focus:ring-primary"
                value={editingModel.context_length}
                onChange={(e) =>
                  setEditingModel({ ...editingModel, context_length: Number(e.target.value) })
                }
              />
            </label>

            <div className="grid grid-cols-2 gap-3">
              <label className="block text-sm">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.modelInputPrice', 'Input Price')}</span>
                <input
                  type="number"
                  step="any"
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono tabular-nums focus:outline-none focus:ring-1 focus:ring-primary"
                  value={editingModel.input_price}
                  onChange={(e) =>
                    setEditingModel({ ...editingModel, input_price: Number(e.target.value) })
                  }
                />
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.modelOutputPrice', 'Output Price')}</span>
                <input
                  type="number"
                  step="any"
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono tabular-nums focus:outline-none focus:ring-1 focus:ring-primary"
                  value={editingModel.output_price}
                  onChange={(e) =>
                    setEditingModel({ ...editingModel, output_price: Number(e.target.value) })
                  }
                />
              </label>
            </div>

            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.modelTags', 'Tags (comma-separated)')}</span>
              <input
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                value={editingModel.tags}
                onChange={(e) => setEditingModel({ ...editingModel, tags: e.target.value })}
                placeholder="reasoning, vision, tool-use"
              />
            </label>

            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.modelStatus', 'Status')}</span>
              <select
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                value={editingModel.status}
                onChange={(e) =>
                  setEditingModel({ ...editingModel, status: Number(e.target.value) })
                }
              >
                {STATUS_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </label>
          </div>
        )}
      </Modal>

      {/* ===== Vendor Create/Edit Modal ===== */}
      <Modal
        open={showVendorModal && !!editingVendor}
        onClose={() => {
          setShowVendorModal(false)
          setEditingVendor(null)
        }}
        title={editingVendor?.id ? t('gateway.editVendor', 'Edit Vendor') : t('gateway.addVendor', 'Add Vendor')}
        icon={editingVendor?.id ? Edit2 : Plus}
        size="md"
        footer={
          <>
            <Button
              variant="secondary"
              size="sm"
              onClick={() => {
                setShowVendorModal(false)
                setEditingVendor(null)
              }}
            >
              {t('settings.data.cancel')}
            </Button>
            <Button size="sm" onClick={handleSaveVendor}>
              {t('settings.save')}
            </Button>
          </>
        }
      >
        {editingVendor && (
          <div className="space-y-3">
            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.vendorName', 'Name')}</span>
              <input
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                value={editingVendor.name}
                onChange={(e) => setEditingVendor({ ...editingVendor, name: e.target.value })}
              />
            </label>

            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.vendorDescription', 'Description')}</span>
              <input
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                value={editingVendor.description}
                onChange={(e) =>
                  setEditingVendor({ ...editingVendor, description: e.target.value })
                }
              />
            </label>

            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.vendorIcon', 'Icon URL')}</span>
              <input
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                value={editingVendor.icon_url}
                onChange={(e) =>
                  setEditingVendor({ ...editingVendor, icon_url: e.target.value })
                }
                placeholder="https://..."
              />
            </label>

            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.vendorWebsite', 'Website')}</span>
              <input
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                value={editingVendor.website}
                onChange={(e) =>
                  setEditingVendor({ ...editingVendor, website: e.target.value })
                }
                placeholder="https://..."
              />
            </label>

            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.vendorStatus', 'Status')}</span>
              <select
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                value={editingVendor.status}
                onChange={(e) =>
                  setEditingVendor({ ...editingVendor, status: Number(e.target.value) })
                }
              >
                {STATUS_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </label>
          </div>
        )}
      </Modal>

      {/* ===== Confirm Delete Modal ===== */}
      <ConfirmModal
        open={!!confirmDelete}
        title={t('gateway.confirmDelete', 'Confirm Delete')}
        desc={
          confirmDelete
            ? t('gateway.confirmDeleteDesc', `Are you sure you want to delete "${confirmDelete.name}"? This action cannot be undone.`)
            : ''
        }
        danger
        onConfirm={handleConfirmDelete}
        onCancel={() => setConfirmDelete(null)}
      />
    </div>
  )
}
