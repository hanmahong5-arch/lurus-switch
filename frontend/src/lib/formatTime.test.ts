import { describe, it, expect, beforeAll, afterAll, vi } from 'vitest'
import {
  formatLocal, formatLocalDate, formatLocalTime, formatUTC,
  formatRelative, formatRange,
} from './formatTime'

// Anchor "now" to a known instant so all relative-time assertions are
// deterministic — without this each test would race the real clock.
const FIXED_NOW = new Date('2026-05-21T12:00:00Z').getTime()

beforeAll(() => {
  vi.useFakeTimers()
  vi.setSystemTime(FIXED_NOW)
})

afterAll(() => {
  vi.useRealTimers()
})

describe('formatLocal', () => {
  it('renders a Date as locale string', () => {
    const d = new Date('2026-05-21T10:30:00Z')
    expect(formatLocal(d)).toBe(d.toLocaleString())
  })
  it('parses an ISO string', () => {
    const d = new Date('2026-05-21T10:30:00Z')
    expect(formatLocal('2026-05-21T10:30:00Z')).toBe(d.toLocaleString())
  })
  it('returns em-dash for null/undefined/invalid', () => {
    expect(formatLocal(null)).toBe('—')
    expect(formatLocal(undefined)).toBe('—')
    expect(formatLocal('not-a-date')).toBe('—')
  })
})

describe('formatLocalDate / formatLocalTime', () => {
  it('formatLocalDate strips time-of-day', () => {
    const d = new Date('2026-05-21T10:30:00Z')
    expect(formatLocalDate(d)).toBe(d.toLocaleDateString())
  })
  it('formatLocalTime uses HH:MM:SS default', () => {
    const d = new Date('2026-05-21T10:30:45Z')
    const out = formatLocalTime(d)
    // Defaults vary by Intl impl, but should contain digits + colons.
    expect(out).toMatch(/\d{1,2}/)
  })
})

describe('formatUTC', () => {
  it('renders YYYY-MM-DD HH:MM:SS UTC', () => {
    const d = new Date('2026-05-21T10:30:45.123Z')
    expect(formatUTC(d)).toBe('2026-05-21 10:30:45 UTC')
  })
  it('handles null', () => {
    expect(formatUTC(null)).toBe('—')
  })
})

describe('formatRelative', () => {
  it('renders "just now" within 5s', () => {
    expect(formatRelative(new Date(FIXED_NOW - 2000), 'zh')).toBe('刚刚')
    expect(formatRelative(new Date(FIXED_NOW - 2000), 'en')).toBe('just now')
  })
  it('renders seconds for <1m', () => {
    expect(formatRelative(new Date(FIXED_NOW - 30 * 1000), 'zh')).toBe('30 秒前')
    expect(formatRelative(new Date(FIXED_NOW - 30 * 1000), 'en')).toBe('30s ago')
  })
  it('renders minutes for <1h', () => {
    expect(formatRelative(new Date(FIXED_NOW - 5 * 60 * 1000), 'zh')).toBe('5 分钟前')
  })
  it('renders hours for <1d', () => {
    expect(formatRelative(new Date(FIXED_NOW - 3 * 60 * 60 * 1000), 'zh')).toBe('3 小时前')
  })
  it('renders days for <30d', () => {
    expect(formatRelative(new Date(FIXED_NOW - 5 * 24 * 60 * 60 * 1000), 'zh')).toBe('5 天前')
  })
  it('falls back to formatLocal beyond 30 days', () => {
    const old = new Date(FIXED_NOW - 100 * 24 * 60 * 60 * 1000)
    expect(formatRelative(old, 'zh')).toBe(old.toLocaleString())
  })
  it('handles future stamps with "后" / "in Xs"', () => {
    expect(formatRelative(new Date(FIXED_NOW + 60 * 60 * 1000), 'zh')).toBe('1 小时后')
    expect(formatRelative(new Date(FIXED_NOW + 60 * 60 * 1000), 'en')).toBe('in 1h')
  })
  it('returns em-dash for invalid input', () => {
    expect(formatRelative('garbage', 'zh')).toBe('—')
    expect(formatRelative(null, 'en')).toBe('—')
  })
})

describe('formatRange', () => {
  it('renders single-side when one endpoint is null', () => {
    const d = new Date('2026-05-21T10:30:00Z')
    expect(formatRange(d, null)).toBe(d.toLocaleString())
    expect(formatRange(null, d)).toBe(d.toLocaleString())
  })
  it('shortens to time-only on the right when same day', () => {
    const a = new Date('2026-05-21T10:00:00Z')
    const b = new Date('2026-05-21T14:30:00Z')
    const out = formatRange(a, b)
    expect(out.includes(a.toLocaleString())).toBe(true)
    expect(out.includes(b.toLocaleTimeString())).toBe(true)
  })
  it('uses full date on both sides when different day', () => {
    const a = new Date('2026-05-20T10:00:00Z')
    const b = new Date('2026-05-21T14:30:00Z')
    const out = formatRange(a, b)
    expect(out.includes(a.toLocaleString())).toBe(true)
    expect(out.includes(b.toLocaleString())).toBe(true)
  })
  it('handles fully-null input', () => {
    expect(formatRange(null, undefined)).toBe('—')
  })
})
