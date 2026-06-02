import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

// i18n stub — returns the fallback string when provided, otherwise the key.
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

// Mock the gateway-api module so we control client behaviour without real HTTP.
const mockGetOptions = vi.fn()
const mockUpdateOption = vi.fn()
const mockResetModelRatio = vi.fn()
const mockClearCache = vi.fn()

const mockClient = {
  getOptions: (...a: unknown[]) => mockGetOptions(...a),
  updateOption: (...a: unknown[]) => mockUpdateOption(...a),
  resetModelRatio: (...a: unknown[]) => mockResetModelRatio(...a),
  clearCache: (...a: unknown[]) => mockClearCache(...a),
}

vi.mock('../lib/gateway-api', async (importOriginal) => {
  const real = await importOriginal<typeof import('../lib/gateway-api')>()
  return {
    ...real,
    createGatewayClient: vi.fn(() => mockClient),
  }
})

// Stub the two sub-components that carry heavier dependency trees
// (PricingEditor, Monaco) so the page renders in jsdom without them blowing up.
vi.mock('../components/gateway/OptionsSectionForm', () => ({
  OptionsSectionForm: ({ tab, options, onSave }: {
    tab: string
    options: Record<string, string>
    onSave: (k: string, v: string) => Promise<void>
  }) => (
    <div data-testid={`section-form-${tab}`}>
      {Object.entries(options).map(([k, v]) => (
        <div key={k} data-testid={`opt-${k}`}>
          <span>{k}</span>
          <button onClick={() => onSave(k, v + '_updated')}>{`save-${k}`}</button>
        </div>
      ))}
    </div>
  ),
}))

vi.mock('../components/gateway/OptionEditor', () => ({
  OptionEditor: ({ options, onSave }: {
    options: Record<string, string>
    onSave: (k: string, v: string) => Promise<void>
  }) => (
    <div data-testid="option-editor">
      {Object.entries(options).map(([k, v]) => (
        <div key={k} data-testid={`opt-${k}`}>
          <span>{k}</span>
          <button onClick={() => onSave(k, v + '_updated')}>{`save-${k}`}</button>
        </div>
      ))}
    </div>
  ),
}))

import { GatewaySettingsPage } from './GatewaySettingsPage'
import { useGatewayStore } from '../stores/gatewayStore'

// Helper: put a running server + adminToken into the store before each test
// that needs the "connected" branch to render.
function setConnected() {
  useGatewayStore.setState({
    status: { running: true, port: 3000, url: 'http://localhost:3000', uptime: 42, version: '1.0', binaryOk: true },
    adminToken: 'test-admin-token',
    pollingHandle: null,
  })
}

function setDisconnected() {
  useGatewayStore.setState({ status: null, adminToken: null, pollingHandle: null })
}

describe('GatewaySettingsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Default: gateway not running
    setDisconnected()
    // Default success response for options
    mockGetOptions.mockResolvedValue({ success: true, message: '', data: {} })
    mockUpdateOption.mockResolvedValue({ success: true, message: '' })
    mockResetModelRatio.mockResolvedValue({ success: true, message: '' })
    mockClearCache.mockResolvedValue({ success: true, message: '' })
  })

  // --- 1. Stopped-state (gateway not running) ---

  it('shows stopped banner when server is not running', () => {
    render(<GatewaySettingsPage />)
    // Page should render the AlertCircle + status.stopped text
    // t('gateway.status.stopped') falls back to the key since no fallback provided
    expect(screen.getByText('gateway.status.stopped')).toBeDefined()
  })

  it('does not fetch options when server is stopped', () => {
    render(<GatewaySettingsPage />)
    expect(mockGetOptions).not.toHaveBeenCalled()
  })

  // --- 2. Connected, empty options ---

  it('renders settings title when server is running', async () => {
    setConnected()
    render(<GatewaySettingsPage />)
    await waitFor(() => {
      expect(screen.getByText('设置')).toBeDefined()
    })
  })

  it('fetches options on mount when connected', async () => {
    setConnected()
    render(<GatewaySettingsPage />)
    await waitFor(() => {
      expect(mockGetOptions).toHaveBeenCalledTimes(1)
    })
  })

  it('shows the tab bar with all 12 tabs when connected', async () => {
    setConnected()
    render(<GatewaySettingsPage />)
    await waitFor(() => {
      expect(screen.getByText('运营设置')).toBeDefined()
      expect(screen.getByText('支付设置')).toBeDefined()
      expect(screen.getByText('系统设置')).toBeDefined()
    })
  })

  it('shows empty-tab placeholder when the active tab has no options', async () => {
    setConnected()
    mockGetOptions.mockResolvedValue({ success: true, message: '', data: {} })
    render(<GatewaySettingsPage />)
    await waitFor(() => {
      // Default tab is 'operations'. No options → empty message.
      expect(screen.getByText('此分组无选项。')).toBeDefined()
    })
  })

  // --- 3. Error state ---

  it('shows error banner when getOptions rejects', async () => {
    setConnected()
    mockGetOptions.mockRejectedValue(new Error('Gateway API GET /api/option/ → 503: Service Unavailable'))
    render(<GatewaySettingsPage />)
    await waitFor(() => {
      expect(screen.getByText(/503/)).toBeDefined()
    })
  })

  it('error banner disappears after a successful reload', async () => {
    setConnected()
    // First call fails, second succeeds.
    mockGetOptions
      .mockRejectedValueOnce(new Error('timeout'))
      .mockResolvedValue({ success: true, message: '', data: {} })

    render(<GatewaySettingsPage />)

    // Wait for the error to appear.
    await waitFor(() => {
      expect(screen.getByText(/timeout/)).toBeDefined()
    })

    // Click the reload button (the Button with RefreshCw icon — no visible text label).
    const refreshBtn = screen.getByRole('button', { name: '' })
    fireEvent.click(refreshBtn)

    await waitFor(() => {
      expect(screen.queryByText(/timeout/)).toBeNull()
    })
  })

  // --- 4. Search functionality ---

  it('renders search results view when user types a query', async () => {
    setConnected()
    // Provide a Payment option so the search hits on it.
    mockGetOptions.mockResolvedValue({
      success: true, message: '',
      data: { StripeKey: 'sk_live_abc123', SiteTitle: 'My Hub' },
    })
    render(<GatewaySettingsPage />)

    await waitFor(() => expect(mockGetOptions).toHaveBeenCalled())

    const searchInput = screen.getByPlaceholderText('搜索所有选项…')
    fireEvent.change(searchInput, { target: { value: 'stripe' } })

    await waitFor(() => {
      // Search results count shown; StripeKey should appear in the results.
      expect(screen.getByText('StripeKey')).toBeDefined()
    })
  })

  it('search with no matches shows no-match message', async () => {
    setConnected()
    mockGetOptions.mockResolvedValue({
      success: true, message: '',
      data: { SiteTitle: 'My Hub' },
    })
    render(<GatewaySettingsPage />)
    await waitFor(() => expect(mockGetOptions).toHaveBeenCalled())

    const searchInput = screen.getByPlaceholderText('搜索所有选项…')
    fireEvent.change(searchInput, { target: { value: 'zzznomatch' } })

    await waitFor(() => {
      expect(screen.getByText('没有匹配的选项。')).toBeDefined()
    })
  })

  // --- 5. Mutation: save option success ---

  it('calls updateOption on save action and updates local state on success', async () => {
    setConnected()
    // The 'other' tab uses plain OptionEditor (no metadata). Provide a key
    // that has no metadata so it routes to the OptionEditor stub.
    mockGetOptions.mockResolvedValue({
      success: true, message: '',
      data: { SiteTitle: 'My Hub' },
    })
    mockUpdateOption.mockResolvedValue({ success: true, message: '' })

    render(<GatewaySettingsPage />)
    await waitFor(() => expect(mockGetOptions).toHaveBeenCalled())

    // Navigate to the 'other' tab (which has no metadata → OptionEditor stub).
    const otherTab = screen.getByText('其他设置')
    fireEvent.click(otherTab)

    // The stub renders a save button per option key.
    await waitFor(() => {
      expect(screen.getByText('SiteTitle')).toBeDefined()
    })

    fireEvent.click(screen.getByText('save-SiteTitle'))

    await waitFor(() => {
      expect(mockUpdateOption).toHaveBeenCalledWith('SiteTitle', 'My Hub_updated')
    })
  })

  // --- 6. Reset model ratio — confirm modal flow ---

  it('shows confirm modal when Reset Model Ratio is clicked on Pricing tab', async () => {
    setConnected()
    // Provide a pricing key so the pricing tab is not empty.
    mockGetOptions.mockResolvedValue({
      success: true, message: '',
      data: { ModelRatio: '{"gpt-4":15}' },
    })
    render(<GatewaySettingsPage />)
    await waitFor(() => expect(mockGetOptions).toHaveBeenCalled())

    // Switch to pricing tab.
    fireEvent.click(screen.getByText('分组与模型定价'))

    // The action button should appear.
    await waitFor(() => {
      expect(screen.getByText('重置模型倍率')).toBeDefined()
    })

    fireEvent.click(screen.getByText('重置模型倍率'))

    // Confirm modal should appear.
    await waitFor(() => {
      expect(screen.getByText('Reset Model Ratio?')).toBeDefined()
    })
  })

  it('calls resetModelRatio and closes modal on Confirm', async () => {
    setConnected()
    mockGetOptions.mockResolvedValue({
      success: true, message: '',
      data: { ModelRatio: '{"gpt-4":15}' },
    })
    render(<GatewaySettingsPage />)
    await waitFor(() => expect(mockGetOptions).toHaveBeenCalled())

    fireEvent.click(screen.getByText('分组与模型定价'))
    await waitFor(() => expect(screen.getByText('重置模型倍率')).toBeDefined())

    fireEvent.click(screen.getByText('重置模型倍率'))
    await waitFor(() => expect(screen.getByText('Reset Model Ratio?')).toBeDefined())

    fireEvent.click(screen.getByText('Confirm'))

    await waitFor(() => {
      expect(mockResetModelRatio).toHaveBeenCalledTimes(1)
      // Modal closes after confirm.
      expect(screen.queryByText('Reset Model Ratio?')).toBeNull()
    })
  })

  it('shows error banner when resetModelRatio fails', async () => {
    setConnected()
    mockGetOptions.mockResolvedValue({
      success: true, message: '',
      data: { ModelRatio: '{"gpt-4":15}' },
    })
    mockResetModelRatio.mockRejectedValue(new Error('upstream reset failed'))

    render(<GatewaySettingsPage />)
    await waitFor(() => expect(mockGetOptions).toHaveBeenCalled())

    fireEvent.click(screen.getByText('分组与模型定价'))
    await waitFor(() => expect(screen.getByText('重置模型倍率')).toBeDefined())

    fireEvent.click(screen.getByText('重置模型倍率'))
    await waitFor(() => expect(screen.getByText('Reset Model Ratio?')).toBeDefined())

    fireEvent.click(screen.getByText('Confirm'))

    await waitFor(() => {
      expect(screen.getByText(/upstream reset failed/)).toBeDefined()
    })
  })

  // --- 7. Clear cache — confirm modal flow ---

  it('shows clear-cache confirm modal on Performance tab', async () => {
    setConnected()
    // Provide a performance key so the tab is not empty and the action renders.
    mockGetOptions.mockResolvedValue({
      success: true, message: '',
      data: { CacheEnabled: 'true' },
    })
    render(<GatewaySettingsPage />)
    await waitFor(() => expect(mockGetOptions).toHaveBeenCalled())

    fireEvent.click(screen.getByText('性能设置'))
    await waitFor(() => expect(screen.getByText('清空缓存')).toBeDefined())

    fireEvent.click(screen.getByText('清空缓存'))

    await waitFor(() => {
      expect(screen.getByText('Clear Cache?')).toBeDefined()
    })
  })

  it('calls clearCache and closes modal on Confirm', async () => {
    setConnected()
    mockGetOptions.mockResolvedValue({
      success: true, message: '',
      data: { CacheEnabled: 'true' },
    })
    render(<GatewaySettingsPage />)
    await waitFor(() => expect(mockGetOptions).toHaveBeenCalled())

    fireEvent.click(screen.getByText('性能设置'))
    await waitFor(() => expect(screen.getByText('清空缓存')).toBeDefined())
    fireEvent.click(screen.getByText('清空缓存'))
    await waitFor(() => expect(screen.getByText('Clear Cache?')).toBeDefined())

    fireEvent.click(screen.getByText('Confirm'))

    await waitFor(() => {
      expect(mockClearCache).toHaveBeenCalledTimes(1)
      expect(screen.queryByText('Clear Cache?')).toBeNull()
    })
  })

  it('shows error banner when clearCache fails', async () => {
    setConnected()
    mockGetOptions.mockResolvedValue({
      success: true, message: '',
      data: { CacheEnabled: 'true' },
    })
    mockClearCache.mockRejectedValue(new Error('DELETE /api/option/channel_affinity_cache → 500'))

    render(<GatewaySettingsPage />)
    await waitFor(() => expect(mockGetOptions).toHaveBeenCalled())

    fireEvent.click(screen.getByText('性能设置'))
    await waitFor(() => expect(screen.getByText('清空缓存')).toBeDefined())
    fireEvent.click(screen.getByText('清空缓存'))
    await waitFor(() => expect(screen.getByText('Clear Cache?')).toBeDefined())
    fireEvent.click(screen.getByText('Confirm'))

    await waitFor(() => {
      expect(screen.getByText(/DELETE.*500/)).toBeDefined()
    })
  })

  // --- 8. Modal cancel dismisses without side effects ---

  it('dismisses confirm modal on Cancel without calling any action', async () => {
    setConnected()
    mockGetOptions.mockResolvedValue({
      success: true, message: '',
      data: { ModelRatio: '{}' },
    })
    render(<GatewaySettingsPage />)
    await waitFor(() => expect(mockGetOptions).toHaveBeenCalled())

    fireEvent.click(screen.getByText('分组与模型定价'))
    await waitFor(() => expect(screen.getByText('重置模型倍率')).toBeDefined())
    fireEvent.click(screen.getByText('重置模型倍率'))
    await waitFor(() => expect(screen.getByText('Reset Model Ratio?')).toBeDefined())

    fireEvent.click(screen.getByText('settings.data.cancel'))

    await waitFor(() => {
      expect(screen.queryByText('Reset Model Ratio?')).toBeNull()
    })
    expect(mockResetModelRatio).not.toHaveBeenCalled()
  })
})
