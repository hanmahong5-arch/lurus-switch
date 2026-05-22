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
  success: 'bg-emerald-500/15 border-emerald-500/30 text-emerald-400',
  error:   'bg-red-500/15 border-red-500/30 text-red-400',
  warning: 'bg-amber-500/15 border-amber-500/30 text-amber-400',
  info:    'bg-blue-500/15 border-blue-500/30 text-blue-400',
}

const ACCENT: Record<ToastType, string> = {
  success: 'border-l-emerald-400',
  error:   'border-l-red-400',
  warning: 'border-l-amber-400',
  info:    'border-l-blue-400',
}

const ACTION_STYLE: Record<ToastType, string> = {
  success: 'bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-400',
  error:   'bg-red-500/20 hover:bg-red-500/30 text-red-400',
  warning: 'bg-amber-500/20 hover:bg-amber-500/30 text-amber-400',
  info:    'bg-blue-500/20 hover:bg-blue-500/30 text-blue-400',
}

const PRIMARY_ACTION_STYLE: Record<ToastType, string> = {
  success: 'bg-emerald-500/30 hover:bg-emerald-500/40 text-emerald-300 font-medium',
  error:   'bg-red-500/30 hover:bg-red-500/40 text-red-300 font-medium',
  warning: 'bg-amber-500/30 hover:bg-amber-500/40 text-amber-300 font-medium',
  info:    'bg-blue-500/30 hover:bg-blue-500/40 text-blue-300 font-medium',
}

// Toast bodies wider than this are auto-split: first chunk stays in the
// header (line-clamped to 3 lines), the rest moves into the expandable
// details section. Keeps a 1KB pre-signed S3 URL from drowning the UI
// while still letting the user copy the full text.
const MESSAGE_OVERFLOW_CHARS = 220

function ToastItem({ toast, onDismiss }: { toast: Toast; onDismiss: () => void }) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const [copied, setCopied] = useState(false)
  const [exiting, setExiting] = useState(false)

  const Icon = ICON[toast.type]
  const allActions = toast.actions ?? (toast.action ? [toast.action] : [])

  // Auto-split long messages: keep a short headline visible, push the full
  // payload into the expandable details so the toast never overflows.
  const overflow = !toast.details && toast.message.length > MESSAGE_OVERFLOW_CHARS
  const headline = overflow ? toast.message.slice(0, MESSAGE_OVERFLOW_CHARS).trimEnd() + '…' : toast.message
  const overflowDetails = overflow ? toast.message : undefined
  const effectiveDetails = toast.details ?? overflowDetails
  const hasDetailsToggle = !!effectiveDetails

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
    const text = effectiveDetails || toast.message
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
        <p className="flex-1 min-w-0 leading-relaxed break-words line-clamp-3">{headline}</p>

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
      {(hasDetailsToggle || toast.type === 'error') && (
        <div className="flex items-center gap-1 px-3.5 pb-2 -mt-1">
          {hasDetailsToggle && (
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
      {expanded && effectiveDetails && (
        <div className="px-3.5 pb-2.5">
          <pre className="text-[10px] leading-tight p-2 rounded max-h-32 overflow-auto break-all whitespace-pre-wrap font-mono bg-black/10">
            {effectiveDetails}
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
      className="fixed top-12 right-3 z-50 flex flex-col gap-2 max-w-sm w-96"
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
