import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

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

// DashboardPage subscribes to EventsOn — stub the Wails runtime so it
// doesn't fall through to the real (undefined) global runtime.
vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn().mockReturnValue(() => {}),
  EventsOff: vi.fn(),
  EventsEmit: vi.fn(),
}))

// Wails bindings — return empty/default shapes so no `undefined.foo` blows up.
vi.mock('../../wailsjs/go/main/App', () => ({
  DetectAllTools: vi.fn().mockResolvedValue({}),
  InstallTool: vi.fn().mockResolvedValue(undefined),
  InstallAllTools: vi.fn().mockResolvedValue(undefined),
  UpdateTool: vi.fn().mockResolvedValue(undefined),
  UpdateAllTools: vi.fn().mockResolvedValue(undefined),
  UninstallTool: vi.fn().mockResolvedValue(undefined),
  CheckAllUpdates: vi.fn().mockResolvedValue({}),
  CheckAllToolHealth: vi.fn().mockResolvedValue({}),
  GetProxySettings: vi.fn().mockResolvedValue({
    apiEndpoint: '', apiKey: '', registrationUrl: '', tenantSlug: '', userToken: '',
  }),
  SaveProxySettings: vi.fn().mockResolvedValue(undefined),
  ConfigureAllProxy: vi.fn().mockResolvedValue({}),
  ConfigureAllToolsRelay: vi.fn().mockResolvedValue({}),
  GetAppVersion: vi.fn().mockResolvedValue('0.5.0'),
  CheckSelfUpdate: vi.fn().mockResolvedValue({ name: 'lurus-switch', currentVersion: '0.5.0', latestVersion: '0.5.0', updateAvailable: false }),
  ApplySelfUpdate: vi.fn().mockResolvedValue(undefined),
  SaveAppSettings: vi.fn().mockResolvedValue(undefined),
  GetAppSettings: vi.fn().mockResolvedValue({}),
  FetchModelCatalog: vi.fn().mockResolvedValue({ models: [] }),
  SwitchModel: vi.fn().mockResolvedValue({}),
  // DashboardQuotaWidget + DepTreePanel deps pulled in via DashboardPage:
  BillingGetUserInfo: vi.fn().mockResolvedValue(null),
  CheckDependencies: vi.fn().mockResolvedValue({ runtimes: [], allMet: true }),
  InstallDependency: vi.fn().mockResolvedValue({ success: true, message: '' }),
  // CostDashboardWidget — Wave3 W3.3 addition.
  GetCostDashboard: vi.fn().mockResolvedValue({
    todayUSD: 0, todayTokensIn: 0, todayTokensOut: 0, todayCalls: 0,
    byModel: [], budgetEnabled: false,
  }),
}))

vi.mock('../../wailsjs/go/models', () => ({
  proxy: { ProxySettings: { createFrom: (x: unknown) => x } },
  appconfig: { AppSettings: { createFrom: (x: unknown) => x } },
}))

import { DashboardPage } from './DashboardPage'
import { useDashboardStore } from '../stores/dashboardStore'

describe('DashboardPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Wipe store state so leftover tools / installing flags from one test
    // don't bleed into the next assertion.
    useDashboardStore.setState({
      tools: {}, installing: {}, updating: {}, detecting: false,
      toolHealth: {},
      proxySettings: { apiEndpoint: '', apiKey: '' },
      proxySaving: false, proxyConfiguring: false,
      appVersion: '', selfUpdateInfo: null, checkingUpdates: false, error: null,
    } as any)
  })

  it('renders without crashing — smoke', async () => {
    render(<DashboardPage />)
    // Title is the cheapest anchor; resolves once GetAppVersion / GetProxySettings settle.
    await waitFor(() => {
      expect(screen.getByText('dashboard.title')).toBeDefined()
    })
  })

  it('renders the subtitle', async () => {
    render(<DashboardPage />)
    await waitFor(() => {
      expect(screen.getByText('dashboard.subtitle')).toBeDefined()
    })
  })

  it('renders the Refresh button', async () => {
    render(<DashboardPage />)
    await waitFor(() => {
      expect(screen.getByText('dashboard.refresh')).toBeDefined()
    })
  })

  it('renders the Install All bulk action button', async () => {
    render(<DashboardPage />)
    await waitFor(() => {
      // Multiple "installAll" buttons may appear (empty state CTA + bottom bulk action).
      const matches = screen.getAllByText('dashboard.installAll')
      expect(matches.length).toBeGreaterThanOrEqual(1)
    })
  })

  it('renders the proxy config disclosure (collapsed by default)', async () => {
    render(<DashboardPage />)
    await waitFor(() => {
      expect(screen.getByText('dashboard.proxyConfig')).toBeDefined()
    })
  })

  it('renders the empty state when no tools are installed', async () => {
    render(<DashboardPage />)
    // Inject pre-detected tools (all uninstalled) into the store to
    // trigger the "no tools" branch — without this the page sits in the
    // "still detecting" state and renders skeleton cards instead.
    useDashboardStore.setState({
      detecting: false,
      tools: Object.fromEntries(
        ['claude', 'codex', 'gemini', 'picoclaw', 'nullclaw', 'zeroclaw', 'openclaw'].map((n) => [
          n, { name: n, installed: false, version: '', latestVersion: '', updateAvailable: false, path: '' },
        ]),
      ),
    } as any)
    await waitFor(() => {
      expect(screen.getByText('dashboard.noToolsTitle')).toBeDefined()
      expect(screen.getByText('dashboard.runWizard')).toBeDefined()
    })
  })

  it('renders the app version footer', async () => {
    useDashboardStore.setState({ appVersion: '0.5.0' } as any)
    render(<DashboardPage />)
    await waitFor(() => {
      // i18n stub interpolates {{version}} → "0.5.0" into the fallback,
      // but no fallback is given here. We can still assert the dashboard
      // footer slot renders the "check for updates" button.
      expect(screen.getByText('dashboard.checkUpdates')).toBeDefined()
    })
  })
})
