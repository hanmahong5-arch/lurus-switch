// Selector for the "undo most recent action" command-palette entry.
//
// The audit store loads entries in reverse-chronological order. We want
// the FIRST one that is:
//   - reversible (Go side declared an inverse handler)
//   - succeeded   (no point undoing a denied/error op)
//   - not yet undone
//
// Pulled out of CommandPalette so the selection logic is testable
// without spinning up the whole palette.

interface UndoableLike {
  reversible: boolean
  outcome: string
  undoneAt?: string | null
}

export function pickLatestUndoableEntry<T extends UndoableLike>(
  entries: readonly T[],
): T | null {
  for (const e of entries) {
    if (e.reversible && e.outcome === 'ok' && !e.undoneAt) {
      return e
    }
  }
  return null
}
