import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

const billingMock = vi.hoisted(() => ({
  BillingGetUserInfo: vi.fn(),
  BillingGetIdentityOverview: vi.fn(),
}))
vi.mock('../../wailsjs/go/main/App', () => billingMock)

import { useBillingStore } from './billingStore'

describe('billingStore polling', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    useBillingStore.getState().reset()
    billingMock.BillingGetUserInfo.mockReset()
    billingMock.BillingGetUserInfo.mockResolvedValue({
      quota: 5_000_000, used_quota: 1_000_000, daily_used: 100_000,
      daily_quota: 0, remaining_quota: 0, group: '', username: 'u', display_name: '',
      aff_code: '', role: 0, subscription: undefined,
    } as any)
    billingMock.BillingGetIdentityOverview.mockReset()
    billingMock.BillingGetIdentityOverview.mockResolvedValue({
      account: { id: 1, lurus_id: 'l-1', display_name: 'Z', avatar_url: '' },
      vip: { level: 1, level_name: 'Pro', level_en: 'Pro', points: 0 },
      wallet: { balance: 100, frozen: 5 },
      subscription: null,
      topup_url: 'https://topup',
    } as any)
  })

  afterEach(() => {
    useBillingStore.getState().stopPolling()
    vi.useRealTimers()
  })

  it('refreshNow populates userInfo + identityOverview + lastRefreshedAt', async () => {
    await useBillingStore.getState().refreshNow()
    const s = useBillingStore.getState()
    expect(s.userInfo?.quota).toBe(5_000_000)
    expect((s.identityOverview as any)?.wallet?.balance).toBe(100)
    expect(s.lastRefreshedAt).toBeTruthy()
  })

  it('startPolling is idempotent — second call does not double-schedule', () => {
    useBillingStore.getState().startPolling()
    const first = (useBillingStore.getState() as any)._pollHandle
    useBillingStore.getState().startPolling()
    const second = (useBillingStore.getState() as any)._pollHandle
    expect(first).toBe(second)
  })

  it('stopPolling clears the interval handle', () => {
    useBillingStore.getState().startPolling()
    expect((useBillingStore.getState() as any)._pollHandle).not.toBeNull()
    useBillingStore.getState().stopPolling()
    expect((useBillingStore.getState() as any)._pollHandle).toBeNull()
  })

  it('polling skips refresh when document.hidden is true', () => {
    Object.defineProperty(document, 'hidden', { configurable: true, value: true })
    useBillingStore.getState().startPolling()
    vi.advanceTimersByTime(60_000)
    // Visibility-guarded skip → no binding calls.
    expect(billingMock.BillingGetUserInfo).not.toHaveBeenCalled()
    Object.defineProperty(document, 'hidden', { configurable: true, value: false })
  })

  it('polling fires refresh when document.hidden is false', () => {
    Object.defineProperty(document, 'hidden', { configurable: true, value: false })
    useBillingStore.getState().startPolling()
    vi.advanceTimersByTime(60_000)
    expect(billingMock.BillingGetUserInfo).toHaveBeenCalledTimes(1)
  })

  it('reset clears polling handle alongside data', () => {
    useBillingStore.getState().startPolling()
    useBillingStore.getState().reset()
    expect((useBillingStore.getState() as any)._pollHandle).toBeNull()
    expect(useBillingStore.getState().userInfo).toBeNull()
    expect(useBillingStore.getState().lastRefreshedAt).toBeNull()
  })
})
