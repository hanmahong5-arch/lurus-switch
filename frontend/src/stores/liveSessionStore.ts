import { create } from 'zustand'
import { getLiveSessions, getAllLiveSessions, type LiveSession } from '../lib/liveSessionApi'

// Live-session store. Hydrates from the Go binding on demand and again on
// every "livesession:update" Wails event the backend emits (see
// app.go startup). The page subscribes once and the rest is push.

interface LiveSessionState {
  sessions: LiveSession[]
  loading: boolean
  lastFetch: number          // epoch ms; helps debounce
  showIdle: boolean          // include sessions outside the active window
  error: string | null

  setShowIdle: (v: boolean) => void
  refresh: () => Promise<void>
}

// Debounce: many livesession:update events can fire close together when a
// session emits a burst of small JSONL records. Coalesce to at most one
// fetch per N ms; the underlying Go state is already aggregated so we
// lose nothing by skipping intermediate ticks.
const REFRESH_MIN_INTERVAL_MS = 400

export const useLiveSessionStore = create<LiveSessionState>((set, get) => ({
  sessions: [],
  loading: false,
  lastFetch: 0,
  showIdle: false,
  error: null,

  setShowIdle: (v) => {
    set({ showIdle: v })
    void get().refresh()
  },

  refresh: async () => {
    const now = Date.now()
    if (now - get().lastFetch < REFRESH_MIN_INTERVAL_MS && get().loading) {
      return
    }
    set({ loading: true })
    try {
      const fetcher = get().showIdle ? getAllLiveSessions : getLiveSessions
      const sessions = await fetcher()
      set({ sessions, loading: false, lastFetch: Date.now(), error: null })
    } catch (e) {
      set({
        loading: false,
        lastFetch: Date.now(),
        error: e instanceof Error ? e.message : String(e),
      })
    }
  },
}))
