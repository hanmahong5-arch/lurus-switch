/**
 * Retry an async operation with exponential backoff.
 * Only retries on network-like errors (ECONNREFUSED, fetch failed, timeout).
 * Non-retryable errors (auth, config, permission) throw immediately.
 */

const RETRYABLE = /ECONNREFUSED|fetch failed|timeout|ETIMEDOUT|ENOTFOUND|ERR_CONNECTION|network.*(?:error|fail)/i

function isRetryable(err: unknown): boolean {
  const msg = err instanceof Error ? err.message : String(err)
  return RETRYABLE.test(msg)
}

interface RetryOptions {
  /** Max retry attempts (default 2) */
  maxRetries?: number
  /** Base delay in ms (default 1000) */
  baseDelay?: number
  /** Jitter factor 0-1 (default 0.3) */
  jitter?: number
}

export async function withRetry<T>(
  fn: () => Promise<T>,
  opts: RetryOptions = {},
): Promise<T> {
  const { maxRetries = 2, baseDelay = 1000, jitter = 0.3 } = opts

  let lastError: unknown
  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      return await fn()
    } catch (err) {
      lastError = err
      if (attempt >= maxRetries || !isRetryable(err)) {
        throw err
      }
      // Exponential backoff with jitter
      const delay = baseDelay * Math.pow(2, attempt)
      const jitterMs = delay * jitter * (Math.random() * 2 - 1)
      await new Promise((r) => setTimeout(r, delay + jitterMs))
    }
  }
  throw lastError
}
