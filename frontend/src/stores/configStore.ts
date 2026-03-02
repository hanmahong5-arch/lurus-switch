import { create } from 'zustand'

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
  activeTool: ActiveTool
  setActiveTool: (tool: ActiveTool) => void

  previewContent: string
  setPreviewContent: (content: string) => void

  status: string
  setStatus: (status: string) => void

  savedConfigs: Record<string, string[]>
  setSavedConfigs: (tool: string, configs: string[]) => void

  cloudPresets: Record<string, ConfigPreset[]>
  setCloudPresets: (tool: string, presets: ConfigPreset[]) => void
}

export const useConfigStore = create<ConfigState>((set) => ({
  activeTool: 'dashboard',
  setActiveTool: (tool) => set({ activeTool: tool }),

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
}))
