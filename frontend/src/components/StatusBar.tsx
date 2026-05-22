import { Bot, Zap } from 'lucide-react'
import { useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useConfigStore } from '../stores/configStore'
import { useAgentStore } from '../stores/agentStore'
import { useRelayStore } from '../stores/relayStore'
import { AccountStatusBadge } from './AccountStatusBadge'
import { useDashboardStore } from '../stores/dashboardStore'
import { useSwitchStore } from '../stores/switchStore'

export function StatusBar() {
  const { status, appMode, setActiveTool } = useConfigStore()
  const { appVersion } = useDashboardStore()
  const gwStatus = useSwitchStore((s) => s.status)
  const envCheck = useSwitchStore((s) => s.envCheck)
  const agentStats = useAgentStore((s) => s.stats)
  const { t } = useTranslation()

  const gwRunning = gwStatus?.running ?? false
  const boundCount = envCheck?.boundCount ?? 0
  const installedCount = envCheck?.installedCount ?? 0

  // Relay circuit summary — flash a chip when *any* endpoint is open or
  // half-open so the user notices fail-over without opening RelayPage.
  const circuitState = useRelayStore((s) => s.circuitState)
  const pollCircuit = useRelayStore((s) => s.pollCircuitState)
  useEffect(() => {
    void pollCircuit()
    const h = setInterval(() => { void pollCircuit() }, 10_000)
    return () => clearInterval(h)
  }, [pollCircuit])
  const relayChip = useMemo(() => {
    const values = Object.values(circuitState)
    if (values.length === 0) return null
    const open = values.find((s) => s.status === 'open')
    if (open) return { tone: 'red' as const, label: t('statusBar.relayOpen', 'Relay degraded') }
    const half = values.find((s) => s.status === 'half_open')
    if (half) return { tone: 'amber' as const, label: t('statusBar.relayProbing', 'Relay probing') }
    return null
  }, [circuitState, t])
  // Agents page is Personal-only. Hide the indicator everywhere else so
  // we don't surface a status pill that can't be clicked through to.
  const showAgents = appMode === 'personal' && agentStats.total > 0

  return (
    <footer className="h-6 bg-card-recessed border-t border-rule-strong flex items-center justify-between px-4 text-xs text-muted-foreground font-mono">
      <span className="tracking-[0.08em]">▸ {t('statusBar.status')}: <span className="text-foreground">{status}</span></span>
      <div className="flex items-center gap-3">
        {/* Gateway status indicator */}
        <span className="flex items-center gap-1.5 tabular-nums">
          <span className={`h-1.5 w-1.5 rounded-full ${gwRunning ? 'bg-emerald-400 animate-pulse' : 'bg-muted-foreground/30'}`} />
          {gwRunning ? (
            <span className="text-emerald-400">
              Gateway :{gwStatus?.port || ''}
            </span>
          ) : (
            <span>▪ {t('statusBar.gatewayOff', 'Gateway off')}</span>
          )}
        </span>

        {/* Connected tools count */}
        {installedCount > 0 && (
          <span className={boundCount === installedCount && boundCount > 0
            ? 'text-emerald-400 tabular-nums'
            : 'text-muted-foreground tabular-nums'
          }>
            {boundCount}/{installedCount} {t('statusBar.tools')}
          </span>
        )}

        {/* Agent fleet indicator — click to jump */}
        {showAgents && (
          <button
            onClick={() => setActiveTool('agents')}
            className="flex items-center gap-1 hover:text-foreground transition-colors tabular-nums"
            title={t('agents.statusBar.tooltip', { running: agentStats.running, total: agentStats.total })}
          >
            <Bot className="h-3 w-3" />
            <span className={agentStats.running > 0 ? 'text-emerald-400' : ''}>
              {agentStats.running}/{agentStats.total}
            </span>
          </button>
        )}

        {relayChip && (
          <button
            onClick={() => setActiveTool('gateway')}
            className={
              'inline-flex items-center gap-1 hover:opacity-80 transition-opacity ' +
              (relayChip.tone === 'red' ? 'text-red-400' : 'text-amber-400')
            }
            title={relayChip.label}
          >
            <Zap className="h-3 w-3" />
            <span>{relayChip.label}</span>
          </button>
        )}

        <AccountStatusBadge />
        <span className="tabular-nums">v{appVersion || '1.0.0'}</span>
      </div>
    </footer>
  )
}
