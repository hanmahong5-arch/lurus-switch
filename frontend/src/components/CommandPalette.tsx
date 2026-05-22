import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import {
  Search, Download, Play, Square, Link2, Wrench, Activity,
  Home, Settings, Briefcase, CreditCard, Radio, Terminal, Wallet,
  Loader2, ArrowLeft, ArrowRight, Clock, ShieldAlert,
  Layers, Key, Gift, FileText, Coins, Building2, Network, Box, User, Undo2,
  Camera, UserCog, Users, KeyRound, Building,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { TOOL_ORDER, toolMeta } from '../lib/toolMeta'
import { useCommandPaletteStore } from '../stores/commandPaletteStore'
import { useConfigStore, type ActiveTool } from '../stores/configStore'
import { useNavHistoryStore } from '../stores/navHistoryStore'
import { useRepoAuditStore } from '../stores/repoAuditStore'
import { useBashGuardStore } from '../stores/bashGuardStore'
import { useBudgetStore } from '../stores/budgetStore'
import { useFeatureTourStore } from '../stores/featureTourStore'
import { goBack, goForward } from '../lib/navigation'
import { toolLabel, subTabLabel } from '../lib/navLabels'
import { useHomeStore } from '../stores/homeStore'
import { useToastStore } from '../stores/toastStore'
import { useAuditStore } from '../stores/auditStore'
import { pickLatestUndoableEntry } from '../lib/auditUndoSelector'
import { useActivityStore } from '../stores/activityStore'
import { useSnapshotsHubStore } from '../stores/snapshotsHubStore'
import {
  InstallTool, InstallAllTools, StartGateway, StopGateway,
  AutoConfigureToolsForGateway, ApplyAllOptimizations,
  FullSetupForGateway, DetectAllTools, CheckAllToolHealth,
  LaunchTool, TakeConfigSnapshot, SetAppMode, IsModeLocked,
} from '../../wailsjs/go/main/App'

type Category = 'recent' | 'install' | 'action' | 'navigate' | 'launch' | 'snapshot' | 'mode'

interface Command {
  id: string
  labelKey: string
  keywords: string[]
  category: Category
  icon: typeof Search
  action: () => void | Promise<void>
  // Pre-rendered label that bypasses t(labelKey). Used for recent-history
  // commands whose labels are computed at runtime from nav state.
  rawLabel?: string
}

const CATEGORY_ORDER: Category[] = ['recent', 'snapshot', 'mode', 'install', 'action', 'navigate', 'launch']

const RECENT_LIMIT = 5

export function CommandPalette() {
  const { t } = useTranslation()
  const { open, setOpen } = useCommandPaletteStore()
  const [query, setQuery] = useState('')
  const [activeIndex, setActiveIndex] = useState(0)
  const [running, setRunning] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const setActiveTool = useConfigStore((s) => s.setActiveTool)
  const setSubTab = useConfigStore((s) => s.setSubTab)
  const historyEntries = useNavHistoryStore((s) => s.entries)
  const historyIndex = useNavHistoryStore((s) => s.index)
  const openRepoAudit = useRepoAuditStore((s) => s.setOpen)
  const openBashGuard = useBashGuardStore((s) => s.setOpen)
  const openBudget = useBudgetStore((s) => s.setOpen)
  const openFeatureTour = useFeatureTourStore((s) => s.setOpen)
  const openActivityDrawer = useActivityStore((s) => s.setDrawerOpen)
  const openSnapshotsHub = useSnapshotsHubStore((s) => s.setOpen)
  const toast = useToastStore((s) => s.addToast)

  const refreshTools = useCallback(async () => {
    try {
      const tools = await DetectAllTools()
      useHomeStore.getState().setTools(tools)
      const health = await CheckAllToolHealth()
      useHomeStore.getState().setToolHealth(health)
    } catch { /* non-critical */ }
  }, [])

  // Recently visited pages, walking backwards from the current history
  // index, deduped, capped at RECENT_LIMIT. Excludes the entry the user
  // is currently on so the palette doesn't suggest "go to where you are".
  const recentCommands = useMemo<Command[]>(() => {
    if (historyIndex < 0) return []
    const seen = new Set<string>()
    const out: Command[] = []
    for (let i = historyIndex - 1; i >= 0 && out.length < RECENT_LIMIT; i--) {
      const entry = historyEntries[i]
      if (!entry) continue
      const key = `${entry.tool}::${entry.subTab ?? ''}`
      if (seen.has(key)) continue
      seen.add(key)
      const tlabel = toolLabel(t, entry.tool)
      const sub = entry.subTab ? subTabLabel(t, entry.tool, entry.subTab) : null
      const display = sub ? `${tlabel} › ${sub}` : tlabel
      const tool = entry.tool as ActiveTool
      const subTab = entry.subTab
      out.push({
        id: `recent-${key}`,
        labelKey: '',
        rawLabel: display,
        keywords: ['recent', tool, sub ?? '', tlabel.toLowerCase()],
        category: 'recent',
        icon: Clock,
        action: () => {
          if (subTab) setSubTab(tool, subTab)
          setActiveTool(tool)
        },
      })
    }
    return out
  }, [historyEntries, historyIndex, t, setActiveTool, setSubTab])

  const commands = useMemo<Command[]>(() => [
    ...recentCommands,
    // Install
    ...TOOL_ORDER.map((name) => ({
      id: `install-${name}`,
      labelKey: `commandPalette.commands.install${toolMeta[name].label.replace(/\s+/g, '')}`,
      keywords: ['install', name, toolMeta[name].label.toLowerCase()],
      category: 'install' as Category,
      icon: Download,
      action: async () => {
        const result = await InstallTool(name)
        refreshTools()
        // Result.success guard — without this a CN-network silent failure
        // would still raise a green "installed" toast (see feedback memory:
        // wails-result-success-silent-failure).
        if (result?.success) {
          toast('success', `${toolMeta[name].label} ${t('dashboard.installSuccess')}`)
        } else {
          toast(
            'error',
            result?.message
              || t('commandPalette.errors.installFailed', '{{tool}} 安装失败', {
                tool: toolMeta[name].label,
              }),
          )
        }
      },
    })),
    {
      id: 'install-all',
      labelKey: 'commandPalette.commands.installAll',
      keywords: ['install', 'all', 'tools', 'everything'],
      category: 'install',
      icon: Download,
      action: async () => {
        const results = await InstallAllTools()
        refreshTools()
        const failed = (results ?? []).filter((r) => !r?.success)
        if (failed.length === 0) {
          toast('success', t('dashboard.installAllSuccess'))
        } else {
          // Surface per-tool failures rather than a green-then-wonder-why
          // bug report. One toast with the first failing tool's message
          // keeps the UI quiet but actionable.
          const first = failed[0]
          toast(
            'error',
            t('commandPalette.errors.installAllFailed', '{{n}} 个工具安装失败：{{msg}}', {
              n: failed.length,
              msg: first?.message || first?.tool || '',
            }),
          )
        }
      },
    },
    // Actions
    {
      id: 'start-gateway', labelKey: 'commandPalette.commands.startGateway',
      keywords: ['start', 'gateway', 'server', 'run'], category: 'action', icon: Play,
      action: async () => { await StartGateway(); toast('success', t('switch.startSuccess')) },
    },
    {
      id: 'stop-gateway', labelKey: 'commandPalette.commands.stopGateway',
      keywords: ['stop', 'gateway', 'server', 'kill'], category: 'action', icon: Square,
      action: async () => { await StopGateway(); toast('success', t('switch.stopSuccess')) },
    },
    {
      id: 'connect-all', labelKey: 'commandPalette.commands.connectAll',
      keywords: ['connect', 'all', 'tools', 'gateway', 'configure'], category: 'action', icon: Link2,
      action: async () => {
        const results = await AutoConfigureToolsForGateway()
        refreshTools()
        const failed = (results ?? []).filter((r) => !r?.success)
        if (failed.length === 0) {
          toast('success', t('home.optimizeSuccess'))
        } else {
          const first = failed[0]
          toast(
            'error',
            t('commandPalette.errors.configFailed', '{{n}} 个工具未能接入：{{msg}}', {
              n: failed.length,
              msg: first?.message || first?.tool || '',
            }),
          )
        }
      },
    },
    {
      id: 'fix-all', labelKey: 'commandPalette.commands.fixAll',
      keywords: ['fix', 'repair', 'optimize', 'all', 'issues'], category: 'action', icon: Wrench,
      action: async () => {
        await ApplyAllOptimizations()
        await FullSetupForGateway()
        refreshTools()
        toast('success', t('home.fixAllSuccess'))
      },
    },
    {
      id: 'diagnostics', labelKey: 'commandPalette.commands.diagnostics',
      keywords: ['diagnostics', 'health', 'check', 'environment', 'score'], category: 'action', icon: Activity,
      action: () => { setActiveTool('home') },
    },
    {
      id: 'audit-repo', labelKey: 'commandPalette.commands.auditRepo',
      keywords: ['audit', 'repo', 'security', 'cve', 'untrusted', '审计', '仓库', '安全'],
      category: 'action', icon: ShieldAlert,
      action: () => { openRepoAudit(true) },
    },
    {
      id: 'bash-guard', labelKey: 'commandPalette.commands.bashGuard',
      keywords: ['bash', 'guard', 'rm', 'protect', 'security', 'hook', '防护', '拦截', '危险'],
      category: 'action', icon: ShieldAlert,
      action: () => { openBashGuard(true) },
    },
    {
      id: 'budget-wall', labelKey: 'commandPalette.commands.budgetWall',
      keywords: ['budget', 'spend', 'cap', 'token', 'limit', 'cost', '预算', '上限', '花费', '限额'],
      category: 'action', icon: Wallet,
      action: () => { openBudget(true) },
    },
    {
      id: 'feature-tour', labelKey: 'commandPalette.commands.featureTour',
      keywords: ['tour', 'help', 'welcome', 'intro', 'onboarding', '功能', '指引', '介绍', '欢迎'],
      category: 'action', icon: Activity,
      action: () => { openFeatureTour(true) },
    },
    {
      id: 'activity-drawer', labelKey: 'commandPalette.commands.openActivityDrawer',
      keywords: ['activity', 'drawer', 'history', 'recent', 'log', '活动', '历史', '记录'],
      category: 'action', icon: Activity,
      action: () => { openActivityDrawer(true) },
    },
    // Snapshot — quick "save current config so I can roll back if the next
    // thing breaks anything". Linked to the snapshot store; rollback UI
    // lives in PR-W1.5.
    {
      id: 'take-snapshot', labelKey: 'commandPalette.commands.takeSnapshot',
      keywords: ['snapshot', 'backup', 'save', '快照', '备份'],
      category: 'snapshot', icon: Camera,
      action: async () => {
        const ts = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
        const name = `palette-${ts}`
        // TakeConfigSnapshot(name, reason) — Wails-typed, throws on failure.
        await TakeConfigSnapshot(name, t('commandPalette.snapshotReason', 'Command palette one-tap snapshot'))
        toast('success', t('commandPalette.snapshotSaved', '已保存配置快照：{{name}}', { name }))
      },
    },
    {
      id: 'open-snapshots-hub', labelKey: 'commandPalette.commands.openSnapshotsHub',
      keywords: ['snapshot', 'rollback', 'restore', 'history', '快照', '回滚', '恢复'],
      category: 'snapshot', icon: Camera,
      action: () => { openSnapshotsHub(true) },
    },
    // Mode switch — Personal / Reseller / EndUser. Suppressed (action no-ops
    // with a toast) when the build is mode-locked (white-label installer or
    // operator pinned via settings).
    {
      id: 'mode-personal', labelKey: 'commandPalette.commands.modePersonal',
      keywords: ['mode', 'personal', 'switch', '个人', '模式'],
      category: 'mode', icon: UserCog,
      action: async () => {
        const locked = await IsModeLocked().catch(() => false)
        if (locked) {
          toast('warning', t('commandPalette.errors.modeLocked', '当前模式已被锁定，无法切换'))
          return
        }
        await SetAppMode('personal')
        useConfigStore.getState().setAppMode('personal')
        toast('success', t('commandPalette.modeSwitched', '已切换到 {{mode}} 模式', { mode: 'Personal' }))
      },
    },
    {
      id: 'mode-reseller', labelKey: 'commandPalette.commands.modeReseller',
      keywords: ['mode', 'reseller', 'switch', '经销商', '模式'],
      category: 'mode', icon: Users,
      action: async () => {
        const locked = await IsModeLocked().catch(() => false)
        if (locked) {
          toast('warning', t('commandPalette.errors.modeLocked', '当前模式已被锁定，无法切换'))
          return
        }
        await SetAppMode('reseller')
        useConfigStore.getState().setAppMode('reseller')
        toast('success', t('commandPalette.modeSwitched', '已切换到 {{mode}} 模式', { mode: 'Reseller' }))
      },
    },
    {
      id: 'mode-enduser', labelKey: 'commandPalette.commands.modeEnduser',
      keywords: ['mode', 'enduser', 'switch', 'customer', '客户', '模式'],
      category: 'mode', icon: KeyRound,
      action: async () => {
        const locked = await IsModeLocked().catch(() => false)
        if (locked) {
          toast('warning', t('commandPalette.errors.modeLocked', '当前模式已被锁定，无法切换'))
          return
        }
        await SetAppMode('enduser')
        useConfigStore.getState().setAppMode('enduser')
        toast('success', t('commandPalette.modeSwitched', '已切换到 {{mode}} 模式', { mode: 'EndUser' }))
      },
    },
    {
      id: 'mode-enterprise', labelKey: 'commandPalette.commands.modeEnterprise',
      keywords: ['mode', 'enterprise', 'switch', '企业', '模式'],
      category: 'mode', icon: Building,
      action: async () => {
        const locked = await IsModeLocked().catch(() => false)
        if (locked) {
          toast('warning', t('commandPalette.errors.modeLocked', '当前模式已被锁定，无法切换'))
          return
        }
        await SetAppMode('enterprise')
        useConfigStore.getState().setAppMode('enterprise')
        toast('success', t('commandPalette.modeSwitched', '已切换到 {{mode}} 模式', { mode: 'Enterprise' }))
      },
    },
    // Navigate
    {
      id: 'nav-back', labelKey: 'commandPalette.commands.navBack',
      keywords: ['back', 'previous', 'undo', '后退', '上一页', '返回'],
      category: 'navigate', icon: ArrowLeft,
      action: () => { goBack() },
    },
    {
      id: 'nav-forward', labelKey: 'commandPalette.commands.navForward',
      keywords: ['forward', 'next', 'redo', '前进', '下一页'],
      category: 'navigate', icon: ArrowRight,
      action: () => { goForward() },
    },
    { id: 'go-home', labelKey: 'commandPalette.commands.goHome', keywords: ['home', 'dashboard'], category: 'navigate', icon: Home, action: () => setActiveTool('home') },
    { id: 'go-conversations', labelKey: 'commandPalette.commands.goConversations', keywords: ['conversations', 'sessions', 'history', 'jsonl', 'replay'], category: 'navigate', icon: Activity, action: () => setActiveTool('conversations') },
    { id: 'go-tools', labelKey: 'commandPalette.commands.goTools', keywords: ['tools', 'config'], category: 'navigate', icon: Wrench, action: () => setActiveTool('tools') },
    { id: 'go-gateway', labelKey: 'commandPalette.commands.goGateway', keywords: ['gateway', 'server'], category: 'navigate', icon: Radio, action: () => setActiveTool('gateway') },
    { id: 'go-workspace', labelKey: 'commandPalette.commands.goWorkspace', keywords: ['workspace', 'prompts', 'context'], category: 'navigate', icon: Briefcase, action: () => setActiveTool('workspace') },
    { id: 'go-account', labelKey: 'commandPalette.commands.goAccount', keywords: ['account', 'billing', 'balance'], category: 'navigate', icon: CreditCard, action: () => setActiveTool('account') },
    { id: 'go-settings', labelKey: 'commandPalette.commands.goSettings', keywords: ['settings', 'preferences', 'theme'], category: 'navigate', icon: Settings, action: () => setActiveTool('settings') },
    // Connected Apps + governance — high-traffic landing pages that
    // were previously only reachable via the sidebar.
    {
      id: 'go-connected-apps', labelKey: 'commandPalette.commands.goConnectedApps',
      keywords: ['apps', 'connected', 'register', 'owner', 'bind', 'chargeback', '应用', '绑定', '归属'],
      category: 'navigate', icon: User,
      action: () => { setActiveTool('gateway'); setSubTab('gateway', 'apps') },
    },
    {
      id: 'go-chargeback', labelKey: 'commandPalette.commands.goChargeback',
      keywords: ['chargeback', 'cost', 'billing', 'department', 'employee', '成本', '归集', '分摊'],
      category: 'navigate', icon: Coins,
      action: () => setActiveTool('chargeback'),
    },
    {
      id: 'go-orgchart', labelKey: 'commandPalette.commands.goOrgChart',
      keywords: ['org', 'chart', 'employee', 'department', 'csv', '组织架构', '员工', '部门'],
      category: 'navigate', icon: Building2,
      action: () => setActiveTool('orgchart'),
    },
    {
      id: 'go-dlp', labelKey: 'commandPalette.commands.goDlp',
      keywords: ['dlp', 'leak', 'redact', 'sensitive', '敏感', '脱敏', '泄露'],
      category: 'navigate', icon: ShieldAlert,
      action: () => setActiveTool('dlp'),
    },
    {
      id: 'go-agent-templates', labelKey: 'commandPalette.commands.goAgentTemplates',
      keywords: ['agent', 'template', 'gallery', 'preset', '模板', '画廊'],
      category: 'navigate', icon: Briefcase,
      action: () => setActiveTool('agent-templates'),
    },
    {
      id: 'go-promotion', labelKey: 'commandPalette.commands.goPromotion',
      keywords: ['promotion', 'promoter', 'referral', 'invite', '推广', '分销'],
      category: 'navigate', icon: Activity,
      action: () => setActiveTool('promotion'),
    },
    {
      id: 'go-packager', labelKey: 'commandPalette.commands.goPackager',
      keywords: ['packager', 'package', 'whitelabel', 'enduser', 'distributor', '打包', '白标'],
      category: 'navigate', icon: Box,
      action: () => setActiveTool('packager'),
    },
    // newapi admin sub-tabs — only meaningful in Reseller (admin scope),
    // but the palette is mode-agnostic; in Personal/EndUser these will
    // route through GatewayRequiredGuard which renders a no-access tile.
    {
      id: 'go-relay', labelKey: 'commandPalette.commands.goRelay',
      keywords: ['relay', 'proxy', 'forward', 'mcp', '中转'],
      category: 'navigate', icon: Network,
      action: () => { setActiveTool('gateway'); setSubTab('gateway', 'relay') },
    },
    {
      id: 'go-channels', labelKey: 'commandPalette.commands.goChannels',
      keywords: ['channels', 'upstream', 'provider', 'newapi', '渠道', '上游'],
      category: 'navigate', icon: Layers,
      action: () => { setActiveTool('gateway'); setSubTab('gateway', 'channels') },
    },
    {
      id: 'go-tokens', labelKey: 'commandPalette.commands.goTokens',
      keywords: ['tokens', 'api key', 'sk-', 'newapi', '令牌'],
      category: 'navigate', icon: Key,
      action: () => { setActiveTool('gateway'); setSubTab('gateway', 'tokens') },
    },
    {
      id: 'go-redemptions', labelKey: 'commandPalette.commands.goRedemptions',
      keywords: ['redemption', 'redeem', 'code', 'voucher', 'newapi', '兑换码'],
      category: 'navigate', icon: Gift,
      action: () => { setActiveTool('gateway'); setSubTab('gateway', 'redemptions') },
    },
    {
      id: 'go-logs', labelKey: 'commandPalette.commands.goLogs',
      keywords: ['logs', 'log', 'request', 'history', 'newapi', '日志'],
      category: 'navigate', icon: FileText,
      action: () => { setActiveTool('gateway'); setSubTab('gateway', 'logs') },
    },
    {
      id: 'go-audit', labelKey: 'commandPalette.commands.goAudit',
      keywords: ['audit', 'journal', 'history', 'undo', '审计'],
      category: 'navigate', icon: Activity,
      action: () => { setActiveTool('gateway'); setSubTab('gateway', 'system') },
    },
    // Action: undo most recent reversible audit entry. Picks the latest
    // ok-outcome reversible non-undone entry from the audit store. If
    // the store hasn't loaded, refresh once before searching.
    {
      id: 'undo-last-audit', labelKey: 'commandPalette.commands.undoLastAudit',
      keywords: ['undo', 'revert', 'last', 'rollback', 'audit', '撤销', '回滚', '上一步'],
      category: 'action', icon: Undo2,
      action: async () => {
        const audit = useAuditStore.getState()
        if (audit.entries.length === 0) {
          await audit.load()
        }
        const latest = pickLatestUndoableEntry(useAuditStore.getState().entries)
        if (!latest) {
          toast('warning', t('commandPalette.undoNothing', '没有可撤销的最近审计条目'))
          return
        }
        await useAuditStore.getState().undo(latest.id)
        toast('success', t('commandPalette.undoDone', '已撤销 {{op}}', { op: latest.operation }))
      },
    },
    // Launch
    ...(['claude', 'codex', 'gemini'] as const).map((name) => ({
      id: `launch-${name}`,
      labelKey: `commandPalette.commands.launch${toolMeta[name].label.replace(/\s+/g, '')}`,
      keywords: ['launch', 'open', 'run', name, toolMeta[name].label.toLowerCase()],
      category: 'launch' as Category,
      icon: Terminal,
      action: async () => {
        await LaunchTool(name, [])
        toast('success', `${toolMeta[name].label} launched`)
      },
    })),
  ], [t, toast, setActiveTool, setSubTab, refreshTools, recentCommands, openRepoAudit, openBashGuard, openBudget, openFeatureTour, openActivityDrawer, openSnapshotsHub])

  const filtered = useMemo(() => {
    if (!query.trim()) return commands
    const q = query.toLowerCase()
    return commands.filter((cmd) => {
      const label = (cmd.rawLabel ?? t(cmd.labelKey)).toLowerCase()
      return label.includes(q) || cmd.keywords.some((kw) => kw.includes(q))
    })
  }, [commands, query, t])

  // Pre-compute grouped structure with stable flat indices
  const { groups, flatList } = useMemo(() => {
    const map = new Map<Category, { cmd: Command; flatIdx: number }[]>()
    const flat: Command[] = []
    let idx = 0
    for (const cmd of filtered) {
      const list = map.get(cmd.category) || []
      list.push({ cmd, flatIdx: idx })
      map.set(cmd.category, list)
      flat.push(cmd)
      idx++
    }
    const groups = CATEGORY_ORDER
      .filter((c) => map.has(c))
      .map((c) => ({ category: c, items: map.get(c)! }))
    return { groups, flatList: flat }
  }, [filtered])

  useEffect(() => {
    if (open) { setQuery(''); setActiveIndex(0) }
  }, [open])

  useEffect(() => { setActiveIndex(0) }, [query])

  // Scroll active item into view
  useEffect(() => {
    const el = listRef.current?.querySelector(`[data-index="${activeIndex}"]`)
    el?.scrollIntoView({ block: 'nearest' })
  }, [activeIndex])

  const execute = useCallback(async (cmd: Command) => {
    if (running) return // Prevent double execution
    setRunning(true)
    setOpen(false)
    try {
      await cmd.action()
    } catch (err: any) {
      toast('error', err?.message || String(err))
    } finally {
      setRunning(false)
    }
  }, [running, setOpen, toast])

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActiveIndex((i) => Math.min(i + 1, flatList.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActiveIndex((i) => Math.max(i - 1, 0))
    } else if (e.key === 'Enter' && flatList[activeIndex]) {
      e.preventDefault()
      execute(flatList[activeIndex])
    }
  }, [flatList, activeIndex, execute])

  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/30 backdrop-blur-sm z-50 animate-in fade-in-0" />
        <Dialog.Content
          className="fixed left-1/2 top-[20%] -translate-x-1/2 w-full max-w-lg bg-card-elevated border border-rule-strong rounded-xl shadow-2xl z-50 overflow-hidden animate-in fade-in-0 slide-in-from-top-4"
          onOpenAutoFocus={(e) => { e.preventDefault(); inputRef.current?.focus() }}
          aria-describedby={undefined}
        >
          <VisuallyHidden>
            <Dialog.Title>{t('commandPalette.placeholder')}</Dialog.Title>
          </VisuallyHidden>

          {/* Search input */}
          <div className="flex items-center gap-3 px-4 py-3 border-b border-border">
            {running ? (
              <Loader2 className="h-4 w-4 text-primary animate-spin shrink-0" />
            ) : (
              <Search className="h-4 w-4 text-primary shrink-0" />
            )}
            <input
              ref={inputRef}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={t('commandPalette.placeholder')}
              className="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground font-mono"
              autoComplete="off"
              spellCheck={false}
            />
            <kbd className="hidden sm:inline-flex items-center gap-0.5 px-1.5 py-0.5 text-[10px] font-mono uppercase tracking-[0.08em] text-muted-foreground bg-card-recessed rounded border border-border">
              ESC
            </kbd>
          </div>

          {/* Results */}
          <div ref={listRef} className="max-h-[320px] overflow-y-auto py-2">
            {flatList.length === 0 ? (
              <p className="px-4 py-6 text-sm text-muted-foreground text-center font-mono">
                ▪ {t('commandPalette.noResults')}
              </p>
            ) : (
              groups.map(({ category, items }) => (
                <div key={category}>
                  <p className="px-4 py-1 font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground">
                    [ {t(`commandPalette.categories.${category}`).toUpperCase()} ]
                  </p>
                  {items.map(({ cmd, flatIdx }) => {
                    const Icon = cmd.icon
                    const isActive = flatIdx === activeIndex
                    return (
                      <button
                        key={cmd.id}
                        data-index={flatIdx}
                        onClick={() => execute(cmd)}
                        className={cn(
                          'w-full flex items-center gap-3 px-4 py-2 text-left transition-all duration-150 border-l-2',
                          isActive
                            ? 'bg-primary/15 text-primary border-primary font-mono text-xs tracking-[0.06em]'
                            : 'text-foreground/80 hover:bg-muted/60 hover:text-foreground border-transparent text-sm',
                        )}
                        onMouseEnter={() => setActiveIndex(flatIdx)}
                      >
                        <Icon className="h-4 w-4 shrink-0" />
                        <span className="flex-1 truncate">{isActive ? `▸ ${cmd.rawLabel ?? t(cmd.labelKey)}` : (cmd.rawLabel ?? t(cmd.labelKey))}</span>
                      </button>
                    )
                  })}
                </div>
              ))
            )}
          </div>

          {/* Footer hint */}
          <div className="flex items-center gap-4 px-4 py-2 border-t border-border text-[10px] text-muted-foreground font-mono">
            <span className="flex items-center gap-1">
              <kbd className="px-1 py-0.5 bg-card-recessed rounded border border-border tabular-nums">↑↓</kbd>
              {t('commandPalette.navigate')}
            </span>
            <span className="flex items-center gap-1">
              <kbd className="px-1 py-0.5 bg-card-recessed rounded border border-border tabular-nums">↵</kbd>
              {t('commandPalette.execute')}
            </span>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
