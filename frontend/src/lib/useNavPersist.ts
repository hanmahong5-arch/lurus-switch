import { useEffect } from 'react'
import { useConfigStore } from '../stores/configStore'

const STORAGE_KEY = 'lurus-switch-nav-state'

interface PersistedNavState {
  subTabState: Record<string, string>
  lastActiveTool: string
  activeTool: string
}

/**
 * Persists navigation state (active page, sub-tabs, last tool tab) to localStorage.
 * Call this once at the app root level.
 */
export function useNavPersist() {
  const { subTabState, lastActiveTool, activeTool } = useConfigStore()

  // Restore on mount
  useEffect(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      if (!raw) return
      const saved: PersistedNavState = JSON.parse(raw)
      const store = useConfigStore.getState()
      if (saved.subTabState && typeof saved.subTabState === 'object') {
        for (const [page, tab] of Object.entries(saved.subTabState)) {
          store.setSubTab(page as any, tab)
        }
      }
      if (saved.lastActiveTool) {
        store.setLastActiveTool(saved.lastActiveTool as any)
      }
    } catch {
      // Corrupt storage — ignore
    }
  }, [])

  // Save on change
  useEffect(() => {
    try {
      const state: PersistedNavState = {
        subTabState,
        lastActiveTool,
        activeTool,
      }
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
    } catch {
      // Storage full or disabled — ignore
    }
  }, [subTabState, lastActiveTool, activeTool])
}
