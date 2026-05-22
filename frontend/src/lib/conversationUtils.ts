// Conversation list / timeline helpers.
//
// These are pure functions so they're easy to unit-test and don't pull in
// the React tree.

import type { conversation } from '../../wailsjs/go/models'

// Go's time.Time zero value marshals to "0001-01-01T00:00:00Z". The JS
// Date parsed from that is technically valid but renders as nonsense
// ("1年12月31日 下午8:00:00" in zh-CN locale). Treat anything before
// 2000 as "no timestamp" and let the UI hide it.
export function parseSaneDate(input: unknown): Date | null {
  if (!input) return null
  const d = new Date(input as string | number)
  if (isNaN(d.getTime())) return null
  if (d.getFullYear() < 2000) return null
  return d
}

export function formatAbsolute(d: Date, locale: string): string {
  return d.toLocaleString(locale, {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit',
  })
}

// Compact "时间感"标签: "刚刚" / "5 分钟前" / "昨天 14:32" / "5 月 8 日"
export function formatRelative(d: Date, now: Date, isZh: boolean): string {
  const diffMs = now.getTime() - d.getTime()
  const diffSec = Math.round(diffMs / 1000)
  if (diffSec < 60) return isZh ? '刚刚' : 'just now'
  const diffMin = Math.round(diffSec / 60)
  if (diffMin < 60) return isZh ? `${diffMin} 分钟前` : `${diffMin}m ago`
  const sameDay = isSameLocalDay(d, now)
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  if (sameDay) return isZh ? `今天 ${hh}:${mm}` : `today ${hh}:${mm}`
  const yest = new Date(now); yest.setDate(yest.getDate() - 1)
  if (isSameLocalDay(d, yest)) return isZh ? `昨天 ${hh}:${mm}` : `yesterday ${hh}:${mm}`
  // Same year — drop the year part.
  if (d.getFullYear() === now.getFullYear()) {
    return isZh
      ? `${d.getMonth() + 1} 月 ${d.getDate()} 日`
      : d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
  }
  return d.toLocaleDateString(isZh ? 'zh-CN' : 'en-US', { year: 'numeric', month: 'short', day: 'numeric' })
}

function isSameLocalDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear()
      && a.getMonth() === b.getMonth()
      && a.getDate() === b.getDate()
}

// Date buckets for the session list left rail. Ordered earliest→latest
// so the rendering loop can detect transitions cheaply.
export type Bucket =
  | 'today' | 'yesterday' | 'past7' | 'past30' | 'thisYear' | 'older' | 'unknown'

const BUCKET_LABEL_ZH: Record<Bucket, string> = {
  today: '今天', yesterday: '昨天', past7: '本周', past30: '本月',
  thisYear: '今年', older: '更早', unknown: '时间未知',
}
const BUCKET_LABEL_EN: Record<Bucket, string> = {
  today: 'Today', yesterday: 'Yesterday', past7: 'Past 7 days', past30: 'Past 30 days',
  thisYear: 'This year', older: 'Older', unknown: 'Unknown',
}

export function bucketLabel(b: Bucket, isZh: boolean): string {
  return (isZh ? BUCKET_LABEL_ZH : BUCKET_LABEL_EN)[b]
}

export function bucketOf(d: Date | null, now: Date): Bucket {
  if (!d) return 'unknown'
  if (isSameLocalDay(d, now)) return 'today'
  const yest = new Date(now); yest.setDate(yest.getDate() - 1)
  if (isSameLocalDay(d, yest)) return 'yesterday'
  const ms = now.getTime() - d.getTime()
  const dayMs = 86_400_000
  if (ms <= 7 * dayMs) return 'past7'
  if (ms <= 30 * dayMs) return 'past30'
  if (d.getFullYear() === now.getFullYear()) return 'thisYear'
  return 'older'
}

// Best-available recency for a conversation row. EndedAt is preferred
// (covers running sessions whose mtime is stale because the lock-holding
// process hasn't flushed); falls back to FileModTime (nanoseconds since
// epoch). Returns null when both are missing/zero.
export function conversationDate(c: conversation.ConversationMeta): Date | null {
  const ended = parseSaneDate((c as any).endedAt)
  if (ended) return ended
  const started = parseSaneDate((c as any).startedAt)
  if (started) return started
  if (c.fileModTime && c.fileModTime > 0) {
    // Go stamps this in nanoseconds. Convert to millis for JS Date.
    return new Date(Math.floor(c.fileModTime / 1_000_000))
  }
  return null
}

// Stable newest-first ordering for the left list.
export function sortByRecencyDesc(
  rows: conversation.ConversationMeta[],
): conversation.ConversationMeta[] {
  return rows.slice().sort((a, b) => {
    const ta = conversationDate(a)?.getTime() ?? 0
    const tb = conversationDate(b)?.getTime() ?? 0
    if (ta !== tb) return tb - ta
    return (b.sessionID || '').localeCompare(a.sessionID || '')
  })
}

// Strip wrapping XML noise that Claude Code emits for slash-commands so
// the timeline doesn't waste space on `<command-name>usage</command-name>`.
// Returns { stripped: true, label, body } when the content was just one
// wrapper, otherwise { stripped: false, body: content }.
const SINGLE_WRAPPER_RE = /^\s*<([a-zA-Z][\w-]*)>\s*([\s\S]*?)\s*<\/\1>\s*$/

export function stripCommandWrapper(content: string | undefined): {
  stripped: boolean
  label?: string
  body: string
} {
  if (!content) return { stripped: false, body: '' }
  const m = content.match(SINGLE_WRAPPER_RE)
  if (!m) return { stripped: false, body: content }
  return { stripped: true, label: m[1], body: m[2] }
}

// Returns the byte offsets of every case-insensitive occurrence of needle
// in haystack. Used by the timeline search to compute the highlight ranges.
export function findMatches(haystack: string, needle: string): Array<[number, number]> {
  if (!needle || !haystack) return []
  const out: Array<[number, number]> = []
  const hlo = haystack.toLowerCase()
  const nlo = needle.toLowerCase()
  let i = 0
  while (i <= hlo.length - nlo.length) {
    const idx = hlo.indexOf(nlo, i)
    if (idx < 0) break
    out.push([idx, idx + nlo.length])
    i = idx + nlo.length
  }
  return out
}
