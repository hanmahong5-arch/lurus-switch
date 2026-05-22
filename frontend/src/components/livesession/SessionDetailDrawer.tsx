import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  X, ExternalLink, Bot, MessageSquare, Wrench, Zap,
  AlertCircle, ChevronRight, ChevronDown, DollarSign, Hash, Cpu,
  FolderOpen,
} from 'lucide-react'
import { BrowserOpenURL } from '../../../wailsjs/runtime'
import {
  getSessionTranscript,
  type LiveSession,
  type TranscriptEvent,
  type TranscriptEventType,
} from '../../lib/liveSessionApi'
import { iconForTranscriptType } from './eventIcons'
import { cn } from '../../lib/utils'

// SessionDetailDrawer — full transcript view for a single live session.
//
// Sliding from the right with a dimmed backdrop, matching the convention
// of BashGuardModal / SnapshotModal / AgentDetailDrawer. The drawer's
// content re-fetches whenever the underlying session's `lastActivity`
// advances, so a live session being actively appended-to keeps the
// drawer in sync without the user reopening it.
//
// Backend caps the response at the last 500 events (see
// bindings_livesession.go). The header surfaces "showing latest N" when
// that cap is plausibly hit so the user knows older events were elided.

interface Props {
  session: LiveSession | null
  onClose: () => void
}

// CONTENT_LINE_CLAMP is the visual cap for `whitespace-pre-wrap` content
// blocks before the user clicks "expand". Tuned so a typical 3-paragraph
// assistant reply still fits the card without scrolling.
const CONTENT_LINE_CLAMP = 10

// The backend caps at this many events; we tell the user when we hit it.
const TRANSCRIPT_CAP = 500

export function SessionDetailDrawer({ session, onClose }: Props) {
  const { t } = useTranslation()
  const [events, setEvents] = useState<TranscriptEvent[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const bodyRef = useRef<HTMLDivElement | null>(null)
  const open = session !== null

  // ESC closes. Mirror the AgentDetailDrawer / Radix Dialog convention so
  // the keyboard behaviour is consistent across the app.
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [open, onClose])

  // Fetch transcript whenever either the open session changes OR its
  // `lastActivity` advances (the live store pushes a new object every
  // poll cycle so referential identity changes; keying on the timestamp
  // string is what keeps drawer + card in sync as events arrive).
  const transcriptPath = session?.transcriptPath ?? ''
  const lastActivity = session?.lastActivity ?? ''

  const fetchTranscript = useCallback(async () => {
    if (!transcriptPath) return
    setLoading(true)
    try {
      const ev = await getSessionTranscript(transcriptPath)
      setEvents(ev ?? [])
      setError(null)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setError(msg)
    } finally {
      setLoading(false)
    }
  }, [transcriptPath])

  useEffect(() => {
    if (!open) {
      setEvents([])
      setError(null)
      return
    }
    void fetchTranscript()
    // Re-fetch is gated on transcriptPath + lastActivity — see comment above.
  }, [open, transcriptPath, lastActivity, fetchTranscript])

  // Auto-scroll to bottom on open AND when new events arrive — but only if
  // the user is already near the bottom, so manual scrolling up to read
  // older events isn't yanked away on the next push.
  useEffect(() => {
    const el = bodyRef.current
    if (!el) return
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 120
    if (nearBottom) el.scrollTop = el.scrollHeight
  }, [events])

  const openTranscriptFile = () => {
    if (!session) return
    // Wails BrowserOpenURL on Windows happily opens file:// URLs via the
    // OS shell (notepad/code/etc. depending on the user's default handler).
    const encoded = session.transcriptPath
      .replace(/\\/g, '/')
      .split('/')
      .map((seg) => encodeURIComponent(seg))
      .join('/')
    BrowserOpenURL('file:///' + encoded.replace(/^\//, ''))
  }

  if (!session) return null

  const totalTokens = session.inputTokens + session.outputTokens
  const reachedCap = events.length >= TRANSCRIPT_CAP

  return (
    <div
      className="fixed inset-0 z-40 flex justify-end bg-black/30 backdrop-blur-sm"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
    >
      <div
        className="relative w-full max-w-[640px] bg-card border-l border-border shadow-2xl flex flex-col h-full"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="px-5 py-4 border-b border-border flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <div className="flex items-baseline gap-2">
              <h2 className="text-base font-semibold truncate">{session.projectName}</h2>
              <span className="text-[10px] uppercase tracking-wider text-muted-foreground/70 flex-shrink-0">
                {session.tool}
              </span>
            </div>
            <p className="text-[11px] text-muted-foreground font-mono truncate mt-0.5 flex items-center gap-1">
              <FolderOpen className="h-3 w-3 flex-shrink-0" />
              {session.cwd}
            </p>
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1 mt-2 text-[11px]">
              <span className="inline-flex items-center gap-1 text-muted-foreground">
                <Cpu className="h-3 w-3" />
                <span className="font-mono text-foreground/80">{shortModel(session.model)}</span>
              </span>
              <span className="inline-flex items-center gap-1 text-muted-foreground">
                <Hash className="h-3 w-3" />
                <span className="font-mono text-foreground/80 tabular-nums">{fmtTok(totalTokens)}</span>
              </span>
              <span className="inline-flex items-center gap-1 text-muted-foreground">
                <DollarSign className="h-3 w-3" />
                <span className="font-mono text-foreground/80 tabular-nums">${session.estimatedUsd.toFixed(3)}</span>
              </span>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded hover:bg-muted text-muted-foreground flex-shrink-0"
            title={t('ui.close', '关闭')}
            aria-label={t('ui.close', '关闭')}
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Status row */}
        <div className="px-5 py-2 border-b border-border/60 flex items-center justify-between text-[11px] text-muted-foreground">
          <div className="flex items-center gap-2">
            {loading && (
              <span className="inline-flex items-center gap-1">
                <span className="h-1.5 w-1.5 rounded-full bg-blue-500 animate-pulse" />
                {t('live.drawerLoading', '加载中…')}
              </span>
            )}
            {!loading && (
              <span>{t('live.drawerEventCount', '共 {{n}} 条事件', { n: events.length })}</span>
            )}
            {reachedCap && (
              <span className="text-amber-500" title={t('live.drawerCapHint', '后端最多返回最近 500 条；更早事件请打开完整 JSONL')}>
                {t('live.drawerCap', '· 已截断至最近 500 条')}
              </span>
            )}
          </div>
          {error && (
            <span className="text-red-500 flex items-center gap-1 truncate max-w-[60%]" title={error}>
              <AlertCircle className="h-3 w-3 flex-shrink-0" />
              {error}
            </span>
          )}
        </div>

        {/* Body */}
        <div
          ref={bodyRef}
          className="flex-1 overflow-y-auto px-5 py-3 space-y-2"
        >
          {events.length === 0 && !loading && !error && (
            <p className="text-xs text-muted-foreground italic py-6 text-center">
              {t('live.drawerEmpty', '尚无事件 — 等待 CLI 写入第一条消息')}
            </p>
          )}
          {events.map((ev, i) => (
            <TranscriptRow key={`${ev.messageUUID || 'noid'}:${i}`} event={ev} />
          ))}
        </div>

        {/* Footer */}
        <div className="px-5 py-2.5 border-t border-border flex items-center justify-between gap-2">
          <span
            className="text-[10px] text-muted-foreground font-mono truncate flex-1"
            title={session.transcriptPath}
          >
            {session.transcriptPath}
          </span>
          <button
            onClick={openTranscriptFile}
            className="flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-border text-xs hover:bg-muted transition-colors flex-shrink-0"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            {t('live.drawerOpenJsonl', '查看完整 JSONL')}
          </button>
        </div>
      </div>
    </div>
  )
}

// --- Row renderer -----------------------------------------------------

function TranscriptRow({ event }: { event: TranscriptEvent }) {
  const Icon = iconForTranscriptType(event.type)
  const [expanded, setExpanded] = useState(false)

  const meta = STYLE_FOR_TYPE[event.type] ?? STYLE_FOR_TYPE.meta
  const ts = useMemo(() => formatTs(event.timestamp), [event.timestamp])
  const tsTitle = event.timestamp || ''

  const content = event.content?.trim() ?? ''
  const isLongContent = content.split('\n').length > CONTENT_LINE_CLAMP || content.length > 600

  return (
    <div className={cn('rounded-md border px-3 py-2', meta.shell)}>
      <div className="flex items-center gap-2 text-[11px] mb-1.5">
        <Icon className={cn('h-3.5 w-3.5 flex-shrink-0', meta.iconColor)} />
        <span className={cn('font-medium uppercase tracking-wider text-[10px]', meta.label)}>
          {meta.title}
        </span>
        {event.model && (
          <span className="text-muted-foreground font-mono">{shortModel(event.model)}</span>
        )}
        <span className="ml-auto text-muted-foreground tabular-nums" title={tsTitle}>{ts}</span>
      </div>

      {event.type === 'tool_use' ? (
        <ToolUseBody event={event} />
      ) : event.type === 'tool_result' ? (
        <ToolResultBody event={event} />
      ) : content ? (
        <div className="text-[12px] text-foreground/90">
          <pre
            className={cn(
              'whitespace-pre-wrap break-words font-sans',
              !expanded && isLongContent && `line-clamp-${CONTENT_LINE_CLAMP}`,
            )}
            style={!expanded && isLongContent ? { display: '-webkit-box', WebkitBoxOrient: 'vertical', WebkitLineClamp: CONTENT_LINE_CLAMP, overflow: 'hidden' } : undefined}
          >
            {content}
          </pre>
          {isLongContent && (
            <button
              onClick={() => setExpanded((v) => !v)}
              className="mt-1 text-[10px] text-muted-foreground hover:text-foreground flex items-center gap-0.5"
            >
              {expanded
                ? (<><ChevronDown className="h-3 w-3 rotate-180" />收起</>)
                : (<><ChevronRight className="h-3 w-3" />展开全部</>)}
            </button>
          )}
        </div>
      ) : (
        <p className="text-[11px] text-muted-foreground italic">(无文本内容)</p>
      )}

      {/* Token usage chip — only assistant events carry it. */}
      {(event.inputTokens || event.outputTokens) ? (
        <div className="mt-1.5 text-[10px] text-muted-foreground font-mono tabular-nums">
          ↑{event.inputTokens ?? 0} ↓{event.outputTokens ?? 0}
        </div>
      ) : null}
    </div>
  )
}

function ToolUseBody({ event }: { event: TranscriptEvent }) {
  const args = summariseArgs(event.toolArgs)
  return (
    <div className="text-[12px]">
      <div className="flex items-center gap-2 mb-1">
        <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-amber-500/10 border border-amber-500/30 text-amber-600 dark:text-amber-400 text-[11px] font-mono">
          <Wrench className="h-3 w-3" />
          {event.toolName || '(unnamed tool)'}
        </span>
      </div>
      {args && (
        <pre className="whitespace-pre-wrap break-words font-mono text-[11px] text-foreground/80 line-clamp-6"
             style={{ display: '-webkit-box', WebkitBoxOrient: 'vertical', WebkitLineClamp: 6, overflow: 'hidden' }}>
          {args}
        </pre>
      )}
    </div>
  )
}

function ToolResultBody({ event }: { event: TranscriptEvent }) {
  const c = (event.content ?? '').trim()
  if (!c) {
    return <p className="text-[11px] text-muted-foreground italic">(空结果)</p>
  }
  const truncated = c.length > 1200 ? c.slice(0, 1200) + '\n…(已截断)' : c
  return (
    <pre className="whitespace-pre-wrap break-words font-mono text-[11px] text-foreground/70 line-clamp-8"
         style={{ display: '-webkit-box', WebkitBoxOrient: 'vertical', WebkitLineClamp: 8, overflow: 'hidden' }}>
      {truncated}
    </pre>
  )
}

// --- Styling per event type ------------------------------------------

interface TypeStyle {
  shell: string
  label: string
  iconColor: string
  title: string
}

const STYLE_FOR_TYPE: Record<TranscriptEventType, TypeStyle> = {
  user: {
    shell: 'border-border bg-card',
    label: 'text-blue-500',
    iconColor: 'text-blue-500',
    title: '用户',
  },
  assistant: {
    shell: 'border-border bg-muted/30',
    label: 'text-foreground/80',
    iconColor: 'text-foreground/70',
    title: '助手',
  },
  tool_use: {
    shell: 'border-amber-500/30 bg-amber-500/5',
    label: 'text-amber-500',
    iconColor: 'text-amber-500',
    title: '工具调用',
  },
  tool_result: {
    shell: 'border-border/60 bg-muted/10 opacity-90',
    label: 'text-emerald-500',
    iconColor: 'text-emerald-500',
    title: '工具结果',
  },
  system: {
    shell: 'border-border bg-muted/20',
    label: 'text-amber-500',
    iconColor: 'text-amber-500',
    title: '系统',
  },
  meta: {
    shell: 'border-border/40 bg-card/60',
    label: 'text-muted-foreground',
    iconColor: 'text-muted-foreground',
    title: '元数据',
  },
}

// --- Formatters -------------------------------------------------------

function fmtTok(n: number): string {
  if (!n) return '0'
  if (n < 1000) return String(n)
  if (n < 1_000_000) return (n / 1000).toFixed(1) + 'k'
  return (n / 1_000_000).toFixed(2) + 'M'
}

function shortModel(model?: string): string {
  if (!model) return ''
  const m = model.toLowerCase()
  const stripped = m.replace(/^claude-/, '').replace(/-2\d{7}.*$/, '')
  return stripped || model
}

// summariseArgs renders a tool_use's `input` payload as a compact string.
// `toolArgs` arrives as the parsed JSON value (object, array, string …)
// because the Go binding marshals it back into the response. We collapse
// the common Bash / Edit / Read shapes inline because they read better as
// "cmd=ls -al" than as `{"command":"ls -al"}`; everything else falls back
// to pretty-printed JSON.
function summariseArgs(args: unknown): string {
  if (args == null) return ''
  if (typeof args === 'string') return args
  if (typeof args !== 'object') return String(args)
  const obj = args as Record<string, unknown>
  if (typeof obj.command === 'string') return 'cmd: ' + obj.command
  if (typeof obj.file_path === 'string') {
    const path = String(obj.file_path)
    if (typeof obj.pattern === 'string') return `path: ${path}\npattern: ${obj.pattern}`
    if (typeof obj.old_string === 'string') return `path: ${path}\nedit (delta)`
    return 'path: ' + path
  }
  if (typeof obj.pattern === 'string') return 'pattern: ' + obj.pattern
  try {
    return JSON.stringify(args, null, 2)
  } catch {
    return String(args)
  }
}

function formatTs(iso?: string): string {
  if (!iso) return ''
  const t = new Date(iso).getTime()
  if (!Number.isFinite(t) || t === 0) return ''
  const diff = Date.now() - t
  if (diff < 0) return '刚刚'
  const s = Math.floor(diff / 1000)
  if (s < 5) return '刚刚'
  if (s < 60) return `${s}s 前`
  if (s < 3600) return `${Math.floor(s / 60)}m 前`
  if (s < 86400) return `${Math.floor(s / 3600)}h 前`
  return new Date(iso).toLocaleString()
}
