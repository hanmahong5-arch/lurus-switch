import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { useGatewayStore } from './gatewayStore'

describe('gatewayStore', () => {
  beforeEach(() => {
    useGatewayStore.setState({
      status: null,
      adminToken: null,
      pollingHandle: null,
    })
  })

  afterEach(() => {
    // Clean up any running intervals
    const handle = useGatewayStore.getState().pollingHandle
    if (handle !== null) clearInterval(handle)
  })

  describe('initial state', () => {
    it('should have null status', () => {
      expect(useGatewayStore.getState().status).toBeNull()
    })

    it('should have null adminToken', () => {
      expect(useGatewayStore.getState().adminToken).toBeNull()
    })

    it('should have null pollingHandle', () => {
      expect(useGatewayStore.getState().pollingHandle).toBeNull()
    })
  })

  describe('setStatus', () => {
    it('should update status', () => {
      const status = {
        running: true,
        port: 19090,
        url: 'http://localhost:19090',
        uptime: 120,
        version: '1.0',
        binaryOk: true,
      }
      useGatewayStore.getState().setStatus(status)
      expect(useGatewayStore.getState().status).toEqual(status)
    })
  })

  describe('setAdminToken', () => {
    it('should update admin token', () => {
      useGatewayStore.getState().setAdminToken('my-token-123')
      expect(useGatewayStore.getState().adminToken).toBe('my-token-123')
    })
  })

  describe('startPolling', () => {
    it('should call fetchStatus immediately', async () => {
      const fetchStatus = vi.fn().mockResolvedValue({
        running: false, port: 19090, url: '', uptime: 0, version: '', binaryOk: true,
      })
      const fetchToken = vi.fn().mockResolvedValue('')

      useGatewayStore.getState().startPolling(fetchStatus, fetchToken)

      // fetchStatus is called async; wait a tick
      await new Promise((r) => setTimeout(r, 10))
      expect(fetchStatus).toHaveBeenCalledTimes(1)
    })

    it('should replace existing polling', () => {
      const fetchStatus = vi.fn().mockResolvedValue({
        running: false, port: 19090, url: '', uptime: 0, version: '', binaryOk: true,
      })
      const fetchToken = vi.fn().mockResolvedValue('')

      useGatewayStore.getState().startPolling(fetchStatus, fetchToken)
      const handle1 = useGatewayStore.getState().pollingHandle

      useGatewayStore.getState().startPolling(fetchStatus, fetchToken)
      const handle2 = useGatewayStore.getState().pollingHandle

      expect(handle2).not.toBe(handle1)
    })
  })

  describe('stopPolling', () => {
    it('should clear polling handle', () => {
      const fetchStatus = vi.fn().mockResolvedValue({
        running: false, port: 19090, url: '', uptime: 0, version: '', binaryOk: true,
      })
      const fetchToken = vi.fn().mockResolvedValue('')

      useGatewayStore.getState().startPolling(fetchStatus, fetchToken)
      expect(useGatewayStore.getState().pollingHandle).not.toBeNull()

      useGatewayStore.getState().stopPolling()
      expect(useGatewayStore.getState().pollingHandle).toBeNull()
    })
  })
})
