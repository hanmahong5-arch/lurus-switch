import { create } from 'zustand'
import {
  CreateAgent, ListAgents, GetAgent, UpdateAgent, DeleteAgent,
  LaunchAgent, StopAgent, CloneAgent, GetAgentStats
} from '../../wailsjs/go/main/App'

export interface AgentProfile {
  id: string
  name: string
  icon: string
  tags: string[]
  toolType: string
  modelId: string
  systemPrompt: string
  mcpServers: string[]
  permissions: { allowShell: boolean; allowFiles: boolean; allowNetwork: boolean }
  projectId?: string
  status: 'created' | 'running' | 'stopped' | 'error'
  configDir?: string
  createdAt: string
  updatedAt: string
  budgetLimitTokens?: number
  budgetLimitCurrency?: number
  budgetPeriod: string
  budgetPolicy: string
}

export interface AgentStats {
  total: number
  running: number
  stopped: number
  error: number
  created: number
}

export interface CreateAgentParams {
  name: string
  icon: string
  tags: string[]
  toolType: string
  modelId: string
  systemPrompt?: string
  mcpServers?: string[]
  permissions?: { allowShell: boolean; allowFiles: boolean; allowNetwork: boolean }
  projectId?: string
  budgetLimitTokens?: number
  budgetLimitCurrency?: number
  budgetPeriod?: string
  budgetPolicy?: string
}

interface AgentState {
  agents: AgentProfile[]
  stats: AgentStats
  loading: boolean
  error: string | null
  selectedAgentId: string | null

  loadAgents: () => Promise<void>
  loadStats: () => Promise<void>
  createAgent: (params: CreateAgentParams) => Promise<AgentProfile>
  updateAgent: (id: string, params: Partial<CreateAgentParams>) => Promise<void>
  deleteAgent: (id: string) => Promise<void>
  launchAgent: (id: string) => Promise<void>
  stopAgent: (id: string) => Promise<void>
  cloneAgent: (id: string, newName: string) => Promise<void>
  selectAgent: (id: string | null) => void
}

export const useAgentStore = create<AgentState>((set, get) => ({
  agents: [],
  stats: { total: 0, running: 0, stopped: 0, error: 0, created: 0 },
  loading: false,
  error: null,
  selectedAgentId: null,

  loadAgents: async () => {
    set({ loading: true, error: null })
    try {
      const agents = await ListAgents(null as any)
      set({ agents: (agents || []) as unknown as AgentProfile[], loading: false })
    } catch (e: any) {
      set({ error: e?.message || String(e), loading: false })
    }
  },

  loadStats: async () => {
    try {
      const stats = await GetAgentStats()
      set({ stats })
    } catch {}
  },

  createAgent: async (params) => {
    const agent = await CreateAgent(params as any)
    await get().loadAgents()
    await get().loadStats()
    return agent as unknown as AgentProfile
  },

  updateAgent: async (id, params) => {
    await UpdateAgent(id, params as any)
    await get().loadAgents()
  },

  deleteAgent: async (id) => {
    await DeleteAgent(id)
    const { selectedAgentId } = get()
    if (selectedAgentId === id) {
      set({ selectedAgentId: null })
    }
    await get().loadAgents()
    await get().loadStats()
  },

  launchAgent: async (id) => {
    await LaunchAgent(id)
    // Refresh to pick up running status
    await get().loadAgents()
    await get().loadStats()
  },

  stopAgent: async (id) => {
    await StopAgent(id)
    await get().loadAgents()
    await get().loadStats()
  },

  cloneAgent: async (id, newName) => {
    await CloneAgent(id, newName)
    await get().loadAgents()
    await get().loadStats()
  },

  selectAgent: (id) => set({ selectedAgentId: id }),
}))
