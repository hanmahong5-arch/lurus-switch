import { create } from 'zustand'
import { GetChargebackReport, SetAppOwnership } from '../../wailsjs/go/main/App'

export interface ChargebackRow {
  kind: 'department' | 'employee'
  deptId?: string
  deptName?: string
  employeeId?: string
  email?: string
  displayName?: string
  costCenter?: string
  totalCalls: number
  tokensIn: number
  tokensOut: number
  uniqueEmployees?: number
}

export interface ChargebackReport {
  fromMs: number
  toMs: number
  byDepartment: ChargebackRow[]
  byEmployee: ChargebackRow[]
}

type ViewKind = 'department' | 'employee'

interface State {
  fromMs: number
  toMs: number
  view: ViewKind
  report: ChargebackReport | null
  loading: boolean
  error: string | null

  setRange: (fromMs: number, toMs: number) => void
  setView: (v: ViewKind) => void
  load: () => Promise<void>
  bindAppOwnership: (appId: string, employeeId: string, costCenter: string) => Promise<void>
}

const DAY = 24 * 60 * 60 * 1000

function defaultRange() {
  const to = Date.now()
  const from = to - 7 * DAY
  return { fromMs: from, toMs: to }
}

export const useChargebackStore = create<State>((set, get) => ({
  ...defaultRange(),
  view: 'department',
  report: null,
  loading: false,
  error: null,

  setRange: (fromMs, toMs) => {
    set({ fromMs, toMs })
    void get().load()
  },

  setView: (v) => set({ view: v }),

  load: async () => {
    const { fromMs, toMs } = get()
    set({ loading: true, error: null })
    try {
      const r = await GetChargebackReport(fromMs, toMs)
      set({ report: r as unknown as ChargebackReport, loading: false })
    } catch (e: any) {
      set({ error: e?.message ?? String(e), loading: false })
    }
  },

  bindAppOwnership: async (appId, employeeId, costCenter) => {
    try {
      await SetAppOwnership(appId, employeeId, costCenter)
      await get().load()
    } catch (e: any) {
      set({ error: e?.message ?? String(e) })
    }
  },
}))
