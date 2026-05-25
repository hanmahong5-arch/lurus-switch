import { useEffect, useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Network, Plus, Trash2, RefreshCw, Loader2, CheckCircle2,
  WifiOff, Zap, ChevronDown, ChevronUp, X, Save, Info,
} from 'lucide-react'
import { cn } from '../lib/utils'
import { useClassifiedError } from '../lib/useClassifiedError'
import { InlineError } from '../components/InlineError'
import { Button, Card } from '../components/ui'
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
import { CircuitStateChip } from '../components/relay/CircuitStateChip'
import { RelayRulesEditor } from '../components/relay/RelayRulesEditor'
import { RouterDryRunPanel } from '../components/relay/RouterDryRunPanel'

const TOOL_ORDER = ['claude', 'codex', 'gemini', 'picoclaw', 'nullclaw', 'zeroclaw', 'openclaw']

const TOOL_LABELS: Record<string, string> = {
  claude: 'Claude Code', codex: 'Codex', gemini: 'Gemini CLI',
  picoclaw: 'PicoClaw', nullclaw: 'NullClaw', zeroclaw: 'ZeroClaw', openclaw: 'OpenClaw',
}

// StepBadge — a tiny numbered chip used to signpost the three sections so
// the user reads the page as a sequence (pick → add → wire up) rather
// than three unrelated blocks of URLs.
function StepBadge({ n }: { n: number }) {
  return (
    <span className="inline-flex items-center justify-center mr-1.5 h-4 w-4 rounded-full bg-primary/15 text-primary font-mono text-[10px] font-semibold tabular-nums">
      {n}
    </span>
  )
}

function latencyColor(ms: number, healthy: boolean) {
  if (!healthy) return 'text-red-400'
  if (ms < 100) return 'text-emerald-400'
  if (ms < 300) return 'text-amber-400'
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

  // Count how many tools use each endpoint, so each row can show
  // "N 个工具正在用" — turns an opaque URL list into something the user
  // can actually reason about ("if I delete this, what breaks?").
  const usageByEndpoint: Record<string, number> = {}
  for (const v of Object.values(mapping)) {
    if (v) usageByEndpoint[v] = (usageByEndpoint[v] ?? 0) + 1
  }

  // endpointById lets the tool-routing rows show the resolved URL inline,
  // so "Claude Code → newapi (https://newapi.lurus.cn)" beats "Claude Code → newapi".
  const endpointById: Record<string, relay.RelayEndpoint> = {}
  for (const e of endpoints) endpointById[e.id] = e

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
          <Button
            variant="secondary"
            size="sm"
            onClick={handleHealthCheck}
            disabled={healthChecking}
            loading={healthChecking}
            icon={!healthChecking ? <RefreshCw className="h-3.5 w-3.5" /> : undefined}
          >
            {t('relay.healthCheck')}
          </Button>
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

        {/* Intro explainer — without this, users see three sections of URLs
            and no narrative about what they're for or why they'd touch this
            page at all. Default behaviour (no relay configured) and the
            divisional concepts are spelled out once, here. */}
        <Card variant="default" className="border-sky-500/30 bg-sky-500/5 p-3 flex items-start gap-2">
          <Info className="h-4 w-4 mt-0.5 shrink-0 text-sky-400" />
          <div className="space-y-1 text-xs">
            <p className="font-mono text-[10px] uppercase tracking-[0.18em] text-sky-400">
              [ {t('relay.intro.title', '什么是中转站？').toUpperCase()} ]
            </p>
            <p className="text-muted-foreground leading-relaxed">
              {t('relay.intro.body', '中转站 = 一个替代 Anthropic / OpenAI 官方 API 的地址。Switch 把你 CLI 工具的请求转发到这里，由它代理上游模型。')}
            </p>
            <p className="text-muted-foreground leading-relaxed">
              <span className="font-medium text-foreground">{t('relay.intro.default', '不配置 = ')}</span>
              {t('relay.intro.defaultBody', '所有工具走 Lurus 自营 API（hub.lurus.cn）。已经能正常使用就不用动这页。')}
            </p>
            <p className="text-muted-foreground leading-relaxed">
              <span className="font-medium text-foreground">{t('relay.intro.whenUse', '什么时候需要？')}</span>
              {t('relay.intro.whenUseBody', '你买了第三方 / 自己搭了 newapi / OpenRouter 等中转，想让某些工具走那边。')}
            </p>
          </div>
        </Card>

        {/* Zone A: Recommended cloud relays */}
        <section className="space-y-3">
          <div className="flex items-baseline justify-between">
            <div>
              <h3 className="text-sm font-semibold">
                <StepBadge n={1} /> {t('relay.recommended')}
              </h3>
              <p className="text-[11px] text-muted-foreground mt-0.5">
                {t('relay.recommendedHint', 'Lurus 已验证过的第三方中转 — 点「使用」直接加入你的列表。')}
              </p>
            </div>
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
                <Card key={ep.id} variant="default" className="p-3 flex items-center gap-3">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-medium">{ep.name}</span>
                      <span className="font-mono text-[10px] px-1.5 py-0.5 rounded bg-sky-500/15 text-sky-400">
                        {KIND_LABELS[ep.kind] || ep.kind}
                      </span>
                      {ep.latencyMs > 0 && (
                        <span className={cn('font-mono text-[10px] tabular-nums', latencyColor(ep.latencyMs, ep.healthy))}>
                          {ep.latencyMs}ms
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground truncate mt-0.5 font-mono tabular-nums">{ep.url}</p>
                    {ep.description && (
                      <p className="text-xs text-muted-foreground/70 mt-0.5">{ep.description}</p>
                    )}
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleUseCloud(ep)}
                    disabled={saving}
                    className="shrink-0 bg-primary/10 hover:bg-primary/20 text-primary"
                  >
                    {t('relay.useRelay')}
                  </Button>
                </Card>
              ))}
            </div>
          )}
        </section>

        {/* Zone B: Custom endpoints */}
        <section className="space-y-3">
          <div className="flex items-baseline justify-between">
            <div>
              <h3 className="text-sm font-semibold">
                <StepBadge n={2} /> {t('relay.custom')}
              </h3>
              <p className="text-[11px] text-muted-foreground mt-0.5">
                {t('relay.customHint', '你自己买的 API 服务 / 内网中转。需要 URL + （可选）API Key。')}
              </p>
            </div>
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
            <Card variant="recessed" className="p-3 space-y-2">
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="block text-xs text-muted-foreground mb-1 font-mono">{t('relay.nameLabel')}</label>
                  <input
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    placeholder={t('relay.namePlaceholder')}
                    className="w-full px-2 py-1.5 text-xs rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="block text-xs text-muted-foreground mb-1 font-mono">URL *</label>
                  <input
                    value={newUrl}
                    onChange={(e) => setNewUrl(e.target.value)}
                    placeholder="https://api.example.com/v1"
                    className="w-full px-2 py-1.5 text-xs rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary font-mono"
                  />
                </div>
              </div>
              <div>
                <label className="block text-xs text-muted-foreground mb-1 font-mono">{t('relay.apiKeyLabel')}</label>
                <input
                  type="password"
                  value={newKey}
                  onChange={(e) => setNewKey(e.target.value)}
                  placeholder="sk-..."
                  className="w-full px-2 py-1.5 text-xs rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary font-mono"
                />
              </div>
              <div className="flex justify-end gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => { setAddOpen(false); setNewName(''); setNewUrl(''); setNewKey('') }}
                >
                  {t('relay.cancel')}
                </Button>
                <Button
                  size="sm"
                  onClick={handleAdd}
                  disabled={saving || !newName.trim() || !newUrl.trim()}
                  loading={saving}
                  icon={!saving ? <Save className="h-3 w-3" /> : undefined}
                >
                  {t('relay.save')}
                </Button>
              </div>
            </Card>
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
              {customEndpoints.map((ep) => {
                const inUse = usageByEndpoint[ep.id] ?? 0
                return (
                <Card key={ep.id} variant="default" className="p-3 flex items-center gap-3">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-xs font-medium">{ep.name}</span>
                      {ep.latencyMs > 0 ? (
                        <span className={cn('font-mono text-[10px] tabular-nums', latencyColor(ep.latencyMs, ep.healthy))}>
                          {ep.healthy ? <CheckCircle2 className="h-2.5 w-2.5 inline mr-0.5" /> : <WifiOff className="h-2.5 w-2.5 inline mr-0.5" />}
                          {ep.latencyMs}ms
                        </span>
                      ) : null}
                      <CircuitStateChip endpointID={ep.id} />
                      {inUse > 0 ? (
                        <span className="font-mono text-[10px] tabular-nums px-1.5 py-0.5 rounded bg-primary/10 text-primary">
                          {t('relay.inUse', '{{n}} 个工具在用', { n: inUse })}
                        </span>
                      ) : (
                        <span className="font-mono text-[10px] px-1.5 py-0.5 rounded bg-card-recessed text-muted-foreground">
                          {t('relay.idle', '未被使用')}
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground truncate mt-0.5 font-mono tabular-nums">{ep.url}</p>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDelete(ep.id)}
                    disabled={inUse > 0}
                    className="shrink-0 text-red-400/60 hover:text-red-400 hover:bg-red-500/10"
                    title={inUse > 0 ? t('relay.deleteBlockedHint', '先在「工具路由映射」中取消使用，再删除') : t('relay.deleteTitle')}
                    icon={<Trash2 className="h-3.5 w-3.5" />}
                  />
                </Card>
                )
              })}
            </div>
          )}
        </section>

        {/* Routing rules — optional, advanced. Hidden by default behind
            its own collapsible section so the page stays approachable. */}
        <section className="space-y-2 border border-border rounded-lg p-3 bg-card/50">
          <RelayRulesEditor />
        </section>

        {/* Dry-run panel: simulate a request without sending traffic so
            users can validate the rule set before wiring a CLI. */}
        <section className="space-y-2 border border-border rounded-lg p-3 bg-card/50">
          <RouterDryRunPanel />
        </section>

        {/* Zone C: Tool → relay mapping */}
        <section className="space-y-3">
          <div>
            <h3 className="text-sm font-semibold">
              <StepBadge n={3} /> {t('relay.toolRouting')}
            </h3>
            <p className="text-[11px] text-muted-foreground mt-0.5">
              {t('relay.toolRoutingHint', '为每个 CLI 单独指定中转站。留「默认」就走全局上游，多数人不用动。')}
            </p>
          </div>
          <div className="border border-border rounded-lg divide-y divide-border">
            {TOOL_ORDER.map((toolName) => {
              const ep = mapping[toolName] ? endpointById[mapping[toolName]] : undefined
              return (
              <div key={toolName} className="flex items-center justify-between gap-3 px-3 py-2">
                <div className="min-w-0 flex-1">
                  <p className="text-xs font-medium">{TOOL_LABELS[toolName] || toolName}</p>
                  <p className="text-[10px] text-muted-foreground truncate font-mono">
                    {ep ? ep.url : t('relay.usesGlobal', '走全局上游（默认）')}
                  </p>
                </div>
                <select
                  value={mapping[toolName] || ''}
                  onChange={(e) => handleMappingChange(toolName, e.target.value)}
                  className="text-xs px-2 py-1 rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary max-w-[200px] shrink-0"
                >
                  <option value="">{t('relay.useDefault', '默认（全局上游）')}</option>
                  {allEndpointOptions.map((ep) => (
                    <option key={ep.id} value={ep.id}>{ep.name}</option>
                  ))}
                </select>
              </div>
              )
            })}
          </div>

          {/* Apply results */}
          {Object.keys(applyResults).length > 0 && (
            <div className="space-y-1">
              {Object.entries(applyResults).map(([tool, result]) => (
                <div key={tool} className={cn(
                  'flex items-center gap-2 text-xs px-2 py-1 rounded font-mono',
                  result === '' ? 'text-emerald-400 bg-emerald-500/5' : 'text-red-400 bg-red-500/5'
                )}>
                  {result === '' ? <CheckCircle2 className="h-3 w-3" /> : <X className="h-3 w-3" />}
                  {TOOL_LABELS[tool] || tool}: {result === '' ? t('relay.applied') : result}
                </div>
              ))}
            </div>
          )}

          <div className="flex gap-2">
            <Button
              variant="secondary"
              size="sm"
              onClick={handleSaveMapping}
              icon={<Save className="h-3.5 w-3.5" />}
            >
              {t('relay.saveMapping')}
            </Button>
            <Button
              size="sm"
              onClick={handleApplyAll}
              disabled={applying}
              loading={applying}
              icon={!applying ? <Zap className="h-3.5 w-3.5" /> : undefined}
            >
              {t('relay.applyAll')}
            </Button>
          </div>
        </section>
      </div>
    </div>
  )
}
