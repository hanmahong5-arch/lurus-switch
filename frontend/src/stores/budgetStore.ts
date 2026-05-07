import { create } from 'zustand'

interface BudgetState {
  open: boolean
  setOpen: (open: boolean) => void
}

export const useBudgetStore = create<BudgetState>((set) => ({
  open: false,
  setOpen: (open) => set({ open }),
}))
