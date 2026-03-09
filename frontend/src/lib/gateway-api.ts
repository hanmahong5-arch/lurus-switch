/** Pre-configured HTTP client for the local gateway (lurus-newapi) API. */

// --- Base Types ---

export interface GatewayChannel {
  id: number
  name: string
  type: number
  key: string
  base_url: string
  models: string
  balance: number
  status: number
  response_time: number
  created_time: number
  test_time: number
  tag: string
  group: string
  model_mapping: string
  priority: number
  weight: number
  auto_ban: number
  other: string
}

export interface GatewayToken {
  id: number
  name: string
  key: string
  status: number
  quota: number
  used_quota: number
  expired_time: number
  unlimited_quota: boolean
  created_time: number
  remain_quota: number
  model_limits: string
  subnet: string
  group: string
}

export interface GatewayUser {
  id: number
  username: string
  display_name: string
  role: number
  status: number
  quota: number
  used_quota: number
  created_time: number
  email: string
  group: string
  aff_code: string
  request_count: number
}

export interface GatewayLog {
  id: number
  user_id: number
  created_at: number
  type: number
  content: string
  username: string
  token_name: string
  model: string
  quota: number
  prompt_tokens: number
  completion_tokens: number
  channel_id: number
  channel_name: string
}

export interface GatewayRedemption {
  id: number
  name: string
  key: string
  status: number
  quota: number
  count: number
  used_count: number
  created_time: number
  redeemed_time: number
}

export interface GatewayModelMeta {
  model_name: string
  developer: string
  type: string
  context_length: number
  input_price: number
  output_price: number
  vendor_id: number
  tags: string[]
  status: number
}

export interface GatewayVendor {
  id: number
  name: string
  description: string
  icon_url: string
  website: string
  status: number
}

export interface GatewaySubscriptionPlan {
  id: number
  name: string
  description: string
  level: number
  pricing: number
  duration: number
  quota: number
  group: string
  features: string
  status: number
}

export interface GatewayOption {
  key: string
  value: string
}

export interface GatewayDashboardData {
  user_count: number
  channel_count: number
  token_count: number
  today_request: number
  today_quota: number
  today_tokens: number
}

export interface GatewayPerformanceStats {
  goroutines: number
  memory_alloc: number
  uptime: number
  requests_total: number
  requests_per_sec: number
}

export interface GatewayLogStat {
  date: string
  request_count: number
  quota: number
  token_count: number
}

export interface GatewayQuotaDate {
  date: string
  quota: number
  request_count: number
  token_count: number
  model_usage: Record<string, number>
}

export interface GatewayGroup {
  name: string
  ratio: number
}

// --- Response Types ---

export interface GatewayApiResponse<T> {
  success: boolean
  message: string
  data?: T
}

export interface PaginatedResponse<T> {
  success: boolean
  message: string
  data: T[]
}

// --- Channel Type Constants ---

export const CHANNEL_TYPES: Record<number, string> = {
  1: 'OpenAI',
  2: 'Custom',
  3: 'Azure OpenAI',
  4: 'CloseAI',
  5: 'OpenAI-SB',
  6: 'OpenAI Max',
  7: 'OhMyGPT',
  8: 'Custom (Adj)',
  9: 'AI.LS',
  10: 'AI.LS (Adj)',
  11: 'API2D',
  12: 'ForwardAI',
  13: 'AI Proxy',
  14: 'Anthropic',
  15: 'Baidu',
  16: 'Zhipu',
  17: 'Ali',
  18: 'Xunfei',
  19: 'AI360',
  20: 'Tencent',
  21: 'Proxy',
  22: 'Google PaLM',
  23: 'Baichuan',
  24: 'Google Gemini',
  25: 'Moonshot',
  26: 'Perplexity',
  27: 'Lingyi',
  28: 'Groq',
  29: 'Claude (AWS)',
  30: 'Cohere',
  31: 'DeepSeek',
  32: 'Cloudflare',
  33: 'Mistral',
  34: 'Together AI',
  35: 'Novita',
  36: 'VertexAI',
  37: 'Coze',
  38: 'SiliconFlow',
  39: 'Doubao',
  40: 'Minimax',
  41: 'xAI',
  42: 'Replicate',
  43: 'HuggingFace',
  44: 'SambaNova',
  45: 'GitHub Models',
}

// --- API Client ---

export class GatewayAPI {
  private baseURL: string
  private token: string

  constructor(baseURL: string, token: string) {
    this.baseURL = baseURL.replace(/\/$/, '')
    this.token = token
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const url = `${this.baseURL}${path}`
    const res = await fetch(url, {
      method,
      headers: {
        'Content-Type': 'application/json',
        ...(this.token ? { Authorization: `Bearer ${this.token}` } : {}),
      },
      ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
    })
    if (!res.ok) {
      const text = await res.text().catch(() => res.statusText)
      throw new Error(`Gateway API ${method} ${path} → ${res.status}: ${text}`)
    }
    return res.json() as Promise<T>
  }

  // ========== Channels ==========

  getChannels(page = 0, perPage = 50): Promise<PaginatedResponse<GatewayChannel>> {
    return this.request('GET', `/api/channel/?p=${page}&page_size=${perPage}`)
  }

  getChannel(id: number): Promise<GatewayApiResponse<GatewayChannel>> {
    return this.request('GET', `/api/channel/${id}`)
  }

  searchChannels(keyword: string, page = 0, perPage = 50): Promise<PaginatedResponse<GatewayChannel>> {
    return this.request('GET', `/api/channel/search?keyword=${encodeURIComponent(keyword)}&p=${page}&page_size=${perPage}`)
  }

  createChannel(ch: Partial<GatewayChannel>): Promise<GatewayApiResponse<GatewayChannel>> {
    return this.request('POST', '/api/channel/', ch)
  }

  updateChannel(ch: Partial<GatewayChannel> & { id: number }): Promise<GatewayApiResponse<GatewayChannel>> {
    return this.request('PUT', '/api/channel/', ch)
  }

  deleteChannel(id: number): Promise<GatewayApiResponse<null>> {
    return this.request('DELETE', `/api/channel/${id}`)
  }

  batchDeleteChannels(ids: number[]): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/channel/batch', { ids, action: 'delete' })
  }

  batchEnableChannels(ids: number[]): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/channel/batch', { ids, action: 'enable' })
  }

  batchDisableChannels(ids: number[]): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/channel/batch', { ids, action: 'disable' })
  }

  batchSetChannelTag(ids: number[], tag: string): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/channel/batch/tag', { ids, tag })
  }

  enableChannelsByTag(tag: string): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/channel/tag/enabled', { tag })
  }

  disableChannelsByTag(tag: string): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/channel/tag/disabled', { tag })
  }

  editChannelTag(oldTag: string, newTag: string): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/channel/tag/edit', { old_tag: oldTag, new_tag: newTag })
  }

  copyChannel(id: number): Promise<GatewayApiResponse<GatewayChannel>> {
    return this.request('POST', `/api/channel/copy/${id}`)
  }

  testChannel(id: number): Promise<GatewayApiResponse<string>> {
    return this.request('GET', `/api/channel/test/${id}`)
  }

  fetchChannelModels(id: number): Promise<GatewayApiResponse<string[]>> {
    return this.request('GET', `/api/channel/fetch_models/${id}`)
  }

  updateChannelBalance(id: number): Promise<GatewayApiResponse<{ balance: number }>> {
    return this.request('GET', `/api/channel/update_balance/${id}`)
  }

  fixChannelAbilities(): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/channel/fix_abilities')
  }

  // ========== Tokens ==========

  getTokens(page = 0, perPage = 50): Promise<PaginatedResponse<GatewayToken>> {
    return this.request('GET', `/api/token/?p=${page}&page_size=${perPage}`)
  }

  getToken(id: number): Promise<GatewayApiResponse<GatewayToken>> {
    return this.request('GET', `/api/token/${id}`)
  }

  searchTokens(keyword: string, page = 0, perPage = 50): Promise<PaginatedResponse<GatewayToken>> {
    return this.request('GET', `/api/token/search?keyword=${encodeURIComponent(keyword)}&p=${page}&page_size=${perPage}`)
  }

  createToken(t: Partial<GatewayToken>): Promise<GatewayApiResponse<GatewayToken>> {
    return this.request('POST', '/api/token/', t)
  }

  updateToken(t: Partial<GatewayToken> & { id: number }): Promise<GatewayApiResponse<GatewayToken>> {
    return this.request('PUT', '/api/token/', t)
  }

  deleteToken(id: number): Promise<GatewayApiResponse<null>> {
    return this.request('DELETE', `/api/token/${id}`)
  }

  batchDeleteTokens(ids: number[]): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/token/batch', { ids, action: 'delete' })
  }

  // ========== Users ==========

  getUsers(page = 0, perPage = 50): Promise<PaginatedResponse<GatewayUser>> {
    return this.request('GET', `/api/user/?p=${page}&page_size=${perPage}`)
  }

  getUser(id: number): Promise<GatewayApiResponse<GatewayUser>> {
    return this.request('GET', `/api/user/${id}`)
  }

  searchUsers(keyword: string, page = 0, perPage = 50): Promise<PaginatedResponse<GatewayUser>> {
    return this.request('GET', `/api/user/search?keyword=${encodeURIComponent(keyword)}&p=${page}&page_size=${perPage}`)
  }

  createUser(u: Partial<GatewayUser> & { password?: string }): Promise<GatewayApiResponse<GatewayUser>> {
    return this.request('POST', '/api/user/', u)
  }

  updateUser(u: Partial<GatewayUser> & { id: number }): Promise<GatewayApiResponse<GatewayUser>> {
    return this.request('PUT', '/api/user/', u)
  }

  deleteUser(id: number): Promise<GatewayApiResponse<null>> {
    return this.request('DELETE', `/api/user/${id}`)
  }

  manageUser(payload: { id: number; action: string }): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/user/manage', payload)
  }

  // ========== Redemptions ==========

  getRedemptions(page = 0, perPage = 50): Promise<PaginatedResponse<GatewayRedemption>> {
    return this.request('GET', `/api/redemption/?p=${page}&page_size=${perPage}`)
  }

  getRedemption(id: number): Promise<GatewayApiResponse<GatewayRedemption>> {
    return this.request('GET', `/api/redemption/${id}`)
  }

  searchRedemptions(keyword: string, page = 0, perPage = 50): Promise<PaginatedResponse<GatewayRedemption>> {
    return this.request('GET', `/api/redemption/search?keyword=${encodeURIComponent(keyword)}&p=${page}&page_size=${perPage}`)
  }

  createRedemption(r: Partial<GatewayRedemption>): Promise<GatewayApiResponse<GatewayRedemption>> {
    return this.request('POST', '/api/redemption/', r)
  }

  updateRedemption(r: Partial<GatewayRedemption> & { id: number }): Promise<GatewayApiResponse<GatewayRedemption>> {
    return this.request('PUT', '/api/redemption/', r)
  }

  deleteRedemption(id: number): Promise<GatewayApiResponse<null>> {
    return this.request('DELETE', `/api/redemption/${id}`)
  }

  deleteInvalidRedemptions(): Promise<GatewayApiResponse<null>> {
    return this.request('DELETE', '/api/redemption/invalid')
  }

  // ========== Models ==========

  getModels(page = 0, perPage = 50): Promise<PaginatedResponse<GatewayModelMeta>> {
    return this.request('GET', `/api/models/?p=${page}&page_size=${perPage}`)
  }

  createModel(m: Partial<GatewayModelMeta>): Promise<GatewayApiResponse<GatewayModelMeta>> {
    return this.request('POST', '/api/models/', m)
  }

  updateModel(m: Partial<GatewayModelMeta> & { model_name: string }): Promise<GatewayApiResponse<GatewayModelMeta>> {
    return this.request('PUT', '/api/models/', m)
  }

  deleteModel(name: string): Promise<GatewayApiResponse<null>> {
    return this.request('DELETE', `/api/models/${encodeURIComponent(name)}`)
  }

  syncUpstreamPreview(): Promise<GatewayApiResponse<GatewayModelMeta[]>> {
    return this.request('GET', '/api/models/sync_upstream/preview')
  }

  syncUpstream(): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/models/sync_upstream')
  }

  getMissingModels(): Promise<GatewayApiResponse<string[]>> {
    return this.request('GET', '/api/models/missing')
  }

  // ========== Vendors ==========

  getVendors(): Promise<GatewayApiResponse<GatewayVendor[]>> {
    return this.request('GET', '/api/models/vendors/')
  }

  createVendor(v: Partial<GatewayVendor>): Promise<GatewayApiResponse<GatewayVendor>> {
    return this.request('POST', '/api/models/vendors/', v)
  }

  updateVendor(v: Partial<GatewayVendor> & { id: number }): Promise<GatewayApiResponse<GatewayVendor>> {
    return this.request('PUT', '/api/models/vendors/', v)
  }

  deleteVendor(id: number): Promise<GatewayApiResponse<null>> {
    return this.request('DELETE', `/api/models/vendors/${id}`)
  }

  // ========== Logs ==========

  getLogs(page = 0, perPage = 50, params?: {
    start_timestamp?: number
    end_timestamp?: number
    username?: string
    model?: string
    channel_id?: number
    token_name?: string
    type?: number
  }): Promise<PaginatedResponse<GatewayLog>> {
    const qs = new URLSearchParams({ p: String(page), page_size: String(perPage) })
    if (params?.start_timestamp) qs.set('start_timestamp', String(params.start_timestamp))
    if (params?.end_timestamp) qs.set('end_timestamp', String(params.end_timestamp))
    if (params?.username) qs.set('username', params.username)
    if (params?.model) qs.set('model', params.model)
    if (params?.channel_id) qs.set('channel_id', String(params.channel_id))
    if (params?.token_name) qs.set('token_name', params.token_name)
    if (params?.type !== undefined) qs.set('type', String(params.type))
    return this.request('GET', `/api/log/?${qs.toString()}`)
  }

  searchLogs(keyword: string, page = 0, perPage = 50): Promise<PaginatedResponse<GatewayLog>> {
    return this.request('GET', `/api/log/search?keyword=${encodeURIComponent(keyword)}&p=${page}&page_size=${perPage}`)
  }

  getLogStats(startTimestamp: number, endTimestamp: number): Promise<GatewayApiResponse<GatewayLogStat[]>> {
    return this.request('GET', `/api/log/stat?start_timestamp=${startTimestamp}&end_timestamp=${endTimestamp}`)
  }

  clearLogs(): Promise<GatewayApiResponse<null>> {
    return this.request('DELETE', '/api/log/')
  }

  // ========== Dashboard ==========

  getDashboardData(): Promise<GatewayApiResponse<GatewayDashboardData>> {
    return this.request('GET', '/api/data/')
  }

  getQuotaDates(startDate: string, endDate: string): Promise<GatewayApiResponse<GatewayQuotaDate[]>> {
    return this.request('GET', `/api/data/quota_dates?start_date=${startDate}&end_date=${endDate}`)
  }

  getPerformanceStats(): Promise<GatewayApiResponse<GatewayPerformanceStats>> {
    return this.request('GET', '/api/performance/stats')
  }

  // ========== Subscriptions ==========

  getSubscriptionPlans(): Promise<GatewayApiResponse<GatewaySubscriptionPlan[]>> {
    return this.request('GET', '/api/subscription/plans/')
  }

  createSubscriptionPlan(p: Partial<GatewaySubscriptionPlan>): Promise<GatewayApiResponse<GatewaySubscriptionPlan>> {
    return this.request('POST', '/api/subscription/plans/', p)
  }

  updateSubscriptionPlan(p: Partial<GatewaySubscriptionPlan> & { id: number }): Promise<GatewayApiResponse<GatewaySubscriptionPlan>> {
    return this.request('PUT', '/api/subscription/plans/', p)
  }

  deleteSubscriptionPlan(id: number): Promise<GatewayApiResponse<null>> {
    return this.request('DELETE', `/api/subscription/plans/${id}`)
  }

  bindSubscription(userId: number, planId: number): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/subscription/bind', { user_id: userId, plan_id: planId })
  }

  getUserSubscriptions(userId: number): Promise<GatewayApiResponse<GatewaySubscriptionPlan[]>> {
    return this.request('GET', `/api/subscription/user/${userId}`)
  }

  // ========== Options ==========

  getOptions(): Promise<GatewayApiResponse<Record<string, string>>> {
    return this.request('GET', '/api/option/')
  }

  updateOption(key: string, value: string): Promise<GatewayApiResponse<null>> {
    return this.request('PUT', '/api/option/', { key, value })
  }

  updateOptions(options: Record<string, string>): Promise<GatewayApiResponse<null>> {
    return this.request('PUT', '/api/option/', options)
  }

  resetModelRatio(): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/option/reset_model_ratio')
  }

  clearCache(): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/option/clear_cache')
  }

  // ========== Groups ==========

  getGroups(): Promise<GatewayApiResponse<GatewayGroup[]>> {
    return this.request('GET', '/api/group/')
  }
}

/** Creates a GatewayAPI client from the running server URL and token. */
export function createGatewayClient(baseURL: string, token: string): GatewayAPI {
  return new GatewayAPI(baseURL, token)
}
