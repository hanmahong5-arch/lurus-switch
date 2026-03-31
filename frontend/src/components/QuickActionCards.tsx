import { Download, Play, Link2, Wrench, Activity, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { TOOL_ORDER } from '../lib/toolMeta'
import { useHomeStore } from '../stores/homeStore'
import { useSwitchStore } from '../stores/switchStore'

interface QuickActionCardsProps {
  onInstallAll: () => void | Promise<void>
  onStartGateway: () => void | Promise<void>
  onConnectAll: () => void | Promise<void>
  onFixAll: () => void | Promise<void>
  onDiagnostics: () => void
  installingAll?: boolean
  fixing?: boolean
}

export function QuickActionCards({
  onInstallAll, onStartGateway, onConnectAll, onFixAll, onDiagnostics,
  installingAll, fixing,
}: QuickActionCardsProps) {
  const { t } = useTranslation()
  const tools = useHomeStore((s) => s.tools)
  const toolHealth = useHomeStore((s) => s.toolHealth)
  const scoreReport = useHomeStore((s) => s.scoreReport)
  const gwStatus = useSwitchStore((s) => s.status)

  const installedCount = TOOL_ORDER.filter((n) => tools[n]?.installed).length
  const totalTools = TOOL_ORDER.length
  const allInstalled = installedCount === totalTools

  const gwRunning = gwStatus?.running ?? false

  const connectedCount = TOOL_ORDER.filter((n) => tools[n]?.installed && toolHealth[n]?.status === 'green').length
  const installedTotal = installedCount

  const issueCount = scoreReport?.suggestions?.length ?? 0

  const healthScore = scoreReport?.totalScore ?? 0
  const maxScore = scoreReport?.maxScore ?? 100

  const cards = [
    {
      key: 'installAll',
      icon: Download,
      label: t('home.quickActionCards.installAll'),
      desc: allInstalled
        ? t('home.quickActionCards.installAllDone')
        : t('home.quickActionCards.installAllDesc', { installed: installedCount, total: totalTools }),
      onClick: onInstallAll,
      disabled: installingAll || allInstalled,
      loading: installingAll,
      accent: allInstalled ? 'text-muted-foreground' : 'text-primary',
      bgAccent: allInstalled ? 'bg-muted' : 'bg-primary/10',
    },
    {
      key: 'startGateway',
      icon: Play,
      label: t('home.quickActionCards.startGateway'),
      desc: gwRunning ? t('home.quickActionCards.gatewayRunning') : t('home.quickActionCards.gatewayStopped'),
      onClick: onStartGateway,
      disabled: gwRunning,
      accent: gwRunning ? 'text-green-500' : 'text-amber-500',
      bgAccent: gwRunning ? 'bg-green-500/10' : 'bg-amber-500/10',
    },
    {
      key: 'connectAll',
      icon: Link2,
      label: t('home.quickActionCards.connectAll'),
      desc: installedTotal > 0
        ? t('home.quickActionCards.connectAllDesc', { connected: connectedCount, total: installedTotal })
        : t('dashboard.noToolsTitle'),
      onClick: onConnectAll,
      disabled: installedTotal === 0 || !gwRunning,
      accent: connectedCount === installedTotal && installedTotal > 0 ? 'text-green-500' : 'text-blue-500',
      bgAccent: connectedCount === installedTotal && installedTotal > 0 ? 'bg-green-500/10' : 'bg-blue-500/10',
    },
    {
      key: 'fixAll',
      icon: Wrench,
      label: t('home.quickActionCards.fixAll'),
      desc: issueCount > 0
        ? t('home.quickActionCards.fixAllDesc', { count: issueCount })
        : t('home.quickActionCards.fixAllClean'),
      onClick: onFixAll,
      disabled: fixing || issueCount === 0,
      loading: fixing,
      accent: issueCount > 0 ? 'text-red-500' : 'text-green-500',
      bgAccent: issueCount > 0 ? 'bg-red-500/10' : 'bg-green-500/10',
      badge: issueCount > 0 ? issueCount : undefined,
    },
    {
      key: 'diagnostics',
      icon: Activity,
      label: t('home.quickActionCards.diagnostics'),
      desc: scoreReport
        ? t('home.quickActionCards.diagnosticsScore', { score: `${healthScore}/${maxScore}` })
        : '...',
      onClick: onDiagnostics,
      accent: !scoreReport ? 'text-muted-foreground' : healthScore >= 80 ? 'text-green-500' : healthScore >= 50 ? 'text-amber-500' : 'text-red-500',
      bgAccent: !scoreReport ? 'bg-muted' : healthScore >= 80 ? 'bg-green-500/10' : healthScore >= 50 ? 'bg-amber-500/10' : 'bg-red-500/10',
    },
  ]

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
      {cards.map((card) => {
        const Icon = card.icon
        return (
          <button
            key={card.key}
            onClick={card.onClick}
            disabled={card.disabled}
            className={cn(
              'relative flex flex-col items-center gap-2 p-4 rounded-lg border border-border bg-card',
              'hover:bg-muted transition-colors text-center',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
          >
            {card.badge && (
              <span className="absolute -top-1.5 -right-1.5 min-w-[20px] h-5 flex items-center justify-center px-1 text-[10px] font-bold text-white bg-red-500 rounded-full">
                {card.badge}
              </span>
            )}
            <div className={cn('p-2 rounded-lg', card.bgAccent)}>
              {card.loading ? (
                <Loader2 className={cn('h-5 w-5 animate-spin', card.accent)} />
              ) : (
                <Icon className={cn('h-5 w-5', card.accent)} />
              )}
            </div>
            <div>
              <p className="text-xs font-semibold">{card.label}</p>
              <p className="text-[10px] text-muted-foreground mt-0.5">{card.desc}</p>
            </div>
          </button>
        )
      })}
    </div>
  )
}
