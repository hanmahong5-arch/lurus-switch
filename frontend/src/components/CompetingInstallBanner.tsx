import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Sparkles, X } from 'lucide-react'
import { DetectCompetingInstalls } from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'

// sessionStorage key — single-shot dismissal per session, not a "never
// remind me" pref. Persisting in AppSettings would require an opt-out
// UI we don't need; rediscovering next launch is fine.
const DISMISS_KEY = 'switch.competingInstallsDismissed'

// CompetingInstallBanner — detects cc-switch / opcode / ccr / hermes /
// openclaw config dirs and surfaces a one-line nudge. Inspired by
// Hermes Desktop's "OpenClaw Installation Detected" pattern.
//
// Click-through goes to Tools page since that's where users would
// reconfigure paths after deciding to migrate. We don't auto-migrate —
// each competitor has different schema and an import wizard would be
// a separate epic.
export function CompetingInstallBanner({ onJumpToTools }: { onJumpToTools?: () => void }) {
  const { t } = useTranslation()
  const [installs, setInstalls] = useState<main.CompetingInstall[]>([])
  const [dismissed, setDismissed] = useState(false)

  useEffect(() => {
    let cancelled = false
    if (sessionStorage.getItem(DISMISS_KEY) === '1') {
      setDismissed(true)
      return
    }
    DetectCompetingInstalls()
      .then((arr) => {
        if (!cancelled && arr) setInstalls(arr)
      })
      .catch(() => { /* detection is best-effort */ })
    return () => { cancelled = true }
  }, [])

  const handleDismiss = () => {
    sessionStorage.setItem(DISMISS_KEY, '1')
    setDismissed(true)
  }

  if (dismissed || installs.length === 0) return null

  const names = installs.map((i) => i.name).join(', ')

  return (
    <div className="rounded-md border border-blue-500/30 bg-blue-500/5 p-3 flex items-start gap-2">
      <Sparkles className="h-4 w-4 mt-0.5 shrink-0 text-blue-500" />
      <div className="flex-1 space-y-1">
        <p className="text-xs font-medium text-blue-600 dark:text-blue-400">
          {t('competingInstall.title', '检测到其他工具的配置')}
        </p>
        <p className="text-[11px] text-muted-foreground">
          {t('competingInstall.body', '本机已安装 {{names}}。Switch 可以并存使用 — 在「工具」页可统一查看配置。', { names })}
        </p>
        <div className="flex items-center gap-2 pt-1">
          {onJumpToTools && (
            <button
              onClick={onJumpToTools}
              className="text-[11px] text-blue-600 dark:text-blue-400 hover:underline"
            >
              {t('competingInstall.goTools', '去工具页 →')}
            </button>
          )}
          <button
            onClick={handleDismiss}
            className="text-[11px] text-muted-foreground hover:text-foreground"
          >
            {t('common.dismiss', '关闭')}
          </button>
        </div>
      </div>
      <button onClick={handleDismiss} className="p-0.5 hover:bg-muted rounded">
        <X className="h-3 w-3" />
      </button>
    </div>
  )
}
