import { useEffect, useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  MessageSquare, RefreshCw, Search, Filter, Download, ShieldAlert,
  FileText, Loader2, X,
} from 'lucide-react'
import { useConversationStore } from '../stores/conversationStore'
import { useToastStore } from '../stores/toastStore'
import { Timeline } from '../components/conversation/Timeline'
import { ProjectContextDrawer } from '../components/conversation/ProjectContextDrawer'
import { ForkTreeView } from '../components/conversation/ForkTreeView'
import { cn } from '../lib/utils'
import { Button } from '../components/ui'
import {
  sortByRecencyDesc, conversationDate, bucketOf, bucketLabel, formatRelative,
  type Bucket,
} from '../lib/conversationUtils'

const TOOL_OPTIONS = ['', 'claude', 'codex', 'gemini']

export function ConversationsPage() {
  const { t, i18n } = useTranslation()
  const isZh = (i18n.language || '').startsWith('zh')
  const toast = useToastStore((s) => s.addToast)
  const {
    conversations, active, filter, loading, loadingActive, reindexing,
    forking, error, dlpHits, contextFiles,
    list, open, setFilter, reindex, exportSession, fork, clearActive,
  } = useConversationStore()

  const [activeSel, setActiveSel] = useState<{ tool: string; sessionID: string } | null>(null)
  const [showContext, setShowContext] = useState(false)

  useEffect(() => { void list() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Facet: unique models present in the index so the dropdown surfaces
  // only what the user actually has, not a hardcoded list.
  const models = useMemo(() => {
    const s = new Set<string>()
    for (const c of conversations) if (c.model) s.add(c.model)
    return Array.from(s).sort()
  }, [conversations])

  // Defensive client-side sort + bucketing. Backend already sorts in
  // Rebuild but the index can drift between rebuilds (new sessions
  // append in mtime order). Cheap on the ~hundreds of rows users have.
  const grouped = useMemo(() => {
    const sorted = sortByRecencyDesc(conversations)
    const now = new Date()
    const groups: Array<{ bucket: Bucket; rows: typeof sorted }> = []
    let current: Bucket | null = null
    for (const row of sorted) {
      const b = bucketOf(conversationDate(row), now)
      if (b !== current) {
        groups.push({ bucket: b, rows: [] })
        current = b
      }
      groups[groups.length - 1].rows.push(row)
    }
    return groups
  }, [conversations])

  const onOpen = (tool: string, sessionID: string) => {
    setActiveSel({ tool, sessionID })
    void open(tool, sessionID)
  }

  const onExport = async (format: 'markdown' | 'json') => {
    if (!activeSel) return
    const p = await exportSession(activeSel.tool, activeSel.sessionID, format, true)
    if (p) toast('success', t('conversations.exportSuccess', 'Exported to {{path}}', { path: p }))
  }

  const onFork = async (uuid: string) => {
    if (!activeSel) return
    const res = await fork(activeSel.tool, activeSel.sessionID, uuid)
    if (res) {
      toast('success', t('conversations.forkSuccess', 'New session {{sid}} ({{n}} messages)', { sid: res.newSessionID, n: res.messagesKept }))
    }
  }

  const onReindex = async () => {
    const r = await reindex()
    if (r) toast('success', t('conversations.reindexed', 'Indexed {{n}} sessions ({{a}} new)', { n: r.scanned, a: r.added }))
  }

  return (
    <div className="flex h-full overflow-hidden">
      {/* LEFT: filterable session list */}
      <aside className="w-80 border-r border-border flex flex-col bg-muted/20">
        <div className="p-3 border-b border-border flex items-center gap-2">
          <MessageSquare className="h-4 w-4 text-primary" />
          <h2 className="text-sm font-semibold flex-1">{t('conversations.title', '会话浏览 · Conversations')}</h2>
          <Button
            variant="ghost"
            size="sm"
            onClick={onReindex}
            disabled={reindexing}
            loading={reindexing}
            title={t('conversations.reindex', 'Reindex')}
            icon={!reindexing ? <RefreshCw className="h-3.5 w-3.5" /> : undefined}
          />
        </div>

        {/* Search + facets */}
        <div className="p-2 space-y-2 border-b border-border">
          <div className="relative">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <input
              type="text"
              value={filter.search}
              onChange={(e) => setFilter({ search: e.target.value })}
              placeholder={t('conversations.searchPlaceholder', 'Search session ID or project path…')}
              className="w-full pl-7 pr-2 py-1.5 text-xs rounded bg-background border border-border focus:border-primary outline-none"
            />
          </div>
          <div className="grid grid-cols-2 gap-1.5">
            <select
              value={filter.tool}
              onChange={(e) => setFilter({ tool: e.target.value })}
              className="text-xs rounded bg-background border border-border px-1.5 py-1"
            >
              {TOOL_OPTIONS.map((o) => (
                <option key={o} value={o}>{o || t('conversations.filterAnyTool', 'Any tool')}</option>
              ))}
            </select>
            <select
              value={filter.model}
              onChange={(e) => setFilter({ model: e.target.value })}
              className="text-xs rounded bg-background border border-border px-1.5 py-1"
            >
              <option value="">{t('conversations.filterAnyModel', 'Any model')}</option>
              {models.map((m) => <option key={m} value={m}>{m}</option>)}
            </select>
          </div>
          <label className="flex items-center gap-2 text-xs text-muted-foreground cursor-pointer">
            <input
              type="checkbox"
              checked={filter.onlyDLPHits}
              onChange={(e) => setFilter({ onlyDLPHits: e.target.checked })}
              className="rounded"
            />
            <ShieldAlert className="h-3 w-3 text-red-400" />
            {t('conversations.onlyDLPHits', 'Only sessions with DLP hits')}
          </label>
        </div>

        {/* Session list */}
        <div className="flex-1 overflow-y-auto">
          {loading && (
            <div className="p-3 text-xs text-muted-foreground flex items-center gap-2">
              <Loader2 className="h-3 w-3 animate-spin" />
              {t('conversations.loading', 'Loading…')}
            </div>
          )}
          {!loading && conversations.length === 0 && (
            <div className="p-3 text-xs text-muted-foreground">
              {t('conversations.empty', 'No sessions yet. Run claude / codex / gemini in any project, then reindex.')}
            </div>
          )}
          {grouped.map((g) => (
            <div key={g.bucket}>
              <div className="sticky top-0 z-10 px-3 py-1.5 bg-card-recessed/95 backdrop-blur font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground border-b border-border">
                [ {bucketLabel(g.bucket, isZh).toUpperCase()} ] <span className="tabular-nums">· {g.rows.length}</span>
              </div>
              {g.rows.map((c) => {
                const isActive = activeSel?.tool === c.tool && activeSel?.sessionID === c.sessionID
                const when = conversationDate(c)
                const relative = when ? formatRelative(when, new Date(), isZh) : (isZh ? '时间未知' : 'unknown')
                return (
                  <button
                    key={c.tool + c.sessionID}
                    onClick={() => onOpen(c.tool, c.sessionID)}
                    className={cn(
                      'w-full text-left px-3 py-2 border-b border-border/40 hover:bg-muted/50 transition-colors',
                      isActive && 'bg-primary/10 border-l-2 border-l-primary',
                    )}
                  >
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">{c.tool}</span>
                      {c.hasDLPHits && <ShieldAlert className="h-3 w-3 text-red-400" />}
                      {c.parentSessionID && <span className="font-mono text-[10px] text-muted-foreground">▸ fork</span>}
                      <span className="ml-auto font-mono text-[10px] text-muted-foreground tabular-nums">{relative}</span>
                    </div>
                    <div className="text-xs font-mono text-foreground truncate tabular-nums">{c.sessionID.slice(0, 16)}…</div>
                    <div className="text-[11px] text-muted-foreground truncate">{c.cwd || '—'}</div>
                    <div className="font-mono text-[10px] text-muted-foreground mt-0.5 tabular-nums">
                      {c.messageCount} msg · {c.totalTokens || 0} tok
                      {c.model && <> · {c.model}</>}
                    </div>
                  </button>
                )
              })}
            </div>
          ))}
        </div>
      </aside>

      {/* RIGHT: Timeline */}
      <main className="flex-1 flex flex-col overflow-hidden">
        {error && (
          <div className="p-2 text-xs text-red-400 bg-red-500/10 border-b border-red-500/30 font-mono">{error}</div>
        )}
        {!activeSel && (
          <div className="flex-1 flex items-center justify-center text-sm text-muted-foreground">
            {t('conversations.pickSession', 'Pick a session on the left to view the transcript.')}
          </div>
        )}
        {activeSel && loadingActive && (
          <div className="flex-1 flex items-center justify-center">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        )}
        {activeSel && !loadingActive && active && (
          <>
            {/* Header */}
            <div className="px-4 py-2 border-b border-border flex items-center gap-2">
              <div className="flex-1 min-w-0">
                <div className="text-sm font-semibold truncate">
                  <span className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground mr-1.5">[ {active.meta.tool.toUpperCase()} ]</span>
                  <span className="font-mono tabular-nums">{active.meta.sessionID}</span>
                </div>
                <div className="text-xs text-muted-foreground truncate font-mono">
                  {active.meta.cwd || '—'}
                  {active.meta.model && <> · {active.meta.model}</>}
                  {' · '}<span className="tabular-nums">{active.meta.messageCount}</span> msg · <span className="tabular-nums">{active.meta.totalTokens || 0}</span> tok
                </div>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowContext((v) => !v)}
                title={t('conversations.contextDrawer', 'Project context')}
                icon={<FileText className="h-3.5 w-3.5" />}
              />
              <Button
                variant="secondary"
                size="sm"
                onClick={() => onExport('markdown')}
                icon={<Download className="h-3 w-3" />}
              >
                MD
              </Button>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => onExport('json')}
                icon={<Download className="h-3 w-3" />}
              >
                JSON
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => { clearActive(); setActiveSel(null) }}
                title="Close"
                icon={<X className="h-3.5 w-3.5" />}
              />
            </div>

            <ForkTreeView current={active.meta} siblings={conversations} onJump={onOpen} />

            <div className="flex-1 flex overflow-hidden">
              <div className="flex-1 overflow-y-auto">
                <Timeline
                  events={active.events || []}
                  dlpHits={dlpHits}
                  onFork={onFork}
                  forking={forking}
                />
              </div>
              {showContext && (
                <aside className="w-80 border-l border-border bg-muted/10">
                  <ProjectContextDrawer files={contextFiles} onClose={() => setShowContext(false)} />
                </aside>
              )}
            </div>
          </>
        )}
      </main>
    </div>
  )
}
