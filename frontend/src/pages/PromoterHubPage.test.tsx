import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'

// i18n stub — returns key so assertions can match translation keys
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
}))

const mockPromoterGetInfo = vi.fn()

vi.mock('../../wailsjs/go/main/App', () => ({
  PromoterGetInfo: (...a: unknown[]) => mockPromoterGetInfo(...a),
}))

vi.mock('../../wailsjs/go/models', () => ({
  main: {},
}))

// Reset Zustand promoterStore between tests so state does not leak.
import { usePromoterStore } from '../stores/promoterStore'

import { PromoterHubPage } from './PromoterHubPage'

const sampleInfo = {
  aff_code: 'AFF123',
  share_link: 'https://hub.lurus.cn/ref/AFF123',
  gateway_url: 'https://hub.lurus.cn',
  total_referrals: 42,
  total_earned: 98.76,
  pending_earned: 12.34,
}

beforeEach(() => {
  vi.clearAllMocks()
  // Reset store to clean state
  usePromoterStore.setState({ info: null, loading: false })
})

describe('PromoterHubPage', () => {
  it('shows a loading spinner while the request is in flight', () => {
    // Never resolve — keeps the page in loading state.
    mockPromoterGetInfo.mockReturnValue(new Promise(() => {}))
    render(<PromoterHubPage />)
    // The spinner has role="img" via lucide-react; we look for it by finding
    // the wrapper in the loading branch which is the only child rendered.
    expect(document.querySelector('svg.animate-spin')).toBeInTheDocument()
  })

  it('renders promo code, share link and KPIs after load', async () => {
    mockPromoterGetInfo.mockResolvedValue(sampleInfo)
    render(<PromoterHubPage />)

    // Wait until aff_code appears (loading spinner gone)
    expect(await screen.findByText('AFF123')).toBeInTheDocument()

    // Share link is shown below the buttons
    expect(screen.getByText('https://hub.lurus.cn/ref/AFF123')).toBeInTheDocument()

    // KPI values
    expect(screen.getByText('42')).toBeInTheDocument()
    expect(screen.getByText('$98.76')).toBeInTheDocument()
    expect(screen.getByText('$12.34')).toBeInTheDocument()
  })

  it('shows dash placeholder when info is null (no data returned)', async () => {
    // Resolve with minimal empty-ish object
    mockPromoterGetInfo.mockResolvedValue({
      aff_code: '',
      share_link: '',
      gateway_url: '',
      total_referrals: 0,
      total_earned: 0,
      pending_earned: 0,
    })
    render(<PromoterHubPage />)

    // Wait for loading to complete
    await waitFor(() => {
      expect(document.querySelector('svg.animate-spin')).not.toBeInTheDocument()
    })

    // Empty aff_code shows the em-dash placeholder
    expect(screen.getByText('—')).toBeInTheDocument()
    // Both earned KPIs show $0.00 (total_earned and pending_earned)
    expect(screen.getAllByText('$0.00')).toHaveLength(2)
  })

  it('renders error banner when PromoterGetInfo rejects', async () => {
    mockPromoterGetInfo.mockRejectedValue(new Error('network timeout'))
    render(<PromoterHubPage />)

    await waitFor(() => {
      expect(screen.getByText(/network timeout/i)).toBeInTheDocument()
    })
  })

  it('retry action in error state re-fetches and shows data on success', async () => {
    mockPromoterGetInfo
      .mockRejectedValueOnce(new Error('first attempt failed'))
      .mockResolvedValueOnce(sampleInfo)

    render(<PromoterHubPage />)

    // First call fails — error banner shows up
    await waitFor(() => {
      expect(screen.getByText(/first attempt failed/i)).toBeInTheDocument()
    })

    // The retry button label comes from translation key 'error.action.retry'
    const retryBtn = screen.getByText('error.action.retry')
    fireEvent.click(retryBtn)

    // After retry resolves, aff_code is visible
    expect(await screen.findByText('AFF123')).toBeInTheDocument()
  })

  it('copy code button calls clipboard.writeText with aff_code', async () => {
    mockPromoterGetInfo.mockResolvedValue(sampleInfo)

    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
    })

    render(<PromoterHubPage />)
    await screen.findByText('AFF123')

    // There are two copy buttons: one for code, one for link.
    // The code copy button is the secondary button next to the code block.
    // We locate all buttons with a Copy icon svg and click the first one.
    const copyButtons = screen.getAllByRole('button')
    // The icon-only copy button is before the full "Copy Link" button
    const iconOnlyCopyBtn = copyButtons.find(
      (btn) => btn.querySelector('svg') && btn.textContent?.trim() === '',
    )
    expect(iconOnlyCopyBtn).toBeTruthy()
    fireEvent.click(iconOnlyCopyBtn!)

    expect(writeText).toHaveBeenCalledWith('AFF123')
  })

  it('copy link button calls clipboard.writeText with share_link', async () => {
    mockPromoterGetInfo.mockResolvedValue(sampleInfo)

    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
    })

    render(<PromoterHubPage />)
    await screen.findByText('AFF123')

    // The copy-link button contains visible text from the translation key
    const copyLinkBtn = screen.getByText('promoter.copyLink').closest('button')
    expect(copyLinkBtn).toBeTruthy()
    fireEvent.click(copyLinkBtn!)

    expect(writeText).toHaveBeenCalledWith('https://hub.lurus.cn/ref/AFF123')
  })
})
