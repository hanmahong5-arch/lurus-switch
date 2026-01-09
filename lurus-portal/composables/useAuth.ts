import type { User } from '@supabase/supabase-js'

export interface UserProfile {
  id: string
  new_api_id: number | null
  display_name: string | null
  avatar_url: string | null
  timezone: string
  language: string
  created_at: string
}

export interface NewApiUser {
  id: number
  username: string
  email: string
  group: string
  quota: number
  used_quota: number
  request_count: number
}

export const useAuth = () => {
  const supabase = useSupabaseClient()
  const user = useSupabaseUser()
  const profile = useState<UserProfile | null>('user-profile', () => null)
  const newApiUser = useState<NewApiUser | null>('new-api-user', () => null)
  const loading = useState<boolean>('auth-loading', () => false)
  const error = useState<string | null>('auth-error', () => null)

  // Sign in with email and password
  const signInWithEmail = async (email: string, password: string) => {
    loading.value = true
    error.value = null

    try {
      const { data, error: signInError } = await supabase.auth.signInWithPassword({
        email,
        password,
      })

      if (signInError) {
        throw signInError
      }

      // Sync with NEW-API after login
      await syncProfile()

      return data
    } catch (e: any) {
      error.value = e.message || 'Login failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  // Sign up with email and password
  const signUpWithEmail = async (email: string, password: string, displayName?: string) => {
    loading.value = true
    error.value = null

    try {
      const { data, error: signUpError } = await supabase.auth.signUp({
        email,
        password,
        options: {
          data: {
            display_name: displayName,
          },
        },
      })

      if (signUpError) {
        throw signUpError
      }

      return data
    } catch (e: any) {
      error.value = e.message || 'Registration failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  // Sign in with OAuth provider
  const signInWithProvider = async (provider: 'github' | 'google') => {
    loading.value = true
    error.value = null

    try {
      const { data, error: oauthError } = await supabase.auth.signInWithOAuth({
        provider,
        options: {
          redirectTo: `${window.location.origin}/auth/callback`,
        },
      })

      if (oauthError) {
        throw oauthError
      }

      return data
    } catch (e: any) {
      error.value = e.message || 'OAuth login failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  // Sign out
  const signOut = async () => {
    loading.value = true
    error.value = null

    try {
      const { error: signOutError } = await supabase.auth.signOut()

      if (signOutError) {
        throw signOutError
      }

      // Clear state
      profile.value = null
      newApiUser.value = null

      // Redirect to home
      await navigateTo('/')
    } catch (e: any) {
      error.value = e.message || 'Logout failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  // Reset password
  const resetPassword = async (email: string) => {
    loading.value = true
    error.value = null

    try {
      const { error: resetError } = await supabase.auth.resetPasswordForEmail(email, {
        redirectTo: `${window.location.origin}/auth/reset-password`,
      })

      if (resetError) {
        throw resetError
      }
    } catch (e: any) {
      error.value = e.message || 'Password reset failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  // Sync user profile with Supabase and NEW-API
  const syncProfile = async () => {
    if (!user.value) return

    try {
      // Fetch profile from Supabase
      const { data: profileData, error: profileError } = await supabase
        .from('user_profiles')
        .select('*')
        .eq('id', user.value.id)
        .single()

      if (profileError && profileError.code !== 'PGRST116') {
        console.error('Failed to fetch profile:', profileError)
      }

      if (profileData) {
        profile.value = profileData as UserProfile

        // If we have new_api_id, fetch NEW-API user info
        if (profileData.new_api_id) {
          await fetchNewApiUser()
        } else {
          // Try to sync with NEW-API
          await syncWithNewApi()
        }
      } else {
        // Create profile if not exists
        const { data: newProfile, error: createError } = await supabase
          .from('user_profiles')
          .insert({
            id: user.value.id,
            display_name: user.value.user_metadata?.display_name || user.value.email?.split('@')[0],
          })
          .select()
          .single()

        if (!createError && newProfile) {
          profile.value = newProfile as UserProfile
          await syncWithNewApi()
        }
      }
    } catch (e) {
      console.error('Failed to sync profile:', e)
    }
  }

  // Sync with NEW-API (create user if not exists)
  const syncWithNewApi = async () => {
    if (!user.value) return

    try {
      const response = await $fetch('/api/auth/sync-newapi', {
        method: 'POST',
        body: {
          email: user.value.email,
          supabaseId: user.value.id,
        },
      })

      if (response?.newApiId) {
        // Update profile with new_api_id
        await supabase
          .from('user_profiles')
          .update({ new_api_id: response.newApiId })
          .eq('id', user.value.id)

        if (profile.value) {
          profile.value.new_api_id = response.newApiId
        }

        await fetchNewApiUser()
      }
    } catch (e) {
      console.error('Failed to sync with NEW-API:', e)
    }
  }

  // Fetch NEW-API user info
  const fetchNewApiUser = async () => {
    if (!profile.value?.new_api_id) return

    try {
      const response = await $fetch<NewApiUser>('/api/user/info')
      newApiUser.value = response
    } catch (e) {
      console.error('Failed to fetch NEW-API user:', e)
    }
  }

  // Initialize auth state
  const init = async () => {
    if (user.value) {
      await syncProfile()
    }
  }

  return {
    user,
    profile,
    newApiUser,
    loading,
    error,
    signInWithEmail,
    signUpWithEmail,
    signInWithProvider,
    signOut,
    resetPassword,
    syncProfile,
    init,
  }
}
