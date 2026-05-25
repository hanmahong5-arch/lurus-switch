import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallbackOrOpts?: string | Record<string, unknown>) => {
      return typeof fallbackOrOpts === 'string' ? fallbackOrOpts : key
    },
    i18n: { language: 'zh', changeLanguage: vi.fn() },
  }),
}))

const dryRunMock = vi.fn()
vi.mock('../../../wailsjs/go/main/App', () => ({
  DryRunRouter: (...args: unknown[]) => dryRunMock(...args),
}))

vi.mock('../../../wailsjs/go/models', () => ({
  relay: {
    PickResult: class {
      Endpoint: any
      MatchedBy: string
      Healthy: any[]
      Ordered: any[]
      constructor(s: any = {}) {
        this.Endpoint = s.Endpoint
        this.MatchedBy = s.MatchedBy
        this.Healthy = s.Healthy ?? []
        this.Ordered = s.Ordered ?? []
      }
    },
  },
}))

import { RouterDryRunPanel } from './RouterDryRunPanel'

describe('RouterDryRunPanel', () => {
  it('renders the picked endpoint + rule + cascade chain after run', async () => {
    dryRunMock.mockResolvedValue({
      Endpoint: { id: 'alpha', name: 'Alpha', latencyMs: 42 },
      MatchedBy: 'claude-to-alpha',
      Healthy: [
        { id: 'alpha', name: 'Alpha', latencyMs: 42 },
        { id: 'beta', name: 'Beta', latencyMs: 120 },
      ],
      Ordered: [
        { id: 'alpha', name: 'Alpha', latencyMs: 42 },
        { id: 'beta', name: 'Beta', latencyMs: 120 },
      ],
    })

    render(<RouterDryRunPanel />)
    fireEvent.click(screen.getByText('运行'))

    await waitFor(() => {
      const result = screen.getByTestId('dry-run-result')
      expect(result).toBeTruthy()
      expect(result.textContent).toContain('Alpha')
      expect(result.textContent).toContain('claude-to-alpha')
      // Ordered chain should mention both endpoints.
      expect(result.textContent).toContain('Beta')
    })
  })

  it('surfaces no-rule case when MatchedBy is empty', async () => {
    dryRunMock.mockResolvedValue({
      Endpoint: { id: 'beta', name: 'Beta', latencyMs: 80 },
      MatchedBy: '',
      Healthy: [{ id: 'beta', name: 'Beta', latencyMs: 80 }],
      Ordered: [{ id: 'beta', name: 'Beta', latencyMs: 80 }],
    })
    render(<RouterDryRunPanel />)
    fireEvent.click(screen.getByText('运行'))

    await waitFor(() => {
      const result = screen.getByTestId('dry-run-result')
      expect(result.textContent).toContain('无规则匹配')
    })
  })

  it('shows error when DryRunRouter rejects', async () => {
    dryRunMock.mockRejectedValue(new Error('no healthy endpoints'))
    render(<RouterDryRunPanel />)
    fireEvent.click(screen.getByText('运行'))

    await waitFor(() => {
      expect(screen.getByText(/no healthy endpoints/i)).toBeTruthy()
    })
  })
})
