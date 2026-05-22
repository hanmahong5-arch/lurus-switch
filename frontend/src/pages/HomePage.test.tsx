import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

// i18n stub — fallback wins, {{vars}} interpolated.
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

// React Flow needs DOM APIs jsdom doesn't ship. Stub the whole module
// with lightweight placeholders so TopologyView renders without
// exercising the real layout engine — we're testing HomePage shell,
// not topology rendering.
vi.mock('@xyflow/react', () => {
  const Pass = ({ children }: { children?: React.ReactNode }) => children ?? null
  return {
    ReactFlow: ({ children }: { children?: React.ReactNode }) => (
      <div data-testid="reactflow-stub">{children}</div>
    ),
    ReactFlowProvider: Pass,
    useReactFlow: () => ({ fitView: vi.fn() }),
    Background: () => null,
    Controls: () => null,
    MiniMap: () => null,
    Handle: () => null,
    Position: { Top: 'top', Bottom: 'bottom', Left: 'left', Right: 'right' },
  }
})

vi.mock('@dagrejs/dagre', () => ({
  default: {
    graphlib: { Graph: class { setDefaultEdgeLabel() { return this } setGraph() { return this } setNode() {} setEdge() {} node() { return { x: 0, y: 0 } } } },
    layout: () => {},
  },
}))

vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn().mockReturnValue(() => {}),
  EventsOff: vi.fn(),
  EventsEmit: vi.fn(),
}))

// Wails bindings: HomePage + child components together import ~30 funcs.
// Default everything to empty/no-op resolves so the page mounts cleanly.
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
  GetAppVersion: vi.fn().mockResolvedValue('0.5.0'),
  CheckSelfUpdate: vi.fn().mockResolvedValue({ updateAvailable: false, latestVersion: '0.5.0' }),
  ApplySelfUpdate: vi.fn().mockResolvedValue(undefined),
  FetchModelCatalog: vi.fn().mockResolvedValue({ models: [] }),
  SwitchModel: vi.fn().mockResolvedValue({}),
  FullSetupForGateway: vi.fn().mockResolvedValue({ errors: [] }),
  AutoConfigureToolsForGateway: vi.fn().mockResolvedValue({}),
  AutoConfigureToolForGateway: vi.fn().mockResolvedValue({}),
  StartGateway: vi.fn().mockResolvedValue(undefined),
  GetGatewayStatus: vi.fn().mockResolvedValue({ running: false }),
  GetGYProducts: vi.fn().mockResolvedValue([]),
  CheckGYStatus: vi.fn().mockResolvedValue([]),
  LaunchGYProduct: vi.fn().mockResolvedValue(undefined),
  InstallDependency: vi.fn().mockResolvedValue({ success: true, message: '' }),
  AutoFixToolConfig: vi.fn().mockResolvedValue(undefined),
  ApplyAllOptimizations: vi.fn().mockResolvedValue([]),
  ComputeHealthScore: vi.fn().mockResolvedValue({ score: 100, suggestions: [] }),
  GetTopologySnapshot: vi.fn().mockResolvedValue({
    nodes: [], edges: [], generatedAt: new Date().toISOString(),
    summary: {
      ok: 0, degraded: 0, down: 0, notconfigured: 0, unknown: 0,
      headline: 'No data',
    },
  }),
  // Children:
  CheckDependencies: vi.fn().mockResolvedValue({ runtimes: [], allMet: true }),
  GetToolRuntimes: vi.fn().mockResolvedValue([]),
  BillingGetQuotaSummary: vi.fn().mockResolvedValue(null),
  BillingGetIdentityOverview: vi.fn().mockResolvedValue(null),
  LaunchToolInTerminal: vi.fn().mockResolvedValue(undefined),
  Login: vi.fn().mockResolvedValue(undefined),
  GetOptimizationOpportunities: vi.fn().mockResolvedValue([]),
  ApplyOptimization: vi.fn().mockResolvedValue({ success: true }),
}))

vi.mock('../../wailsjs/go/models', () => ({
  proxy: { ProxySettings: { createFrom: (x: unknown) => x } },
  appconfig: { AppSettings: { createFrom: (x: unknown) => x } },
  gy: {},
  topology: {},
  toolruntime: {},
}))

// Polyfill ResizeObserver — some lucide / Card layouts touch it
// indirectly via dependencies.
beforeEach(() => {
  if (!(globalThis as any).ResizeObserver) {
    (globalThis as any).ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  }
})

import { HomePage } from './HomePage'

describe('HomePage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders without crashing — smoke', async () => {
    render(<HomePage />)
    // The HomeIntentPanel ships the bilingual intent cards. The Chinese
    // labels are hard-coded into the component (not i18n-routed) so they
    // are stable across the i18n stub.
    await waitFor(() => {
      expect(screen.getByText('接中转站')).toBeDefined()
    }, { timeout: 3000 })
  })

  it('renders the HomeIntentPanel verb-first intent cards', async () => {
    render(<HomePage />)
    await waitFor(() => {
      // Three intent cards drive most of the home page CTA surface area.
      expect(screen.getByText('接中转站')).toBeDefined()
      expect(screen.getByText('换服务商')).toBeDefined()
      expect(screen.getByText('启用 Bash-Guard')).toBeDefined()
    })
  })

  it('renders the action rail (install/start/connect/fix) labels via QuickActionCards', async () => {
    render(<HomePage />)
    await waitFor(() => {
      // QuickActionCards labels come through i18n — keys render verbatim
      // since the stub returns the key when no fallback is provided.
      // Anchor on dashboard.installAll (also used as the bulk button label).
      const installs = screen.getAllByText('dashboard.installAll')
      expect(installs.length).toBeGreaterThanOrEqual(1)
    })
  })

  it('renders the topology stub area (TopologyView mounts ReactFlow)', async () => {
    render(<HomePage />)
    await waitFor(() => {
      // Our ReactFlow mock emits a data-testid wrapper. Its presence
      // proves the topology slot rendered without crashing on the real
      // React Flow engine inside jsdom.
      expect(screen.getByTestId('reactflow-stub')).toBeDefined()
    })
  })

  it('renders the app version footer with the update-check button', async () => {
    render(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText('dashboard.checkUpdates')).toBeDefined()
    })
  })

  it('renders the bulk Install All button at the bottom of the page', async () => {
    render(<HomePage />)
    await waitFor(() => {
      // Multiple install-all buttons can exist (empty state + bulk bar)
      // — at minimum one must render.
      expect(screen.getAllByText('dashboard.installAll').length).toBeGreaterThanOrEqual(1)
    })
  })
})
