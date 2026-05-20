import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

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
    i18n: { language: 'zh', changeLanguage: vi.fn() },
  }),
  Trans: ({ children, i18nKey }: { children?: React.ReactNode; i18nKey?: string }) =>
    children ?? i18nKey ?? null,
  initReactI18next: { type: '3rdParty', init: () => {} },
}))

const getModelsMock = vi.fn()
const getAppsMock = vi.fn()
vi.mock('../../../wailsjs/go/main/App', () => ({
  GetModelSummaries: (w: string) => getModelsMock(w),
  GetAppSummaries: (w: string) => getAppsMock(w),
}))

import { UsageBreakdown, buildSlices } from './UsageBreakdown'

beforeEach(() => {
  getModelsMock.mockReset()
  getAppsMock.mockReset()
})

describe('buildSlices', () => {
  it('returns empty when input is null', () => {
    const { slices, total } = buildSlices(null, () => ({ key: 'k', display: 'd' }))
    expect(slices).toEqual([])
    expect(total).toBe(0)
  })

  it('sums tokensIn + tokensOut and sorts desc', () => {
    const rows = [
      { model: 'a', totalCalls: 5, tokensIn: 100, tokensOut: 200 }, // 300
      { model: 'b', totalCalls: 3, tokensIn: 50, tokensOut: 50 }, // 100
      { model: 'c', totalCalls: 10, tokensIn: 500, tokensOut: 100 }, // 600
    ]
    const { slices, total } = buildSlices(rows, (r) => ({ key: r.model, display: r.model }))
    expect(total).toBe(1000)
    expect(slices.map((s) => s.value)).toEqual([600, 300, 100])
    expect(slices.map((s) => s.pct)).toEqual([60, 30, 10])
  })

  it('skips rows with zero tokens', () => {
    const rows = [
      { model: 'a', totalCalls: 0, tokensIn: 0, tokensOut: 0 },
      { model: 'b', totalCalls: 3, tokensIn: 100, tokensOut: 100 },
    ]
    const { slices, total } = buildSlices(rows, (r) => ({ key: r.model, display: r.model }))
    expect(slices.length).toBe(1)
    expect(total).toBe(200)
  })

  it('groups by key when labelOf returns duplicates', () => {
    const rows = [
      { totalCalls: 1, tokensIn: 50, tokensOut: 50 },
      { totalCalls: 2, tokensIn: 100, tokensOut: 100 },
    ]
    const { slices, total } = buildSlices(rows, () => ({ key: 'same', display: 'Same' }))
    expect(slices.length).toBe(1)
    expect(total).toBe(300)
    expect(slices[0].value).toBe(300)
  })

  it('assigns distinct colours up to palette length', () => {
    const rows = Array.from({ length: 4 }, (_, i) => ({
      model: `model-${i}`,
      totalCalls: 1,
      tokensIn: 100 - i * 10,
      tokensOut: 0,
    }))
    const { slices } = buildSlices(rows, (r) => ({ key: r.model, display: r.model }))
    const colours = new Set(slices.map((s) => s.colour))
    expect(colours.size).toBe(4)
  })
})

describe('UsageBreakdown component', () => {
  it('renders empty placeholder when both summaries are empty', async () => {
    getModelsMock.mockResolvedValue([])
    getAppsMock.mockResolvedValue([])
    render(<UsageBreakdown />)
    await waitFor(() => {
      expect(screen.getAllByText(/暂无调用记录/).length).toBeGreaterThan(0)
    })
  })

  it('renders model + tool donuts with data', async () => {
    getModelsMock.mockResolvedValue([
      { model: 'claude-opus-4-7', totalCalls: 10, tokensIn: 5000, tokensOut: 5000 },
      { model: 'gpt-4', totalCalls: 5, tokensIn: 2000, tokensOut: 1000 },
    ])
    getAppsMock.mockResolvedValue([
      { appId: 'claude', totalCalls: 8, tokensIn: 4000, tokensOut: 4000, cacheHits: 0 },
      { appId: 'codex', totalCalls: 7, tokensIn: 3000, tokensOut: 2000, cacheHits: 0 },
    ])
    render(<UsageBreakdown />)
    await waitFor(() => {
      expect(screen.getByText(/claude-opus-4-7/)).toBeInTheDocument()
    })
    // Tool labels resolve through TOOL_DISPLAY
    expect(screen.getByText('Claude Code')).toBeInTheDocument()
    expect(screen.getByText('Codex')).toBeInTheDocument()
  })

  it('falls back to "Unknown model" when model field is empty', async () => {
    getModelsMock.mockResolvedValue([
      { model: '', totalCalls: 1, tokensIn: 100, tokensOut: 100 },
    ])
    getAppsMock.mockResolvedValue([])
    render(<UsageBreakdown />)
    await waitFor(() => {
      expect(screen.getByText(/Unknown/)).toBeInTheDocument()
    })
  })
})
