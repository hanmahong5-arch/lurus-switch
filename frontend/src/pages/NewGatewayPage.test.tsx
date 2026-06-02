import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import React from 'react'

// ---------------------------------------------------------------------------
// i18n stub — fallback wins when provided, key rendered otherwise.
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
// Stub all heavy child pages — we test the tab-routing shell, not their internals.
// ---------------------------------------------------------------------------
vi.mock('./SwitchHubPage', () => ({
  SwitchHubPage: ({ section }: { section: string }) => (
    <div data-testid={`switch-hub-${section}`}>SwitchHubPage:{section}</div>
  ),
}))

vi.mock('./RelayPage', () => ({
  RelayPage: () => <div data-testid="relay-page">RelayPage</div>,
}))

vi.mock('./GatewayDashboardPage', () => ({
  GatewayDashboardPage: () => <div data-testid="gw-dashboard">GatewayDashboardPage</div>,
}))

vi.mock('./GatewayChannelPage', () => ({
  GatewayChannelPage: () => <div data-testid="gw-channels">GatewayChannelPage</div>,
}))

vi.mock('./GatewayTokenPage', () => ({
  GatewayTokenPage: () => <div data-testid="gw-tokens">GatewayTokenPage</div>,
}))

vi.mock('./GatewayModelPage', () => ({
  GatewayModelPage: () => <div data-testid="gw-models">GatewayModelPage</div>,
}))

vi.mock('./GatewayUserPage', () => ({
  GatewayUserPage: () => <div data-testid="gw-users">GatewayUserPage</div>,
}))

vi.mock('./GatewayRedemptionPage', () => ({
  GatewayRedemptionPage: () => <div data-testid="gw-redemptions">GatewayRedemptionPage</div>,
}))

vi.mock('./GatewayLogPage', () => ({
  GatewayLogPage: () => <div data-testid="gw-logs">GatewayLogPage</div>,
}))

vi.mock('./GatewaySubscriptionPage', () => ({
  GatewaySubscriptionPage: () => <div data-testid="gw-subscriptions">GatewaySubscriptionPage</div>,
}))

vi.mock('./GatewayWalletPage', () => ({
  GatewayWalletPage: () => <div data-testid="gw-wallet">GatewayWalletPage</div>,
}))

vi.mock('./GatewaySettingsPage', () => ({
  GatewaySettingsPage: () => <div data-testid="gw-settings">GatewaySettingsPage</div>,
}))

vi.mock('./ToolReleasePage', () => ({
  ToolReleasePage: () => <div data-testid="tool-releases">ToolReleasePage</div>,
}))

vi.mock('./AdminPage', () => ({
  AdminPage: () => <div data-testid="admin-page">AdminPage</div>,
}))

vi.mock('../components/gateway/AuditLogPanel', () => ({
  AuditLogPanel: () => <div data-testid="audit-log">AuditLogPanel</div>,
}))

// GatewayRequiredGuard — renders children directly (gateway "running" from our perspective).
vi.mock('../components/GatewayRequiredGuard', () => ({
  GatewayRequiredGuard: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

// ---------------------------------------------------------------------------
// configStore mock — two controlled knobs: appMode and activeSubTab.
// ---------------------------------------------------------------------------
let mockAppMode = 'personal'
let mockSubTab = 'control'
const mockSetSubTab = vi.fn()

vi.mock('../stores/configStore', () => ({
  useConfigStore: (selector?: (s: unknown) => unknown) => {
    const state = {
      appMode: mockAppMode,
      getSubTab: (_page: string, defaultTab: string) => mockSubTab || defaultTab,
      setSubTab: mockSetSubTab,
    }
    return selector ? selector(state) : state
  },
}))

// ---------------------------------------------------------------------------
// billingStore mock — controlled userInfo for role-gating logic.
// ---------------------------------------------------------------------------
let mockUserInfo: { role?: number } | null = null

vi.mock('../stores/billingStore', () => ({
  useBillingStore: (selector: (s: { userInfo: { role?: number } | null }) => unknown) =>
    selector({ userInfo: mockUserInfo }),
}))

// ---------------------------------------------------------------------------
// Wails bindings — not directly used by NewGatewayPage shell, but imported
// transitively by lucide-react icons and child stubs sometimes; provide a
// permissive no-op default so no import can crash.
// ---------------------------------------------------------------------------
vi.mock('../../wailsjs/go/main/App', () => ({}))
vi.mock('../../wailsjs/go/models', () => ({}))

// ---------------------------------------------------------------------------
// Import under test (after all vi.mock calls)
// ---------------------------------------------------------------------------
import { NewGatewayPage } from './NewGatewayPage'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function renderPage() {
  return render(<NewGatewayPage />)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------
describe('NewGatewayPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockAppMode = 'personal'
    mockSubTab = 'control'
    mockUserInfo = null
  })

  // ── Basic tabs always present ────────────────────────────────────────────

  it('renders all four basic tabs in every mode', () => {
    renderPage()
    // The active tab (control) is rendered in bracket form: [ HOME.GWCONTROL ].
    // Inactive tabs render as plain i18n keys.
    expect(screen.getByText('[ HOME.GWCONTROL ]')).toBeInTheDocument()
    expect(screen.getByText('home.gwUsage')).toBeInTheDocument()
    expect(screen.getByText('home.gwApps')).toBeInTheDocument()
    expect(screen.getByText('nav.relay')).toBeInTheDocument()
  })

  // ── Default content: control tab → SwitchHubPage ─────────────────────────

  it('renders SwitchHubPage for the control tab by default', () => {
    renderPage()
    expect(screen.getByTestId('switch-hub-control')).toBeInTheDocument()
  })

  // ── Personal mode: no admin or root tabs ─────────────────────────────────

  it('hides admin and root tabs in personal mode', () => {
    mockAppMode = 'personal'
    renderPage()
    expect(screen.queryByText('gateway.dashboard')).not.toBeInTheDocument()
    expect(screen.queryByText('gateway.channels')).not.toBeInTheDocument()
    // root tab: t('gateway.system', t('nav.admin')) → renders as 'nav.admin'
    expect(screen.queryByText('nav.admin')).not.toBeInTheDocument()
  })

  // ── Reseller with no role: shows admin tabs, no root tab ─────────────────

  it('shows admin tabs in reseller mode when role is unknown (undefined)', () => {
    mockAppMode = 'reseller'
    mockUserInfo = null  // role is undefined
    renderPage()
    expect(screen.getByText('gateway.dashboard')).toBeInTheDocument()
    expect(screen.getByText('gateway.channels')).toBeInTheDocument()
    expect(screen.getByText('gateway.tokens')).toBeInTheDocument()
    // t('gateway.wallet', '钱包') → stub returns fallback '钱包'
    expect(screen.getByText('钱包')).toBeInTheDocument()
    // Root tab (system) should NOT appear yet — role not confirmed >= 100
    // root tab: t('gateway.system', t('nav.admin')) → renders as 'nav.admin'
    expect(screen.queryByText('nav.admin')).not.toBeInTheDocument()
  })

  // ── Reseller with role 10: admin yes, root no ─────────────────────────────

  it('shows admin tabs but NOT root tab when reseller role = 10', () => {
    mockAppMode = 'reseller'
    mockUserInfo = { role: 10 }
    renderPage()
    expect(screen.getByText('gateway.dashboard')).toBeInTheDocument()
    // root tab: t('gateway.system', t('nav.admin')) → renders as 'nav.admin'
    expect(screen.queryByText('nav.admin')).not.toBeInTheDocument()
  })

  // ── Reseller with role 100: both admin and root tabs ─────────────────────

  it('shows both admin and root tabs when reseller role = 100', () => {
    mockAppMode = 'reseller'
    mockUserInfo = { role: 100 }
    renderPage()
    expect(screen.getByText('gateway.dashboard')).toBeInTheDocument()
    // t('gateway.system', t('nav.admin')) → inner t('nav.admin') returns 'nav.admin',
    // outer t('gateway.system', 'nav.admin') returns fallback 'nav.admin'
    expect(screen.getByText('nav.admin')).toBeInTheDocument()
  })

  // ── Reseller with role < ROLE_ADMIN (e.g. 5): no admin, no root ──────────

  it('hides admin and root tabs when reseller role < 10', () => {
    mockAppMode = 'reseller'
    mockUserInfo = { role: 5 }
    renderPage()
    expect(screen.queryByText('gateway.dashboard')).not.toBeInTheDocument()
    // root tab: t('gateway.system', t('nav.admin')) → renders as 'nav.admin'
    expect(screen.queryByText('nav.admin')).not.toBeInTheDocument()
    // Basic tabs still present — active tab renders in bracket form
    expect(screen.getByText('[ HOME.GWCONTROL ]')).toBeInTheDocument()
  })

  // ── Tab click fires setSubTab ─────────────────────────────────────────────

  it('calls setSubTab when a tab is clicked', () => {
    renderPage()
    fireEvent.click(screen.getByText('nav.relay'))
    expect(mockSetSubTab).toHaveBeenCalledWith('gateway', 'relay')
  })

  it('calls setSubTab for usage tab click', () => {
    renderPage()
    fireEvent.click(screen.getByText('home.gwUsage'))
    expect(mockSetSubTab).toHaveBeenCalledWith('gateway', 'usage')
  })

  // ── Reseller: clicking an admin tab fires setSubTab ───────────────────────

  it('calls setSubTab for an admin tab click in reseller mode', () => {
    mockAppMode = 'reseller'
    mockUserInfo = { role: 10 }
    renderPage()
    fireEvent.click(screen.getByText('gateway.channels'))
    expect(mockSetSubTab).toHaveBeenCalledWith('gateway', 'channels')
  })

  // ── Content routing for sub-tabs ─────────────────────────────────────────

  it('renders RelayPage when active tab is relay', () => {
    mockSubTab = 'relay'
    renderPage()
    expect(screen.getByTestId('relay-page')).toBeInTheDocument()
  })

  it('renders SwitchHubPage with section=usage when active tab is usage', () => {
    mockSubTab = 'usage'
    renderPage()
    expect(screen.getByTestId('switch-hub-usage')).toBeInTheDocument()
  })

  it('renders SwitchHubPage with section=apps when active tab is apps', () => {
    mockSubTab = 'apps'
    renderPage()
    expect(screen.getByTestId('switch-hub-apps')).toBeInTheDocument()
  })

  it('renders GatewayDashboardPage inside guard when tab is dashboard', () => {
    mockAppMode = 'reseller'
    mockUserInfo = { role: 10 }
    mockSubTab = 'dashboard'
    renderPage()
    expect(screen.getByTestId('gw-dashboard')).toBeInTheDocument()
  })

  it('renders GatewayTokenPage inside guard when tab is tokens', () => {
    mockAppMode = 'reseller'
    mockUserInfo = { role: 10 }
    mockSubTab = 'tokens'
    renderPage()
    expect(screen.getByTestId('gw-tokens')).toBeInTheDocument()
  })

  it('renders GatewayWalletPage inside guard when tab is wallet', () => {
    mockAppMode = 'reseller'
    mockUserInfo = { role: 10 }
    mockSubTab = 'wallet'
    renderPage()
    expect(screen.getByTestId('gw-wallet')).toBeInTheDocument()
  })

  it('renders ToolReleasePage when tab is tool-releases', () => {
    mockAppMode = 'reseller'
    mockUserInfo = { role: 10 }
    mockSubTab = 'tool-releases'
    renderPage()
    expect(screen.getByTestId('tool-releases')).toBeInTheDocument()
  })

  it('renders AuditLogPanel and AdminPage when tab is system (root)', () => {
    mockAppMode = 'reseller'
    mockUserInfo = { role: 100 }
    mockSubTab = 'system'
    renderPage()
    expect(screen.getByTestId('audit-log')).toBeInTheDocument()
    expect(screen.getByTestId('admin-page')).toBeInTheDocument()
  })

  // ── Active tab styling: active tab renders with [ LABEL ] format ──────────

  it('marks the active tab with bracket formatting', () => {
    mockSubTab = 'control'
    renderPage()
    // The active tab label is uppercased and wrapped in brackets.
    // Since i18n stub returns 'home.gwControl' for that key, the active tab
    // renders as '[ HOME.GWCONTROL ]'
    expect(screen.getByText('[ HOME.GWCONTROL ]')).toBeInTheDocument()
  })

  it('does not apply bracket format to inactive tabs', () => {
    mockSubTab = 'control'
    renderPage()
    // relay tab is inactive — rendered without brackets
    expect(screen.getByText('nav.relay')).toBeInTheDocument()
    expect(screen.queryByText(/\[\s*NAV.RELAY\s*\]/)).not.toBeInTheDocument()
  })
})
