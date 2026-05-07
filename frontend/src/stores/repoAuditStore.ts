import { create } from 'zustand'

// Tiny store for the Repo Trust Audit modal — opened from the Home
// intent panel and from the Command Palette. Lives at App-level so the
// modal is mounted once and any caller can pop it open.
interface RepoAuditState {
  open: boolean
  setOpen: (open: boolean) => void
}

export const useRepoAuditStore = create<RepoAuditState>((set) => ({
  open: false,
  setOpen: (open) => set({ open }),
}))
