import { Bot, Zap, Sparkles, Download, RefreshCw, Settings, Loader2, CheckCircle2, XCircle } from 'lucide-react'
import { cn } from '../lib/utils'
import type { ToolStatus } from '../stores/dashboardStore'

const toolMeta: Record<string, { label: string; icon: typeof Bot; color: string; bgColor: string }> = {
  claude: { label: 'Claude Code', icon: Bot, color: 'text-orange-500', bgColor: 'bg-orange-500/10' },
  codex: { label: 'Codex', icon: Zap, color: 'text-green-500', bgColor: 'bg-green-500/10' },
  gemini: { label: 'Gemini CLI', icon: Sparkles, color: 'text-blue-500', bgColor: 'bg-blue-500/10' },
}

interface ToolCardProps {
  tool: ToolStatus
  installing: boolean
  updating: boolean
  onInstall: () => void
  onUpdate: () => void
  onConfigure: () => void
}

export function ToolCard({ tool, installing, updating, onInstall, onUpdate, onConfigure }: ToolCardProps) {
  const meta = toolMeta[tool.name] || { label: tool.name, icon: Bot, color: 'text-gray-500', bgColor: 'bg-gray-500/10' }
  const Icon = meta.icon
  const busy = installing || updating

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
            <p className="text-xs text-muted-foreground truncate">v{tool.version || 'unknown'}</p>
          ) : (
            <p className="text-xs text-muted-foreground">Not installed</p>
          )}
        </div>
        {/* Status indicator */}
        {tool.installed ? (
          <CheckCircle2 className="h-4 w-4 text-green-500 shrink-0" />
        ) : (
          <XCircle className="h-4 w-4 text-muted-foreground shrink-0" />
        )}
      </div>

      {/* Update available badge */}
      {tool.updateAvailable && tool.latestVersion && (
        <div className="text-xs bg-amber-500/10 text-amber-600 rounded px-2 py-1">
          Update available: v{tool.latestVersion}
        </div>
      )}

      {/* Actions */}
      <div className="flex gap-2 mt-auto">
        {!tool.installed ? (
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
            {installing ? 'Installing...' : 'Install'}
          </button>
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
                {updating ? 'Updating...' : 'Update'}
              </button>
            )}
            <button
              onClick={onConfigure}
              className={cn(
                'flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
                'border border-border hover:bg-muted',
                tool.updateAvailable ? '' : 'flex-1'
              )}
            >
              <Settings className="h-3.5 w-3.5" />
              Configure
            </button>
          </>
        )}
      </div>
    </div>
  )
}
