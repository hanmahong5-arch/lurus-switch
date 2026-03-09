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
      json: () => Promise.resolve({ success: true, data: {} }),
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

  // === Channels ===
  describe('channels', () => {
    it('getChannels should use GET with page params', async () => {
      await api.getChannels(2, 25)
      const [url, opts] = fetchSpy.mock.calls[0]
      expect(url).toBe(`${BASE_URL}/api/channel/?p=2&page_size=25`)
      expect(opts.method).toBe('GET')
    })

    it('getChannel should GET by id', async () => {
      await api.getChannel(42)
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/channel/42`)
    })

    it('searchChannels should encode keyword', async () => {
      await api.searchChannels('test query')
      expect(fetchSpy.mock.calls[0][0]).toContain('keyword=test%20query')
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

    it('fixChannelAbilities should POST', async () => {
      await api.fixChannelAbilities()
      const [url, opts] = fetchSpy.mock.calls[0]
      expect(url).toBe(`${BASE_URL}/api/channel/fix_abilities`)
      expect(opts.method).toBe('POST')
    })
  })

  // === Tokens ===
  describe('tokens', () => {
    it('getTokens should use GET', async () => {
      await api.getTokens(0, 50)
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/token/?p=0&page_size=50`)
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

    it('deleteModel should encode model name', async () => {
      await api.deleteModel('gpt-4o')
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/models/gpt-4o`)
    })
  })

  // === Logs ===
  describe('logs', () => {
    it('getLogs should append filter params', async () => {
      await api.getLogs(0, 50, { username: 'root', model: 'gpt-4' })
      const url = fetchSpy.mock.calls[0][0] as string
      expect(url).toContain('username=root')
      expect(url).toContain('model=gpt-4')
    })

    it('clearLogs should DELETE', async () => {
      await api.clearLogs()
      expect(fetchSpy.mock.calls[0][1].method).toBe('DELETE')
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/log/`)
    })
  })

  // === Dashboard ===
  describe('dashboard', () => {
    it('getDashboardData should GET /api/data/', async () => {
      await api.getDashboardData()
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/data/`)
    })

    it('getPerformanceStats should GET', async () => {
      await api.getPerformanceStats()
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/performance/stats`)
    })
  })

  // === Options ===
  describe('options', () => {
    it('getOptions should GET', async () => {
      await api.getOptions()
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/option/`)
    })

    it('updateOption should PUT with key/value', async () => {
      await api.updateOption('Theme', 'dark')
      const body = JSON.parse(fetchSpy.mock.calls[0][1].body)
      expect(body.key).toBe('Theme')
      expect(body.value).toBe('dark')
    })

    it('clearCache should POST', async () => {
      await api.clearCache()
      expect(fetchSpy.mock.calls[0][0]).toBe(`${BASE_URL}/api/option/clear_cache`)
      expect(fetchSpy.mock.calls[0][1].method).toBe('POST')
    })
  })

  // === Subscriptions ===
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
    it('getGroups should GET', async () => {
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
