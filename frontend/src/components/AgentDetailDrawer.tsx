import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import {
  X, Play, Square, Copy as CopyIcon, RefreshCw,
  Terminal, ShieldCheck, Settings as SettingsIcon, Folder,
} from 'lucide-react'
import type { AgentProfile } from '../stores/agentStore'
import { GetAgentOutput } from '../../wailsjs/go/main/App'
import { cn } from '../lib/utils'
import { formatLocal } from '../lib/formatTime'

const OUTPUT_POLL_MS = 2_000
const OUTPUT_MAX_LINES = 200

interface Props {
  agent: AgentProfile | null
  onClose: () => void
  onLaunch: (agent: AgentProfile) => void | Promise<void>
  onStop: (agent: AgentProfile) => void | Promise<void>
  onClone: (agent: AgentProfile) => void | Promise<void>
}

export function AgentDetailDrawer({ agent, onClose, onLaunch, onStop, onClone }: Props) {
  const { t } = useTranslation()
  const [output, setOutput] = useState<string[]>([])
  const [outputLoading, setOutputLoading] = useState(false)
  const [outputError, setOutputError] = useState<string | null>(null)
  const outputBoxRef = useRef<HTMLDivElement | null>(null)
  const isRunning = agent?.status === 'running'

  // Live output: poll while drawer is open AND tab is visible. Stops as soon
  // as the agent isn't running so we don't burn IPC for nothing.
  useEffect(() => {
    if (!agent) {
      setOutput([])
      setOutputError(null)
      return
    }
    let cancelled = false

    const pull = async () => {
      try {
        const lines = await GetAgentOutput(agent.id, OUTPUT_MAX_LINES)
        if (!cancelled) {
          setOutput(lines || [])
          setOutputError(null)
        }
      } catch (e: any) {
        if (!cancelled) setOutputError(e?.message || String(e))
      }
    }

    pull()
    if (!isRunning) return

    const handle = setInterval(() => {
      if (document.hidden) return
      pull()
    }, OUTPUT_POLL_MS)
    return () => {
      cancelled = true
      clearInterval(handle)
    }
  }, [agent?.id, isRunning])

  // Auto-scroll to bottom when output grows.
  useEffect(() => {
    const el = outputBoxRef.current
    if (!el) return
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80
    if (nearBottom) el.scrollTop = el.scrollHeight
  }, [output])

  if (!agent) return null

  const refresh = async () => {
    setOutputLoading(true)
    try {
      const lines = await GetAgentOutput(agent.id, OUTPUT_MAX_LINES)
      setOutput(lines || [])
      setOutputError(null)
    } catch (e: any) {
      setOutputError(e?.message || String(e))
    } finally {
      setOutputLoading(false)
    }
  }

  const perms = agent.permissions || { allowShell: false, allowFiles: false, allowNetwork: false }

  return (
    <div className="fixed inset-0 z-40 flex justify-end" onClick={onClose}>
      <div className="absolute inset-0 bg-black/40" />
      <div
        className="relative w-full max-w-2xl bg-card border-l border-border shadow-2xl flex flex-col h-full"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="px-5 py-4 border-b border-border flex items-start justify-between">
          <div className="flex items-center gap-3 min-w-0">
            <span className="text-2xl shrink-0">{agent.icon}</span>
            <div className="min-w-0">
              <h2 className="text-base font-semibold truncate">{agent.name}</h2>
              <p className="text-xs text-muted-foreground truncate">
                {agent.toolType} · {agent.modelId}
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded hover:bg-muted text-muted-foreground"
            title={t('ui.close', 'Close')}
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Action bar */}
        <div className="px-5 py-2.5 border-b border-border flex items-center gap-2">
          {isRunning ? (
            <button
              onClick={() => onStop(agent)}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-md border border-yellow-500/30 text-yellow-600 hover:bg-yellow-500/10"
            >
              <Square className="h-3.5 w-3.5" /> {t('agents.filterStopped', 'Stop')}
            </button>
          ) : (
            <button
              onClick={() => onLaunch(agent)}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-md border border-green-500/30 text-green-600 hover:bg-green-500/10"
            >
              <Play className="h-3.5 w-3.5" /> {t('agents.filterRunning', 'Start')}
            </button>
          )}
          <button
            onClick={() => onClone(agent)}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-md border border-border hover:bg-muted text-muted-foreground"
          >
            <CopyIcon className="h-3.5 w-3.5" /> Clone
          </button>
          <div className="flex-1" />
          <span className="text-[10px] text-muted-foreground tabular-nums">
            ID {agent.id.slice(0, 8)}
          </span>
        </div>

        {/* Content (scrolls) */}
        <div className="flex-1 overflow-y-auto px-5 py-4 space-y-5 text-sm">
          {/* Config */}
          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex items-center gap-1.5">
              <SettingsIcon className="h-3 w-3" /> {t('agents.detail.config')}
            </h3>
            <div className="rounded-md border border-border bg-background/50 divide-y divide-border/50">
              <Field label={t('agents.tags', 'Tags')}>
                {agent.tags.length === 0 ? (
                  <span className="text-muted-foreground text-xs">—</span>
                ) : (
                  <div className="flex flex-wrap gap-1">
                    {agent.tags.map((tag) => (
                      <span key={tag} className="px-1.5 py-0.5 bg-muted rounded text-[10px]">{tag}</span>
                    ))}
                  </div>
                )}
              </Field>
              <Field label={t('agents.detail.systemPrompt')}>
                {agent.systemPrompt ? (
                  <div className="text-xs whitespace-pre-wrap font-mono text-muted-foreground max-h-32 overflow-y-auto">
                    {agent.systemPrompt}
                  </div>
                ) : (
                  <span className="text-muted-foreground text-xs">{t('agents.detail.noPrompt')}</span>
                )}
              </Field>
              <Field label={t('agents.detail.tools')}>
                {agent.mcpServers && agent.mcpServers.length > 0 ? (
                  <div className="flex flex-wrap gap-1">
                    {agent.mcpServers.map((mcp) => (
                      <span key={mcp} className="px-1.5 py-0.5 bg-muted rounded text-[10px]">{mcp}</span>
                    ))}
                  </div>
                ) : (
                  <span className="text-muted-foreground text-xs">—</span>
                )}
              </Field>
              <Field label={t('agents.detail.permissions')}>
                <div className="flex flex-wrap gap-3 text-xs">
                  <PermBadge label={t('agents.detail.shell')} on={perms.allowShell} t={t} />
                  <PermBadge label={t('agents.detail.files')} on={perms.allowFiles} t={t} />
                  <PermBadge label={t('agents.detail.network')} on={perms.allowNetwork} t={t} />
                </div>
              </Field>
              <Field label={t('agents.detail.budget')}>
                <div className="text-xs space-y-0.5">
                  <div>
                    <span className="text-muted-foreground">{t('agents.detail.tokenBudget')}: </span>
                    {agent.budgetLimitTokens
                      ? <span className="tabular-nums">{agent.budgetLimitTokens.toLocaleString()}</span>
                      : <span className="text-muted-foreground">{t('agents.detail.noBudget')}</span>}
                  </div>
                  <div>
                    <span className="text-muted-foreground">{t('agents.detail.currencyBudget')}: </span>
                    {agent.budgetLimitCurrency
                      ? <span className="tabular-nums">${agent.budgetLimitCurrency.toFixed(2)}</span>
                      : <span className="text-muted-foreground">{t('agents.detail.noBudget')}</span>}
                  </div>
                  {(agent.budgetLimitTokens || agent.budgetLimitCurrency) && (
                    <div className="text-muted-foreground">
                      {t('agents.detail.period')}: {agent.budgetPeriod || '—'} · {t('agents.detail.policy')}: {agent.budgetPolicy || '—'}
                    </div>
                  )}
                </div>
              </Field>
              {agent.configDir && (
                <Field label={<span className="flex items-center gap-1.5"><Folder className="h-3 w-3" />{t('agents.detail.configDir')}</span>}>
                  <code className="text-[10px] font-mono break-all text-muted-foreground">{agent.configDir}</code>
                </Field>
              )}
            </div>
            <div className="text-[10px] text-muted-foreground mt-1.5 px-1">
              {t('agents.detail.createdAt')}: {formatTime(agent.createdAt)}
              {' · '}
              {t('agents.detail.updatedAt')}: {formatTime(agent.updatedAt)}
            </div>
          </section>

          {/* Output */}
          <section>
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-1.5">
                <Terminal className="h-3 w-3" /> {t('agents.detail.output')}
                <span className="text-[10px] text-muted-foreground/70 normal-case font-normal">
                  ({t('agents.detail.outputLines', { count: output.length })})
                </span>
              </h3>
              <button
                onClick={refresh}
                disabled={outputLoading}
                className="p-1 rounded hover:bg-muted text-muted-foreground disabled:opacity-50"
                title={t('agents.detail.refresh')}
              >
                <RefreshCw className={cn('h-3 w-3', outputLoading && 'animate-spin')} />
              </button>
            </div>
            <div
              ref={outputBoxRef}
              className="rounded-md border border-border bg-zinc-950 text-zinc-100 font-mono text-[11px] leading-relaxed p-3 h-64 overflow-y-auto"
            >
              {outputError ? (
                <div className="text-red-400">{outputError}</div>
              ) : output.length === 0 ? (
                <div className="text-zinc-500 italic">{t('agents.detail.noOutput')}</div>
              ) : (
                output.map((line, i) => (
                  <div key={i} className="whitespace-pre-wrap break-all">{line}</div>
                ))
              )}
            </div>
          </section>
        </div>
      </div>
    </div>
  )
}

function Field({ label, children }: { label: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="px-3 py-2 grid grid-cols-[100px_1fr] gap-3 items-start">
      <div className="text-[11px] text-muted-foreground pt-0.5">{label}</div>
      <div className="min-w-0">{children}</div>
    </div>
  )
}

function PermBadge({ label, on, t }: { label: string; on: boolean; t: TFunction }) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px]',
        on
          ? 'border border-emerald-500/30 text-emerald-600 bg-emerald-500/5'
          : 'border border-border text-muted-foreground bg-muted/30'
      )}
    >
      {on && <ShieldCheck className="h-2.5 w-2.5" />}
      {label}: {on ? t('agents.detail.allowed') : t('agents.detail.denied')}
    </span>
  )
}

function formatTime(iso: string): string {
  if (!iso) return '—'
  return formatLocal(iso)
}

