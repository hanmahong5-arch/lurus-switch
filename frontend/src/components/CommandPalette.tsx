import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import {
  Search, Download, Play, Square, Link2, Wrench, Activity,
  Home, Settings, Briefcase, CreditCard, Radio, Terminal,
  Loader2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { TOOL_ORDER, toolMeta } from '../lib/toolMeta'
import { useCommandPaletteStore } from '../stores/commandPaletteStore'
import { useConfigStore } from '../stores/configStore'
import { useHomeStore } from '../stores/homeStore'
import { useToastStore } from '../stores/toastStore'
import {
  InstallTool, InstallAllTools, StartGateway, StopGateway,
  AutoConfigureToolsForGateway, ApplyAllOptimizations,
  FullSetupForGateway, DetectAllTools, CheckAllToolHealth,
  LaunchTool,
} from '../../wailsjs/go/main/App'

type Category = 'install' | 'action' | 'navigate' | 'launch'

interface Command {
  id: string
  labelKey: string
  keywords: string[]
  category: Category
  icon: typeof Search
  action: () => void | Promise<void>
}

const CATEGORY_ORDER: Category[] = ['install', 'action', 'navigate', 'launch']

export function CommandPalette() {
  const { t } = useTranslation()
  const { open, setOpen } = useCommandPaletteStore()
  const [query, setQuery] = useState('')
  const [activeIndex, setActiveIndex] = useState(0)
  const [running, setRunning] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const setActiveTool = useConfigStore((s) => s.setActiveTool)
  const toast = useToastStore((s) => s.addToast)

  const refreshTools = useCallback(async () => {
    try {
      const tools = await DetectAllTools()
      useHomeStore.getState().setTools(tools)
      const health = await CheckAllToolHealth()
      useHomeStore.getState().setToolHealth(health)
    } catch { /* non-critical */ }
  }, [])

  const commands = useMemo<Command[]>(() => [
    // Install
    ...TOOL_ORDER.map((name) => ({
      id: `install-${name}`,
      labelKey: `commandPalette.commands.install${toolMeta[name].label.replace(/\s+/g, '')}`,
      keywords: ['install', name, toolMeta[name].label.toLowerCase()],
      category: 'install' as Category,
      icon: Download,
      action: async () => {
        await InstallTool(name)
        refreshTools()
        toast('success', `${toolMeta[name].label} ${t('dashboard.installSuccess')}`)
      },
    })),
    {
      id: 'install-all',
      labelKey: 'commandPalette.commands.installAll',
      keywords: ['install', 'all', 'tools', 'everything'],
      category: 'install',
      icon: Download,
      action: async () => {
        await InstallAllTools()
        refreshTools()
        toast('success', t('dashboard.installAllSuccess'))
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
      action: async () => { await AutoConfigureToolsForGateway(); refreshTools(); toast('success', t('home.optimizeSuccess')) },
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
    // Navigate
    { id: 'go-home', labelKey: 'commandPalette.commands.goHome', keywords: ['home', 'dashboard'], category: 'navigate', icon: Home, action: () => setActiveTool('home') },
    { id: 'go-tools', labelKey: 'commandPalette.commands.goTools', keywords: ['tools', 'config'], category: 'navigate', icon: Wrench, action: () => setActiveTool('tools') },
    { id: 'go-gateway', labelKey: 'commandPalette.commands.goGateway', keywords: ['gateway', 'server'], category: 'navigate', icon: Radio, action: () => setActiveTool('gateway') },
    { id: 'go-workspace', labelKey: 'commandPalette.commands.goWorkspace', keywords: ['workspace', 'prompts', 'context'], category: 'navigate', icon: Briefcase, action: () => setActiveTool('workspace') },
    { id: 'go-account', labelKey: 'commandPalette.commands.goAccount', keywords: ['account', 'billing', 'balance'], category: 'navigate', icon: CreditCard, action: () => setActiveTool('account') },
    { id: 'go-settings', labelKey: 'commandPalette.commands.goSettings', keywords: ['settings', 'preferences', 'theme'], category: 'navigate', icon: Settings, action: () => setActiveTool('settings') },
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
  ], [t, toast, setActiveTool, refreshTools])

  const filtered = useMemo(() => {
    if (!query.trim()) return commands
    const q = query.toLowerCase()
    return commands.filter((cmd) => {
      const label = t(cmd.labelKey).toLowerCase()
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
        <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50 animate-in fade-in-0" />
        <Dialog.Content
          className="fixed left-1/2 top-[20%] -translate-x-1/2 w-full max-w-lg bg-card border border-border rounded-xl shadow-2xl z-50 overflow-hidden animate-in fade-in-0 slide-in-from-top-4"
          onOpenAutoFocus={(e) => { e.preventDefault(); inputRef.current?.focus() }}
          aria-describedby={undefined}
        >
          <VisuallyHidden>
            <Dialog.Title>{t('commandPalette.placeholder')}</Dialog.Title>
          </VisuallyHidden>

          {/* Search input */}
          <div className="flex items-center gap-3 px-4 py-3 border-b border-border">
            {running ? (
              <Loader2 className="h-4 w-4 text-muted-foreground animate-spin shrink-0" />
            ) : (
              <Search className="h-4 w-4 text-muted-foreground shrink-0" />
            )}
            <input
              ref={inputRef}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={t('commandPalette.placeholder')}
              className="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
              autoComplete="off"
              spellCheck={false}
            />
            <kbd className="hidden sm:inline-flex items-center gap-0.5 px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground bg-muted rounded border border-border">
              ESC
            </kbd>
          </div>

          {/* Results */}
          <div ref={listRef} className="max-h-[320px] overflow-y-auto py-2">
            {flatList.length === 0 ? (
              <p className="px-4 py-6 text-sm text-muted-foreground text-center">
                {t('commandPalette.noResults')}
              </p>
            ) : (
              groups.map(({ category, items }) => (
                <div key={category}>
                  <p className="px-4 py-1 text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">
                    {t(`commandPalette.categories.${category}`)}
                  </p>
                  {items.map(({ cmd, flatIdx }) => {
                    const Icon = cmd.icon
                    return (
                      <button
                        key={cmd.id}
                        data-index={flatIdx}
                        onClick={() => execute(cmd)}
                        className={cn(
                          'w-full flex items-center gap-3 px-4 py-2 text-sm text-left transition-colors',
                          flatIdx === activeIndex ? 'bg-muted text-foreground' : 'text-muted-foreground hover:bg-muted/50'
                        )}
                        onMouseEnter={() => setActiveIndex(flatIdx)}
                      >
                        <Icon className="h-4 w-4 shrink-0" />
                        <span className="flex-1 truncate">{t(cmd.labelKey)}</span>
                      </button>
                    )
                  })}
                </div>
              ))
            )}
          </div>

          {/* Footer hint */}
          <div className="flex items-center gap-4 px-4 py-2 border-t border-border text-[10px] text-muted-foreground">
            <span className="flex items-center gap-1">
              <kbd className="px-1 py-0.5 bg-muted rounded border border-border">↑↓</kbd>
              {t('commandPalette.navigate')}
            </span>
            <span className="flex items-center gap-1">
              <kbd className="px-1 py-0.5 bg-muted rounded border border-border">↵</kbd>
              {t('commandPalette.execute')}
            </span>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
