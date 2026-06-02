import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'

// i18n stub — returns fallback string when provided, or the key itself.
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

// Mock the gateway API client factory so tests control network calls without
// spinning up a real HTTP server.
const mockGetUsers = vi.fn()
const mockSearchUsers = vi.fn()
const mockCreateUser = vi.fn()
const mockUpdateUser = vi.fn()
const mockDeleteUser = vi.fn()
const mockManageUser = vi.fn()

vi.mock('../lib/gateway-api', () => ({
  createGatewayClient: (_url: string, _token: string) => ({
    getUsers: (...a: unknown[]) => mockGetUsers(...a),
    searchUsers: (...a: unknown[]) => mockSearchUsers(...a),
    createUser: (...a: unknown[]) => mockCreateUser(...a),
    updateUser: (...a: unknown[]) => mockUpdateUser(...a),
    deleteUser: (...a: unknown[]) => mockDeleteUser(...a),
    manageUser: (...a: unknown[]) => mockManageUser(...a),
  }),
}))

// Use the real Zustand store — control state via .setState() so the Zustand
// selector path (the page calls useGatewayStore() with no selector) works.
import { GatewayUserPage } from './GatewayUserPage'
import { useGatewayStore } from '../stores/gatewayStore'

// Default gateway state: server running with a valid admin token.
const runningState = {
  status: { running: true, url: 'http://127.0.0.1:3000', port: 3000, uptime: 0, version: '', binaryOk: true },
  adminToken: 'test-admin-token',
  pollingHandle: null,
}

const stoppedState = {
  status: { running: false, url: '', port: 0, uptime: 0, version: '', binaryOk: false },
  adminToken: null,
  pollingHandle: null,
}

// A minimal valid GatewayUser fixture.
function makeUser(overrides: Partial<{
  id: number; username: string; display_name: string; role: number; status: number;
  quota: number; used_quota: number; email: string; group: string; request_count: number;
  aff_code: string; created_time: number;
}> = {}) {
  return {
    id: 1, username: 'alice', display_name: 'Alice', role: 1, status: 1,
    quota: 1000, used_quota: 100, email: 'alice@example.com', group: 'default',
    request_count: 42, aff_code: '', created_time: 0,
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  useGatewayStore.setState(runningState as any)
  mockGetUsers.mockResolvedValue({ data: [], total: 0, success: true, message: '' })
})

describe('GatewayUserPage', () => {
  it('shows stopped notice when gateway is not running', () => {
    useGatewayStore.setState(stoppedState as any)
    render(<GatewayUserPage />)
    // Page renders the stopped-state sentinel instead of the user table.
    expect(screen.getByText('gateway.status.stopped')).toBeInTheDocument()
  })

  it('renders empty state when no users are returned', async () => {
    mockGetUsers.mockResolvedValue({ data: [], total: 0, success: true, message: '' })
    render(<GatewayUserPage />)
    // The td contains "▪ gateway.noUsers" — match with a partial regex.
    await waitFor(() => {
      expect(screen.getByText(/gateway\.noUsers/)).toBeInTheDocument()
    })
  })

  it('renders user rows after successful fetch', async () => {
    const users = [
      makeUser({ id: 1, username: 'alice', display_name: 'Alice', role: 1, status: 1 }),
      makeUser({ id: 2, username: 'bob', display_name: '', role: 10, email: '', status: 2 }),
    ]
    mockGetUsers.mockResolvedValue({ data: users, total: 2, success: true, message: '' })
    render(<GatewayUserPage />)
    await waitFor(() => {
      expect(screen.getByText('alice')).toBeInTheDocument()
      expect(screen.getByText('bob')).toBeInTheDocument()
    })
    // Role labels should render.
    expect(screen.getByText('User')).toBeInTheDocument()
    expect(screen.getByText('Admin')).toBeInTheDocument()
  })

  it('surfaces fetch error in the error banner', async () => {
    mockGetUsers.mockRejectedValue(new Error('Gateway API GET /api/user/ → 503: Service Unavailable'))
    render(<GatewayUserPage />)
    await waitFor(() => {
      expect(screen.getByText(/503/)).toBeInTheDocument()
    })
  })

  it('shows near-limit badge when used_quota / quota ≥ 0.9', async () => {
    const users = [
      makeUser({ id: 1, username: 'heavyuser', quota: 1000, used_quota: 950 }),
    ]
    mockGetUsers.mockResolvedValue({ data: users, total: 1, success: true, message: '' })
    render(<GatewayUserPage />)
    // The badge text is provided via t() fallback.
    await waitFor(() => {
      expect(screen.getByText('即将耗尽')).toBeInTheDocument()
    })
  })

  it('does NOT show near-limit badge when quota is 0 (unlimited)', async () => {
    const users = [makeUser({ id: 1, username: 'unlimiteduser', quota: 0, used_quota: 999 })]
    mockGetUsers.mockResolvedValue({ data: users, total: 1, success: true, message: '' })
    render(<GatewayUserPage />)
    await waitFor(() => {
      expect(screen.getByText('unlimiteduser')).toBeInTheDocument()
    })
    expect(screen.queryByText('即将耗尽')).not.toBeInTheDocument()
  })

  it('opens create-user modal and calls createUser on save (success path)', async () => {
    mockGetUsers.mockResolvedValue({ data: [], total: 0, success: true, message: '' })
    mockCreateUser.mockResolvedValue({ success: true, message: '', data: makeUser({ id: 99, username: 'newuser' }) })

    render(<GatewayUserPage />)

    // Wait for the page to settle (empty state).
    await waitFor(() => expect(screen.getByText(/gateway\.noUsers/)).toBeInTheDocument())

    // Click "Create User" button.
    const createBtn = screen.getByText('gateway.createUser')
    fireEvent.click(createBtn)

    // Modal should open; type in the username field (first empty input).
    const usernameInputs = screen.getAllByRole('textbox')
    fireEvent.change(usernameInputs[0], { target: { value: 'newuser' } })

    // Click Save.
    const saveBtn = screen.getByText('gateway.save')
    fireEvent.click(saveBtn)

    await waitFor(() => {
      expect(mockCreateUser).toHaveBeenCalledTimes(1)
    })
    // Modal should close after successful save (no more save button).
    await waitFor(() => {
      expect(screen.queryByText('gateway.save')).not.toBeInTheDocument()
    })
  })

  it('surfaces error when createUser fails', async () => {
    mockGetUsers.mockResolvedValue({ data: [], total: 0, success: true, message: '' })
    mockCreateUser.mockRejectedValue(new Error('create user: HTTP 422'))

    render(<GatewayUserPage />)
    await waitFor(() => expect(screen.getByText(/gateway\.noUsers/)).toBeInTheDocument())

    // Open modal.
    fireEvent.click(screen.getByText('gateway.createUser'))

    // Click save to trigger the create (and its failure).
    fireEvent.click(screen.getByText('gateway.save'))

    await waitFor(() => {
      expect(screen.getByText(/HTTP 422/)).toBeInTheDocument()
    })
  })

  it('calls deleteUser when delete is confirmed', async () => {
    const users = [makeUser({ id: 7, username: 'todelete' })]
    mockGetUsers.mockResolvedValue({ data: users, total: 1, success: true, message: '' })
    mockDeleteUser.mockResolvedValue({ success: true, message: '' })

    render(<GatewayUserPage />)
    await waitFor(() => expect(screen.getByText('todelete')).toBeInTheDocument())

    // Find and click the delete (trash) button — the title comes from i18n stub.
    const deleteBtn = screen.getByTitle('gateway.delete')
    fireEvent.click(deleteBtn)

    // ConfirmModal is now open; click Confirm.
    const confirmBtn = screen.getByText('Confirm')
    fireEvent.click(confirmBtn)

    await waitFor(() => {
      expect(mockDeleteUser).toHaveBeenCalledWith(7)
    })
    // User row should vanish from the list.
    await waitFor(() => {
      expect(screen.queryByText('todelete')).not.toBeInTheDocument()
    })
  })

  it('surfaces error when deleteUser fails', async () => {
    const users = [makeUser({ id: 8, username: 'faildelete' })]
    mockGetUsers.mockResolvedValue({ data: users, total: 1, success: true, message: '' })
    mockDeleteUser.mockRejectedValue(new Error('delete user: HTTP 500'))

    render(<GatewayUserPage />)
    await waitFor(() => expect(screen.getByText('faildelete')).toBeInTheDocument())

    fireEvent.click(screen.getByTitle('gateway.delete'))
    fireEvent.click(screen.getByText('Confirm'))

    await waitFor(() => {
      expect(screen.getByText(/HTTP 500/)).toBeInTheDocument()
    })
  })

  it('toggles user status via manageUser when the status badge is clicked', async () => {
    const users = [makeUser({ id: 3, username: 'charlie', status: 1 })]
    mockGetUsers.mockResolvedValue({ data: users, total: 1, success: true, message: '' })
    mockManageUser.mockResolvedValue({ success: true, message: '' })

    render(<GatewayUserPage />)
    await waitFor(() => expect(screen.getByText('charlie')).toBeInTheDocument())

    // The status cell wraps StatusBadge in a button — clicking it triggers toggle.
    const statusToggleBtn = screen.getByText('charlie')
      .closest('tr')
      ?.querySelector('button[class*="transition-opacity"]') as HTMLElement
    expect(statusToggleBtn).toBeTruthy()
    fireEvent.click(statusToggleBtn)

    await waitFor(() => {
      expect(mockManageUser).toHaveBeenCalledWith({ id: 3, action: 'disable' })
    })
  })
})
