import { useTranslation } from 'react-i18next'
import {
  Download, Zap, Wrench, ArrowUpCircle, CheckCircle2, AlertTriangle, Play,
} from 'lucide-react'
import { cn } from '../lib/utils'
import type { Suggestion } from '../stores/homeStore'

const ACTION_ICONS: Record<string, React.ComponentType<{ className?: string }>> = {
  'install-tool': Download,
  'install-runtime': Download,
  'update-tool': ArrowUpCircle,
  'connect-gateway': Zap,
  'start-gateway': Play,
  'fix-config': Wrench,
  'install-git': Download,
}

const PRIORITY_COLORS: Record<number, string> = {
  1: 'border-red-500/30 bg-red-500/5',
  2: 'border-yellow-500/30 bg-yellow-500/5',
  3: 'border-border bg-card',
}

interface Props {
  suggestions: Suggestion[]
  onAction: (suggestion: Suggestion) => void
  executing: Record<string, boolean>
}

export function QuickActions({ suggestions, onAction, executing }: Props) {
  const { t } = useTranslation()

  if (suggestions.length === 0) {
    return (
      <div className="rounded-xl border border-green-500/30 bg-green-500/5 p-6 flex items-center gap-3">
        <CheckCircle2 className="h-6 w-6 text-green-500 flex-shrink-0" />
        <div>
          <h3 className="text-sm font-semibold text-green-600 dark:text-green-400">
            {t('home.allGood')}
          </h3>
          <p className="text-xs text-muted-foreground mt-0.5">
            {t('home.allGoodDesc')}
          </p>
        </div>
      </div>
    )
  }

  // Sort by priority (1=critical first)
  const sorted = [...suggestions].sort((a, b) => a.priority - b.priority)

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <AlertTriangle className="h-4 w-4 text-muted-foreground" />
        <h3 className="text-sm font-semibold">{t('home.quickActions')}</h3>
        <span className="text-xs text-muted-foreground">
          ({suggestions.length})
        </span>
      </div>

      <div className="space-y-2">
        {sorted.map((suggestion) => {
          const Icon = ACTION_ICONS[suggestion.action] || Wrench
          const isExecuting = executing[suggestion.id] || false
          const colorClass = PRIORITY_COLORS[suggestion.priority] || PRIORITY_COLORS[3]

          return (
            <div
              key={suggestion.id}
              className={cn(
                'flex items-center gap-3 rounded-lg border p-3',
                colorClass
              )}
            >
              <Icon className="h-4 w-4 text-muted-foreground flex-shrink-0" />
              <span className="flex-1 text-sm">{suggestion.title}</span>
              <button
                onClick={() => onAction(suggestion)}
                disabled={isExecuting}
                className={cn(
                  'px-3 py-1 rounded-md text-xs font-medium transition-colors',
                  'bg-primary text-primary-foreground hover:bg-primary/90',
                  'disabled:opacity-50 disabled:cursor-not-allowed'
                )}
              >
                {isExecuting ? '...' : t(`home.actionLabel.${suggestion.action}`)}
              </button>
            </div>
          )
        })}
      </div>
    </div>
  )
}
