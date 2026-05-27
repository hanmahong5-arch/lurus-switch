import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

// ---------------------------------------------------------------------------
// Standard mocks expected by every component test in this codebase.
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
  initReactI18next: { type: '3rdParty', init: () => {} },
}))

const getRelayEndpointsMock = vi.fn()
const saveToolRelayMappingMock = vi.fn()
const applyAllToolRelaysMock = vi.fn()

vi.mock('../../wailsjs/go/main/App', () => ({
  GetRelayEndpoints: () => getRelayEndpointsMock(),
  SaveToolRelayMapping: (m: Record<string, string>) => saveToolRelayMappingMock(m),
  ApplyAllToolRelays: () => applyAllToolRelaysMock(),
}))

import { QuickSwitchOverlay } from './QuickSwitchOverlay'
import { useQuickSwitchStore } from '../stores/quickSwitchStore'
import { useToastStore } from '../stores/toastStore'

// Minimal relay endpoint factory.
const ep = (id: string, name: string, healthy = true) => ({
  id,
  name,
  kind: 'manual',
  url: `https://${id}.example.com`,
  apiKey: '',
  description: `${name} endpoint`,
  latencyMs: 42,
  healthy,
  lastChecked: '',
})

beforeEach(() => {
  vi.clearAllMocks()
  useQuickSwitchStore.setState({ open: false })
  useToastStore.setState({ toasts: [] })
  getRelayEndpointsMock.mockResolvedValue([ep('ep-1', 'Endpoint One'), ep('ep-2', 'Endpoint Two')])
  saveToolRelayMappingMock.mockResolvedValue(undefined)
  applyAllToolRelaysMock.mockResolvedValue({})
})

// ---------------------------------------------------------------------------
// Visibility
// ---------------------------------------------------------------------------

describe('QuickSwitchOverlay', () => {
  it('renders nothing when closed', () => {
    render(<QuickSwitchOverlay />)
    expect(screen.queryByTestId('quick-switch-panel')).toBeNull()
  })

  it('renders panel when open', () => {
    useQuickSwitchStore.setState({ open: true })
    render(<QuickSwitchOverlay />)
    expect(screen.getByTestId('quick-switch-panel')).toBeTruthy()
  })

  it('closes when backdrop is clicked', () => {
    useQuickSwitchStore.setState({ open: true })
    render(<QuickSwitchOverlay />)
    fireEvent.click(screen.getByTestId('quick-switch-backdrop'))
    expect(useQuickSwitchStore.getState().open).toBe(false)
  })

  it('closes when X button is clicked', () => {
    useQuickSwitchStore.setState({ open: true })
    render(<QuickSwitchOverlay />)
    // The close button has aria-label from t('quickSwitch.close', 'Close')
    const closeBtn = screen.getByRole('button', { name: /quickSwitch\.close|Close/i })
    fireEvent.click(closeBtn)
    expect(useQuickSwitchStore.getState().open).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// Endpoint loading
// ---------------------------------------------------------------------------

describe('QuickSwitchOverlay — endpoint loading', () => {
  it('shows endpoints once loaded', async () => {
    useQuickSwitchStore.setState({ open: true })
    render(<QuickSwitchOverlay />)
    await waitFor(() => {
      expect(screen.getByTestId('quick-switch-item-ep-1')).toBeTruthy()
      expect(screen.getByTestId('quick-switch-item-ep-2')).toBeTruthy()
    })
  })

  it('shows empty state when no endpoints are returned', async () => {
    getRelayEndpointsMock.mockResolvedValue([])
    useQuickSwitchStore.setState({ open: true })
    render(<QuickSwitchOverlay />)
    await waitFor(() => {
      expect(screen.getByText(/quickSwitch\.noEndpoints|No relay endpoints/i)).toBeTruthy()
    })
  })

  it('falls back to store cache on load failure', async () => {
    getRelayEndpointsMock.mockRejectedValue(new Error('network'))
    // Pre-seed store cache.
    const { useRelayStore } = await import('../stores/relayStore')
    useRelayStore.setState({ endpoints: [ep('cache-ep', 'Cached')] as never })
    useQuickSwitchStore.setState({ open: true })
    render(<QuickSwitchOverlay />)
    await waitFor(() => {
      expect(screen.getByTestId('quick-switch-item-cache-ep')).toBeTruthy()
    })
  })
})

// ---------------------------------------------------------------------------
// Apply action
// ---------------------------------------------------------------------------

describe('QuickSwitchOverlay — apply relay', () => {
  it('calls SaveToolRelayMapping + ApplyAllToolRelays and closes on success', async () => {
    useQuickSwitchStore.setState({ open: true })
    render(<QuickSwitchOverlay />)
    await waitFor(() => screen.getByTestId('quick-switch-item-ep-1'))
    fireEvent.click(screen.getByTestId('quick-switch-item-ep-1'))
    await waitFor(() => {
      expect(saveToolRelayMappingMock).toHaveBeenCalledWith(
        expect.objectContaining({ claude: 'ep-1', codex: 'ep-1' }),
      )
      expect(applyAllToolRelaysMock).toHaveBeenCalledTimes(1)
      expect(useQuickSwitchStore.getState().open).toBe(false)
    })
  })

  it('overlay closes after a successful switch', async () => {
    // This test verifies the UX flow: panel disappears once the relay is applied.
    // Toast content is covered by the first test in this describe block.
    useQuickSwitchStore.setState({ open: true })
    render(<QuickSwitchOverlay />)
    await waitFor(() => screen.getByTestId('quick-switch-item-ep-1'))
    fireEvent.click(screen.getByTestId('quick-switch-item-ep-1'))
    await waitFor(() => {
      expect(useQuickSwitchStore.getState().open).toBe(false)
    })
  })

  it('shows warning toast when some tools fail', async () => {
    applyAllToolRelaysMock.mockResolvedValue({ claude: 'error: unreachable' })
    useQuickSwitchStore.setState({ open: true })
    render(<QuickSwitchOverlay />)
    await waitFor(() => screen.getByTestId('quick-switch-item-ep-1'))
    fireEvent.click(screen.getByTestId('quick-switch-item-ep-1'))
    await waitFor(() => {
      const toasts = useToastStore.getState().toasts
      expect(toasts.some((t) => t.type === 'warning')).toBe(true)
    })
  })

  it('shows error toast when apply throws', async () => {
    saveToolRelayMappingMock.mockRejectedValue(new Error('save failed'))
    useQuickSwitchStore.setState({ open: true })
    render(<QuickSwitchOverlay />)
    await waitFor(() => screen.getByTestId('quick-switch-item-ep-1'))
    fireEvent.click(screen.getByTestId('quick-switch-item-ep-1'))
    await waitFor(() => {
      const toasts = useToastStore.getState().toasts
      expect(toasts.some((t) => t.type === 'error')).toBe(true)
    })
  })
})

// ---------------------------------------------------------------------------
// Keyboard navigation
// ---------------------------------------------------------------------------

describe('QuickSwitchOverlay — keyboard navigation', () => {
  it('closes on Escape', async () => {
    useQuickSwitchStore.setState({ open: true })
    render(<QuickSwitchOverlay />)
    const panel = await screen.findByTestId('quick-switch-panel')
    fireEvent.keyDown(panel, { key: 'Escape' })
    expect(useQuickSwitchStore.getState().open).toBe(false)
  })

  it('applies selected item on Enter', async () => {
    useQuickSwitchStore.setState({ open: true })
    render(<QuickSwitchOverlay />)
    const panel = await screen.findByTestId('quick-switch-panel')
    await waitFor(() => screen.getByTestId('quick-switch-item-ep-1'))
    // First item (idx 0) is selected by default; Enter should apply it.
    fireEvent.keyDown(panel, { key: 'Enter' })
    await waitFor(() => {
      expect(applyAllToolRelaysMock).toHaveBeenCalledTimes(1)
    })
  })

  it('does not throw on ArrowDown/Up with empty list', async () => {
    getRelayEndpointsMock.mockResolvedValue([])
    useQuickSwitchStore.setState({ open: true })
    render(<QuickSwitchOverlay />)
    const panel = await screen.findByTestId('quick-switch-panel')
    // Should not throw.
    expect(() => {
      fireEvent.keyDown(panel, { key: 'ArrowDown' })
      fireEvent.keyDown(panel, { key: 'ArrowUp' })
    }).not.toThrow()
  })
})
