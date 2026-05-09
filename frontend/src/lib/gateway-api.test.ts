import { describe, it, expect, beforeEach, vi } from 'vitest'
import { GatewayAPI, CHANNEL_TYPES } from './gateway-api'

const BASE_URL = 'http://localhost:19090'
const TOKEN = 'test-admin-token'

describe('GatewayAPI', () => {
  let api: GatewayAPI
  let fetchSpy: ReturnType<typeof vi.fn>

  beforeEach(() => {
    api = new GatewayAPI(BASE_URL, TOKEN)
    fetchSpy = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ success: true, data: { items: [], total: 0 } }),
      text: () => Promise.resolve(''),
    })
    vi.stubGlobal('fetch', fetchSpy)
  })

  // === Constructor ===
  describe('constructor', () => {
    it('should strip trailing slash from baseURL', () => {
      const api2 = new GatewayAPI('http://localhost:19090/', TOKEN)
      api2.getChannels()
      expect(fetchSpy).toHaveBeenCalledWith(
        expect.stringMatching(/^http:\/\/localhost:19090\/api\//),
        expect.any(Object)
      )
    })
  })

  // === Request plumbing ===
  describe('request', () => {
    it('should include Authorization header', async () => {
      await api.getChannels()
      const [, opts] = fetchSpy.mock.calls[0]
      expect(opts.headers.Authorization).toBe(`Bearer ${TOKEN}`)
    })

    it('should include Content-Type header', async () => {
      await api.getChannels()
      const [, opts] = fetchSpy.mock.calls[0]
      expect(opts.headers['Content-Type']).toBe('application/json')
    })

    it('should throw on non-OK response', async () => {
      fetchSpy.mockResolvedValueOnce({
        ok: false,
        status: 403,
        statusText: 'Forbidden',
        text: () => Promise.resolve('access denied'),
      })
      await expect(api.getChannels()).rejects.toThrow('403')
    })

    it('should include status code and path in error message', async () => {
      fetchSpy.mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        text: () => Promise.resolve('not found'),
      })
      await expect(api.getChannels()).rejects.toThrow('/api/channel/')
    })
  })

  // === Pagination semantics ===
  describe('pagination', () => {
    it('shifts 0-indexed page to 1-indexed (newapi treats <1 as page 1)', async () => {
      await api.getChannels(0, 25)
      const url = fetchSpy.mock.calls[0][0] as string
      expect(url).toMatch(/p=1/)
      expect(url).toMatch(/page_size=25/)
    })

    it('shifts page=2 to p=3', async () => {
      await api.getChannels(2, 25)
      const url = fetchSpy.mock.calls[0][0] as string
      expect(url).toMatch(/p=3/)
    })

    it('unwraps {data: {items, total}} into flat {data: T[], total}', async () => {
      fetchSpy.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({
          success: true,
          message: '',
          data: { items: [{ id: 1 }, { id: 2 }], total: 99, page: 1, page_size: 50 },
        }),
        text: () => Promise.resolve(''),
      })
      const res = await api.getChannels()
      expect(res.data.length).toBe(2)
      expect(res.total).toBe(99)
    })

    it('falls back to bare-array data for forward-compat', async () => {
      fetchSpy.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ success: true, data: [{ id: 1 }] }),
        text: () => Promise.resolve(''),
      })
      const res = await api.getChannels()
      expect(res.data.length).toBe(1)
      expect(res.total).toBe(1)
    })
  })

  // === Channels ===
  describe('channels', () => {
    it('getChannel should GET by id (no pagination shift)', async () => {
      await api.getChannel(42)
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/channel/42`)
    })

    it('searchChannels should encode keyword + still 1-index', async () => {
      await api.searchChannels('test query', 0, 50)
      const url = fetchSpy.mock.calls[0][0] as string
      expect(url).toContain('keyword=test+query')
      expect(url).toMatch(/p=1/)
    })

    it('createChannel should POST', async () => {
      await api.createChannel({ name: 'ch1', type: 1 })
      const [url, opts] = fetchSpy.mock.calls[0]
      expect(url).toBe(`${BASE_URL}/api/channel/`)
      expect(opts.method).toBe('POST')
      expect(JSON.parse(opts.body)).toEqual({ name: 'ch1', type: 1 })
    })

    it('updateChannel should PUT', async () => {
      await api.updateChannel({ id: 1, name: 'updated' })
      const [, opts] = fetchSpy.mock.calls[0]
      expect(opts.method).toBe('PUT')
    })

    it('deleteChannel should DELETE by id', async () => {
      await api.deleteChannel(5)
      const [url, opts] = fetchSpy.mock.calls[0]
      expect(url).toBe(`${BASE_URL}/api/channel/5`)
      expect(opts.method).toBe('DELETE')
    })

    it('batchDeleteChannels should POST ids', async () => {
      await api.batchDeleteChannels([1, 2, 3])
      const body = JSON.parse(fetchSpy.mock.calls[0][1].body)
      expect(body.ids).toEqual([1, 2, 3])
      expect(body.action).toBe('delete')
    })

    it('testChannel should GET test endpoint', async () => {
      await api.testChannel(7)
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/channel/test/7`)
    })

    it('copyChannel should POST', async () => {
      await api.copyChannel(3)
      const [url, opts] = fetchSpy.mock.calls[0]
      expect(url).toBe(`${BASE_URL}/api/channel/copy/3`)
      expect(opts.method).toBe('POST')
    })

    it('fetchChannelModels should GET', async () => {
      await api.fetchChannelModels(5)
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/channel/fetch_models/5`)
    })

    it('fixChannelAbilities should POST /api/channel/fix (not /fix_abilities)', async () => {
      await api.fixChannelAbilities()
      const [url, opts] = fetchSpy.mock.calls[0]
      expect(url).toBe(`${BASE_URL}/api/channel/fix`)
      expect(opts.method).toBe('POST')
    })

    it('editChannelTag should PUT /api/channel/tag with {tag, new_tag}', async () => {
      await api.editChannelTag('old', 'new')
      const [url, opts] = fetchSpy.mock.calls[0]
      expect(url).toBe(`${BASE_URL}/api/channel/tag`)
      expect(opts.method).toBe('PUT')
      const body = JSON.parse(opts.body)
      expect(body.tag).toBe('old')
      expect(body.new_tag).toBe('new')
    })
  })

  // === Tokens ===
  describe('tokens', () => {
    it('getTokens should use GET (1-indexed)', async () => {
      await api.getTokens(0, 50)
      expect(fetchSpy.mock.calls[0][0]).toMatch(/\/api\/token\/\?p=1&page_size=50/)
    })

    it('createToken should POST', async () => {
      await api.createToken({ name: 'tk1' })
      expect(fetchSpy.mock.calls[0][1].method).toBe('POST')
    })

    it('updateToken should PUT', async () => {
      await api.updateToken({ id: 1, name: 'updated' })
      expect(fetchSpy.mock.calls[0][1].method).toBe('PUT')
    })

    it('deleteToken should DELETE', async () => {
      await api.deleteToken(3)
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/token/3`)
      expect(fetchSpy.mock.calls[0][1].method).toBe('DELETE')
    })

    it('revealTokenKey should POST to rate-limited reveal route', async () => {
      await api.revealTokenKey(7)
      const [url, opts] = fetchSpy.mock.calls[0]
      expect(url).toBe(`${BASE_URL}/api/token/7/key`)
      expect(opts.method).toBe('POST')
    })
  })

  // === Users ===
  describe('users', () => {
    it('getUsers should use GET', async () => {
      await api.getUsers()
      expect(fetchSpy.mock.calls[0][0]).toContain('/api/user/')
    })

    it('createUser should POST with password', async () => {
      await api.createUser({ username: 'new', password: 'pass' })
      const body = JSON.parse(fetchSpy.mock.calls[0][1].body)
      expect(body.username).toBe('new')
      expect(body.password).toBe('pass')
    })

    it('deleteUser should DELETE by id', async () => {
      await api.deleteUser(5)
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/user/5`)
    })

    it('manageUser should POST action', async () => {
      await api.manageUser({ id: 1, action: 'disable' })
      const body = JSON.parse(fetchSpy.mock.calls[0][1].body)
      expect(body.action).toBe('disable')
    })
  })

  // === Redemptions ===
  describe('redemptions', () => {
    it('getRedemptions should use GET', async () => {
      await api.getRedemptions()
      expect(fetchSpy.mock.calls[0][0]).toContain('/api/redemption/')
    })

    it('createRedemption should POST', async () => {
      await api.createRedemption({ name: 'r1', quota: 100, count: 10 })
      expect(fetchSpy.mock.calls[0][1].method).toBe('POST')
    })

    it('deleteInvalidRedemptions should DELETE', async () => {
      await api.deleteInvalidRedemptions()
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/redemption/invalid`)
      expect(fetchSpy.mock.calls[0][1].method).toBe('DELETE')
    })
  })

  // === Models ===
  describe('models', () => {
    it('getModels should use GET', async () => {
      await api.getModels()
      expect(fetchSpy.mock.calls[0][0]).toContain('/api/models/')
    })

    it('syncUpstream should POST', async () => {
      await api.syncUpstream()
      expect(fetchSpy.mock.calls[0][1].method).toBe('POST')
      expect(fetchSpy.mock.calls[0][0]).toContain('sync_upstream')
    })

    it('deleteModel should DELETE by integer id (not name)', async () => {
      await api.deleteModel(42)
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/models/42`)
      expect(fetchSpy.mock.calls[0][1].method).toBe('DELETE')
    })
  })

  // === Vendors ===
  describe('vendors', () => {
    it('getVendors should hit /api/vendors/ (NOT /api/models/vendors/)', async () => {
      await api.getVendors()
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/vendors/`)
    })

    it('deleteVendor should DELETE under /api/vendors/', async () => {
      await api.deleteVendor(3)
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/vendors/3`)
    })
  })

  // === Logs ===
  describe('logs', () => {
    it('getLogs should rename model→model_name and channel→channel', async () => {
      await api.getLogs(0, 50, { username: 'root', model_name: 'gpt-4', channel: 7 })
      const url = fetchSpy.mock.calls[0][0] as string
      expect(url).toContain('username=root')
      expect(url).toContain('model_name=gpt-4')
      expect(url).toContain('channel=7')
      // Old-style filter names must not appear.
      expect(url).not.toMatch(/[?&]model=/)
      expect(url).not.toMatch(/channel_id=/)
    })

    it('clearLogs should DELETE', async () => {
      await api.clearLogs()
      expect(fetchSpy.mock.calls[0][1].method).toBe('DELETE')
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/log/`)
    })

    it('searchLogs rejects (deprecated upstream)', async () => {
      await expect(api.searchLogs('q')).rejects.toThrow(/deprecated/i)
    })
  })

  // === Dashboard ===
  describe('dashboard', () => {
    it('getQuotaDates should pass timestamp range', async () => {
      await api.getQuotaDates(1700000000, 1700100000)
      const url = fetchSpy.mock.calls[0][0] as string
      expect(url).toContain('start_timestamp=1700000000')
      expect(url).toContain('end_timestamp=1700100000')
    })

    it('getPerformanceStats should GET /api/performance/', async () => {
      await api.getPerformanceStats()
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/performance/`)
    })
  })

  // === Options ===
  describe('options', () => {
    it('getOptions should flatten [{key,value}] into a map', async () => {
      fetchSpy.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({
          success: true,
          data: [
            { key: 'Theme', value: 'dark' },
            { key: 'SiteName', value: 'Lurus' },
          ],
        }),
        text: () => Promise.resolve(''),
      })
      const res = await api.getOptions()
      expect(res.data).toEqual({ Theme: 'dark', SiteName: 'Lurus' })
    })

    it('updateOption should PUT with {key, value}', async () => {
      await api.updateOption('Theme', 'dark')
      const body = JSON.parse(fetchSpy.mock.calls[0][1].body)
      expect(body.key).toBe('Theme')
      expect(body.value).toBe('dark')
    })

    it('resetModelRatio uses upstream typo path /rest_model_ratio', async () => {
      await api.resetModelRatio()
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/option/rest_model_ratio`)
    })

    it('clearChannelAffinityCache should DELETE the affinity cache', async () => {
      await api.clearChannelAffinityCache()
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/option/channel_affinity_cache`)
      expect(fetchSpy.mock.calls[0][1].method).toBe('DELETE')
    })
  })

  // === Subscriptions ===
  // Subscription paths kept on legacy /api/subscription/* until the page is
  // rewritten for rc.4. See audit memory for the upstream namespace move
  // (admin/* + {plan} body wrapper + PATCH-disable instead of DELETE).
  describe('subscriptions', () => {
    it('getSubscriptionPlans should GET', async () => {
      await api.getSubscriptionPlans()
      expect(fetchSpy.mock.calls[0][0]).toContain('/api/subscription/plans/')
    })

    it('bindSubscription should POST', async () => {
      await api.bindSubscription(1, 2)
      const body = JSON.parse(fetchSpy.mock.calls[0][1].body)
      expect(body.user_id).toBe(1)
      expect(body.plan_id).toBe(2)
    })
  })

  // === Groups ===
  describe('groups', () => {
    it('getGroups should GET (returns string[])', async () => {
      await api.getGroups()
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/group/`)
    })
  })

  // === CHANNEL_TYPES constant ===
  describe('CHANNEL_TYPES', () => {
    it('should include OpenAI as type 1', () => {
      expect(CHANNEL_TYPES[1]).toBe('OpenAI')
    })

    it('should include Anthropic as type 14', () => {
      expect(CHANNEL_TYPES[14]).toBe('Anthropic')
    })

    it('should have more than 30 types', () => {
      expect(Object.keys(CHANNEL_TYPES).length).toBeGreaterThan(30)
    })
  })
})
