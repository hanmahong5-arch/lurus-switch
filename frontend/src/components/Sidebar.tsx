import {
  Settings, Home, Wrench, Wallet, Briefcase,
  Megaphone, ShieldCheck, Radio,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useConfigStore, type ActiveTool } from '../stores/configStore'
import { useGatewayStore } from '../stores/gatewayStore'
import { HelpTip } from './HelpTip'

type NavButtonProps = {
  id: ActiveTool
  name: string
  icon: React.ComponentType<{ className?: string }>
  iconColor: string
  active: boolean
  onClick: () => void
  badge?: React.ReactNode
}

function NavButton({ id, name, icon: Icon, iconColor, active, onClick, badge }: NavButtonProps) {
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
      <span className="flex-1 text-left">{name}</span>
      {badge}
    </button>
  )
}

// Pages that are only visible in promoter mode
export const PROMOTER_ONLY_PAGES: Set<string> = new Set([
  'promotion', 'api-admin',
])

export function Sidebar() {
  const { activeTool, setActiveTool, appMode } = useConfigStore()
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
          {/* 1. Home */}
          <NavButton
            id="home"
            name={t('nav.home')}
            icon={Home}
            iconColor="text-purple-500"
            active={activeTool === 'home'}
            onClick={() => setActiveTool('home')}
          />

          {/* 2. Tools */}
          <NavButton
            id="tools"
            name={t('nav.tools')}
            icon={Wrench}
            iconColor="text-amber-500"
            active={activeTool === 'tools'}
            onClick={() => setActiveTool('tools')}
          />

          {/* 3. Gateway */}
          <div className="flex items-center">
            <div className="flex-1">
              <NavButton
                id="gateway"
                name={t('nav.gateway')}
                icon={Radio}
                iconColor="text-cyan-500"
                active={activeTool === 'gateway'}
                onClick={() => setActiveTool('gateway')}
                badge={gatewayRunning ? (
                  <span className="h-2 w-2 rounded-full bg-green-500 flex-shrink-0" />
                ) : undefined}
              />
            </div>
            <HelpTip
              titleKey="help.gateway.title"
              bodyKey="help.gateway.body"
              showFor={['beginner']}
              placement="right"
              size="sm"
            />
          </div>

          {/* 4. Workspace */}
          <div className="flex items-center">
            <div className="flex-1">
              <NavButton
                id="workspace"
                name={t('nav.workspace')}
                icon={Briefcase}
                iconColor="text-blue-500"
                active={activeTool === 'workspace'}
                onClick={() => setActiveTool('workspace')}
              />
            </div>
            <HelpTip
              titleKey="help.workspace.title"
              bodyKey="help.workspace.body"
              showFor={['beginner']}
              placement="right"
              size="sm"
            />
          </div>

          {/* 5. Account */}
          <NavButton
            id="account"
            name={t('nav.account')}
            icon={Wallet}
            iconColor="text-emerald-500"
            active={activeTool === 'account'}
            onClick={() => setActiveTool('account')}
          />

          {/* Promoter-only sections */}
          {isPromoter && (
            <>
              <div className="border-t border-border my-2" />

              {/* 7. Promotion */}
              <NavButton
                id="promotion"
                name={t('nav.promotion')}
                icon={Megaphone}
                iconColor="text-orange-500"
                active={activeTool === 'promotion'}
                onClick={() => setActiveTool('promotion')}
              />

              {/* 8. API Admin */}
              <NavButton
                id="api-admin"
                name={t('nav.apiAdmin')}
                icon={ShieldCheck}
                iconColor="text-red-500"
                active={activeTool === 'api-admin'}
                onClick={() => setActiveTool('api-admin')}
              />
            </>
          )}
        </div>
      </nav>

      {/* Settings (bottom) */}
      <div className="p-2 border-t border-border">
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
