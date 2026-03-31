import { create } from 'zustand'
import { GetAuthState, Login, Logout, RefreshAuth } from '../../wailsjs/go/main/App'

export interface UserInfo {
  sub: string
  name: string
  email: string
  picture: string
}

export interface AuthState {
  is_logged_in: boolean
  user?: UserInfo
  expires_at?: string
  has_gateway_token?: boolean
}

interface AuthStoreState {
  authState: AuthState
  isLoggingIn: boolean
  loginError: string | null

  /** Load the current auth state from the backend. */
  load: () => Promise<void>
  /** Start the PKCE login flow (opens browser). */
  login: () => Promise<void>
  /** Clear the session and log out. */
  logout: () => Promise<void>
  /** Refresh the access token silently. */
  refresh: () => Promise<void>
}

export const useAuthStore = create<AuthStoreState>((set) => ({
  authState: { is_logged_in: false },
  isLoggingIn: false,
  loginError: null,

  load: async () => {
    try {
      const state = await GetAuthState()
      set({ authState: state ?? { is_logged_in: false }, loginError: null })
    } catch {
      set({ authState: { is_logged_in: false } })
    }
  },

  login: async () => {
    set({ isLoggingIn: true, loginError: null })
    try {
      const state = await Login()
      set({ authState: state ?? { is_logged_in: false }, isLoggingIn: false })
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      set({ isLoggingIn: false, loginError: message })
    }
  },

  logout: async () => {
    try {
      await Logout()
    } catch {
      // Swallow — clear local state regardless.
    }
    set({ authState: { is_logged_in: false }, loginError: null })
  },

  refresh: async () => {
    try {
      const state = await RefreshAuth()
      set({ authState: state ?? { is_logged_in: false } })
    } catch {
      set({ authState: { is_logged_in: false } })
    }
  },
}))
