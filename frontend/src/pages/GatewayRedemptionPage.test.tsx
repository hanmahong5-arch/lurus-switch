import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'

// i18n stub: returns the fallback when provided, otherwise the key.
// Supports {{var}} interpolation for banner copy.
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

// ──────────────────────────────────────────────────────────────
// Inject a controllable RedemptionSource rather than patching
// every individual Wails binding.  The mock captures factory
// calls and delegates to `mockSource`, which tests can override.
// ──────────────────────────────────────────────────────────────

const mockSource = {
  kind: 'hub' as const,
  list: vi.fn(),
  create: vi.fn(),
  delete: vi.fn(),
  deleteInvalid: vi.fn(),
}

vi.mock('../lib/redemptionSource', () => ({
  makeRedemptionSource: vi.fn(() => mockSource),
  downloadRedemptionsCSV: vi.fn(),
}))

// Wails bindings — the page does NOT call them directly; they are used by
// redemptionSource.ts which is mocked above.  Stub them to prevent the
// "window['go'] is not defined" error that jsdom surfaces when the module
// file is loaded.
vi.mock('../../wailsjs/go/main/App', () => ({
  HubListRedemptions: vi.fn(),
  HubCreateRedemptions: vi.fn(),
  HubDeleteRedemption: vi.fn(),
  HubDeleteInvalidRedemptions: vi.fn(),
}))

vi.mock('../../wailsjs/go/models', () => ({ admin: {} }))

// formatTime — tested separately; give a stable output.
vi.mock('../lib/formatTime', () => ({
  formatLocal: (_: unknown) => '2026-01-01 00:00',
}))

// ──────────────────────────────────────────────────────────────
// Store stubs
// ──────────────────────────────────────────────────────────────

// useGatewayStore() is called with NO selector — it returns the whole state.
const mockGatewayState = {
  status: null as { running: boolean; url: string } | null,
  adminToken: null as string | null,
}
vi.mock('../stores/gatewayStore', () => ({
  useGatewayStore: () => mockGatewayState,
}))

const mockUseConfigStore = vi.fn()
vi.mock('../stores/configStore', () => ({
  useConfigStore: <T,>(selector: (s: { appMode: string }) => T) =>
    selector({ appMode: mockUseConfigStore() }),
}))

import { GatewayRedemptionPage } from './GatewayRedemptionPage'

// ──────────────────────────────────────────────────────────────
// Sample data
// ──────────────────────────────────────────────────────────────

const SAMPLE_REDEMPTION = {
  id: 1,
  name: 'batch-alpha',
  key: 'ALPHA-1234',
  status: 1,
  quota: 500,
  count: 10,
  used_count: 3,
  created_time: 1748736000,
  redeemed_time: 0,
}

const EMPTY_PAGE = { items: [], total: 0 }

// ──────────────────────────────────────────────────────────────
// Setup
// ──────────────────────────────────────────────────────────────

beforeEach(() => {
  vi.clearAllMocks()
  // Default: reseller mode with a running gateway (source becomes non-null).
  mockUseConfigStore.mockReturnValue('reseller')
  mockGatewayState.status = { running: true, url: 'http://localhost:3000' }
  mockGatewayState.adminToken = 'tok-admin'
  // Default list resolves with one item.
  mockSource.list.mockResolvedValue({ items: [SAMPLE_REDEMPTION], total: 1 })
  mockSource.create.mockResolvedValue([SAMPLE_REDEMPTION])
  mockSource.delete.mockResolvedValue(undefined)
  mockSource.deleteInvalid.mockResolvedValue(undefined)
})

// ──────────────────────────────────────────────────────────────
// Tests
// ──────────────────────────────────────────────────────────────

describe('GatewayRedemptionPage', () => {
  it('shows no-source notice when source is null (personal mode, gateway stopped)', () => {
    mockUseConfigStore.mockReturnValue('personal')
    mockGatewayState.status = null
    mockGatewayState.adminToken = null

    render(<GatewayRedemptionPage />)

    // The i18n stub returns the key itself: 'gateway.status.stopped'.
    // The AlertCircle + <p> div is the "no source" branch.
    expect(screen.getByText('gateway.status.stopped')).toBeInTheDocument()
  })

  it('renders redemption list when source resolves successfully', async () => {
    render(<GatewayRedemptionPage />)

    // Wait for the async load to settle.
    expect(await screen.findByText('batch-alpha')).toBeInTheDocument()
    // Masked key: first 6 chars + bullets.
    expect(screen.getByText(/ALPHA-/)).toBeInTheDocument()
    // Quota and count/used columns.
    expect(screen.getByText('500')).toBeInTheDocument()
    expect(screen.getByText('10 / 3')).toBeInTheDocument()
  })

  it('renders empty-state row when source returns no items', async () => {
    mockSource.list.mockResolvedValue(EMPTY_PAGE)

    render(<GatewayRedemptionPage />)

    // The empty-state <td> text is "▪ No redemptions" — match the fallback substring.
    expect(await screen.findByText(/No redemptions/)).toBeInTheDocument()
  })

  it('renders error banner when source.list rejects', async () => {
    mockSource.list.mockRejectedValue(new Error('hub: HTTP 502 Bad Gateway'))

    render(<GatewayRedemptionPage />)

    await waitFor(() => {
      expect(screen.getByText(/HTTP 502 Bad Gateway/)).toBeInTheDocument()
    })
  })

  it('surfaces error via banner when source.create rejects', async () => {
    // First load resolves with no items so the table is empty.
    mockSource.list.mockResolvedValue(EMPTY_PAGE)
    mockSource.create.mockRejectedValue(new Error('create: quota exceeded'))

    render(<GatewayRedemptionPage />)

    // Wait for initial load (empty-state has "▪ No redemptions" prefix).
    await screen.findByText(/No redemptions/)

    // Open the create modal.
    fireEvent.click(screen.getByText('Create'))

    // Fill in the name field (the Save button is disabled until a name exists).
    const nameInput = screen.getByPlaceholderText('redemption-batch-01')
    fireEvent.change(nameInput, { target: { value: 'fail-batch' } })

    // Submit the form.
    fireEvent.click(screen.getByText('settings.save'))

    await waitFor(() => {
      expect(screen.getByText(/quota exceeded/)).toBeInTheDocument()
    })
  })
})
