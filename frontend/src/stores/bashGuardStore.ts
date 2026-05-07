import { create } from 'zustand'

interface BashGuardState {
  open: boolean
  setOpen: (open: boolean) => void
}

export const useBashGuardStore = create<BashGuardState>((set) => ({
  open: false,
  setOpen: (open) => set({ open }),
}))
