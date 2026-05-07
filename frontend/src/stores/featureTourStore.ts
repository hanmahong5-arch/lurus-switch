import { create } from 'zustand'

interface FeatureTourState {
  open: boolean
  setOpen: (open: boolean) => void
}

export const useFeatureTourStore = create<FeatureTourState>((set) => ({
  open: false,
  setOpen: (open) => set({ open }),
}))
