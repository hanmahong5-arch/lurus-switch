import { create } from 'zustand'

export interface ServerStatus {
  running: boolean
  port: number
  url: string
  uptime: number
  version: string
  binaryOk: boolean
}

export interface ServerConfig {
  port: number
  session_secret: string
  admin_token: string
  auto_start: boolean
}

interface GatewayState {
  status: ServerStatus | null
  adminToken: string | null
  pollingHandle: ReturnType<typeof setInterval> | null

  setStatus: (s: ServerStatus) => void
  setAdminToken: (t: string) => void
  startPolling: (fetchStatus: () => Promise<ServerStatus>, fetchToken: () => Promise<string>) => void
  stopPolling: () => void
}

export const useGatewayStore = create<GatewayState>((set, get) => ({
  status: null,
  adminToken: null,
  pollingHandle: null,

  setStatus: (s) => set({ status: s }),
  setAdminToken: (t) => set({ adminToken: t }),

  startPolling: (fetchStatus, fetchToken) => {
    // Stop any existing polling first.
    const existing = get().pollingHandle
    if (existing !== null) clearInterval(existing)

    const poll = async () => {
      try {
        const s = await fetchStatus()
        set({ status: s })
        if (s.running) {
          const token = await fetchToken()
          if (token) set({ adminToken: token })
        }
      } catch {
        // Non-fatal polling error — ignore
      }
    }

    // Run immediately, then every 5 seconds.
    poll()
    const handle = setInterval(poll, 5000)
    set({ pollingHandle: handle })
  },

  stopPolling: () => {
    const handle = get().pollingHandle
    if (handle !== null) clearInterval(handle)
    set({ pollingHandle: null })
  },
}))
