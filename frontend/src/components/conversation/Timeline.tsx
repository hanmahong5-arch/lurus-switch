import { useState, useMemo } from 'react'
import { User, Bot, Wrench, FileText, GitBranch, Copy, ChevronDown, ChevronRight } from 'lucide-react'
import type { conversation, audit } from '../../../wailsjs/go/models'
import { DLPHitBadge } from './DLPHitBadge'
import { parseSaneDate, formatAbsolute, stripCommandWrapper } from '../../lib/conversationUtils'

interface Props {
  events: conversation.Event[]
  dlpHits: audit.Entry[]
  onFork?: (messageUUID: string) => void
  onOpenAuditEntry?: (entryID: string) => void
  forking?: boolean
}

// Timeline renders a session's events top-down. Tool_use / tool_result
// rows collapse by default — they're noisy and most readers care about
// the user/assistant text first.
export function Timeline({ events, dlpHits, onFork, onOpenAuditEntry, forking }: Props) {
  const hitsByMessage = useMemo(() => {
    const map = new Map<string, audit.Entry[]>()
    for (const h of dlpHits) {
      const uuid = h.metadata?.conv_message_uuid
      if (!uuid) continue
      const cur = map.get(uuid) || []
      cur.push(h)
      map.set(uuid, cur)
    }
    return map
  }, [dlpHits])

  if (!events || events.length === 0) {
    return <div className="text-sm text-muted-foreground p-6">No messages in this session.</div>
  }

  return (
    <div className="divide-y divide-border">
      {events.map((ev, i) => (
        <TimelineRow
          key={ev.messageUUID || i}
          event={ev}
          hits={hitsByMessage.get(ev.messageUUID || '') || []}
          onFork={onFork}
          onOpenAuditEntry={onOpenAuditEntry}
          forking={forking}
        />
      ))}
    </div>
  )
}

function TimelineRow({
  event, hits, onFork, onOpenAuditEntry, forking,
}: {
  event: conversation.Event
  hits: audit.Entry[]
  onFork?: (uuid: string) => void
  onOpenAuditEntry?: (entryID: string) => void
  forking?: boolean
}) {
  // Tool rows collapse by default; user/assistant rows always expand.
  const isToolRow = event.type === 'tool_use' || event.type === 'tool_result'
  const [collapsed, setCollapsed] = useState(isToolRow)

  const icon = roleIcon(event.type)
  const roleLabel = roleText(event.type, event.toolName)
  const ts = parseSaneDate(event.timestamp)
  const time = ts ? formatAbsolute(ts, navigator.language || 'en-US') : ''

  // Meta rows often hold just an XML wrapper (e.g. `<command-name>usage`);
  // unwrap so the timeline shows the inner text instead of "<command-name>"
  // noise stacked vertically.
  const wrapper = stripCommandWrapper(event.content)
  const displayBody = wrapper.body
  const displayLabel = wrapper.stripped && wrapper.label ? `${roleLabel} · ${wrapper.label}` : roleLabel

  const copy = () => {
    if (!displayBody) return
    void navigator.clipboard.writeText(displayBody)
  }

  return (
    <div className="group relative px-4 py-3 hover:bg-muted/30">
      <div className="flex items-start gap-3">
        <div className="mt-0.5 text-muted-foreground">{icon}</div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 text-xs text-muted-foreground mb-1">
            <span className="font-semibold text-foreground">{displayLabel}</span>
            {time && <span>· {time}</span>}
            {event.model && <span className="px-1.5 py-0.5 rounded bg-muted text-[10px]">{event.model}</span>}
            {(event.inputTokens || event.outputTokens) ? (
              <span className="text-[10px]">in {event.inputTokens || 0} / out {event.outputTokens || 0}</span>
            ) : null}
            <DLPHitBadge hits={hits} onOpenEntry={onOpenAuditEntry} />
            {isToolRow && (
              <button onClick={() => setCollapsed(!collapsed)} className="ml-auto text-muted-foreground hover:text-foreground">
                {collapsed ? <ChevronRight className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
              </button>
            )}
          </div>
          {!collapsed && (
            <>
              {displayBody && (
                <pre className="whitespace-pre-wrap break-words text-sm font-mono text-foreground/90">
                  {displayBody}
                </pre>
              )}
              {event.toolArgs && (
                <pre className="mt-2 p-2 bg-muted/40 rounded text-xs overflow-x-auto">
                  {typeof event.toolArgs === 'string' ? event.toolArgs : JSON.stringify(event.toolArgs, null, 2)}
                </pre>
              )}
            </>
          )}
        </div>
        <div className="opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1">
          {displayBody && (
            <button
              onClick={copy}
              title="Copy"
              className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground"
            >
              <Copy className="h-3.5 w-3.5" />
            </button>
          )}
          {event.messageUUID && onFork && (
            <button
              onClick={() => onFork(event.messageUUID!)}
              disabled={forking}
              title="Fork conversation here"
              className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground disabled:opacity-50"
            >
              <GitBranch className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

function roleIcon(type: string) {
  switch (type) {
    case 'user': return <User className="h-4 w-4" />
    case 'assistant': return <Bot className="h-4 w-4" />
    case 'tool_use':
    case 'tool_result': return <Wrench className="h-4 w-4" />
    default: return <FileText className="h-4 w-4" />
  }
}

function roleText(type: string, toolName?: string): string {
  switch (type) {
    case 'user': return 'User'
    case 'assistant': return 'Assistant'
    case 'tool_use': return `tool_use · ${toolName || '(unknown)'}`
    case 'tool_result': return 'tool_result'
    case 'system': return 'System'
    default: return type
  }
}
