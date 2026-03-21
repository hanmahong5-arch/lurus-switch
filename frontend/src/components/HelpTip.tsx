import { useState, useRef, useEffect } from 'react'
import { HelpCircle, X, ExternalLink } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useConfigStore } from '../stores/configStore'

interface HelpTipProps {
  /** i18n key for the tooltip title */
  titleKey: string
  /** i18n key for the tooltip body */
  bodyKey: string
  /** Optional i18n key for a "learn more" link label */
  learnMoreKey?: string
  /** Optional navigation target when "learn more" is clicked */
  learnMoreNav?: { page: string; subTab?: string }
  /** Size of the trigger icon */
  size?: 'sm' | 'md'
  /** Only show for certain user levels (default: all) */
  showFor?: Array<'beginner' | 'regular' | 'power'>
  /** Preferred placement */
  placement?: 'top' | 'bottom' | 'left' | 'right'
  /** Inline children to wrap (renders inline instead of icon trigger) */
  children?: React.ReactNode
}

/**
 * Contextual help tooltip. Provides accurate, context-aware explanations.
 * Respects user level — hides for power users unless they hover.
 */
export function HelpTip({
  titleKey,
  bodyKey,
  learnMoreKey,
  learnMoreNav,
  size = 'sm',
  showFor,
  placement = 'top',
  children,
}: HelpTipProps) {
  const { t } = useTranslation()
  const { userLevel, setActiveTool, setSubTab } = useConfigStore()
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const tooltipRef = useRef<HTMLDivElement>(null)

  // Visibility check based on user level
  if (showFor && !showFor.includes(userLevel)) return null

  // Close on click outside
  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (
        triggerRef.current?.contains(e.target as Node) ||
        tooltipRef.current?.contains(e.target as Node)
      ) return
      setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  const handleLearnMore = () => {
    if (!learnMoreNav) return
    setActiveTool(learnMoreNav.page as any)
    if (learnMoreNav.subTab) {
      setSubTab(learnMoreNav.page as any, learnMoreNav.subTab)
    }
    setOpen(false)
  }

  const iconSize = size === 'sm' ? 'h-3.5 w-3.5' : 'h-4 w-4'

  const placementClasses: Record<string, string> = {
    top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
    bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
    left: 'right-full top-1/2 -translate-y-1/2 mr-2',
    right: 'left-full top-1/2 -translate-y-1/2 ml-2',
  }

  return (
    <span className="relative inline-flex items-center">
      {children}
      <button
        ref={triggerRef}
        onClick={() => setOpen(!open)}
        onMouseEnter={() => userLevel !== 'beginner' && setOpen(true)}
        onMouseLeave={() => userLevel !== 'beginner' && setOpen(false)}
        className={cn(
          'inline-flex items-center justify-center rounded-full transition-colors',
          'text-muted-foreground/50 hover:text-muted-foreground',
          children ? 'ml-1' : '',
        )}
        aria-label={t(titleKey)}
      >
        <HelpCircle className={iconSize} />
      </button>

      {open && (
        <div
          ref={tooltipRef}
          className={cn(
            'absolute z-50 w-64 p-3 rounded-lg shadow-lg',
            'bg-popover border border-border text-popover-foreground',
            'animate-in fade-in-0 zoom-in-95 duration-150',
            placementClasses[placement] || placementClasses.top,
          )}
        >
          <div className="flex items-start justify-between gap-2">
            <h4 className="text-xs font-semibold">{t(titleKey)}</h4>
            {userLevel === 'beginner' && (
              <button
                onClick={() => setOpen(false)}
                className="text-muted-foreground hover:text-foreground shrink-0"
              >
                <X className="h-3 w-3" />
              </button>
            )}
          </div>
          <p className="text-xs text-muted-foreground mt-1 leading-relaxed">
            {t(bodyKey)}
          </p>
          {learnMoreKey && learnMoreNav && (
            <button
              onClick={handleLearnMore}
              className="flex items-center gap-1 mt-2 text-xs text-primary hover:underline"
            >
              <ExternalLink className="h-3 w-3" />
              {t(learnMoreKey)}
            </button>
          )}
        </div>
      )}
    </span>
  )
}

interface FeatureGateProps {
  /** Minimum user level required to see children */
  minLevel: 'beginner' | 'regular' | 'power'
  /** Content to render when gated */
  children: React.ReactNode
  /** Optional collapsed placeholder */
  placeholder?: React.ReactNode
}

/**
 * Conditionally renders content based on user level.
 * Shows an "expand" placeholder for hidden content so nothing is truly locked.
 */
export function FeatureGate({ minLevel, children, placeholder }: FeatureGateProps) {
  const { userLevel } = useConfigStore()
  const [expanded, setExpanded] = useState(false)
  const { t } = useTranslation()

  const levels = { beginner: 0, regular: 1, power: 2 }
  const visible = levels[userLevel] >= levels[minLevel]

  if (visible || expanded) return <>{children}</>

  if (placeholder) return <>{placeholder}</>

  return (
    <button
      onClick={() => setExpanded(true)}
      className="w-full py-2 text-xs text-muted-foreground hover:text-foreground border border-dashed border-border rounded-md transition-colors"
    >
      {t('ui.showAdvanced')}
    </button>
  )
}
