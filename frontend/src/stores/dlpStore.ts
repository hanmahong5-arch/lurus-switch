import { create } from 'zustand'
import {
  ListDLPPatterns, ListDLPHits, GetDLPStats, SetDLPPolicy, ScanText,
  AddDLPPattern, RemoveDLPPattern,
} from '../../wailsjs/go/main/App'

// Mirrors of internal/dlp Go types. Keeping a hand-rolled copy avoids
// crossing the Wails models barrel and lets the UI evolve field types
// (e.g., `Severity` widening) without breaking tsc.

export type DLPSeverity = 'info' | 'warning' | 'critical'
export type DLPPolicy = 'allow' | 'redact' | 'block' | 'warn'

export interface DLPPattern {
  name: string
  description: string
  regex: string
  severity: DLPSeverity
  policy: DLPPolicy
  tags?: string[]
}

export interface DLPHit {
  patternName: string
  severity: DLPSeverity
  policy: DLPPolicy
  start: number
  end: number
  snippet: string
}

export interface DLPHitRecord {
  timestamp: string
  source: string
  path: string
  hit: DLPHit
}

export interface DLPResult {
  hits: DLPHit[]
  highestPolicy: DLPPolicy
  blocked: boolean
  redacted: string
}

export interface DLPStats {
  total: number
  bySeverity: Record<string, number>
  byPolicy: Record<string, number>
  byPattern: Record<string, number>
  bySource: Record<string, number>
}

interface State {
  patterns: DLPPattern[]
  hits: DLPHitRecord[]
  stats: DLPStats | null
  loading: boolean
  error: string | null
  scanResult: DLPResult | null
  scanInput: string

  load: () => Promise<void>
  setPolicy: (name: string, policy: DLPPolicy) => Promise<void>
  removePattern: (name: string) => Promise<void>
  addPattern: (p: DLPPattern) => Promise<void>
  scan: (input: string) => Promise<void>
  setScanInput: (input: string) => void
}

export const useDLPStore = create<State>((set, get) => ({
  patterns: [],
  hits: [],
  stats: null,
  loading: false,
  error: null,
  scanResult: null,
  scanInput: '',

  load: async () => {
    set({ loading: true, error: null })
    try {
      const [patterns, hits, stats] = await Promise.all([
        ListDLPPatterns(),
        ListDLPHits(50),
        GetDLPStats(),
      ])
      set({
        patterns: (patterns || []) as unknown as DLPPattern[],
        hits: (hits || []) as unknown as DLPHitRecord[],
        stats: (stats || null) as unknown as DLPStats | null,
        loading: false,
      })
    } catch (e: any) {
      set({ error: e?.message ?? String(e), loading: false })
    }
  },

  setPolicy: async (name, policy) => {
    try {
      await SetDLPPolicy(name, policy)
      await get().load()
    } catch (e: any) {
      set({ error: e?.message ?? String(e) })
    }
  },

  removePattern: async (name) => {
    try {
      await RemoveDLPPattern(name)
      await get().load()
    } catch (e: any) {
      set({ error: e?.message ?? String(e) })
    }
  },

  addPattern: async (p) => {
    try {
      await AddDLPPattern(p as any)
      await get().load()
    } catch (e: any) {
      set({ error: e?.message ?? String(e) })
    }
  },

  scan: async (input) => {
    if (!input.trim()) {
      set({ scanResult: null })
      return
    }
    try {
      const r = await ScanText(input)
      set({ scanResult: r as unknown as DLPResult })
    } catch (e: any) {
      set({ error: e?.message ?? String(e) })
    }
  },

  setScanInput: (input) => set({ scanInput: input }),
}))
