import { describe, it, expect, beforeEach } from 'vitest'
import {
  savePendingOwnerBind,
  resolvePendingOwnerBind,
  type MinimalStorage,
} from './ownerBindResume'

const KEY = 'switch.pendingOwnerBind'

// Simple in-memory storage that mirrors the contract resolvePendingOwnerBind
// expects. Keeping it tiny exposes any contract drift loudly.
function memStorage(initial?: Record<string, string>): MinimalStorage & { _data: Record<string, string> } {
  const data: Record<string, string> = { ...(initial ?? {}) }
  return {
    _data: data,
    getItem: (k) => (k in data ? data[k] : null),
    setItem: (k, v) => { data[k] = v },
    removeItem: (k) => { delete data[k] },
  }
}

describe('savePendingOwnerBind', () => {
  it('writes a hint with appId, appName, ts', () => {
    const store = memStorage()
    savePendingOwnerBind('app-1', 'Cursor', store, 1_700_000_000)
    expect(store._data[KEY]).toBeDefined()
    const parsed = JSON.parse(store._data[KEY])
    expect(parsed).toEqual({ appId: 'app-1', appName: 'Cursor', ts: 1_700_000_000 })
  })

  it('does nothing when storage is null', () => {
    // Should not throw — treating "no storage" as a no-op is intentional;
    // the resume flow is best-effort.
    expect(() => savePendingOwnerBind('app-1', 'Cursor', null)).not.toThrow()
  })

  it('rejects empty appId (would store unresolvable hint)', () => {
    const store = memStorage()
    savePendingOwnerBind('', 'Cursor', store)
    expect(store._data[KEY]).toBeUndefined()
  })

  it('swallows storage failures (quota, disabled)', () => {
    const broken: MinimalStorage = {
      getItem: () => null,
      setItem: () => { throw new Error('QuotaExceeded') },
      removeItem: () => {},
    }
    expect(() => savePendingOwnerBind('app-1', 'Cursor', broken)).not.toThrow()
  })
})

describe('resolvePendingOwnerBind', () => {
  let store: ReturnType<typeof memStorage>

  beforeEach(() => {
    store = memStorage()
  })

  const apps = [
    { id: 'app-1', name: 'Cursor' },
    { id: 'app-2', name: 'Claude Code' },
  ]

  it('returns null when storage is null', () => {
    expect(resolvePendingOwnerBind(apps, null)).toBeNull()
  })

  it('returns null when no hint stored', () => {
    expect(resolvePendingOwnerBind(apps, store)).toBeNull()
  })

  it('returns matched app when hint is fresh and id exists', () => {
    savePendingOwnerBind('app-2', 'Claude Code', store, 1_700_000_000)
    const out = resolvePendingOwnerBind(apps, store, 1_700_000_001)
    expect(out).toEqual({ id: 'app-2', name: 'Claude Code' })
  })

  it('consumes the hint regardless of match (single-shot)', () => {
    savePendingOwnerBind('app-1', 'Cursor', store, 1_700_000_000)
    resolvePendingOwnerBind(apps, store, 1_700_000_001)
    // Calling again should now miss — first call consumed it.
    expect(resolvePendingOwnerBind(apps, store, 1_700_000_002)).toBeNull()
    expect(store._data[KEY]).toBeUndefined()
  })

  it('consumes the hint even when stale (so user is not re-prompted)', () => {
    savePendingOwnerBind('app-1', 'Cursor', store, 0)
    // Far past the 30-min window.
    expect(resolvePendingOwnerBind(apps, store, 60 * 60 * 1000)).toBeNull()
    expect(store._data[KEY]).toBeUndefined()
  })

  it('returns null when hint older than 30 min', () => {
    savePendingOwnerBind('app-1', 'Cursor', store, 0)
    // Just past 30-min boundary.
    const past30 = 30 * 60 * 1000 + 1
    expect(resolvePendingOwnerBind(apps, store, past30)).toBeNull()
  })

  it('accepts hint exactly at 30-min boundary (inclusive)', () => {
    savePendingOwnerBind('app-1', 'Cursor', store, 0)
    expect(resolvePendingOwnerBind(apps, store, 30 * 60 * 1000)).not.toBeNull()
  })

  it('returns null when matched id no longer in apps list', () => {
    savePendingOwnerBind('app-deleted', 'Gone', store, 1_700_000_000)
    expect(resolvePendingOwnerBind(apps, store, 1_700_000_001)).toBeNull()
  })

  it('returns null on malformed JSON, still consumes', () => {
    store.setItem(KEY, '{not json')
    expect(resolvePendingOwnerBind(apps, store, 1_700_000_001)).toBeNull()
    expect(store._data[KEY]).toBeUndefined()
  })

  it('returns null when hint missing appId field', () => {
    store.setItem(KEY, JSON.stringify({ ts: Date.now() }))
    expect(resolvePendingOwnerBind(apps, store)).toBeNull()
  })

  it('returns null when hint missing ts field', () => {
    store.setItem(KEY, JSON.stringify({ appId: 'app-1' }))
    expect(resolvePendingOwnerBind(apps, store)).toBeNull()
  })

  it('returns null when getItem throws (storage disabled)', () => {
    const broken: MinimalStorage = {
      getItem: () => { throw new Error('disabled') },
      setItem: () => {},
      removeItem: () => {},
    }
    expect(resolvePendingOwnerBind(apps, broken)).toBeNull()
  })

  it('handles empty apps list without crash', () => {
    savePendingOwnerBind('app-1', 'Cursor', store, 1_700_000_000)
    expect(resolvePendingOwnerBind([], store, 1_700_000_001)).toBeNull()
    // Still consumed.
    expect(store._data[KEY]).toBeUndefined()
  })
})

describe('default storage (jsdom sessionStorage)', () => {
  // jsdom provides a real sessionStorage; these tests exercise the
  // default-parameter path so the helper isn't only validated against
  // the in-memory fake. If sessionStorage is ever taken away (e.g. an
  // integration env that disables it) the swallow-on-error branches
  // already covered above keep the helper safe.
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('save → resolve roundtrip writes and reads real sessionStorage', () => {
    savePendingOwnerBind('app-x', 'Real', undefined, 1_700_000_000)
    expect(sessionStorage.getItem(KEY)).not.toBeNull()
    const found = resolvePendingOwnerBind(
      [{ id: 'app-x', name: 'Real' }],
      undefined,
      1_700_000_001,
    )
    expect(found?.id).toBe('app-x')
    // Single-shot — the real sessionStorage entry is gone.
    expect(sessionStorage.getItem(KEY)).toBeNull()
  })

  it('resolve returns null when no hint is in real sessionStorage', () => {
    expect(resolvePendingOwnerBind([{ id: 'app-x' }])).toBeNull()
  })
})
