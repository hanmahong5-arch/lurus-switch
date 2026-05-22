import { useEffect, useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Plus, Play, Square, Copy, Trash2, Bot, RefreshCw,
  Sparkles, ChevronRight, X as XIcon,
} from 'lucide-react'
import { useAgentStore, type AgentProfile, type CreateAgentParams } from '../stores/agentStore'
import { useToastStore } from '../stores/toastStore'
import { AgentDetailDrawer } from '../components/AgentDetailDrawer'
import { Button, Card, Modal } from '../components/ui'

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
  created: { color: 'text-muted-foreground', bg: 'bg-muted-foreground', label: 'Created' },
  running: { color: 'text-emerald-400', bg: 'bg-emerald-400', label: 'Running' },
  stopped: { color: 'text-amber-400', bg: 'bg-amber-400', label: 'Stopped' },
  error:   { color: 'text-red-400', bg: 'bg-red-400', label: 'Error' },
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
  const isRunning = agent.status === 'running'

  return (
    <Card
      variant="default"
      glow={isRunning}
      className="p-4 hover:border-rule-strong hover:bg-card/60 cursor-pointer group"
      onClick={() => onSelect(agent)}
      title={t('agents.card.openDetail')}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-xl shrink-0">{agent.icon}</span>
          <div className="min-w-0">
            <h3 className="text-sm font-medium leading-tight truncate">{agent.name}</h3>
            <p className="text-xs text-muted-foreground truncate font-mono">
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
            <span key={tag} className="px-1.5 py-0.5 bg-card-recessed rounded text-[10px] text-muted-foreground font-mono">
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
          <Button variant="ghost" size="sm" onClick={() => onAction('launch', agent)} title="Start" className="text-emerald-400">
            <Play className="h-3 w-3" />
          </Button>
        )}
        {agent.status === 'running' && (
          <Button variant="ghost" size="sm" onClick={() => onAction('stop', agent)} title="Stop" className="text-amber-400">
            <Square className="h-3 w-3" />
          </Button>
        )}
        <Button variant="ghost" size="sm" onClick={() => onAction('clone', agent)} title="Clone">
          <Copy className="h-3 w-3" />
        </Button>
        <div className="flex-1" />
        <Button variant="ghost" size="sm" onClick={() => onAction('delete', agent)} title="Delete" className="text-red-400 hover:bg-red-500/10">
          <Trash2 className="h-3 w-3" />
        </Button>
      </div>
    </Card>
  )
}

function CapabilityBanner({ onDismiss }: { onDismiss: () => void }) {
  const { t } = useTranslation()
  return (
    <div className="mx-6 mt-4 rounded-lg border border-primary/30 bg-primary/5 p-4 relative shadow-glow-orange">
      <button
        onClick={onDismiss}
        className="absolute top-2 right-2 p-1 rounded hover:bg-primary/10 text-muted-foreground transition-colors"
        title={t('agents.banner.hide')}
      >
        <XIcon className="h-3.5 w-3.5" />
      </button>
      <div className="flex items-start gap-3">
        <div className="shrink-0 h-9 w-9 rounded-full bg-primary/10 border border-primary/40 flex items-center justify-center">
          <Sparkles className="h-4 w-4 text-primary" />
        </div>
        <div className="flex-1 min-w-0 pr-6">
          <h2 className="font-mono text-[10px] uppercase tracking-[0.18em] text-primary mb-1">
            [ {t('agents.banner.title').toUpperCase()} ]
          </h2>
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
      <Bot className="h-12 w-12 mb-3 text-muted-foreground/40" strokeWidth={1.5} />
      <p className="text-sm font-medium mb-1">{t('agents.empty')}</p>
      <p className="text-xs text-muted-foreground mb-6">{t('agents.howItWorks.title')}</p>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-3 max-w-3xl w-full mb-6">
        {steps.map((s) => (
          <Card key={s.n} variant="elevated" className="p-4">
            <div className="h-6 w-6 rounded-full bg-primary/10 border border-primary/40 text-primary font-mono text-xs font-semibold flex items-center justify-center mb-2 tabular-nums">
              {s.n}
            </div>
            <p className="text-xs text-muted-foreground leading-relaxed">{s.body}</p>
          </Card>
        ))}
      </div>

      <Button size="lg" onClick={onCreate} icon={<Plus className="h-4 w-4" />}>
        {t('agents.createFirst', 'Create your first agent')}
      </Button>
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
    <Modal
      open
      onClose={onClose}
      title={t('agents.createTitle', 'Create Agent')}
      icon={Bot}
      size="md"
      footer={
        <>
          <Button variant="secondary" size="sm" onClick={onClose}>
            {t('ui.cancel', 'Cancel')}
          </Button>
          <Button
            size="sm"
            disabled={!name.trim()}
            onClick={(e) => { e.preventDefault(); handleSubmit(e as any) }}
            icon={<Plus className="h-3.5 w-3.5" />}
          >
            {t('agents.create', 'Create')}
          </Button>
        </>
      }
    >
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
              className="mt-1 w-full px-3 py-2 rounded-md border border-border bg-background text-sm focus:outline-none focus:ring-1 focus:ring-primary"
              maxLength={4}
            />
          </div>
          <div>
            <label className="text-sm text-muted-foreground">{t('agents.tool', 'Tool')}</label>
            <select
              value={toolType}
              onChange={e => setToolType(e.target.value)}
              className="mt-1 w-full px-3 py-2 rounded-md border border-border bg-background text-sm focus:outline-none focus:ring-1 focus:ring-primary"
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
            className="mt-1 w-full px-3 py-2 rounded-md border border-border bg-background text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div>
          <label className="text-sm text-muted-foreground">{t('agents.tags', 'Tags (comma-separated)')}</label>
          <input
            type="text"
            value={tags}
            onChange={e => setTags(e.target.value)}
            placeholder="dev, review, frontend"
            className="mt-1 w-full px-3 py-2 rounded-md border border-border bg-background text-sm focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
      </form>
    </Modal>
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
        <Button onClick={() => setShowCreate(true)} icon={<Plus className="h-4 w-4" />}>
          {t('agents.new', 'New Agent')}
        </Button>
      </div>

      {/* Capability banner — dismissible, persists across reloads via localStorage. */}
      {!bannerHidden && <CapabilityBanner onDismiss={dismissBanner} />}

      {/* Stats bar */}
      <div className="px-6 py-3 border-b border-border flex items-center gap-4 text-xs">
        <span className="text-muted-foreground font-mono">{t('agents.total', 'Total')}: <strong className="tabular-nums">{stats.total}</strong></span>
        <span className="text-emerald-400 font-mono">● <span className="tabular-nums">{stats.running}</span> {t('agents.running', 'running')}</span>
        <span className="text-amber-400 font-mono">● <span className="tabular-nums">{stats.stopped + stats.created}</span> {t('agents.idle', 'idle')}</span>
        {stats.error > 0 && <span className="text-red-400 font-mono">● <span className="tabular-nums">{stats.error}</span> {t('agents.errors', 'errors')}</span>}

        <div className="flex-1" />

        {/* Filter */}
        <select
          value={filter}
          onChange={e => setFilter(e.target.value)}
          className="px-2 py-1 rounded border border-border bg-background text-xs focus:outline-none focus:ring-1 focus:ring-primary"
        >
          <option value="all">{t('agents.filterAll', 'All')}</option>
          <option value="running">{t('agents.filterRunning', 'Running')}</option>
          <option value="stopped">{t('agents.filterStopped', 'Stopped')}</option>
          <option value="created">{t('agents.filterCreated', 'Created')}</option>
          <option value="error">{t('agents.filterError', 'Error')}</option>
        </select>

        <Button
          variant="ghost"
          size="sm"
          onClick={() => { loadAgents(); loadStats() }}
          title={t('ui.refresh', 'Refresh')}
          loading={loading}
          icon={!loading ? <RefreshCw className="h-3.5 w-3.5" /> : undefined}
        />
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

      {/* Create dialog — Modal handles its own mount/unmount via AnimatePresence */}
      {showCreate && <CreateAgentDialog onClose={() => setShowCreate(false)} onCreate={handleCreate} />}

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
