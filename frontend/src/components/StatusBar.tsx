import { Bot } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useConfigStore } from '../stores/configStore'
import { useAgentStore } from '../stores/agentStore'
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
  // Agents page is Personal-only. Hide the indicator everywhere else so
  // we don't surface a status pill that can't be clicked through to.
  const showAgents = appMode === 'personal' && agentStats.total > 0

  return (
    <footer className="h-6 bg-muted/50 border-t border-border flex items-center justify-between px-4 text-xs text-muted-foreground">
      <span>{t('statusBar.status')}: {status}</span>
      <div className="flex items-center gap-3">
        {/* Gateway status indicator */}
        <span className="flex items-center gap-1.5">
          <span className={`h-1.5 w-1.5 rounded-full ${gwRunning ? 'bg-green-500' : 'bg-muted-foreground/30'}`} />
          {gwRunning ? (
            <span className="text-green-600 dark:text-green-400">
              Gateway :{gwStatus?.port || ''}
            </span>
          ) : (
            <span>{t('statusBar.gatewayOff', 'Gateway off')}</span>
          )}
        </span>

        {/* Connected tools count */}
        {installedCount > 0 && (
          <span className={boundCount === installedCount && boundCount > 0
            ? 'text-green-600 dark:text-green-400'
            : 'text-muted-foreground'
          }>
            {boundCount}/{installedCount} {t('statusBar.tools')}
          </span>
        )}

        {/* Agent fleet indicator — click to jump */}
        {showAgents && (
          <button
            onClick={() => setActiveTool('agents')}
            className="flex items-center gap-1 hover:text-foreground transition-colors"
            title={t('agents.statusBar.tooltip', { running: agentStats.running, total: agentStats.total })}
          >
            <Bot className="h-3 w-3" />
            <span className={agentStats.running > 0 ? 'text-green-600 dark:text-green-400' : ''}>
              {agentStats.running}/{agentStats.total}
            </span>
          </button>
        )}

        <AccountStatusBadge />
        <span>v{appVersion || '1.0.0'}</span>
      </div>
    </footer>
  )
}
