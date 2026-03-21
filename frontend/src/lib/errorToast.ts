import type { ActiveTool } from '../stores/configStore'
import type { AddToastOptions, ToastAction } from '../stores/toastStore'
import { classifyError } from './errorClassifier'
import { appendErrorLog } from './errorLog'
import { useConnectivityStore } from '../stores/connectivityStore'

interface ErrorToastContext {
  /** Function to navigate to a page (typically configStore.setActiveTool) */
  navigate?: (page: ActiveTool) => void
  /** Current page — suppresses navigation if already there */
  currentPage?: ActiveTool
  /** Retry callback — always shown as primary action */
  retry?: () => void
  /** i18n translate function */
  t?: (key: string) => string
}

/**
 * Show a smart error toast: classifies the error, builds navigation + retry
 * actions, includes expandable details for debugging, logs to local storage,
 * and updates connectivity state.
 *
 * Usage:
 *   errorToast(toast, err, { navigate: setActiveTool, retry: () => load(), t })
 */
export function errorToast(
  addToast: (type: 'error', message: string, opts?: AddToastOptions) => void,
  err: unknown,
  ctx: ErrorToastContext = {},
) {
  const classified = classifyError(err)
  const { navigate, currentPage, retry, t } = ctx
  const tr = (key: string, fallback: string) => (t ? t(key) : fallback)

  // Log to local error history
  appendErrorLog(classified)

  // Update global connectivity state
  if (classified.category === 'network') {
    useConnectivityStore.getState().recordFailure()
  }

  const actions: ToastAction[] = []

  // Primary: retry action (most common user intent after an error)
  if (retry) {
    actions.push({
      label: tr('error.action.retry', 'Retry'),
      onClick: retry,
      primary: true,
    })
  }

  // Secondary: navigate to suggested resolution page
  if (classified.navigateTo && navigate && classified.navigateTo !== currentPage) {
    const actionLabel = classified.actionKey
      ? tr(classified.actionKey, classified.actionKey)
      : tr('error.action.goTo', 'Go')
    actions.push({
      label: actionLabel,
      onClick: () => navigate(classified.navigateTo!),
    })
  }

  addToast('error', classified.message, {
    actions: actions.length > 0 ? actions : undefined,
    details: classified.details,
    persistent: classified.persistent,
  })
}

/**
 * Aggregate multiple errors from a batch operation into a single toast.
 * Shows a summary message with per-item details in the expandable section.
 *
 * Usage:
 *   const failures = results.filter(r => !r.success)
 *   if (failures.length > 0) {
 *     batchErrorToast(toast, failures.map(f => ({ label: f.tool, message: f.message })), { t })
 *   }
 */
export function batchErrorToast(
  addToast: (type: 'error', message: string, opts?: AddToastOptions) => void,
  items: Array<{ label: string; message: string }>,
  ctx: { t?: (key: string, opts?: Record<string, unknown>) => string; retry?: () => void } = {},
) {
  if (items.length === 0) return

  const { t, retry } = ctx
  const tr = (key: string, fallback: string, opts?: Record<string, unknown>) =>
    t ? t(key, opts) : fallback

  const summary = tr(
    'error.batchFailed',
    `${items.length} operation(s) failed`,
    { count: items.length },
  )

  const details = items.map((i) => `${i.label}: ${i.message}`).join('\n')

  const actions: ToastAction[] = []
  if (retry) {
    actions.push({
      label: tr('error.action.retry', 'Retry'),
      onClick: retry,
      primary: true,
    })
  }

  addToast('error', summary, {
    actions: actions.length > 0 ? actions : undefined,
    details,
    persistent: items.length >= 3,
  })
}
