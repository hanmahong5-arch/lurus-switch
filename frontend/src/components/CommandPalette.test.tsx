import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, fireEvent, screen, waitFor } from '@testing-library/react'

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

const installToolMock = vi.fn()
const installAllToolsMock = vi.fn()
const setAppModeMock = vi.fn()
const isModeLockedMock = vi.fn()
const takeSnapshotMock = vi.fn()

vi.mock('../../wailsjs/go/main/App', () => ({
  InstallTool: (n: string) => installToolMock(n),
  InstallAllTools: () => installAllToolsMock(),
  StartGateway: vi.fn(),
  StopGateway: vi.fn(),
  AutoConfigureToolsForGateway: vi.fn().mockResolvedValue([]),
  ApplyAllOptimizations: vi.fn(),
  FullSetupForGateway: vi.fn(),
  DetectAllTools: vi.fn().mockResolvedValue({}),
  CheckAllToolHealth: vi.fn().mockResolvedValue({}),
  LaunchTool: vi.fn(),
  TakeConfigSnapshot: (n: string, r: string) => takeSnapshotMock(n, r),
  SetAppMode: (m: string) => setAppModeMock(m),
  IsModeLocked: () => isModeLockedMock(),
}))

import { CommandPalette } from './CommandPalette'
import { useCommandPaletteStore } from '../stores/commandPaletteStore'
import { useToastStore } from '../stores/toastStore'

beforeEach(() => {
  installToolMock.mockReset()
  installAllToolsMock.mockReset()
  setAppModeMock.mockReset()
  isModeLockedMock.mockReset()
  takeSnapshotMock.mockReset()
  useCommandPaletteStore.setState({ open: true })
  useToastStore.setState({ toasts: [] })
})

const lastToast = () => {
  const ts = useToastStore.getState().toasts
  return ts[ts.length - 1]
}

const clickByText = async (regex: RegExp) => {
  const matches = await screen.findAllByText(regex)
  fireEvent.click(matches[0])
}

describe('CommandPalette install action — Result.success guard', () => {
  it('shows error toast when InstallTool returns success=false', async () => {
    installToolMock.mockResolvedValue({ tool: 'claude', success: false, version: '', message: 'network blocked' })
    render(<CommandPalette />)
    await clickByText(/installClaudeCode/)
    await waitFor(() => {
      expect(installToolMock).toHaveBeenCalledWith('claude')
      const t = lastToast()
      expect(t?.type).toBe('error')
      expect(t?.message).toBe('network blocked')
    })
  })

  it('shows success toast when InstallTool returns success=true', async () => {
    installToolMock.mockResolvedValue({ tool: 'claude', success: true, version: '1.0.0', message: '' })
    render(<CommandPalette />)
    await clickByText(/installClaudeCode/)
    await waitFor(() => {
      const t = lastToast()
      expect(t?.type).toBe('success')
    })
  })

  it('shows aggregated error toast when InstallAllTools has failures', async () => {
    installAllToolsMock.mockResolvedValue([
      { tool: 'claude', success: true, version: '1', message: '' },
      { tool: 'codex', success: false, version: '', message: 'permission denied' },
      { tool: 'gemini', success: false, version: '', message: 'timeout' },
    ])
    render(<CommandPalette />)
    await clickByText(/installAll/)
    await waitFor(() => {
      const t = lastToast()
      expect(t?.type).toBe('error')
      expect(t?.message).toMatch(/2/)
      expect(t?.message).toMatch(/permission denied/)
    })
  })
})

describe('CommandPalette mode switching', () => {
  it('warns when mode is locked', async () => {
    isModeLockedMock.mockResolvedValue(true)
    render(<CommandPalette />)
    await clickByText(/modePersonal/)
    await waitFor(() => {
      const t = lastToast()
      expect(t?.type).toBe('warning')
    })
    expect(setAppModeMock).not.toHaveBeenCalled()
  })

  it('calls SetAppMode when unlocked', async () => {
    isModeLockedMock.mockResolvedValue(false)
    setAppModeMock.mockResolvedValue(undefined)
    render(<CommandPalette />)
    await clickByText(/modeReseller/)
    await waitFor(() => {
      expect(setAppModeMock).toHaveBeenCalledWith('reseller')
      const t = lastToast()
      expect(t?.type).toBe('success')
    })
  })
})

describe('CommandPalette snapshot action', () => {
  it('calls TakeConfigSnapshot with a timestamped name', async () => {
    takeSnapshotMock.mockResolvedValue(undefined)
    render(<CommandPalette />)
    await clickByText(/takeSnapshot/)
    await waitFor(() => {
      expect(takeSnapshotMock).toHaveBeenCalled()
      const [name] = takeSnapshotMock.mock.calls[0]
      expect(name).toMatch(/^palette-/)
      const t = lastToast()
      expect(t?.type).toBe('success')
    })
  })
})
