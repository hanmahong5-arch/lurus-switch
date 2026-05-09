import { useEffect, useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Plus, Play, Square, Copy, Trash2, Bot, RefreshCw,
  Sparkles, ChevronRight, X as XIcon,
} from 'lucide-react'
import { useAgentStore, type AgentProfile, type CreateAgentParams } from '../stores/agentStore'
import { useToastStore } from '../stores/toastStore'
import { AgentDetailDrawer } from '../components/AgentDetailDrawer'

const TOOL_OPTIONS = [
  { value: 'claude', label: 'Claude Code', icon: '🟣' },
  { value: 'codex', label: 'Codex', icon: '🟢' },
  { value: 'gemini', label: 'Gemini CLI', icon: '🔵' },
  { value: 'openclaw', label: 'OpenClaw', icon: '🦞' },
  { value: 'zeroclaw', label: 'ZeroClaw', icon: '⚡' },
  { value: 'picoclaw', label: 'PicoClaw', icon: '🔸' },
  { value: 'nullclaw', label: 'NullClaw', icon: '⬛' },
]

const STATUS_CONFIG: Record<string, { color: string; bg: string; label: string }> = {
  created: { color: 'text-gray-400', bg: 'bg-gray-400', label: 'Created' },
  running: { color: 'text-green-500', bg: 'bg-green-500', label: 'Running' },
  stopped: { color: 'text-yellow-500', bg: 'bg-yellow-500', label: 'Stopped' },
  error:   { color: 'text-red-500', bg: 'bg-red-500', label: 'Error' },
}

const POLL_INTERVAL_MS = 5_000
const BANNER_DISMISS_KEY = 'lurus-switch-agents-banner-dismissed'

function StatusBadge({ status }: { status: string }) {
  const cfg = STATUS_CONFIG[status] || STATUS_CONFIG.created
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className={`h-2 w-2 rounded-full ${cfg.bg}`} />
      <span className={`text-xs ${cfg.color}`}>{cfg.label}</span>
    </span>
  )
}

function AgentCard({ agent, onAction, onSelect }: {
  agent: AgentProfile
  onAction: (action: string, agent: AgentProfile) => void
  onSelect: (agent: AgentProfile) => void
}) {
  const { t } = useTranslation()
  const toolInfo = TOOL_OPTIONS.find(t => t.value === agent.toolType)

  return (
    <div
      className="rounded-lg border border-border bg-card p-4 hover:border-primary/50 transition-colors cursor-pointer group"
      onClick={() => onSelect(agent)}
      title={t('agents.card.openDetail')}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-xl shrink-0">{agent.icon}</span>
          <div className="min-w-0">
            <h3 className="text-sm font-medium leading-tight truncate">{agent.name}</h3>
            <p className="text-xs text-muted-foreground truncate">
              {toolInfo?.icon} {toolInfo?.label || agent.toolType} · {agent.modelId}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <StatusBadge status={agent.status} />
          <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/50 group-hover:text-primary transition-colors" />
        </div>
      </div>

      {agent.tags.length > 0 && (
        <div className="flex flex-wrap gap-1 mb-3">
          {agent.tags.map(tag => (
            <span key={tag} className="px-1.5 py-0.5 bg-muted rounded text-[10px] text-muted-foreground">
              {tag}
            </span>
          ))}
        </div>
      )}

      <div
        className="flex items-center gap-1 pt-2 border-t border-border"
        onClick={(e) => e.stopPropagation()}
      >
        {(agent.status === 'created' || agent.status === 'stopped' || agent.status === 'error') && (
          <button
            onClick={() => onAction('launch', agent)}
            className="flex items-center gap-1 px-2 py-1 text-xs rounded hover:bg-muted transition-colors text-green-600"
            title="Start"
          >
            <Play className="h-3 w-3" />
          </button>
        )}
        {agent.status === 'running' && (
          <button
            onClick={() => onAction('stop', agent)}
            className="flex items-center gap-1 px-2 py-1 text-xs rounded hover:bg-muted transition-colors text-yellow-600"
            title="Stop"
          >
            <Square className="h-3 w-3" />
          </button>
        )}
        <button
          onClick={() => onAction('clone', agent)}
          className="flex items-center gap-1 px-2 py-1 text-xs rounded hover:bg-muted transition-colors text-muted-foreground"
          title="Clone"
        >
          <Copy className="h-3 w-3" />
        </button>
        <div className="flex-1" />
        <button
          onClick={() => onAction('delete', agent)}
          className="flex items-center gap-1 px-2 py-1 text-xs rounded hover:bg-destructive/10 transition-colors text-destructive"
          title="Delete"
        >
          <Trash2 className="h-3 w-3" />
        </button>
      </div>
    </div>
  )
}

function CapabilityBanner({ onDismiss }: { onDismiss: () => void }) {
  const { t } = useTranslation()
  return (
    <div className="mx-6 mt-4 rounded-lg border border-primary/20 bg-primary/5 p-4 relative">
      <button
        onClick={onDismiss}
        className="absolute top-2 right-2 p-1 rounded hover:bg-primary/10 text-muted-foreground"
        title={t('agents.banner.hide')}
      >
        <XIcon className="h-3.5 w-3.5" />
      </button>
      <div className="flex items-start gap-3">
        <div className="shrink-0 h-9 w-9 rounded-full bg-primary/10 border border-primary/30 flex items-center justify-center">
          <Sparkles className="h-4 w-4 text-primary" />
        </div>
        <div className="flex-1 min-w-0 pr-6">
          <h2 className="text-sm font-semibold mb-1">{t('agents.banner.title')}</h2>
          <p className="text-xs text-muted-foreground leading-relaxed">{t('agents.banner.body')}</p>
        </div>
      </div>
    </div>
  )
}

function HowItWorks({ onCreate }: { onCreate: () => void }) {
  const { t } = useTranslation()
  const steps = [
    { n: 1, body: t('agents.howItWorks.step1') },
    { n: 2, body: t('agents.howItWorks.step2') },
    { n: 3, body: t('agents.howItWorks.step3') },
  ]
  return (
    <div className="flex flex-col items-center justify-center py-10 px-6">
      <Bot className="h-12 w-12 mb-3 text-muted-foreground/40" />
      <p className="text-sm font-medium mb-1">{t('agents.empty')}</p>
      <p className="text-xs text-muted-foreground mb-6">{t('agents.howItWorks.title')}</p>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-3 max-w-3xl w-full mb-6">
        {steps.map((s) => (
          <div key={s.n} className="rounded-md border border-border bg-card p-4">
            <div className="h-6 w-6 rounded-full bg-primary/10 border border-primary/30 text-primary text-xs font-semibold flex items-center justify-center mb-2">
              {s.n}
            </div>
            <p className="text-xs text-muted-foreground leading-relaxed">{s.body}</p>
          </div>
        ))}
      </div>

      <button
        onClick={onCreate}
        className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md text-sm hover:bg-primary/90 transition-colors"
      >
        <Plus className="h-4 w-4" />
        {t('agents.createFirst', 'Create your first agent')}
      </button>
    </div>
  )
}

function CreateAgentDialog({ onClose, onCreate }: {
  onClose: () => void
  onCreate: (params: CreateAgentParams) => void
}) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [icon, setIcon] = useState('🤖')
  const [toolType, setToolType] = useState('claude')
  const [modelId, setModelId] = useState('claude-sonnet-4-6')
  const [tags, setTags] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    onCreate({
      name: name.trim(),
      icon,
      toolType,
      modelId,
      tags: tags ? tags.split(',').map(t => t.trim()).filter(Boolean) : [],
    })
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div className="bg-card border border-border rounded-xl p-6 w-[420px] shadow-lg" onClick={e => e.stopPropagation()}>
        <h2 className="text-lg font-semibold mb-4">{t('agents.createTitle', 'Create Agent')}</h2>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="text-sm text-muted-foreground">{t('agents.name', 'Name')}</label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder={t('agents.namePlaceholder', 'e.g. Frontend Reviewer')}
              className="mt-1 w-full px-3 py-2 rounded-md border border-border bg-background text-sm focus:outline-none focus:ring-1 focus:ring-primary"
              autoFocus
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-sm text-muted-foreground">{t('agents.icon', 'Icon')}</label>
              <input
                type="text"
                value={icon}
                onChange={e => setIcon(e.target.value)}
                className="mt-1 w-full px-3 py-2 rounded-md border border-border bg-background text-sm"
                maxLength={4}
              />
            </div>
            <div>
              <label className="text-sm text-muted-foreground">{t('agents.tool', 'Tool')}</label>
              <select
                value={toolType}
                onChange={e => setToolType(e.target.value)}
                className="mt-1 w-full px-3 py-2 rounded-md border border-border bg-background text-sm"
              >
                {TOOL_OPTIONS.map(t => (
                  <option key={t.value} value={t.value}>{t.icon} {t.label}</option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <label className="text-sm text-muted-foreground">{t('agents.model', 'Model')}</label>
            <input
              type="text"
              value={modelId}
              onChange={e => setModelId(e.target.value)}
              className="mt-1 w-full px-3 py-2 rounded-md border border-border bg-background text-sm"
            />
          </div>

          <div>
            <label className="text-sm text-muted-foreground">{t('agents.tags', 'Tags (comma-separated)')}</label>
            <input
              type="text"
              value={tags}
              onChange={e => setTags(e.target.value)}
              placeholder="dev, review, frontend"
              className="mt-1 w-full px-3 py-2 rounded-md border border-border bg-background text-sm"
            />
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm rounded-md border border-border hover:bg-muted transition-colors"
            >
              {t('ui.cancel', 'Cancel')}
            </button>
            <button
              type="submit"
              disabled={!name.trim()}
              className="px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              {t('agents.create', 'Create')}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export function AgentsPage() {
  const { t } = useTranslation()
  const {
    agents, stats, loading,
    loadAgents, loadStats, createAgent, deleteAgent, launchAgent, stopAgent, cloneAgent,
  } = useAgentStore()
  const addToast = useToastStore(s => s.addToast)
  const [showCreate, setShowCreate] = useState(false)
  const [filter, setFilter] = useState<string>('all')
  const [selectedAgent, setSelectedAgent] = useState<AgentProfile | null>(null)
  const [bannerHidden, setBannerHidden] = useState<boolean>(() => {
    try { return localStorage.getItem(BANNER_DISMISS_KEY) === '1' } catch { return false }
  })

  // Polling — pauses when the document is hidden so a backgrounded
  // window doesn't keep firing IPC at the Wails backend.
  useEffect(() => {
    let cancelled = false
    const tick = () => {
      if (cancelled || document.hidden) return
      loadAgents()
      loadStats()
    }
    tick()
    const handle = setInterval(tick, POLL_INTERVAL_MS)
    const onVisibility = () => {
      if (!document.hidden) tick()
    }
    document.addEventListener('visibilitychange', onVisibility)
    return () => {
      cancelled = true
      clearInterval(handle)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [loadAgents, loadStats])

  // Keep selectedAgent in sync with the live agent list (so the drawer
  // reflects status changes after launch/stop without a manual reopen).
  useEffect(() => {
    if (!selectedAgent) return
    const fresh = agents.find((a) => a.id === selectedAgent.id) ?? null
    if (fresh && fresh !== selectedAgent) setSelectedAgent(fresh)
    if (!fresh) setSelectedAgent(null) // deleted
  }, [agents, selectedAgent])

  const filteredAgents = filter === 'all'
    ? agents
    : agents.filter(a => a.status === filter)

  const handleAction = useCallback(async (action: string, agent: AgentProfile) => {
    try {
      switch (action) {
        case 'launch':
          await launchAgent(agent.id)
          addToast('success', t('agents.started', { name: agent.name }))
          break
        case 'stop':
          await stopAgent(agent.id)
          addToast('success', t('agents.stopped', { name: agent.name }))
          break
        case 'clone':
          await cloneAgent(agent.id, `${agent.name} (copy)`)
          addToast('success', t('agents.cloned', { name: agent.name }))
          break
        case 'delete':
          if (confirm(t('agents.confirmDelete', { name: agent.name }))) {
            await deleteAgent(agent.id)
            addToast('success', t('agents.deleted', { name: agent.name }))
          }
          break
      }
    } catch (e: any) {
      addToast('error', e?.message || String(e))
    }
  }, [addToast, cloneAgent, deleteAgent, launchAgent, stopAgent, t])

  const handleCreate = useCallback(async (params: CreateAgentParams) => {
    try {
      await createAgent(params)
      setShowCreate(false)
      addToast('success', t('agents.created', 'Agent created'))
    } catch (e: any) {
      addToast('error', e?.message || String(e))
    }
  }, [addToast, createAgent, t])

  const dismissBanner = useCallback(() => {
    setBannerHidden(true)
    try { localStorage.setItem(BANNER_DISMISS_KEY, '1') } catch {}
  }, [])

  const isEmpty = agents.length === 0

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* Header */}
      <div className="px-6 py-4 border-b border-border flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold flex items-center gap-2">
            <Bot className="h-5 w-5" />
            {t('agents.title', 'Agents')}
          </h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            {t('agents.subtitle', 'Manage your AI assistant fleet')}
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md text-sm hover:bg-primary/90 transition-colors"
        >
          <Plus className="h-4 w-4" />
          {t('agents.new', 'New Agent')}
        </button>
      </div>

      {/* Capability banner — dismissible, persists across reloads via localStorage. */}
      {!bannerHidden && <CapabilityBanner onDismiss={dismissBanner} />}

      {/* Stats bar */}
      <div className="px-6 py-3 border-b border-border flex items-center gap-4 text-xs">
        <span className="text-muted-foreground">{t('agents.total', 'Total')}: <strong>{stats.total}</strong></span>
        <span className="text-green-500">● {stats.running} {t('agents.running', 'running')}</span>
        <span className="text-yellow-500">● {stats.stopped + stats.created} {t('agents.idle', 'idle')}</span>
        {stats.error > 0 && <span className="text-red-500">● {stats.error} {t('agents.errors', 'errors')}</span>}

        <div className="flex-1" />

        {/* Filter */}
        <select
          value={filter}
          onChange={e => setFilter(e.target.value)}
          className="px-2 py-1 rounded border border-border bg-background text-xs"
        >
          <option value="all">{t('agents.filterAll', 'All')}</option>
          <option value="running">{t('agents.filterRunning', 'Running')}</option>
          <option value="stopped">{t('agents.filterStopped', 'Stopped')}</option>
          <option value="created">{t('agents.filterCreated', 'Created')}</option>
          <option value="error">{t('agents.filterError', 'Error')}</option>
        </select>

        <button
          onClick={() => { loadAgents(); loadStats() }}
          className="p-1 rounded hover:bg-muted transition-colors"
          title={t('ui.refresh', 'Refresh')}
        >
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
        </button>
      </div>

      {/* Agent grid */}
      <div className="flex-1 overflow-y-auto">
        {isEmpty ? (
          <HowItWorks onCreate={() => setShowCreate(true)} />
        ) : filteredAgents.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground p-6">
            <Bot className="h-12 w-12 mb-3 opacity-30" />
            <p className="text-sm">{t('agents.empty', 'No agents yet')}</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 p-6">
            {filteredAgents.map(agent => (
              <AgentCard
                key={agent.id}
                agent={agent}
                onAction={handleAction}
                onSelect={setSelectedAgent}
              />
            ))}
          </div>
        )}
      </div>

      {/* Create dialog */}
      {showCreate && (
        <CreateAgentDialog onClose={() => setShowCreate(false)} onCreate={handleCreate} />
      )}

      {/* Detail drawer */}
      <AgentDetailDrawer
        agent={selectedAgent}
        onClose={() => setSelectedAgent(null)}
        onLaunch={(a) => handleAction('launch', a)}
        onStop={(a) => handleAction('stop', a)}
        onClone={(a) => handleAction('clone', a)}
      />
    </div>
  )
}
