import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

// i18n stub: returns the fallback string when provided so the assertions
// can match user-facing copy without depending on locale resolution.
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

const mockGetInfo = vi.fn()
const mockListTransactions = vi.fn()

vi.mock('../../wailsjs/go/main/App', () => ({
  HubGetWalletInfo: (...a: unknown[]) => mockGetInfo(...a),
  HubListWalletTransactions: (...a: unknown[]) => mockListTransactions(...a),
}))

vi.mock('../../wailsjs/go/models', () => ({
  admin: {},
}))

// Mock the configStore so the page sees appMode = 'reseller'.
const mockUseConfigStore = vi.fn()
vi.mock('../stores/configStore', () => ({
  useConfigStore: <T,>(selector: (s: { appMode: string }) => T) =>
    selector({ appMode: mockUseConfigStore() }),
}))

import { GatewayWalletPage } from './GatewayWalletPage'

beforeEach(() => {
  vi.clearAllMocks()
  mockUseConfigStore.mockReturnValue('reseller')
})

describe('GatewayWalletPage', () => {
  it('shows resellerOnly notice in non-Reseller mode', () => {
    mockUseConfigStore.mockReturnValue('personal')
    render(<GatewayWalletPage />)
    expect(screen.getByText('钱包功能仅在 Reseller 模式可用')).toBeInTheDocument()
  })

  it('renders balance KPI and transactions when platform-backed', async () => {
    mockGetInfo.mockResolvedValue({
      source: 'platform',
      balance: 250,
      frozen: 5,
      available: 245,
      lifetime_topup: 1000,
      lifetime_spend: 760,
      topup_url: 'https://identity.lurus.cn/wallet/topup',
    })
    mockListTransactions.mockResolvedValue({
      items: [
        {
          id: 1,
          account_id: 7,
          type: 'topup',
          amount: 100,
          balance_after: 250,
          product_id: '',
          reference_type: '',
          reference_id: '',
          description: '充值',
          created_at: '2026-05-26T01:00:00Z',
        },
        {
          id: 2,
          account_id: 7,
          type: 'debit',
          amount: 12.5,
          balance_after: 237.5,
          product_id: 'lurus-api',
          reference_type: '',
          reference_id: '',
          description: 'API 消费',
          created_at: '2026-05-26T02:00:00Z',
        },
      ],
      total: 12,
      page: 1,
      page_size: 20,
    })

    render(<GatewayWalletPage />)

    expect(await screen.findByText('¥ 250.00')).toBeInTheDocument()
    expect(screen.getByText('¥ 245.00')).toBeInTheDocument()
    expect(screen.getByText('topup')).toBeInTheDocument()
    expect(screen.getByText('debit')).toBeInTheDocument()
    expect(screen.getByText('+100.00')).toBeInTheDocument()
    expect(screen.getByText('-12.50')).toBeInTheDocument()

    // Withdraw button is enabled when info is platform-backed.
    expect(screen.getByText('申请提现').closest('button')).not.toBeDisabled()
  })

  it('renders "not linked" banner when source is internal', async () => {
    mockGetInfo.mockResolvedValue({
      source: 'internal',
      balance: 50,
      available: 50,
      frozen: 0,
      lifetime_topup: 0,
      lifetime_spend: 0,
    })
    mockListTransactions.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20 })

    render(<GatewayWalletPage />)

    expect(await screen.findByText('当前账号未绑定 lurus-platform 钱包')).toBeInTheDocument()
    // Withdraw button is disabled when not platform-backed.
    expect(screen.getByText('申请提现').closest('button')).toBeDisabled()
  })

  it('renders error banner on binding failure', async () => {
    mockGetInfo.mockRejectedValue(new Error('hub admin: HTTP 503'))
    mockListTransactions.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20 })

    render(<GatewayWalletPage />)

    await waitFor(() => {
      expect(screen.getByText(/HTTP 503/)).toBeInTheDocument()
    })
  })
})
