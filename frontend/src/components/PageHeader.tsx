import { useEffect, useState } from 'react'
import { ArrowLeft, ArrowRight, ChevronRight, Sun, Moon, Monitor, Languages } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useConfigStore } from '../stores/configStore'
import { useNavHistoryStore } from '../stores/navHistoryStore'
import { goBack, goForward } from '../lib/navigation'
import { toolLabel, subTabLabel } from '../lib/navLabels'
import { setLanguage, setTheme, type Language, type Theme } from '../lib/appPrefs'
import { GetAppSettings } from '../../wailsjs/go/main/App'

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

  // Track the current preference values for icon rendering. We seed from
  // i18n / document.documentElement at mount; future changes from this
  // header re-set the local state directly.
  const [lang, setLangState] = useState<Language>((i18n.language?.startsWith('en') ? 'en' : 'zh'))
  const [theme, setThemeState] = useState<Theme>(() => {
    return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
  })

  // Keep local lang state in sync with i18n if anything else (e.g. the
  // Settings page) calls changeLanguage().
  useEffect(() => {
    const handler = (l: string) => setLangState(l.startsWith('en') ? 'en' : 'zh')
    i18n.on('languageChanged', handler)
    return () => { i18n.off('languageChanged', handler) }
  }, [i18n])

  // Seed the theme icon from the persisted setting so 'auto' is shown
  // correctly (classList.contains('dark') would mis-classify auto+dark
  // as dark). One-shot: subsequent changes from this header set the
  // local state directly.
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

  const ThemeIcon = THEME_ICON[theme]

  return (
    <header className="flex items-center gap-2 px-4 h-10 border-b border-border bg-muted/20 shrink-0">
      <button
        type="button"
        onClick={goBack}
        disabled={!canGoBack}
        title={t('nav.back')}
        aria-label={t('nav.back')}
        className={cn(
          'h-7 w-7 inline-flex items-center justify-center rounded-md transition-colors',
          canGoBack
            ? 'hover:bg-muted text-foreground'
            : 'text-muted-foreground/40 cursor-not-allowed',
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
          'h-7 w-7 inline-flex items-center justify-center rounded-md transition-colors',
          canGoForward
            ? 'hover:bg-muted text-foreground'
            : 'text-muted-foreground/40 cursor-not-allowed',
        )}
      >
        <ArrowRight className="h-4 w-4" />
      </button>

      <nav
        className="flex-1 flex items-center gap-1 text-xs text-muted-foreground min-w-0"
        aria-label={t('nav.breadcrumb')}
      >
        <span className="text-foreground/80 font-medium truncate">{tool}</span>
        {sub && (
          <>
            <ChevronRight className="h-3 w-3 shrink-0" />
            <span className="truncate">{sub}</span>
          </>
        )}
      </nav>

      <button
        type="button"
        onClick={toggleLang}
        title={t('header.langToggleHint', lang === 'zh' ? '切换到英文 / Switch to English' : 'Switch to Chinese / 切换到中文')}
        aria-label={t('header.langToggle', 'Toggle language')}
        className="h-7 px-2 inline-flex items-center gap-1 rounded-md hover:bg-muted text-foreground/80"
      >
        <Languages className="h-3.5 w-3.5" />
        <span className="text-[11px] font-mono uppercase">{lang === 'zh' ? '中' : 'EN'}</span>
      </button>

      <button
        type="button"
        onClick={cycleTheme}
        title={t('header.themeToggleHint', `Theme: ${theme} (click to cycle)`)}
        aria-label={t('header.themeToggle', 'Toggle theme')}
        className="h-7 w-7 inline-flex items-center justify-center rounded-md hover:bg-muted text-foreground/80"
      >
        <ThemeIcon className="h-3.5 w-3.5" />
      </button>
    </header>
  )
}
