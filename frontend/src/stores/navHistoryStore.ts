import { create } from 'zustand'
import type { ActiveTool } from './configStore'

export interface NavEntry {
  tool: ActiveTool
  subTab?: string
}

interface NavHistoryState {
  entries: NavEntry[]
  index: number
  push: (entry: NavEntry) => void
  back: () => NavEntry | null
  forward: () => NavEntry | null
  canGoBack: () => boolean
  canGoForward: () => boolean
  current: () => NavEntry | null
  hydrate: (entries: NavEntry[], index: number) => void
  reset: () => void
}

const MAX_ENTRIES = 50

function sameEntry(a: NavEntry | undefined, b: NavEntry | undefined): boolean {
  if (!a || !b) return false
  return a.tool === b.tool && (a.subTab ?? '') === (b.subTab ?? '')
}

export const useNavHistoryStore = create<NavHistoryState>((set, get) => ({
  entries: [],
  index: -1,

  push: (entry) =>
    set((state) => {
      const top = state.entries[state.index]
      if (sameEntry(top, entry)) return {}
      let next = state.entries.slice(0, state.index + 1)
      next.push(entry)
      if (next.length > MAX_ENTRIES) {
        const overflow = next.length - MAX_ENTRIES
        next = next.slice(overflow)
      }
      return { entries: next, index: next.length - 1 }
    }),

  back: () => {
    const { entries, index } = get()
    if (index <= 0) return null
    const nextIndex = index - 1
    set({ index: nextIndex })
    return entries[nextIndex]
  },

  forward: () => {
    const { entries, index } = get()
    if (index < 0 || index >= entries.length - 1) return null
    const nextIndex = index + 1
    set({ index: nextIndex })
    return entries[nextIndex]
  },

  canGoBack: () => get().index > 0,
  canGoForward: () => {
    const { entries, index } = get()
    return index >= 0 && index < entries.length - 1
  },

  current: () => {
    const { entries, index } = get()
    return index >= 0 ? entries[index] ?? null : null
  },

  hydrate: (entries, index) =>
    set(() => {
      const trimmed = entries.slice(-MAX_ENTRIES)
      const adjusted = Math.min(Math.max(index, -1), trimmed.length - 1)
      return { entries: trimmed, index: adjusted }
    }),

  reset: () => set({ entries: [], index: -1 }),
}))
