import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

// i18n stub — returns the fallback string when provided so assertions can
// match user-facing copy without depending on locale resolution.
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

// ── TokenSource mock ──────────────────────────────────────────────────────────
// We intercept makeTokenSource so each test can control what the source returns
// without hitting the real Wails bindings.
const mockList = vi.fn()
const mockCreate = vi.fn()
const mockUpdate = vi.fn()
const mockDelete = vi.fn()
const mockBatchDelete = vi.fn()

const mockSource = {
  kind: 'hub' as const,
  capabilities: { search: false },
  list: (...args: unknown[]) => mockList(...args),
  create: (...args: unknown[]) => mockCreate(...args),
  update: (...args: unknown[]) => mockUpdate(...args),
  delete: (...args: unknown[]) => mockDelete(...args),
  batchDelete: (...args: unknown[]) => mockBatchDelete(...args),
}

// makeTokenSourceEnabled controls whether the source factory returns a real
// source (true) or null — null triggers the "gateway not running" splash.
let makeTokenSourceEnabled = true

vi.mock('../lib/tokenSource', () => ({
  makeTokenSource: () => (makeTokenSourceEnabled ? mockSource : null),
}))

// ── Wails stub — minimal surface needed so the module resolves ────────────────
vi.mock('../../wailsjs/go/main/App', () => ({
  HubListTokens: vi.fn(),
  HubAddToken: vi.fn(),
  HubUpdateToken: vi.fn(),
  HubDeleteToken: vi.fn(),
  HubDeleteTokenBatch: vi.fn(),
}))

vi.mock('../../wailsjs/go/models', () => ({
  admin: {},
}))

// ── Store mocks ───────────────────────────────────────────────────────────────
let gatewayState = {
  status: { running: true, url: 'http://127.0.0.1:3000', port: 3000, uptime: 10, version: '1.0', binaryOk: true },
  adminToken: 'test-admin-token',
}

vi.mock('../stores/gatewayStore', () => ({
  useGatewayStore: (selector?: (s: typeof gatewayState) => unknown) => {
    if (typeof selector === 'function') return selector(gatewayState)
    return gatewayState
  },
}))

const mockUseConfigStore = vi.fn()
vi.mock('../stores/configStore', () => ({
  useConfigStore: <T,>(selector: (s: { appMode: string }) => T) =>
    selector({ appMode: mockUseConfigStore() }),
}))

// ── Clipboard stub ────────────────────────────────────────────────────────────
const mockClipboardWriteText = vi.fn().mockResolvedValue(undefined)

import { GatewayTokenPage } from './GatewayTokenPage'

// ── Fixtures ──────────────────────────────────────────────────────────────────
function makeToken(overrides: Partial<{
  id: number; name: string; key: string; status: number;
  quota: number; used_quota: number; expired_time: number;
  unlimited_quota: boolean; created_time: number; remain_quota: number;
  model_limits: string; subnet: string; group: string;
}> = {}) {
  return {
    id: 1,
    name: 'Test Token',
    key: 'sk-abcdefghijklmnop',
    status: 1,
    quota: 500000,
    used_quota: 12345,
    expired_time: 0,
    unlimited_quota: true,
    created_time: 1700000000,
    remain_quota: 500000,
    model_limits: '',
    subnet: '',
    group: '',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  makeTokenSourceEnabled = true
  mockUseConfigStore.mockReturnValue('reseller')
  gatewayState = {
    status: { running: true, url: 'http://127.0.0.1:3000', port: 3000, uptime: 10, version: '1.0', binaryOk: true },
    adminToken: 'test-admin-token',
  }
  // Install clipboard mock fresh each test so it's definitely active
  Object.defineProperty(navigator, 'clipboard', {
    value: { writeText: mockClipboardWriteText },
    writable: true,
    configurable: true,
  })
})

describe('GatewayTokenPage', () => {
  // ── Source-not-available state ──────────────────────────────────────────────
  describe('when no source is available', () => {
    it('shows hub-not-configured notice in reseller mode', () => {
      makeTokenSourceEnabled = false
      render(<GatewayTokenPage />)
      expect(screen.getByText(/请先在「设置」中配置/)).toBeInTheDocument()
    })

    it('shows gateway stopped notice in personal mode', () => {
      makeTokenSourceEnabled = false
      mockUseConfigStore.mockReturnValue('personal')
      render(<GatewayTokenPage />)
      expect(screen.getByText('gateway.status.stopped')).toBeInTheDocument()
    })
  })

  // ── Initial load ────────────────────────────────────────────────────────────
  describe('initial load', () => {
    it('renders the tokens table heading and create button', async () => {
      mockList.mockResolvedValue({ items: [], total: 0 })
      render(<GatewayTokenPage />)
      expect(screen.getByText('gateway.tokens')).toBeInTheDocument()
      expect(screen.getByText('gateway.createToken')).toBeInTheDocument()
    })

    it('shows empty-state row when list returns no tokens', async () => {
      mockList.mockResolvedValue({ items: [], total: 0 })
      render(<GatewayTokenPage />)
      await waitFor(() => {
        expect(screen.getByText(/gateway\.noTokens/)).toBeInTheDocument()
      })
    })

    it('renders token rows when list returns data', async () => {
      mockList.mockResolvedValue({
        items: [
          makeToken({ id: 1, name: 'Alpha Token', key: 'sk-alpha1234567890', unlimited_quota: true }),
          makeToken({ id: 2, name: 'Beta Token', key: 'sk-beta9876543210', unlimited_quota: false, quota: 100000, used_quota: 5000 }),
        ],
        total: 2,
      })
      render(<GatewayTokenPage />)
      await waitFor(() => {
        expect(screen.getByText('Alpha Token')).toBeInTheDocument()
        expect(screen.getByText('Beta Token')).toBeInTheDocument()
      })
    })

    it('masks the token key to first 8 chars + bullets', async () => {
      mockList.mockResolvedValue({
        items: [makeToken({ key: 'sk-abcdefghijklmnop' })],
        total: 1,
      })
      render(<GatewayTokenPage />)
      // maskKey: 'sk-abcde' are the first 8 chars of 'sk-abcdefghijklmnop'
      await waitFor(() => {
        expect(screen.getByText(/sk-abcde/)).toBeInTheDocument()
      })
    })

    it('shows "∞" for unlimited quota tokens', async () => {
      mockList.mockResolvedValue({
        items: [makeToken({ unlimited_quota: true })],
        total: 1,
      })
      render(<GatewayTokenPage />)
      await waitFor(() => {
        expect(screen.getByText('∞')).toBeInTheDocument()
      })
    })

    it('shows used/quota fraction for limited tokens', async () => {
      mockList.mockResolvedValue({
        items: [makeToken({ unlimited_quota: false, used_quota: 100, quota: 500 })],
        total: 1,
      })
      render(<GatewayTokenPage />)
      await waitFor(() => {
        expect(screen.getByText('100 / 500')).toBeInTheDocument()
      })
    })

    it('shows "never expires" label for expired_time = 0', async () => {
      mockList.mockResolvedValue({
        items: [makeToken({ expired_time: 0 })],
        total: 1,
      })
      render(<GatewayTokenPage />)
      await waitFor(() => {
        expect(screen.getByText('gateway.tokenNeverExpires')).toBeInTheDocument()
      })
    })
  })

  // ── Error state ─────────────────────────────────────────────────────────────
  describe('error handling', () => {
    it('shows error banner when list throws', async () => {
      mockList.mockRejectedValue(new Error('HTTP 502 Bad Gateway'))
      render(<GatewayTokenPage />)
      await waitFor(() => {
        expect(screen.getByText(/HTTP 502 Bad Gateway/)).toBeInTheDocument()
      })
    })

    it('no error banner on successful load', async () => {
      mockList.mockResolvedValue({ items: [], total: 0 })
      render(<GatewayTokenPage />)
      await waitFor(() => {
        // Error banner prefix "▸" should not appear
        const errorEl = screen.queryByText(/▸/)
        expect(errorEl).not.toBeInTheDocument()
      })
    })
  })

  // ── Create token (success path) ─────────────────────────────────────────────
  describe('create token', () => {
    it('opens create modal when the Create button is clicked', async () => {
      mockList.mockResolvedValue({ items: [], total: 0 })
      const user = userEvent.setup()
      render(<GatewayTokenPage />)
      await waitFor(() => expect(screen.getByText(/gateway\.noTokens/)).toBeInTheDocument())

      await user.click(screen.getByText('gateway.createToken'))
      // Modal should be visible — save button appears in the footer
      expect(screen.getByText('gateway.save')).toBeInTheDocument()
    })

    it('calls source.create and reloads on save', async () => {
      mockList
        .mockResolvedValueOnce({ items: [], total: 0 })
        .mockResolvedValueOnce({
          items: [makeToken({ id: 10, name: 'New Token' })],
          total: 1,
        })
      mockCreate.mockResolvedValue(undefined)

      const user = userEvent.setup()
      render(<GatewayTokenPage />)
      await waitFor(() => expect(screen.getByText(/gateway\.noTokens/)).toBeInTheDocument())

      await user.click(screen.getByText('gateway.createToken'))
      await user.click(screen.getByText('gateway.save'))

      await waitFor(() => {
        expect(mockCreate).toHaveBeenCalledTimes(1)
        expect(screen.getByText('New Token')).toBeInTheDocument()
      })
    })

    it('shows error banner when source.create throws', async () => {
      mockList.mockResolvedValue({ items: [], total: 0 })
      mockCreate.mockRejectedValue(new Error('create failed: 403'))

      const user = userEvent.setup()
      render(<GatewayTokenPage />)
      await waitFor(() => expect(screen.getByText(/gateway\.noTokens/)).toBeInTheDocument())

      await user.click(screen.getByText('gateway.createToken'))
      await user.click(screen.getByText('gateway.save'))

      await waitFor(() => {
        expect(screen.getByText(/create failed: 403/)).toBeInTheDocument()
      })
    })

    it('modal is closed when Cancel is clicked', async () => {
      mockList.mockResolvedValue({ items: [], total: 0 })
      const user = userEvent.setup()
      render(<GatewayTokenPage />)
      await waitFor(() => expect(screen.getByText(/gateway\.noTokens/)).toBeInTheDocument())

      await user.click(screen.getByText('gateway.createToken'))
      expect(screen.getByText('gateway.save')).toBeInTheDocument()

      await user.click(screen.getByText('gateway.cancel'))
      // AnimatePresence delays unmount — wait for save button to leave the DOM
      await waitFor(() => {
        expect(screen.queryByText('gateway.save')).not.toBeInTheDocument()
      })
    })
  })

  // ── Delete token (success + failure) ────────────────────────────────────────
  describe('delete token', () => {
    it('removes the token row after confirmed delete', async () => {
      const token = makeToken({ id: 42, name: 'ToDelete Token' })
      mockList.mockResolvedValue({ items: [token], total: 1 })
      mockDelete.mockResolvedValue(undefined)

      const user = userEvent.setup()
      render(<GatewayTokenPage />)
      await waitFor(() => expect(screen.getByText('ToDelete Token')).toBeInTheDocument())

      // Open delete confirm
      const deleteBtn = screen.getByTitle('gateway.delete')
      await user.click(deleteBtn)

      // ConfirmModal appears — click its "Confirm" button
      await waitFor(() => expect(screen.getByText('gateway.deleteConfirmTitle')).toBeInTheDocument())
      fireEvent.click(screen.getByText('Confirm'))

      await waitFor(() => {
        expect(mockDelete).toHaveBeenCalledWith(42)
        expect(screen.queryByText('ToDelete Token')).not.toBeInTheDocument()
      })
    })

    it('shows error banner when delete throws', async () => {
      const token = makeToken({ id: 99, name: 'ErrToken' })
      mockList.mockResolvedValue({ items: [token], total: 1 })
      mockDelete.mockRejectedValue(new Error('delete: forbidden'))

      const user = userEvent.setup()
      render(<GatewayTokenPage />)
      await waitFor(() => expect(screen.getByText('ErrToken')).toBeInTheDocument())

      const deleteBtn = screen.getByTitle('gateway.delete')
      await user.click(deleteBtn)

      await waitFor(() => expect(screen.getByText('gateway.deleteConfirmTitle')).toBeInTheDocument())
      fireEvent.click(screen.getByText('Confirm'))

      await waitFor(() => {
        expect(screen.getByText(/delete: forbidden/)).toBeInTheDocument()
      })
    })

    it('does not call delete when Cancel is clicked in confirm modal', async () => {
      const token = makeToken({ id: 77, name: 'CancelToken' })
      mockList.mockResolvedValue({ items: [token], total: 1 })

      const user = userEvent.setup()
      render(<GatewayTokenPage />)
      await waitFor(() => expect(screen.getByText('CancelToken')).toBeInTheDocument())

      const deleteBtn = screen.getByTitle('gateway.delete')
      await user.click(deleteBtn)

      await waitFor(() => expect(screen.getByText('gateway.deleteConfirmTitle')).toBeInTheDocument())
      fireEvent.click(screen.getByText('Cancel'))

      expect(mockDelete).not.toHaveBeenCalled()
      expect(screen.queryByText('CancelToken')).toBeInTheDocument()
    })
  })

  // ── Toggle status mutation ─────────────────────────────────────────────────
  describe('toggle token status', () => {
    it('flips the status badge from Enabled to Disabled when clicked', async () => {
      const token = makeToken({ id: 5, name: 'StatusToken', status: 1 })
      mockList.mockResolvedValue({ items: [token], total: 1 })
      mockUpdate.mockResolvedValue(undefined)

      const user = userEvent.setup()
      render(<GatewayTokenPage />)
      await waitFor(() => expect(screen.getByText('StatusToken')).toBeInTheDocument())

      // StatusBadge renders "Enabled" (capital E) wrapped in a button
      const enabledBadge = screen.getByText('Enabled')
      await user.click(enabledBadge.closest('button')!)

      await waitFor(() => {
        expect(mockUpdate).toHaveBeenCalledWith({ id: 5, status: 2 })
        expect(screen.getByText('Disabled')).toBeInTheDocument()
      })
    })

    it('shows error banner when status toggle throws', async () => {
      const token = makeToken({ id: 7, name: 'FailStatusToken', status: 1 })
      mockList.mockResolvedValue({ items: [token], total: 1 })
      mockUpdate.mockRejectedValue(new Error('toggle: 500'))

      const user = userEvent.setup()
      render(<GatewayTokenPage />)
      await waitFor(() => expect(screen.getByText('FailStatusToken')).toBeInTheDocument())

      const enabledBadge = screen.getByText('Enabled')
      await user.click(enabledBadge.closest('button')!)

      await waitFor(() => {
        expect(screen.getByText(/toggle: 500/)).toBeInTheDocument()
      })
    })
  })

  // ── Copy key ────────────────────────────────────────────────────────────────
  describe('copy key', () => {
    it('calls clipboard.writeText with the full unmasked key', async () => {
      const token = makeToken({ key: 'sk-fullkeytest1234567890' })
      mockList.mockResolvedValue({ items: [token], total: 1 })

      render(<GatewayTokenPage />)
      // maskKey returns first 8 chars: 'sk-fullk' + bullets
      await waitFor(() => expect(screen.getByText(/sk-fullk/)).toBeInTheDocument())

      const copyBtn = screen.getByTitle('gateway.copyKey')
      // Use fireEvent to avoid userEvent's clipboard interception
      fireEvent.click(copyBtn)

      await waitFor(() => {
        expect(mockClipboardWriteText).toHaveBeenCalledWith('sk-fullkeytest1234567890')
      })
    })
  })

  // ── Row selection (batch) ────────────────────────────────────────────────────
  describe('batch selection', () => {
    it('shows batch delete button when rows are selected', async () => {
      const tokens = [
        makeToken({ id: 1, name: 'Token One' }),
        makeToken({ id: 2, name: 'Token Two' }),
      ]
      mockList.mockResolvedValue({ items: tokens, total: 2 })

      const user = userEvent.setup()
      render(<GatewayTokenPage />)
      await waitFor(() => expect(screen.getByText('Token One')).toBeInTheDocument())

      // Check one row checkbox — first td in first data row
      const checkboxes = screen.getAllByRole('checkbox')
      // checkboxes[0] is the "select all" header, checkboxes[1] is first row
      await user.click(checkboxes[1])

      await waitFor(() => {
        expect(screen.getByText(/gateway.batchDelete/)).toBeInTheDocument()
      })
    })

    it('calls batchDelete with selected ids', async () => {
      const tokens = [
        makeToken({ id: 11, name: 'BatchA' }),
        makeToken({ id: 22, name: 'BatchB' }),
      ]
      mockList
        .mockResolvedValueOnce({ items: tokens, total: 2 })
        .mockResolvedValueOnce({ items: [], total: 0 })
      mockBatchDelete.mockResolvedValue(undefined)

      const user = userEvent.setup()
      render(<GatewayTokenPage />)
      await waitFor(() => expect(screen.getByText('BatchA')).toBeInTheDocument())

      // Select all via header checkbox
      const checkboxes = screen.getAllByRole('checkbox')
      await user.click(checkboxes[0]) // select-all

      await waitFor(() => expect(screen.getByText(/gateway.batchDelete/)).toBeInTheDocument())
      await user.click(screen.getByText(/gateway.batchDelete/))

      await waitFor(() => {
        expect(mockBatchDelete).toHaveBeenCalledTimes(1)
        const calledWith = mockBatchDelete.mock.calls[0][0] as number[]
        expect(calledWith).toContain(11)
        expect(calledWith).toContain(22)
      })
    })
  })
})
