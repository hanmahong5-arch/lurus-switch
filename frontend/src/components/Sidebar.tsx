import {
  Settings, LayoutDashboard, Wrench,
  CreditCard, Activity, BookOpen, FileText, Shield,
  Server, Layers, Key, Users, BarChart3, Box, Gift, Settings2,
  Terminal, Network, Package, Megaphone, Cpu,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useConfigStore, type ActiveTool } from '../stores/configStore'
import { useGatewayStore } from '../stores/gatewayStore'

const TOOL_PAGES: ActiveTool[] = [
  'claude', 'codex', 'gemini', 'picoclaw', 'nullclaw', 'zeroclaw', 'openclaw',
]

function isToolPage(t: ActiveTool): boolean {
  return TOOL_PAGES.includes(t)
}

// Utility nav items shown below the tools
const utilNav: { id: ActiveTool; i18nKey: string; icon: React.ComponentType<{ className?: string }>; color: string }[] = [
  { id: 'relay', i18nKey: 'nav.relay', icon: Network, color: 'text-sky-500' },
  { id: 'cli-runner', i18nKey: 'nav.cliRunner', icon: Terminal, color: 'text-gray-400' },
  { id: 'process', i18nKey: 'nav.process', icon: Activity, color: 'text-yellow-500' },
  { id: 'prompts', i18nKey: 'nav.prompts', icon: BookOpen, color: 'text-purple-400' },
  { id: 'documents', i18nKey: 'nav.documents', icon: FileText, color: 'text-teal-500' },
]

// Gateway section items — data-driven
const gatewayNav: { id: ActiveTool; i18nKey: string; icon: React.ComponentType<{ className?: string }>; color: string }[] = [
  { id: 'gateway', i18nKey: 'gateway.server', icon: Server, color: 'text-indigo-400' },
  { id: 'gateway-dashboard', i18nKey: 'gateway.dashboard', icon: BarChart3, color: 'text-emerald-400' },
  { id: 'gateway-channels', i18nKey: 'gateway.channels', icon: Layers, color: 'text-blue-400' },
  { id: 'gateway-tokens', i18nKey: 'gateway.tokens', icon: Key, color: 'text-yellow-400' },
  { id: 'gateway-models', i18nKey: 'gateway.models', icon: Box, color: 'text-orange-400' },
  { id: 'gateway-users', i18nKey: 'gateway.users', icon: Users, color: 'text-purple-400' },
  { id: 'gateway-redemptions', i18nKey: 'gateway.redemptions', icon: Gift, color: 'text-pink-400' },
  { id: 'gateway-logs', i18nKey: 'gateway.logs', icon: FileText, color: 'text-teal-400' },
  { id: 'gateway-subscriptions', i18nKey: 'gateway.subscriptions', icon: CreditCard, color: 'text-cyan-400' },
  { id: 'gateway-settings', i18nKey: 'gateway.gatewaySettings', icon: Settings2, color: 'text-gray-400' },
]

type NavButtonProps = {
  id: ActiveTool
  name: string
  icon: React.ComponentType<{ className?: string }>
  iconColor: string
  active: boolean
  onClick: () => void
}

function NavButton({ id, name, icon: Icon, iconColor, active, onClick }: NavButtonProps) {
  return (
    <button
      key={id}
      onClick={onClick}
      className={cn(
        'w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors',
        active
          ? 'bg-primary text-primary-foreground'
          : 'hover:bg-muted text-muted-foreground hover:text-foreground'
      )}
    >
      <Icon className={cn('h-5 w-5', !active && iconColor)} />
      {name}
    </button>
  )
}

// Pages that are only visible in promoter mode
export const PROMOTER_ONLY_PAGES: Set<string> = new Set([
  'gateway', 'gateway-dashboard', 'gateway-channels', 'gateway-tokens',
  'gateway-models', 'gateway-users', 'gateway-redemptions', 'gateway-logs',
  'gateway-subscriptions', 'gateway-settings', 'admin', 'promoter-hub',
])

export function Sidebar() {
  const { activeTool, setActiveTool, lastActiveTool, appMode } = useConfigStore()
  const { t } = useTranslation()
  const gatewayRunning = useGatewayStore((s) => s.status?.running ?? false)
  const isPromoter = appMode === 'promoter'

  return (
    <aside className="w-56 bg-muted/50 border-r border-border flex flex-col">
      {/* Logo / Title */}
      <div className="p-4 border-b border-border wails-drag">
        <h1 className="text-lg font-semibold">Lurus Switch</h1>
        <p className="text-xs text-muted-foreground">{t('app.subtitle')}</p>
      </div>

      {/* Navigation */}
      <nav className="flex-1 p-2 overflow-y-auto">
        <div className="space-y-1">
          {/* GY Products */}
          <NavButton
            id="gy-products"
            name={t('nav.gyProducts')}
            icon={Package}
            iconColor="text-violet-500"
            active={activeTool === 'gy-products'}
            onClick={() => setActiveTool('gy-products')}
          />

          {/* Dashboard */}
          <NavButton
            id="dashboard"
            name={t('nav.dashboard')}
            icon={LayoutDashboard}
            iconColor="text-purple-500"
            active={activeTool === 'dashboard'}
            onClick={() => setActiveTool('dashboard')}
          />

          {/* Switch Gateway Hub */}
          <NavButton
            id="switch-hub"
            name={t('nav.switchHub')}
            icon={Cpu}
            iconColor="text-cyan-500"
            active={activeTool === 'switch-hub'}
            onClick={() => setActiveTool('switch-hub')}
          />

          {/* Separator */}
          <div className="border-t border-border my-2" />

          {/* Single tool config entry */}
          <NavButton
            id={lastActiveTool as ActiveTool}
            name={t('nav.toolConfig')}
            icon={Wrench}
            iconColor="text-amber-500"
            active={isToolPage(activeTool)}
            onClick={() => setActiveTool(lastActiveTool as ActiveTool)}
          />

          {/* Separator */}
          <div className="border-t border-border my-2" />

          {/* Utility pages */}
          {utilNav.map((item) => (
            <NavButton
              key={item.id}
              id={item.id}
              name={t(item.i18nKey)}
              icon={item.icon}
              iconColor={item.color}
              active={activeTool === item.id}
              onClick={() => setActiveTool(item.id)}
            />
          ))}

          {/* Promoter-only sections */}
          {isPromoter && (
            <>
              {/* Separator */}
              <div className="border-t border-border my-2" />

              {/* Promoter Hub */}
              <NavButton
                id="promoter-hub"
                name={t('nav.promoterHub')}
                icon={Megaphone}
                iconColor="text-orange-500"
                active={activeTool === 'promoter-hub'}
                onClick={() => setActiveTool('promoter-hub')}
              />

              {/* Gateway section header */}
              <div className="px-3 py-1 text-xs font-semibold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                {t('gateway.title')}
                {gatewayRunning && <span className="h-1.5 w-1.5 rounded-full bg-green-500 inline-block" />}
              </div>

              {/* Gateway nav items — data-driven */}
              {gatewayNav.map((item) => (
                <NavButton
                  key={item.id}
                  id={item.id}
                  name={t(item.i18nKey)}
                  icon={item.icon}
                  iconColor={item.color}
                  active={activeTool === item.id}
                  onClick={() => setActiveTool(item.id)}
                />
              ))}
            </>
          )}
        </div>
      </nav>

      {/* Billing, Admin & Settings */}
      <div className="p-2 border-t border-border space-y-1">
        <NavButton
          id="billing"
          name={t('nav.billing')}
          icon={CreditCard}
          iconColor="text-emerald-500"
          active={activeTool === 'billing'}
          onClick={() => setActiveTool('billing')}
        />
        {isPromoter && (
          <NavButton
            id="admin"
            name={t('nav.admin')}
            icon={Shield}
            iconColor="text-red-500"
            active={activeTool === 'admin'}
            onClick={() => setActiveTool('admin')}
          />
        )}
        <NavButton
          id="settings"
          name={t('nav.settings')}
          icon={Settings}
          iconColor="text-muted-foreground"
          active={activeTool === 'settings'}
          onClick={() => setActiveTool('settings')}
        />
      </div>
    </aside>
  )
}
