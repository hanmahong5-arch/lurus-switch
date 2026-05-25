import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallbackOrOpts?: string | Record<string, unknown>, opts?: Record<string, unknown>) => {
      const fallback = typeof fallbackOrOpts === 'string' ? fallbackOrOpts : key
      const vars = typeof fallbackOrOpts === 'object' ? fallbackOrOpts : opts
      if (vars && typeof vars === 'object') {
        return Object.entries(vars).reduce<string>(
          (s, [k, v]) => s.replace(new RegExp(`{{\\s*${k}\\s*}}`, 'g'), String(v)),
          fallback,
        )
      }
      return fallback
    },
    i18n: { language: 'zh' },
  }),
}))

vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn(() => () => undefined),
}))

const lastResultsMock = vi.fn()
const runMock = vi.fn()
vi.mock('../../wailsjs/go/main/App', () => ({
  GetLastModelAuthResults: () => lastResultsMock(),
  RunModelAuthCheck: (b: boolean) => runMock(b),
}))

import { ModelAuthenticityPanel } from './ModelAuthenticityPanel'

describe('ModelAuthenticityPanel', () => {
  afterEach(() => {
    cleanup()
    lastResultsMock.mockReset()
    runMock.mockReset()
  })

  it('renders cached results with verdict badges', async () => {
    lastResultsMock.mockResolvedValue([
      {
        providerId: 'a', providerName: 'Alpha',
        requestedModel: 'claude-sonnet-4-6', reportedModel: 'claude-sonnet-4-6',
        verdict: 'match', latencyMs: 50, testedAt: '2026-05-25T10:00:00Z',
      },
      {
        providerId: 'b', providerName: 'FakeSeller',
        requestedModel: 'claude-opus-4-7', reportedModel: 'claude-haiku-4-5',
        verdict: 'mismatch', latencyMs: 80, testedAt: '2026-05-25T10:00:01Z',
      },
    ])

    render(<ModelAuthenticityPanel includeCustom={false} />)

    await waitFor(() => {
      expect(screen.getByText('Alpha')).toBeTruthy()
      expect(screen.getByText('FakeSeller')).toBeTruthy()
      // Verdict labels surface in Chinese mode.
      expect(screen.getByText('一致')).toBeTruthy()
      expect(screen.getByText('不一致')).toBeTruthy()
      // Mismatch count chip.
      expect(screen.getByText(/1 不一致/)).toBeTruthy()
    })
  })

  it('requires confirm before triggering a real probe (cost warning)', async () => {
    lastResultsMock.mockResolvedValue([])
    runMock.mockResolvedValue(undefined)
    render(<ModelAuthenticityPanel />)

    fireEvent.click(screen.getByText('检测真伪'))
    // Run button shouldn't have fired yet — only the confirm button appears.
    expect(runMock).not.toHaveBeenCalled()
    fireEvent.click(screen.getByText('确认烧 token'))
    await waitFor(() => {
      expect(runMock).toHaveBeenCalledTimes(1)
    })
  })

  it('renders the disclaimer about declaration-layer-only detection', async () => {
    lastResultsMock.mockResolvedValue([])
    render(<ModelAuthenticityPanel />)
    await waitFor(() => {
      expect(screen.getByText(/无法识别更深层的模型指纹冒充/)).toBeTruthy()
    })
  })
})
