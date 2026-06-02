import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'

// i18n stub — returns fallback string so assertions match user-facing copy
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

// Wails runtime stub (ModelHealthMatrix uses EventsOn)
vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn().mockReturnValue(() => {}),
  EventsOff: vi.fn(),
  EventsEmit: vi.fn(),
}))

// Wails Go bindings used by sub-components (ModelHealthMatrix / ModelAuthenticityPanel)
vi.mock('../../wailsjs/go/main/App', () => ({
  RunModelHealthCheck: vi.fn().mockResolvedValue(undefined),
  GetLastHealthCheckResults: vi.fn().mockResolvedValue([]),
  RunModelAuthenticityCheck: vi.fn().mockResolvedValue(undefined),
  GetLastAuthenticityResults: vi.fn().mockResolvedValue([]),
}))

vi.mock('../../wailsjs/go/models', () => ({
  admin: {},
}))

// Stub heavy sub-components that have their own async dependencies so the
// GatewayModelPage tests stay focused on model/vendor/sync behaviour.
vi.mock('../components/ModelHealthMatrix', () => ({
  ModelHealthMatrix: () => <div data-testid="model-health-matrix" />,
}))

vi.mock('../components/ModelAuthenticityPanel', () => ({
  ModelAuthenticityPanel: () => <div data-testid="model-authenticity-panel" />,
}))

// --- Gateway API mock ---
// We expose per-method vi.fn() so each test can configure return values.
const mockGetModels = vi.fn()
const mockGetVendors = vi.fn()
const mockCreateModel = vi.fn()
const mockUpdateModel = vi.fn()
const mockDeleteModel = vi.fn()
const mockCreateVendor = vi.fn()
const mockDeleteVendor = vi.fn()
const mockSyncUpstreamPreview = vi.fn()
const mockSyncUpstream = vi.fn()
const mockGetMissingModels = vi.fn()

vi.mock('../lib/gateway-api', () => ({
  createGatewayClient: () => ({
    getModels: (...a: unknown[]) => mockGetModels(...a),
    getVendors: (...a: unknown[]) => mockGetVendors(...a),
    createModel: (...a: unknown[]) => mockCreateModel(...a),
    updateModel: (...a: unknown[]) => mockUpdateModel(...a),
    deleteModel: (...a: unknown[]) => mockDeleteModel(...a),
    createVendor: (...a: unknown[]) => mockCreateVendor(...a),
    updateVendor: vi.fn().mockResolvedValue({ success: true, message: '' }),
    deleteVendor: (...a: unknown[]) => mockDeleteVendor(...a),
    syncUpstreamPreview: (...a: unknown[]) => mockSyncUpstreamPreview(...a),
    syncUpstream: (...a: unknown[]) => mockSyncUpstream(...a),
    getMissingModels: (...a: unknown[]) => mockGetMissingModels(...a),
  }),
}))

// --- gatewayStore mock ---
// The page calls useGatewayStore() with NO selector — destructuring the whole state.
// So we mock it as a function returning the state object directly.
const mockGatewayState = {
  status: { running: true, port: 3000, url: 'http://localhost:3000', uptime: 0, version: '1.0', binaryOk: true },
  adminToken: 'test-admin-token' as string | null,
}

vi.mock('../stores/gatewayStore', () => ({
  useGatewayStore: () => mockGatewayState,
}))

// --- Pagination / SearchBar / ConfirmModal stubs ---
// These are real components; no need to stub — they render fine in jsdom.

import { GatewayModelPage } from './GatewayModelPage'

// Helper: make getModels resolve with a list of models
function makeModel(overrides: Partial<{
  id: number; model_name: string; developer: string; type: string
  context_length: number; input_price: number; output_price: number
  vendor_id: number; tags: string[]; status: number
}> = {}) {
  return {
    id: overrides.id ?? 1,
    model_name: overrides.model_name ?? 'gpt-4',
    developer: overrides.developer ?? 'OpenAI',
    type: overrides.type ?? 'chat',
    context_length: overrides.context_length ?? 128000,
    input_price: overrides.input_price ?? 0.03,
    output_price: overrides.output_price ?? 0.06,
    vendor_id: overrides.vendor_id ?? 0,
    tags: overrides.tags ?? ['reasoning'],
    status: overrides.status ?? 1,
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  // Default happy-path stubs
  mockGetModels.mockResolvedValue({ data: [], total: 0 })
  mockGetVendors.mockResolvedValue({ data: [] })
  mockCreateModel.mockResolvedValue({ success: true, message: '' })
  mockUpdateModel.mockResolvedValue({ success: true, message: '' })
  mockDeleteModel.mockResolvedValue({ success: true, message: '' })
  mockCreateVendor.mockResolvedValue({ success: true, message: '' })
  mockDeleteVendor.mockResolvedValue({ success: true, message: '' })
  mockSyncUpstreamPreview.mockResolvedValue({ data: [] })
  mockSyncUpstream.mockResolvedValue({ success: true, message: '' })
  mockGetMissingModels.mockResolvedValue({ data: [] })
})

describe('GatewayModelPage', () => {
  // ===== Server-stopped guard =====
  it('shows stopped notice when server is not running', () => {
    mockGatewayState.status = {
      running: false, port: 3000, url: '', uptime: 0, version: '', binaryOk: false,
    }
    render(<GatewayModelPage />)
    expect(screen.getByText('gateway.status.stopped')).toBeInTheDocument()
    // Restore for subsequent tests
    mockGatewayState.status = {
      running: true, port: 3000, url: 'http://localhost:3000', uptime: 0, version: '1.0', binaryOk: true,
    }
  })

  // ===== Initial load — empty models =====
  it('renders empty-state row when models list is empty', async () => {
    mockGetModels.mockResolvedValue({ data: [], total: 0 })
    render(<GatewayModelPage />)
    // "No models" is the fallback for gateway.noModels; the text is split as "▪ " + "No models"
    expect(await screen.findByText(/No models/)).toBeInTheDocument()
  })

  // ===== Models tab — data renders =====
  it('renders model rows returned from the API', async () => {
    mockGetModels.mockResolvedValue({
      data: [
        makeModel({ id: 1, model_name: 'claude-opus-4', developer: 'Anthropic', status: 1 }),
        makeModel({ id: 2, model_name: 'gpt-4o', developer: 'OpenAI', status: 0 }),
      ],
      total: 2,
    })
    render(<GatewayModelPage />)

    expect(await screen.findByText('claude-opus-4')).toBeInTheDocument()
    expect(screen.getByText('Anthropic')).toBeInTheDocument()
    expect(screen.getByText('gpt-4o')).toBeInTheDocument()
    // Enabled / disabled badge
    expect(screen.getByText('▸ Enabled')).toBeInTheDocument()
    expect(screen.getByText('▪ Disabled')).toBeInTheDocument()
  })

  // ===== Error banner on load failure =====
  it('shows error banner when getModels rejects', async () => {
    mockGetModels.mockRejectedValue(new Error('HTTP 503: gateway down'))
    render(<GatewayModelPage />)

    await waitFor(() => {
      expect(screen.getByText(/HTTP 503/)).toBeInTheDocument()
    })
  })

  // ===== Create model — success path =====
  it('calls createModel and reloads list when Add Model form is submitted', async () => {
    mockGetModels.mockResolvedValue({ data: [], total: 0 })
    render(<GatewayModelPage />)

    // Wait for initial load
    await screen.findByText(/No models/)

    // Open the modal
    fireEvent.click(screen.getByText('Add Model'))

    // Fill in model name (find by placeholder / label)
    const nameInput = screen.getAllByRole('textbox')[0]
    fireEvent.change(nameInput, { target: { value: 'my-new-model' } })

    // Second load after save
    mockGetModels.mockResolvedValue({
      data: [makeModel({ id: 10, model_name: 'my-new-model' })],
      total: 1,
    })

    // Click Save (fallback t('settings.save'))
    fireEvent.click(screen.getByText('settings.save'))

    await waitFor(() => {
      expect(mockCreateModel).toHaveBeenCalledTimes(1)
    })
    expect(mockCreateModel).toHaveBeenCalledWith(
      expect.objectContaining({ model_name: 'my-new-model' }),
    )
    // After save the modal is closed and models reload
    await waitFor(() => {
      expect(screen.getByText('my-new-model')).toBeInTheDocument()
    })
  })

  // ===== Create model — failure surfaces error =====
  it('shows error banner when createModel rejects', async () => {
    mockGetModels.mockResolvedValue({ data: [], total: 0 })
    render(<GatewayModelPage />)
    await screen.findByText(/No models/)

    fireEvent.click(screen.getByText('Add Model'))

    mockCreateModel.mockRejectedValue(new Error('create model: HTTP 500'))
    fireEvent.click(screen.getByText('settings.save'))

    await waitFor(() => {
      expect(screen.getByText(/HTTP 500/)).toBeInTheDocument()
    })
  })

  // ===== Delete model via ConfirmModal =====
  it('calls deleteModel and reloads after confirm-delete', async () => {
    mockGetModels.mockResolvedValue({
      data: [makeModel({ id: 7, model_name: 'model-to-delete' })],
      total: 1,
    })
    render(<GatewayModelPage />)
    await screen.findByText('model-to-delete')

    // Click the delete icon button for the first model row
    const deleteButtons = screen.getAllByTitle('Delete')
    fireEvent.click(deleteButtons[0])

    // ConfirmModal appears — click the "Confirm" button
    const confirmBtn = await screen.findByRole('button', { name: 'Confirm' })

    mockGetModels.mockResolvedValue({ data: [], total: 0 })
    fireEvent.click(confirmBtn)

    await waitFor(() => {
      expect(mockDeleteModel).toHaveBeenCalledWith(7)
    })
  })

  // ===== Tab navigation — Vendors tab =====
  it('loads vendors when Vendors tab is clicked', async () => {
    mockGetModels.mockResolvedValue({ data: [], total: 0 })
    mockGetVendors.mockResolvedValue({
      data: [
        { id: 3, name: 'Anthropic Inc', description: 'Claude family', icon_url: '', website: 'https://anthropic.com', status: 1 },
      ],
    })
    render(<GatewayModelPage />)
    await screen.findByText(/No models/)

    fireEvent.click(screen.getByText('Vendors'))

    expect(await screen.findByText('Anthropic Inc')).toBeInTheDocument()
    expect(screen.getByText('https://anthropic.com')).toBeInTheDocument()
  })

  // ===== Vendors tab error =====
  it('shows error banner when getVendors rejects', async () => {
    mockGetModels.mockResolvedValue({ data: [], total: 0 })
    mockGetVendors.mockRejectedValue(new Error('vendors: HTTP 502'))
    render(<GatewayModelPage />)
    await screen.findByText(/No models/)

    fireEvent.click(screen.getByText('Vendors'))

    await waitFor(() => {
      expect(screen.getByText(/HTTP 502/)).toBeInTheDocument()
    })
  })

  // ===== Sync tab — Preview Upstream =====
  it('shows sync preview results after clicking Preview Upstream', async () => {
    mockGetModels.mockResolvedValue({ data: [], total: 0 })
    mockSyncUpstreamPreview.mockResolvedValue({
      data: [
        makeModel({ model_name: 'llama-4-scout', developer: 'Meta' }),
        makeModel({ model_name: 'gemini-2-flash', developer: 'Google' }),
      ],
    })
    render(<GatewayModelPage />)
    await screen.findByText(/No models/)

    fireEvent.click(screen.getByText('Sync'))
    fireEvent.click(screen.getByText('Preview Upstream'))

    expect(await screen.findByText('llama-4-scout')).toBeInTheDocument()
    expect(screen.getByText('gemini-2-flash')).toBeInTheDocument()
  })

  // ===== Sync tab — Sync Now error =====
  it('shows error banner when syncUpstream rejects', async () => {
    mockGetModels.mockResolvedValue({ data: [], total: 0 })
    mockSyncUpstream.mockRejectedValue(new Error('sync: HTTP 504'))
    render(<GatewayModelPage />)
    await screen.findByText(/No models/)

    fireEvent.click(screen.getByText('Sync'))
    fireEvent.click(screen.getByText('Sync Now'))

    await waitFor(() => {
      expect(screen.getByText(/HTTP 504/)).toBeInTheDocument()
    })
  })

  // ===== Health tab — sub-components rendered =====
  it('renders health sub-components when Health tab is clicked', async () => {
    mockGetModels.mockResolvedValue({ data: [], total: 0 })
    render(<GatewayModelPage />)
    await screen.findByText(/No models/)

    fireEvent.click(screen.getByText('可用性自检'))

    expect(screen.getByTestId('model-health-matrix')).toBeInTheDocument()
    expect(screen.getByTestId('model-authenticity-panel')).toBeInTheDocument()
  })
})
