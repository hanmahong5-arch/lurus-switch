import { create } from 'zustand'

export type ToastType = 'success' | 'error' | 'warning' | 'info'

export interface Toast {
  id: string
  type: ToastType
  message: string
  /** Optional action button (e.g. retry, navigate) */
  action?: {
    label: string
    onClick: () => void
  }
}

/** Auto-dismiss durations by type (ms). Error toasts stay longer. */
const DURATIONS: Record<ToastType, number> = {
  success: 3000,
  info: 4000,
  warning: 5000,
  error: 8000,
}

let nextId = 0

interface ToastState {
  toasts: Toast[]
  addToast: (type: ToastType, message: string, action?: Toast['action']) => void
  dismissToast: (id: string) => void
}

export const useToastStore = create<ToastState>((set) => ({
  toasts: [],

  addToast: (type, message, action) => {
    const id = `toast-${++nextId}`
    set((state) => ({
      toasts: [...state.toasts.slice(-4), { id, type, message, action }],
    }))
    setTimeout(() => {
      set((state) => ({
        toasts: state.toasts.filter((t) => t.id !== id),
      }))
    }, DURATIONS[type])
  },

  dismissToast: (id) =>
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    })),
}))
