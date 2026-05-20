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

const getStatusMock = vi.fn()
vi.mock('../../wailsjs/go/main/App', () => ({
  ClearActivation: vi.fn().mockResolvedValue(undefined),
  GetEndUserStatus: () => getStatusMock(),
  HeartbeatNow: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('../stores/dashboardStore', () => ({
  useDashboardStore: (selector: (s: { tools: Record<string, unknown> }) => unknown) =>
    selector({ tools: {} }),
}))

vi.mock('../stores/configStore', () => ({
  useConfigStore: (selector: (s: { setActiveTool: (id: string) => void }) => unknown) =>
    selector({ setActiveTool: vi.fn() }),
}))

import { EndUserMainPage } from './EndUserMainPage'

const baseStatus = (overrides: Record<string, unknown> = {}) => ({
  state: 'active',
  activated: true,
  hubUrl: 'https://hub.example',
  tenantSlug: 'acme',
  userId: 42,
  quota: 250_000,
  expiresAt: '2030-01-01T00:00:00Z',
  activatedAt: '2025-01-01T00:00:00Z',
  lastHeartbeat: new Date().toISOString(),
  ...overrides,
})

beforeEach(() => {
  getStatusMock.mockReset()
})

const daysFromNow = (n: number) => {
  const d = new Date()
  d.setDate(d.getDate() + n)
  return d.toISOString()
}

describe('EndUserMainPage expiry banner', () => {
  it('shows no expiry banner when expiry > 30 days', async () => {
    getStatusMock.mockResolvedValue(baseStatus({ expiresAt: daysFromNow(60) }))
    render(<EndUserMainPage onDeactivated={vi.fn()} />)
    await screen.findByText(/我的服务/)
    expect(screen.queryByTestId('expiry-banner-warning')).not.toBeInTheDocument()
    expect(screen.queryByTestId('expiry-banner-critical')).not.toBeInTheDocument()
  })

  it('shows warning banner when 7 < daysLeft ≤ 30', async () => {
    getStatusMock.mockResolvedValue(baseStatus({ expiresAt: daysFromNow(15) }))
    render(<EndUserMainPage onDeactivated={vi.fn()} />)
    const banner = await screen.findByTestId('expiry-banner-warning')
    expect(banner).toBeInTheDocument()
    expect(banner.textContent).toMatch(/15/)
  })

  it('shows critical banner when daysLeft ≤ 7', async () => {
    getStatusMock.mockResolvedValue(baseStatus({ expiresAt: daysFromNow(3) }))
    render(<EndUserMainPage onDeactivated={vi.fn()} />)
    const banner = await screen.findByTestId('expiry-banner-critical')
    expect(banner).toBeInTheDocument()
    expect(banner.textContent).toMatch(/3/)
    expect(screen.queryByTestId('expiry-banner-warning')).not.toBeInTheDocument()
  })

  it('shows "expired" copy when daysLeft ≤ 0', async () => {
    getStatusMock.mockResolvedValue(baseStatus({ expiresAt: daysFromNow(-2) }))
    render(<EndUserMainPage onDeactivated={vi.fn()} />)
    const banner = await screen.findByTestId('expiry-banner-critical')
    expect(banner.textContent).toMatch(/激活已到期/)
  })

  it('suppresses banner when activation is revoked', async () => {
    getStatusMock.mockResolvedValue(baseStatus({ state: 'revoked', expiresAt: daysFromNow(3) }))
    render(<EndUserMainPage onDeactivated={vi.fn()} />)
    await screen.findByText(/我的服务/)
    expect(screen.queryByTestId('expiry-banner-critical')).not.toBeInTheDocument()
  })

  it('suppresses banner when no expiresAt set', async () => {
    getStatusMock.mockResolvedValue(baseStatus({ expiresAt: '' }))
    render(<EndUserMainPage onDeactivated={vi.fn()} />)
    await screen.findByText(/我的服务/)
    expect(screen.queryByTestId('expiry-banner-warning')).not.toBeInTheDocument()
    expect(screen.queryByTestId('expiry-banner-critical')).not.toBeInTheDocument()
  })
})
