import { create } from 'zustand'
import type { billing } from '../../wailsjs/go/models'

// Re-export Wails types for convenience
export type UserBillingInfo = billing.UserInfo
export type SubscriptionInfo = billing.SubscriptionInfo
export type SubscriptionPlan = billing.SubscriptionPlan
export type TopUpInfo = billing.TopUpInfo
export type IdentityOverview = billing.IdentityOverview

interface BillingState {
  userInfo: UserBillingInfo | null
  plans: SubscriptionPlan[]
  subscriptions: SubscriptionInfo[]
  topUpInfo: TopUpInfo | null
  identityOverview: IdentityOverview | null
  loading: boolean
  error: string | null

  setUserInfo: (info: UserBillingInfo | null) => void
  setPlans: (plans: SubscriptionPlan[]) => void
  setSubscriptions: (subs: SubscriptionInfo[]) => void
  setTopUpInfo: (info: TopUpInfo | null) => void
  setIdentityOverview: (ov: IdentityOverview | null) => void
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void
}

export const useBillingStore = create<BillingState>((set) => ({
  userInfo: null,
  plans: [],
  subscriptions: [],
  topUpInfo: null,
  identityOverview: null,
  loading: false,
  error: null,

  setUserInfo: (info) => set({ userInfo: info }),
  setPlans: (plans) => set({ plans }),
  setSubscriptions: (subs) => set({ subscriptions: subs }),
  setTopUpInfo: (info) => set({ topUpInfo: info }),
  setIdentityOverview: (ov) => set({ identityOverview: ov }),
  setLoading: (loading) => set({ loading }),
  setError: (error) => set({ error }),
}))
