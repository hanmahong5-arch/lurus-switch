import { useEffect, useRef } from 'react'
import { useConfigStore } from '../stores/configStore'
import { useNavHistoryStore, type NavEntry } from '../stores/navHistoryStore'

const STORAGE_KEY = 'lurus-switch-nav-state'
const HISTORY_PERSIST_LIMIT = 20

interface PersistedNavState {
  subTabState: Record<string, string>
  lastActiveTool: string
  activeTool: string
  history?: { entries: NavEntry[]; index: number }
}

/**
 * Persists navigation state (active page, sub-tabs, last tool tab, back-stack)
 * to localStorage. Call this once at the app root level.
 */
export function useNavPersist() {
  const { subTabState, lastActiveTool, activeTool } = useConfigStore()
  const historyEntries = useNavHistoryStore((s) => s.entries)
  const historyIndex = useNavHistoryStore((s) => s.index)
  const hydrated = useRef(false)

  // Restore on mount (runs once)
  useEffect(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      if (!raw) {
        hydrated.current = true
        return
      }
      const saved: PersistedNavState = JSON.parse(raw)
      const store = useConfigStore.getState()
      if (saved.subTabState && typeof saved.subTabState === 'object') {
        for (const [page, tab] of Object.entries(saved.subTabState)) {
          store.setSubTabSilent(page as any, tab)
        }
      }
      if (saved.lastActiveTool) {
        store.setLastActiveTool(saved.lastActiveTool as any)
      }
      if (saved.history?.entries && Array.isArray(saved.history.entries)) {
        useNavHistoryStore.getState().hydrate(
          saved.history.entries,
          typeof saved.history.index === 'number' ? saved.history.index : -1,
        )
      }
    } catch {
      // Corrupt storage — ignore
    } finally {
      // Seed the back-stack with the current page so the very first sidebar
      // click has somewhere to go back to. Only seeds when history is still
      // empty after restore — keeps existing back-stacks intact across reloads.
      const navStore = useNavHistoryStore.getState()
      if (navStore.entries.length === 0) {
        const cfg = useConfigStore.getState()
        navStore.push({ tool: cfg.activeTool, subTab: cfg.subTabState[cfg.activeTool] })
      }
      hydrated.current = true
    }
  }, [])

  // Save on change (skipped until first hydrate completes so we don't blow
  // away the persisted history on mount before restore finishes)
  useEffect(() => {
    if (!hydrated.current) return
    try {
      // Only keep the tail of history so storage doesn't grow without bound.
      const tailStart = Math.max(0, historyEntries.length - HISTORY_PERSIST_LIMIT)
      const persistedEntries = historyEntries.slice(tailStart)
      const persistedIndex =
        historyIndex >= 0 ? Math.max(0, historyIndex - tailStart) : -1
      const state: PersistedNavState = {
        subTabState,
        lastActiveTool,
        activeTool,
        history: { entries: persistedEntries, index: persistedIndex },
      }
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
    } catch {
      // Storage full or disabled — ignore
    }
  }, [subTabState, lastActiveTool, activeTool, historyEntries, historyIndex])
}
