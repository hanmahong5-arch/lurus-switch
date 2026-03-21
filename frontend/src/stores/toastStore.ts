import { create } from 'zustand'

export type ToastType = 'success' | 'error' | 'warning' | 'info'

export interface ToastAction {
  label: string
  onClick: () => void
  /** Primary actions get stronger visual emphasis */
  primary?: boolean
}

export interface Toast {
  id: string
  type: ToastType
  message: string
  /** Optional single action button (backward compat) */
  action?: ToastAction
  /** Multiple action buttons (overrides single action when present) */
  actions?: ToastAction[]
  /** Raw error details — shown in expandable section */
  details?: string
  /** Persistent toasts stay until user dismisses */
  persistent?: boolean
}

/** Auto-dismiss durations by type (ms). Error toasts stay longer. */
const DURATIONS: Record<ToastType, number> = {
  success: 3000,
  info: 4000,
  warning: 5000,
  error: 8000,
}

let nextId = 0
/** Track recent messages to suppress duplicates within a short window. */
const recentMessages = new Map<string, number>()
const DEDUP_WINDOW_MS = 3000

/**
 * Frequency circuit breaker: track error-type toast frequency.
 * If more than FREQ_THRESHOLD errors fire within FREQ_WINDOW_MS,
 * subsequent errors are suppressed until the window resets.
 */
const FREQ_WINDOW_MS = 5000
const FREQ_THRESHOLD = 5
let freqWindowStart = 0
let freqCount = 0

export interface AddToastOptions {
  action?: ToastAction
  actions?: ToastAction[]
  details?: string
  persistent?: boolean
}

interface ToastState {
  toasts: Toast[]
  addToast: (type: ToastType, message: string, opts?: ToastAction | AddToastOptions) => void
  dismissToast: (id: string) => void
}

/** Type guard: distinguish legacy single-action from new options object */
function isToastAction(v: unknown): v is ToastAction {
  return v != null && typeof v === 'object' && 'onClick' in v && 'label' in v
}

export const useToastStore = create<ToastState>((set) => ({
  toasts: [],

  addToast: (type, message, opts) => {
    // Deduplicate: suppress identical messages within the dedup window.
    const dedupKey = `${type}:${message}`
    const lastSeen = recentMessages.get(dedupKey) ?? 0
    if (Date.now() - lastSeen < DEDUP_WINDOW_MS) {
      return // suppress duplicate
    }
    recentMessages.set(dedupKey, Date.now())
    // Prune old entries to prevent memory leak.
    if (recentMessages.size > 50) {
      const now = Date.now()
      for (const [k, t] of recentMessages) {
        if (now - t > DEDUP_WINDOW_MS * 2) recentMessages.delete(k)
      }
    }

    // Frequency circuit breaker: suppress error flood
    if (type === 'error') {
      const now = Date.now()
      if (now - freqWindowStart > FREQ_WINDOW_MS) {
        freqWindowStart = now
        freqCount = 1
      } else {
        freqCount++
        if (freqCount > FREQ_THRESHOLD) {
          return // suppress — too many errors in rapid succession
        }
      }
    }

    // Normalize: legacy callers pass ToastAction directly, new callers pass AddToastOptions
    let action: ToastAction | undefined
    let actions: ToastAction[] | undefined
    let details: string | undefined
    let persistent: boolean | undefined

    if (isToastAction(opts)) {
      action = opts
    } else if (opts) {
      action = opts.action
      actions = opts.actions
      details = opts.details
      persistent = opts.persistent
    }

    const id = `toast-${++nextId}`
    set((state) => ({
      toasts: [...state.toasts.slice(-4), { id, type, message, action, actions, details, persistent }],
    }))

    // Persistent toasts do not auto-dismiss
    if (!persistent) {
      setTimeout(() => {
        set((state) => ({
          toasts: state.toasts.filter((t) => t.id !== id),
        }))
      }, DURATIONS[type])
    }
  },

  dismissToast: (id) =>
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    })),
}))
