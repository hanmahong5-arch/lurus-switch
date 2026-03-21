import type { ActiveTool } from '../stores/configStore'

export type ErrorCategory =
  | 'network'
  | 'auth'
  | 'config'
  | 'tool'
  | 'runtime'
  | 'gateway'
  | 'permission'
  | 'unknown'

export interface ClassifiedError {
  category: ErrorCategory
  /** Cleaned, user-friendly message */
  message: string
  /** Raw error string for debugging (only set when different from message) */
  details?: string
  /** Whether the toast should stay until user dismisses it */
  persistent: boolean
  /** Suggested page to navigate to for resolution */
  navigateTo?: ActiveTool
  /** i18n key for the navigation action button label */
  actionKey?: string
}

interface Pattern {
  test: RegExp
  category: ErrorCategory
  navigateTo?: ActiveTool
  actionKey: string
  persistent: boolean
}

// Ordered by specificity — first match wins.
const PATTERNS: Pattern[] = [
  // Authentication / authorization — check before network (401/403 are HTTP responses, not network failures)
  {
    test: /\b401\b|unauthorized|token.*(?:invalid|expired|missing)|forbidden|\b403\b|authentication.*fail|unauthenticated/i,
    category: 'auth',
    navigateTo: 'account',
    actionKey: 'error.action.reconnect',
    persistent: true,
  },
  // Gateway port / crash
  {
    test: /EADDRINUSE|port.*(?:in use|occupied)|gateway.*(?:crash|fail|stop)|bind.*fail|already.*listen|address already in use/i,
    category: 'gateway',
    navigateTo: 'gateway',
    actionKey: 'error.action.checkGateway',
    persistent: true,
  },
  // Network / connectivity
  {
    test: /ECONNREFUSED|fetch failed|timeout|ETIMEDOUT|ENOTFOUND|connection refused|no such host|dial tcp|connect:|network.*(?:error|fail)|unreachable|ERR_CONNECTION/i,
    category: 'network',
    navigateTo: 'home',
    actionKey: 'error.action.checkProxy',
    persistent: true,
  },
  // Config file parsing
  {
    test: /parse|JSON.*syntax|TOML.*error|malformed|invalid.*config|unexpected token|unmarshal|SyntaxError/i,
    category: 'config',
    actionKey: 'error.action.openConfig',
    persistent: false,
  },
  // Tool not installed
  {
    test: /not installed|command not found|executable file not found|ENOENT.*(?:claude|codex|gemini|picoclaw|nullclaw|zeroclaw|openclaw)/i,
    category: 'tool',
    navigateTo: 'home',
    actionKey: 'error.action.installTool',
    persistent: false,
  },
  // Runtime dependency missing
  {
    test: /node.*not found|bun.*not found|runtime.*(?:missing|not found)|npm.*not found|python.*not found/i,
    category: 'runtime',
    navigateTo: 'home',
    actionKey: 'error.action.installRuntime',
    persistent: false,
  },
  // File permission
  {
    test: /EACCES|permission denied|access.*denied|operation not permitted/i,
    category: 'permission',
    actionKey: 'error.action.checkPermission',
    persistent: false,
  },
]

function getRawMessage(err: unknown): string {
  if (err == null) return ''
  if (err instanceof Error) return err.message
  if (typeof err === 'string') return err
  return String(err)
}

function cleanMessage(raw: string): string {
  let msg = raw
  // Strip Go RPC prefix
  msg = msg.replace(/^rpc error:.*desc\s*=\s*/i, '')
  // Strip redundant "Error:" prefix
  msg = msg.replace(/^Error:\s*/i, '')
  // Truncate excessively long messages
  if (msg.length > 200) msg = msg.slice(0, 200) + '\u2026'
  return msg || 'Unknown error'
}

/**
 * Classify a raw error into a structured object with category, message,
 * suggested navigation target, and persistence hint.
 */
export function classifyError(err: unknown): ClassifiedError {
  const raw = getRawMessage(err)
  const msg = cleanMessage(raw)
  const hasExtraDetails = raw.length > 0 && raw !== msg

  for (const p of PATTERNS) {
    if (p.test.test(raw)) {
      return {
        category: p.category,
        message: msg,
        details: hasExtraDetails ? raw : undefined,
        persistent: p.persistent,
        navigateTo: p.navigateTo,
        actionKey: p.actionKey,
      }
    }
  }

  return {
    category: 'unknown',
    message: msg,
    details: hasExtraDetails ? raw : undefined,
    persistent: false,
  }
}
