import { create } from 'zustand'

interface ConnectivityState {
  /** Whether the app can reach the backend */
  online: boolean
  /** Consecutive failure count */
  failCount: number
  /** Record a successful backend call */
  recordSuccess: () => void
  /** Record a failed backend call */
  recordFailure: () => void
}

const OFFLINE_THRESHOLD = 3

export const useConnectivityStore = create<ConnectivityState>((set, get) => ({
  online: true,
  failCount: 0,

  recordSuccess: () => {
    if (!get().online || get().failCount > 0) {
      set({ online: true, failCount: 0 })
    }
  },

  recordFailure: () => {
    const next = get().failCount + 1
    set({
      failCount: next,
      online: next < OFFLINE_THRESHOLD,
    })
  },
}))
