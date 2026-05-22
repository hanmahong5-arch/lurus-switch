import { useEffect, useRef, useState } from 'react'
import { Rocket, Loader2, Check, AlertCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useHomeStore } from '../stores/homeStore'
import { useToastStore } from '../stores/toastStore'
import { TOOL_DISPLAY } from '../lib/toolMeta'
import { DetectAllTools, LaunchToolInTerminal } from '../../wailsjs/go/main/App'

// CLIs that have a sane terminal launch flag wired in bindings_launch.go.
// Order matters — popular first.
const LAUNCH_ORDER = ['claude', 'codex', 'gemini', 'picoclaw', 'nullclaw', 'zeroclaw', 'openclaw']

export function HeaderQuickLaunch() {
  const { t } = useTranslation()
  const tools = useHomeStore((s) => s.tools)
  const setTools = useHomeStore((s) => s.setTools)
  const toast = useToastStore((s) => s.addToast)

  const [open, setOpen] = useState(false)
  const [launching, setLaunching] = useState<string | null>(null)
  const rootRef = useRef<HTMLDivElement>(null)

  // Lazy-load tool detection the first time the menu opens so the icon
  // doesn't pay for it on every page mount. Stale data still renders —
  // worst case the user sees "未安装" briefly until the refresh lands.
  useEffect(() => {
    if (!open) return
    if (Object.keys(tools).length > 0) return
    DetectAllTools().then((s) => setTools(s)).catch(() => { /* leave empty */ })
  }, [open, tools, setTools])

  // Close on outside click / Escape.
  useEffect(() => {
    if (!open) return
    const onClick = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false) }
    document.addEventListener('mousedown', onClick)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onClick)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const handleLaunch = async (name: string) => {
    if (launching) return
    setLaunching(name)
    try {
      await LaunchToolInTerminal(name)
      toast('success', t('header.launchOk', { tool: TOOL_DISPLAY[name] || name }))
      setOpen(false)
    } catch (e: any) {
      const msg = e?.message || String(e)
      if (typeof msg === 'string' && msg.startsWith('not-found:')) {
        toast('warning', msg.replace(/^not-found:\s*/, ''))
      } else {
        toast('error', msg)
      }
    } finally {
      setLaunching(null)
    }
  }

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        title={t('header.quickLaunch', '快速启动 CLI / Quick launch CLI')}
        aria-label={t('header.quickLaunch', '快速启动 CLI / Quick launch CLI')}
        className={cn(
          'h-7 w-7 inline-flex items-center justify-center rounded-md transition-colors',
          open ? 'bg-muted text-foreground' : 'hover:bg-muted text-foreground/80',
        )}
      >
        <Rocket className="h-3.5 w-3.5" />
      </button>
      {open && (
        <div className="absolute right-0 top-8 z-50 w-56 rounded-md border border-border bg-card shadow-lg py-1">
          <div className="px-3 py-1.5 text-[10px] uppercase tracking-wider text-muted-foreground">
            {t('header.quickLaunchTitle', '在新终端启动')}
          </div>
          {LAUNCH_ORDER.map((name) => {
            const ts = tools[name]
            const installed = !!ts?.installed
            const display = TOOL_DISPLAY[name] || name
            const isLaunching = launching === name
            return (
              <button
                key={name}
                onClick={() => handleLaunch(name)}
                disabled={isLaunching || !installed}
                className={cn(
                  'w-full flex items-center gap-2 px-3 py-1.5 text-xs text-left',
                  installed ? 'hover:bg-muted text-foreground' : 'text-muted-foreground/60 cursor-not-allowed',
                )}
              >
                {isLaunching ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin shrink-0" />
                ) : installed ? (
                  <Check className="h-3.5 w-3.5 text-[var(--lt-ok)] shrink-0" />
                ) : (
                  <AlertCircle className="h-3.5 w-3.5 shrink-0" />
                )}
                <span className="flex-1 truncate">{display}</span>
                {ts?.version && installed && (
                  <span className="text-[10px] text-muted-foreground font-mono">{ts.version}</span>
                )}
                {!installed && (
                  <span className="text-[10px] text-muted-foreground">{t('header.notInstalled', '未安装')}</span>
                )}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}
