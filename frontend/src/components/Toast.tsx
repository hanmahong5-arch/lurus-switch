import { CheckCircle2, AlertTriangle, Info, XCircle, X } from 'lucide-react'
import { useToastStore, type ToastType } from '../stores/toastStore'
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

const ACTION_STYLE: Record<ToastType, string> = {
  success: 'bg-green-500/20 hover:bg-green-500/30 text-green-600',
  error: 'bg-red-500/20 hover:bg-red-500/30 text-red-500',
  warning: 'bg-amber-500/20 hover:bg-amber-500/30 text-amber-600',
  info: 'bg-blue-500/20 hover:bg-blue-500/30 text-blue-500',
}

export function ToastContainer() {
  const { toasts, dismissToast } = useToastStore()

  if (toasts.length === 0) return null

  return (
    <div className="fixed top-3 right-3 z-50 flex flex-col gap-2 max-w-sm">
      {toasts.map((toast) => {
        const Icon = ICON[toast.type]
        return (
          <div
            key={toast.id}
            className={cn(
              'flex items-start gap-2.5 px-3.5 py-2.5 rounded-lg border shadow-lg text-xs toast-enter',
              STYLE[toast.type]
            )}
          >
            <Icon className="h-4 w-4 shrink-0 mt-0.5" />
            <p className="flex-1 leading-relaxed break-words">{toast.message}</p>
            {toast.action && (
              <button
                onClick={() => {
                  toast.action!.onClick()
                  dismissToast(toast.id)
                }}
                className={cn(
                  'px-2 py-1 rounded text-xs font-medium whitespace-nowrap transition-colors',
                  ACTION_STYLE[toast.type]
                )}
              >
                {toast.action.label}
              </button>
            )}
            <button
              onClick={() => dismissToast(toast.id)}
              className="p-0.5 opacity-60 hover:opacity-100 transition-opacity shrink-0"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        )
      })}
    </div>
  )
}
