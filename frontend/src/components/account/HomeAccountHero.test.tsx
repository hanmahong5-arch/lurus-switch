import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'

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
  initReactI18next: { type: '3rdParty', init: () => {} },
}))

vi.mock('../../../wailsjs/go/main/App', () => ({
  BillingGetUserInfo: vi.fn().mockResolvedValue(null),
  BillingGetIdentityOverview: vi.fn().mockResolvedValue(null),
  BillingOpenTopup: vi.fn().mockResolvedValue(undefined),
}))

import { HomeAccountHero, estimateMonthCost } from './HomeAccountHero'
import { useAuthStore } from '../../stores/authStore'
import { useBillingStore } from '../../stores/billingStore'
import { useConfigStore } from '../../stores/configStore'

beforeEach(() => {
  useAuthStore.setState({
    authState: { is_logged_in: true, user: { sub: 's', name: '张三', email: 'z@x.com', picture: '' } },
    isLoggingIn: false,
    loginError: null,
  })
  useBillingStore.getState().reset()
})

describe('estimateMonthCost', () => {
  it('returns undefined when dailyUsed missing', () => {
    expect(estimateMonthCost(undefined)).toBeUndefined()
  })

  it('day 1 of month: ≈ 1× daily', () => {
    const d = new Date(2026, 4, 1) // May 1
    // 500k quota = $1 = ¥7.2, so 500_000 dailyUsed × 1 day = ¥7.2
    expect(estimateMonthCost(500_000, d)).toBeCloseTo(7.2, 1)
  })

  it('day 30 of month: ≈ 30× daily', () => {
    const d = new Date(2026, 4, 30)
    expect(estimateMonthCost(500_000, d)).toBeCloseTo(216, 0)
  })

  it('handles zero dailyUsed', () => {
    expect(estimateMonthCost(0)).toBe(0)
  })
})

describe('HomeAccountHero', () => {
  it('renders welcome with display name when logged in (Personal)', () => {
    useConfigStore.setState({ appMode: 'personal' } as any)
    render(<HomeAccountHero />)
    expect(screen.getByRole('region')).toBeInTheDocument()
    // Welcome line uses display name from auth user.
    expect(screen.getByText(/张三/)).toBeInTheDocument()
  })

  it('returns null in EndUser mode', () => {
    useConfigStore.setState({ appMode: 'enduser' } as any)
    const { container } = render(<HomeAccountHero />)
    expect(container.firstChild).toBeNull()
  })

  it('returns null when not logged in (Personal)', () => {
    useAuthStore.setState({ authState: { is_logged_in: false }, isLoggingIn: false, loginError: null })
    useConfigStore.setState({ appMode: 'personal' } as any)
    const { container } = render(<HomeAccountHero />)
    expect(container.firstChild).toBeNull()
  })

  it('renders in Reseller mode even without OIDC login', () => {
    useAuthStore.setState({ authState: { is_logged_in: false }, isLoggingIn: false, loginError: null })
    useConfigStore.setState({ appMode: 'reseller' } as any)
    render(<HomeAccountHero />)
    expect(screen.getByRole('region')).toBeInTheDocument()
  })
})
