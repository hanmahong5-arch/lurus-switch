import { forwardRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import {
  User, Bot, Wrench, FileText, GitBranch, Copy, ChevronDown, ChevronRight,
  Settings, CheckCircle2,
} from 'lucide-react'
import type { conversation, audit } from '../../../wailsjs/go/models'
import { DLPHitBadge } from './DLPHitBadge'
import { MarkdownBody } from './MarkdownBody'
import { parseSaneDate, formatAbsolute, stripCommandWrapper } from '../../lib/conversationUtils'
import { cn } from '../../lib/utils'

export type RoleKind = 'user' | 'assistant' | 'system' | 'tool_use' | 'tool_result' | 'meta'

interface RoleStyle {
  border: string
  bg: string
  icon: ReactNode
  i18nKey: string
}

export function roleKindOf(type: string): RoleKind {
  switch (type) {
    case 'user': return 'user'
    case 'assistant': return 'assistant'
    case 'system': return 'system'
    case 'tool_use': return 'tool_use'
    case 'tool_result': return 'tool_result'
    default: return 'meta'
  }
}

export const ROLE_STYLE: Record<RoleKind, RoleStyle> = {
  user:        { border: 'border-l-blue-500',    bg: 'bg-blue-500/5',    icon: <User className="h-4 w-4 text-blue-400" />,           i18nKey: 'conversations.role.user' },
  assistant:   { border: 'border-l-amber-500',   bg: 'bg-amber-500/5',   icon: <Bot className="h-4 w-4 text-amber-400" />,           i18nKey: 'conversations.role.assistant' },
  system:      { border: 'border-l-slate-500',   bg: 'bg-slate-500/5',   icon: <Settings className="h-4 w-4 text-slate-400" />,      i18nKey: 'conversations.role.system' },
  tool_use:    { border: 'border-l-purple-500',  bg: 'bg-purple-500/5',  icon: <Wrench className="h-4 w-4 text-purple-400" />,       i18nKey: 'conversations.role.tool_use' },
  tool_result: { border: 'border-l-emerald-500', bg: 'bg-emerald-500/5', icon: <CheckCircle2 className="h-4 w-4 text-emerald-400" />, i18nKey: 'conversations.role.tool_result' },
  meta:        { border: 'border-l-zinc-600',    bg: 'bg-zinc-500/5',    icon: <FileText className="h-4 w-4 text-zinc-400" />,       i18nKey: 'conversations.role.meta' },
}

interface Props {
  event: conversation.Event
  hits: audit.Entry[]
  onFork?: (uuid: string) => void
  onOpenAuditEntry?: (entryID: string) => void
  forking?: boolean
}

export const MessageCard = forwardRef<HTMLDivElement, Props>(function MessageCard(
  { event, hits, onFork, onOpenAuditEntry, forking }, ref,
) {
  const { t } = useTranslation()
  const role = roleKindOf(event.type)
  const style = ROLE_STYLE[role]

  const isToolRow = role === 'tool_use' || role === 'tool_result'
  const [collapsed, setCollapsed] = useState(isToolRow)

  const wrapper = stripCommandWrapper(event.content)
  const displayBody = wrapper.body
  const ts = parseSaneDate(event.timestamp)
  const time = ts ? formatAbsolute(ts, navigator.language || 'en-US') : ''

  const roleLabel = t(style.i18nKey, defaultRoleLabel(role))
  const subLabel = role === 'tool_use'
    ? `· ${event.toolName || '(unknown)'}`
    : wrapper.stripped && wrapper.label ? `· ${wrapper.label}` : ''

  const totalTokens = (event.inputTokens || 0) + (event.outputTokens || 0)
  const hasTokens = totalTokens > 0 || (event.cacheReadTokens || 0) > 0

  const copy = () => {
    if (!displayBody) return
    void navigator.clipboard.writeText(displayBody)
  }

  return (
    <div
      ref={ref}
      data-role={role}
      data-testid="message-card"
      className={cn(
        'group relative border-l-4 my-2 mx-3 rounded-r-md hover:shadow-sm transition-shadow',
        style.border,
        style.bg,
      )}
    >
      <div className="flex items-start gap-2.5 px-3 py-2.5">
        <div className="mt-0.5 shrink-0">{style.icon}</div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center flex-wrap gap-1.5 text-xs text-muted-foreground mb-1.5">
            <span className="font-semibold text-foreground">{roleLabel}</span>
            {subLabel && <span className="font-mono">{subLabel}</span>}
            {time && <span>· {time}</span>}
            {event.model && (
              <span className="px-1.5 py-0.5 rounded bg-muted text-[10px] font-mono">{event.model}</span>
            )}
            {hasTokens && (
              <span className="text-[10px] font-mono tabular-nums">
                in {event.inputTokens || 0} · out {event.outputTokens || 0}
                {(event.cacheReadTokens || 0) > 0 && <> · cache {event.cacheReadTokens}</>}
              </span>
            )}
            <DLPHitBadge hits={hits} onOpenEntry={onOpenAuditEntry} />
            {isToolRow && (
              <button
                onClick={() => setCollapsed(!collapsed)}
                className="ml-auto text-muted-foreground hover:text-foreground"
                aria-label={collapsed ? 'Expand' : 'Collapse'}
              >
                {collapsed ? <ChevronRight className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
              </button>
            )}
          </div>
          {!collapsed && (
            <>
              {displayBody && <MarkdownBody content={displayBody} />}
              {event.toolArgs && (
                <details className="mt-2">
                  <summary className="text-xs text-muted-foreground cursor-pointer hover:text-foreground select-none">
                    {t('conversations.toolArgs', 'tool args')}
                  </summary>
                  <pre className="mt-1 p-2 bg-muted/40 rounded text-xs overflow-x-auto font-mono">
                    {formatToolArgs(event.toolArgs)}
                  </pre>
                </details>
              )}
            </>
          )}
        </div>
        <div className="opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1 shrink-0">
          {displayBody && (
            <button
              onClick={copy}
              title={t('conversations.copy', 'Copy')}
              aria-label={t('conversations.copy', 'Copy')}
              className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground"
            >
              <Copy className="h-3.5 w-3.5" />
            </button>
          )}
          {event.messageUUID && onFork && (
            <button
              onClick={() => onFork(event.messageUUID!)}
              disabled={forking}
              title={t('conversations.forkHere', 'Fork conversation here')}
              aria-label={t('conversations.forkHere', 'Fork conversation here')}
              className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground disabled:opacity-50"
            >
              <GitBranch className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      </div>
    </div>
  )
})

function defaultRoleLabel(role: RoleKind): string {
  switch (role) {
    case 'user': return 'User'
    case 'assistant': return 'Assistant'
    case 'system': return 'System'
    case 'tool_use': return 'Tool call'
    case 'tool_result': return 'Tool result'
    case 'meta': return 'Meta'
  }
}

function formatToolArgs(args: unknown): string {
  if (typeof args === 'string') return args
  if (Array.isArray(args) && args.every((n) => typeof n === 'number')) {
    try {
      const s = String.fromCharCode(...(args as number[]))
      try { return JSON.stringify(JSON.parse(s), null, 2) } catch { return s }
    } catch { /* fallthrough */ }
  }
  try { return JSON.stringify(args, null, 2) } catch { return String(args) }
}
