import { create } from 'zustand'

export interface DeepLinkProviderData {
  id?: string
  name?: string
  baseUrl?: string
  icon?: string
  iconColor?: string
  category?: string
  keyFormat?: string
  docsUrl?: string
  models?: string
  description?: string
  freeTier?: boolean
  needsProxy?: boolean
}

export interface DeepLinkPayload {
  type: string
  data: unknown
  raw: string
}

interface DeepLinkImportState {
  open: boolean
  payload: DeepLinkPayload | null
  openWith: (payload: DeepLinkPayload) => void
  close: () => void
}

export const useDeepLinkImportStore = create<DeepLinkImportState>((set) => ({
  open: false,
  payload: null,
  openWith: (payload) => set({ open: true, payload }),
  close: () => set({ open: false, payload: null }),
}))
