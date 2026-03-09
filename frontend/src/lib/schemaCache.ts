import type { ToolSchema } from './toolSchema'

const CACHE_KEY = 'lurus_schemas_v1'
const CACHE_TTL_MS = 24 * 60 * 60 * 1000 // 24h

interface CacheEntry {
  schemas: ToolSchema[]
  ts: number
}

export function loadFromCache(): CacheEntry | null {
  try {
    const raw = localStorage.getItem(CACHE_KEY)
    if (!raw) return null
    const entry = JSON.parse(raw) as CacheEntry
    if (Date.now() - entry.ts > CACHE_TTL_MS) return null
    return entry
  } catch {
    return null
  }
}

export function saveToCache(schemas: ToolSchema[]): void {
  try {
    const entry: CacheEntry = { schemas, ts: Date.now() }
    localStorage.setItem(CACHE_KEY, JSON.stringify(entry))
  } catch {
    // localStorage might be unavailable — silently ignore
  }
}

/** Fetches schemas from a remote URL. Returns null on any failure (network, timeout, parse). */
export async function fetchRemote(url: string): Promise<ToolSchema[] | null> {
  const ctrl = new AbortController()
  const timer = setTimeout(() => ctrl.abort(), 5000)
  try {
    const resp = await fetch(url, { signal: ctrl.signal })
    if (!resp.ok) return null
    const data = (await resp.json()) as ToolSchema[]
    return Array.isArray(data) ? data : null
  } catch {
    return null
  } finally {
    clearTimeout(timer)
  }
}
