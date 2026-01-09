import { serverSupabaseUser, serverSupabaseClient } from '#supabase/server'

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig()
  const body = await readBody(event)

  // Get authenticated user
  const user = await serverSupabaseUser(event)
  if (!user) {
    throw createError({
      statusCode: 401,
      message: 'Unauthorized',
    })
  }

  const email = body.email || user.email
  if (!email) {
    throw createError({
      statusCode: 400,
      message: 'Email is required',
    })
  }

  const supabase = await serverSupabaseClient(event)

  // Check if user already synced
  const { data: existingProfile } = await supabase
    .from('user_profiles')
    .select('new_api_id')
    .eq('id', user.id)
    .single()

  if (existingProfile?.new_api_id) {
    return { newApiId: existingProfile.new_api_id }
  }

  try {
    // Try to find existing NEW-API user by email
    const searchResponse = await $fetch<any>(
      `${config.newApiUrl}/api/user/search`,
      {
        method: 'GET',
        query: { keyword: email },
        headers: {
          'Authorization': `Bearer ${process.env.NEW_API_ADMIN_TOKEN || ''}`,
        },
      }
    ).catch(() => null)

    if (searchResponse?.data?.length > 0) {
      // Found existing user
      const newApiUser = searchResponse.data[0]

      // Update profile with new_api_id
      await supabase
        .from('user_profiles')
        .update({ new_api_id: newApiUser.id })
        .eq('id', user.id)

      return { newApiId: newApiUser.id }
    }

    // Create new user in NEW-API
    const createResponse = await $fetch<any>(
      `${config.newApiUrl}/api/user/register`,
      {
        method: 'POST',
        body: {
          username: email.split('@')[0] + '_' + Date.now().toString(36),
          password: crypto.randomUUID(), // Random password, user will use Supabase auth
          email: email,
        },
      }
    ).catch(() => null)

    if (createResponse?.success && createResponse?.data?.id) {
      // Update profile with new_api_id
      await supabase
        .from('user_profiles')
        .update({ new_api_id: createResponse.data.id })
        .eq('id', user.id)

      return { newApiId: createResponse.data.id }
    }

    // If registration fails, create a placeholder ID based on Supabase ID
    // This allows the app to function while backend sync is pending
    const placeholderId = Math.abs(hashCode(user.id)) % 1000000

    await supabase
      .from('user_profiles')
      .update({ new_api_id: placeholderId })
      .eq('id', user.id)

    return { newApiId: placeholderId, placeholder: true }
  } catch (error: any) {
    console.error('Failed to sync with NEW-API:', error)

    // Return a placeholder ID on error
    const placeholderId = Math.abs(hashCode(user.id)) % 1000000
    return { newApiId: placeholderId, error: error.message }
  }
})

// Simple hash function for generating placeholder IDs
function hashCode(str: string): number {
  let hash = 0
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i)
    hash = ((hash << 5) - hash) + char
    hash = hash & hash
  }
  return hash
}
