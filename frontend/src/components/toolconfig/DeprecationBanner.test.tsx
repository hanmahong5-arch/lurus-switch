import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { DeprecationBanner } from './DeprecationBanner'
import type { GeminiDeprecationStatus } from './DeprecationBanner'

// i18n stub — returns interpolated key text so we can assert on content.
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) => {
      if (!opts) return key
      return Object.entries(opts).reduce<string>(
        (s, [k, v]) => s.replace(new RegExp(`{{\\s*${k}\\s*}}`, 'g'), String(v)),
        key,
      )
    },
    i18n: { language: 'en', changeLanguage: vi.fn() },
  }),
}))

// Wails binding mocks
const mockBuildPlan = vi.fn()
const mockApplyMigration = vi.fn()
vi.mock('../../../wailsjs/go/main/App', () => ({
  GetGeminiDeprecationStatus: vi.fn(),
  BuildGeminiMigrationPlan: () => mockBuildPlan(),
  ApplyGeminiMigration: () => mockApplyMigration(),
}))

// Toast store stub
vi.mock('../../stores/toastStore', () => ({
  useToastStore: () => vi.fn(),
}))

const activeStatus: GeminiDeprecationStatus = {
  isDeprecated: true,
  eolDate: '2026-06-18',
  daysRemaining: 22,
  successorTool: 'antigravity',
}

const passedStatus: GeminiDeprecationStatus = {
  isDeprecated: true,
  eolDate: '2026-06-18',
  daysRemaining: -5,
  successorTool: 'antigravity',
}

const notDeprecatedStatus: GeminiDeprecationStatus = {
  isDeprecated: false,
  eolDate: '2099-01-01',
  daysRemaining: 999,
  successorTool: 'antigravity',
}

describe('DeprecationBanner', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockBuildPlan.mockResolvedValue({
      sourcePath: '/home/.gemini/settings.json',
      targetPath: '/home/Antigravity/config.json',
      fields: [
        { geminiField: 'model.name', antigravityField: 'model.name', value: 'gemini-2.5-flash', needsManualReview: false },
        { geminiField: 'proxy', antigravityField: 'proxy', value: 'http://very-long-value'.repeat(20), needsManualReview: true, note: 'Needs review' },
      ],
      warnings: [],
    })
    mockApplyMigration.mockResolvedValue({ success: true, message: 'done', tool: 'gemini' })
  })

  it('renders the banner when deprecated', () => {
    render(<DeprecationBanner cli="gemini" status={activeStatus} />)
    expect(screen.getByTestId('deprecation-banner')).toBeDefined()
  })

  it('does not render when not deprecated', () => {
    render(<DeprecationBanner cli="gemini" status={notDeprecatedStatus} />)
    expect(screen.queryByTestId('deprecation-banner')).toBeNull()
  })

  it('shows days remaining text when EOL is in the future', () => {
    const { container } = render(<DeprecationBanner cli="gemini" status={activeStatus} />)
    // i18n stub returns the key string — the days label key must be present
    expect(container.textContent).toContain('toolConfig.deprecation.daysLeft')
  })

  it('shows EOL-passed message when daysRemaining is negative', () => {
    render(<DeprecationBanner cli="gemini" status={passedStatus} />)
    // i18n stub returns the key as-is when no interpolation vars
    expect(screen.getByText('toolConfig.deprecation.eolPassed')).toBeDefined()
  })

  it('shows migrate button', () => {
    render(<DeprecationBanner cli="gemini" status={activeStatus} />)
    expect(screen.getByTestId('migrate-btn')).toBeDefined()
  })

  it('dismisses when X is clicked', () => {
    render(<DeprecationBanner cli="gemini" status={activeStatus} />)
    fireEvent.click(screen.getByTestId('dismiss-banner-btn'))
    expect(screen.queryByTestId('deprecation-banner')).toBeNull()
  })

  it('opens plan modal on migrate click and shows field mapping', async () => {
    render(<DeprecationBanner cli="gemini" status={activeStatus} />)
    fireEvent.click(screen.getByTestId('migrate-btn'))

    await waitFor(() => {
      expect(mockBuildPlan).toHaveBeenCalledOnce()
    })

    // model.name appears in both gemini and antigravity columns → multiple matches
    expect(screen.getAllByText('model.name').length).toBeGreaterThanOrEqual(1)
  })

  it('shows needsManualReview indicator in plan', async () => {
    render(<DeprecationBanner cli="gemini" status={activeStatus} />)
    fireEvent.click(screen.getByTestId('migrate-btn'))

    await waitFor(() => {
      // proxy appears in both columns → use getAllByText
      expect(screen.getAllByText('proxy').length).toBeGreaterThanOrEqual(1)
    })
    // i18n key for needsReview — may appear multiple times (one per review field)
    expect(screen.getAllByText(/toolConfig\.deprecation\.needsReview|Needs review/i).length).toBeGreaterThanOrEqual(1)
  })

  it('calls ApplyGeminiMigration when apply button is clicked', async () => {
    render(<DeprecationBanner cli="gemini" status={activeStatus} />)
    fireEvent.click(screen.getByTestId('migrate-btn'))

    await waitFor(() => {
      expect(screen.getByTestId('apply-migration-btn')).toBeDefined()
    })

    fireEvent.click(screen.getByTestId('apply-migration-btn'))

    await waitFor(() => {
      expect(mockApplyMigration).toHaveBeenCalledOnce()
    })
  })

  it('calls onMigrated callback after successful migration', async () => {
    const onMigrated = vi.fn()
    render(<DeprecationBanner cli="gemini" status={activeStatus} onMigrated={onMigrated} />)
    fireEvent.click(screen.getByTestId('migrate-btn'))

    await waitFor(() => {
      expect(screen.getByTestId('apply-migration-btn')).toBeDefined()
    })

    fireEvent.click(screen.getByTestId('apply-migration-btn'))

    await waitFor(() => {
      expect(onMigrated).toHaveBeenCalledOnce()
    })
  })

  it('does NOT call onMigrated when migration fails', async () => {
    mockApplyMigration.mockResolvedValue({ success: false, message: 'disk full', tool: 'gemini' })
    const onMigrated = vi.fn()
    render(<DeprecationBanner cli="gemini" status={activeStatus} onMigrated={onMigrated} />)
    fireEvent.click(screen.getByTestId('migrate-btn'))

    await waitFor(() => {
      expect(screen.getByTestId('apply-migration-btn')).toBeDefined()
    })

    fireEvent.click(screen.getByTestId('apply-migration-btn'))

    await waitFor(() => {
      expect(mockApplyMigration).toHaveBeenCalledOnce()
    })
    expect(onMigrated).not.toHaveBeenCalled()
  })
})
