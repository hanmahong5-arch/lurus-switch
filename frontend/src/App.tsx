import { useEffect, useState } from 'react'
import { Loader2 } from 'lucide-react'
import { AnimatePresence, motion } from 'framer-motion'
import './style.css'
import { Sidebar, RESELLER_ONLY_PAGES, PERSONAL_ONLY_PAGES, ENDUSER_VISIBLE_PAGES, ADMIN_PAGES } from './components/Sidebar'
import { PageHeader } from './components/PageHeader'
import { StatusBar } from './components/StatusBar'
import { ErrorBoundary } from './components/ErrorBoundary'
import { ToastContainer } from './components/Toast'
import { ConnectionBanner } from './components/ConnectionBanner'
import { SetupWizard } from './components/SetupWizard'
import { CommandPalette } from './components/CommandPalette'
import { DeepLinkImportModal } from './components/DeepLinkImportModal'
import { RepoAuditModal } from './components/RepoAuditModal'
import { useRepoAuditStore } from './stores/repoAuditStore'
import { BashGuardModal } from './components/BashGuardModal'
import { useBashGuardStore } from './stores/bashGuardStore'
import { BudgetModal } from './components/BudgetModal'
import { useBudgetStore } from './stores/budgetStore'
import { ActivityPane } from './components/ActivityPane'
import { ActivityDrawer, useActivityIngest } from './components/ActivityDrawer'
import { SnapshotsHub } from './components/snapshots/SnapshotsHub'
import { FeatureTourModal } from './components/FeatureTourModal'
import { useFeatureTourStore } from './stores/featureTourStore'
import { HomePage } from './pages/HomePage'
import { AgentsPage } from './pages/AgentsPage'
import { NewToolsPage } from './pages/NewToolsPage'
import { NewGatewayPage } from './pages/NewGatewayPage'
import { WorkspacePage } from './pages/WorkspacePage'
import { AccountPage } from './pages/AccountPage'
import { SettingsPage } from './pages/SettingsPage'
import { PromoterHubPage } from './pages/PromoterHubPage'
import { PackagerPage } from './pages/PackagerPage'
import { DLPAdminPage } from './pages/DLPAdminPage'
import { OrgChartPage } from './pages/OrgChartPage'
import { AgentTemplateGalleryPage } from './pages/AgentTemplateGalleryPage'
import { ChargebackPage } from './pages/ChargebackPage'
import { ConversationsPage } from './pages/ConversationsPage'
import { LiveSessionsPage } from './pages/LiveSessionsPage'
import { AppModeSelectPage } from './pages/AppModeSelectPage'
import { ResellerSetupWizard } from './pages/ResellerSetupWizard'
import { EndUserActivationPage } from './pages/EndUserActivationPage'
import { EndUserMainPage } from './pages/EndUserMainPage'
import { DesignSystemPage } from './pages/_DesignSystemPage'
import { useConfigStore, migrateLegacyRoute, migrateLegacyAppMode, type AppMode, type UserLevel } from './stores/configStore'
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts'
import { usePlatformEvents } from './hooks/usePlatformEvents'
import { useNavPersist } from './lib/useNavPersist'
import { useGatewayStore } from './stores/gatewayStore'
import { useBillingStore } from './stores/billingStore'
import { useDashboardStore } from './stores/dashboardStore'
import { useSwitchStore } from './stores/switchStore'
import { GetAppSettings, GetProxySettings, GetServerStatus, GetServerAdminToken, BillingGetUserInfo, BillingGetQuotaSummary, GetGatewayStatus, HasResellerConfig, GetEndUserStatus } from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'
import i18n from './i18n'

// Legacy startup pages map to new routes
const VALID_STARTUP_PAGES: ReadonlySet<string> = new Set([
  'home', 'agents', 'conversations', 'tools', 'gateway', 'workspace', 'account', 'settings',
  // Legacy values still accepted for backward compatibility
  'dashboard', 'claude', 'codex', 'gemini', 'picoclaw', 'nullclaw',
])

function App() {
  // Dev-only Storybook-lite gate. Visit `/#design` to load the design system
  // showcase, bypassing onboarding / mode picker / wizards. Hash is reactive
  // so back/forward works without page reload.
  const [designMode, setDesignMode] = useState(() => window.location.hash === '#design')
  useEffect(() => {
    const onHash = () => setDesignMode(window.location.hash === '#design')
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  }, [])

  const { activeTool, setActiveTool, appMode, setAppMode, setUserLevel, setSubTab } = useConfigStore()
  useNavPersist()
  useKeyboardShortcuts()
  usePlatformEvents()
  // Subscribe to activity:event once for the whole session — required so
  // the drawer collects history while it's closed too.
  useActivityIngest()
  const [showWizard, setShowWizard] = useState<boolean | null>(null)
  // null = still loading; once resolved, holds the boot-time mode for routing.
  const [bootMode, setBootMode] = useState<AppMode | null>(null)
  // Reseller-specific gate: null while we're checking, true/false once known.
  // When true AND mode is reseller, we render the ResellerSetupWizard before
  // the main shell.
  const [needsResellerSetup, setNeedsResellerSetup] = useState<boolean | null>(null)
  // EndUser activation gate: null while checking, then 'unactivated' /
  // 'active' / 'revoked' / 'device_mismatch' / 'stale'. Anything other
  // than 'active' or 'stale' routes to the activation page.
  const [endUserState, setEndUserState] = useState<string | null>(null)
  const [endUserHubURL, setEndUserHubURL] = useState<string>('')
  const { startPolling, stopPolling } = useGatewayStore()
  const { setUserInfo } = useBillingStore()
  const { proxySettings } = useDashboardStore()
  const setGwStatus = useSwitchStore((s) => s.setStatus)
  const openFeatureTour = useFeatureTourStore((s) => s.setOpen)

  useEffect(() => {
    GetAppSettings()
      .then((s) => {
        setShowWizard(!s.onboardingCompleted)
        // Show the feature tour automatically after the user has finished
        // initial setup but before they've seen the tour. Skipped if
        // setup wizard is still pending — we don't want to stack two
        // modals on first run.
        if (s.onboardingCompleted && !(s as any).featureTourSeen) {
          // Defer briefly so the main shell paints first.
          setTimeout(() => openFeatureTour(true), 500)
        }

        // Resolve app mode (auto-migrate legacy 'user'/'promoter' values).
        const resolved = migrateLegacyAppMode(s.appMode)
        setAppMode(resolved)
        setBootMode(resolved)

        // Reseller mode needs Hub URL + admin token before any of the
        // GatewayXxx pages can do anything useful — gate the main UI behind
        // the setup wizard until the config is on disk.
        if (resolved === 'reseller') {
          HasResellerConfig()
            .then((has) => setNeedsResellerSetup(!has))
            .catch(() => setNeedsResellerSetup(true))
        } else {
          setNeedsResellerSetup(false)
        }

        // EndUser mode needs an activation file. Read the lifecycle state
        // and remember the locked Hub URL so the activation page can show
        // it read-only.
        if (resolved === 'enduser') {
          setEndUserHubURL((s as any).lockedHubUrl ?? '')
          GetEndUserStatus()
            .then((st) => setEndUserState(st?.state ?? 'unactivated'))
            .catch(() => setEndUserState('unactivated'))
        } else {
          setEndUserState('skip')
        }

        // Apply saved user level
        const level = (s as any).userLevel
        if (level === 'beginner' || level === 'regular' || level === 'power') {
          setUserLevel(level as UserLevel)
        }

        // Apply saved language preference at startup
        if (s.language && s.language !== i18n.language) {
          i18n.changeLanguage(s.language)
        }

        // Apply saved theme preference at startup
        const root = document.documentElement
        if (s.theme === 'auto') {
          const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
          root.classList.toggle('dark', prefersDark)
        } else if (s.theme) {
          root.classList.toggle('dark', s.theme === 'dark')
        }

        // White-label branding: when a signed sidecar has populated brand
        // fields on disk, apply them now so the EndUser sees the
        // distributor's identity, not stock Lurus. Persisted to a CSS
        // variable so any component can read it via `var(--brand-color)`.
        const brandColor = (s as any).brandPrimaryColor
        if (typeof brandColor === 'string' && brandColor) {
          root.style.setProperty('--brand-color', brandColor)
        }
        // Brand name + logo + support contact are picked up by Sidebar /
        // EndUserMainPage via the same GetAppSettings call.

        // Navigate to saved startup page (with legacy migration)
        if (s.startupPage && VALID_STARTUP_PAGES.has(s.startupPage)) {
          const migrated = migrateLegacyRoute(s.startupPage)
          setActiveTool(migrated.tool)
          if (migrated.subTab) {
            setSubTab(migrated.tool, migrated.subTab)
          }
        }
      })
      .catch((e) => {
        console.error('GetAppSettings failed:', e)
        setShowWizard(false)
        setBootMode('unset')
        setNeedsResellerSetup(false)
        setEndUserState('skip')
      })
  }, [])

  // Load proxy settings into dashboardStore on startup so billing polling and
  // AccountStatusBadge work immediately (not just after visiting Account page).
  useEffect(() => {
    GetProxySettings()
      .then((r) => useDashboardStore.getState().setProxySettings(r))
      .catch(() => {})
  }, [])

  // Start global gateway status polling when the main app mounts.
  useEffect(() => {
    startPolling(
      () => GetServerStatus() as ReturnType<typeof GetServerStatus>,
      () => GetServerAdminToken(),
    )
    return () => stopPolling()
  }, [])

  // Poll Switch gateway status globally so StatusBar indicator stays current.
  useEffect(() => {
    const poll = () => { GetGatewayStatus().then(setGwStatus).catch(() => {}) }
    poll()
    const h = setInterval(poll, 10_000)
    return () => clearInterval(h)
  }, [setGwStatus])

  // Poll billing user info every 5 minutes to keep AccountStatusBadge fresh.
  useEffect(() => {
    if (!proxySettings.userToken) return
    const fetchBilling = () => {
      BillingGetUserInfo().then(setUserInfo).catch(() => {})
    }
    fetchBilling()
    const handle = setInterval(fetchBilling, 5 * 60 * 1000)
    return () => clearInterval(handle)
  }, [proxySettings.userToken, setUserInfo])

  // EndUser heartbeat listener: when the Hub revokes a token or the
  // device fingerprint mismatches, bounce back to the activation page.
  // The Wails event payload mirrors redemption.StatusEvent.
  useEffect(() => {
    if (appMode !== 'enduser') return
    const unsub = EventsOn('redemption:heartbeat', (ev: { state?: string }) => {
      if (!ev?.state) return
      if (ev.state === 'revoked' || ev.state === 'device_mismatch') {
        setEndUserState(ev.state)
      } else if (ev.state === 'active' || ev.state === 'stale') {
        setEndUserState(ev.state)
      }
    })
    return () => { if (unsub) unsub() }
  }, [appMode])

  // Dev-only design-system showcase. `/#design` short-circuits every gate.
  if (designMode) {
    return <DesignSystemPage />
  }

  // Loading state while checking onboarding status. We surface which
  // step is still pending so users see *what* Switch is doing during
  // the boot serial-load (GetAppSettings → mode resolve → reseller /
  // enduser gate checks). Empty spinner felt like the app was stuck.
  if (
    showWizard === null ||
    bootMode === null ||
    needsResellerSetup === null ||
    endUserState === null
  ) {
    const steps = [
      { done: showWizard !== null, zh: '读取应用配置', en: 'Reading app settings' },
      { done: bootMode !== null, zh: '解析运行模式', en: 'Resolving app mode' },
      { done: needsResellerSetup !== null, zh: '检查 Reseller Hub', en: 'Checking Reseller Hub' },
      { done: endUserState !== null, zh: '检查 EndUser 激活', en: 'Checking EndUser activation' },
    ]
    return (
      <div className="h-screen flex items-center justify-center bg-background text-foreground">
        <div className="text-center space-y-4 max-w-xs">
          <div className="inline-flex items-center justify-center h-14 w-14 rounded-full bg-primary/10 border border-primary/30">
            <Loader2 className="h-6 w-6 animate-spin text-primary" />
          </div>
          <div>
            <h1 className="text-base font-semibold">Lurus Switch</h1>
            <p className="text-xs text-muted-foreground mt-0.5">本地 AI 网关 · Local AI Gateway</p>
          </div>
          <ul className="space-y-1 text-xs text-left">
            {steps.map((s, i) => (
              <li key={i} className="flex items-center gap-2">
                <span className={`inline-block h-1.5 w-1.5 rounded-full ${s.done ? 'bg-emerald-400' : 'bg-muted-foreground/40 animate-pulse'}`} />
                <span className={s.done ? 'text-muted-foreground line-through opacity-60' : 'text-foreground'}>
                  {s.zh}
                  <span className="ml-1.5 text-[10px] text-muted-foreground/60">{s.en}</span>
                </span>
              </li>
            ))}
          </ul>
        </div>
      </div>
    )
  }

  // First-launch mode picker — gates everything else, including the legacy
  // setup wizard. EndUser white-label builds bypass this because the backend
  // mode is already set by the packaging step.
  if (appMode === 'unset') {
    return <AppModeSelectPage onPick={(picked) => setAppMode(picked)} />
  }

  // Reseller-specific gate: must have a Hub URL + admin token before the
  // GatewayXxx pages can be used. Runs after the AppMode picker but before
  // the legacy SetupWizard, since SetupWizard targets Personal CLI install.
  if (appMode === 'reseller' && needsResellerSetup) {
    return <ResellerSetupWizard onComplete={() => setNeedsResellerSetup(false)} />
  }

  // EndUser activation gate: anything other than active/stale forces the
  // activation page. Stale tokens still let the user into the main app
  // (the heartbeat banner takes over from there).
  if (
    appMode === 'enduser' &&
    endUserState !== 'active' &&
    endUserState !== 'stale' &&
    endUserState !== 'skip'
  ) {
    return (
      <EndUserActivationPage
        hubURL={endUserHubURL}
        onActivated={() => setEndUserState('active')}
      />
    )
  }

  // Show wizard if onboarding not completed
  if (showWizard) {
    return <SetupWizard onComplete={() => setShowWizard(false)} />
  }

  const renderPage = () => {
    // Mode-based route guards (S-Xa.3). Hidden pages route to mode-appropriate
    // home so a stale activeTool from before a mode switch can't expose UI.
    if (RESELLER_ONLY_PAGES.has(activeTool) && appMode !== 'reseller') {
      return appMode === 'enduser'
        ? <EndUserMainPage onDeactivated={() => setEndUserState('unactivated')} />
        : <HomePage />
    }
    if (PERSONAL_ONLY_PAGES.has(activeTool) && appMode !== 'personal') {
      return appMode === 'enduser'
        ? <EndUserMainPage onDeactivated={() => setEndUserState('unactivated')} />
        : <HomePage />
    }
    // Admin pages (DLP / org chart / agent gallery) require Reseller
    // or Enterprise mode. Stale state from a mode switch shouldn't
    // expose them.
    if (ADMIN_PAGES.has(activeTool) && appMode !== 'reseller' && appMode !== 'enterprise') {
      return appMode === 'enduser'
        ? <EndUserMainPage onDeactivated={() => setEndUserState('unactivated')} />
        : <HomePage />
    }
    if (appMode === 'enduser' && !ENDUSER_VISIBLE_PAGES.has(activeTool)) {
      return <EndUserMainPage onDeactivated={() => setEndUserState('unactivated')} />
    }

    switch (activeTool) {
      case 'home':
        return appMode === 'enduser'
          ? <EndUserMainPage onDeactivated={() => setEndUserState('unactivated')} />
          : <HomePage />
      case 'live':
        return <LiveSessionsPage />
      case 'agents':
        return <AgentsPage />
      case 'tools':
        return <NewToolsPage />
      case 'gateway':
        return <NewGatewayPage />
      case 'workspace':
        return <WorkspacePage />
      case 'account':
        return <AccountPage />
      case 'settings':
        return <SettingsPage />
      case 'promotion':
        return <PromoterHubPage />
      case 'packager':
        return <PackagerPage />
      case 'dlp':
        return <DLPAdminPage />
      case 'orgchart':
        return <OrgChartPage />
      case 'agent-templates':
        return <AgentTemplateGalleryPage />
      case 'chargeback':
        return <ChargebackPage />
      case 'conversations':
        return <ConversationsPage />
      default:
        return <HomePage />
    }
  }

  return (
    <div className="flex flex-col h-screen bg-background text-foreground">
      <ConnectionBanner />
      <ToastContainer />
      <CommandPalette />
      <DeepLinkImportModal />
      <RepoAuditMount />
      <BashGuardMount />
      <BudgetMount />
      <FeatureTourMount />
      <ActivityPane />
      <ActivityDrawer />
      <SnapshotsHub />
      <div className="flex flex-1 overflow-hidden">
        <Sidebar />
        <main className="flex-1 flex flex-col overflow-hidden">
          <PageHeader />
          <div className="flex-1 overflow-hidden">
            <ErrorBoundary key={activeTool} name={`page:${activeTool}`} page={activeTool}>
              <AnimatePresence mode="wait" initial={false}>
                <motion.div
                  key={`${activeTool}:${useConfigStore.getState().subTabState[activeTool] ?? ''}`}
                  initial={{ opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -2 }}
                  transition={{ duration: 0.15, ease: 'easeOut' }}
                  className="h-full"
                >
                  {renderPage()}
                </motion.div>
              </AnimatePresence>
            </ErrorBoundary>
          </div>
        </main>
      </div>
      <StatusBar />
    </div>
  )
}

// Tiny mount wrapper so we can subscribe to the audit store without
// re-rendering the whole App tree on open/close transitions.
function RepoAuditMount() {
  const open = useRepoAuditStore((s) => s.open)
  const setOpen = useRepoAuditStore((s) => s.setOpen)
  return <RepoAuditModal open={open} onClose={() => setOpen(false)} />
}

function BashGuardMount() {
  const open = useBashGuardStore((s) => s.open)
  const setOpen = useBashGuardStore((s) => s.setOpen)
  return <BashGuardModal open={open} onClose={() => setOpen(false)} />
}

function BudgetMount() {
  const open = useBudgetStore((s) => s.open)
  const setOpen = useBudgetStore((s) => s.setOpen)
  return <BudgetModal open={open} onClose={() => setOpen(false)} />
}

function FeatureTourMount() {
  const open = useFeatureTourStore((s) => s.open)
  const setOpen = useFeatureTourStore((s) => s.setOpen)
  return <FeatureTourModal open={open} onClose={() => setOpen(false)} />
}

export default App
