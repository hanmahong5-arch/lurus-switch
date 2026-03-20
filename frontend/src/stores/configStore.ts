import { create } from 'zustand'

export type AppMode = 'user' | 'promoter'

export type ActiveTool =
  | 'dashboard'
  | 'claude'
  | 'codex'
  | 'gemini'
  | 'picoclaw'
  | 'nullclaw'
  | 'zeroclaw'
  | 'openclaw'
  | 'billing'
  | 'settings'
  | 'process'
  | 'prompts'
  | 'documents'
  | 'admin'
  | 'relay'
  | 'gy-products'
  | 'cli-runner'
  | 'promoter-hub'
  | 'gateway'
  | 'gateway-dashboard'
  | 'gateway-channels'
  | 'gateway-tokens'
  | 'gateway-models'
  | 'gateway-users'
  | 'gateway-redemptions'
  | 'gateway-logs'
  | 'gateway-subscriptions'
  | 'gateway-settings'
  | 'switch-hub'

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

  activeTool: ActiveTool
  setActiveTool: (tool: ActiveTool) => void

  lastActiveTool: ActiveTool
  setLastActiveTool: (tool: ActiveTool) => void

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

export const useConfigStore = create<ConfigState>((set) => ({
  appMode: 'user',
  setAppMode: (mode) => set({ appMode: mode }),

  activeTool: 'dashboard',
  setActiveTool: (tool) => set({ activeTool: tool }),

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
