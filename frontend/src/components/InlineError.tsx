import { useState } from 'react'
import { AlertTriangle, XCircle, WifiOff, KeyRound, Settings, Terminal, Cpu, ShieldAlert, Copy, Check, ChevronDown, ChevronUp } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import type { ErrorCategory } from '../lib/errorClassifier'

const CATEGORY_ICON: Record<ErrorCategory, typeof AlertTriangle> = {
  network: WifiOff,
  auth: KeyRound,
  config: Settings,
  tool: Terminal,
  runtime: Cpu,
  gateway: AlertTriangle,
  permission: ShieldAlert,
  unknown: XCircle,
}

const CATEGORY_STYLE: Record<ErrorCategory, string> = {
  network: 'border-amber-500/30 bg-amber-500/10 text-amber-600',
  auth: 'border-red-500/30 bg-red-500/10 text-red-500',
  config: 'border-orange-500/30 bg-orange-500/10 text-orange-600',
  tool: 'border-blue-500/30 bg-blue-500/10 text-blue-500',
  runtime: 'border-purple-500/30 bg-purple-500/10 text-purple-500',
  gateway: 'border-amber-500/30 bg-amber-500/10 text-amber-600',
  permission: 'border-red-500/30 bg-red-500/10 text-red-500',
  unknown: 'border-red-500/30 bg-red-500/10 text-red-500',
}

interface InlineErrorProps {
  category: ErrorCategory
  message: string
  details?: string
  action?: { label: string; onClick: () => void }
  onDismiss?: () => void
  className?: string
}

/**
 * Rich inline error banner with category icon, expandable details,
 * copy button, optional action, and dismiss.
 */
export function InlineError({ category, message, details, action, onDismiss, className }: InlineErrorProps) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const [copied, setCopied] = useState(false)

  const Icon = CATEGORY_ICON[category]

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(details || message)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch { /* silent */ }
  }

  return (
    <div className={cn('border rounded-lg px-3 py-2 text-xs space-y-1.5', CATEGORY_STYLE[category], className)}>
      <div className="flex items-start gap-2">
        <Icon className="h-3.5 w-3.5 shrink-0 mt-0.5" />
        <p className="flex-1 leading-relaxed break-words">{message}</p>
        <div className="flex items-center gap-1 shrink-0">
          {action && (
            <button
              onClick={action.onClick}
              className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-current/10 hover:bg-current/20 transition-colors"
            >
              {action.label}
            </button>
          )}
          {onDismiss && (
            <button onClick={onDismiss} className="opacity-60 hover:opacity-100 transition-opacity">
              ✕
            </button>
          )}
        </div>
      </div>
      {/* Detail controls */}
      {(details || category !== 'unknown') && (
        <div className="flex items-center gap-1 pl-5.5">
          {details && (
            <button
              onClick={() => setExpanded(!expanded)}
              className="flex items-center gap-0.5 px-1 py-0.5 rounded text-[10px] opacity-70 hover:opacity-100 transition-opacity"
            >
              {expanded ? <ChevronUp className="h-2.5 w-2.5" /> : <ChevronDown className="h-2.5 w-2.5" />}
              {expanded ? t('error.action.hideDetails') : t('error.action.showDetails')}
            </button>
          )}
          <button
            onClick={handleCopy}
            className="flex items-center gap-0.5 px-1 py-0.5 rounded text-[10px] opacity-70 hover:opacity-100 transition-opacity"
          >
            {copied ? <Check className="h-2.5 w-2.5" /> : <Copy className="h-2.5 w-2.5" />}
            {copied ? t('error.action.copied') : t('error.action.copyError')}
          </button>
        </div>
      )}
      {expanded && details && (
        <pre className="text-[10px] leading-tight p-1.5 rounded bg-black/10 max-h-24 overflow-auto break-all whitespace-pre-wrap font-mono ml-5.5">
          {details}
        </pre>
      )}
    </div>
  )
}
