import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

// i18n stub mirrors EndUserActivationPage.test.tsx — fallback wins,
// `{{vars}}` interpolated. SettingsPage uses fallbacks heavily because
// the new info strip + notify tab were added before zh.json caught up.
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
  // `src/i18n/index.ts` is pulled in transitively (via `lib/dirtyGuard`
  // → other libs) and calls `i18n.use(initReactI18next).init(...)`. The
  // stub must accept that without throwing.
  initReactI18next: { type: '3rdParty', init: () => {} },
}))

// matchMedia polyfill — SettingsPage subscribes to system theme changes
// when theme === 'auto'. jsdom doesn't ship matchMedia so we stub it.
beforeEach(() => {
  if (!window.matchMedia) {
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        addListener: vi.fn(),
        removeListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    })
  }
})

const mockGetAppSettings = vi.fn().mockResolvedValue({
  theme: 'dark',
  language: 'zh',
  autoUpdate: true,
  editorFontSize: 13,
  startupPage: 'home',
  onboardingCompleted: true,
  appMode: 'personal',
})
const mockIsModeLocked = vi.fn().mockResolvedValue(false)
const mockGetSystemInfo = vi.fn().mockResolvedValue({
  appVersion: '0.5.0', goos: 'windows', goarch: 'amd64',
})
const mockGetConfigDir = vi.fn().mockResolvedValue('C:\\Users\\test\\AppData\\Roaming\\lurus-switch')

vi.mock('../../wailsjs/go/main/App', () => ({
  GetAppSettings: (...a: unknown[]) => mockGetAppSettings(...a),
  SaveAppSettings: vi.fn().mockResolvedValue(undefined),
  ClearAllSnapshots: vi.fn().mockResolvedValue(0),
  ClearAllUserPrompts: vi.fn().mockResolvedValue(0),
  SetAppMode: vi.fn().mockResolvedValue(undefined),
  IsModeLocked: (...a: unknown[]) => mockIsModeLocked(...a),
  GetSystemInfo: (...a: unknown[]) => mockGetSystemInfo(...a),
  GetConfigDir: (...a: unknown[]) => mockGetConfigDir(...a),
  OpenConfigDir: vi.fn().mockResolvedValue(undefined),
  // UpstreamProxySection deps (rendered inside the proxy tab):
  GetProxySettings: vi.fn().mockResolvedValue({
    apiEndpoint: '', apiKey: '', registrationUrl: '', tenantSlug: '', userToken: '',
  }),
  SaveProxySettings: vi.fn().mockResolvedValue(undefined),
  GetUpstreamProxy: vi.fn().mockResolvedValue({ enabled: false, url: '', noProxy: '', testUrl: '' }),
  TestUpstreamProxy: vi.fn().mockResolvedValue({ ok: false }),
  DetectLocalProxies: vi.fn().mockResolvedValue([]),
  // ConnectivityDoctor is nested under UpstreamProxySection and runs a
  // diagnostic on mount of the proxy tab. Stub the call so the rejection
  // doesn't surface as a stderr noise line.
  RunConnectivityDiagnostic: vi.fn().mockResolvedValue({ steps: [], summary: '' }),
  // CompetingInstallBanner:
  DetectCompetingInstalls: vi.fn().mockResolvedValue([]),
}))

vi.mock('../../wailsjs/go/models', () => ({
  appconfig: { AppSettings: { createFrom: (x: unknown) => x } },
  proxy: { ProxySettings: { createFrom: (x: unknown) => x } },
  netproxy: {},
  main: {},
}))

// notifyApi is hand-rolled outside the Wails generator — stub the
// surface SettingsPage's NotifyTab calls into so importing the page
// doesn't blow up. Not exercised here; the notify tab isn't asserted.
vi.mock('../lib/notifyApi', () => ({
  DEFAULT_NOTIFY_CONFIG: {
    enabled: false,
    feishu: { webhookUrl: '', secret: '' },
    rules: { notifyStuck: true, notifyDone: true },
  },
  getNotifyConfig: vi.fn().mockResolvedValue({
    enabled: false,
    feishu: { webhookUrl: '', secret: '' },
    rules: { notifyStuck: true, notifyDone: true },
  }),
  getRecentNotifications: vi.fn().mockResolvedValue([]),
  saveNotifyConfig: vi.fn().mockResolvedValue(undefined),
  testNotify: vi.fn().mockResolvedValue(undefined),
}))

import { SettingsPage } from './SettingsPage'

describe('SettingsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders without crashing — smoke', async () => {
    render(<SettingsPage />)
    // Loader shows first; wait for the header to appear once GetAppSettings resolves.
    await waitFor(() => {
      expect(screen.getByText('settings.title')).toBeDefined()
    })
  })

  it('renders the Save button and section subtitle', async () => {
    render(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText('settings.subtitle')).toBeDefined()
      expect(screen.getByText('settings.save')).toBeDefined()
    })
  })

  it('renders the version / platform info strip', async () => {
    render(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText('v0.5.0')).toBeDefined()
      expect(screen.getByText('windows/amd64')).toBeDefined()
    })
  })

  it('renders all 5 tab labels (appearance, proxy, notify, update, data)', async () => {
    render(<SettingsPage />)
    await waitFor(() => {
      // Default active tab is "appearance" — shown bracketed-uppercase.
      expect(screen.getByText(/SETTINGS\.TABS\.APPEARANCE/)).toBeDefined()
      // Inactive tabs render as the raw key (plain-text mode).
      expect(screen.getByText('settings.tabs.proxy')).toBeDefined()
      expect(screen.getByText('通知')).toBeDefined() // notify tab uses zh fallback
      expect(screen.getByText('settings.tabs.update')).toBeDefined()
      expect(screen.getByText('settings.tabs.data')).toBeDefined()
    })
  })

  it('renders the mode toggle section with personal/reseller/enduser pills', async () => {
    render(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText('settings.appMode')).toBeDefined()
      // "personal" also appears in the info-strip mode badge — assert that
      // each label is present at least once across both spots, but expect
      // multiple occurrences for the active mode.
      expect(screen.getAllByText('personal').length).toBeGreaterThanOrEqual(1)
      expect(screen.getByText('reseller')).toBeDefined()
      expect(screen.getByText('enduser')).toBeDefined()
    })
  })

  it('renders the user level toggle (beginner/regular/power)', async () => {
    render(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText('settings.level.beginner')).toBeDefined()
      expect(screen.getByText('settings.level.regular')).toBeDefined()
      expect(screen.getByText('settings.level.power')).toBeDefined()
    })
  })

  it('switches to the Proxy tab and shows the upstream-proxy area', async () => {
    render(<SettingsPage />)
    // Wait for tabs to be ready.
    await waitFor(() => screen.getByText('settings.tabs.proxy'))
    // Click the proxy tab.
    fireEvent.click(screen.getByText('settings.tabs.proxy'))
    await waitFor(() => {
      // The proxy tab body includes a "moved notice" footer string that's
      // i18n-driven, plus the UpstreamProxySection above it. The notice key
      // is the cheapest anchor that proves we're inside the proxy tab.
      expect(screen.getByText('settings.proxy.movedNotice')).toBeDefined()
    })
  })

  it('opens the mode-switch confirmation when clicking a different mode', async () => {
    render(<SettingsPage />)
    await waitFor(() => screen.getByText('settings.appMode'))
    // Current is "personal" — click "reseller" to trigger the confirm dialog.
    fireEvent.click(screen.getByText('reseller'))
    await waitFor(() => {
      // The confirm dialog uses `settings.modeSwitchConfirm` with a fallback
      // that contains "{{mode}}" → mode label. The stub interpolates.
      expect(screen.getByText('settings.modeSwitchConfirm')).toBeDefined()
      expect(screen.getByText('settings.data.confirm')).toBeDefined()
      expect(screen.getByText('settings.data.cancel')).toBeDefined()
    })
  })
})
