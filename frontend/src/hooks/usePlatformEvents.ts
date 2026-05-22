import { useEffect } from 'react'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { StartGateway, StopGateway, OpenConfigDir, GetServerStatus, ApplyAllToolRelays, SaveToolRelayMapping, GetToolRelayMapping, ListConversations } from '../../wailsjs/go/main/App'
import { useCommandPaletteStore } from '../stores/commandPaletteStore'
import { useToastStore } from '../stores/toastStore'
import { useDeepLinkImportStore, type DeepLinkPayload } from '../stores/deeplinkImportStore'
import { useConfigStore } from '../stores/configStore'
import { useConversationStore } from '../stores/conversationStore'

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
    // Window surface is already done by the backend (WindowShow in the
    // hotkey callback). Here we only need to route to the Live Sessions
    // Inspector page so one keystroke lands the user there.
    offs.push(EventsOn('hotkey:show-live', () => {
      useConfigStore.getState().setActiveTool('live')
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

    // Tray relay quick-switch: payload is the RelayEndpoint ID to map
    // every configured tool to, then re-apply across all tool configs.
    offs.push(EventsOn('tray:apply-relay', async (relayID: string) => {
      if (!relayID) return
      try {
        const current = (await GetToolRelayMapping()) || {}
        const next: Record<string, string> = {}
        // Map all known tools to the chosen relay so "switch all" actually
        // routes them all — not just the ones the user had previously bound.
        for (const tool of ['claude', 'codex', 'gemini', 'picoclaw', 'nullclaw', 'zeroclaw', 'openclaw']) {
          next[tool] = relayID
        }
        await SaveToolRelayMapping({ ...current, ...next })
        const results = await ApplyAllToolRelays()
        const errors = Object.entries(results || {}).filter(([, v]) => v)
        if (errors.length === 0) {
          addToast('success', 'Switched all tools to selected relay')
        } else {
          addToast('warning', `Partial switch: ${errors.length} tool(s) failed`)
        }
      } catch (err) {
        addToast('error', `Switch relay failed: ${(err as Error)?.message ?? String(err)}`)
      }
    }))

    // Tray "Open last conversation" — jump to the Conversations page and
    // pre-open the most-recent session.
    offs.push(EventsOn('tray:open-last-session', async () => {
      try {
        const rows = await ListConversations({
          tool: '', cwdSubstring: '', model: '', startAfter: '', endBefore: '',
          onlyDLPHits: false, search: '',
        } as any)
        useConfigStore.getState().setActiveTool('conversations')
        if (rows && rows.length > 0) {
          await useConversationStore.getState().open(rows[0].tool, rows[0].sessionID)
        }
      } catch (err) {
        addToast('error', `Open last session failed: ${(err as Error)?.message ?? String(err)}`)
      }
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
