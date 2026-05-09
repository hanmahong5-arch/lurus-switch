import { useEffect, useState } from 'react'
import {
  Settings, Home, Wrench, Wallet, Briefcase,
  Megaphone, Radio, Bot, Package,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useConfigStore, type ActiveTool } from '../stores/configStore'
import { useGatewayStore } from '../stores/gatewayStore'
import { HelpTip } from './HelpTip'
import { GetAppSettings } from '../../wailsjs/go/main/App'

type NavButtonProps = {
  id: ActiveTool
  name: string
  icon: React.ComponentType<{ className?: string }>
  iconColor: string
  active: boolean
  onClick: () => void
  badge?: React.ReactNode
}

function NavSectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60 px-3 pt-3 pb-1 first:pt-0">
      {children}
    </p>
  )
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

// Pages restricted to Reseller mode (经销商运营台).
// In Personal / EndUser modes these are hidden from the sidebar AND blocked
// by the route guard in App.tsx.
export const RESELLER_ONLY_PAGES: Set<string> = new Set([
  'promotion', 'packager',
])

// Backwards-compat alias used by App.tsx until the rename propagates.
export const PROMOTER_ONLY_PAGES = RESELLER_ONLY_PAGES

// Pages restricted to Personal mode (Agent Fleet — frozen for Phase A-C, see ADR-020).
export const PERSONAL_ONLY_PAGES: Set<string> = new Set([
  'agents',
])

// Pages visible in EndUser mode (white-label client). Anything else is hidden.
// Kept intentionally narrow to match the simplified EndUser dashboard.
export const ENDUSER_VISIBLE_PAGES: Set<string> = new Set([
  'home', 'tools', 'account', 'settings',
])

interface BrandInfo {
  name: string
  logoBase64: string
}

export function Sidebar() {
  const { activeTool, setActiveTool, appMode } = useConfigStore()
  const { t } = useTranslation()
  const gatewayRunning = useGatewayStore((s) => s.status?.running ?? false)
  const isReseller = appMode === 'reseller'
  const isEndUser = appMode === 'enduser'
  const showAgents = appMode === 'personal' // Agent Fleet is Personal-mode only

  // White-label branding pulled from GetAppSettings on mount. Empty
  // strings → render the stock Lurus mark/title. EndUser mode that came
  // from a properly-signed sidecar will have these populated.
  const [brand, setBrand] = useState<BrandInfo>({ name: '', logoBase64: '' })
  useEffect(() => {
    GetAppSettings()
      .then((s) => {
        const info: BrandInfo = {
          name: ((s as any).brandName as string) || '',
          logoBase64: ((s as any).brandLogoBase64 as string) || '',
        }
        setBrand(info)
      })
      .catch(() => { /* fall back to stock branding */ })
  }, [])

  return (
    <aside className="w-56 bg-muted/50 border-r border-border flex flex-col">
      {/* Logo / Title */}
      <div className="p-4 border-b border-border wails-drag">
        <div className="flex items-center gap-2 mb-0.5">
          {brand.logoBase64 ? (
            <img
              src={brand.logoBase64.startsWith('data:')
                ? brand.logoBase64
                : `data:image/png;base64,${brand.logoBase64}`}
              alt={brand.name || 'brand'}
              className="h-5 w-5 object-contain"
            />
          ) : (
            /* Geometric whale SVG mark */
            <svg width="22" height="22" viewBox="0 0 22 22" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
              <path d="M3 14 Q2 8 7 6 Q10 5 12 7 L18 5 Q20 5 20 8 L19 13 Q18 16 15 16 L13 16 Q12 18 10 19 L8 20 Q7 20 7.5 18.5 L8 17 Q5 17 3 14 Z" fill="currentColor" className="text-primary"/>
              <circle cx="16" cy="9" r="1.2" fill="white" opacity="0.85"/>
              <path d="M19 8 Q21 6 20 4" stroke="white" strokeWidth="1" strokeLinecap="round" opacity="0.6"/>
            </svg>
          )}
          <h1 className="text-lg font-semibold">{brand.name || 'Lurus Switch'}</h1>
        </div>
        <div className="flex items-center gap-1.5">
          <p className="text-xs text-muted-foreground">{t('app.subtitle')}</p>
          {appMode !== 'unset' && (
            <span
              className={
                'text-[10px] px-1.5 py-0.5 rounded-sm font-medium ' +
                (appMode === 'personal'
                  ? 'bg-blue-500/15 text-blue-400'
                  : appMode === 'reseller'
                  ? 'bg-purple-500/15 text-purple-400'
                  : 'bg-emerald-500/15 text-emerald-400')
              }
              title={t(`mode.${appMode}.desc`, '')}
            >
              {t(`mode.${appMode}.label`, appMode)}
            </span>
          )}
        </div>
      </div>

      {/* Navigation — grouped into 3 coarse sections so users navigate by
          purpose, not by individual feature. Section labels stay visible
          even with one entry, since they're the primary signposting. */}
      <nav className="flex-1 p-2 overflow-y-auto">
        <div className="space-y-1">
          {/* Group 1: Overview — landing + agent fleet */}
          <NavSectionLabel>{t('nav.section.overview', '总览 · Overview')}</NavSectionLabel>
          <NavButton
            id="home"
            name={t('nav.home')}
            icon={Home}
            iconColor="text-purple-500"
            active={activeTool === 'home'}
            onClick={() => setActiveTool('home')}
          />
          {showAgents && (
            <NavButton
              id="agents"
              name={t('nav.agents', 'Agents')}
              icon={Bot}
              iconColor="text-violet-500"
              active={activeTool === 'agents'}
              onClick={() => setActiveTool('agents')}
            />
          )}

          {/* Group 2: Configure — Tools + Gateway. EndUser hides Gateway. */}
          <NavSectionLabel>{t('nav.section.configure', '配置 · Configure')}</NavSectionLabel>
          <NavButton
            id="tools"
            name={t('nav.tools')}
            icon={Wrench}
            iconColor="text-amber-500"
            active={activeTool === 'tools'}
            onClick={() => setActiveTool('tools')}
          />
          {!isEndUser && (
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
          )}

          {/* Group 3: Work — Workspace + Account. EndUser hides Workspace. */}
          <NavSectionLabel>{t('nav.section.work', '工作 · Work')}</NavSectionLabel>
          {!isEndUser && (
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
          )}
          <NavButton
            id="account"
            name={t('nav.account')}
            icon={Wallet}
            iconColor="text-emerald-500"
            active={activeTool === 'account'}
            onClick={() => setActiveTool('account')}
          />

          {/* Reseller-only group: distribution / billing / packaging. */}
          {isReseller && (
            <>
              <NavSectionLabel>{t('nav.section.reseller', '经销 · Reseller')}</NavSectionLabel>

              {/* 7. Promotion */}
              <NavButton
                id="promotion"
                name={t('nav.promotion')}
                icon={Megaphone}
                iconColor="text-orange-500"
                active={activeTool === 'promotion'}
                onClick={() => setActiveTool('promotion')}
              />

              {/* api-admin entry removed — its 11 sub-tabs are merged into
                  the unified Gateway page (see NewGatewayPage). */}

              {/* 8. Packager (white-label EndUser builds) */}
              <NavButton
                id="packager"
                name={t('nav.packager', '白标打包')}
                icon={Package}
                iconColor="text-fuchsia-500"
                active={activeTool === 'packager'}
                onClick={() => setActiveTool('packager')}
              />
            </>
          )}
        </div>
      </nav>

      {/* Settings (bottom) */}
      <div className="p-2 border-t border-border space-y-2">
        <NavButton
          id="settings"
          name={t('nav.settings')}
          icon={Settings}
          iconColor="text-muted-foreground"
          active={activeTool === 'settings'}
          onClick={() => setActiveTool('settings')}
        />
        {/* Keyboard shortcut hints */}
        <p className="text-[10px] text-muted-foreground/50 text-center leading-relaxed px-1">
          Ctrl+1~5 {t('nav.switchPage', 'switch page')} &middot; Ctrl+S {t('nav.save', 'save')}
        </p>
      </div>
    </aside>
  )
}
