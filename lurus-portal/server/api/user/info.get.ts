import { serverSupabaseUser, serverSupabaseClient } from '#supabase/server'

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig()

  // Get authenticated user
  const user = await serverSupabaseUser(event)
  if (!user) {
    throw createError({
      statusCode: 401,
      message: 'Unauthorized',
    })
  }

  // Get user profile with new_api_id
  const supabase = await serverSupabaseClient(event)
  const { data: profile } = await supabase
    .from('user_profiles')
    .select('new_api_id')
    .eq('id', user.id)
    .single()

  if (!profile?.new_api_id) {
    return {
      id: 0,
      username: user.email?.split('@')[0] || 'user',
      email: user.email,
      group: 'free',
      quota: 500,
      used_quota: 0,
      request_count: 0,
    }
  }

  try {
    // Fetch user info from NEW-API
    const response = await $fetch<any>(
      `${config.newApiUrl}/api/user/${profile.new_api_id}`,
      {
        headers: {
          'Authorization': `Bearer ${process.env.NEW_API_ADMIN_TOKEN || ''}`,
        },
      }
    )

    if (response?.success && response?.data) {
      const userData = response.data
      return {
        id: userData.id,
        username: userData.username,
        email: userData.email,
        group: userData.group || 'free',
        quota: userData.quota || 500,
        used_quota: userData.used_quota || 0,
        request_count: userData.request_count || 0,
      }
    }
  } catch (e) {
    console.error('Failed to fetch NEW-API user:', e)
  }

  // Return default on error
  return {
    id: profile.new_api_id,
    username: user.email?.split('@')[0] || 'user',
    email: user.email,
    group: 'free',
    quota: 500,
    used_quota: 0,
    request_count: 0,
  }
})
