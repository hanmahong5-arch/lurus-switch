import { create } from 'zustand'
import type { relay } from '../../wailsjs/go/models'
import { GetRelayCircuitState } from '../../wailsjs/go/main/App'

/** Tool name → relay endpoint ID */
type ToolRelayMapping = Record<string, string>

interface RelayState {
  endpoints: relay.RelayEndpoint[]
  cloudEndpoints: relay.RelayEndpoint[]
  mapping: ToolRelayMapping
  loading: boolean
  applying: boolean
  /** Per-endpoint circuit-breaker state, keyed by RelayEndpoint.ID */
  circuitState: Record<string, relay.CircuitState>

  setEndpoints: (e: relay.RelayEndpoint[]) => void
  setCloudEndpoints: (e: relay.RelayEndpoint[]) => void
  setMapping: (m: ToolRelayMapping) => void
  setLoading: (l: boolean) => void
  setApplying: (a: boolean) => void
  pollCircuitState: () => Promise<void>
}

export const useRelayStore = create<RelayState>((set) => ({
  endpoints: [],
  cloudEndpoints: [],
  mapping: {},
  loading: false,
  applying: false,
  circuitState: {},

  setEndpoints: (e) => set({ endpoints: e }),
  setCloudEndpoints: (e) => set({ cloudEndpoints: e }),
  setMapping: (m) => set({ mapping: m }),
  setLoading: (l) => set({ loading: l }),
  setApplying: (a) => set({ applying: a }),
  pollCircuitState: async () => {
    try {
      const rows = (await GetRelayCircuitState()) as relay.CircuitState[]
      const map: Record<string, relay.CircuitState> = {}
      for (const r of rows || []) {
        if (r && r.endpointID) map[r.endpointID] = r
      }
      set({ circuitState: map })
    } catch {
      // non-critical; leave previous state in place
    }
  },
}))
