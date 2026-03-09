import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { DepTreePanel } from './DepTreePanel'

// Mock i18n
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) => {
      // Return the key with interpolated values for testing
      if (opts) {
        let result = key
        for (const [k, v] of Object.entries(opts)) {
          result = result.replace(`{{${k}}}`, String(v))
        }
        return result
      }
      return key
    },
  }),
}))

// Mock Wails bindings
const mockCheckDependencies = vi.fn()
const mockInstallDependency = vi.fn()

vi.mock('../../wailsjs/go/main/App', () => ({
  CheckDependencies: (...args: unknown[]) => mockCheckDependencies(...args),
  InstallDependency: (...args: unknown[]) => mockInstallDependency(...args),
}))

const allMetResult = {
  runtimes: [
    { id: 'nodejs', name: 'Node.js', installed: true, version: '22.14.0', path: '/usr/bin/node', required: true, tools: ['claude', 'codex', 'gemini', 'openclaw'] },
    { id: 'bun', name: 'Bun', installed: true, version: '1.2.1', path: '/home/.bun/bin/bun', required: true, tools: ['claude', 'codex', 'gemini', 'openclaw'] },
    { id: 'none', name: 'Standalone', installed: true, version: '', path: '', required: false, tools: ['picoclaw', 'nullclaw', 'zeroclaw'] },
  ],
  allMet: true,
}

const missingResult = {
  runtimes: [
    { id: 'nodejs', name: 'Node.js', installed: false, version: '', path: '', required: true, tools: ['claude', 'codex', 'gemini', 'openclaw'] },
    { id: 'bun', name: 'Bun', installed: false, version: '', path: '', required: true, tools: ['claude', 'codex', 'gemini', 'openclaw'] },
    { id: 'none', name: 'Standalone', installed: true, version: '', path: '', required: false, tools: ['picoclaw', 'nullclaw', 'zeroclaw'] },
  ],
  allMet: false,
}

describe('DepTreePanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should show all-met status when all deps installed', async () => {
    mockCheckDependencies.mockResolvedValue(allMetResult)

    render(<DepTreePanel />)

    await waitFor(() => {
      expect(screen.getByText('dashboard.deps.title')).toBeDefined()
      expect(screen.getByText('dashboard.deps.allMet')).toBeDefined()
    })
  })

  it('should show missing count when deps missing', async () => {
    mockCheckDependencies.mockResolvedValue(missingResult)

    render(<DepTreePanel />)

    await waitFor(() => {
      expect(screen.getByText('dashboard.deps.missing')).toBeDefined()
    })
  })

  it('should auto-expand when deps are missing', async () => {
    mockCheckDependencies.mockResolvedValue(missingResult)

    render(<DepTreePanel />)

    await waitFor(() => {
      // Install button should be visible for Node.js
      expect(screen.getByText('dashboard.deps.install')).toBeDefined()
    })
  })

  it('should show install all button when deps missing', async () => {
    mockCheckDependencies.mockResolvedValue(missingResult)

    render(<DepTreePanel />)

    await waitFor(() => {
      expect(screen.getByText('dashboard.deps.installAll')).toBeDefined()
    })
  })

  it('should call InstallDependency when install clicked', async () => {
    mockCheckDependencies.mockResolvedValue(missingResult)
    mockInstallDependency.mockResolvedValue({ success: true, message: 'installed' })

    render(<DepTreePanel />)

    await waitFor(() => {
      const installBtn = screen.getByText('dashboard.deps.install')
      fireEvent.click(installBtn)
    })

    await waitFor(() => {
      expect(mockInstallDependency).toHaveBeenCalledWith('nodejs')
    })
  })

  it('should show standalone tools section', async () => {
    mockCheckDependencies.mockResolvedValue(allMetResult)

    render(<DepTreePanel />)

    // Click to expand
    await waitFor(() => {
      const title = screen.getByText('dashboard.deps.title')
      fireEvent.click(title)
    })

    await waitFor(() => {
      expect(screen.getByText('dashboard.deps.standalone')).toBeDefined()
    })
  })
})
