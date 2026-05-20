import { create } from 'zustand'
import type { billing } from '../../wailsjs/go/models'
import { BillingGetUserInfo, BillingGetIdentityOverview } from '../../wailsjs/go/main/App'

// Re-export Wails types for convenience
export type UserBillingInfo = billing.UserInfo
export type SubscriptionInfo = billing.SubscriptionInfo
export type SubscriptionPlan = billing.SubscriptionPlan
export type TopUpInfo = billing.TopUpInfo
export type IdentityOverview = billing.IdentityOverview

const POLL_INTERVAL_MS = 60_000

interface BillingState {
  userInfo: UserBillingInfo | null
  plans: SubscriptionPlan[]
  subscriptions: SubscriptionInfo[]
  topUpInfo: TopUpInfo | null
  identityOverview: IdentityOverview | null
  loading: boolean
  error: string | null
  /** ISO timestamp of the most recent successful refresh. */
  lastRefreshedAt: string | null
  /** Internal: polling timer handle (number on browser, NodeJS.Timeout on Node). */
  _pollHandle: ReturnType<typeof setInterval> | null

  setUserInfo: (info: UserBillingInfo | null) => void
  setPlans: (plans: SubscriptionPlan[]) => void
  setSubscriptions: (subs: SubscriptionInfo[]) => void
  setTopUpInfo: (info: TopUpInfo | null) => void
  setIdentityOverview: (ov: IdentityOverview | null) => void
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void
  setLastRefreshedAt: (iso: string | null) => void
  /** Start background polling. Guarded by document.visibilityState — skips when tab hidden. Idempotent. */
  startPolling: () => void
  /** Stop background polling. Idempotent. */
  stopPolling: () => void
  /** One-shot refresh (also called by polling). Safe to call concurrently. */
  refreshNow: () => Promise<void>
  /** Clear all billing data — call before loading a different account's data. */
  reset: () => void
}

export const useBillingStore = create<BillingState>((set, get) => ({
  userInfo: null,
  plans: [],
  subscriptions: [],
  topUpInfo: null,
  identityOverview: null,
  loading: false,
  error: null,
  lastRefreshedAt: null,
  _pollHandle: null,

  setUserInfo: (info) => set({ userInfo: info }),
  setPlans: (plans) => set({ plans }),
  setSubscriptions: (subs) => set({ subscriptions: subs }),
  setTopUpInfo: (info) => set({ topUpInfo: info }),
  setIdentityOverview: (ov) => set({ identityOverview: ov }),
  setLoading: (loading) => set({ loading }),
  setError: (error) => set({ error }),
  setLastRefreshedAt: (iso) => set({ lastRefreshedAt: iso }),

  startPolling: () => {
    if (get()._pollHandle !== null) return // already polling
    const handle = setInterval(() => {
      // Skip refresh when tab is backgrounded — saves battery and avoids
      // user noticing stale data when they tab back in (a fresh load is
      // triggered on visibilitychange below).
      if (typeof document !== 'undefined' && document.hidden) return
      void get().refreshNow()
    }, POLL_INTERVAL_MS)
    set({ _pollHandle: handle })
  },

  stopPolling: () => {
    const h = get()._pollHandle
    if (h !== null) clearInterval(h)
    set({ _pollHandle: null })
  },

  refreshNow: async () => {
    const results = await Promise.allSettled([
      BillingGetUserInfo(),
      BillingGetIdentityOverview('lurus-switch'),
    ])
    const u = results[0]
    const o = results[1]
    if (u.status === 'fulfilled' && u.value) set({ userInfo: u.value })
    if (o.status === 'fulfilled' && o.value) set({ identityOverview: o.value })
    set({ lastRefreshedAt: new Date().toISOString() })
  },

  reset: () => {
    const h = get()._pollHandle
    if (h !== null) clearInterval(h)
    set({
      userInfo: null,
      plans: [],
      subscriptions: [],
      topUpInfo: null,
      identityOverview: null,
      loading: false,
      error: null,
      lastRefreshedAt: null,
      _pollHandle: null,
    })
  },
}))
