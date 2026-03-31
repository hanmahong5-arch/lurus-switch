import { useEffect, useState } from 'react'
import { Loader2 } from 'lucide-react'
import './style.css'
import { Sidebar, PROMOTER_ONLY_PAGES } from './components/Sidebar'
import { StatusBar } from './components/StatusBar'
import { ErrorBoundary } from './components/ErrorBoundary'
import { ToastContainer } from './components/Toast'
import { ConnectionBanner } from './components/ConnectionBanner'
import { SetupWizard } from './components/SetupWizard'
import { CommandPalette } from './components/CommandPalette'
import { HomePage } from './pages/HomePage'
import { NewToolsPage } from './pages/NewToolsPage'
import { NewGatewayPage } from './pages/NewGatewayPage'
import { WorkspacePage } from './pages/WorkspacePage'
import { AccountPage } from './pages/AccountPage'
import { SettingsPage } from './pages/SettingsPage'
import { PromoterHubPage } from './pages/PromoterHubPage'
import { ApiAdminPage } from './pages/ApiAdminPage'
import { useConfigStore, migrateLegacyRoute, type ActiveTool, type UserLevel } from './stores/configStore'
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts'
import { useNavPersist } from './lib/useNavPersist'
import { useGatewayStore } from './stores/gatewayStore'
import { useBillingStore } from './stores/billingStore'
import { useDashboardStore } from './stores/dashboardStore'
import { useSwitchStore } from './stores/switchStore'
import { GetAppSettings, GetProxySettings, GetServerStatus, GetServerAdminToken, BillingGetUserInfo, BillingGetQuotaSummary, GetGatewayStatus } from '../wailsjs/go/main/App'
import i18n from './i18n'

// Legacy startup pages map to new routes
const VALID_STARTUP_PAGES: ReadonlySet<string> = new Set([
  'home', 'tools', 'gateway', 'workspace', 'account', 'settings',
  // Legacy values still accepted for backward compatibility
  'dashboard', 'claude', 'codex', 'gemini', 'picoclaw', 'nullclaw',
])

function App() {
  const { activeTool, setActiveTool, setAppMode, setUserLevel, setSubTab } = useConfigStore()
  useNavPersist()
  useKeyboardShortcuts()
  const [showWizard, setShowWizard] = useState<boolean | null>(null)
  const { startPolling, stopPolling } = useGatewayStore()
  const { setUserInfo } = useBillingStore()
  const { proxySettings } = useDashboardStore()
  const setGwStatus = useSwitchStore((s) => s.setStatus)

  useEffect(() => {
    GetAppSettings()
      .then((s) => {
        setShowWizard(!s.onboardingCompleted)

        // Apply saved app mode
        if (s.appMode === 'promoter' || s.appMode === 'user') {
          setAppMode(s.appMode)
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

        // Navigate to saved startup page (with legacy migration)
        if (s.startupPage && VALID_STARTUP_PAGES.has(s.startupPage)) {
          const migrated = migrateLegacyRoute(s.startupPage)
          setActiveTool(migrated.tool)
          if (migrated.subTab) {
            setSubTab(migrated.tool, migrated.subTab)
          }
        }
      })
      .catch(() => setShowWizard(false))
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

  // Loading state while checking onboarding status
  if (showWizard === null) {
    return (
      <div className="h-screen flex items-center justify-center bg-background text-foreground">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  // Show wizard if onboarding not completed
  if (showWizard) {
    return <SetupWizard onComplete={() => setShowWizard(false)} />
  }

  const renderPage = () => {
    // Guard promoter-only pages
    const appMode = useConfigStore.getState().appMode
    if (PROMOTER_ONLY_PAGES.has(activeTool) && appMode !== 'promoter') {
      return <HomePage />
    }

    switch (activeTool) {
      case 'home':
        return <HomePage />
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
      case 'api-admin':
        return <ApiAdminPage />
      default:
        return <HomePage />
    }
  }

  return (
    <div className="flex flex-col h-screen bg-background text-foreground">
      <ConnectionBanner />
      <ToastContainer />
      <CommandPalette />
      <div className="flex flex-1 overflow-hidden">
        <Sidebar />
        <main className="flex-1 overflow-hidden">
          <ErrorBoundary>
            {renderPage()}
          </ErrorBoundary>
        </main>
      </div>
      <StatusBar />
    </div>
  )
}

export default App
