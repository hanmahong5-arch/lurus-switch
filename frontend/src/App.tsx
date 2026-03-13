import { useEffect, useState } from 'react'
import { Loader2 } from 'lucide-react'
import './style.css'
import { Sidebar } from './components/Sidebar'
import { StatusBar } from './components/StatusBar'
import { ErrorBoundary } from './components/ErrorBoundary'
import { SetupWizard } from './components/SetupWizard'
import { DashboardPage } from './pages/DashboardPage'
import { ToolConfigPage } from './pages/ToolConfigPage'
import { BillingPage } from './pages/BillingPage'
import { SettingsPage } from './pages/SettingsPage'
import { ProcessPage } from './pages/ProcessPage'
import { PromptLibraryPage } from './pages/PromptLibraryPage'
import { DocumentPage } from './pages/DocumentPage'
import { AdminPage } from './pages/AdminPage'
import { GatewayPage } from './pages/GatewayPage'
import { GatewayDashboardPage } from './pages/GatewayDashboardPage'
import { GatewayChannelPage } from './pages/GatewayChannelPage'
import { GatewayTokenPage } from './pages/GatewayTokenPage'
import { GatewayModelPage } from './pages/GatewayModelPage'
import { GatewayUserPage } from './pages/GatewayUserPage'
import { GatewayRedemptionPage } from './pages/GatewayRedemptionPage'
import { GatewayLogPage } from './pages/GatewayLogPage'
import { GatewaySubscriptionPage } from './pages/GatewaySubscriptionPage'
import { GatewaySettingsPage } from './pages/GatewaySettingsPage'
import { useConfigStore, type ActiveTool } from './stores/configStore'
import { useGatewayStore } from './stores/gatewayStore'
import { GetAppSettings, GetServerStatus, GetServerAdminToken } from '../wailsjs/go/main/App'
import i18n from './i18n'

// Pages that can be used as a startup page
const VALID_STARTUP_PAGES: ReadonlySet<string> = new Set([
  'dashboard', 'claude', 'codex', 'gemini', 'picoclaw', 'nullclaw',
])

function App() {
  const { activeTool, setActiveTool } = useConfigStore()
  const [showWizard, setShowWizard] = useState<boolean | null>(null)
  const { startPolling, stopPolling } = useGatewayStore()

  useEffect(() => {
    GetAppSettings()
      .then((s) => {
        setShowWizard(!s.onboardingCompleted)

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

        // Navigate to saved startup page if it is a valid page
        if (s.startupPage && VALID_STARTUP_PAGES.has(s.startupPage)) {
          setActiveTool(s.startupPage as ActiveTool)
        }
      })
      .catch(() => setShowWizard(false))
  }, [])

  // Start global gateway status polling when the main app mounts.
  useEffect(() => {
    startPolling(
      () => GetServerStatus() as ReturnType<typeof GetServerStatus>,
      () => GetServerAdminToken(),
    )
    return () => stopPolling()
  }, [])

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
    switch (activeTool) {
      case 'dashboard':
        return <DashboardPage />
      case 'claude':
      case 'codex':
      case 'gemini':
      case 'picoclaw':
      case 'nullclaw':
      case 'zeroclaw':
      case 'openclaw':
        return <ToolConfigPage />
      case 'billing':
        return <BillingPage />
      case 'settings':
        return <SettingsPage />
      case 'process':
        return <ProcessPage />
      case 'prompts':
        return <PromptLibraryPage />
      case 'documents':
        return <DocumentPage />
      case 'admin':
        return <AdminPage />
      case 'gateway':
        return <GatewayPage />
      case 'gateway-dashboard':
        return <GatewayDashboardPage />
      case 'gateway-channels':
        return <GatewayChannelPage />
      case 'gateway-tokens':
        return <GatewayTokenPage />
      case 'gateway-models':
        return <GatewayModelPage />
      case 'gateway-users':
        return <GatewayUserPage />
      case 'gateway-redemptions':
        return <GatewayRedemptionPage />
      case 'gateway-logs':
        return <GatewayLogPage />
      case 'gateway-subscriptions':
        return <GatewaySubscriptionPage />
      case 'gateway-settings':
        return <GatewaySettingsPage />
      default:
        return <DashboardPage />
    }
  }

  return (
    <div className="flex flex-col h-screen bg-background text-foreground">
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
