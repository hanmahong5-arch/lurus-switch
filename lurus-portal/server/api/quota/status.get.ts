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
  const { data: profile, error: profileError } = await supabase
    .from('user_profiles')
    .select('new_api_id')
    .eq('id', user.id)
    .single()

  if (profileError || !profile?.new_api_id) {
    // Return default quota for users not synced with NEW-API
    return {
      dailyQuota: 100,
      dailyUsed: 0,
      dailyRemaining: 100,
      monthlyQuota: 500,
      monthlyUsed: 0,
      monthlyRemaining: 500,
      currentGroup: 'free',
      originalGroup: 'free',
      isFallback: false,
      balance: 0,
      allowed: true,
    }
  }

  try {
    // Fetch quota from Billing Service
    const billingResponse = await $fetch<any>(
      `${config.billingServiceUrl}/api/v1/billing/quota/${profile.new_api_id}`
    )

    // Fetch subscription status from Subscription Service
    let subscriptionData = null
    try {
      subscriptionData = await $fetch<any>(
        `${config.subscriptionServiceUrl}/api/v1/quota/${profile.new_api_id}/status`
      )
    } catch {
      // Subscription service might not be running
    }

    return {
      dailyQuota: subscriptionData?.daily_quota ?? billingResponse?.daily_quota ?? 100,
      dailyUsed: subscriptionData?.daily_used ?? billingResponse?.daily_used ?? 0,
      dailyRemaining: subscriptionData?.daily_remaining ?? billingResponse?.daily_remaining ?? 100,
      monthlyQuota: billingResponse?.quota_limit ?? 500,
      monthlyUsed: billingResponse?.quota_used ?? 0,
      monthlyRemaining: billingResponse?.quota_remaining ?? 500,
      currentGroup: subscriptionData?.current_group ?? billingResponse?.group ?? 'free',
      originalGroup: subscriptionData?.original_group ?? billingResponse?.group ?? 'free',
      isFallback: subscriptionData?.is_fallback ?? false,
      balance: billingResponse?.balance ?? 0,
      allowed: billingResponse?.allowed ?? true,
    }
  } catch (e: any) {
    console.error('Failed to fetch quota:', e)

    // Return default values on error
    return {
      dailyQuota: 100,
      dailyUsed: 0,
      dailyRemaining: 100,
      monthlyQuota: 500,
      monthlyUsed: 0,
      monthlyRemaining: 500,
      currentGroup: 'free',
      originalGroup: 'free',
      isFallback: false,
      balance: 0,
      allowed: true,
    }
  }
})
