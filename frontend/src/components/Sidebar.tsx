import { useEffect, useState } from 'react'
import {
  Settings, Home, Wrench, Wallet, Briefcase,
  Megaphone, Radio, Bot, Package, Shield, Building2, BookOpen, Coins,
  MessageSquare, Activity, FileText, BookMarked, Network, ListTree, Compass,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useConfigStore, type ActiveTool } from '../stores/configStore'
import { useGatewayStore } from '../stores/gatewayStore'
import { HelpTip } from './HelpTip'
import { AccountSummaryCard } from './account/AccountSummaryCard'
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

// NavSection wraps a group with a label + (optional) one-liner description
// and a faint top divider — gives users a visual cue that "tools that go
// together visually go together" without adding heavy borders or boxes.
function NavSection({ label, hint, children }: {
  label: string
  hint?: string
  children: React.ReactNode
}) {
  return (
    <div className="pt-3 first:pt-0">
      <div className="px-3 pb-1">
        <p className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground/70">
          [ {label.toUpperCase()} ]
        </p>
        {hint && (
          <p className="text-[10px] text-muted-foreground/50 leading-tight mt-0.5">
            {hint}
          </p>
        )}
      </div>
      {children}
    </div>
  )
}

function NavButton({ id, name, icon: Icon, iconColor, active, onClick, badge }: NavButtonProps) {
  return (
    <button
      key={id}
      onClick={onClick}
      className={cn(
        'w-full flex items-center gap-3 px-3 py-2 rounded-md transition-all duration-150',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary',
        active
          ? 'bg-primary/10 text-primary border-l-2 border-l-primary -ml-px font-mono text-xs tracking-[0.06em]'
          : 'hover:bg-muted text-muted-foreground hover:text-foreground text-sm font-medium',
      )}
    >
      <Icon className={cn('h-5 w-5', !active && iconColor)} />
      <span className="flex-1 text-left">{active ? `[ ${name.toUpperCase()} ]` : name}</span>
      {badge}
    </button>
  )
}

// Sidebar shortcut into an existing sub-tab. Smaller (no font-mono active
// styling, indented), so the parent NavButton stays the visual primary and
// these read as "jump-into" tools rather than competing top-level pages.
function SubNavButton({ name, icon: Icon, iconColor, active, onClick }: {
  name: string
  icon: React.ComponentType<{ className?: string }>
  iconColor: string
  active: boolean
  onClick: () => void
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        'w-full flex items-center gap-2.5 pl-7 pr-3 py-1.5 rounded-md transition-all duration-150',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary',
        active
          ? 'bg-primary/8 text-primary'
          : 'hover:bg-muted/50 text-muted-foreground/80 hover:text-foreground',
      )}
    >
      <Icon className={cn('h-3.5 w-3.5', !active && iconColor)} />
      <span className="flex-1 text-left text-xs">{name}</span>
    </button>
  )
}

// Pages restricted to Reseller mode (经销商运营台).
// In Personal / EndUser modes these are hidden from the sidebar AND blocked
// by the route guard in App.tsx.
export const RESELLER_ONLY_PAGES: Set<string> = new Set([
  'promotion', 'packager',
])

// Pages available to both Reseller and Enterprise admins. DLP, org
// chart, agent template gallery, and chargeback are governance /
// admin tools that belong with the operating-system layer, not
// Personal mode.
export const ADMIN_PAGES: Set<string> = new Set([
  'dlp', 'orgchart', 'agent-templates', 'chargeback',
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
  const { activeTool, setActiveTool, appMode, getSubTab, setSubTab } = useConfigStore()
  const { t } = useTranslation()
  // Active sub-tabs for shortcut highlighting.
  const workspaceSubTab = getSubTab('workspace', 'prompts')
  const gatewaySubTab = getSubTab('gateway', 'control')
  const gotoWorkspace = (sub: 'prompts' | 'context' | 'process') => {
    setActiveTool('workspace')
    setSubTab('workspace', sub)
  }
  const gotoGatewayRelay = () => {
    setActiveTool('gateway')
    setSubTab('gateway', 'relay')
  }
  const gotoHomeEcosystem = () => {
    // GYProductsPage content is already inlined at the bottom of HomePage —
    // jump there and let the user scroll. Avoids carving out a new
    // top-level ActiveTool just for the ecosystem grid.
    setActiveTool('home')
  }
  const gatewayRunning = useGatewayStore((s) => s.status?.running ?? false)
  const isReseller = appMode === 'reseller'
  const isEnterprise = appMode === 'enterprise'
  const isEndUser = appMode === 'enduser'
  const showAgents = appMode === 'personal' // Agent Fleet is Personal-mode only
  // Admin section visible to Reseller (operating their channel) and
  // Enterprise (managing their internal users) — both need governance.
  const showAdminSection = isReseller || isEnterprise

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
    <aside className="w-56 bg-card-recessed border-r border-border flex flex-col">
      {/* Logo / Title */}
      <div className="p-4 border-b border-rule-strong wails-drag">
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
          <p className="text-xs text-muted-foreground font-mono">{t('app.subtitle')}</p>
          {appMode !== 'unset' && (
            <span
              className={
                'font-mono text-[10px] px-1.5 py-0.5 rounded-sm uppercase tracking-[0.12em] ' +
                (appMode === 'personal'
                  ? 'bg-blue-500/15 text-blue-400'
                  : appMode === 'reseller'
                  ? 'bg-primary/15 text-primary'
                  : 'bg-emerald-500/15 text-emerald-400')
              }
              title={t(`mode.${appMode}.desc`, '')}
            >
              [ {t(`mode.${appMode}.label`, appMode).toUpperCase()} ]
            </span>
          )}
        </div>
      </div>

      {/* Account summary card — progressive disclosure: click to reveal the
          full Popover (wallet / usage / subscription / service). Replaces the
          earlier "no account info anywhere in Sidebar" gap. */}
      <AccountSummaryCard />

      {/* Navigation — grouped into 3 coarse sections so users navigate by
          purpose, not by individual feature. Section labels stay visible
          even with one entry, since they're the primary signposting. */}
      <nav className="flex-1 p-2 overflow-y-auto">
        <div className="space-y-0.5">
          {/* Group 1: Overview — landing + agent fleet */}
          <NavSection
            label={t('nav.section.overview', '总览 · Overview')}
            hint={t('nav.section.overviewHint', '看现在的状态')}
          >
          <NavButton
            id="home"
            name={t('nav.home')}
            icon={Home}
            iconColor="text-purple-500"
            active={activeTool === 'home'}
            onClick={() => setActiveTool('home')}
          />
          {!isEndUser && (
            <NavButton
              id="live"
              name={t('nav.live', '实时观察')}
              icon={Activity}
              iconColor="text-[var(--lt-accent,#FF5D1F)]"
              active={activeTool === 'live'}
              onClick={() => setActiveTool('live')}
            />
          )}
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
          {!isEndUser && (
            <NavButton
              id="conversations"
              name={t('nav.conversations', '会话')}
              icon={MessageSquare}
              iconColor="text-violet-400"
              active={activeTool === 'conversations'}
              onClick={() => setActiveTool('conversations')}
            />
          )}
          </NavSection>

          {/* Group 2: Configure — Tools + Gateway. EndUser hides Gateway. */}
          <NavSection
            label={t('nav.section.configure', '配置 · Configure')}
            hint={t('nav.section.configureHint', '装 CLI 工具、配上游网关')}
          >
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
                  <span className="h-2 w-2 rounded-full bg-emerald-400 animate-pulse flex-shrink-0" />
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
          {!isEndUser && (
            <SubNavButton
              name={t('nav.shortcut.relay', '中转规则')}
              icon={Network}
              iconColor="text-cyan-400"
              active={activeTool === 'gateway' && gatewaySubTab === 'relay'}
              onClick={gotoGatewayRelay}
            />
          )}
          </NavSection>

          {/* Group 3: Work — Workspace + Account. EndUser hides Workspace. */}
          <NavSection
            label={t('nav.section.work', '工作 · Work')}
            hint={t('nav.section.workHint', '日常使用 + 账户')}
          >
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
          {!isEndUser && (
            <>
              <SubNavButton
                name={t('nav.shortcut.docs', '上下文文件')}
                icon={FileText}
                iconColor="text-blue-400"
                active={activeTool === 'workspace' && workspaceSubTab === 'context'}
                onClick={() => gotoWorkspace('context')}
              />
              <SubNavButton
                name={t('nav.shortcut.prompts', '提示词库')}
                icon={BookMarked}
                iconColor="text-violet-400"
                active={activeTool === 'workspace' && workspaceSubTab === 'prompts'}
                onClick={() => gotoWorkspace('prompts')}
              />
              <SubNavButton
                name={t('nav.shortcut.processes', '进程监控')}
                icon={ListTree}
                iconColor="text-amber-400"
                active={activeTool === 'workspace' && workspaceSubTab === 'process'}
                onClick={() => gotoWorkspace('process')}
              />
            </>
          )}
          <NavButton
            id="account"
            name={t('nav.account')}
            icon={Wallet}
            iconColor="text-emerald-500"
            active={activeTool === 'account'}
            onClick={() => setActiveTool('account')}
          />
          </NavSection>

          {/* Admin group — DLP / org chart / agent gallery. Visible to
              Reseller and Enterprise; both need governance, just for
              different audiences (channel vs internal employees). */}
          {showAdminSection && (
            <NavSection
              label={t('nav.section.admin', '治理 · Admin')}
              hint={t('nav.section.adminHint', '组织、合规、模板')}
            >
              <NavButton
                id="dlp"
                name={t('nav.dlp', 'DLP 数据保护')}
                icon={Shield}
                iconColor="text-blue-500"
                active={activeTool === 'dlp'}
                onClick={() => setActiveTool('dlp')}
              />
              <NavButton
                id="orgchart"
                name={t('nav.orgchart', '组织架构')}
                icon={Building2}
                iconColor="text-amber-500"
                active={activeTool === 'orgchart'}
                onClick={() => setActiveTool('orgchart')}
              />
              <NavButton
                id="agent-templates"
                name={t('nav.agentTemplates', 'Agent 模板库')}
                icon={BookOpen}
                iconColor="text-violet-500"
                active={activeTool === 'agent-templates'}
                onClick={() => setActiveTool('agent-templates')}
              />
              <NavButton
                id="chargeback"
                name={t('nav.chargeback', '成本归集')}
                icon={Coins}
                iconColor="text-amber-500"
                active={activeTool === 'chargeback'}
                onClick={() => setActiveTool('chargeback')}
              />
            </NavSection>
          )}

          {/* Reseller-only group: distribution / billing / packaging. */}
          {isReseller && (
            <NavSection
              label={t('nav.section.reseller', '经销 · Reseller')}
              hint={t('nav.section.resellerHint', '推广、激活码、白标')}
            >
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
            </NavSection>
          )}

          {/* Explore — entry point to the wider Lurus product ecosystem.
              Visible to all modes so EndUser clients can discover sibling
              products. Jumps to HomePage where the ecosystem grid is
              inlined (no separate top-level route). */}
          {!isEndUser && (
            <NavSection
              label={t('nav.section.explore', '探索 · Explore')}
              hint={t('nav.section.exploreHint', 'Lurus 生态产品')}
            >
              <SubNavButton
                name={t('nav.shortcut.ecosystem', 'Lurus 生态')}
                icon={Compass}
                iconColor="text-fuchsia-400"
                active={false}
                onClick={gotoHomeEcosystem}
              />
            </NavSection>
          )}
        </div>
      </nav>

      {/* Settings (bottom) */}
      <div className="p-2 border-t border-rule-strong space-y-2">
        <NavButton
          id="settings"
          name={t('nav.settings')}
          icon={Settings}
          iconColor="text-muted-foreground"
          active={activeTool === 'settings'}
          onClick={() => setActiveTool('settings')}
        />
        {/* Keyboard shortcut hints */}
        <p className="font-mono text-[10px] text-muted-foreground/50 text-center leading-relaxed px-1 tracking-[0.08em]">
          Ctrl+1~5 {t('nav.switchPage', 'switch page')} · Ctrl+S {t('nav.save', 'save')}
        </p>
      </div>
    </aside>
  )
}
