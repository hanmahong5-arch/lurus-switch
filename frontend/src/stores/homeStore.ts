import { create } from 'zustand'
import type { ToolStatus, ProxySettings, UpdateInfo, ToolHealthResult } from './dashboardStore'
import type {
  GatewayLocalStatus, EnvironmentCheckResult,
  ToolConfigResult,
} from './switchStore'

// Health score types matching Go healthscore package
export interface CategoryScore {
  category: string
  score: number
  max: number
  label: string
  issues: string[]
}

export interface Suggestion {
  id: string
  priority: number
  title: string
  action: string
  target: string
}

export interface ScoreReport {
  totalScore: number
  maxScore: number
  categories: CategoryScore[]
  suggestions: Suggestion[]
}

interface HomeState {
  // Health score
  scoreReport: ScoreReport | null
  scoreLoading: boolean

  // Tool status (from dashboardStore)
  tools: Record<string, ToolStatus>
  installing: Record<string, boolean>
  updating: Record<string, boolean>
  detecting: boolean
  toolHealth: Record<string, ToolHealthResult>

  // Proxy (from dashboardStore)
  proxySettings: ProxySettings
  proxySaving: boolean
  proxyConfiguring: boolean

  // App version & self-update
  appVersion: string
  selfUpdateInfo: UpdateInfo | null
  checkingUpdates: boolean

  // Gateway status (from switchStore)
  gatewayStatus: GatewayLocalStatus | null

  // Environment check (from switchStore)
  envCheck: EnvironmentCheckResult | null
  envLoading: boolean
  configResults: ToolConfigResult[]
  configuring: boolean

  // GY Products loading
  gyLoading: boolean

  // Error
  error: string | null

  // Actions
  setScoreReport: (r: ScoreReport | null) => void
  setScoreLoading: (l: boolean) => void
  setTools: (tools: Record<string, ToolStatus>) => void
  setInstalling: (tool: string, installing: boolean) => void
  setUpdating: (tool: string, updating: boolean) => void
  setDetecting: (d: boolean) => void
  setToolHealth: (h: Record<string, ToolHealthResult>) => void
  setProxySettings: (s: ProxySettings) => void
  setProxySaving: (s: boolean) => void
  setProxyConfiguring: (c: boolean) => void
  setAppVersion: (v: string) => void
  setSelfUpdateInfo: (i: UpdateInfo | null) => void
  setCheckingUpdates: (c: boolean) => void
  setGatewayStatus: (s: GatewayLocalStatus | null) => void
  setEnvCheck: (e: EnvironmentCheckResult | null) => void
  setEnvLoading: (l: boolean) => void
  setConfigResults: (r: ToolConfigResult[]) => void
  setConfiguring: (c: boolean) => void
  setGYLoading: (l: boolean) => void
  setError: (e: string | null) => void
}

export const useHomeStore = create<HomeState>((set) => ({
  scoreReport: null,
  scoreLoading: false,

  tools: {},
  installing: {},
  updating: {},
  detecting: false,
  toolHealth: {},

  proxySettings: { apiEndpoint: '', apiKey: '' },
  proxySaving: false,
  proxyConfiguring: false,

  appVersion: '',
  selfUpdateInfo: null,
  checkingUpdates: false,

  gatewayStatus: null,

  envCheck: null,
  envLoading: false,
  configResults: [],
  configuring: false,

  gyLoading: false,

  error: null,

  setScoreReport: (r) => set({ scoreReport: r }),
  setScoreLoading: (l) => set({ scoreLoading: l }),
  setTools: (tools) => set({ tools }),
  setInstalling: (tool, installing) =>
    set((state) => ({
      installing: { ...state.installing, [tool]: installing },
    })),
  setUpdating: (tool, updating) =>
    set((state) => ({
      updating: { ...state.updating, [tool]: updating },
    })),
  setDetecting: (d) => set({ detecting: d }),
  setToolHealth: (h) => set({ toolHealth: h }),
  setProxySettings: (s) => set({ proxySettings: s }),
  setProxySaving: (s) => set({ proxySaving: s }),
  setProxyConfiguring: (c) => set({ proxyConfiguring: c }),
  setAppVersion: (v) => set({ appVersion: v }),
  setSelfUpdateInfo: (i) => set({ selfUpdateInfo: i }),
  setCheckingUpdates: (c) => set({ checkingUpdates: c }),
  setGatewayStatus: (s) => set({ gatewayStatus: s }),
  setEnvCheck: (e) => set({ envCheck: e }),
  setEnvLoading: (l) => set({ envLoading: l }),
  setConfigResults: (r) => set({ configResults: r }),
  setConfiguring: (c) => set({ configuring: c }),
  setGYLoading: (l) => set({ gyLoading: l }),
  setError: (e) => set({ error: e }),
}))
