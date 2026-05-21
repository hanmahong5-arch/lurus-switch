import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallbackOrOpts?: string | Record<string, unknown>, opts?: Record<string, unknown>) => {
      const o = (typeof fallbackOrOpts === 'object' ? fallbackOrOpts : opts) ?? {}
      let s = typeof fallbackOrOpts === 'string' ? fallbackOrOpts : key
      for (const [k, v] of Object.entries(o)) s = s.replace(`{{${k}}}`, String(v))
      return s
    },
    i18n: { language: 'zh' },
  }),
}))

// Capture EventsOn handlers so the test can emit synthetic stream events.
const handlers: Record<string, (raw: unknown) => void> = {}
vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: (name: string, cb: (raw: unknown) => void) => {
    handlers[name] = cb
    return () => { delete handlers[name] }
  },
}))

const RunModelHealthCheck = vi.fn()
const GetLastHealthCheckResults = vi.fn()
vi.mock('../../wailsjs/go/main/App', () => ({
  RunModelHealthCheck: (...a: unknown[]) => RunModelHealthCheck(...a),
  GetLastHealthCheckResults: (...a: unknown[]) => GetLastHealthCheckResults(...a),
}))

import { ModelHealthMatrix } from './ModelHealthMatrix'

beforeEach(() => {
  RunModelHealthCheck.mockReset().mockResolvedValue(undefined)
  GetLastHealthCheckResults.mockReset().mockResolvedValue([])
  for (const k of Object.keys(handlers)) delete handlers[k]
})

describe('ModelHealthMatrix', () => {
  it('renders empty state initially', async () => {
    render(<ModelHealthMatrix />)
    await waitFor(() => expect(screen.getByText(/尚未检测/)).toBeTruthy())
  })

  it('streams progress events into rows', async () => {
    render(<ModelHealthMatrix />)
    await waitFor(() => expect(handlers['model:test:progress']).toBeTruthy())

    fireEvent.click(screen.getByText('检测全部供应商'))
    expect(RunModelHealthCheck).toHaveBeenCalledWith(true)

    act(() => {
      handlers['model:test:progress']({
        providerId: 'p1', providerName: 'Alpha', status: 'ok', latencyMs: 30,
        models: ['m1', 'm2'], testedAt: '2026-05-21T10:00:00Z',
      })
      handlers['model:test:progress']({
        providerId: 'p2', providerName: 'Beta', status: 'timeout', latencyMs: 5000,
        models: [], error: 'deadline exceeded', testedAt: '2026-05-21T10:00:00Z',
      })
    })

    await waitFor(() => {
      expect(screen.getByText('Alpha')).toBeTruthy()
      expect(screen.getByText('Beta')).toBeTruthy()
      expect(screen.getByText('2 个模型')).toBeTruthy()
    })
  })

  it('passes includeCustom=false through to the binding', async () => {
    render(<ModelHealthMatrix includeCustom={false} />)
    await waitFor(() => expect(handlers['model:test:progress']).toBeTruthy())
    fireEvent.click(screen.getByText('检测全部供应商'))
    expect(RunModelHealthCheck).toHaveBeenCalledWith(false)
  })
})
