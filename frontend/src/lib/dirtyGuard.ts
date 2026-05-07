import { useDirtyStore } from '../stores/dirtyStore'
import i18n from '../i18n'

// Single chokepoint for "are you sure you want to leave with unsaved changes?"
// dialog. Returns true if it's safe to proceed (either nothing dirty, or user
// confirmed discard). Clears the dirty registry on confirm so subsequent nav
// in the same intent doesn't re-prompt.
export function confirmIfDirty(): boolean {
  const dirty = useDirtyStore.getState()
  if (!dirty.hasDirty()) return true
  if (typeof window === 'undefined' || typeof window.confirm !== 'function') {
    // Headless / non-browser env (e.g. tests) — allow through.
    dirty.clear()
    return true
  }
  const msg = i18n.t(
    'common.unsavedChangesConfirm',
    '当前页面有未保存的改动，离开会丢失。是否继续？',
  )
  if (!window.confirm(msg)) return false
  dirty.clear()
  return true
}
