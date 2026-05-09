import type { TFunction } from 'i18next'
import type { ActiveTool } from '../stores/configStore'

// Sub-tab → display label resolver. Reuses existing i18n keys where they
// exist (gateway/account/workspace) and falls back to literal product names
// for tool tabs (Claude, Codex, …) which are intentionally not translated.
const TOOLS_LITERAL: Record<string, string> = {
  claude: 'Claude',
  codex: 'Codex',
  gemini: 'Gemini',
  picoclaw: 'PicoClaw',
  nullclaw: 'NullClaw',
  zeroclaw: 'ZeroClaw',
  openclaw: 'OpenClaw',
}

const SUBTAB_I18N_KEY: Partial<Record<ActiveTool, Record<string, string>>> = {
  gateway: {
    // Basic
    control: 'home.gwControl',
    usage: 'home.gwUsage',
    apps: 'home.gwApps',
    relay: 'nav.relay',
    // Admin (Reseller-only)
    dashboard: 'gateway.dashboard',
    channels: 'gateway.channels',
    tokens: 'gateway.tokens',
    models: 'gateway.models',
    users: 'gateway.users',
    redemptions: 'gateway.redemptions',
    logs: 'gateway.logs',
    subscriptions: 'gateway.subscriptions',
    'admin-settings': 'gateway.gatewaySettings',
    // Root
    system: 'gateway.system',
  },
  workspace: {
    prompts: 'nav.prompts',
    context: 'nav.documents',
    process: 'nav.process',
  },
  account: {
    connection: 'home.connection',
    billing: 'nav.billing',
  },
}

export function toolLabel(t: TFunction, tool: ActiveTool): string {
  return t(`nav.${tool}`, tool as string)
}

export function subTabLabel(
  t: TFunction,
  tool: ActiveTool,
  subTab: string | undefined,
): string | null {
  if (!subTab) return null
  if (tool === 'tools') {
    if (subTab === 'mcp') return 'MCP'
    if (subTab === 'snapshots') return t('snapshots.title', 'Snapshots')
    return TOOLS_LITERAL[subTab] ?? subTab
  }
  const key = SUBTAB_I18N_KEY[tool]?.[subTab]
  return key ? t(key, subTab) : null
}
