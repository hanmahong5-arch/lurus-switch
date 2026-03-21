import { useTranslation } from 'react-i18next'
import { Settings, BarChart3, Smartphone, Network } from 'lucide-react'
import { useConfigStore, type GatewaySubTab } from '../stores/configStore'
import { TabBar } from '../components/TabBar'
import { SwitchHubPage } from './SwitchHubPage'
import { RelayPage } from './RelayPage'

// Gateway page — 4 tabs: Control, Usage, Apps, Relay
// Control + Usage + Apps all come from SwitchHubPage sections.
// Relay is the full RelayPage.

export function NewGatewayPage() {
  const { t } = useTranslation()
  const { getSubTab, setSubTab } = useConfigStore()
  const activeTab = getSubTab('gateway', 'control') as GatewaySubTab

  const tabs = [
    { id: 'control', label: t('home.gwControl'), icon: Settings },
    { id: 'usage', label: t('home.gwUsage'), icon: BarChart3 },
    { id: 'apps', label: t('home.gwApps'), icon: Smartphone },
    { id: 'relay', label: t('nav.relay'), icon: Network },
  ]

  return (
    <div className="h-full flex flex-col overflow-hidden">
      <TabBar
        tabs={tabs}
        activeTab={activeTab}
        onTabChange={(id) => setSubTab('gateway', id)}
      />
      <div className="flex-1 overflow-hidden">
        {activeTab === 'relay' ? (
          <RelayPage />
        ) : (
          // SwitchHubPage contains all three sections (control, usage, apps).
          // We pass the activeTab to it so it can filter if needed, but for now
          // it renders the full page (the sections are already organized).
          <SwitchHubPage />
        )}
      </div>
    </div>
  )
}
