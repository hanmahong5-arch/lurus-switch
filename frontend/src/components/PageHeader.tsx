import { useEffect, useState } from 'react'
import {
  ArrowLeft, ArrowRight, ChevronRight, Sun, Moon, Monitor, Languages,
  RefreshCw, Loader2, Copy, Search, HelpCircle, Check,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useConfigStore } from '../stores/configStore'
import { useNavHistoryStore } from '../stores/navHistoryStore'
import { usePageActionsStore } from '../stores/pageActionsStore'
import { useCommandPaletteStore } from '../stores/commandPaletteStore'
import { useToastStore } from '../stores/toastStore'
import { goBack, goForward } from '../lib/navigation'
import { toolLabel, subTabLabel } from '../lib/navLabels'
import { setLanguage, setTheme, type Language, type Theme } from '../lib/appPrefs'
import { useSelection } from '../lib/useSelection'
import { GetAppSettings } from '../../wailsjs/go/main/App'
import { HeaderQuickLaunch } from './HeaderQuickLaunch'
import { ShortcutsModal } from './ShortcutsModal'

const THEME_CYCLE: Theme[] = ['dark', 'light', 'auto']
const THEME_ICON: Record<Theme, typeof Sun> = {
  dark: Moon,
  light: Sun,
  auto: Monitor,
}

export function PageHeader() {
  const { t, i18n } = useTranslation()
  const activeTool = useConfigStore((s) => s.activeTool)
  const subTabState = useConfigStore((s) => s.subTabState)
  const canGoBack = useNavHistoryStore((s) => s.index > 0)
  const canGoForward = useNavHistoryStore(
    (s) => s.index >= 0 && s.index < s.entries.length - 1,
  )
  const toast = useToastStore((s) => s.addToast)
  const openCmdPalette = useCommandPaletteStore((s) => s.setOpen)

  const [lang, setLangState] = useState<Language>((i18n.language?.startsWith('en') ? 'en' : 'zh'))
  const [theme, setThemeState] = useState<Theme>(() => {
    return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
  })
  const [showShortcuts, setShowShortcuts] = useState(false)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    const handler = (l: string) => setLangState(l.startsWith('en') ? 'en' : 'zh')
    i18n.on('languageChanged', handler)
    return () => { i18n.off('languageChanged', handler) }
  }, [i18n])

  useEffect(() => {
    GetAppSettings()
      .then((s) => {
        const v = (s as any)?.theme
        if (v === 'dark' || v === 'light' || v === 'auto') setThemeState(v)
      })
      .catch(() => { /* settle for the document-class guess */ })
  }, [])

  const subTab = subTabState[activeTool]
  const sub = subTabLabel(t, activeTool, subTab)
  const tool = toolLabel(t, activeTool)

  const refreshHandler = usePageActionsStore((s) => s.refreshHandler)
  const refreshing = usePageActionsStore((s) => s.refreshing)
  const runRefresh = usePageActionsStore((s) => s.runRefresh)

  // Track the live text selection so the Copy button can disable itself
  // when there's nothing to copy. Listens to selectionchange globally.
  const selectedText = useSelection()
  const hasSelection = selectedText.trim().length > 0

  const toggleLang = async () => {
    const next: Language = lang === 'zh' ? 'en' : 'zh'
    setLangState(next)
    await setLanguage(next)
  }

  const cycleTheme = async () => {
    const idx = THEME_CYCLE.indexOf(theme)
    const next = THEME_CYCLE[(idx + 1) % THEME_CYCLE.length]
    setThemeState(next)
    await setTheme(next)
  }

  const copySelection = async () => {
    if (!hasSelection) return
    try {
      await navigator.clipboard.writeText(selectedText)
      setCopied(true)
      setTimeout(() => setCopied(false), 1200)
      const preview = selectedText.length > 24
        ? selectedText.slice(0, 24) + '…'
        : selectedText
      toast('success', t('header.copyOk', { preview, count: selectedText.length }))
    } catch (e: any) {
      toast('error', t('header.copyFailed', '复制失败：') + (e?.message || String(e)))
    }
  }

  const ThemeIcon = THEME_ICON[theme]

  const iconBtn =
    'h-7 w-7 inline-flex items-center justify-center rounded-md transition-colors'

  return (
    <>
      <header className="flex items-center gap-1 px-4 h-10 border-b border-rule-strong bg-card-recessed shrink-0">
        <button
          type="button"
          onClick={goBack}
          disabled={!canGoBack}
          title={t('nav.back')}
          aria-label={t('nav.back')}
          className={cn(
            iconBtn,
            'transition-all duration-150',
            canGoBack ? 'hover:bg-muted text-foreground' : 'text-muted-foreground/40 cursor-not-allowed',
          )}
        >
          <ArrowLeft className="h-4 w-4" />
        </button>
        <button
          type="button"
          onClick={goForward}
          disabled={!canGoForward}
          title={t('nav.forward')}
          aria-label={t('nav.forward')}
          className={cn(
            iconBtn,
            'transition-all duration-150',
            canGoForward ? 'hover:bg-muted text-foreground' : 'text-muted-foreground/40 cursor-not-allowed',
          )}
        >
          <ArrowRight className="h-4 w-4" />
        </button>

        <nav
          className="flex-1 flex items-center gap-1 text-xs text-muted-foreground min-w-0 ml-1 font-mono"
          aria-label={t('nav.breadcrumb')}
        >
          <span className="text-primary uppercase tracking-[0.12em] truncate">▸ {tool}</span>
          {sub && (
            <>
              <ChevronRight className="h-3 w-3 shrink-0" />
              <span className="truncate uppercase tracking-[0.08em]">{sub}</span>
            </>
          )}
        </nav>

        {/* Action cluster — order: search → launch → copy → refresh → divider → prefs → help */}
        <button
          type="button"
          onClick={() => openCmdPalette(true)}
          title={t('header.cmdPalette', '命令面板 (Ctrl+K)')}
          aria-label={t('header.cmdPalette', '命令面板 (Ctrl+K)')}
          className={cn(iconBtn, 'hover:bg-muted text-foreground/80')}
        >
          <Search className="h-3.5 w-3.5" />
        </button>

        <HeaderQuickLaunch />

        <button
          type="button"
          onClick={copySelection}
          disabled={!hasSelection}
          title={hasSelection
            ? t('header.copySelectionHint', { count: selectedText.length, defaultValue: '复制选中（{{count}} 字符）' })
            : t('header.copyNoSelection', '没有选中文本')}
          aria-label={t('header.copySelection', '复制选中')}
          className={cn(
            iconBtn,
            hasSelection
              ? 'hover:bg-muted text-foreground/80'
              : 'text-muted-foreground/40 cursor-not-allowed',
            copied && 'text-[var(--lt-ok)]',
          )}
        >
          {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
        </button>

        {refreshHandler && (
          <button
            type="button"
            onClick={() => { void runRefresh() }}
            disabled={refreshing}
            title={t('header.refreshHint', '刷新当前页')}
            aria-label={t('header.refresh', 'Refresh')}
            className={cn(
              iconBtn,
              refreshing ? 'text-muted-foreground/60 cursor-not-allowed' : 'hover:bg-muted text-foreground/80',
            )}
          >
            {refreshing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
          </button>
        )}

        <div className="w-px h-5 bg-rule-strong mx-1" aria-hidden="true" />

        <button
          type="button"
          onClick={toggleLang}
          title={t('header.langToggleHint', lang === 'zh' ? '切换到英文' : '切换到中文')}
          aria-label={t('header.langToggle', 'Toggle language')}
          className="h-7 px-2 inline-flex items-center gap-1 rounded-md hover:bg-muted text-foreground/80 transition-colors"
        >
          <Languages className="h-3.5 w-3.5" />
          <span className="text-[11px] font-mono uppercase tracking-[0.08em]">{lang === 'zh' ? '中' : 'EN'}</span>
        </button>

        <button
          type="button"
          onClick={cycleTheme}
          title={t('header.themeToggleHint', `Theme: ${theme}`)}
          aria-label={t('header.themeToggle', 'Toggle theme')}
          className={cn(iconBtn, 'hover:bg-muted text-foreground/80')}
        >
          <ThemeIcon className="h-3.5 w-3.5" />
        </button>

        <button
          type="button"
          onClick={() => setShowShortcuts(true)}
          title={t('header.shortcuts', '键盘快捷键')}
          aria-label={t('header.shortcuts', '键盘快捷键')}
          className={cn(iconBtn, 'hover:bg-muted text-foreground/80')}
        >
          <HelpCircle className="h-3.5 w-3.5" />
        </button>
      </header>

      <ShortcutsModal open={showShortcuts} onClose={() => setShowShortcuts(false)} />
    </>
  )
}
