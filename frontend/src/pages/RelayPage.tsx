import { useEffect, useState, useCallback } from 'react'
import {
  Network, Plus, Trash2, RefreshCw, Loader2, CheckCircle2,
  WifiOff, Zap, ChevronDown, ChevronUp, X, Save,
} from 'lucide-react'
import { cn } from '../lib/utils'
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

const KIND_LABELS: Record<string, string> = {
  lurus: 'Lurus 官方',
  third_party: '第三方',
  custom: '自定义',
}

function latencyColor(ms: number, healthy: boolean) {
  if (!healthy) return 'text-red-500'
  if (ms < 100) return 'text-green-500'
  if (ms < 300) return 'text-amber-500'
  return 'text-red-400'
}

export function RelayPage() {
  const {
    endpoints, setEndpoints,
    cloudEndpoints, setCloudEndpoints,
    mapping, setMapping,
    loading, setLoading,
    applying, setApplying,
  } = useRelayStore()

  const [healthChecking, setHealthChecking] = useState(false)
  const [applyResults, setApplyResults] = useState<Record<string, string>>({})
  const [addOpen, setAddOpen] = useState(false)
  const [newName, setNewName] = useState('')
  const [newUrl, setNewUrl] = useState('')
  const [newKey, setNewKey] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [eps, m] = await Promise.all([
        GetRelayEndpoints(),
        GetToolRelayMapping(),
      ])
      setEndpoints(eps || [])
      setMapping(m || {})
    } catch (err) {
      setError(`加载失败: ${err}`)
    } finally {
      setLoading(false)
    }
  }, [setEndpoints, setMapping, setLoading])

  useEffect(() => { load() }, [load])

  const handleFetchCloud = async () => {
    setError('')
    try {
      const eps = await FetchCloudRelayEndpoints()
      setCloudEndpoints(eps || [])
    } catch (err) {
      setError(`获取云端中转站失败: ${err}`)
    }
  }

  const handleAdd = async () => {
    if (!newName.trim() || !newUrl.trim()) return
    setSaving(true)
    setError('')
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
      setError(`保存失败: ${err}`)
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await DeleteRelayEndpoint(id)
      await load()
    } catch (err) {
      setError(`删除失败: ${err}`)
    }
  }

  const handleUseCloud = async (ep: relay.RelayEndpoint) => {
    setSaving(true)
    setError('')
    try {
      await SaveRelayEndpoint(ep)
      await load()
    } catch (err) {
      setError(`添加失败: ${err}`)
    } finally {
      setSaving(false)
    }
  }

  const handleHealthCheck = async () => {
    setHealthChecking(true)
    setError('')
    try {
      const updated = await CheckRelayHealth()
      setEndpoints(updated || [])
    } catch (err) {
      setError(`健康检查失败: ${err}`)
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
      setError(`保存映射失败: ${err}`)
    }
  }

  const handleApplyAll = async () => {
    setApplying(true)
    setApplyResults({})
    setError('')
    try {
      const results = await ApplyAllToolRelays()
      setApplyResults(results || {})
    } catch (err) {
      setError(`应用失败: ${err}`)
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
              中转站管理
            </h2>
            <p className="text-sm text-muted-foreground mt-0.5">管理 API 中转站端点及工具路由映射</p>
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
            健康检查
          </button>
        </div>

        {/* Error banner */}
        {error && (
          <div className="flex items-center justify-between px-4 py-2 bg-red-500/10 text-red-500 text-xs rounded-md border border-red-500/20">
            <span>{error}</span>
            <button onClick={() => setError('')}><X className="h-3.5 w-3.5" /></button>
          </div>
        )}

        {/* Zone A: Recommended cloud relays */}
        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold">推荐中转站</h3>
            <button
              onClick={handleFetchCloud}
              className="text-xs text-primary hover:underline flex items-center gap-1"
            >
              <RefreshCw className="h-3 w-3" />
              从云端获取
            </button>
          </div>
          {cloudEndpoints.length === 0 ? (
            <p className="text-xs text-muted-foreground py-2">点击"从云端获取"加载推荐中转站列表</p>
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
                    使用此中转站
                  </button>
                </div>
              ))}
            </div>
          )}
        </section>

        {/* Zone B: Custom endpoints */}
        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold">自定义中转站</h3>
            <button
              onClick={() => setAddOpen(!addOpen)}
              className="flex items-center gap-1 text-xs text-primary hover:underline"
            >
              <Plus className="h-3 w-3" />
              添加中转站
              {addOpen ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
            </button>
          </div>

          {/* Add form */}
          {addOpen && (
            <div className="border border-border rounded-lg p-3 bg-muted/30 space-y-2">
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="block text-xs text-muted-foreground mb-1">名称 *</label>
                  <input
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    placeholder="我的中转站"
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
                <label className="block text-xs text-muted-foreground mb-1">API Key（可选）</label>
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
                  取消
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
                  保存
                </button>
              </div>
            </div>
          )}

          {loading ? (
            <div className="flex items-center gap-2 py-4">
              <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
              <span className="text-xs text-muted-foreground">加载中...</span>
            </div>
          ) : customEndpoints.length === 0 ? (
            <p className="text-xs text-muted-foreground py-2">暂无自定义中转站</p>
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
                    title="删除"
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
          <h3 className="text-sm font-semibold">工具路由映射</h3>
          <div className="border border-border rounded-lg divide-y divide-border">
            {TOOL_ORDER.map((toolName) => (
              <div key={toolName} className="flex items-center justify-between px-3 py-2">
                <span className="text-xs font-medium">{TOOL_LABELS[toolName] || toolName}</span>
                <select
                  value={mapping[toolName] || ''}
                  onChange={(e) => handleMappingChange(toolName, e.target.value)}
                  className="text-xs px-2 py-1 rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary max-w-[200px]"
                >
                  <option value="">— 不指定 —</option>
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
                  {TOOL_LABELS[tool] || tool}: {result === '' ? '已应用' : result}
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
              保存映射
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
              一键应用
            </button>
          </div>
        </section>
      </div>
    </div>
  )
}
