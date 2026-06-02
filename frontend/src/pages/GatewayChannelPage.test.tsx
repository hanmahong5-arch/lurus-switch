import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import type { ChannelSource, ChannelSourceCapabilities } from '../lib/channelSource'
import type { GatewayChannel } from '../lib/gateway-api'

// ---------------------------------------------------------------------------
// i18n stub — returns fallback string or key.
// ---------------------------------------------------------------------------
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

// ---------------------------------------------------------------------------
// Wails binding stubs — channelSource is the real consumer of these;
// we mock channelSource directly, so these stubs just prevent import errors.
// ---------------------------------------------------------------------------
vi.mock('../../wailsjs/go/main/App', () => ({
  HubListChannels: vi.fn(),
  HubSearchChannels: vi.fn(),
  HubAddChannel: vi.fn(),
  HubUpdateChannel: vi.fn(),
  HubDeleteChannel: vi.fn(),
  HubDeleteChannelBatch: vi.fn(),
  HubTestChannel: vi.fn(),
  HubCopyChannel: vi.fn(),
  HubBatchSetChannelTag: vi.fn(),
  HubEnableChannelsByTag: vi.fn(),
  HubDisableChannelsByTag: vi.fn(),
  HubEditChannelTag: vi.fn(),
  HubFetchChannelModels: vi.fn(),
  HubFixChannelAbilities: vi.fn(),
}))

vi.mock('../../wailsjs/go/models', () => ({
  admin: {},
}))

// ---------------------------------------------------------------------------
// gatewayStore — the page calls useGatewayStore() with NO selector, so the
// mock must return the state object directly (not apply a selector fn).
// ---------------------------------------------------------------------------
const mockUseGatewayStore = vi.fn()
vi.mock('../stores/gatewayStore', () => ({
  useGatewayStore: () => mockUseGatewayStore(),
}))

// ---------------------------------------------------------------------------
// configStore — default to 'personal'. Reseller tests override per-case.
// ---------------------------------------------------------------------------
const mockUseConfigStore = vi.fn()
vi.mock('../stores/configStore', () => ({
  useConfigStore: <T,>(selector: (s: { appMode: string }) => T) =>
    selector({ appMode: mockUseConfigStore() }),
}))

// ---------------------------------------------------------------------------
// channelSource — fully replace makeChannelSource so we control the source
// that the page uses. The factory vi.fn() decides what to return per test.
// ---------------------------------------------------------------------------
const mockMakeChannelSource = vi.fn()

// Full-capability stub as a baseline — tests override individual methods.
const fullCaps: ChannelSourceCapabilities = {
  search: true,
  copy: true,
  fetchModels: true,
  batchEnableDisable: true,
  batchSetTag: true,
  tagOperations: true,
  fixAbilities: true,
}

function buildMockSource(overrides: Partial<ChannelSource> = {}): ChannelSource {
  return {
    kind: 'hub',
    capabilities: fullCaps,
    list: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    create: vi.fn().mockResolvedValue(undefined),
    update: vi.fn().mockResolvedValue(undefined),
    delete: vi.fn().mockResolvedValue(undefined),
    batchDelete: vi.fn().mockResolvedValue(undefined),
    test: vi.fn().mockResolvedValue('OK'),
    copy: vi.fn().mockResolvedValue(undefined),
    fetchModels: vi.fn().mockResolvedValue([]),
    batchEnable: vi.fn().mockResolvedValue(undefined),
    batchDisable: vi.fn().mockResolvedValue(undefined),
    batchSetTag: vi.fn().mockResolvedValue(undefined),
    enableByTag: vi.fn().mockResolvedValue(undefined),
    disableByTag: vi.fn().mockResolvedValue(undefined),
    editTag: vi.fn().mockResolvedValue(undefined),
    fixAbilities: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  }
}

vi.mock('../lib/channelSource', async (importOriginal) => {
  const orig = await importOriginal<typeof import('../lib/channelSource')>()
  return {
    ...orig,
    makeChannelSource: (...args: unknown[]) => mockMakeChannelSource(...args),
  }
})

// ---------------------------------------------------------------------------
// Import under test — after all vi.mock() hoists.
// ---------------------------------------------------------------------------
import { GatewayChannelPage } from './GatewayChannelPage'

// ---------------------------------------------------------------------------
// Sample fixture data.
// ---------------------------------------------------------------------------
const SAMPLE_CHANNELS: GatewayChannel[] = [
  {
    id: 1,
    name: 'Anthropic Direct',
    type: 1,
    key: 'sk-ant-1',
    base_url: 'https://api.anthropic.com',
    models: 'claude-3-5-sonnet',
    balance: 42.5,
    status: 1,
    response_time: 120,
    created_time: 1716000000,
    test_time: 0,
    tag: 'prod',
    group: 'default',
    model_mapping: '',
    priority: 10,
    weight: 1,
    auto_ban: 0,
    other: '',
  },
  {
    id: 2,
    name: 'OpenAI Fallback',
    type: 2,
    key: 'sk-openai-2',
    base_url: '',
    models: 'gpt-4o',
    balance: 8.0,
    status: 2,
    response_time: 0,
    created_time: 1716000001,
    test_time: 0,
    tag: '',
    group: 'default',
    model_mapping: '',
    priority: 5,
    weight: 1,
    auto_ban: 0,
    other: '',
  },
]

// ---------------------------------------------------------------------------
// beforeEach: reset mocks; set Personal mode with running gateway by default.
// ---------------------------------------------------------------------------
beforeEach(() => {
  vi.clearAllMocks()
  mockUseConfigStore.mockReturnValue('personal')
  mockUseGatewayStore.mockReturnValue({
    status: { running: true, url: 'http://localhost:3000' } as { running: boolean; url: string },
    adminToken: 'admin-token-xxx',
  })
  // Default source: empty channel list.
  mockMakeChannelSource.mockReturnValue(buildMockSource())
})

// ---------------------------------------------------------------------------
describe('GatewayChannelPage', () => {
  // --- Source unavailable states -----------------------------------------------

  it('shows gateway-stopped notice in Personal mode when gateway is not running', () => {
    mockUseGatewayStore.mockReturnValue({ status: null as null, adminToken: null as null })
    mockMakeChannelSource.mockReturnValue(null)

    render(<GatewayChannelPage />)

    // page falls back to unavailable state — shows the stopped key.
    expect(screen.getByText(/gateway.status.stopped/)).toBeInTheDocument()
  })

  it('shows hub-not-configured notice in Reseller mode when makeChannelSource returns null', () => {
    mockUseConfigStore.mockReturnValue('reseller')
    mockMakeChannelSource.mockReturnValue(null)

    render(<GatewayChannelPage />)

    expect(
      screen.getByText('请先在「设置」中配置 Reseller Hub URL 与管理员 Token')
    ).toBeInTheDocument()
  })

  // --- Initial render with data -----------------------------------------------

  it('renders the channel list after load', async () => {
    const src = buildMockSource({
      list: vi.fn().mockResolvedValue({ items: SAMPLE_CHANNELS, total: 2 }),
    })
    mockMakeChannelSource.mockReturnValue(src)

    render(<GatewayChannelPage />)

    expect(await screen.findByText('Anthropic Direct')).toBeInTheDocument()
    expect(screen.getByText('OpenAI Fallback')).toBeInTheDocument()
    // Tag rendered for channel 1; channel 2 has no tag.
    expect(screen.getByText('prod')).toBeInTheDocument()
    // IDs shown as monospace numerals.
    expect(screen.getByText('1')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
  })

  // --- Empty state -----------------------------------------------

  it('shows "No channels" empty placeholder when list returns empty', async () => {
    const src = buildMockSource({
      list: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    })
    mockMakeChannelSource.mockReturnValue(src)

    render(<GatewayChannelPage />)

    expect(await screen.findByText(/No channels/)).toBeInTheDocument()
  })

  // --- Error state -----------------------------------------------

  it('shows error banner when list() rejects', async () => {
    const src = buildMockSource({
      list: vi.fn().mockRejectedValue(new Error('upstream 503: service unavailable')),
    })
    mockMakeChannelSource.mockReturnValue(src)

    render(<GatewayChannelPage />)

    await waitFor(() => {
      expect(screen.getByText(/upstream 503: service unavailable/)).toBeInTheDocument()
    })
  })

  // --- Delete action: success path -----------------------------------------------

  it('removes a channel from the table after successful delete via confirm modal', async () => {
    const mockDelete = vi.fn().mockResolvedValue(undefined)
    const src = buildMockSource({
      list: vi.fn().mockResolvedValue({ items: SAMPLE_CHANNELS, total: 2 }),
      delete: mockDelete,
    })
    mockMakeChannelSource.mockReturnValue(src)

    render(<GatewayChannelPage />)

    // Wait for rows to render.
    expect(await screen.findByText('Anthropic Direct')).toBeInTheDocument()

    // Click the trash icon for channel 1 (first delete button).
    const deleteButtons = screen.getAllByTitle('Delete')
    fireEvent.click(deleteButtons[0])

    // Confirm modal should appear.
    expect(await screen.findByText('Confirm Delete')).toBeInTheDocument()

    // Mock source.list to return only channel 2 after delete so the table updates.
    ;(src.list as ReturnType<typeof vi.fn>).mockResolvedValue({
      items: [SAMPLE_CHANNELS[1]],
      total: 1,
    })

    // Click the danger confirm button inside the modal.
    const confirmBtn = screen.getAllByRole('button').find(
      (b) => b.textContent?.includes('Confirm') && b !== deleteButtons[0]
    )
    expect(confirmBtn).toBeDefined()
    fireEvent.click(confirmBtn!)

    // delete was called with channel id=1.
    await waitFor(() => {
      expect(mockDelete).toHaveBeenCalledWith(1)
    })
  })

  // --- Delete action: failure path (error surfaced in banner) -----------------------------------------------

  it('shows error banner when delete() rejects', async () => {
    const mockDelete = vi.fn().mockRejectedValue(new Error('delete forbidden: read-only key'))
    const src = buildMockSource({
      list: vi.fn().mockResolvedValue({ items: SAMPLE_CHANNELS, total: 2 }),
      delete: mockDelete,
    })
    mockMakeChannelSource.mockReturnValue(src)

    render(<GatewayChannelPage />)
    expect(await screen.findByText('Anthropic Direct')).toBeInTheDocument()

    const deleteButtons = screen.getAllByTitle('Delete')
    fireEvent.click(deleteButtons[0])

    // Confirm via modal.
    const confirmBtn = screen.getAllByRole('button').find(
      (b) => b.textContent?.includes('Confirm') && b !== deleteButtons[0]
    )
    fireEvent.click(confirmBtn!)

    await waitFor(() => {
      expect(screen.getByText(/delete forbidden: read-only key/)).toBeInTheDocument()
    })
  })

  // --- Test action -----------------------------------------------

  it('shows "testing..." then the result in the Test column after clicking test', async () => {
    const mockTest = vi.fn().mockResolvedValue('response_time: 142ms')
    const src = buildMockSource({
      list: vi.fn().mockResolvedValue({ items: [SAMPLE_CHANNELS[0]], total: 1 }),
      test: mockTest,
    })
    mockMakeChannelSource.mockReturnValue(src)

    render(<GatewayChannelPage />)
    expect(await screen.findByText('Anthropic Direct')).toBeInTheDocument()

    const testBtn = screen.getByTitle('Test')
    fireEvent.click(testBtn)

    await waitFor(() => {
      expect(mockTest).toHaveBeenCalledWith(1)
      expect(screen.getByText('response_time: 142ms')).toBeInTheDocument()
    })
  })

  // --- Add channel modal -----------------------------------------------

  it('opens Create modal when "Add Channel" is clicked', async () => {
    const src = buildMockSource({
      list: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    })
    mockMakeChannelSource.mockReturnValue(src)

    render(<GatewayChannelPage />)
    await screen.findByText(/No channels/)

    fireEvent.click(screen.getByText('Add Channel'))

    // After click the modal renders — the form field labels become visible.
    // The Name field label appears in the modal form.
    expect(await screen.findByPlaceholderText('sk-...')).toBeInTheDocument()
  })

  // --- Reseller mode: source is Hub -----------------------------------------------

  it('calls makeChannelSource with mode=hub in Reseller mode', async () => {
    mockUseConfigStore.mockReturnValue('reseller')
    const src = buildMockSource({
      kind: 'hub',
      list: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    })
    mockMakeChannelSource.mockReturnValue(src)

    render(<GatewayChannelPage />)
    await screen.findByText(/No channels/)

    expect(mockMakeChannelSource).toHaveBeenCalledWith({ mode: 'hub' })
  })

  // --- Status toggle: optimistic update -----------------------------------------------

  it('toggles channel status optimistically on click', async () => {
    const mockUpdate = vi.fn().mockResolvedValue(undefined)
    const src = buildMockSource({
      list: vi.fn().mockResolvedValue({ items: [SAMPLE_CHANNELS[0]], total: 1 }),
      update: mockUpdate,
    })
    mockMakeChannelSource.mockReturnValue(src)

    render(<GatewayChannelPage />)
    expect(await screen.findByText('Anthropic Direct')).toBeInTheDocument()

    // The status badge for a status=1 channel is "enabled". Click to disable.
    const statusBtn = screen.getByTitle('Click to disable')
    fireEvent.click(statusBtn)

    await waitFor(() => {
      expect(mockUpdate).toHaveBeenCalledWith(expect.objectContaining({ id: 1, status: 2 }))
    })
  })

  // --- update() failure -----------------------------------------------

  it('shows error banner when status toggle update() rejects', async () => {
    const mockUpdate = vi.fn().mockRejectedValue(new Error('update: HTTP 422 Unprocessable Entity'))
    const src = buildMockSource({
      list: vi.fn().mockResolvedValue({ items: [SAMPLE_CHANNELS[0]], total: 1 }),
      update: mockUpdate,
    })
    mockMakeChannelSource.mockReturnValue(src)

    render(<GatewayChannelPage />)
    expect(await screen.findByText('Anthropic Direct')).toBeInTheDocument()

    fireEvent.click(screen.getByTitle('Click to disable'))

    await waitFor(() => {
      expect(screen.getByText(/HTTP 422 Unprocessable Entity/)).toBeInTheDocument()
    })
  })
})
