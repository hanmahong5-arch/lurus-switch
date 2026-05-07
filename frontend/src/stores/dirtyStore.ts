import { create } from 'zustand'

// Tracks pages that have unsaved changes. Pages register themselves via
// useDirtyGuard; navigation primitives (goBack/goForward, setActiveTool,
// setSubTab) consult hasDirty() before mutating state and prompt the user
// if any page is dirty.
interface DirtyState {
  dirtyPages: Set<string>
  setDirty: (pageId: string, isDirty: boolean) => void
  hasDirty: () => boolean
  list: () => string[]
  clear: () => void
}

export const useDirtyStore = create<DirtyState>((set, get) => ({
  dirtyPages: new Set(),
  setDirty: (pageId, isDirty) =>
    set((state) => {
      const has = state.dirtyPages.has(pageId)
      if (isDirty === has) return {}
      const next = new Set(state.dirtyPages)
      if (isDirty) next.add(pageId)
      else next.delete(pageId)
      return { dirtyPages: next }
    }),
  hasDirty: () => get().dirtyPages.size > 0,
  list: () => Array.from(get().dirtyPages),
  clear: () => set({ dirtyPages: new Set() }),
}))
