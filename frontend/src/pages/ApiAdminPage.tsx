import { useTranslation } from 'react-i18next'
import {
  Server, BarChart3, Layers, Key, Box, Users,
  Gift, FileText, CreditCard, Settings2, Shield,
} from 'lucide-react'
import { useConfigStore, type ApiAdminSubTab } from '../stores/configStore'
import { TabBar } from '../components/TabBar'
import { GatewayRequiredGuard } from '../components/GatewayRequiredGuard'
import { GatewayPage as LegacyGatewayPage } from './GatewayPage'
import { GatewayDashboardPage } from './GatewayDashboardPage'
import { GatewayChannelPage } from './GatewayChannelPage'
import { GatewayTokenPage } from './GatewayTokenPage'
import { GatewayModelPage } from './GatewayModelPage'
import { GatewayUserPage } from './GatewayUserPage'
import { GatewayRedemptionPage } from './GatewayRedemptionPage'
import { GatewayLogPage } from './GatewayLogPage'
import { GatewaySubscriptionPage } from './GatewaySubscriptionPage'
import { GatewaySettingsPage } from './GatewaySettingsPage'
import { AdminPage } from './AdminPage'

export function ApiAdminPage() {
  const { t } = useTranslation()
  const { getSubTab, setSubTab } = useConfigStore()
  const activeTab = getSubTab('api-admin', 'server') as ApiAdminSubTab

  const tabs = [
    { id: 'server', label: t('gateway.server'), icon: Server },
    { id: 'dashboard', label: t('gateway.dashboard'), icon: BarChart3 },
    { id: 'channels', label: t('gateway.channels'), icon: Layers },
    { id: 'tokens', label: t('gateway.tokens'), icon: Key },
    { id: 'models', label: t('gateway.models'), icon: Box },
    { id: 'users', label: t('gateway.users'), icon: Users },
    { id: 'redemptions', label: t('gateway.redemptions'), icon: Gift },
    { id: 'logs', label: t('gateway.logs'), icon: FileText },
    { id: 'subscriptions', label: t('gateway.subscriptions'), icon: CreditCard },
    { id: 'admin-settings', label: t('gateway.gatewaySettings'), icon: Settings2 },
    { id: 'system', label: t('nav.admin'), icon: Shield },
  ]

  const renderContent = () => {
    switch (activeTab) {
      case 'server':
        return <LegacyGatewayPage />
      case 'dashboard':
        return <GatewayRequiredGuard><GatewayDashboardPage /></GatewayRequiredGuard>
      case 'channels':
        return <GatewayRequiredGuard><GatewayChannelPage /></GatewayRequiredGuard>
      case 'tokens':
        return <GatewayRequiredGuard><GatewayTokenPage /></GatewayRequiredGuard>
      case 'models':
        return <GatewayRequiredGuard><GatewayModelPage /></GatewayRequiredGuard>
      case 'users':
        return <GatewayRequiredGuard><GatewayUserPage /></GatewayRequiredGuard>
      case 'redemptions':
        return <GatewayRequiredGuard><GatewayRedemptionPage /></GatewayRequiredGuard>
      case 'logs':
        return <GatewayRequiredGuard><GatewayLogPage /></GatewayRequiredGuard>
      case 'subscriptions':
        return <GatewayRequiredGuard><GatewaySubscriptionPage /></GatewayRequiredGuard>
      case 'admin-settings':
        return <GatewayRequiredGuard><GatewaySettingsPage /></GatewayRequiredGuard>
      case 'system':
        return <AdminPage />
      default:
        return <LegacyGatewayPage />
    }
  }

  return (
    <div className="h-full flex flex-col overflow-hidden">
      <div className="overflow-x-auto">
        <TabBar
          tabs={tabs}
          activeTab={activeTab}
          onTabChange={(id) => setSubTab('api-admin', id)}
        />
      </div>
      <div className="flex-1 overflow-hidden">
        {renderContent()}
      </div>
    </div>
  )
}
