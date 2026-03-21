import type { ClassifiedError } from './errorClassifier'

interface ErrorLogEntry {
  timestamp: string
  category: string
  message: string
  details?: string
}

const STORAGE_KEY = 'lurus-switch-error-log'
const MAX_ENTRIES = 100

/**
 * Append a classified error to the local error log (localStorage).
 * Keeps the most recent MAX_ENTRIES entries.
 */
export function appendErrorLog(classified: ClassifiedError) {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    const log: ErrorLogEntry[] = raw ? JSON.parse(raw) : []
    log.push({
      timestamp: new Date().toISOString(),
      category: classified.category,
      message: classified.message,
      details: classified.details,
    })
    // Trim to max entries
    if (log.length > MAX_ENTRIES) {
      log.splice(0, log.length - MAX_ENTRIES)
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(log))
  } catch {
    // localStorage quota exceeded or unavailable — silent
  }
}

/** Read the full error log. */
export function readErrorLog(): ErrorLogEntry[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? JSON.parse(raw) : []
  } catch {
    return []
  }
}

/** Clear the error log. */
export function clearErrorLog() {
  try {
    localStorage.removeItem(STORAGE_KEY)
  } catch {
    // silent
  }
}

/**
 * Export the error log as a human-readable string for diagnostics.
 */
export function exportErrorLog(): string {
  const log = readErrorLog()
  if (log.length === 0) return '(no errors recorded)'
  return log
    .map((e) => `[${e.timestamp}] [${e.category}] ${e.message}${e.details ? '\n  ' + e.details : ''}`)
    .join('\n')
}
