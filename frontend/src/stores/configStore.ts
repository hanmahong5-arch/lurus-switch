import { create } from 'zustand'

export type AppMode = 'user' | 'promoter'
export type UserLevel = 'beginner' | 'regular' | 'power'

// New consolidated navigation — 6 user-facing + 2 promoter-only
export type ActiveTool =
  | 'home'
  | 'tools'
  | 'gateway'
  | 'workspace'
  | 'account'
  | 'settings'
  | 'promotion'
  | 'api-admin'

// Sub-tab identifiers for each page
export type ToolsSubTab = 'claude' | 'codex' | 'gemini' | 'picoclaw' | 'nullclaw' | 'zeroclaw' | 'openclaw' | 'mcp' | 'snapshots'
export type GatewaySubTab = 'control' | 'usage' | 'apps' | 'relay'
export type WorkspaceSubTab = 'prompts' | 'context' | 'process'
export type AccountSubTab = 'connection' | 'billing'
export type ApiAdminSubTab = 'server' | 'dashboard' | 'channels' | 'tokens' | 'models' | 'users' | 'redemptions' | 'logs' | 'subscriptions' | 'admin-settings' | 'system'

// Legacy route values for backward compatibility (startupPage, etc.)
type LegacyActiveTool =
  | 'dashboard' | 'switch-hub' | 'gy-products'
  | 'claude' | 'codex' | 'gemini' | 'picoclaw' | 'nullclaw' | 'zeroclaw' | 'openclaw'
  | 'billing' | 'process' | 'prompts' | 'documents' | 'admin'
  | 'relay' | 'cli-runner' | 'promoter-hub'
  | 'gateway-old' | 'gateway-dashboard' | 'gateway-channels' | 'gateway-tokens'
  | 'gateway-models' | 'gateway-users' | 'gateway-redemptions' | 'gateway-logs'
  | 'gateway-subscriptions' | 'gateway-settings'

const TOOL_NAMES: ToolsSubTab[] = ['claude', 'codex', 'gemini', 'picoclaw', 'nullclaw', 'zeroclaw', 'openclaw']

// Map legacy route values to new navigation
export function migrateLegacyRoute(legacy: string): { tool: ActiveTool; subTab?: string } {
  switch (legacy) {
    case 'dashboard':
    case 'switch-hub':
    case 'gy-products':
      return { tool: 'home' }
    case 'claude':
    case 'codex':
    case 'gemini':
    case 'picoclaw':
    case 'nullclaw':
    case 'zeroclaw':
    case 'openclaw':
      return { tool: 'tools', subTab: legacy }
    case 'relay':
      return { tool: 'gateway', subTab: 'relay' }
    case 'billing':
      return { tool: 'account', subTab: 'billing' }
    case 'process':
    case 'cli-runner':
      return { tool: 'workspace', subTab: 'process' }
    case 'prompts':
      return { tool: 'workspace', subTab: 'prompts' }
    case 'documents':
      return { tool: 'workspace', subTab: 'context' }
    case 'admin':
      return { tool: 'api-admin', subTab: 'system' }
    case 'promoter-hub':
      return { tool: 'promotion' }
    case 'gateway-old':
      return { tool: 'api-admin', subTab: 'server' }
    case 'gateway-dashboard':
      return { tool: 'api-admin', subTab: 'dashboard' }
    case 'gateway-channels':
      return { tool: 'api-admin', subTab: 'channels' }
    case 'gateway-tokens':
      return { tool: 'api-admin', subTab: 'tokens' }
    case 'gateway-models':
      return { tool: 'api-admin', subTab: 'models' }
    case 'gateway-users':
      return { tool: 'api-admin', subTab: 'users' }
    case 'gateway-redemptions':
      return { tool: 'api-admin', subTab: 'redemptions' }
    case 'gateway-logs':
      return { tool: 'api-admin', subTab: 'logs' }
    case 'gateway-subscriptions':
      return { tool: 'api-admin', subTab: 'subscriptions' }
    case 'gateway-settings':
      return { tool: 'api-admin', subTab: 'admin-settings' }
    case 'settings':
      return { tool: 'settings' }
    default:
      return { tool: 'home' }
  }
}

export function isToolSubTab(tab: string): tab is ToolsSubTab {
  return TOOL_NAMES.includes(tab as ToolsSubTab)
}

export interface ConfigPreset {
  id: string
  tool: string
  name: string
  description: string
  category: string
  config_json: Record<string, unknown>
  is_official: boolean
}

interface ConfigState {
  appMode: AppMode
  setAppMode: (mode: AppMode) => void

  userLevel: UserLevel
  setUserLevel: (level: UserLevel) => void

  activeTool: ActiveTool
  setActiveTool: (tool: ActiveTool) => void

  // Sub-tab state per page, persisted across navigation
  subTabState: Record<string, string>
  setSubTab: (page: ActiveTool, tab: string) => void
  getSubTab: (page: ActiveTool, defaultTab: string) => string

  // Legacy compat: last active tool config tab
  lastActiveTool: ToolsSubTab
  setLastActiveTool: (tool: ToolsSubTab) => void

  activeSection: string
  setActiveSection: (section: string) => void

  previewContent: string
  setPreviewContent: (content: string) => void

  status: string
  setStatus: (status: string) => void

  savedConfigs: Record<string, string[]>
  setSavedConfigs: (tool: string, configs: string[]) => void

  cloudPresets: Record<string, ConfigPreset[]>
  setCloudPresets: (tool: string, presets: ConfigPreset[]) => void

  highlightField: string
  setHighlightField: (field: string) => void
}

export const useConfigStore = create<ConfigState>((set, get) => ({
  appMode: 'user',
  setAppMode: (mode) => set({ appMode: mode }),

  userLevel: 'beginner',
  setUserLevel: (level) => set({ userLevel: level }),

  activeTool: 'home',
  setActiveTool: (tool) => set({ activeTool: tool }),

  subTabState: {},
  setSubTab: (page, tab) =>
    set((state) => ({
      subTabState: { ...state.subTabState, [page]: tab },
    })),
  getSubTab: (page, defaultTab) => {
    return get().subTabState[page] || defaultTab
  },

  lastActiveTool: 'claude',
  setLastActiveTool: (tool) => set({ lastActiveTool: tool }),

  activeSection: 'core',
  setActiveSection: (section) => set({ activeSection: section }),

  previewContent: '',
  setPreviewContent: (content) => set({ previewContent: content }),

  status: 'Ready',
  setStatus: (status) => set({ status: status }),

  savedConfigs: {
    claude: [],
    codex: [],
    gemini: [],
    picoclaw: [],
    nullclaw: [],
    zeroclaw: [],
    openclaw: [],
  },
  setSavedConfigs: (tool, configs) =>
    set((state) => ({
      savedConfigs: { ...state.savedConfigs, [tool]: configs },
    })),

  cloudPresets: {},
  setCloudPresets: (tool, presets) =>
    set((state) => ({
      cloudPresets: { ...state.cloudPresets, [tool]: presets },
    })),

  highlightField: '',
  setHighlightField: (field) => set({ highlightField: field }),
}))
