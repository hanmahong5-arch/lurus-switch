import { useEffect, useState, useCallback } from 'react'
import { Search, Package, RefreshCw, CheckCircle, XCircle, Star, Globe, BookMarked } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { Button, Modal } from '../components/ui'
import { useToastStore } from '../stores/toastStore'
import {
  McpMarketList,
  McpMarketRefresh,
  McpMarketInstall,
  McpMarketSavePreset,
} from '../../wailsjs/go/main/App'
import type { mcpmarket } from '../../wailsjs/go/models'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type MarketServer = mcpmarket.MarketServer
type ToolInstallStatus = mcpmarket.ToolInstallStatus

type TabId = 'registry' | 'builtin'
type Category = string

const TARGET_TOOLS: Array<{ id: string; label: string }> = [
  { id: 'claude_code', label: 'Claude Code' },
  { id: 'cursor', label: 'Cursor' },
  { id: 'gemini', label: 'Gemini' },
  { id: 'antigravity', label: 'Antigravity' },
]

// ---------------------------------------------------------------------------
// Server card
// ---------------------------------------------------------------------------

interface ServerCardProps {
  server: MarketServer
  onInstall: (server: MarketServer) => void
}

export function ServerCard({ server, onInstall }: ServerCardProps) {
  const { t } = useTranslation()
  return (
    <div
      className={cn(
        'group flex flex-col gap-2 p-3 rounded-lg border border-border',
        'bg-card hover:border-primary/40 transition-colors',
      )}
      data-testid="server-card"
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-foreground truncate">{server.name}</p>
          {server.qualifiedName && (
            <p className="text-[11px] text-muted-foreground mt-0.5 truncate font-mono">
              {server.qualifiedName}
            </p>
          )}
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          {server.verified && (
            <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-primary/10 text-primary">
              {t('mcpmarket.verified', 'Verified')}
            </span>
          )}
          {server.builtin ? (
            <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-muted text-muted-foreground">
              {t('mcpmarket.builtin', 'Built-in')}
            </span>
          ) : (
            <Globe className="h-3 w-3 text-muted-foreground" aria-label="registry" />
          )}
        </div>
      </div>

      <p className="text-xs text-muted-foreground line-clamp-2 leading-relaxed">
        {server.description}
      </p>

      <div className="flex items-center justify-between mt-1">
        <div className="flex items-center gap-2">
          <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-muted text-muted-foreground font-mono uppercase tracking-wide">
            {t(`mcpmarket.category.${server.category}`, server.category)}
          </span>
          {server.stars > 0 && (
            <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
              <Star className="h-2.5 w-2.5" />
              {server.stars.toLocaleString()}
            </span>
          )}
        </div>
        <Button
          variant="secondary"
          size="sm"
          icon={<Package className="h-3 w-3" />}
          onClick={() => onInstall(server)}
        >
          {t('mcpmarket.install', 'Install')}
        </Button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Install modal
// ---------------------------------------------------------------------------

interface InstallModalProps {
  server: MarketServer | null
  onClose: () => void
  onInstall: (
    server: MarketServer,
    userConfig: Record<string, string>,
    targetTools: string[],
    savePreset: boolean,
  ) => Promise<void>
  installing: boolean
  installStatuses: ToolInstallStatus[]
}

export function InstallModal({
  server,
  onClose,
  onInstall,
  installing,
  installStatuses,
}: InstallModalProps) {
  const { t } = useTranslation()
  const [selectedTools, setSelectedTools] = useState<Set<string>>(
    () => new Set(TARGET_TOOLS.map((t) => t.id)),
  )
  const [configValues, setConfigValues] = useState<Record<string, string>>({})
  const [savePreset, setSavePreset] = useState(false)

  // Reset state when a new server is selected.
  useEffect(() => {
    if (server) {
      setSelectedTools(new Set(TARGET_TOOLS.map((t) => t.id)))
      setConfigValues({})
      setSavePreset(false)
    }
  }, [server?.id])

  if (!server) return null

  const configKeys = server.configSchema?.properties
    ? Object.keys(server.configSchema.properties as Record<string, unknown>)
    : []

  const toggleTool = (id: string) => {
    setSelectedTools((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const canInstall = selectedTools.size > 0 && !installing

  return (
    <Modal
      open={!!server}
      onClose={onClose}
      title={t('mcpmarket.modal.title', 'Install MCP Server')}
      icon={Package}
      size="md"
      footer={
        <div className="flex items-center justify-end gap-2 p-4 border-t border-border">
          <Button variant="ghost" size="sm" onClick={onClose} disabled={installing}>
            {t('mcpmarket.modal.cancel', 'Cancel')}
          </Button>
          <Button
            variant="primary"
            size="sm"
            loading={installing}
            disabled={!canInstall}
            onClick={() =>
              onInstall(server, configValues, Array.from(selectedTools), savePreset)
            }
          >
            {t('mcpmarket.modal.confirm', 'Install')}
          </Button>
        </div>
      }
    >
      <div className="p-4 space-y-4">
        <p className="text-xs text-muted-foreground">
          {t('mcpmarket.modal.desc', 'Select which AI coding tools to configure.')}
        </p>

        {/* Target tool checkboxes */}
        <div>
          <label className="block text-xs font-medium text-foreground mb-2">
            {t('mcpmarket.modal.targetTools', 'Target tools')}
          </label>
          <div className="grid grid-cols-2 gap-1.5" data-testid="tool-checkboxes">
            {TARGET_TOOLS.map((tool) => (
              <label
                key={tool.id}
                className={cn(
                  'flex items-center gap-2 px-3 py-2 rounded-md border cursor-pointer select-none',
                  'text-xs transition-colors',
                  selectedTools.has(tool.id)
                    ? 'border-primary bg-primary/10 text-primary'
                    : 'border-border text-muted-foreground hover:border-primary/40',
                )}
              >
                <input
                  type="checkbox"
                  checked={selectedTools.has(tool.id)}
                  onChange={() => toggleTool(tool.id)}
                  className="h-3 w-3 accent-primary"
                  data-testid={`tool-checkbox-${tool.id}`}
                />
                {tool.label}
              </label>
            ))}
          </div>
        </div>

        {/* Config schema inputs */}
        {configKeys.length > 0 ? (
          <div>
            <label className="block text-xs font-medium text-foreground mb-2">
              {t('mcpmarket.modal.config', 'Server configuration')}
            </label>
            <div className="space-y-2">
              {configKeys.map((key) => {
                const fieldDef = (server.configSchema?.properties as Record<string, { description?: string }>)[key]
                return (
                  <div key={key}>
                    <label className="block text-[11px] text-muted-foreground mb-1">
                      {key}
                      {fieldDef?.description && (
                        <span className="ml-1 text-muted-foreground/60">— {fieldDef.description}</span>
                      )}
                    </label>
                    <input
                      type="text"
                      value={configValues[key] ?? ''}
                      onChange={(e) =>
                        setConfigValues((prev) => ({ ...prev, [key]: e.target.value }))
                      }
                      placeholder={key}
                      data-testid={`config-input-${key}`}
                      className={cn(
                        'w-full h-7 px-2 text-xs rounded-md border border-border',
                        'bg-input text-foreground placeholder:text-muted-foreground',
                        'focus:outline-none focus:ring-1 focus:ring-primary',
                      )}
                    />
                  </div>
                )
              })}
            </div>
          </div>
        ) : (
          <p className="text-xs text-muted-foreground italic">
            {t('mcpmarket.modal.noConfig', 'No configuration required for this server.')}
          </p>
        )}

        {/* Save as preset toggle */}
        <label className="flex items-center gap-2 cursor-pointer select-none">
          <input
            type="checkbox"
            checked={savePreset}
            onChange={(e) => setSavePreset(e.target.checked)}
            className="h-3.5 w-3.5 rounded accent-primary"
            data-testid="save-preset-checkbox"
          />
          <span className="text-xs text-muted-foreground">
            {t('mcpmarket.modal.savePreset', 'Save as preset for future use')}
          </span>
        </label>

        {/* Per-tool install statuses (shown after install attempt) */}
        {installStatuses.length > 0 && (
          <div>
            <p className="text-xs font-medium text-foreground mb-2">
              {t('mcpmarket.modal.toolStatus', 'Tool install results')}
            </p>
            <div className="space-y-1" data-testid="install-statuses">
              {installStatuses.map((st) => (
                <div
                  key={st.tool}
                  className="flex items-center gap-2 text-xs"
                  data-testid={`install-status-${st.tool}`}
                >
                  {st.ok ? (
                    <CheckCircle className="h-3.5 w-3.5 text-green-500 shrink-0" />
                  ) : (
                    <XCircle className="h-3.5 w-3.5 text-destructive shrink-0" />
                  )}
                  <span className={st.ok ? 'text-foreground' : 'text-destructive'}>
                    {st.tool}
                  </span>
                  {!st.ok && st.error && (
                    <span className="text-muted-foreground truncate">{st.error}</span>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </Modal>
  )
}

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------

export function McpMarketPage() {
  const { t } = useTranslation()
  const addToast = useToastStore((s) => s.addToast)

  const [servers, setServers] = useState<MarketServer[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [tab, setTab] = useState<TabId>('builtin')
  const [category, setCategory] = useState<Category>('all')
  const [search, setSearch] = useState('')
  const [installing, setInstalling] = useState(false)
  const [selectedServer, setSelectedServer] = useState<MarketServer | null>(null)
  const [installStatuses, setInstallStatuses] = useState<ToolInstallStatus[]>([])

  const loadServers = useCallback(async () => {
    setLoading(true)
    try {
      const list = await McpMarketList()
      setServers(list || [])
    } catch {
      addToast('error', t('mcpmarket.installFailed', 'Install failed'))
    } finally {
      setLoading(false)
    }
  }, [addToast, t])

  useEffect(() => {
    loadServers()
  }, [loadServers])

  // Derive categories from loaded servers.
  const allCategories: Category[] = ['all', ...Array.from(new Set(servers.map((s) => s.category))).sort()]

  const filtered = servers.filter((s) => {
    const isBuiltin = s.builtin
    if (tab === 'builtin' && !isBuiltin) return false
    if (tab === 'registry' && isBuiltin) return false
    if (category !== 'all' && s.category !== category) return false
    const q = search.toLowerCase()
    if (q === '') return true
    return (
      s.name.toLowerCase().includes(q) ||
      s.description.toLowerCase().includes(q) ||
      (s.qualifiedName ?? '').toLowerCase().includes(q) ||
      s.category.toLowerCase().includes(q)
    )
  })

  const handleRefresh = async () => {
    setRefreshing(true)
    try {
      const result = await McpMarketRefresh('')
      if (!result.success) {
        addToast('error', result.message || t('mcpmarket.refreshFailed', 'Refresh failed'))
      } else {
        addToast('success', t('mcpmarket.refreshed', 'Registry updated'))
        await loadServers()
      }
    } catch (err) {
      addToast('error', err instanceof Error ? err.message : t('mcpmarket.refreshFailed', 'Refresh failed'))
    } finally {
      setRefreshing(false)
    }
  }

  const handleInstall = async (
    server: MarketServer,
    userConfig: Record<string, string>,
    targetTools: string[],
    savePreset: boolean,
  ) => {
    setInstalling(true)
    setInstallStatuses([])
    try {
      const result = await McpMarketInstall(server.id, userConfig, targetTools)
      if (result.statuses) {
        setInstallStatuses(result.statuses)
      }
      if (!result.success) {
        addToast('error', result.message || t('mcpmarket.installFailed', 'Install failed'))
      } else {
        addToast('success', t('mcpmarket.installed', 'MCP server installed'))
      }

      if (savePreset) {
        const presetResult = await McpMarketSavePreset(server.id, userConfig)
        if (!presetResult.success) {
          addToast('error', presetResult.message || t('mcpmarket.presetFailed', 'Failed to save preset'))
        } else {
          addToast('success', t('mcpmarket.presetSaved', 'Preset saved'))
        }
      }

      if (result.success) {
        setSelectedServer(null)
      }
    } catch (err) {
      addToast('error', err instanceof Error ? err.message : t('mcpmarket.installFailed', 'Install failed'))
    } finally {
      setInstalling(false)
    }
  }

  return (
    <div className="h-full flex overflow-hidden">
      {/* Sidebar: tabs + categories */}
      <div className="w-44 border-r border-border bg-card-recessed flex flex-col shrink-0">
        <div className="p-3 border-b border-border">
          <h2 className="text-sm font-semibold flex items-center gap-2">
            <Package className="h-4 w-4 text-primary" />
            {t('mcpmarket.title', 'MCP Market')}
          </h2>
        </div>

        {/* Source tabs */}
        <div className="p-2 border-b border-border flex gap-1">
          {(['builtin', 'registry'] as TabId[]).map((tid) => (
            <button
              key={tid}
              onClick={() => setTab(tid)}
              className={cn(
                'flex-1 text-[11px] py-1 rounded-md transition-colors',
                tab === tid
                  ? 'bg-primary/15 text-primary font-medium'
                  : 'text-muted-foreground hover:text-foreground',
              )}
              data-testid={`tab-${tid}`}
            >
              {t(`mcpmarket.tab.${tid}`, tid)}
            </button>
          ))}
        </div>

        {/* Category filter */}
        <nav className="p-2 space-y-0.5 overflow-y-auto flex-1">
          {allCategories.map((cat) => {
            const active = category === cat
            return (
              <button
                key={cat}
                onClick={() => setCategory(cat)}
                className={cn(
                  'w-full text-left px-3 py-1.5 rounded-md transition-all duration-150 text-sm',
                  active
                    ? 'bg-primary/15 text-primary border-l-2 border-l-primary font-mono text-xs tracking-[0.06em]'
                    : 'text-muted-foreground hover:bg-muted hover:text-foreground',
                )}
              >
                {active
                  ? `[ ${t(`mcpmarket.category.${cat}`, cat).toUpperCase()} ]`
                  : t(`mcpmarket.category.${cat}`, cat)}
              </button>
            )
          })}
        </nav>
      </div>

      {/* Main content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Search + refresh bar */}
        <div className="p-3 border-b border-border shrink-0 flex gap-2 items-center">
          <div className="relative flex-1">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('mcpmarket.search', 'Search MCP servers…')}
              className={cn(
                'w-full h-8 pl-8 pr-3 text-xs rounded-md border border-border',
                'bg-input text-foreground placeholder:text-muted-foreground',
                'focus:outline-none focus:ring-1 focus:ring-primary',
              )}
            />
          </div>
          <Button
            variant="ghost"
            size="sm"
            icon={<RefreshCw className={cn('h-3.5 w-3.5', refreshing && 'animate-spin')} />}
            loading={refreshing}
            onClick={handleRefresh}
            title={t('mcpmarket.refresh', 'Refresh')}
          >
            {t('mcpmarket.refresh', 'Refresh')}
          </Button>
        </div>

        {/* Server grid */}
        <div className="flex-1 overflow-y-auto p-3">
          {loading ? (
            <div className="flex items-center justify-center h-32 text-xs text-muted-foreground">
              {t('mcpmarket.loading', 'Loading servers…')}
            </div>
          ) : filtered.length === 0 ? (
            <div className="flex items-center justify-center h-32 text-xs text-muted-foreground">
              {t('mcpmarket.noResults', 'No servers match your search.')}
            </div>
          ) : (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-2.5">
              {filtered.map((server) => (
                <ServerCard
                  key={server.id}
                  server={server}
                  onInstall={(s) => {
                    setInstallStatuses([])
                    setSelectedServer(s)
                  }}
                />
              ))}
            </div>
          )}
        </div>

        {/* Footer: unique cross-CLI pitch */}
        <div className="p-2 border-t border-border shrink-0 flex items-center gap-2 text-[10px] text-muted-foreground">
          <BookMarked className="h-3 w-3 text-primary shrink-0" />
          {t('mcpmarket.subtitle', 'Install once across Claude Code, Cursor, Gemini, and Antigravity.')}
        </div>
      </div>

      {/* Install modal */}
      <InstallModal
        server={selectedServer}
        onClose={() => setSelectedServer(null)}
        onInstall={handleInstall}
        installing={installing}
        installStatuses={installStatuses}
      />
    </div>
  )
}
