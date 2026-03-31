import { useTranslation } from 'react-i18next'
import { Link2, CreditCard } from 'lucide-react'
import { useConfigStore, type AccountSubTab } from '../stores/configStore'
import { TabBar } from '../components/TabBar'
import { BillingPage } from './BillingPage'
import { AuthLoginPanel } from '../components/AuthLoginPanel'
import { ProxyConfigPanel } from '../components/ProxyConfigPanel'
import { AccountPanel } from '../components/AccountPanel'
import { DashboardQuotaWidget } from '../components/DashboardQuotaWidget'
import { useDashboardStore } from '../stores/dashboardStore'
import { useAuthStore } from '../stores/authStore'
import { useToastStore } from '../stores/toastStore'
import { errorToast } from '../lib/errorToast'
import { GetProxySettings, SaveProxySettings, ConfigureAllProxy } from '../../wailsjs/go/main/App'
import { proxy } from '../../wailsjs/go/models'
import { useEffect, useCallback } from 'react'
import type { ProxySettings } from '../stores/dashboardStore'

function ConnectionTab() {
  const { t } = useTranslation()
  const { proxySettings, proxySaving, proxyConfiguring, setProxySettings, setProxySaving, setProxyConfiguring } = useDashboardStore()
  const { authState } = useAuthStore()
  const toast = useToastStore((s) => s.addToast)

  useEffect(() => {
    GetProxySettings().then(setProxySettings).catch(() => {})
  }, [setProxySettings])

  const handleSaveProxy = useCallback(async (settings: ProxySettings) => {
    setProxySaving(true)
    try {
      await SaveProxySettings(proxy.ProxySettings.createFrom(settings))
      setProxySettings(settings)
      toast('success', t('dashboard.proxySaved'))
    } catch (err) {
      errorToast(toast, err, { t })
    } finally {
      setProxySaving(false)
    }
  }, [t, toast, setProxySettings, setProxySaving])

  const handleConfigureAll = useCallback(async () => {
    setProxyConfiguring(true)
    try {
      await SaveProxySettings(proxy.ProxySettings.createFrom(proxySettings))
      const errors = await ConfigureAllProxy()
      if (Object.keys(errors).length > 0) {
        const failed = Object.entries(errors).map(([tool, e]) => `${tool}: ${e}`).join('; ')
        toast('warning', failed)
      } else {
        toast('success', t('dashboard.proxyConfigured'))
      }
    } catch (err) {
      errorToast(toast, err, { t })
    } finally {
      setProxyConfiguring(false)
    }
  }, [t, toast, proxySettings, setProxyConfiguring])

  const isLoggedIn = authState.is_logged_in

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-3xl mx-auto p-6 space-y-6">
        {/* OIDC Login — primary authentication method */}
        <AuthLoginPanel />
        <DashboardQuotaWidget />
        {/* Proxy config — shown when not logged in, or always available for manual override */}
        {!isLoggedIn && (
          <ProxyConfigPanel
            settings={proxySettings}
            saving={proxySaving}
            configuring={proxyConfiguring}
            onSave={handleSaveProxy}
            onConfigureAll={handleConfigureAll}
          />
        )}
      </div>
    </div>
  )
}

export function AccountPage() {
  const { t } = useTranslation()
  const { getSubTab, setSubTab } = useConfigStore()
  const activeTab = getSubTab('account', 'connection') as AccountSubTab

  const tabs = [
    { id: 'connection', label: t('home.connection'), icon: Link2 },
    { id: 'billing', label: t('nav.billing'), icon: CreditCard },
  ]

  return (
    <div className="h-full flex flex-col overflow-hidden">
      <TabBar
        tabs={tabs}
        activeTab={activeTab}
        onTabChange={(id) => setSubTab('account', id)}
      />
      <div className="flex-1 overflow-hidden">
        {activeTab === 'connection' && <ConnectionTab />}
        {activeTab === 'billing' && <BillingPage />}
      </div>
    </div>
  )
}
