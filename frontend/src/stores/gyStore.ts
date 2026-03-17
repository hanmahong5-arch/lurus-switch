import { create } from 'zustand'
import type { gy } from '../../wailsjs/go/models'

interface GYState {
  products: gy.GYProduct[]
  statuses: Record<string, gy.GYStatus>
  loading: boolean
  checking: boolean

  setProducts: (p: gy.GYProduct[]) => void
  setStatuses: (s: Record<string, gy.GYStatus>) => void
  setLoading: (l: boolean) => void
  setChecking: (c: boolean) => void
}

export const useGYStore = create<GYState>((set) => ({
  products: [],
  statuses: {},
  loading: false,
  checking: false,

  setProducts: (p) => set({ products: p }),
  setStatuses: (s) => set({ statuses: s }),
  setLoading: (l) => set({ loading: l }),
  setChecking: (c) => set({ checking: c }),
}))
