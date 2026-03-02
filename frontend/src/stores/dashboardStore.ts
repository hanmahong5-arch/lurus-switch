import { create } from 'zustand'

export interface ToolStatus {
  name: string
  installed: boolean
  version: string
  latestVersion: string
  updateAvailable: boolean
  path: string
}

export interface ProxySettings {
  apiEndpoint: string
  apiKey: string
  registrationUrl?: string
  tenantSlug?: string
  userToken?: string
}

export interface UpdateInfo {
  name: string
  currentVersion: string
  latestVersion: string
  updateAvailable: boolean
  downloadUrl?: string
}

export interface ToolHealthResult {
  tool: string
  status: string // "green" | "yellow" | "red"
  issues: string[]
}

interface DashboardState {
  tools: Record<string, ToolStatus>
  installing: Record<string, boolean>
  updating: Record<string, boolean>
  detecting: boolean
  proxySettings: ProxySettings
  proxySaving: boolean
  proxyConfiguring: boolean
  appVersion: string
  selfUpdateInfo: UpdateInfo | null
  checkingUpdates: boolean
  error: string | null
  toolHealth: Record<string, ToolHealthResult>

  setTools: (tools: Record<string, ToolStatus>) => void
  setInstalling: (tool: string, installing: boolean) => void
  setUpdating: (tool: string, updating: boolean) => void
  setDetecting: (detecting: boolean) => void
  setProxySettings: (settings: ProxySettings) => void
  setProxySaving: (saving: boolean) => void
  setProxyConfiguring: (configuring: boolean) => void
  setAppVersion: (version: string) => void
  setSelfUpdateInfo: (info: UpdateInfo | null) => void
  setCheckingUpdates: (checking: boolean) => void
  setError: (error: string | null) => void
  setToolHealth: (h: Record<string, ToolHealthResult>) => void
}

export const useDashboardStore = create<DashboardState>((set) => ({
  tools: {},
  installing: {},
  updating: {},
  detecting: false,
  proxySettings: { apiEndpoint: '', apiKey: '' },
  proxySaving: false,
  proxyConfiguring: false,
  appVersion: '',
  selfUpdateInfo: null,
  checkingUpdates: false,
  error: null,
  toolHealth: {},

  setTools: (tools) => set({ tools }),
  setInstalling: (tool, installing) =>
    set((state) => ({
      installing: { ...state.installing, [tool]: installing },
    })),
  setUpdating: (tool, updating) =>
    set((state) => ({
      updating: { ...state.updating, [tool]: updating },
    })),
  setDetecting: (detecting) => set({ detecting }),
  setProxySettings: (settings) => set({ proxySettings: settings }),
  setProxySaving: (saving) => set({ proxySaving: saving }),
  setProxyConfiguring: (configuring) => set({ proxyConfiguring: configuring }),
  setAppVersion: (version) => set({ appVersion: version }),
  setSelfUpdateInfo: (info) => set({ selfUpdateInfo: info }),
  setCheckingUpdates: (checking) => set({ checkingUpdates: checking }),
  setError: (error) => set({ error }),
  setToolHealth: (h) => set({ toolHealth: h }),
}))
