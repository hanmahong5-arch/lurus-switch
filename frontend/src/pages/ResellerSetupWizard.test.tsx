import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

// i18n stub — returns the fallback string when provided, otherwise the key.
// Reseller wizard uses fallback strings for all user-facing copy.
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

// Wails binding mocks — the wizard imports ListResellerDeployKinds,
// TestHubConnection, ProvisionResellerHub, SetAppMode from this module.
const mockListResellerDeployKinds = vi.fn()
const mockTestHubConnection = vi.fn()
const mockProvisionResellerHub = vi.fn()
const mockSetAppMode = vi.fn()
// ModelHealthMatrix also uses these two bindings; stub them so the
// done-step renders without unhandled promise rejections.
const mockRunModelHealthCheck = vi.fn()
const mockGetLastHealthCheckResults = vi.fn()

vi.mock('../../wailsjs/go/main/App', () => ({
  ListResellerDeployKinds: (...a: unknown[]) => mockListResellerDeployKinds(...a),
  TestHubConnection: (...a: unknown[]) => mockTestHubConnection(...a),
  ProvisionResellerHub: (...a: unknown[]) => mockProvisionResellerHub(...a),
  SetAppMode: (...a: unknown[]) => mockSetAppMode(...a),
  RunModelHealthCheck: (...a: unknown[]) => mockRunModelHealthCheck(...a),
  GetLastHealthCheckResults: (...a: unknown[]) => mockGetLastHealthCheckResults(...a),
}))

// Wails runtime EventsOn is imported by ModelHealthMatrix.
vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn(() => () => {}),
}))

// configStore — the wizard reads setAppMode from here.
const mockSetAppModeLocal = vi.fn()
vi.mock('../stores/configStore', () => ({
  useConfigStore: (selector: (s: { setAppMode: (...a: unknown[]) => void }) => unknown) =>
    selector({ setAppMode: mockSetAppModeLocal }),
}))

// dirtyStore — useDirtyGuard writes into it; a no-op stub is sufficient.
vi.mock('../stores/dirtyStore', () => ({
  useDirtyStore: (selector: (s: { setDirty: (...a: unknown[]) => void }) => unknown) =>
    selector({ setDirty: vi.fn() }),
}))

import { ResellerSetupWizard } from './ResellerSetupWizard'

// Two manual-kind entries: one implemented, one stub (coming-soon).
const KINDS_FIXTURE = [
  {
    kind: 'manual',
    implemented: true,
    labelZh: '手动接入',
    labelEn: 'Manual',
    descriptionZh: '自行运维 newhub 实例',
    descriptionEn: 'Self-host newhub',
  },
  {
    kind: 'cloud',
    implemented: false,
    labelZh: '云托管',
    labelEn: 'Cloud',
    descriptionZh: '即将推出',
    descriptionEn: 'Coming soon',
  },
]

beforeEach(() => {
  vi.clearAllMocks()
  // Default: listing succeeds with two kinds.
  mockListResellerDeployKinds.mockResolvedValue(KINDS_FIXTURE)
  // ModelHealthMatrix: succeed silently so the done step doesn't error.
  mockGetLastHealthCheckResults.mockResolvedValue([])
  mockRunModelHealthCheck.mockResolvedValue(undefined)
})

describe('ResellerSetupWizard — step 1: pick provider', () => {
  it('renders the wizard title and first-step header on initial load', async () => {
    render(<ResellerSetupWizard onComplete={vi.fn()} />)
    await waitFor(() => {
      expect(screen.getByText('配置 Reseller Hub')).toBeDefined()
      expect(screen.getByText('选择部署方式')).toBeDefined()
    })
  })

  it('shows deploy-kind options returned by ListResellerDeployKinds', async () => {
    render(<ResellerSetupWizard onComplete={vi.fn()} />)
    await waitFor(() => {
      expect(screen.getByText('手动接入')).toBeDefined()
      expect(screen.getByText('云托管')).toBeDefined()
    })
  })

  it('shows an "unavailable" badge for un-implemented (cloud) kinds', async () => {
    render(<ResellerSetupWizard onComplete={vi.fn()} />)
    await waitFor(() => {
      // Auto-deploy backends are stubs; the wizard honestly marks them 暂不可用.
      expect(screen.getAllByText('暂不可用').length).toBeGreaterThan(0)
    })
  })

  it('falls back to a single manual entry when ListResellerDeployKinds rejects', async () => {
    mockListResellerDeployKinds.mockRejectedValueOnce(new Error('network'))
    render(<ResellerSetupWizard onComplete={vi.fn()} />)
    await waitFor(() => {
      expect(screen.getByText('手动接入')).toBeDefined()
    })
  })

  it('navigates to the manual-entry step when an implemented kind is clicked', async () => {
    render(<ResellerSetupWizard onComplete={vi.fn()} />)
    await waitFor(() => screen.getByText('手动接入'))
    fireEvent.click(screen.getByText('手动接入'))
    await waitFor(() => {
      expect(screen.getByText('填写 Hub 连接信息')).toBeDefined()
    })
  })

  it('does NOT navigate when a not-yet-available (cloud) kind is clicked — it is gated', async () => {
    render(<ResellerSetupWizard onComplete={vi.fn()} />)
    await waitFor(() => screen.getByText('云托管'))
    fireEvent.click(screen.getByText('云托管'))
    // Cloud auto-deploy is a backend stub, so the option is disabled + guarded:
    // clicking must NOT advance to the manual hub-connection step.
    await new Promise((r) => setTimeout(r, 30))
    expect(screen.queryByText('填写 Hub 连接信息')).toBeNull()
    expect(screen.getByText('云托管')).toBeDefined()
  })
})

describe('ResellerSetupWizard — step 2: manual entry form', () => {
  // Helper: advance to manual step.
  async function renderAtManual() {
    render(<ResellerSetupWizard onComplete={vi.fn()} />)
    await waitFor(() => screen.getByText('手动接入'))
    fireEvent.click(screen.getByText('手动接入'))
    await waitFor(() => screen.getByText('填写 Hub 连接信息'))
  }

  it('renders Hub URL and Admin Token inputs', async () => {
    await renderAtManual()
    expect(screen.getByPlaceholderText('https://hub.acme.example')).toBeDefined()
    expect(screen.getByPlaceholderText('••••••••')).toBeDefined()
  })

  it('"下一步" button is disabled until both Hub URL and Admin Token are filled', async () => {
    await renderAtManual()
    const nextBtn = screen.getByText('下一步').closest('button') as HTMLButtonElement
    expect(nextBtn.disabled).toBe(true)

    fireEvent.change(screen.getByPlaceholderText('https://hub.acme.example'), {
      target: { value: 'https://hub.example' },
    })
    // Still disabled — token is still empty.
    expect(nextBtn.disabled).toBe(true)

    fireEvent.change(screen.getByPlaceholderText('••••••••'), {
      target: { value: 'tok-secret' },
    })
    expect(nextBtn.disabled).toBe(false)
  })

  it('"上一步" navigates back to the pick step', async () => {
    await renderAtManual()
    fireEvent.click(screen.getByText('上一步').closest('button') as HTMLButtonElement)
    await waitFor(() => {
      expect(screen.getByText('选择部署方式')).toBeDefined()
    })
  })
})

describe('ResellerSetupWizard — step 3: connection test', () => {
  // Helper: advance through pick → manual (filled) → test step.
  async function renderAtTest() {
    render(<ResellerSetupWizard onComplete={vi.fn()} />)
    await waitFor(() => screen.getByText('手动接入'))
    fireEvent.click(screen.getByText('手动接入'))
    await waitFor(() => screen.getByText('填写 Hub 连接信息'))

    fireEvent.change(screen.getByPlaceholderText('https://hub.acme.example'), {
      target: { value: 'https://hub.example' },
    })
    fireEvent.change(screen.getByPlaceholderText('••••••••'), {
      target: { value: 'tok-secret' },
    })
    // Advance to test step.
    fireEvent.click(screen.getByText('下一步').closest('button') as HTMLButtonElement)
    await waitFor(() => screen.getByText('验证连接并保存'))
  }

  it('shows summary of entered URL and masked token in test step', async () => {
    await renderAtTest()
    expect(screen.getByText('https://hub.example')).toBeDefined()
    // Token is masked — at most 8 bullets.
    expect(screen.getByText('••••••••')).toBeDefined()
  })

  it('"保存配置" is disabled before a successful connection test', async () => {
    await renderAtTest()
    const saveBtn = screen.getByText('保存配置').closest('button') as HTMLButtonElement
    expect(saveBtn.disabled).toBe(true)
  })

  it('successful connection test enables "保存配置"', async () => {
    mockTestHubConnection.mockResolvedValueOnce('连接成功 — version 1.2.3')
    await renderAtTest()

    fireEvent.click(screen.getByText('测试连接').closest('button') as HTMLButtonElement)
    await waitFor(() => {
      expect(screen.getByText('连接成功 — version 1.2.3')).toBeDefined()
    })
    const saveBtn = screen.getByText('保存配置').closest('button') as HTMLButtonElement
    expect(saveBtn.disabled).toBe(false)
  })

  it('failed connection test surfaces error message and keeps save disabled', async () => {
    mockTestHubConnection.mockRejectedValueOnce(new Error('HTTP 502 Bad Gateway'))
    await renderAtTest()

    fireEvent.click(screen.getByText('测试连接').closest('button') as HTMLButtonElement)
    await waitFor(() => {
      expect(screen.getByText(/HTTP 502 Bad Gateway/)).toBeDefined()
    })
    const saveBtn = screen.getByText('保存配置').closest('button') as HTMLButtonElement
    expect(saveBtn.disabled).toBe(true)
  })

  it('provision failure shows error banner and does not advance to done step', async () => {
    // First test succeeds so save is enabled.
    mockTestHubConnection.mockResolvedValueOnce('ok')
    // Provision rejects — error must surface in the UI.
    mockProvisionResellerHub.mockRejectedValueOnce(new Error('DB write failed'))
    await renderAtTest()

    fireEvent.click(screen.getByText('测试连接').closest('button') as HTMLButtonElement)
    await waitFor(() => screen.getByText('ok'))

    fireEvent.click(screen.getByText('保存配置').closest('button') as HTMLButtonElement)
    await waitFor(() => {
      expect(screen.getByText(/DB write failed/)).toBeDefined()
    })
    // Must still be on the test step, not the done step.
    expect(screen.queryByText('Hub 已就绪')).toBeNull()
  })

  it('"上一步" navigates back to manual entry from test step', async () => {
    await renderAtTest()
    const backBtns = screen.getAllByText('上一步')
    fireEvent.click(backBtns[backBtns.length - 1].closest('button') as HTMLButtonElement)
    await waitFor(() => {
      expect(screen.getByText('填写 Hub 连接信息')).toBeDefined()
    })
  })
})

describe('ResellerSetupWizard — step 4: done', () => {
  async function renderAtDone(onComplete = vi.fn()) {
    render(<ResellerSetupWizard onComplete={onComplete} />)
    await waitFor(() => screen.getByText('手动接入'))
    fireEvent.click(screen.getByText('手动接入'))
    await waitFor(() => screen.getByText('填写 Hub 连接信息'))

    fireEvent.change(screen.getByPlaceholderText('https://hub.acme.example'), {
      target: { value: 'https://hub.example' },
    })
    fireEvent.change(screen.getByPlaceholderText('••••••••'), {
      target: { value: 'tok-secret' },
    })
    fireEvent.click(screen.getByText('下一步').closest('button') as HTMLButtonElement)
    await waitFor(() => screen.getByText('验证连接并保存'))

    mockTestHubConnection.mockResolvedValueOnce('ok')
    mockProvisionResellerHub.mockResolvedValueOnce(undefined)

    fireEvent.click(screen.getByText('测试连接').closest('button') as HTMLButtonElement)
    await waitFor(() => screen.getByText('ok'))

    fireEvent.click(screen.getByText('保存配置').closest('button') as HTMLButtonElement)
    await waitFor(() => screen.getByText('Hub 已就绪'))

    return onComplete
  }

  it('shows done-step success content after a successful provision', async () => {
    await renderAtDone()
    expect(screen.getByText('Hub 已就绪')).toBeDefined()
    expect(
      screen.getByText('你现在可以在「Gateway 管理」页配置 channel、生成激活码、查看日志。'),
    ).toBeDefined()
  })

  it('"进入 Reseller 控制台" button calls onComplete', async () => {
    const onComplete = await renderAtDone()
    fireEvent.click(screen.getByText('进入 Reseller 控制台'))
    expect(onComplete).toHaveBeenCalledTimes(1)
  })
})

describe('ResellerSetupWizard — switch-mode escape hatch', () => {
  it('does NOT switch mode when the user cancels the confirm dialog', async () => {
    vi.spyOn(window, 'confirm').mockReturnValueOnce(false)
    render(<ResellerSetupWizard onComplete={vi.fn()} />)
    await waitFor(() => screen.getByText('切换模式'))
    fireEvent.click(screen.getByText('切换模式').closest('button') as HTMLButtonElement)
    expect(mockSetAppMode).not.toHaveBeenCalled()
    expect(mockSetAppModeLocal).not.toHaveBeenCalled()
  })

  it('calls SetAppMode("personal") and updates configStore when user confirms', async () => {
    vi.spyOn(window, 'confirm').mockReturnValueOnce(true)
    mockSetAppMode.mockResolvedValueOnce(undefined)
    render(<ResellerSetupWizard onComplete={vi.fn()} />)
    await waitFor(() => screen.getByText('切换模式'))
    fireEvent.click(screen.getByText('切换模式').closest('button') as HTMLButtonElement)
    await waitFor(() => {
      expect(mockSetAppMode).toHaveBeenCalledWith('personal')
      expect(mockSetAppModeLocal).toHaveBeenCalledWith('personal')
    })
  })

  it('shows alert and does not crash when SetAppMode rejects', async () => {
    vi.spyOn(window, 'confirm').mockReturnValueOnce(true)
    vi.spyOn(window, 'alert').mockImplementationOnce(() => {})
    mockSetAppMode.mockRejectedValueOnce(new Error('rpc failed'))
    render(<ResellerSetupWizard onComplete={vi.fn()} />)
    await waitFor(() => screen.getByText('切换模式'))
    fireEvent.click(screen.getByText('切换模式').closest('button') as HTMLButtonElement)
    await waitFor(() => {
      expect(window.alert).toHaveBeenCalledWith(expect.stringContaining('rpc failed'))
    })
    // Wizard still visible — not in a broken state.
    expect(screen.getByText('配置 Reseller Hub')).toBeDefined()
  })
})
