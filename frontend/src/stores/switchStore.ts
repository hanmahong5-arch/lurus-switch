import { create } from 'zustand'

// Types matching the Go bindings (gateway.Status, appreg.App, metering.*)
export interface GatewayLocalStatus {
  running: boolean
  port: number
  url: string
  uptime: number
  totalRequests: number
  activeConns: number
}

export interface GatewayLocalConfig {
  port: number
  upstreamUrl: string
  userToken: string
  autoStart: boolean
}

export interface RegisteredApp {
  id: string
  name: string
  kind: string // 'builtin' | 'user'
  tier: number
  token: string
  icon: string
  description: string
  createdAt: string
  lastSeenAt?: string
  connected: boolean
}

export interface DailySummary {
  date: string
  totalCalls: number
  tokensIn: number
  tokensOut: number
  cacheHits: number
}

export interface AppSummary {
  appId: string
  totalCalls: number
  tokensIn: number
  tokensOut: number
  cacheHits: number
}

export interface ModelSummary {
  model: string
  totalCalls: number
  tokensIn: number
  tokensOut: number
}

export interface ActivityEntry {
  timestamp: string
  appId: string
  model: string
  tokens: number
}

// Environment diagnostics types (matches Go main.ToolDiagnostic etc.)
export interface ToolDiagnostic {
  tool: string
  installed: boolean
  version: string
  path: string
  configExists: boolean
  healthStatus: string // 'green' | 'yellow' | 'red' | 'unknown'
  healthIssues: string[]
  gatewayBound: boolean
  connected: boolean
  currentEndpoint: string
  currentModel: string
}

export interface RuntimeDiagnostic {
  id: string
  name: string
  installed: boolean
  version: string
  required: boolean
}

export interface EnvironmentCheckResult {
  tools: ToolDiagnostic[]
  runtimes: RuntimeDiagnostic[]
  gatewayRunning: boolean
  gatewayUrl: string
  allToolsBound: boolean
  installedCount: number
  boundCount: number
}

export interface FullSetupResult {
  gatewayStarted: boolean
  snapshotsTaken: number
  configResults: ToolConfigResult[]
  gatewayUrl: string
  errors: string[]
}

export interface ToolConfigResult {
  tool: string
  success: boolean
  message: string
}

export interface ToolSnapshotInfo {
  id: string
  tool: string
  label: string
  createdAt: string
  size: number
}

interface SwitchState {
  // Gateway status
  status: GatewayLocalStatus | null
  config: GatewayLocalConfig | null
  loading: boolean
  starting: boolean
  stopping: boolean

  // App registry
  apps: RegisteredApp[]
  appsLoading: boolean

  // Metering
  todaySummary: DailySummary | null
  daySummaries: DailySummary[]
  appSummaries: AppSummary[]
  modelSummaries: ModelSummary[]
  recentActivity: ActivityEntry[]
  meteringPeriod: string // 'today' | 'week' | 'month'

  // Environment diagnostics
  envCheck: EnvironmentCheckResult | null
  envLoading: boolean
  configResults: ToolConfigResult[]
  configuring: boolean

  // Polling
  pollHandle: ReturnType<typeof setInterval> | null

  // Actions
  setStatus: (s: GatewayLocalStatus) => void
  setConfig: (c: GatewayLocalConfig) => void
  setLoading: (l: boolean) => void
  setStarting: (s: boolean) => void
  setStopping: (s: boolean) => void
  setApps: (apps: RegisteredApp[]) => void
  setAppsLoading: (l: boolean) => void
  setTodaySummary: (s: DailySummary) => void
  setDaySummaries: (s: DailySummary[]) => void
  setAppSummaries: (s: AppSummary[]) => void
  setModelSummaries: (s: ModelSummary[]) => void
  setRecentActivity: (a: ActivityEntry[]) => void
  setMeteringPeriod: (p: string) => void
  setPollHandle: (h: ReturnType<typeof setInterval> | null) => void
  setEnvCheck: (e: EnvironmentCheckResult) => void
  setEnvLoading: (l: boolean) => void
  setConfigResults: (r: ToolConfigResult[]) => void
  setConfiguring: (c: boolean) => void
}

export const useSwitchStore = create<SwitchState>((set) => ({
  status: null,
  config: null,
  loading: false,
  starting: false,
  stopping: false,

  apps: [],
  appsLoading: false,

  todaySummary: null,
  daySummaries: [],
  appSummaries: [],
  modelSummaries: [],
  recentActivity: [],
  meteringPeriod: 'today',

  envCheck: null,
  envLoading: false,
  configResults: [],
  configuring: false,

  pollHandle: null,

  setStatus: (s) => set({ status: s }),
  setConfig: (c) => set({ config: c }),
  setLoading: (l) => set({ loading: l }),
  setStarting: (s) => set({ starting: s }),
  setStopping: (s) => set({ stopping: s }),
  setApps: (apps) => set({ apps }),
  setAppsLoading: (l) => set({ appsLoading: l }),
  setTodaySummary: (s) => set({ todaySummary: s }),
  setDaySummaries: (s) => set({ daySummaries: s }),
  setAppSummaries: (s) => set({ appSummaries: s }),
  setModelSummaries: (s) => set({ modelSummaries: s }),
  setRecentActivity: (a) => set({ recentActivity: a }),
  setMeteringPeriod: (p) => set({ meteringPeriod: p }),
  setPollHandle: (h) => set({ pollHandle: h }),
  setEnvCheck: (e) => set({ envCheck: e }),
  setEnvLoading: (l) => set({ envLoading: l }),
  setConfigResults: (r) => set({ configResults: r }),
  setConfiguring: (c) => set({ configuring: c }),
}))
