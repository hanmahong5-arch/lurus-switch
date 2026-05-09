import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Bot, ChevronRight, Plus } from 'lucide-react'
import { useAgentStore } from '../stores/agentStore'
import { useConfigStore } from '../stores/configStore'

// Compact home-page entry for the Agents page. Auto-loads stats so the
// card numbers reflect reality even before the user opens the page. Polling
// is light (one request on mount) — the Agents page itself is the place
// for live polling.
export function AgentFleetCard() {
  const { t } = useTranslation()
  const stats = useAgentStore((s) => s.stats)
  const loadStats = useAgentStore((s) => s.loadStats)
  const setActiveTool = useConfigStore((s) => s.setActiveTool)

  useEffect(() => {
    loadStats()
  }, [loadStats])

  const total = stats.total
  const running = stats.running
  const empty = total === 0

  const goToAgents = () => setActiveTool('agents')

  return (
    <div
      onClick={goToAgents}
      className="rounded-lg border border-border bg-card hover:border-primary/40 transition-colors cursor-pointer p-4 flex items-center gap-4 group"
    >
      <div className="h-10 w-10 rounded-full bg-primary/10 border border-primary/30 flex items-center justify-center shrink-0">
        <Bot className="h-5 w-5 text-primary" />
      </div>
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium flex items-center gap-2">
          {t('agents.fleet.homeTitle')}
          {running > 0 && (
            <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] bg-green-500/10 text-green-600 border border-green-500/20">
              <span className="h-1.5 w-1.5 rounded-full bg-green-500 animate-pulse" />
              {running}
            </span>
          )}
        </div>
        <p className="text-xs text-muted-foreground mt-0.5">
          {empty
            ? t('agents.fleet.homeEmpty')
            : t('agents.fleet.homeBody', { total, running })}
        </p>
      </div>
      {empty ? (
        <button
          onClick={(e) => { e.stopPropagation(); goToAgents() }}
          className="flex items-center gap-1 px-3 py-1.5 text-xs rounded-md bg-primary text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-3 w-3" />
          {t('agents.new', 'New')}
        </button>
      ) : (
        <span className="text-xs text-muted-foreground group-hover:text-primary flex items-center gap-1">
          {t('agents.fleet.manage')}
          <ChevronRight className="h-3.5 w-3.5" />
        </span>
      )}
    </div>
  )
}
