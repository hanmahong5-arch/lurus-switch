// Time formatting helpers. Pure functions — no hooks, no React, safe to
// call from store reducers and event handlers as well as components.
//
// We deliberately do NOT depend on the user's i18n locale because date
// formats inside Switch should follow OS locale (toLocale*) — that's what
// users expect from a desktop app. The exception is `formatRelative`,
// which picks Chinese vs English strings off the i18n short code.

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

function toDate(input: Date | string | number | null | undefined): Date | null {
  if (input == null) return null
  if (input instanceof Date) return isNaN(input.getTime()) ? null : input
  const d = new Date(input)
  return isNaN(d.getTime()) ? null : d
}

// formatLocal renders a wall-clock date in the OS locale, including
// both date and time of day. Use for table cells, log rows, anywhere
// the user needs to read an exact instant.
export function formatLocal(input: Date | string | number | null | undefined, opts?: Intl.DateTimeFormatOptions): string {
  const d = toDate(input)
  if (!d) return '—'
  return d.toLocaleString(undefined, opts)
}

// formatLocalDate omits the time-of-day part.
export function formatLocalDate(input: Date | string | number | null | undefined): string {
  const d = toDate(input)
  if (!d) return '—'
  return d.toLocaleDateString()
}

// formatLocalTime omits the date part. Defaults to HH:MM:SS.
export function formatLocalTime(input: Date | string | number | null | undefined, opts?: Intl.DateTimeFormatOptions): string {
  const d = toDate(input)
  if (!d) return '—'
  return d.toLocaleTimeString(undefined, opts ?? { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

// formatUTC renders an ISO-style UTC stamp. Use for log export, audit
// reports, anywhere the absolute time matters for cross-tz comparison.
export function formatUTC(input: Date | string | number | null | undefined): string {
  const d = toDate(input)
  if (!d) return '—'
  return d.toISOString().replace('T', ' ').replace(/\.\d+Z$/, ' UTC')
}

// formatRelative renders "5s ago" / "3m ago" / "2h ago" / "5d ago".
// Falls back to formatLocal when the date is older than a month.
//
// locale: 'zh' or 'en' (anything else is treated as English).
export function formatRelative(input: Date | string | number | null | undefined, locale: string = 'zh'): string {
  const d = toDate(input)
  if (!d) return '—'
  const isZh = locale?.startsWith('zh') ?? false
  const diff = Date.now() - d.getTime()
  const abs = Math.abs(diff)
  const future = diff < 0
  if (abs < 5 * SECOND) return isZh ? '刚刚' : 'just now'
  if (abs < MINUTE) {
    const n = Math.round(abs / SECOND)
    return isZh
      ? (future ? `${n} 秒后` : `${n} 秒前`)
      : (future ? `in ${n}s` : `${n}s ago`)
  }
  if (abs < HOUR) {
    const n = Math.round(abs / MINUTE)
    return isZh
      ? (future ? `${n} 分钟后` : `${n} 分钟前`)
      : (future ? `in ${n}m` : `${n}m ago`)
  }
  if (abs < DAY) {
    const n = Math.round(abs / HOUR)
    return isZh
      ? (future ? `${n} 小时后` : `${n} 小时前`)
      : (future ? `in ${n}h` : `${n}h ago`)
  }
  if (abs < 30 * DAY) {
    const n = Math.round(abs / DAY)
    return isZh
      ? (future ? `${n} 天后` : `${n} 天前`)
      : (future ? `in ${n}d` : `${n}d ago`)
  }
  return formatLocal(d)
}

// formatRange renders "from–to" using formatLocal for both endpoints. If
// the two stamps are on the same calendar day, the date is dropped from
// the right side for a tighter "11:00–14:30" style.
export function formatRange(
  start: Date | string | number | null | undefined,
  end: Date | string | number | null | undefined,
): string {
  const a = toDate(start)
  const b = toDate(end)
  if (!a && !b) return '—'
  if (!a || !b) return formatLocal(a ?? b)
  const sameDay = a.toDateString() === b.toDateString()
  if (sameDay) {
    return `${a.toLocaleString()} – ${b.toLocaleTimeString()}`
  }
  return `${a.toLocaleString()} – ${b.toLocaleString()}`
}
