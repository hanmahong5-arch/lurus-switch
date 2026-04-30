import { useEffect } from 'react'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { StartGateway, StopGateway, OpenConfigDir, GetServerStatus } from '../../wailsjs/go/main/App'
import { useCommandPaletteStore } from '../stores/commandPaletteStore'
import { useToastStore } from '../stores/toastStore'
import { useDeepLinkImportStore, type DeepLinkPayload } from '../stores/deeplinkImportStore'

/**
 * Subscribes to all backend-originated desktop events:
 *   - tray:*     — system tray menu clicks
 *   - hotkey:*   — global shortcut presses
 *   - deeplink:* — switch:// URL imports
 *
 * Each subscription is torn down on unmount to avoid duplicate handlers
 * when the hook re-runs.
 */
export function usePlatformEvents() {
  const addToast = useToastStore((s) => s.addToast)

  useEffect(() => {
    const offs: Array<() => void> = []

    // Hotkeys — backend already WindowShow()s before emitting.
    offs.push(EventsOn('hotkey:quickSwitch', () => {
      useCommandPaletteStore.getState().setOpen(true)
    }))
    offs.push(EventsOn('hotkey:showWindow', () => {
      // Window is already restored by the backend; nothing else to do.
    }))

    // Tray menu clicks.
    offs.push(EventsOn('tray:switch-provider', () => {
      useCommandPaletteStore.getState().setOpen(true)
    }))
    offs.push(EventsOn('tray:gateway-toggle', async () => {
      try {
        const status = await GetServerStatus()
        if ((status as { running?: boolean })?.running) {
          await StopGateway()
          addToast('success', 'Gateway stopped')
        } else {
          await StartGateway()
          addToast('success', 'Gateway started')
        }
      } catch (err) {
        addToast('error', `Gateway toggle failed: ${(err as Error)?.message ?? String(err)}`)
      }
    }))
    offs.push(EventsOn('tray:open-config-dir', () => {
      OpenConfigDir().catch((err) => {
        addToast('error', `Open config dir failed: ${(err as Error)?.message ?? String(err)}`)
      })
    }))

    // Deep-link import — open confirmation modal so the user can review the
    // payload before it's applied. Apply writes baseURL into proxy settings
    // and routes to /gateway for the user to paste their API key.
    offs.push(EventsOn('deeplink:import', (payload: DeepLinkPayload) => {
      useDeepLinkImportStore.getState().openWith(payload)
    }))

    return () => {
      for (const off of offs) off()
    }
  }, [addToast])
}
