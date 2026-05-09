import { create } from 'zustand'
import {
  ListAuditEntries, GetAuditStats, UndoAuditEntry, ListAuditCapabilities,
  GetCurrentPrincipal,
} from '../../wailsjs/go/main/App'

// Mirror of audit.Entry from internal/audit. Keep names in sync with the
// Go side — Wails generates the model but we re-declare here to avoid
// crossing the namespace import boundary in components.
export interface AuditEntry {
  id: string
  timestamp: string
  principal: string
  capsHeld: string[]
  operation: string
  target: string
  before?: unknown
  after?: unknown
  outcome: 'ok' | 'denied' | 'error'
  error?: string
  undoneAt?: string | null
  undoneBy?: string
  reversible: boolean
  metadata?: Record<string, string>
}

export interface AuditStats {
  total: number
  ok: number
  denied: number
  error: number
  undone: number
  byPrincipal: Record<string, number>
  byOperation: Record<string, number>
}

export interface AuditFilter {
  principal: string
  operation: string
  outcome: string
  onlyReversible: boolean
  onlyUndone: boolean
  onlyNotUndone: boolean
}

const defaultFilter: AuditFilter = {
  principal: '',
  operation: '',
  outcome: '',
  onlyReversible: false,
  onlyUndone: false,
  onlyNotUndone: false,
}

interface State {
  entries: AuditEntry[]
  stats: AuditStats | null
  capabilities: Record<string, string>
  principal: string
  filter: AuditFilter
  loading: boolean
  error: string | null
  undoingId: string | null

  load: () => Promise<void>
  loadCapabilities: () => Promise<void>
  setFilter: (patch: Partial<AuditFilter>) => void
  resetFilter: () => void
  undo: (entryId: string) => Promise<void>
}

export const useAuditStore = create<State>((set, get) => ({
  entries: [],
  stats: null,
  capabilities: {},
  principal: '',
  filter: defaultFilter,
  loading: false,
  error: null,
  undoingId: null,

  load: async () => {
    set({ loading: true, error: null })
    try {
      const f = get().filter
      const [entries, stats, principal] = await Promise.all([
        ListAuditEntries(200, f as any),
        GetAuditStats(),
        GetCurrentPrincipal(),
      ])
      set({
        entries: (entries || []) as unknown as AuditEntry[],
        stats: stats as unknown as AuditStats,
        principal,
        loading: false,
      })
    } catch (e: any) {
      set({ error: e?.message ?? String(e), loading: false })
    }
  },

  loadCapabilities: async () => {
    try {
      const caps = await ListAuditCapabilities()
      set({ capabilities: caps || {} })
    } catch {
      // best-effort; leave map empty
    }
  },

  setFilter: (patch) => {
    set((s) => ({ filter: { ...s.filter, ...patch } }))
    get().load()
  },

  resetFilter: () => {
    set({ filter: defaultFilter })
    get().load()
  },

  undo: async (entryId) => {
    set({ undoingId: entryId, error: null })
    try {
      await UndoAuditEntry(entryId)
      await get().load()
    } catch (e: any) {
      set({ error: e?.message ?? String(e) })
    } finally {
      set({ undoingId: null })
    }
  },
}))
