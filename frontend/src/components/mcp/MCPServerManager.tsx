import { useEffect, useState } from 'react'
import { Plus, Trash2, Loader2, Server, ChevronDown, ChevronUp } from 'lucide-react'
import { cn } from '../../lib/utils'
import { ListMCPPresets, SaveMCPPreset, DeleteMCPPreset, GetBuiltinMCPPresets, ApplyMCPServerToTool } from '../../../wailsjs/go/main/App'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyPreset = any
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyServer = any

const EMPTY_SERVER: AnyServer = {
  name: '',
  command: '',
  args: [],
  env: {},
  url: '',
  type: 'stdio',
}

const TOOLS = ['claude', 'codex', 'gemini', 'picoclaw', 'nullclaw']

export function MCPServerManager() {
  const [presets, setPresets] = useState<AnyPreset[]>([])
  const [builtins, setBuiltins] = useState<AnyPreset[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [editPreset, setEditPreset] = useState<AnyPreset>({})
  const [saving, setSaving] = useState(false)
  const [applyTool, setApplyTool] = useState('claude')
  const [applying, setApplying] = useState<Record<string, boolean>>({})
  const [expandedBuiltins, setExpandedBuiltins] = useState(false)
  const [newEnvKey, setNewEnvKey] = useState('')
  const [newEnvVal, setNewEnvVal] = useState('')
  const [newArg, setNewArg] = useState('')

  const load = async () => {
    setLoading(true)
    try {
      const [p, b] = await Promise.all([ListMCPPresets(), GetBuiltinMCPPresets()])
      setPresets(p || [])
      setBuiltins(b || [])
    } catch (err) {
      setError(`Failed to load: ${err}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])

  const handleSave = async () => {
    setSaving(true)
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      await SaveMCPPreset({
        id: editPreset.id || '',
        name: editPreset.name || 'Untitled',
        description: editPreset.description || '',
        server: editPreset.server || { ...EMPTY_SERVER },
        tags: editPreset.tags || [],
      } as any)
      setShowForm(false)
      setEditPreset({})
      await load()
    } catch (err) {
      setError(`Failed to save: ${err}`)
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await DeleteMCPPreset(id)
      await load()
    } catch (err) {
      setError(`Failed to delete: ${err}`)
    }
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleApply = async (preset: any) => {
    setApplying((prev) => ({ ...prev, [preset.id]: true }))
    try {
      await ApplyMCPServerToTool(applyTool, preset.server)
      // Success feedback could be shown here
    } catch (err) {
      setError(`Failed to apply: ${err}`)
    } finally {
      setApplying((prev) => ({ ...prev, [preset.id]: false }))
    }
  }

  const updateServer = (field: string, value: unknown) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    setEditPreset((prev: any) => ({
      ...prev,
      server: { ...(prev.server || EMPTY_SERVER), [field]: value },
    }))
  }

  const addArg = () => {
    if (!newArg.trim()) return
    updateServer('args', [...(editPreset.server?.args || []), newArg.trim()])
    setNewArg('')
  }

  const removeArg = (i: number) => {
    const args = [...(editPreset.server?.args || [])]
    args.splice(i, 1)
    updateServer('args', args)
  }

  const addEnv = () => {
    if (!newEnvKey.trim()) return
    updateServer('env', { ...(editPreset.server?.env || {}), [newEnvKey.trim()]: newEnvVal })
    setNewEnvKey('')
    setNewEnvVal('')
  }

  const removeEnv = (key: string) => {
    const env = { ...(editPreset.server?.env || {}) }
    delete env[key]
    updateServer('env', env)
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {error && (
        <div className="px-3 py-2 text-xs text-red-500 bg-red-500/10 rounded border border-red-500/20 flex items-center justify-between">
          {error}
          <button onClick={() => setError('')}>✕</button>
        </div>
      )}

      {/* Apply-to-tool selector */}
      <div className="flex items-center gap-2 text-sm">
        <span className="text-muted-foreground text-xs">应用到工具:</span>
        <select
          value={applyTool}
          onChange={(e) => setApplyTool(e.target.value)}
          className="px-2 py-1 text-xs bg-muted border border-border rounded focus:outline-none focus:ring-1 focus:ring-primary"
        >
          {TOOLS.map((t) => <option key={t} value={t}>{t}</option>)}
        </select>
      </div>

      {/* Built-in presets */}
      <div className="border border-border rounded-lg overflow-hidden">
        <button
          onClick={() => setExpandedBuiltins(!expandedBuiltins)}
          className="w-full flex items-center justify-between px-4 py-3 bg-muted/30 text-sm font-medium"
        >
          <span className="flex items-center gap-2">
            <Server className="h-4 w-4 text-muted-foreground" />
            内置 MCP 预设 ({builtins.length})
          </span>
          {expandedBuiltins ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
        </button>
        {expandedBuiltins && (
          <div className="divide-y divide-border">
            {builtins.map((p) => (
              <div key={p.id} className="px-4 py-3 flex items-start justify-between gap-3">
                <div>
                  <p className="text-sm font-medium">{p.name}</p>
                  <p className="text-xs text-muted-foreground">{p.description}</p>
                  <p className="text-xs font-mono text-muted-foreground mt-1">
                    {p.server.type === 'stdio' ? p.server.command : p.server.url}
                  </p>
                </div>
                <button
                  onClick={() => handleApply(p)}
                  disabled={applying[p.id]}
                  className="shrink-0 px-2 py-1 text-xs border border-primary text-primary rounded hover:bg-primary/10 transition-colors disabled:opacity-50"
                >
                  {applying[p.id] ? <Loader2 className="h-3 w-3 animate-spin" /> : '应用'}
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* User presets */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium">自定义预设</span>
          <button
            onClick={() => { setEditPreset({ server: { ...EMPTY_SERVER } }); setShowForm(true) }}
            className="flex items-center gap-1 px-2 py-1 text-xs bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors"
          >
            <Plus className="h-3 w-3" /> 新建
          </button>
        </div>
        {presets.length === 0 ? (
          <p className="text-xs text-muted-foreground text-center py-4">暂无自定义预设</p>
        ) : (
          <div className="border border-border rounded-lg divide-y divide-border">
            {presets.map((p) => (
              <div key={p.id} className="px-4 py-3 flex items-start justify-between gap-3">
                <div>
                  <p className="text-sm font-medium">{p.name}</p>
                  <p className="text-xs text-muted-foreground">{p.description}</p>
                  <div className="flex gap-1 mt-1">
                    {p.tags?.map((t: string) => (
                      <span key={t} className="text-xs bg-muted px-1.5 py-0.5 rounded">{t}</span>
                    ))}
                  </div>
                </div>
                <div className="flex items-center gap-1 shrink-0">
                  <button
                    onClick={() => handleApply(p)}
                    disabled={applying[p.id]}
                    className="px-2 py-1 text-xs border border-primary text-primary rounded hover:bg-primary/10 transition-colors disabled:opacity-50"
                  >
                    {applying[p.id] ? <Loader2 className="h-3 w-3 animate-spin" /> : '应用'}
                  </button>
                  <button
                    onClick={() => handleDelete(p.id)}
                    className="p-1 text-muted-foreground hover:text-red-500 transition-colors"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Form Modal */}
      {showForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-card border border-border rounded-lg p-6 max-w-lg w-full mx-4 shadow-xl max-h-[80vh] overflow-y-auto space-y-4">
            <h3 className="font-semibold">新建 MCP 预设</h3>

            <input
              type="text"
              value={editPreset.name || ''}
              onChange={(e) => setEditPreset({ ...editPreset, name: e.target.value })}
              placeholder="预设名称"
              className="w-full px-3 py-2 text-sm bg-muted border border-border rounded focus:outline-none focus:ring-1 focus:ring-primary"
            />
            <input
              type="text"
              value={editPreset.description || ''}
              onChange={(e) => setEditPreset({ ...editPreset, description: e.target.value })}
              placeholder="描述（可选）"
              className="w-full px-3 py-2 text-sm bg-muted border border-border rounded focus:outline-none focus:ring-1 focus:ring-primary"
            />

            <div className="space-y-2">
              <label className="text-xs font-medium">传输类型</label>
              <select
                value={editPreset.server?.type || 'stdio'}
                onChange={(e) => updateServer('type', e.target.value)}
                className="w-full px-3 py-2 text-sm bg-muted border border-border rounded focus:outline-none focus:ring-1 focus:ring-primary"
              >
                <option value="stdio">stdio</option>
                <option value="sse">SSE</option>
                <option value="http">HTTP</option>
              </select>
            </div>

            {editPreset.server?.type === 'stdio' ? (
              <>
                <input
                  type="text"
                  value={editPreset.server?.command || ''}
                  onChange={(e) => updateServer('command', e.target.value)}
                  placeholder="命令（如 npx、python）"
                  className="w-full px-3 py-2 text-sm bg-muted border border-border rounded focus:outline-none focus:ring-1 focus:ring-primary"
                />
                <div className="space-y-1">
                  <label className="text-xs font-medium">参数</label>
                  <div className="flex gap-1">
                    <input
                      type="text"
                      value={newArg}
                      onChange={(e) => setNewArg(e.target.value)}
                      onKeyDown={(e) => e.key === 'Enter' && addArg()}
                      placeholder="添加参数..."
                      className="flex-1 px-2 py-1 text-xs bg-muted border border-border rounded focus:outline-none"
                    />
                    <button onClick={addArg} className="px-2 py-1 text-xs bg-muted border border-border rounded hover:bg-muted/80">
                      添加
                    </button>
                  </div>
                  {editPreset.server?.args?.map((arg: string, i: number) => (
                    <div key={i} className="flex items-center gap-1 text-xs bg-muted/50 px-2 py-1 rounded">
                      <span className="font-mono flex-1">{arg}</span>
                      <button onClick={() => removeArg(i)} className="text-muted-foreground hover:text-red-500">✕</button>
                    </div>
                  ))}
                </div>
              </>
            ) : (
              <input
                type="text"
                value={editPreset.server?.url || ''}
                onChange={(e) => updateServer('url', e.target.value)}
                placeholder="URL（如 http://localhost:3000/sse）"
                className="w-full px-3 py-2 text-sm bg-muted border border-border rounded focus:outline-none focus:ring-1 focus:ring-primary"
              />
            )}

            <div className="space-y-1">
              <label className="text-xs font-medium">环境变量</label>
              <div className="flex gap-1">
                <input
                  type="text"
                  value={newEnvKey}
                  onChange={(e) => setNewEnvKey(e.target.value)}
                  placeholder="KEY"
                  className="w-1/3 px-2 py-1 text-xs bg-muted border border-border rounded focus:outline-none"
                />
                <input
                  type="text"
                  value={newEnvVal}
                  onChange={(e) => setNewEnvVal(e.target.value)}
                  placeholder="VALUE"
                  className="flex-1 px-2 py-1 text-xs bg-muted border border-border rounded focus:outline-none"
                />
                <button onClick={addEnv} className="px-2 py-1 text-xs bg-muted border border-border rounded hover:bg-muted/80">
                  添加
                </button>
              </div>
              {Object.entries(editPreset.server?.env || {}).map(([k, v]) => (
                <div key={k} className="flex items-center gap-1 text-xs bg-muted/50 px-2 py-1 rounded">
                  <span className="font-mono text-primary">{k}</span>
                  <span className="text-muted-foreground">=</span>
                  <span className="font-mono flex-1">{String(v)}</span>
                  <button onClick={() => removeEnv(k)} className="text-muted-foreground hover:text-red-500">✕</button>
                </div>
              ))}
            </div>

            <div className="flex gap-2 pt-2">
              <button
                onClick={() => { setShowForm(false); setEditPreset({}) }}
                className="flex-1 px-4 py-2 text-sm border border-border rounded hover:bg-muted transition-colors"
              >
                取消
              </button>
              <button
                onClick={handleSave}
                disabled={saving}
                className="flex-1 px-4 py-2 text-sm bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {saving ? <Loader2 className="h-4 w-4 animate-spin inline" /> : '保存'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
