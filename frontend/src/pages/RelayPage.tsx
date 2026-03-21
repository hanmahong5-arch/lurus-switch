import { useEffect, useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Network, Plus, Trash2, RefreshCw, Loader2, CheckCircle2,
  WifiOff, Zap, ChevronDown, ChevronUp, X, Save,
} from 'lucide-react'
import { cn } from '../lib/utils'
import { useClassifiedError } from '../lib/useClassifiedError'
import { InlineError } from '../components/InlineError'
import { useRelayStore } from '../stores/relayStore'
import {
  GetRelayEndpoints,
  FetchCloudRelayEndpoints,
  SaveRelayEndpoint,
  DeleteRelayEndpoint,
  GetToolRelayMapping,
  SaveToolRelayMapping,
  CheckRelayHealth,
  ApplyAllToolRelays,
} from '../../wailsjs/go/main/App'
import { relay } from '../../wailsjs/go/models'

const TOOL_ORDER = ['claude', 'codex', 'gemini', 'picoclaw', 'nullclaw', 'zeroclaw', 'openclaw']

const TOOL_LABELS: Record<string, string> = {
  claude: 'Claude Code', codex: 'Codex', gemini: 'Gemini CLI',
  picoclaw: 'PicoClaw', nullclaw: 'NullClaw', zeroclaw: 'ZeroClaw', openclaw: 'OpenClaw',
}

function latencyColor(ms: number, healthy: boolean) {
  if (!healthy) return 'text-red-500'
  if (ms < 100) return 'text-green-500'
  if (ms < 300) return 'text-amber-500'
  return 'text-red-400'
}

export function RelayPage() {
  const { t } = useTranslation()
  const {
    endpoints, setEndpoints,
    cloudEndpoints, setCloudEndpoints,
    mapping, setMapping,
    loading, setLoading,
    applying, setApplying,
  } = useRelayStore()

  const KIND_LABELS: Record<string, string> = {
    lurus: t('relay.kindLabels.lurus'),
    third_party: t('relay.kindLabels.third_party'),
    custom: t('relay.kindLabels.custom'),
  }

  const [healthChecking, setHealthChecking] = useState(false)
  const [applyResults, setApplyResults] = useState<Record<string, string>>({})
  const [addOpen, setAddOpen] = useState(false)
  const [newName, setNewName] = useState('')
  const [newUrl, setNewUrl] = useState('')
  const [newKey, setNewKey] = useState('')
  const [saving, setSaving] = useState(false)
  const { classified: error, setError, clearError } = useClassifiedError()

  const load = useCallback(async () => {
    setLoading(true)
    clearError()
    try {
      const [eps, m] = await Promise.all([
        GetRelayEndpoints(),
        GetToolRelayMapping(),
      ])
      setEndpoints(eps || [])
      setMapping(m || {})
    } catch (err) {
      setError(err)
    } finally {
      setLoading(false)
    }
  }, [setEndpoints, setMapping, setLoading])

  useEffect(() => { load() }, [load])

  const handleFetchCloud = async () => {
    clearError()
    try {
      const eps = await FetchCloudRelayEndpoints()
      setCloudEndpoints(eps || [])
    } catch (err) {
      setError(err)
    }
  }

  const handleAdd = async () => {
    if (!newName.trim() || !newUrl.trim()) return
    setSaving(true)
    clearError()
    try {
      const ep = relay.RelayEndpoint.createFrom({
        id: '',
        name: newName.trim(),
        kind: 'custom',
        url: newUrl.trim(),
        apiKey: newKey.trim(),
        description: '',
        latencyMs: 0,
        healthy: false,
        lastChecked: '',
      })
      await SaveRelayEndpoint(ep)
      setNewName(''); setNewUrl(''); setNewKey('')
      setAddOpen(false)
      await load()
    } catch (err) {
      setError(err)
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await DeleteRelayEndpoint(id)
      await load()
    } catch (err) {
      setError(err)
    }
  }

  const handleUseCloud = async (ep: relay.RelayEndpoint) => {
    setSaving(true)
    clearError()
    try {
      await SaveRelayEndpoint(ep)
      await load()
    } catch (err) {
      setError(err)
    } finally {
      setSaving(false)
    }
  }

  const handleHealthCheck = async () => {
    setHealthChecking(true)
    clearError()
    try {
      const updated = await CheckRelayHealth()
      setEndpoints(updated || [])
    } catch (err) {
      setError(err)
    } finally {
      setHealthChecking(false)
    }
  }

  const handleMappingChange = (tool: string, relayId: string) => {
    setMapping({ ...mapping, [tool]: relayId })
  }

  const handleSaveMapping = async () => {
    try {
      await SaveToolRelayMapping(mapping)
    } catch (err) {
      setError(err)
    }
  }

  const handleApplyAll = async () => {
    setApplying(true)
    setApplyResults({})
    clearError()
    try {
      const results = await ApplyAllToolRelays()
      setApplyResults(results || {})
    } catch (err) {
      setError(err)
    } finally {
      setApplying(false)
    }
  }

  const customEndpoints = endpoints.filter((e) => e.kind === 'custom')
  const allEndpointOptions = endpoints

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-3xl mx-auto p-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold flex items-center gap-2">
              <Network className="h-5 w-5 text-sky-500" />
              {t('relay.title')}
            </h2>
            <p className="text-sm text-muted-foreground mt-0.5">{t('relay.subtitle')}</p>
          </div>
          <button
            onClick={handleHealthCheck}
            disabled={healthChecking}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
              'border border-border hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            {healthChecking ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
            {t('relay.healthCheck')}
          </button>
        </div>

        {/* Error banner */}
        {error && (
          <InlineError
            category={error.category}
            message={error.message}
            details={error.details}
            onDismiss={clearError}
          />
        )}

        {/* Zone A: Recommended cloud relays */}
        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold">{t('relay.recommended')}</h3>
            <button
              onClick={handleFetchCloud}
              className="text-xs text-primary hover:underline flex items-center gap-1"
            >
              <RefreshCw className="h-3 w-3" />
              {t('relay.fetchFromCloud')}
            </button>
          </div>
          {cloudEndpoints.length === 0 ? (
            <p className="text-xs text-muted-foreground py-2">{t('relay.fetchHint')}</p>
          ) : (
            <div className="space-y-2">
              {cloudEndpoints.map((ep) => (
                <div key={ep.id} className="border border-border rounded-lg p-3 bg-card flex items-center gap-3">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-medium">{ep.name}</span>
                      <span className="text-[10px] px-1.5 py-0.5 rounded bg-sky-500/10 text-sky-500">
                        {KIND_LABELS[ep.kind] || ep.kind}
                      </span>
                      {ep.latencyMs > 0 && (
                        <span className={cn('text-[10px]', latencyColor(ep.latencyMs, ep.healthy))}>
                          {ep.latencyMs}ms
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground truncate mt-0.5">{ep.url}</p>
                    {ep.description && (
                      <p className="text-xs text-muted-foreground/70 mt-0.5">{ep.description}</p>
                    )}
                  </div>
                  <button
                    onClick={() => handleUseCloud(ep)}
                    disabled={saving}
                    className="shrink-0 px-2.5 py-1 rounded text-xs font-medium bg-primary/10 hover:bg-primary/20 text-primary transition-colors disabled:opacity-50"
                  >
                    {t('relay.useRelay')}
                  </button>
                </div>
              ))}
            </div>
          )}
        </section>

        {/* Zone B: Custom endpoints */}
        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold">{t('relay.custom')}</h3>
            <button
              onClick={() => setAddOpen(!addOpen)}
              className="flex items-center gap-1 text-xs text-primary hover:underline"
            >
              <Plus className="h-3 w-3" />
              {t('relay.addRelay')}
              {addOpen ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
            </button>
          </div>

          {/* Add form */}
          {addOpen && (
            <div className="border border-border rounded-lg p-3 bg-muted/30 space-y-2">
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="block text-xs text-muted-foreground mb-1">{t('relay.nameLabel')}</label>
                  <input
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    placeholder={t('relay.namePlaceholder')}
                    className="w-full px-2 py-1.5 text-xs rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="block text-xs text-muted-foreground mb-1">URL *</label>
                  <input
                    value={newUrl}
                    onChange={(e) => setNewUrl(e.target.value)}
                    placeholder="https://api.example.com/v1"
                    className="w-full px-2 py-1.5 text-xs rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
              </div>
              <div>
                <label className="block text-xs text-muted-foreground mb-1">{t('relay.apiKeyLabel')}</label>
                <input
                  type="password"
                  value={newKey}
                  onChange={(e) => setNewKey(e.target.value)}
                  placeholder="sk-..."
                  className="w-full px-2 py-1.5 text-xs rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
              <div className="flex justify-end gap-2">
                <button
                  onClick={() => { setAddOpen(false); setNewName(''); setNewUrl(''); setNewKey('') }}
                  className="px-3 py-1.5 text-xs rounded border border-border hover:bg-muted transition-colors"
                >
                  {t('relay.cancel')}
                </button>
                <button
                  onClick={handleAdd}
                  disabled={saving || !newName.trim() || !newUrl.trim()}
                  className={cn(
                    'flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium transition-colors',
                    'bg-primary text-primary-foreground hover:bg-primary/90',
                    'disabled:opacity-50 disabled:cursor-not-allowed'
                  )}
                >
                  {saving ? <Loader2 className="h-3 w-3 animate-spin" /> : <Save className="h-3 w-3" />}
                  {t('relay.save')}
                </button>
              </div>
            </div>
          )}

          {loading ? (
            <div className="flex items-center gap-2 py-4">
              <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
              <span className="text-xs text-muted-foreground">{t('relay.loading')}</span>
            </div>
          ) : customEndpoints.length === 0 ? (
            <p className="text-xs text-muted-foreground py-2">{t('relay.emptyCustom')}</p>
          ) : (
            <div className="space-y-2">
              {customEndpoints.map((ep) => (
                <div key={ep.id} className="border border-border rounded-lg p-3 bg-card flex items-center gap-3">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-medium">{ep.name}</span>
                      {ep.latencyMs > 0 ? (
                        <span className={cn('text-[10px]', latencyColor(ep.latencyMs, ep.healthy))}>
                          {ep.healthy ? <CheckCircle2 className="h-2.5 w-2.5 inline mr-0.5" /> : <WifiOff className="h-2.5 w-2.5 inline mr-0.5" />}
                          {ep.latencyMs}ms
                        </span>
                      ) : null}
                    </div>
                    <p className="text-xs text-muted-foreground truncate mt-0.5">{ep.url}</p>
                  </div>
                  <button
                    onClick={() => handleDelete(ep.id)}
                    className="shrink-0 p-1.5 rounded hover:bg-red-500/10 text-red-500/60 hover:text-red-500 transition-colors"
                    title={t('relay.deleteTitle')}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </section>

        {/* Zone C: Tool → relay mapping */}
        <section className="space-y-3">
          <h3 className="text-sm font-semibold">{t('relay.toolRouting')}</h3>
          <div className="border border-border rounded-lg divide-y divide-border">
            {TOOL_ORDER.map((toolName) => (
              <div key={toolName} className="flex items-center justify-between px-3 py-2">
                <span className="text-xs font-medium">{TOOL_LABELS[toolName] || toolName}</span>
                <select
                  value={mapping[toolName] || ''}
                  onChange={(e) => handleMappingChange(toolName, e.target.value)}
                  className="text-xs px-2 py-1 rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary max-w-[200px]"
                >
                  <option value="">{t('relay.noRelay')}</option>
                  {allEndpointOptions.map((ep) => (
                    <option key={ep.id} value={ep.id}>{ep.name}</option>
                  ))}
                </select>
              </div>
            ))}
          </div>

          {/* Apply results */}
          {Object.keys(applyResults).length > 0 && (
            <div className="space-y-1">
              {Object.entries(applyResults).map(([tool, result]) => (
                <div key={tool} className={cn(
                  'flex items-center gap-2 text-xs px-2 py-1 rounded',
                  result === '' ? 'text-green-500 bg-green-500/5' : 'text-red-500 bg-red-500/5'
                )}>
                  {result === '' ? <CheckCircle2 className="h-3 w-3" /> : <X className="h-3 w-3" />}
                  {TOOL_LABELS[tool] || tool}: {result === '' ? t('relay.applied') : result}
                </div>
              ))}
            </div>
          )}

          <div className="flex gap-2">
            <button
              onClick={handleSaveMapping}
              className={cn(
                'flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium transition-colors',
                'border border-border hover:bg-muted'
              )}
            >
              <Save className="h-3.5 w-3.5" />
              {t('relay.saveMapping')}
            </button>
            <button
              onClick={handleApplyAll}
              disabled={applying}
              className={cn(
                'flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium transition-colors',
                'bg-primary text-primary-foreground hover:bg-primary/90',
                'disabled:opacity-50 disabled:cursor-not-allowed'
              )}
            >
              {applying ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Zap className="h-3.5 w-3.5" />}
              {t('relay.applyAll')}
            </button>
          </div>
        </section>
      </div>
    </div>
  )
}
