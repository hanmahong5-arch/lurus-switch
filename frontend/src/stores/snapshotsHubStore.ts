import { create } from 'zustand'

interface State {
  open: boolean
  // When set, the hub opens with this tool pre-selected in the
  // left-side filter. Otherwise it shows "all tools".
  focusTool: string | null
  setOpen: (open: boolean, focusTool?: string | null) => void
}

export const useSnapshotsHubStore = create<State>((set) => ({
  open: false,
  focusTool: null,
  setOpen: (open, focusTool = null) => set({ open, focusTool }),
}))
