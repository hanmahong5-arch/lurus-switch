import { useEffect } from 'react'
import { useDirtyStore } from '../stores/dirtyStore'

/**
 * Registers this page/form as having unsaved changes when `dirty` is true.
 * Auto-clears on unmount so a forced unmount (e.g. after user confirms
 * discard) doesn't leave the registry sticky.
 */
export function useDirtyGuard(pageId: string, dirty: boolean) {
  const setDirty = useDirtyStore((s) => s.setDirty)
  useEffect(() => {
    setDirty(pageId, dirty)
  }, [pageId, dirty, setDirty])
  useEffect(() => {
    return () => {
      setDirty(pageId, false)
    }
  }, [pageId, setDirty])
}
