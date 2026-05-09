import { useTranslation } from 'react-i18next'
import {
  Settings, BarChart3, Smartphone, Network,
  Layers, Key, Box, Users, Gift, FileText, CreditCard, Settings2, Shield,
} from 'lucide-react'
import { useConfigStore, type GatewaySubTab } from '../stores/configStore'
import { cn } from '../lib/utils'
import { SwitchHubPage } from './SwitchHubPage'
import { RelayPage } from './RelayPage'
import { GatewayRequiredGuard } from '../components/GatewayRequiredGuard'
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

// Single Gateway page covering basic gateway ops (control / usage / apps /
// relay) plus the full newapi admin console (channels / tokens / models / …
// / system). Admin and Root sections only appear in Reseller mode where the
// user is operating their own newapi instance.

type TabDef = {
  id: GatewaySubTab
  label: string
  icon: React.ComponentType<{ className?: string }>
}

export function NewGatewayPage() {
  const { t } = useTranslation()
  const { getSubTab, setSubTab, appMode } = useConfigStore()
  const activeTab = getSubTab('gateway', 'control') as GatewaySubTab

  // Reseller mode owns the upstream newapi instance — show admin+root tabs.
  // Personal mode points at Lurus Cloud's hub; admin tabs would be useless.
  // EndUser mode never reaches this page (gated upstream).
  const showAdmin = appMode === 'reseller'
  // TODO(role): split admin vs root once newhub returns user.role and the
  // value is plumbed into Switch. For now, Reseller users see both.
  const showRoot = appMode === 'reseller'

  const basicTabs: TabDef[] = [
    { id: 'control', label: t('home.gwControl'), icon: Settings },
    { id: 'usage', label: t('home.gwUsage'), icon: BarChart3 },
    { id: 'apps', label: t('home.gwApps'), icon: Smartphone },
    { id: 'relay', label: t('nav.relay'), icon: Network },
  ]
  const adminTabs: TabDef[] = [
    { id: 'dashboard', label: t('gateway.dashboard'), icon: BarChart3 },
    { id: 'channels', label: t('gateway.channels'), icon: Layers },
    { id: 'tokens', label: t('gateway.tokens'), icon: Key },
    { id: 'models', label: t('gateway.models'), icon: Box },
    { id: 'users', label: t('gateway.users'), icon: Users },
    { id: 'redemptions', label: t('gateway.redemptions'), icon: Gift },
    { id: 'logs', label: t('gateway.logs'), icon: FileText },
    { id: 'subscriptions', label: t('gateway.subscriptions'), icon: CreditCard },
    { id: 'admin-settings', label: t('gateway.gatewaySettings'), icon: Settings2 },
  ]
  const rootTabs: TabDef[] = [
    { id: 'system', label: t('gateway.system', t('nav.admin')), icon: Shield },
  ]

  const renderTab = (tab: TabDef) => {
    const Icon = tab.icon
    const isActive = activeTab === tab.id
    return (
      <button
        key={tab.id}
        onClick={() => setSubTab('gateway', tab.id)}
        className={cn(
          'flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-t-md transition-colors whitespace-nowrap',
          isActive
            ? 'border-b-2 border-primary text-foreground bg-background'
            : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
        )}
      >
        <Icon className="h-4 w-4" />
        {tab.label}
      </button>
    )
  }

  const renderContent = () => {
    switch (activeTab) {
      case 'control':
      case 'usage':
      case 'apps':
        return <SwitchHubPage section={activeTab} />
      case 'relay':
        return <RelayPage />
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
        return <SwitchHubPage section="control" />
    }
  }

  return (
    <div className="h-full flex flex-col overflow-hidden">
      <div className="overflow-x-auto border-b border-border px-4 pt-3">
        <div className="flex gap-1 items-end min-w-fit">
          {basicTabs.map(renderTab)}
          {showAdmin && (
            <>
              <span
                className="self-center mx-2 text-[10px] font-medium uppercase tracking-wider text-muted-foreground/60"
                title={t('gateway.adminGroupHint', '管理员功能 · 仅 Reseller 自部署 newapi 时生效')}
              >
                ⋮
              </span>
              {adminTabs.map(renderTab)}
            </>
          )}
          {showRoot && (
            <>
              <span
                className="self-center mx-2 text-[10px] font-medium uppercase tracking-wider text-muted-foreground/60"
                title={t('gateway.rootGroupHint', 'Root 权限 · 全局选项与认证')}
              >
                ⋮
              </span>
              {rootTabs.map(renderTab)}
            </>
          )}
        </div>
      </div>
      <div className="flex-1 overflow-hidden">{renderContent()}</div>
    </div>
  )
}
