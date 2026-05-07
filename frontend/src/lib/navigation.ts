import { useConfigStore } from '../stores/configStore'
import { useNavHistoryStore, type NavEntry } from '../stores/navHistoryStore'
import { confirmIfDirty } from './dirtyGuard'

function applyEntrySilently(entry: NavEntry) {
  const store = useConfigStore.getState()
  if (entry.subTab) {
    store.setSubTabSilent(entry.tool, entry.subTab)
  }
  store.setActiveToolSilent(entry.tool)
}

export function goBack(): boolean {
  if (!confirmIfDirty()) return false
  const target = useNavHistoryStore.getState().back()
  if (!target) return false
  applyEntrySilently(target)
  return true
}

export function goForward(): boolean {
  if (!confirmIfDirty()) return false
  const target = useNavHistoryStore.getState().forward()
  if (!target) return false
  applyEntrySilently(target)
  return true
}
