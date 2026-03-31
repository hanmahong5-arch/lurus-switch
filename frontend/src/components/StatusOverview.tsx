import { RefreshCw, Loader2, CheckCircle2, XCircle, Minus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { TOOL_ORDER, toolMeta, DEFAULT_TOOL_META } from '../lib/toolMeta'
import { useHomeStore } from '../stores/homeStore'
import { useSwitchStore } from '../stores/switchStore'
import { useConfigStore } from '../stores/configStore'

const healthDotColor: Record<string, string> = {
  green: 'bg-green-500',
  yellow: 'bg-amber-400',
  red: 'bg-red-500',
}

interface StatusOverviewProps {
  onRefresh: () => void
  refreshing?: boolean
}

export function StatusOverview({ onRefresh, refreshing }: StatusOverviewProps) {
  const { t } = useTranslation()
  const tools = useHomeStore((s) => s.tools)
  const toolHealth = useHomeStore((s) => s.toolHealth)
  const gwStatus = useSwitchStore((s) => s.status)
  const setActiveTool = useConfigStore((s) => s.setActiveTool)
  const setSubTab = useConfigStore((s) => s.setSubTab)
  const gwRunning = gwStatus?.running ?? false

  return (
    <div className="border border-border rounded-lg bg-card overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2.5 border-b border-border">
        <h3 className="text-sm font-semibold">{t('home.statusOverview.title')}</h3>
        <button
          onClick={onRefresh}
          disabled={refreshing}
          className={cn(
            'flex items-center gap-1 px-2 py-1 rounded text-xs text-muted-foreground',
            'hover:bg-muted transition-colors',
            'disabled:opacity-50 disabled:cursor-not-allowed',
          )}
        >
          {refreshing ? <Loader2 className="h-3 w-3 animate-spin" /> : <RefreshCw className="h-3 w-3" />}
          {t('dashboard.refresh')}
        </button>
      </div>

      {/* Table */}
      <div className="overflow-x-auto">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-b border-border text-muted-foreground">
              <th className="text-left px-4 py-2 font-medium">{t('home.statusOverview.tool')}</th>
              <th className="text-left px-3 py-2 font-medium">{t('home.statusOverview.status')}</th>
              <th className="text-left px-3 py-2 font-medium">{t('home.statusOverview.version')}</th>
              <th className="text-center px-3 py-2 font-medium">{t('home.statusOverview.health')}</th>
              <th className="text-left px-3 py-2 font-medium">{t('home.statusOverview.gateway')}</th>
            </tr>
          </thead>
          <tbody>
            {TOOL_ORDER.map((name) => {
              const tool = tools[name]
              const meta = toolMeta[name] || DEFAULT_TOOL_META
              const health = toolHealth[name]
              const Icon = meta.icon
              const installed = tool?.installed ?? false

              const gwConnected = installed && gwRunning && health?.status === 'green'

              return (
                <tr
                  key={name}
                  className="border-b border-border/50 last:border-0 hover:bg-muted/30 cursor-pointer transition-colors"
                  onClick={() => { setActiveTool('tools'); setSubTab('tools', name) }}
                >
                  <td className="px-4 py-2">
                    <div className="flex items-center gap-2">
                      <Icon className={cn('h-3.5 w-3.5', meta.color)} />
                      <span className="font-medium">{meta.label}</span>
                    </div>
                  </td>
                  <td className="px-3 py-2">
                    {installed ? (
                      <span className="flex items-center gap-1 text-green-500">
                        <CheckCircle2 className="h-3 w-3" />
                        {t('home.statusOverview.installed')}
                      </span>
                    ) : (
                      <span className="flex items-center gap-1 text-muted-foreground">
                        <XCircle className="h-3 w-3" />
                        {t('home.statusOverview.notInstalled')}
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-muted-foreground">
                    {installed ? `v${tool?.version || '?'}` : <Minus className="h-3 w-3" />}
                  </td>
                  <td className="px-3 py-2 text-center">
                    {installed && health ? (
                      <span
                        className={cn('inline-block w-2.5 h-2.5 rounded-full', healthDotColor[health.status] || 'bg-gray-400')}
                        title={health.issues?.join('; ') || t('dashboard.healthOk')}
                      />
                    ) : (
                      <Minus className="h-3 w-3 text-muted-foreground mx-auto" />
                    )}
                  </td>
                  <td className="px-3 py-2">
                    {!installed ? (
                      <Minus className="h-3 w-3 text-muted-foreground" />
                    ) : gwConnected ? (
                      <span className="text-green-500">{t('home.statusOverview.connected')}</span>
                    ) : (
                      <span className="text-muted-foreground">{t('home.statusOverview.notConnected')}</span>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}
