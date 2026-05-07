import { useEffect } from 'react'
import { useConfigStore, type ActiveTool } from '../stores/configStore'
import { useCommandPaletteStore } from '../stores/commandPaletteStore'
import { goBack, goForward } from '../lib/navigation'

/**
 * Global keyboard shortcuts for the Switch application.
 *
 * Ctrl+K       — Command Palette
 * Ctrl+1..5    — Page jumps (Home/Tools/Gateway/Workspace/Account)
 * Ctrl+S       — Save (triggers [data-shortcut="save"] button)
 * Alt+←/→      — Browser-style back/forward through nav history
 * Mouse Back/Fwd — Same as Alt+←/→ (XButton1 / XButton2)
 */
export function useKeyboardShortcuts() {
  const { setActiveTool } = useConfigStore()

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement
      const isInput =
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.tagName === 'SELECT' ||
        target.isContentEditable

      // Alt+Left / Alt+Right — back/forward (works even inside inputs so users
      // who land in a form by mistake can still escape)
      if (e.altKey && !e.ctrlKey && !e.metaKey) {
        if (e.key === 'ArrowLeft') {
          e.preventDefault()
          goBack()
          return
        }
        if (e.key === 'ArrowRight') {
          e.preventDefault()
          goForward()
          return
        }
      }

      const ctrl = e.ctrlKey || e.metaKey
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

    // Mouse XButton1 (back) / XButton2 (forward). Wails forwards these as
    // standard `mouseup` events with button === 3 / 4. preventDefault on
    // mousedown stops the WebView from also handling them.
    const mouseHandler = (e: MouseEvent) => {
      if (e.button === 3) {
        e.preventDefault()
        goBack()
      } else if (e.button === 4) {
        e.preventDefault()
        goForward()
      }
    }

    window.addEventListener('keydown', handler)
    window.addEventListener('mouseup', mouseHandler)
    window.addEventListener('mousedown', mouseHandler)
    return () => {
      window.removeEventListener('keydown', handler)
      window.removeEventListener('mouseup', mouseHandler)
      window.removeEventListener('mousedown', mouseHandler)
    }
  }, [setActiveTool])
}
