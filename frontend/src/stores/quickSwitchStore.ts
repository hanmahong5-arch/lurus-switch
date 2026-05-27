import { create } from 'zustand'

interface QuickSwitchState {
  open: boolean
  open_(): void
  close(): void
}

export const useQuickSwitchStore = create<QuickSwitchState>((set) => ({
  open: false,
  open_: () => set({ open: true }),
  close: () => set({ open: false }),
}))
