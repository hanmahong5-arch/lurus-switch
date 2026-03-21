import { useState, useEffect } from 'react'
import { CheckCircle2, AlertTriangle, Info, XCircle, X, ChevronDown, ChevronUp, Copy, Check } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useToastStore, type Toast, type ToastType } from '../stores/toastStore'
import { cn } from '../lib/utils'

const ICON: Record<ToastType, typeof CheckCircle2> = {
  success: CheckCircle2,
  error: XCircle,
  warning: AlertTriangle,
  info: Info,
}

const STYLE: Record<ToastType, string> = {
  success: 'bg-green-500/15 border-green-500/30 text-green-600',
  error: 'bg-red-500/15 border-red-500/30 text-red-500',
  warning: 'bg-amber-500/15 border-amber-500/30 text-amber-600',
  info: 'bg-blue-500/15 border-blue-500/30 text-blue-500',
}

const ACCENT: Record<ToastType, string> = {
  success: 'border-l-green-500',
  error: 'border-l-red-500',
  warning: 'border-l-amber-500',
  info: 'border-l-blue-500',
}

const ACTION_STYLE: Record<ToastType, string> = {
  success: 'bg-green-500/20 hover:bg-green-500/30 text-green-600',
  error: 'bg-red-500/20 hover:bg-red-500/30 text-red-500',
  warning: 'bg-amber-500/20 hover:bg-amber-500/30 text-amber-600',
  info: 'bg-blue-500/20 hover:bg-blue-500/30 text-blue-500',
}

const PRIMARY_ACTION_STYLE: Record<ToastType, string> = {
  success: 'bg-green-500/30 hover:bg-green-500/40 text-green-700 font-medium',
  error: 'bg-red-500/30 hover:bg-red-500/40 text-red-600 font-medium',
  warning: 'bg-amber-500/30 hover:bg-amber-500/40 text-amber-700 font-medium',
  info: 'bg-blue-500/30 hover:bg-blue-500/40 text-blue-600 font-medium',
}

function ToastItem({ toast, onDismiss }: { toast: Toast; onDismiss: () => void }) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const [copied, setCopied] = useState(false)
  const [exiting, setExiting] = useState(false)

  const Icon = ICON[toast.type]
  const allActions = toast.actions ?? (toast.action ? [toast.action] : [])

  const handleDismiss = () => {
    setExiting(true)
  }

  // After exit animation completes, actually remove from store
  useEffect(() => {
    if (!exiting) return
    const timer = setTimeout(onDismiss, 250)
    return () => clearTimeout(timer)
  }, [exiting, onDismiss])

  const handleCopy = async () => {
    const text = toast.details || toast.message
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard API can fail in some contexts
    }
  }

  return (
    <div
      role="alert"
      aria-live={toast.type === 'error' ? 'assertive' : 'polite'}
      className={cn(
        'flex flex-col rounded-lg border shadow-lg text-xs overflow-hidden',
        exiting ? 'toast-exit' : 'toast-enter',
        STYLE[toast.type],
        toast.persistent && 'border-l-[3px]',
        toast.persistent && ACCENT[toast.type],
      )}
    >
      {/* Header row */}
      <div className="flex items-start gap-2.5 px-3.5 py-2.5">
        <Icon className="h-4 w-4 shrink-0 mt-0.5" />
        <p className="flex-1 leading-relaxed break-words">{toast.message}</p>

        {/* Inline action buttons */}
        <div className="flex items-center gap-1 shrink-0">
          {allActions.map((a, i) => (
            <button
              key={i}
              onClick={() => { a.onClick(); handleDismiss() }}
              className={cn(
                'px-2 py-1 rounded text-xs whitespace-nowrap transition-colors',
                a.primary ? PRIMARY_ACTION_STYLE[toast.type] : ACTION_STYLE[toast.type],
              )}
            >
              {a.label}
            </button>
          ))}
          <button
            onClick={handleDismiss}
            className="p-0.5 opacity-60 hover:opacity-100 transition-opacity"
            aria-label="Dismiss"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      {/* Details toolbar — for error toasts with details */}
      {(toast.details || toast.type === 'error') && (
        <div className="flex items-center gap-1 px-3.5 pb-2 -mt-1">
          {toast.details && (
            <button
              onClick={() => setExpanded(!expanded)}
              className={cn(
                'flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] transition-colors',
                ACTION_STYLE[toast.type],
              )}
            >
              {expanded
                ? <><ChevronUp className="h-3 w-3" />{t('error.action.hideDetails')}</>
                : <><ChevronDown className="h-3 w-3" />{t('error.action.showDetails')}</>
              }
            </button>
          )}
          <button
            onClick={handleCopy}
            className={cn(
              'flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] transition-colors',
              ACTION_STYLE[toast.type],
            )}
          >
            {copied
              ? <><Check className="h-3 w-3" />{t('error.action.copied')}</>
              : <><Copy className="h-3 w-3" />{t('error.action.copyError')}</>
            }
          </button>
        </div>
      )}

      {/* Expandable details */}
      {expanded && toast.details && (
        <div className="px-3.5 pb-2.5">
          <pre className="text-[10px] leading-tight p-2 rounded max-h-32 overflow-auto break-all whitespace-pre-wrap font-mono bg-black/10">
            {toast.details}
          </pre>
        </div>
      )}
    </div>
  )
}

export function ToastContainer() {
  const { toasts, dismissToast } = useToastStore()

  if (toasts.length === 0) return null

  return (
    <div
      className="fixed top-3 right-3 z-50 flex flex-col gap-2 max-w-sm w-96"
      aria-label="Notifications"
      role="region"
    >
      {toasts.map((toast) => (
        <ToastItem
          key={toast.id}
          toast={toast}
          onDismiss={() => dismissToast(toast.id)}
        />
      ))}
    </div>
  )
}
