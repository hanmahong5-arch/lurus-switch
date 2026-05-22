import { create } from 'zustand'

type RefreshFn = () => void | Promise<void>

interface PageActionsState {
  refreshHandler: RefreshFn | null
  refreshing: boolean
  setRefreshHandler: (fn: RefreshFn | null) => void
  runRefresh: () => Promise<void>
}

// Lets the global PageHeader trigger a page-specific refresh without coupling
// the header to individual page components. A page registers its refresh in
// useEffect; PageHeader shows the icon only when a handler is registered.
export const usePageActionsStore = create<PageActionsState>((set, get) => ({
  refreshHandler: null,
  refreshing: false,
  setRefreshHandler: (fn) => set({ refreshHandler: fn }),
  runRefresh: async () => {
    const fn = get().refreshHandler
    if (!fn || get().refreshing) return
    set({ refreshing: true })
    try {
      await fn()
    } finally {
      set({ refreshing: false })
    }
  },
}))
