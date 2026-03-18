import { create } from 'zustand'

export interface PromoterInfo {
  aff_code: string
  share_link: string
  gateway_url: string
  total_referrals: number
  total_earned: number
  pending_earned: number
}

interface PromoterState {
  info: PromoterInfo | null
  loading: boolean
  setInfo: (info: PromoterInfo) => void
  setLoading: (l: boolean) => void
}

export const usePromoterStore = create<PromoterState>((set) => ({
  info: null,
  loading: false,
  setInfo: (info) => set({ info }),
  setLoading: (loading) => set({ loading }),
}))
