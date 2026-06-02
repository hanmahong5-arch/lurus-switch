import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'

// i18n stub — returns fallback string so assertions can match user-facing copy
// without depending on locale resolution.
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

// Wails models stub — admin namespace shapes.
vi.mock('../../wailsjs/go/models', () => ({
  admin: {},
}))

// dashboardSource — module-level mock so the page's useMemo never calls real
// Wails bindings. mockFetch is swapped per test.
const mockFetch = vi.fn()

vi.mock('../lib/dashboardSource', () => ({
  makeDashboardSource: vi.fn(() => ({ kind: 'local', fetch: mockFetch })),
}))

// walletSource — used by ResellerWalletStrip; default to rejecting so the
// strip silently hides. Tests that want it visible override this.
const mockGetInfo = vi.fn()

vi.mock('../lib/walletSource', () => ({
  makeWalletSource: vi.fn(() => ({
    kind: 'hub',
    getInfo: mockGetInfo,
    listTransactions: vi.fn(),
  })),
}))

// gatewayStore — the page calls useGatewayStore() with NO selector (full
// state destructure), so the mock must return the whole state object directly.
const mockUseGatewayStore = vi.fn()
vi.mock('../stores/gatewayStore', () => ({
  useGatewayStore: (selector?: (s: { status: unknown; adminToken: string | null }) => unknown) => {
    const state = mockUseGatewayStore()
    return selector ? selector(state) : state
  },
}))

// configStore — selector-based mock similar to sibling tests.
const mockUseConfigStore = vi.fn()
vi.mock('../stores/configStore', () => ({
  useConfigStore: <T,>(selector: (s: { appMode: string }) => T) =>
    selector({ appMode: mockUseConfigStore() }),
}))

import { GatewayDashboardPage } from './GatewayDashboardPage'

// ---------------------------------------------------------------------------
// Shared fixture data
// ---------------------------------------------------------------------------

const SUMMARY = {
  user_count: 12,
  channel_count: 5,
  token_count: 30,
  today_request: 200,
  today_quota: 1000,
  today_tokens: 50000,
}

const QUOTA_DATES = [
  { date: '2026-05-28', quota: 400, request_count: 80, token_count: 12000, model_usage: {} },
  { date: '2026-05-29', quota: 600, request_count: 120, token_count: 18000, model_usage: {} },
]

const PERF_STATS = {
  goroutines: 42,
  memory_alloc: 10 * 1024 * 1024, // 10 MB
  uptime: 7200,                    // 2 h
  requests_total: 5000,
  requests_per_sec: 1.23,
}

beforeEach(() => {
  vi.clearAllMocks()

  // Default: personal mode with a running local server + admin token.
  mockUseConfigStore.mockReturnValue('personal')
  mockUseGatewayStore.mockReturnValue({
    status: { running: true, url: 'http://localhost:3000', port: 3000, uptime: 7200, version: '0.5.0', binaryOk: true },
    adminToken: 'test-admin-token',
  })

  // Default dashboard fetch returns a valid bundle.
  mockFetch.mockResolvedValue({
    summary: SUMMARY,
    quota: QUOTA_DATES,
    performance: PERF_STATS,
  })

  // Default wallet: reject so the strip hides (its design intent on failure).
  mockGetInfo.mockRejectedValue(new Error('no admin token'))
})

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('GatewayDashboardPage', () => {
  // --- no-source guard ---

  it('shows stopped banner when server is not running (personal mode, no adminToken)', () => {
    mockUseGatewayStore.mockReturnValue({ status: null, adminToken: null })
    render(<GatewayDashboardPage />)
    // The page renders a "gateway.status.stopped" message (t-key fallback).
    expect(screen.getByText('gateway.status.stopped')).toBeInTheDocument()
  })

  it('shows stopped banner when server is running=false and no adminToken (personal mode)', () => {
    mockUseConfigStore.mockReturnValue('personal')
    mockUseGatewayStore.mockReturnValue({
      status: { running: false, url: '', port: 0, uptime: 0, version: '', binaryOk: false },
      adminToken: null,
    })
    render(<GatewayDashboardPage />)
    // The page renders the "gateway.status.stopped" i18n key as fallback text.
    expect(screen.getByText('gateway.status.stopped')).toBeInTheDocument()
  })

  // --- successful data load ---

  it('renders the Dashboard heading after successful load', async () => {
    render(<GatewayDashboardPage />)
    await waitFor(() => {
      expect(screen.getByText('Dashboard')).toBeInTheDocument()
    })
  })

  it('renders KPI cards with correct values after load', async () => {
    render(<GatewayDashboardPage />)
    // KpiCard renders value via toLocaleString(); for small ints that is the
    // digit string itself.
    await waitFor(() => {
      expect(screen.getByText('12')).toBeInTheDocument()   // user_count
      expect(screen.getByText('5')).toBeInTheDocument()    // channel_count
      expect(screen.getByText('30')).toBeInTheDocument()   // token_count
      expect(screen.getByText('200')).toBeInTheDocument()  // today_request
    })
  })

  it('renders KPI card labels', async () => {
    render(<GatewayDashboardPage />)
    // KpiCard wraps the label in "[ LABEL ]" uppercase, so we match with a
    // partial text regex rather than exact string.
    await waitFor(() => {
      expect(screen.getByText(/USERS/i)).toBeInTheDocument()
      expect(screen.getByText(/CHANNELS/i)).toBeInTheDocument()
      expect(screen.getByText(/TOKENS/i)).toBeInTheDocument()
      expect(screen.getByText(/TODAY REQUESTS/i)).toBeInTheDocument()
    })
  })

  it('renders quota trend chart section header', async () => {
    render(<GatewayDashboardPage />)
    await waitFor(() => {
      expect(screen.getByText(/QUOTA TREND/i)).toBeInTheDocument()
    })
  })

  it('renders performance panel when stats are present', async () => {
    render(<GatewayDashboardPage />)
    await waitFor(() => {
      expect(screen.getByText(/PERFORMANCE/i)).toBeInTheDocument()
      // Goroutines row
      expect(screen.getByText('42')).toBeInTheDocument()
      // Memory: 10 MB
      expect(screen.getByText('10.0 MB')).toBeInTheDocument()
      // Uptime: 2h
      expect(screen.getByText('2.0 h')).toBeInTheDocument()
      // Requests total
      expect(screen.getByText('5000')).toBeInTheDocument()
      // Req/s
      expect(screen.getByText('1.23')).toBeInTheDocument()
    })
  })

  // --- error path ---

  it('renders error banner when fetch rejects', async () => {
    mockFetch.mockRejectedValue(new Error('connection refused'))
    render(<GatewayDashboardPage />)
    await waitFor(() => {
      expect(screen.getByText(/connection refused/)).toBeInTheDocument()
    })
  })

  // --- refresh action ---

  it('re-fetches data when the refresh button is clicked', async () => {
    render(<GatewayDashboardPage />)
    // Wait for initial load to complete.
    await waitFor(() => {
      expect(screen.getByText('Dashboard')).toBeInTheDocument()
    })

    // At this point mockFetch has been called once (initial load).
    const callsAfterMount = mockFetch.mock.calls.length
    expect(callsAfterMount).toBeGreaterThanOrEqual(1)

    // Find the refresh button (Button with no text label, just an icon).
    // The Button component renders a <button>; disabled=false while idle.
    const buttons = screen.getAllByRole('button')
    // The only button on the page is the refresh button.
    const refreshButton = buttons[0]
    expect(refreshButton).not.toBeDisabled()

    fireEvent.click(refreshButton)

    await waitFor(() => {
      expect(mockFetch.mock.calls.length).toBeGreaterThan(callsAfterMount)
    })
  })

  // --- reseller mode ---

  it('renders in reseller mode (hub source path)', async () => {
    mockUseConfigStore.mockReturnValue('reseller')
    // In reseller mode, source is always non-null regardless of server status.
    mockUseGatewayStore.mockReturnValue({ status: null, adminToken: null })

    render(<GatewayDashboardPage />)
    await waitFor(() => {
      expect(screen.getByText('Dashboard')).toBeInTheDocument()
    })
    // KPI cards should show the mocked values.
    await waitFor(() => {
      expect(screen.getByText('12')).toBeInTheDocument()
    })
  })

  it('shows ResellerWalletStrip when reseller mode and wallet getInfo succeeds', async () => {
    mockUseConfigStore.mockReturnValue('reseller')
    mockUseGatewayStore.mockReturnValue({ status: null, adminToken: null })
    mockGetInfo.mockResolvedValue({
      balance: 500,
      available: 490,
      frozen: 10,
      lifetime_topup: 2000,
      lifetime_spend: 1510,
      source: 'platform',
    })

    render(<GatewayDashboardPage />)
    // The strip shows balance KPI card.
    await waitFor(() => {
      expect(screen.getByText('¥ 500.00')).toBeInTheDocument()
    })
  })

  it('hides ResellerWalletStrip silently when wallet getInfo fails', async () => {
    mockUseConfigStore.mockReturnValue('reseller')
    mockUseGatewayStore.mockReturnValue({ status: null, adminToken: null })
    mockGetInfo.mockRejectedValue(new Error('HTTP 403'))

    render(<GatewayDashboardPage />)
    await waitFor(() => {
      expect(screen.getByText('Dashboard')).toBeInTheDocument()
    })
    // The strip renders nothing on error — balance text must NOT appear.
    expect(screen.queryByText(/¥/)).not.toBeInTheDocument()
  })

  // --- empty data path ---

  it('shows "No data" in chart area when quota array is empty', async () => {
    mockFetch.mockResolvedValue({
      summary: SUMMARY,
      quota: [],
      performance: null,
    })
    render(<GatewayDashboardPage />)
    await waitFor(() => {
      expect(screen.getByText('No data')).toBeInTheDocument()
    })
  })

  it('hides performance panel when performance stats are null', async () => {
    mockFetch.mockResolvedValue({
      summary: SUMMARY,
      quota: QUOTA_DATES,
      performance: null,
    })
    render(<GatewayDashboardPage />)
    await waitFor(() => {
      expect(screen.getByText('Dashboard')).toBeInTheDocument()
    })
    // PERFORMANCE header must not appear.
    expect(screen.queryByText(/PERFORMANCE/i)).not.toBeInTheDocument()
  })
})
