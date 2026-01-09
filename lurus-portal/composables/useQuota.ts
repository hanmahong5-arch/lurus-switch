export interface QuotaStatus {
  dailyQuota: number
  dailyUsed: number
  dailyRemaining: number
  monthlyQuota: number
  monthlyUsed: number
  monthlyRemaining: number
  currentGroup: string
  originalGroup: string
  isFallback: boolean
  balance: number
  allowed: boolean
}

export interface UsageRecord {
  id: string
  time: string
  model: string
  provider: string
  inputTokens: number
  outputTokens: number
  cost: number
}

export const useQuota = () => {
  const { profile } = useAuth()

  const quota = useState<QuotaStatus>('quota-status', () => ({
    dailyQuota: 0,
    dailyUsed: 0,
    dailyRemaining: 0,
    monthlyQuota: 0,
    monthlyUsed: 0,
    monthlyRemaining: 0,
    currentGroup: 'free',
    originalGroup: 'free',
    isFallback: false,
    balance: 0,
    allowed: true,
  }))

  const recentUsage = useState<UsageRecord[]>('recent-usage', () => [])
  const loading = useState<boolean>('quota-loading', () => false)
  const connected = useState<boolean>('quota-connected', () => false)

  let eventSource: EventSource | null = null

  // Fetch quota status
  const fetchQuota = async () => {
    if (!profile.value?.new_api_id) return

    loading.value = true
    try {
      const data = await $fetch<QuotaStatus>('/api/quota/status')
      quota.value = data
    } catch (e) {
      console.error('Failed to fetch quota:', e)
    } finally {
      loading.value = false
    }
  }

  // Connect to SSE for real-time updates
  const connectRealtime = () => {
    if (!profile.value?.new_api_id || typeof window === 'undefined') return

    // Close existing connection
    disconnectRealtime()

    const userId = profile.value.new_api_id
    eventSource = new EventSource(`/api/billing/stream?userId=${userId}`)

    eventSource.onopen = () => {
      connected.value = true
      console.log('SSE connected')
    }

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)

        if (data.type === 'sync' || data.type === 'heartbeat') {
          // Update quota from SSE data
          quota.value = {
            ...quota.value,
            monthlyUsed: data.quota_used ?? quota.value.monthlyUsed,
            monthlyRemaining: data.quota_remaining ?? quota.value.monthlyRemaining,
            balance: data.balance ?? quota.value.balance,
            allowed: data.allowed ?? quota.value.allowed,
          }
        }

        if (data.type === 'quota_updated') {
          // Full quota update
          fetchQuota()
        }

        if (data.type === 'quota_low') {
          // Show warning notification
          console.warn('Quota running low:', data)
        }

        if (data.type === 'quota_exhausted') {
          // Show error notification
          console.error('Quota exhausted:', data)
        }
      } catch (e) {
        console.error('Failed to parse SSE message:', e)
      }
    }

    eventSource.onerror = () => {
      connected.value = false
      console.error('SSE connection error, will retry...')

      // Reconnect after 5 seconds
      setTimeout(() => {
        if (profile.value?.new_api_id) {
          connectRealtime()
        }
      }, 5000)
    }
  }

  // Disconnect SSE
  const disconnectRealtime = () => {
    if (eventSource) {
      eventSource.close()
      eventSource = null
      connected.value = false
    }
  }

  // Fetch recent usage history
  const fetchRecentUsage = async (limit = 10) => {
    if (!profile.value?.new_api_id) return

    try {
      const data = await $fetch<UsageRecord[]>('/api/quota/history', {
        query: { limit },
      })
      recentUsage.value = data
    } catch (e) {
      console.error('Failed to fetch usage history:', e)
    }
  }

  // Calculate percentages
  const dailyPercentage = computed(() =>
    quota.value.dailyQuota > 0
      ? Math.round((quota.value.dailyUsed / quota.value.dailyQuota) * 100)
      : 0
  )

  const monthlyPercentage = computed(() =>
    quota.value.monthlyQuota > 0
      ? Math.round((quota.value.monthlyUsed / quota.value.monthlyQuota) * 100)
      : 0
  )

  // Status helpers
  const isQuotaLow = computed(() => monthlyPercentage.value >= 80)
  const isQuotaExhausted = computed(() => monthlyPercentage.value >= 100)
  const isDailyExhausted = computed(() => quota.value.dailyRemaining <= 0)

  // Cleanup on unmount
  onUnmounted(() => {
    disconnectRealtime()
  })

  return {
    quota,
    recentUsage,
    loading,
    connected,
    dailyPercentage,
    monthlyPercentage,
    isQuotaLow,
    isQuotaExhausted,
    isDailyExhausted,
    fetchQuota,
    fetchRecentUsage,
    connectRealtime,
    disconnectRealtime,
  }
}
