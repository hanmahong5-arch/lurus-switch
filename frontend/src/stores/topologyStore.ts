import { create } from 'zustand'
import { GetTopologySnapshot } from '../../wailsjs/go/main/App'
import type { topology } from '../../wailsjs/go/models'

// Wails-emitted Snapshot uses `any` for the Go time.Time field. The store
// works at the JSON shape directly — we never use generatedAt as a Date.
export type SnapshotJSON = topology.Snapshot

interface TopologyState {
  snapshot: SnapshotJSON | null
  loading: boolean
  error: string | null
  lastUpdated: number | null
  pollHandle: ReturnType<typeof setInterval> | null

  refresh: () => Promise<void>
  startPolling: (intervalMs?: number) => void
  stopPolling: () => void
}

const DEFAULT_POLL_MS = 10_000

export const useTopologyStore = create<TopologyState>((set, get) => ({
  snapshot: null,
  loading: false,
  error: null,
  lastUpdated: null,
  pollHandle: null,

  refresh: async () => {
    set({ loading: true })
    try {
      const snap = await GetTopologySnapshot()
      set({ snapshot: snap, error: null, lastUpdated: Date.now(), loading: false })
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      set({ error: msg, loading: false })
    }
  },

  startPolling: (intervalMs = DEFAULT_POLL_MS) => {
    const { pollHandle, refresh } = get()
    if (pollHandle) return // already running
    refresh()
    const handle = setInterval(refresh, intervalMs)
    set({ pollHandle: handle })
  },

  stopPolling: () => {
    const { pollHandle } = get()
    if (pollHandle) {
      clearInterval(pollHandle)
      set({ pollHandle: null })
    }
  },
}))
