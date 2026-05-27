import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

// ---------------------------------------------------------------------------
// Stubs
// ---------------------------------------------------------------------------

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

const mockAddToast = vi.fn()
vi.mock('../stores/toastStore', () => ({
  useToastStore: (selector: (s: { addToast: typeof mockAddToast }) => unknown) =>
    selector({ addToast: mockAddToast }),
}))

// Wails binding stubs
const mockList = vi.fn()
const mockRefresh = vi.fn()
const mockInstall = vi.fn()
const mockSavePreset = vi.fn()

vi.mock('../../wailsjs/go/main/App', () => ({
  McpMarketList: (...a: unknown[]) => mockList(...a),
  McpMarketRefresh: (...a: unknown[]) => mockRefresh(...a),
  McpMarketInstall: (...a: unknown[]) => mockInstall(...a),
  McpMarketSavePreset: (...a: unknown[]) => mockSavePreset(...a),
}))

vi.mock('../../wailsjs/go/models', () => ({
  mcpmarket: {},
}))

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const SAMPLE_SERVERS = [
  {
    id: 'github',
    qualifiedName: '@modelcontextprotocol/server-github',
    name: 'GitHub',
    description: 'Access GitHub repositories and issues.',
    category: 'vcs',
    stars: 5200,
    verified: true,
    builtin: true,
    configSchema: {
      type: 'object',
      properties: {
        GITHUB_PERSONAL_ACCESS_TOKEN: { type: 'string', description: 'GitHub PAT' },
      },
    },
  },
  {
    id: 'filesystem',
    qualifiedName: '@modelcontextprotocol/server-filesystem',
    name: 'Filesystem',
    description: 'Read and write files on the local filesystem.',
    category: 'files',
    stars: 4800,
    verified: true,
    builtin: true,
    configSchema: {},
  },
  {
    id: 'postgres',
    qualifiedName: '@modelcontextprotocol/server-postgres',
    name: 'PostgreSQL',
    description: 'Read-only PostgreSQL access.',
    category: 'database',
    stars: 3600,
    verified: false,
    builtin: false,
    configSchema: {
      type: 'object',
      properties: {
        connectionString: { type: 'string', description: 'Connection string' },
      },
    },
  },
]

import { McpMarketPage, ServerCard, InstallModal } from './McpMarketPage'

beforeEach(() => {
  vi.clearAllMocks()
  mockList.mockResolvedValue(SAMPLE_SERVERS)
  mockRefresh.mockResolvedValue({ success: true, message: 'ok' })
  mockInstall.mockResolvedValue({ success: true, message: 'installed', statuses: [] })
  mockSavePreset.mockResolvedValue({ success: true, message: 'saved', presetId: 'p1' })
})

// ---------------------------------------------------------------------------
// McpMarketPage — smoke
// ---------------------------------------------------------------------------

describe('McpMarketPage', () => {
  it('renders the page title', async () => {
    render(<McpMarketPage />)
    await waitFor(() => {
      expect(screen.getByText('MCP Market')).toBeDefined()
    })
  })

  it('shows loading text while fetching', () => {
    mockList.mockReturnValue(new Promise(() => {}))
    render(<McpMarketPage />)
    expect(screen.getByText('Loading servers…')).toBeDefined()
  })

  it('shows builtin cards after loading (default tab = builtin)', async () => {
    render(<McpMarketPage />)
    await waitFor(() => {
      const cards = screen.getAllByTestId('server-card')
      // github + filesystem are builtin; postgres is not
      expect(cards.length).toBe(2)
    }, { timeout: 3000 })
  })

  it('shows registry cards when registry tab is clicked', async () => {
    render(<McpMarketPage />)
    await waitFor(() => screen.getAllByTestId('server-card'), { timeout: 3000 })

    fireEvent.click(screen.getByTestId('tab-registry'))

    await waitFor(() => {
      const cards = screen.getAllByTestId('server-card')
      // Only postgres is from registry
      expect(cards.length).toBe(1)
    }, { timeout: 3000 })
  })
})

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

describe('McpMarketPage search', () => {
  it('filters by search query within builtin tab', async () => {
    render(<McpMarketPage />)
    await waitFor(() => screen.getAllByTestId('server-card'), { timeout: 3000 })

    const input = screen.getByPlaceholderText('Search MCP servers…')
    fireEvent.change(input, { target: { value: 'GitHub' } })

    await waitFor(() => {
      const cards = screen.getAllByTestId('server-card')
      expect(cards.length).toBe(1)
    }, { timeout: 3000 })
  })

  it('shows no-results when query matches nothing', async () => {
    render(<McpMarketPage />)
    await waitFor(() => screen.getAllByTestId('server-card'), { timeout: 3000 })

    const input = screen.getByPlaceholderText('Search MCP servers…')
    fireEvent.change(input, { target: { value: 'xyzzyquux_impossible' } })

    await waitFor(() => {
      expect(screen.getByText('No servers match your search.')).toBeDefined()
    }, { timeout: 3000 })
  })
})

// ---------------------------------------------------------------------------
// Install modal
// ---------------------------------------------------------------------------

describe('InstallModal', () => {
  const noop = vi.fn()

  it('does not render when server is null', () => {
    const { container } = render(
      <InstallModal
        server={null}
        onClose={noop}
        onInstall={noop}
        installing={false}
        installStatuses={[]}
      />,
    )
    expect(container.firstChild).toBeNull()
  })

  it('renders when a server is provided', async () => {
    render(
      <InstallModal
        server={SAMPLE_SERVERS[0] as any}
        onClose={noop}
        onInstall={noop}
        installing={false}
        installStatuses={[]}
      />,
    )
    await waitFor(() => {
      expect(screen.getByText('Install MCP Server')).toBeDefined()
    })
  })

  it('shows config input for server with configSchema', async () => {
    render(
      <InstallModal
        server={SAMPLE_SERVERS[0] as any}
        onClose={noop}
        onInstall={noop}
        installing={false}
        installStatuses={[]}
      />,
    )
    await waitFor(() => {
      expect(screen.getByTestId('config-input-GITHUB_PERSONAL_ACCESS_TOKEN')).toBeDefined()
    })
  })

  it('shows no-config message for server without configSchema properties', async () => {
    render(
      <InstallModal
        server={SAMPLE_SERVERS[1] as any}
        onClose={noop}
        onInstall={noop}
        installing={false}
        installStatuses={[]}
      />,
    )
    await waitFor(() => {
      expect(screen.getByText('No configuration required for this server.')).toBeDefined()
    })
  })
})

// ---------------------------------------------------------------------------
// Multi-CLI checkbox selection
// ---------------------------------------------------------------------------

describe('InstallModal multi-CLI checkboxes', () => {
  it('renders all target tool checkboxes', async () => {
    render(
      <InstallModal
        server={SAMPLE_SERVERS[0] as any}
        onClose={vi.fn()}
        onInstall={vi.fn()}
        installing={false}
        installStatuses={[]}
      />,
    )
    await waitFor(() => {
      expect(screen.getByTestId('tool-checkbox-claude_code')).toBeDefined()
      expect(screen.getByTestId('tool-checkbox-cursor')).toBeDefined()
      expect(screen.getByTestId('tool-checkbox-gemini')).toBeDefined()
      expect(screen.getByTestId('tool-checkbox-antigravity')).toBeDefined()
    })
  })

  it('allows unchecking a tool', async () => {
    render(
      <InstallModal
        server={SAMPLE_SERVERS[0] as any}
        onClose={vi.fn()}
        onInstall={vi.fn()}
        installing={false}
        installStatuses={[]}
      />,
    )
    await waitFor(() => {
      expect(screen.getByTestId('tool-checkbox-cursor')).toBeDefined()
    })
    const cursorCheckbox = screen.getByTestId('tool-checkbox-cursor') as HTMLInputElement
    expect(cursorCheckbox.checked).toBe(true)
    fireEvent.click(cursorCheckbox)
    expect(cursorCheckbox.checked).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// Builtin manifest fallback
// ---------------------------------------------------------------------------

describe('McpMarketPage builtin fallback', () => {
  it('shows builtin servers when registry list returns empty', async () => {
    mockList.mockResolvedValue(SAMPLE_SERVERS.filter((s) => s.builtin))
    render(<McpMarketPage />)
    await waitFor(() => {
      const cards = screen.getAllByTestId('server-card')
      expect(cards.length).toBeGreaterThanOrEqual(1)
    }, { timeout: 3000 })
  })
})

// ---------------------------------------------------------------------------
// Preset save
// ---------------------------------------------------------------------------

describe('McpMarketPage preset save', () => {
  it('shows save-as-preset checkbox in the modal', async () => {
    render(
      <InstallModal
        server={SAMPLE_SERVERS[0] as any}
        onClose={vi.fn()}
        onInstall={vi.fn()}
        installing={false}
        installStatuses={[]}
      />,
    )
    await waitFor(() => {
      expect(screen.getByTestId('save-preset-checkbox')).toBeDefined()
    })
  })
})

// ---------------------------------------------------------------------------
// ServerCard
// ---------------------------------------------------------------------------

describe('ServerCard', () => {
  it('renders name, category, and verified badge', () => {
    const onInstall = vi.fn()
    render(<ServerCard server={SAMPLE_SERVERS[0] as any} onInstall={onInstall} />)
    expect(screen.getByText('GitHub')).toBeDefined()
    expect(screen.getByText('vcs')).toBeDefined()
    expect(screen.getByText('Verified')).toBeDefined()
  })

  it('calls onInstall when Install button is clicked', () => {
    const onInstall = vi.fn()
    render(<ServerCard server={SAMPLE_SERVERS[0] as any} onInstall={onInstall} />)
    fireEvent.click(screen.getByText('Install'))
    expect(onInstall).toHaveBeenCalledWith(SAMPLE_SERVERS[0])
  })

  it('shows Built-in badge for builtin servers', () => {
    const onInstall = vi.fn()
    render(<ServerCard server={SAMPLE_SERVERS[0] as any} onInstall={onInstall} />)
    expect(screen.getByText('Built-in')).toBeDefined()
  })
})
