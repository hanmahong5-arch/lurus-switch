import { serverSupabaseUser, serverSupabaseClient } from '#supabase/server'

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig()
  const query = getQuery(event)
  const limit = parseInt(query.limit as string) || 10

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
  const { data: profile, error: profileError } = await supabase
    .from('user_profiles')
    .select('new_api_id')
    .eq('id', user.id)
    .single()

  if (profileError || !profile?.new_api_id) {
    return []
  }

  try {
    // Fetch usage history from Billing Service
    const response = await $fetch<any>(
      `${config.billingServiceUrl}/api/v1/billing/stats/${profile.new_api_id}`,
      {
        query: { limit },
      }
    )

    // Transform to frontend format
    return (response?.records || []).map((record: any) => ({
      id: record.id || record.trace_id,
      time: new Date(record.timestamp || record.created_at).toLocaleTimeString('en-US', {
        hour: '2-digit',
        minute: '2-digit',
      }),
      model: record.model,
      provider: record.provider,
      inputTokens: record.input_tokens || 0,
      outputTokens: record.output_tokens || 0,
      cost: record.total_cost || 0,
    }))
  } catch (e: any) {
    console.error('Failed to fetch usage history:', e)
    return []
  }
})
