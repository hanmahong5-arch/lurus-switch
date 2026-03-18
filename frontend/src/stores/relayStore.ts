import { create } from 'zustand'
import type { relay } from '../../wailsjs/go/models'

/** Tool name → relay endpoint ID */
type ToolRelayMapping = Record<string, string>

interface RelayState {
  endpoints: relay.RelayEndpoint[]
  cloudEndpoints: relay.RelayEndpoint[]
  mapping: ToolRelayMapping
  loading: boolean
  applying: boolean

  setEndpoints: (e: relay.RelayEndpoint[]) => void
  setCloudEndpoints: (e: relay.RelayEndpoint[]) => void
  setMapping: (m: ToolRelayMapping) => void
  setLoading: (l: boolean) => void
  setApplying: (a: boolean) => void
}

export const useRelayStore = create<RelayState>((set) => ({
  endpoints: [],
  cloudEndpoints: [],
  mapping: {},
  loading: false,
  applying: false,

  setEndpoints: (e) => set({ endpoints: e }),
  setCloudEndpoints: (e) => set({ cloudEndpoints: e }),
  setMapping: (m) => set({ mapping: m }),
  setLoading: (l) => set({ loading: l }),
  setApplying: (a) => set({ applying: a }),
}))
