import { create } from 'zustand'
import { ListBuiltinTemplates } from '../../wailsjs/go/main/App'

// Mirror of internal/agenttemplate.Template. Templates are read-only
// recipes shipped with the build — there's no create/update/delete.

export interface AgentTemplate {
  id: string
  displayName: string
  icon: string
  toolType: string
  modelId: string
  systemPrompt: string
  tags: string[]
  mcpServers: string[]
  capabilities: string[]
  budgetTokens: number
  budgetUsd: number
  budgetPeriod: string // 'daily' / 'weekly' / 'monthly'
  budgetPolicy: string // 'hard_stop' / 'soft_warn' / 'approval'
  guardrails: string[]
  useCases: string[]
  notes: string
}

interface State {
  templates: AgentTemplate[]
  loading: boolean
  error: string | null
  selectedId: string
  load: () => Promise<void>
  select: (id: string) => void
}

export const useAgentTemplateStore = create<State>((set) => ({
  templates: [],
  loading: false,
  error: null,
  selectedId: '',

  load: async () => {
    set({ loading: true, error: null })
    try {
      const list = await ListBuiltinTemplates()
      set({ templates: (list || []) as unknown as AgentTemplate[], loading: false })
    } catch (e: any) {
      set({ error: e?.message ?? String(e), loading: false })
    }
  },

  select: (id) => set({ selectedId: id }),
}))
