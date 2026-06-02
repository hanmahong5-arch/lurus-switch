import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import type { LogSource, GatewayLog } from '../lib/logSource'

// i18n stub — returns fallback string when provided so assertions match
// user-facing copy without depending on locale resolution.
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

// Wails bindings are not available in jsdom — stub them.
vi.mock('../../wailsjs/go/main/App', () => ({
  HubListLogs: vi.fn(),
}))

vi.mock('../../wailsjs/go/models', () => ({
  admin: {},
}))

// Gateway sub-components — keep them light stubs so tests don't depend on
// their internal rendering details.
vi.mock('../components/gateway/SearchBar', () => ({
  SearchBar: ({ value, onChange, onSearch, placeholder }: {
    value: string; onChange: (v: string) => void; onSearch: () => void; placeholder?: string
  }) => (
    <input
      data-testid="search-bar"
      value={value}
      placeholder={placeholder}
      onChange={(e) => onChange(e.target.value)}
      onKeyDown={(e) => e.key === 'Enter' && onSearch()}
    />
  ),
}))

vi.mock('../components/gateway/Pagination', () => ({
  Pagination: ({ page, total, perPage, onPageChange }: {
    page: number; total: number; perPage: number; onPageChange: (p: number) => void
  }) => (
    <div data-testid="pagination" data-page={page} data-total={total} data-perpage={perPage}>
      <button onClick={() => onPageChange(page + 1)}>next</button>
    </div>
  ),
}))

vi.mock('../components/gateway/DateRangePicker', () => ({
  DateRangePicker: ({ start, end, onChange }: {
    start: string; end: string; onChange: (s: string, e: string) => void
  }) => (
    <div data-testid="date-range-picker" data-start={start} data-end={end}>
      <button onClick={() => onChange('2026-05-01', '2026-05-31')}>set-range</button>
    </div>
  ),
}))

vi.mock('../components/gateway/ConfirmModal', () => ({
  ConfirmModal: ({ open, title, onConfirm, onCancel }: {
    open: boolean; title: string; desc?: string; danger?: boolean
    onConfirm: () => void; onCancel: () => void
  }) =>
    open ? (
      <div data-testid="confirm-modal">
        <span>{title}</span>
        <button onClick={onConfirm}>confirm</button>
        <button onClick={onCancel}>cancel</button>
      </div>
    ) : null,
}))

vi.mock('../components/gateway/SimpleBarChart', () => ({
  SimpleBarChart: () => <div data-testid="simple-bar-chart" />,
}))

// ---- logSource mock ---------------------------------------------------------
// The page calls makeLogSource(...) inside useMemo; we intercept the factory
// and return a fake LogSource whose methods are vi.fn() instances that tests
// can configure per-scenario.

const mockList = vi.fn()
const mockStats = vi.fn()
const mockClearHistory = vi.fn()

let fakeSourceEnabled = true // set false to simulate source === null

vi.mock('../lib/logSource', () => ({
  makeLogSource: (_args: unknown) => {
    if (!fakeSourceEnabled) return null
    return {
      kind: 'local',
      capabilities: { stats: true, clearHistory: true },
      list: (...a: unknown[]) => mockList(...a),
      stats: (...a: unknown[]) => mockStats(...a),
      clearHistory: (...a: unknown[]) => mockClearHistory(...a),
    } satisfies LogSource
  },
}))

// ---- store mocks ------------------------------------------------------------

// The page calls useGatewayStore() without a selector and destructures the
// result directly — mock returns the whole state object.
const mockUseGatewayStore = vi.fn()
vi.mock('../stores/gatewayStore', () => ({
  useGatewayStore: () => mockUseGatewayStore(),
}))

const mockUseConfigStore = vi.fn()
vi.mock('../stores/configStore', () => ({
  useConfigStore: <T,>(selector: (s: { appMode: string }) => T) =>
    selector({ appMode: mockUseConfigStore() }),
}))

import { GatewayLogPage } from './GatewayLogPage'

// ---------------------------------------------------------------------------

const SAMPLE_LOGS: GatewayLog[] = [
  {
    id: 1,
    user_id: 42,
    created_at: 1717200000, // unix seconds
    type: 2,
    content: 'ok',
    username: 'alice',
    token_name: 'dev-key',
    model_name: 'claude-3-5-sonnet',
    quota: 100,
    prompt_tokens: 80,
    completion_tokens: 120,
    channel: 3,
    channel_name: 'Anthropic',
  },
  {
    id: 2,
    user_id: 43,
    created_at: 1717200060,
    type: 4,
    content: 'task',
    username: 'bob',
    token_name: '',
    model_name: '',
    quota: 0,
    prompt_tokens: 0,
    completion_tokens: 0,
    channel: 0,
    channel_name: '',
  },
]

beforeEach(() => {
  vi.clearAllMocks()
  fakeSourceEnabled = true

  mockUseConfigStore.mockReturnValue('personal')
  mockUseGatewayStore.mockReturnValue({
    status: { running: true, url: 'http://127.0.0.1:3000' },
    adminToken: 'test-admin-token',
  })

  // Default: successful empty list
  mockList.mockResolvedValue({ items: [], total: 0 })
  mockStats.mockResolvedValue({ quota: 500, rpm: 10, tpm: 200 })
  mockClearHistory.mockResolvedValue(undefined)
})

// ---------------------------------------------------------------------------

describe('GatewayLogPage', () => {
  // ── 1. No-source state ──────────────────────────────────────────────────

  it('shows stopped-gateway notice when source is null (personal, server stopped)', async () => {
    // Personal mode with server NOT running — makeLogSource returns null
    // because the page only calls makeLogSource when serverStatus.running+adminToken
    // BUT we control fakeSourceEnabled directly.
    fakeSourceEnabled = false
    mockUseConfigStore.mockReturnValue('personal')
    mockUseGatewayStore.mockReturnValue({ status: null, adminToken: null })

    render(<GatewayLogPage />)
    // When source is null the page shows the stopped message
    expect(screen.getByText('gateway.status.stopped')).toBeInTheDocument()
  })

  it('shows hubNotConfigured notice in reseller mode when source is null', async () => {
    fakeSourceEnabled = false
    mockUseConfigStore.mockReturnValue('reseller')
    mockUseGatewayStore.mockReturnValue({ status: null, adminToken: null })

    render(<GatewayLogPage />)
    expect(
      screen.getByText('请先在「设置」中配置 Reseller Hub URL 与管理员 Token'),
    ).toBeInTheDocument()
  })

  // ── 2. Initial render / empty-list state ────────────────────────────────

  it('renders the page header and tab bar', async () => {
    render(<GatewayLogPage />)

    await waitFor(() => expect(mockList).toHaveBeenCalled())

    // Header text key
    expect(screen.getByText('gateway.logs')).toBeInTheDocument()
    // Active tab transforms label to "[ LABEL.UPPERCASE ]"; inactive tabs show plain key.
    // We check the draw and task tabs (inactive) are present as plain text,
    // and the usage tab (active) is present in the uppercased bracket form.
    expect(screen.getByText(/GATEWAY\.USAGELOGS/i)).toBeInTheDocument()
    expect(screen.getByText('gateway.drawLogs')).toBeInTheDocument()
    expect(screen.getByText('gateway.taskLogs')).toBeInTheDocument()
  })

  it('shows noLogs placeholder when list returns empty', async () => {
    mockList.mockResolvedValue({ items: [], total: 0 })
    render(<GatewayLogPage />)

    await waitFor(() => expect(mockList).toHaveBeenCalled())
    expect(screen.getByText(/gateway\.noLogs/)).toBeInTheDocument()
  })

  // ── 3. Populated list ───────────────────────────────────────────────────

  it('renders log rows when list returns data', async () => {
    mockList.mockResolvedValue({ items: SAMPLE_LOGS, total: 2 })
    render(<GatewayLogPage />)

    await waitFor(() => expect(mockList).toHaveBeenCalled())

    expect(screen.getByText('alice')).toBeInTheDocument()
    expect(screen.getByText('claude-3-5-sonnet')).toBeInTheDocument()
    expect(screen.getByText('80+120')).toBeInTheDocument()
    expect(screen.getByText('Anthropic')).toBeInTheDocument()
    expect(screen.getByText('dev-key')).toBeInTheDocument()
    expect(screen.getByText('Consume')).toBeInTheDocument() // type=2
    expect(screen.getByText('System')).toBeInTheDocument()  // type=4
  })

  // ── 4. Error state ──────────────────────────────────────────────────────

  it('shows error banner when list rejects', async () => {
    mockList.mockRejectedValue(new Error('gateway: HTTP 502'))
    render(<GatewayLogPage />)

    await waitFor(() => {
      expect(screen.getByText(/HTTP 502/)).toBeInTheDocument()
    })
  })

  // ── 5. Stats panel (toggle + load) ──────────────────────────────────────

  it('loads and shows stats when the stats button is clicked', async () => {
    mockList.mockResolvedValue({ items: [], total: 0 })
    mockStats.mockResolvedValue({ quota: 1234, rpm: 42, tpm: 9999 })

    render(<GatewayLogPage />)
    await waitFor(() => expect(mockList).toHaveBeenCalled())

    // Click the stats toggle button (BarChart3 icon button, title = gateway.logStats)
    const statsBtn = screen.getByTitle('gateway.logStats')
    fireEvent.click(statsBtn)

    await waitFor(() => expect(mockStats).toHaveBeenCalled())

    expect(screen.getByText('1234')).toBeInTheDocument()
    expect(screen.getByText('42')).toBeInTheDocument()
    expect(screen.getByText('9999')).toBeInTheDocument()
  })

  it('surfaces error in banner when stats call rejects', async () => {
    mockList.mockResolvedValue({ items: [], total: 0 })
    mockStats.mockRejectedValue(new Error('stats unavailable'))

    render(<GatewayLogPage />)
    await waitFor(() => expect(mockList).toHaveBeenCalled())

    fireEvent.click(screen.getByTitle('gateway.logStats'))

    await waitFor(() => {
      expect(screen.getByText(/stats unavailable/)).toBeInTheDocument()
    })
  })

  // ── 6. Clear-history flow (success) ─────────────────────────────────────

  it('shows confirm modal on clear-history click and clears logs on confirm', async () => {
    mockList.mockResolvedValue({ items: SAMPLE_LOGS, total: 2 })

    render(<GatewayLogPage />)
    await waitFor(() => expect(mockList).toHaveBeenCalled())

    // Row visible before clear
    expect(screen.getByText('alice')).toBeInTheDocument()

    fireEvent.click(screen.getByText('gateway.clearHistory'))
    expect(screen.getByTestId('confirm-modal')).toBeInTheDocument()
    expect(screen.getByText('gateway.clearConfirmTitle')).toBeInTheDocument()

    // Confirm the action
    fireEvent.click(screen.getByText('confirm'))

    await waitFor(() => expect(mockClearHistory).toHaveBeenCalled())

    // Modal dismissed and logs cleared from view
    expect(screen.queryByTestId('confirm-modal')).not.toBeInTheDocument()
    expect(screen.queryByText('alice')).not.toBeInTheDocument()
  })

  // ── 7. Clear-history failure ─────────────────────────────────────────────

  it('surfaces error banner when clearHistory rejects', async () => {
    mockList.mockResolvedValue({ items: SAMPLE_LOGS, total: 2 })
    mockClearHistory.mockRejectedValue(new Error('clear failed: 500'))

    render(<GatewayLogPage />)
    await waitFor(() => expect(mockList).toHaveBeenCalled())

    fireEvent.click(screen.getByText('gateway.clearHistory'))
    fireEvent.click(screen.getByText('confirm'))

    await waitFor(() => {
      expect(screen.getByText(/clear failed: 500/)).toBeInTheDocument()
    })
  })

  // ── 8. Dismiss confirm modal via cancel ───────────────────────────────────

  it('cancels clear-history confirm modal without clearing logs', async () => {
    mockList.mockResolvedValue({ items: SAMPLE_LOGS, total: 2 })

    render(<GatewayLogPage />)
    await waitFor(() => expect(mockList).toHaveBeenCalled())

    fireEvent.click(screen.getByText('gateway.clearHistory'))
    expect(screen.getByTestId('confirm-modal')).toBeInTheDocument()

    fireEvent.click(screen.getByText('cancel'))

    expect(screen.queryByTestId('confirm-modal')).not.toBeInTheDocument()
    expect(mockClearHistory).not.toHaveBeenCalled()
    // Rows still present
    expect(screen.getByText('alice')).toBeInTheDocument()
  })

  // ── 9. Tab switching triggers new list call ──────────────────────────────

  it('switches tab and calls list with type=4 when task tab is clicked', async () => {
    mockList.mockResolvedValue({ items: [], total: 0 })
    render(<GatewayLogPage />)

    await waitFor(() => expect(mockList).toHaveBeenCalledTimes(1))

    // Click the "task" tab
    fireEvent.click(screen.getByText('gateway.taskLogs'))

    await waitFor(() => expect(mockList).toHaveBeenCalledTimes(2))

    // The second call should pass type: 4
    const secondCallFilter = mockList.mock.calls[1][0] as { type?: number }
    expect(secondCallFilter.type).toBe(4)
  })
})
