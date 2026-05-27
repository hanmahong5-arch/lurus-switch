import { useCallback, useEffect, useRef, useState } from 'react'
import { Zap, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useQuickSwitchStore } from '../stores/quickSwitchStore'
import { useRelayStore } from '../stores/relayStore'
import { useToastStore } from '../stores/toastStore'
import { GetRelayEndpoints, SaveToolRelayMapping, ApplyAllToolRelays } from '../../wailsjs/go/main/App'
import type { relay } from '../../wailsjs/go/models'

const TOOL_NAMES = ['claude', 'codex', 'gemini', 'picoclaw', 'nullclaw', 'zeroclaw', 'openclaw'] as const

/**
 * QuickSwitchOverlay — a keyboard-driven floating panel for switching all CLI
 * tools to a different relay endpoint in one gesture.
 *
 * Triggered by:
 *   - Global hotkey Ctrl+Shift+P  →  backend emits "hotkey:quickSwitch"
 *   - Tray "Switch Provider…"    →  backend emits "tray:switch-provider"
 *
 * Navigation:
 *   - ArrowUp / ArrowDown  — move selection
 *   - Enter                — apply selected relay
 *   - Esc                  — dismiss
 */
export function QuickSwitchOverlay() {
  const { t } = useTranslation()
  const { open, close } = useQuickSwitchStore()
  const addToast = useToastStore((s) => s.addToast)
  const cachedEndpoints = useRelayStore((s) => s.endpoints)

  const [endpoints, setEndpoints] = useState<relay.RelayEndpoint[]>([])
  const [activeIdx, setActiveIdx] = useState(0)
  const [applying, setApplying] = useState(false)
  const listRef = useRef<HTMLUListElement>(null)

  // Load relay endpoints when the overlay opens.
  useEffect(() => {
    if (!open) return
    setActiveIdx(0)

    const load = async () => {
      try {
        const eps = await GetRelayEndpoints()
        setEndpoints(eps || [])
      } catch {
        // Fall back to store cache so the overlay never shows empty on load failure.
        setEndpoints(cachedEndpoints)
      }
    }
    load()
  }, [open, cachedEndpoints])

  // Scroll active item into view.
  useEffect(() => {
    const el = listRef.current?.children[activeIdx] as HTMLElement | undefined
    el?.scrollIntoView({ block: 'nearest' })
  }, [activeIdx])

  const applyRelay = useCallback(
    async (ep: relay.RelayEndpoint) => {
      if (applying) return
      setApplying(true)
      try {
        const mapping: Record<string, string> = {}
        for (const tool of TOOL_NAMES) {
          mapping[tool] = ep.id
        }
        await SaveToolRelayMapping(mapping as never)
        const results = await ApplyAllToolRelays()
        const errors = Object.entries(results || {}).filter(([, v]) => v)
        if (errors.length === 0) {
          addToast('success', t('quickSwitch.switched', 'Switched to {{name}}', { name: ep.name }))
        } else {
          addToast('warning', t('quickSwitch.partialSwitch', '{{count}} tool(s) failed to switch', { count: errors.length }))
        }
        close()
      } catch (err) {
        addToast('error', t('quickSwitch.error', 'Switch failed: {{msg}}', { msg: (err as Error)?.message ?? String(err) }))
      } finally {
        setApplying(false)
      }
    },
    [applying, addToast, close, t],
  )

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Escape') {
        close()
        return
      }
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setActiveIdx((i) => Math.min(i + 1, endpoints.length - 1))
        return
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        setActiveIdx((i) => Math.max(i - 1, 0))
        return
      }
      if (e.key === 'Enter' && endpoints[activeIdx]) {
        e.preventDefault()
        applyRelay(endpoints[activeIdx])
      }
    },
    [endpoints, activeIdx, applyRelay, close],
  )

  if (!open) return null

  return (
    // Backdrop
    <div
      className="fixed inset-0 z-50 flex items-start justify-center pt-[20vh] bg-black/40 backdrop-blur-sm"
      data-testid="quick-switch-backdrop"
      onClick={close}
    >
      {/* Panel */}
      <div
        role="dialog"
        aria-label={t('quickSwitch.title', 'Quick Switch Provider')}
        aria-modal="true"
        data-testid="quick-switch-panel"
        className={cn(
          'w-full max-w-sm bg-card border border-border rounded-xl shadow-2xl overflow-hidden',
          'focus:outline-none',
        )}
        onClick={(e) => e.stopPropagation()}
        onKeyDown={handleKeyDown}
        // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex
        tabIndex={0}
        // Auto-focus the panel so keyboard events are captured immediately.
        ref={(el) => el?.focus()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <div className="flex items-center gap-2">
            <Zap className="h-4 w-4 text-primary" />
            <span className="text-sm font-semibold">
              {t('quickSwitch.title', 'Quick Switch Provider')}
            </span>
          </div>
          <button
            aria-label={t('quickSwitch.close', 'Close')}
            onClick={close}
            className="p-1 rounded hover:bg-muted"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>

        {/* Relay list */}
        {endpoints.length === 0 ? (
          <div className="px-4 py-8 text-sm text-center text-muted-foreground">
            {t('quickSwitch.noEndpoints', 'No relay endpoints configured')}
          </div>
        ) : (
          <ul
            ref={listRef}
            role="listbox"
            aria-label={t('quickSwitch.listLabel', 'Relay endpoints')}
            className="max-h-72 overflow-y-auto py-1"
          >
            {endpoints.map((ep, idx) => (
              <li
                key={ep.id}
                role="option"
                aria-selected={idx === activeIdx}
                data-testid={`quick-switch-item-${ep.id}`}
                className={cn(
                  'flex items-center gap-3 px-4 py-2.5 cursor-pointer transition-colors',
                  idx === activeIdx
                    ? 'bg-primary/10 text-primary'
                    : 'hover:bg-muted',
                  applying && 'pointer-events-none opacity-60',
                )}
                onClick={() => applyRelay(ep)}
                onMouseEnter={() => setActiveIdx(idx)}
              >
                {/* Health dot */}
                <span
                  className={cn(
                    'w-2 h-2 rounded-full flex-shrink-0',
                    ep.healthy ? 'bg-green-500' : 'bg-red-500',
                  )}
                  aria-hidden="true"
                />

                {/* Endpoint info */}
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium truncate">{ep.name || ep.id}</div>
                  {ep.description && (
                    <div className="text-xs text-muted-foreground truncate">{ep.description}</div>
                  )}
                </div>

                {/* Latency hint */}
                {ep.latencyMs > 0 && (
                  <span className="text-xs text-muted-foreground flex-shrink-0">
                    {ep.latencyMs}ms
                  </span>
                )}
              </li>
            ))}
          </ul>
        )}

        {/* Footer */}
        <div className="px-4 py-2 border-t border-border text-xs text-muted-foreground flex justify-between items-center">
          <span>{t('quickSwitch.hint', '↑↓ navigate · Enter apply · Esc close')}</span>
          {applying && (
            <span className="text-primary animate-pulse">
              {t('quickSwitch.applying', 'Applying…')}
            </span>
          )}
        </div>
      </div>
    </div>
  )
}
