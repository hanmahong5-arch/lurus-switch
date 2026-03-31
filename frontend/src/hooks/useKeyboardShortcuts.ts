import { useEffect } from 'react'
import { useConfigStore, type ActiveTool } from '../stores/configStore'
import { useCommandPaletteStore } from '../stores/commandPaletteStore'

/**
 * Global keyboard shortcuts for the Switch application.
 *
 * Ctrl+K — Command Palette
 * Ctrl+1 — Home
 * Ctrl+2 — Tools
 * Ctrl+3 — Gateway
 * Ctrl+4 — Workspace
 * Ctrl+5 — Account
 * Ctrl+S — Save (triggers [data-shortcut="save"] button)
 */
export function useKeyboardShortcuts() {
  const { setActiveTool } = useConfigStore()

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const ctrl = e.ctrlKey || e.metaKey
      const target = e.target as HTMLElement
      const isInput = target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.tagName === 'SELECT'

      if (!ctrl) return

      // Ctrl+K — command palette
      if (e.key === 'k') {
        e.preventDefault()
        useCommandPaletteStore.getState().toggle()
        return
      }

      // Ctrl+S — trigger save button
      if (e.key === 's') {
        const btn = document.querySelector('[data-shortcut="save"]') as HTMLButtonElement | null
        if (btn && !btn.disabled) {
          e.preventDefault()
          btn.click()
        }
        return
      }

      // Ctrl+number — page shortcuts (only when not in input)
      if (!isInput) {
        const pageMap: Record<string, ActiveTool> = {
          '1': 'home',
          '2': 'tools',
          '3': 'gateway',
          '4': 'workspace',
          '5': 'account',
        }
        const page = pageMap[e.key]
        if (page) {
          e.preventDefault()
          setActiveTool(page)
        }
      }
    }

    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [setActiveTool])
}
