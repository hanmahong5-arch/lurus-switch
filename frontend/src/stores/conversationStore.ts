import { create } from 'zustand'
import {
  ListConversations, GetConversation, ExportConversation, ReindexConversations,
  GetDLPHitsForSession, GetProjectContextFiles, ForkConversation,
} from '../../wailsjs/go/main/App'
import type { conversation, audit, main } from '../../wailsjs/go/models'

// Filter shape mirrors conversation.ConversationFilter on the Go side.
// Kept loose (every field optional) so the page can compose partial
// queries without juggling defaults.
export interface ConversationFilter {
  tool: string
  cwdSubstring: string
  model: string
  startAfter: string
  endBefore: string
  onlyDLPHits: boolean
  search: string
}

const defaultFilter: ConversationFilter = {
  tool: '', cwdSubstring: '', model: '', startAfter: '', endBefore: '',
  onlyDLPHits: false, search: '',
}

interface State {
  conversations: conversation.ConversationMeta[]
  active: main.ConversationEvents | null
  filter: ConversationFilter
  loading: boolean
  loadingActive: boolean
  reindexing: boolean
  forking: boolean
  error: string | null
  dlpHits: audit.Entry[]
  contextFiles: conversation.ContextFile[]

  list: () => Promise<void>
  open: (tool: string, sessionID: string) => Promise<void>
  setFilter: (patch: Partial<ConversationFilter>) => void
  resetFilter: () => void
  reindex: () => Promise<conversation.ReindexResult | null>
  exportSession: (tool: string, sessionID: string, format: 'markdown' | 'json', redact: boolean) => Promise<string | null>
  fork: (tool: string, sessionID: string, messageUUID: string) => Promise<conversation.ForkResult | null>
  clearActive: () => void
}

export const useConversationStore = create<State>((set, get) => ({
  conversations: [],
  active: null,
  filter: defaultFilter,
  loading: false,
  loadingActive: false,
  reindexing: false,
  forking: false,
  error: null,
  dlpHits: [],
  contextFiles: [],

  list: async () => {
    set({ loading: true, error: null })
    try {
      const rows = await ListConversations(get().filter as any)
      set({ conversations: (rows || []) as conversation.ConversationMeta[], loading: false })
    } catch (e: any) {
      set({ error: e?.message ?? String(e), loading: false })
    }
  },

  open: async (tool, sessionID) => {
    set({ loadingActive: true, error: null, active: null, dlpHits: [], contextFiles: [] })
    try {
      const data = await GetConversation(tool, sessionID)
      set({ active: data as main.ConversationEvents, loadingActive: false })
      // Side-channel: fetch DLP hits and context files in parallel —
      // they're not critical for first paint so we don't block on them.
      void GetDLPHitsForSession(tool, sessionID).then((hits) => {
        set({ dlpHits: (hits || []) as audit.Entry[] })
      }).catch(() => { /* leave hits empty */ })
      const cwd = (data as any)?.meta?.cwd
      if (cwd) {
        void GetProjectContextFiles(cwd).then((files) => {
          set({ contextFiles: (files || []) as conversation.ContextFile[] })
        }).catch(() => { /* leave empty */ })
      }
    } catch (e: any) {
      set({ error: e?.message ?? String(e), loadingActive: false })
    }
  },

  setFilter: (patch) => {
    set((s) => ({ filter: { ...s.filter, ...patch } }))
    void get().list()
  },

  resetFilter: () => {
    set({ filter: defaultFilter })
    void get().list()
  },

  reindex: async () => {
    set({ reindexing: true })
    try {
      const res = await ReindexConversations()
      await get().list()
      return res as conversation.ReindexResult
    } catch (e: any) {
      set({ error: e?.message ?? String(e) })
      return null
    } finally {
      set({ reindexing: false })
    }
  },

  exportSession: async (tool, sessionID, format, redact) => {
    try {
      return await ExportConversation(tool, sessionID, format, redact)
    } catch (e: any) {
      set({ error: e?.message ?? String(e) })
      return null
    }
  },

  fork: async (tool, sessionID, messageUUID) => {
    set({ forking: true, error: null })
    try {
      const res = await ForkConversation(tool, sessionID, messageUUID)
      // Refresh the list so the new child appears.
      await get().list()
      return res as conversation.ForkResult
    } catch (e: any) {
      set({ error: e?.message ?? String(e) })
      return null
    } finally {
      set({ forking: false })
    }
  },

  clearActive: () => set({ active: null, dlpHits: [], contextFiles: [] }),
}))
