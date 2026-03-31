import {
  Download, RefreshCw, Settings, Loader2, CheckCircle2, XCircle, Trash2, AlertTriangle, Rocket,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { toolMeta, DEFAULT_TOOL_META } from '../lib/toolMeta'
import type { ToolStatus, ToolHealthResult } from '../stores/dashboardStore'

const healthDotColor: Record<string, string> = {
  green: 'bg-green-500',
  yellow: 'bg-amber-400',
  red: 'bg-red-500',
}

interface ToolCardProps {
  tool: ToolStatus
  installing: boolean
  updating: boolean
  uninstalling?: boolean
  health?: ToolHealthResult
  onInstall: () => void
  onUpdate: () => void
  onConfigure: () => void
  onUninstall?: () => void
  onViewIssues?: () => void
  onQuickStart?: () => void
  quickStarting?: boolean
}

export function ToolCard({
  tool, installing, updating, uninstalling = false, health,
  onInstall, onUpdate, onConfigure, onUninstall, onViewIssues,
  onQuickStart, quickStarting = false,
}: ToolCardProps) {
  const { t } = useTranslation()
  const meta = toolMeta[tool.name] || DEFAULT_TOOL_META
  const Icon = meta.icon
  const busy = installing || updating || uninstalling || quickStarting

  const healthTooltip = health?.issues?.length
    ? health.issues.join('; ')
    : health?.status === 'green' ? t('dashboard.healthOk') : ''

  return (
    <div className="border border-border rounded-lg p-4 flex flex-col gap-3 bg-card">
      {/* Header */}
      <div className="flex items-center gap-3">
        <div className={cn('p-2 rounded-md', meta.bgColor)}>
          <Icon className={cn('h-5 w-5', meta.color)} />
        </div>
        <div className="flex-1 min-w-0">
          <h3 className="font-medium text-sm">{meta.label}</h3>
          {tool.installed ? (
            <p className="text-xs text-muted-foreground truncate">v{tool.version || '?'}</p>
          ) : (
            <div>
              <p className="text-xs text-muted-foreground">{t('dashboard.notInstalled')}</p>
              <p className="text-[10px] text-muted-foreground/70">
                {meta.dep === 'bun'
                  ? t('dashboard.deps.depRequires', { runtime: 'Bun' })
                  : t('dashboard.deps.depStandalone')}
              </p>
            </div>
          )}
        </div>
        {/* Status + health indicator */}
        <div className="flex items-center gap-1.5 shrink-0">
          {health && tool.installed && (
            <span
              className={cn('w-2 h-2 rounded-full', healthDotColor[health.status] || 'bg-gray-400')}
              title={healthTooltip}
            />
          )}
          {tool.installed ? (
            <CheckCircle2 className="h-4 w-4 text-green-500" />
          ) : (
            <XCircle className="h-4 w-4 text-muted-foreground" />
          )}
        </div>
      </div>

      {/* Update available badge */}
      {tool.updateAvailable && tool.latestVersion && (
        <div className="text-xs bg-amber-500/10 text-amber-600 rounded px-2 py-1">
          {t('dashboard.updateAvailable')}: v{tool.latestVersion}
        </div>
      )}

      {/* Health issues summary */}
      {health?.status === 'red' && health.issues?.length > 0 && (
        <div className="text-xs bg-red-500/10 text-red-500 rounded px-2 py-1 truncate" title={health.issues.join('; ')}>
          {health.issues[0]}
        </div>
      )}

      {/* Actions */}
      <div className="flex gap-2 mt-auto flex-wrap">
        {!tool.installed ? (
          onQuickStart ? (
            <button
              onClick={onQuickStart}
              disabled={busy}
              className={cn(
                'flex-1 flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
                'bg-gradient-to-r from-primary to-blue-500 text-white hover:opacity-90',
                'disabled:opacity-50 disabled:cursor-not-allowed'
              )}
            >
              {quickStarting ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Rocket className="h-3.5 w-3.5" />
              )}
              {quickStarting ? t('home.quickStarting') : t('home.quickStart')}
            </button>
          ) : (
          <button
            onClick={onInstall}
            disabled={busy}
            className={cn(
              'flex-1 flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
              'bg-primary text-primary-foreground hover:bg-primary/90',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            {installing ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Download className="h-3.5 w-3.5" />
            )}
            {installing ? t('dashboard.installing') : t('dashboard.install')}
          </button>
          )
        ) : (
          <>
            {tool.updateAvailable && (
              <button
                onClick={onUpdate}
                disabled={busy}
                className={cn(
                  'flex-1 flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
                  'bg-amber-500 text-white hover:bg-amber-600',
                  'disabled:opacity-50 disabled:cursor-not-allowed'
                )}
              >
                {updating ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <RefreshCw className="h-3.5 w-3.5" />
                )}
                {updating ? t('dashboard.updating') : t('dashboard.update')}
              </button>
            )}
            <button
              onClick={onConfigure}
              className={cn(
                'flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
                'border border-border hover:bg-muted',
                !tool.updateAvailable && !onUninstall ? 'flex-1' : ''
              )}
            >
              <Settings className="h-3.5 w-3.5" />
              {t('dashboard.config')}
            </button>
            {health?.status === 'red' && onViewIssues && (
              <button
                onClick={onViewIssues}
                className={cn(
                  'flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
                  'border border-amber-500/30 text-amber-500 hover:bg-amber-500/10'
                )}
                title={t('toolCard.viewIssuesTitle')}
              >
                <AlertTriangle className="h-3.5 w-3.5" />
                {t('toolCard.viewIssues')}
              </button>
            )}
            {onUninstall && (
              <button
                onClick={onUninstall}
                disabled={busy}
                title={t('dashboard.uninstallTool')}
                className={cn(
                  'flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
                  'border border-red-500/30 text-red-500 hover:bg-red-500/10',
                  'disabled:opacity-50 disabled:cursor-not-allowed'
                )}
              >
                {uninstalling ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Trash2 className="h-3.5 w-3.5" />
                )}
              </button>
            )}
          </>
        )}
      </div>
    </div>
  )
}
