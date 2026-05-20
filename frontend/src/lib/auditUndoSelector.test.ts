import { describe, it, expect } from 'vitest'
import { pickLatestUndoableEntry } from './auditUndoSelector'

// Convenience constructor — keeps each test focused on the field that
// matters rather than restating boilerplate. `id` is included only so
// assertions can identify which entry was picked.
function entry(over: Partial<{ id: string; reversible: boolean; outcome: string; undoneAt: string | null }>) {
  return {
    id: over.id ?? 'e1',
    reversible: over.reversible ?? true,
    outcome: over.outcome ?? 'ok',
    undoneAt: over.undoneAt,
  }
}

describe('pickLatestUndoableEntry', () => {
  it('returns null on empty list', () => {
    expect(pickLatestUndoableEntry([])).toBeNull()
  })

  it('returns first reversible+ok+not-undone entry', () => {
    const got = pickLatestUndoableEntry([
      entry({ id: 'a', reversible: true, outcome: 'ok' }),
      entry({ id: 'b', reversible: true, outcome: 'ok' }),
    ])
    expect(got?.id).toBe('a') // newest-first ordering — pick head
  })

  it('skips non-reversible entries', () => {
    const got = pickLatestUndoableEntry([
      entry({ id: 'a', reversible: false, outcome: 'ok' }),
      entry({ id: 'b', reversible: true, outcome: 'ok' }),
    ])
    expect(got?.id).toBe('b')
  })

  it('skips non-ok outcomes (denied / error)', () => {
    const got = pickLatestUndoableEntry([
      entry({ id: 'a', outcome: 'denied' }),
      entry({ id: 'b', outcome: 'error' }),
      entry({ id: 'c', outcome: 'ok' }),
    ])
    expect(got?.id).toBe('c')
  })

  it('skips already-undone entries (undoneAt set)', () => {
    const got = pickLatestUndoableEntry([
      entry({ id: 'a', undoneAt: '2026-05-10T12:00:00Z' }),
      entry({ id: 'b' }),
    ])
    expect(got?.id).toBe('b')
  })

  it('treats undoneAt=null as not-undone', () => {
    // The store types `undoneAt?: string | null`. Both null and undefined
    // mean "not undone yet" — only a string timestamp counts as undone.
    const got = pickLatestUndoableEntry([entry({ id: 'a', undoneAt: null })])
    expect(got?.id).toBe('a')
  })

  it('returns null when nothing matches', () => {
    const got = pickLatestUndoableEntry([
      entry({ id: 'a', reversible: false }),
      entry({ id: 'b', outcome: 'denied' }),
      entry({ id: 'c', undoneAt: '2026-05-10T12:00:00Z' }),
    ])
    expect(got).toBeNull()
  })

  it('does not require all fields to match — first hit wins', () => {
    // A perfectly-undoable entry buried under junk: still picked.
    const got = pickLatestUndoableEntry([
      entry({ id: 'a', reversible: false, outcome: 'denied' }),
      entry({ id: 'b' }),
      entry({ id: 'c' }),
    ])
    expect(got?.id).toBe('b')
  })
})
