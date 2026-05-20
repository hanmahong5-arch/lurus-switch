import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (_key: string, fallbackOrOpts?: string | Record<string, unknown>, opts?: Record<string, unknown>) => {
      const fallback = typeof fallbackOrOpts === 'string' ? fallbackOrOpts : _key
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
  BillingOpenTopup: vi.fn(),
  GetEndUserStatus: vi.fn().mockResolvedValue({
    state: 'activated', activated: true, hubUrl: 'https://hub.x', tenantSlug: 'acme',
    quota: 5_000_000, expiresAt: '2026-12-31T00:00:00Z',
  }),
  GetAppSettings: vi.fn().mockResolvedValue({ brandName: 'AcmeAI', reseller: { hubUrl: 'https://hub.x', tenantSlug: 'acme' } }),
}))

vi.mock('../../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn().mockReturnValue(() => {}),
  EventsOff: vi.fn(),
}))

import { AccountSummaryCard } from './AccountSummaryCard'
import { useAuthStore } from '../../stores/authStore'
import { useBillingStore } from '../../stores/billingStore'
import { useConfigStore } from '../../stores/configStore'

beforeEach(() => {
  useBillingStore.getState().reset()
  useAuthStore.setState({
    authState: { is_logged_in: false },
    isLoggingIn: false,
    loginError: null,
  })
})

describe('AccountSummaryCard', () => {
  it('shows sign-in CTA when not logged in (Personal)', () => {
    useConfigStore.setState({ appMode: 'personal' } as any)
    render(<AccountSummaryCard />)
    expect(screen.getByText('登录账户')).toBeInTheDocument()
  })

  it('renders user name when logged in (Personal)', () => {
    useAuthStore.setState({
      authState: { is_logged_in: true, user: { sub: 's', name: '李四', email: 'l@x', picture: '' } },
      isLoggingIn: false, loginError: null,
    })
    useConfigStore.setState({ appMode: 'personal' } as any)
    render(<AccountSummaryCard />)
    expect(screen.getByText('李四')).toBeInTheDocument()
  })

  it('renders reseller hint without wallet (Reseller)', () => {
    useConfigStore.setState({ appMode: 'reseller' } as any)
    render(<AccountSummaryCard />)
    expect(screen.getByText('经销商控制台')).toBeInTheDocument()
  })

  it('renders enduser hint (EndUser)', async () => {
    useConfigStore.setState({ appMode: 'enduser' } as any)
    render(<AccountSummaryCard />)
    expect(screen.getByText('激活码客户端')).toBeInTheDocument()
  })

  it('toggles popover on click', () => {
    useAuthStore.setState({
      authState: { is_logged_in: true, user: { sub: 's', name: 'X', email: '', picture: '' } },
      isLoggingIn: false, loginError: null,
    })
    useConfigStore.setState({ appMode: 'personal' } as any)
    render(<AccountSummaryCard />)
    const trigger = screen.getByRole('button', { expanded: false })
    fireEvent.click(trigger)
    // After open, the same trigger reports expanded=true.
    expect(screen.getByRole('button', { expanded: true })).toBe(trigger)
  })

  it('shows usage bar when quota / used_quota are set', () => {
    useAuthStore.setState({
      authState: { is_logged_in: true, user: { sub: 's', name: 'X', email: '', picture: '' } },
      isLoggingIn: false, loginError: null,
    })
    useConfigStore.setState({ appMode: 'personal' } as any)
    useBillingStore.setState({
      userInfo: { quota: 1000, used_quota: 800 } as any,
      identityOverview: null,
    })
    render(<AccountSummaryCard />)
    // 80% — falls into the warn band.
    expect(screen.getByText('80%')).toBeInTheDocument()
  })
})
