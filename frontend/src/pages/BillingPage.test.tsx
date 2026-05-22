import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

// i18n stub — same shape as EndUserActivationPage.test.tsx. Returns the
// fallback when provided, otherwise the key. BillingPage uses static
// English strings for most labels so the stub is rarely hit.
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallbackOrOpts?: string | Record<string, unknown>, opts?: Record<string, unknown>) => {
      const fallback = typeof fallbackOrOpts === 'string' ? fallbackOrOpts : key
      const vars = typeof fallbackOrOpts === 'object' ? fallbackOrOpts : opts
      if (vars && typeof vars === 'object') {
        return Object.entries(vars).reduce<string>(
          (s, [k, v]) => s.replace(new RegExp(`{{\\s*${k}\\s*}}`, 'g'), String(v)),
          fallback,
        )
      }
      return fallback
    },
    i18n: { language: 'zh', changeLanguage: vi.fn() },
  }),
  Trans: ({ children, i18nKey }: { children?: React.ReactNode; i18nKey?: string }) =>
    children ?? i18nKey ?? null,
}))

// Default proxy settings — empty userToken means "disconnected" branch.
const mockGetProxySettings = vi.fn().mockResolvedValue({
  apiEndpoint: '',
  apiKey: '',
  registrationUrl: '',
  tenantSlug: '',
  userToken: '',
})
const mockSaveProxySettings = vi.fn().mockResolvedValue(undefined)
const mockBillingGetUserInfo = vi.fn().mockResolvedValue({
  username: 'tester',
  display_name: 'Test User',
  quota: 1000,
  used_quota: 100,
  daily_quota: 100,
  daily_used: 10,
  aff_code: 'AFF-XYZ',
})
const mockBillingGetPlans = vi.fn().mockResolvedValue([])
const mockBillingGetTopUpInfo = vi.fn().mockResolvedValue(null)
const mockBillingGetSubscriptions = vi.fn().mockResolvedValue([])
const mockBillingGetIdentityOverview = vi.fn().mockResolvedValue(null)

vi.mock('../../wailsjs/go/main/App', () => ({
  BillingGetUserInfo: (...a: unknown[]) => mockBillingGetUserInfo(...a),
  BillingGetPlans: (...a: unknown[]) => mockBillingGetPlans(...a),
  BillingGetTopUpInfo: (...a: unknown[]) => mockBillingGetTopUpInfo(...a),
  BillingGetSubscriptions: (...a: unknown[]) => mockBillingGetSubscriptions(...a),
  BillingCreateTopUp: vi.fn().mockResolvedValue({}),
  BillingSubscribe: vi.fn().mockResolvedValue({}),
  BillingRedeemCode: vi.fn().mockResolvedValue(0),
  BillingOpenPaymentURL: vi.fn().mockResolvedValue(undefined),
  BillingGetIdentityOverview: (...a: unknown[]) => mockBillingGetIdentityOverview(...a),
  BillingOpenTopup: vi.fn().mockResolvedValue(undefined),
  GetProxySettings: (...a: unknown[]) => mockGetProxySettings(...a),
  SaveProxySettings: (...a: unknown[]) => mockSaveProxySettings(...a),
  // UsageBreakdown (Wave1 by-model/by-tool donuts) is rendered by BillingPage
  // and calls these bindings; stub them so the page renders in the harness.
  GetModelSummaries: vi.fn().mockResolvedValue([]),
  GetAppSummaries: vi.fn().mockResolvedValue([]),
}))

// `proxy.ProxySettings.createFrom` needs to exist because BillingPage uses
// it to coerce the settings object. Provide a passthrough so the test
// doesn't blow up on the Wails model factory.
vi.mock('../../wailsjs/go/models', () => ({
  proxy: { ProxySettings: { createFrom: (x: unknown) => x } },
  billing: {},
}))

import { BillingPage } from './BillingPage'
import { useBillingStore } from '../stores/billingStore'
import { useDashboardStore } from '../stores/dashboardStore'

describe('BillingPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Reset zustand stores between tests so state from a connected test
    // doesn't leak into the next.
    useBillingStore.getState().reset()
    useDashboardStore.setState({
      proxySettings: { apiEndpoint: '', apiKey: '', registrationUrl: '', tenantSlug: '', userToken: '' },
    } as any)
    mockGetProxySettings.mockResolvedValue({
      apiEndpoint: '', apiKey: '', registrationUrl: '', tenantSlug: '', userToken: '',
    })
  })

  it('renders without crashing — smoke', async () => {
    render(<BillingPage />)
    await waitFor(() => {
      expect(screen.getByText('Billing')).toBeDefined()
    })
  })

  it('shows the page header subtitle', async () => {
    render(<BillingPage />)
    await waitFor(() => {
      expect(screen.getByText('Manage your quota, subscription, and payments')).toBeDefined()
    })
  })

  it('shows the "Connect to Billing" panel when not connected', async () => {
    render(<BillingPage />)
    await waitFor(() => {
      expect(screen.getByText('Connect to Billing')).toBeDefined()
      expect(screen.getByPlaceholderText('Paste your token here')).toBeDefined()
      expect(screen.getByText('Connect')).toBeDefined()
    })
  })

  it('Connect button is disabled until a token is typed', async () => {
    render(<BillingPage />)
    await waitFor(() => screen.getByText('Connect'))
    const button = screen.getByText('Connect').closest('button') as HTMLButtonElement
    expect(button.disabled).toBe(true)
    const input = screen.getByPlaceholderText('Paste your token here') as HTMLInputElement
    fireEvent.change(input, { target: { value: 'mytoken' } })
    expect(button.disabled).toBe(false)
  })

  it('does not load billing data when userToken is empty', async () => {
    render(<BillingPage />)
    await waitFor(() => screen.getByText('Connect to Billing'))
    // BillingGetUserInfo should only fire after a token is present.
    expect(mockBillingGetUserInfo).not.toHaveBeenCalled()
  })

  it('shows the Redeem panel when connected', async () => {
    // Pre-load proxy with a token so the connected branch renders.
    mockGetProxySettings.mockResolvedValueOnce({
      apiEndpoint: 'https://hub.example', apiKey: '', registrationUrl: '',
      tenantSlug: '', userToken: 'tok-abc',
    })
    render(<BillingPage />)
    await waitFor(() => {
      // RedeemPanel renders its static "Redeem Code" heading + button.
      expect(screen.getByText('Redeem Code')).toBeDefined()
      expect(screen.getByPlaceholderText('Enter redeem code')).toBeDefined()
    })
  })

  it('shows quota cards (balance section) when connected and userInfo loads', async () => {
    mockGetProxySettings.mockResolvedValueOnce({
      apiEndpoint: 'https://hub.example', apiKey: '', registrationUrl: '',
      tenantSlug: '', userToken: 'tok-abc',
    })
    render(<BillingPage />)
    await waitFor(() => {
      // QuotaCard label props "Total Quota" and "Daily Quota" — these are
      // the balance section labels.
      expect(screen.getByText('Total Quota')).toBeDefined()
      expect(screen.getByText('Daily Quota')).toBeDefined()
    })
  })

  it('displays the connected username after billing data loads', async () => {
    mockGetProxySettings.mockResolvedValueOnce({
      apiEndpoint: 'https://hub.example', apiKey: '', registrationUrl: '',
      tenantSlug: '', userToken: 'tok-abc',
    })
    render(<BillingPage />)
    await waitFor(() => {
      expect(screen.getByText('Test User')).toBeDefined()
      expect(screen.getByText('Connected as')).toBeDefined()
    })
  })
})
