/** Pre-configured HTTP client for the local gateway (lurus-newapi) API.
 *
 * Wire-format calibrated to QuantumNous/new-api v1.0.0-rc.4. The previous
 * iteration of this file lagged behind upstream and most paginated endpoints
 * silently returned empty pages because the response shape had changed.
 * Specifically: newapi wraps list responses as
 *
 *     { success, message, data: { items: T[], total, page, page_size } }
 *
 * and treats `?p=0` as page 1 (anything <1 floors to 1). Both are handled
 * inside the `paginated` helper so callers see the flat
 * `{ data: T[], total }` shape they expect.
 */

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
  model_name: string
  quota: number
  prompt_tokens: number
  completion_tokens: number
  channel: number
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
  id: number
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

// SubscriptionPlan keeps its legacy fields (name/level/status) so existing
// pages compile. newapi v1.0.0-rc.4 actually uses title/sort_order/enabled
// — Reseller deployments hit this through newhub, which still translates
// to the legacy shape, so consumers don't need to migrate yet. When newhub
// catches up, swap to the new names. See audit memory 2026-05-09.
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

// Legacy dashboard summary shape — newapi rc.4 has no equivalent single
// endpoint. Reseller mode aggregates upstream via newhub
// (HubGetDashboardSummary). Local mode returns zeros from the stub below.
export interface GatewayDashboardData {
  user_count: number
  channel_count: number
  token_count: number
  today_request: number
  today_quota: number
  today_tokens: number
}

// Legacy performance shape — newapi /api/performance returns a richer
// structure (cache_stats, memory_stats, …). For now we keep the old fields
// so the dashboard page typechecks; reseller mode populates via Hub.
export interface GatewayPerformanceStats {
  goroutines: number
  memory_alloc: number
  uptime: number
  requests_total: number
  requests_per_sec: number
}

export interface GatewayOption {
  key: string
  value: string
}

export interface GatewayLogStat {
  // newapi /api/log/stat returns a single aggregate, not a date series.
  quota: number
  rpm: number
  tpm: number
}

export interface GatewayQuotaDate {
  date: string
  quota: number
  request_count: number
  token_count: number
  model_usage: Record<string, number>
}

// --- Response Types ---

export interface GatewayApiResponse<T> {
  success: boolean
  message: string
  data?: T
}

/**
 * Post-unwrap paginated shape. The on-the-wire envelope is
 *   { success, message, data: { items, total, page, page_size, ... } }
 * The `paginated` helper flattens it so consumers can keep using
 * `res.data` as the row array and `res.total` for total count.
 */
export interface PaginatedResponse<T> {
  success: boolean
  message: string
  data: T[]
  total: number
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

  /**
   * Unwrap a newapi paginated GET. Accepts both the wrapped
   * `{ data: { items, total } }` shape (rc.4+) and a bare-array
   * `{ data: [] }` for forward-compat with hypothetical older endpoints.
   * Always 1-indexes pages — caller passes 0-indexed `page` and we
   * translate.
   */
  private async paginated<T>(
    basePath: string,
    page: number,
    perPage: number,
    extraQuery?: URLSearchParams,
  ): Promise<PaginatedResponse<T>> {
    const qs = new URLSearchParams({
      p: String(page + 1), // newapi treats <1 as 1; UI is 0-indexed, shift here.
      page_size: String(perPage),
    })
    if (extraQuery) {
      extraQuery.forEach((v, k) => qs.set(k, v))
    }
    const sep = basePath.includes('?') ? '&' : '?'
    const raw = await this.request<{
      success: boolean
      message: string
      data?: { items?: T[]; total?: number } | T[]
    }>('GET', `${basePath}${sep}${qs.toString()}`)
    const wrapper = raw.data
    if (wrapper && !Array.isArray(wrapper) && Array.isArray(wrapper.items)) {
      return {
        success: !!raw.success,
        message: raw.message ?? '',
        data: (wrapper.items ?? []) as T[],
        total: typeof wrapper.total === 'number' ? wrapper.total : (wrapper.items?.length ?? 0),
      }
    }
    if (Array.isArray(wrapper)) {
      return { success: !!raw.success, message: raw.message ?? '', data: wrapper as T[], total: wrapper.length }
    }
    return { success: !!raw.success, message: raw.message ?? '', data: [], total: 0 }
  }

  // ========== Channels ==========

  getChannels(page = 0, perPage = 50): Promise<PaginatedResponse<GatewayChannel>> {
    return this.paginated('/api/channel/', page, perPage)
  }

  getChannel(id: number): Promise<GatewayApiResponse<GatewayChannel>> {
    return this.request('GET', `/api/channel/${id}`)
  }

  searchChannels(keyword: string, page = 0, perPage = 50): Promise<PaginatedResponse<GatewayChannel>> {
    const qs = new URLSearchParams({ keyword })
    return this.paginated('/api/channel/search', page, perPage, qs)
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

  // newapi expects PUT /api/channel/tag with body { tag, new_tag } — NOT
  // POST /tag/edit nor body { old_tag, new_tag } as a previous version did.
  editChannelTag(oldTag: string, newTag: string): Promise<GatewayApiResponse<null>> {
    return this.request('PUT', '/api/channel/tag', { tag: oldTag, new_tag: newTag })
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

  // newapi route is POST /api/channel/fix (not /fix_abilities).
  fixChannelAbilities(): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/channel/fix')
  }

  // ========== Tokens ==========

  getTokens(page = 0, perPage = 50): Promise<PaginatedResponse<GatewayToken>> {
    return this.paginated('/api/token/', page, perPage)
  }

  getToken(id: number): Promise<GatewayApiResponse<GatewayToken>> {
    return this.request('GET', `/api/token/${id}`)
  }

  searchTokens(keyword: string, page = 0, perPage = 50): Promise<PaginatedResponse<GatewayToken>> {
    const qs = new URLSearchParams({ keyword })
    return this.paginated('/api/token/search', page, perPage, qs)
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

  // The token list endpoint is UserAuth-scoped — it returns only the
  // current user's tokens. Reveal-key requires hitting the rate-limited
  // /:id/key route since the list response always returns a masked key.
  revealTokenKey(id: number): Promise<GatewayApiResponse<{ key: string }>> {
    return this.request('POST', `/api/token/${id}/key`)
  }

  // ========== Users ==========

  getUsers(page = 0, perPage = 50): Promise<PaginatedResponse<GatewayUser>> {
    return this.paginated('/api/user/', page, perPage)
  }

  getUser(id: number): Promise<GatewayApiResponse<GatewayUser>> {
    return this.request('GET', `/api/user/${id}`)
  }

  searchUsers(keyword: string, page = 0, perPage = 50): Promise<PaginatedResponse<GatewayUser>> {
    const qs = new URLSearchParams({ keyword })
    return this.paginated('/api/user/search', page, perPage, qs)
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
    return this.paginated('/api/redemption/', page, perPage)
  }

  getRedemption(id: number): Promise<GatewayApiResponse<GatewayRedemption>> {
    return this.request('GET', `/api/redemption/${id}`)
  }

  searchRedemptions(keyword: string, page = 0, perPage = 50): Promise<PaginatedResponse<GatewayRedemption>> {
    const qs = new URLSearchParams({ keyword })
    return this.paginated('/api/redemption/search', page, perPage, qs)
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
    return this.paginated('/api/models/', page, perPage)
  }

  createModel(m: Partial<GatewayModelMeta>): Promise<GatewayApiResponse<GatewayModelMeta>> {
    return this.request('POST', '/api/models/', m)
  }

  updateModel(m: Partial<GatewayModelMeta> & { id: number }): Promise<GatewayApiResponse<GatewayModelMeta>> {
    return this.request('PUT', '/api/models/', m)
  }

  // newapi expects an integer ID at /api/models/:id (parsed via strconv.Atoi).
  deleteModel(id: number): Promise<GatewayApiResponse<null>> {
    return this.request('DELETE', `/api/models/${id}`)
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
  // newapi exposes vendors at /api/vendors/, NOT /api/models/vendors/.

  getVendors(): Promise<GatewayApiResponse<GatewayVendor[]>> {
    return this.request('GET', '/api/vendors/')
  }

  createVendor(v: Partial<GatewayVendor>): Promise<GatewayApiResponse<GatewayVendor>> {
    return this.request('POST', '/api/vendors/', v)
  }

  updateVendor(v: Partial<GatewayVendor> & { id: number }): Promise<GatewayApiResponse<GatewayVendor>> {
    return this.request('PUT', '/api/vendors/', v)
  }

  deleteVendor(id: number): Promise<GatewayApiResponse<null>> {
    return this.request('DELETE', `/api/vendors/${id}`)
  }

  // ========== Logs ==========

  getLogs(page = 0, perPage = 50, params?: {
    start_timestamp?: number
    end_timestamp?: number
    username?: string
    model_name?: string
    channel?: number
    token_name?: string
    type?: number
  }): Promise<PaginatedResponse<GatewayLog>> {
    const qs = new URLSearchParams()
    if (params?.start_timestamp) qs.set('start_timestamp', String(params.start_timestamp))
    if (params?.end_timestamp) qs.set('end_timestamp', String(params.end_timestamp))
    if (params?.username) qs.set('username', params.username)
    // newapi reads `model_name` and `channel` (not `model` / `channel_id`).
    if (params?.model_name) qs.set('model_name', params.model_name)
    if (params?.channel) qs.set('channel', String(params.channel))
    if (params?.token_name) qs.set('token_name', params.token_name)
    if (params?.type !== undefined) qs.set('type', String(params.type))
    return this.paginated('/api/log/', page, perPage, qs)
  }

  // /api/log/search has been deprecated upstream — the controller returns
  // a static "该接口已废弃" envelope. We expose the call so callers can detect
  // and swap to getLogs() with filters, but this method always rejects with
  // a clear message rather than silently returning an empty page.
  searchLogs(_keyword: string, _page = 0, _perPage = 50): Promise<PaginatedResponse<GatewayLog>> {
    return Promise.reject(new Error('searchLogs: /api/log/search is deprecated upstream — use getLogs with filter params instead'))
  }

  // newapi /api/log/stat returns a single aggregate object, not a date series.
  getLogStats(startTimestamp: number, endTimestamp: number): Promise<GatewayApiResponse<GatewayLogStat>> {
    return this.request('GET', `/api/log/stat?start_timestamp=${startTimestamp}&end_timestamp=${endTimestamp}`)
  }

  clearLogs(): Promise<GatewayApiResponse<null>> {
    return this.request('DELETE', '/api/log/')
  }

  // ========== Dashboard ==========

  /**
   * Legacy summary endpoint. newapi v1.0.0-rc.4 does NOT expose a single
   * counters endpoint — `/api/data/` returns quota-date rows, not summary
   * cards. Reseller deployments aggregate via newhub. We return zeros so
   * Local-mode dashboard renders without crashing; real values require a
   * client-side aggregation pass over /api/user/?p=1&page_size=1 etc.
   */
  async getDashboardData(): Promise<GatewayApiResponse<GatewayDashboardData>> {
    return {
      success: false,
      message: 'newapi rc.4 has no /api/data/ summary endpoint — use Hub mode',
      data: {
        user_count: 0, channel_count: 0, token_count: 0,
        today_request: 0, today_quota: 0, today_tokens: 0,
      },
    }
  }

  /**
   * Quota-date series. Accepts either ISO date strings (legacy callers) or
   * Unix timestamps in seconds (correct rc.4 contract). Strings get parsed
   * via Date.UTC and converted before the request goes out.
   */
  getQuotaDates(start: string | number, end: string | number): Promise<GatewayApiResponse<GatewayQuotaDate[]>> {
    const toTs = (v: string | number): number => typeof v === 'number'
      ? v
      : Math.floor(Date.parse(v) / 1000) || 0
    const startTs = toTs(start)
    const endTs = toTs(end)
    return this.request('GET', `/api/data/?start_timestamp=${startTs}&end_timestamp=${endTs}`)
  }

  /**
   * newapi /api/performance returns
   *   { cache_stats, memory_stats, disk_cache_info, disk_space_info, config }
   * which doesn't match the legacy GatewayPerformanceStats. We attempt a
   * best-effort flatten of memory_stats fields so the legacy dashboard
   * widget at least shows non-zero memory; everything else falls back to 0.
   */
  async getPerformanceStats(): Promise<GatewayApiResponse<GatewayPerformanceStats>> {
    const raw = await this.request<{ success: boolean; message: string; data?: any }>('GET', '/api/performance/')
    const mem = raw.data?.memory_stats ?? {}
    return {
      success: !!raw.success,
      message: raw.message ?? '',
      data: {
        goroutines: typeof mem.num_goroutine === 'number' ? mem.num_goroutine : 0,
        memory_alloc: typeof mem.alloc === 'number' ? mem.alloc : 0,
        uptime: 0,
        requests_total: 0,
        requests_per_sec: 0,
      },
    }
  }

  // ========== Subscriptions ==========
  //
  // newapi v1.0.0-rc.4 actually serves admin CRUD under
  // /api/subscription/admin/* with body wrapped as { plan: ... }, and the
  // SubscriptionPlan model uses title/sort_order/enabled instead of
  // name/level/status. The page below is too tangled with the legacy shape
  // to migrate piecemeal — it's a Reseller-only feature reached through
  // newhub (which still translates) so the legacy paths keep working.
  // When the page gets rewritten for rc.4 we'll move these to /admin/.
  // See audit memory project_admin_audit_202605.md for the punch list.

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
  // newapi /api/option/ returns an array of { key, value } pairs. We keep
  // the consumer-facing shape as Record<string, string> and flatten here
  // so callers don't have to.

  async getOptions(): Promise<GatewayApiResponse<Record<string, string>>> {
    const raw = await this.request<{ success: boolean; message: string; data?: GatewayOption[] }>('GET', '/api/option/')
    const map: Record<string, string> = {}
    for (const entry of raw.data ?? []) {
      if (entry && typeof entry.key === 'string') {
        map[entry.key] = String(entry.value ?? '')
      }
    }
    return { success: !!raw.success, message: raw.message ?? '', data: map }
  }

  // newapi only accepts one option per request; for bulk updates the caller
  // must call this in a loop.
  updateOption(key: string, value: string): Promise<GatewayApiResponse<null>> {
    return this.request('PUT', '/api/option/', { key, value })
  }

  /** Convenience wrapper — sequential per-key PUTs. Returns on first failure. */
  async updateOptions(options: Record<string, string>): Promise<GatewayApiResponse<null>> {
    let last: GatewayApiResponse<null> = { success: true, message: '' }
    for (const [k, v] of Object.entries(options)) {
      last = await this.updateOption(k, v)
      if (!last.success) return last
    }
    return last
  }

  // newapi has the typo `rest_model_ratio` — yes really. Keeping the call
  // hidden behind a sane name; see api-router.go:186.
  resetModelRatio(): Promise<GatewayApiResponse<null>> {
    return this.request('POST', '/api/option/rest_model_ratio')
  }

  /**
   * newapi has no generic /clear_cache endpoint — closest equivalent is
   * the channel-affinity cache, which is what the UI's "clear cache" button
   * actually means in practice. `clearCache` retained as alias.
   */
  clearChannelAffinityCache(): Promise<GatewayApiResponse<null>> {
    return this.request('DELETE', '/api/option/channel_affinity_cache')
  }
  clearCache(): Promise<GatewayApiResponse<null>> {
    return this.clearChannelAffinityCache()
  }

  // ========== Groups ==========
  // newapi /api/group/ returns a flat string array of group names.

  getGroups(): Promise<GatewayApiResponse<string[]>> {
    return this.request('GET', '/api/group/')
  }
}

/** Creates a GatewayAPI client from the running server URL and token. */
export function createGatewayClient(baseURL: string, token: string): GatewayAPI {
  return new GatewayAPI(baseURL, token)
}
