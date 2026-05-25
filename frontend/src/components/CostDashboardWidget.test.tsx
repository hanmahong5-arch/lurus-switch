import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: string) => fallback ?? key,
    i18n: { language: 'zh' },
  }),
}))

const fetchMock = vi.fn()
vi.mock('../../wailsjs/go/main/App', () => ({
  GetCostDashboard: (...args: unknown[]) => fetchMock(...args),
}))

vi.mock('../../wailsjs/go/models', () => ({
  main: {},
}))

import { CostDashboardWidget } from './CostDashboardWidget'

describe('CostDashboardWidget', () => {
  afterEach(() => {
    cleanup()
    fetchMock.mockReset()
  })

  it('renders today USD + tokens + top model breakdown', async () => {
    fetchMock.mockResolvedValue({
      todayUSD: 1.234,
      todayTokensIn: 50000,
      todayTokensOut: 25000,
      todayCalls: 7,
      byModel: [
        { model: 'claude-opus-4-7', totalCalls: 3, tokensIn: 30000, tokensOut: 20000, costUSD: 1.0 },
        { model: 'claude-sonnet-4-6', totalCalls: 4, tokensIn: 20000, tokensOut: 5000, costUSD: 0.234 },
      ],
      budgetEnabled: false,
    })

    render(<CostDashboardWidget />)

    await waitFor(() => {
      const card = screen.getByTestId('cost-dashboard')
      expect(card.textContent).toContain('$1.23')
      expect(card.textContent).toContain('claude-opus-4-7')
      expect(card.textContent).toContain('claude-sonnet-4-6')
    })
  })

  it('shows budget wall progress when enabled', async () => {
    fetchMock.mockResolvedValue({
      todayUSD: 0,
      todayTokensIn: 0,
      todayTokensOut: 0,
      todayCalls: 0,
      byModel: [],
      budgetEnabled: true,
      budgetDailyTokens: 1_000_000,
      budgetDailyUsed: 800_000,
      budgetDailyPct: 80,
      budgetHitDaily: false,
    })

    render(<CostDashboardWidget />)

    await waitFor(() => {
      expect(screen.getByText(/预算墙/)).toBeTruthy()
      expect(screen.getByText(/800.0k/)).toBeTruthy()
    })
  })

  it('surfaces quota when present', async () => {
    fetchMock.mockResolvedValue({
      todayUSD: 0,
      todayTokensIn: 0,
      todayTokensOut: 0,
      todayCalls: 0,
      byModel: [],
      budgetEnabled: false,
      quota: { quota: 1000, used_quota: 250, remaining_quota: 750 },
    })

    render(<CostDashboardWidget />)

    await waitFor(() => {
      expect(screen.getByText(/远端配额/)).toBeTruthy()
      expect(screen.getByText(/250.*1000/)).toBeTruthy()
    })
  })
})
