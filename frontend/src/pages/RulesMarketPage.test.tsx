import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

// i18n stub: returns the fallback string when provided
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallbackOrOpts?: string | Record<string, unknown>) => {
      const fallback = typeof fallbackOrOpts === 'string' ? fallbackOrOpts : key
      return fallback
    },
    i18n: { language: 'en', changeLanguage: vi.fn() },
  }),
  Trans: ({ children, i18nKey }: { children?: React.ReactNode; i18nKey?: string }) =>
    children ?? i18nKey ?? null,
}))

// Toast stub
const mockAddToast = vi.fn()
vi.mock('../stores/toastStore', () => ({
  useToastStore: (selector: (s: { addToast: typeof mockAddToast }) => unknown) =>
    selector({ addToast: mockAddToast }),
}))

// Wails binding stubs
const mockList = vi.fn()
const mockWrite = vi.fn()

vi.mock('../../wailsjs/go/main/App', () => ({
  RulesMarketList: (...a: unknown[]) => mockList(...a),
  RulesMarketWrite: (...a: unknown[]) => mockWrite(...a),
}))

vi.mock('../../wailsjs/go/models', () => ({
  rulesmarket: {},
}))

// Sample templates fixture
const SAMPLE_TEMPLATES = [
  {
    id: 'nextjs-best-practices',
    name: 'Next.js Best Practices',
    category: 'framework',
    framework: 'Next.js',
    description: 'TypeScript-first rules for Next.js App Router projects.',
    format: 'cursorrules',
    source_url: '',
    content: '## Next.js rules',
  },
  {
    id: 'golang-api-service',
    name: 'Go API Service',
    category: 'language',
    framework: 'Go',
    description: 'Rules for idiomatic Go HTTP services.',
    format: 'agents_md',
    source_url: '',
    content: '# Project Rules\n\nGo rules.',
  },
  {
    id: 'react-typescript-component',
    name: 'React TypeScript Components',
    category: 'framework',
    framework: 'React',
    description: 'Rules for writing clean React components.',
    format: 'cursorrules',
    source_url: '',
    content: '## React rules',
  },
]

import { RulesMarketPage, TemplateCard, InstallModal } from './RulesMarketPage'

beforeEach(() => {
  vi.clearAllMocks()
  mockList.mockResolvedValue(SAMPLE_TEMPLATES)
  mockWrite.mockResolvedValue({ success: true, message: 'ok' })
})

// ---------------------------------------------------------------------------
// RulesMarketPage smoke + loading
// ---------------------------------------------------------------------------

describe('RulesMarketPage', () => {
  it('renders the page title', async () => {
    render(<RulesMarketPage />)
    await waitFor(() => {
      expect(screen.getByText('Rules Market')).toBeDefined()
    })
  })

  it('shows all template cards after loading', async () => {
    render(<RulesMarketPage />)
    await waitFor(() => {
      const cards = screen.getAllByTestId('template-card')
      expect(cards.length).toBe(SAMPLE_TEMPLATES.length)
    }, { timeout: 3000 })
  })

  it('shows loading text while fetching', () => {
    // Never resolve so the page stays in loading state
    mockList.mockReturnValue(new Promise(() => {}))
    render(<RulesMarketPage />)
    expect(screen.getByText('Loading templates…')).toBeDefined()
  })
})

// ---------------------------------------------------------------------------
// Category filter
// ---------------------------------------------------------------------------

describe('RulesMarketPage category filter', () => {
  it('filters to framework category on click', async () => {
    render(<RulesMarketPage />)
    // Wait for load (allow longer for async state)
    await waitFor(() => screen.getAllByTestId('template-card'), { timeout: 3000 })

    // i18n stub returns fallback arg — buttons use t(`rulesmarket.category.${cat}`, cat)
    // so the text is the cat value itself (e.g. 'framework').
    // The sidebar nav contains exactly one button per category.
    const allButtons = screen.getAllByRole('button')
    const frameworkBtn = allButtons.find((b) => b.textContent?.trim() === 'framework')
    if (!frameworkBtn) throw new Error('framework button not found')
    fireEvent.click(frameworkBtn)

    await waitFor(() => {
      const cards = screen.getAllByTestId('template-card')
      // Only nextjs + react are framework
      expect(cards.length).toBe(2)
    }, { timeout: 3000 })
  })

  it('shows no-results message when filter yields nothing', async () => {
    render(<RulesMarketPage />)
    await waitFor(() => screen.getAllByTestId('template-card'), { timeout: 3000 })

    // 'testing' category has no fixtures
    const allButtons = screen.getAllByRole('button')
    const testingBtn = allButtons.find((b) => b.textContent?.trim() === 'testing')
    if (!testingBtn) throw new Error('testing button not found')
    fireEvent.click(testingBtn)

    await waitFor(() => {
      expect(screen.getByText('No templates match your search.')).toBeDefined()
    }, { timeout: 3000 })
  })
})

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

describe('RulesMarketPage search', () => {
  it('filters by search query', async () => {
    render(<RulesMarketPage />)
    await waitFor(() => screen.getAllByTestId('template-card'), { timeout: 3000 })

    const input = screen.getByPlaceholderText('Search templates…')
    fireEvent.change(input, { target: { value: 'Go' } })

    await waitFor(() => {
      const cards = screen.getAllByTestId('template-card')
      // "Go API Service" has framework 'Go' — at least 1 should match
      expect(cards.length).toBeGreaterThanOrEqual(1)
    }, { timeout: 3000 })
  })

  it('shows no-results when query matches nothing', async () => {
    render(<RulesMarketPage />)
    await waitFor(() => screen.getAllByTestId('template-card'), { timeout: 3000 })

    const input = screen.getByPlaceholderText('Search templates…')
    fireEvent.change(input, { target: { value: 'xyzzyquux_impossible' } })

    await waitFor(() => {
      expect(screen.getByText('No templates match your search.')).toBeDefined()
    }, { timeout: 3000 })
  })
})

// ---------------------------------------------------------------------------
// Install modal
// ---------------------------------------------------------------------------

describe('InstallModal', () => {
  const mockOnClose = vi.fn()
  const mockOnInstall = vi.fn()

  it('does not render when template is null', () => {
    const { container } = render(
      <InstallModal
        template={null}
        onClose={mockOnClose}
        onInstall={mockOnInstall}
        installing={false}
      />,
    )
    expect(container.firstChild).toBeNull()
  })

  it('renders when a template is provided', () => {
    render(
      <InstallModal
        template={SAMPLE_TEMPLATES[0]}
        onClose={mockOnClose}
        onInstall={mockOnInstall}
        installing={false}
      />,
    )
    expect(screen.getByText('Install rule template')).toBeDefined()
  })

  it('Install button is disabled when projectDir is empty', () => {
    render(
      <InstallModal
        template={SAMPLE_TEMPLATES[0]}
        onClose={mockOnClose}
        onInstall={mockOnInstall}
        installing={false}
      />,
    )
    const installBtn = screen.getAllByText('Install').find((el) => el.tagName === 'BUTTON' || el.closest('button'))
    const btn = installBtn?.closest('button') ?? installBtn
    expect((btn as HTMLButtonElement)?.disabled).toBe(true)
  })

  it('shows loading state when installing', () => {
    render(
      <InstallModal
        template={SAMPLE_TEMPLATES[0]}
        onClose={mockOnClose}
        onInstall={mockOnInstall}
        installing={true}
      />,
    )
    // Cancel button should be disabled while installing
    const cancelBtn = screen.getByText('Cancel').closest('button') as HTMLButtonElement
    expect(cancelBtn.disabled).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// Format selection
// ---------------------------------------------------------------------------

describe('InstallModal format selection', () => {
  it('allows selecting claude_md format', async () => {
    render(
      <InstallModal
        template={SAMPLE_TEMPLATES[0]}
        onClose={vi.fn()}
        onInstall={vi.fn()}
        installing={false}
      />,
    )
    // Modal uses AnimatePresence + Radix portal — wait for the content
    await waitFor(() => {
      expect(screen.getByText('Install rule template')).toBeDefined()
    })
    // i18n stub: t('rulesmarket.formatLabel.claude_md', 'claude_md') → 'claude_md'
    const claudeBtn = screen.getByText('claude_md')
    fireEvent.click(claudeBtn)
    expect(claudeBtn.className).toContain('text-primary')
  })

  it('allows selecting cursorrules format', async () => {
    render(
      <InstallModal
        template={SAMPLE_TEMPLATES[0]}
        onClose={vi.fn()}
        onInstall={vi.fn()}
        installing={false}
      />,
    )
    await waitFor(() => {
      expect(screen.getByText('Install rule template')).toBeDefined()
    })
    // Default selected format is agents_md; switch to cursorrules
    // i18n stub: t('rulesmarket.formatLabel.cursorrules', 'cursorrules') → 'cursorrules'
    const cursorBtn = screen.getByText('cursorrules')
    fireEvent.click(cursorBtn)
    expect(cursorBtn.className).toContain('text-primary')
  })
})

// ---------------------------------------------------------------------------
// TemplateCard
// ---------------------------------------------------------------------------

describe('TemplateCard', () => {
  it('renders name, framework, and category', () => {
    const onInstall = vi.fn()
    render(<TemplateCard template={SAMPLE_TEMPLATES[0]} onInstall={onInstall} />)
    expect(screen.getByText('Next.js Best Practices')).toBeDefined()
    expect(screen.getByText('Next.js')).toBeDefined()
    expect(screen.getByText('framework')).toBeDefined()
  })

  it('calls onInstall when Install button is clicked', () => {
    const onInstall = vi.fn()
    render(<TemplateCard template={SAMPLE_TEMPLATES[0]} onInstall={onInstall} />)
    fireEvent.click(screen.getByText('Install'))
    expect(onInstall).toHaveBeenCalledWith(SAMPLE_TEMPLATES[0])
  })

  it('shows Built-in badge for templates without a source_url', () => {
    const onInstall = vi.fn()
    render(<TemplateCard template={SAMPLE_TEMPLATES[0]} onInstall={onInstall} />)
    expect(screen.getByText('Built-in')).toBeDefined()
  })
})
