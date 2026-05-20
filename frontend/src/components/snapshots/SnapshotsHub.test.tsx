import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

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

const listMock = vi.fn()
const takeMock = vi.fn()
const restoreMock = vi.fn()
const deleteMock = vi.fn()

vi.mock('../../../wailsjs/go/main/App', () => ({
  ListConfigSnapshots: (tool: string) => listMock(tool),
  TakeConfigSnapshot: (tool: string, label: string) => takeMock(tool, label),
  RestoreConfigSnapshot: (tool: string, id: string) => restoreMock(tool, id),
  DeleteConfigSnapshot: (tool: string, id: string) => deleteMock(tool, id),
}))

import { SnapshotsHub } from './SnapshotsHub'
import { useSnapshotsHubStore } from '../../stores/snapshotsHubStore'
import { useToastStore } from '../../stores/toastStore'

beforeEach(() => {
  listMock.mockReset()
  takeMock.mockReset()
  restoreMock.mockReset()
  deleteMock.mockReset()
  useToastStore.setState({ toasts: [] })
  useSnapshotsHubStore.setState({ open: false, focusTool: null })
})

const lastToast = () => {
  const ts = useToastStore.getState().toasts
  return ts[ts.length - 1]
}

const baseSnap = (overrides: Record<string, unknown> = {}) => ({
  id: 'snap-1',
  tool: 'claude',
  label: 'before redirect',
  createdAt: '2026-05-20T10:00:00Z',
  size: 1234,
  ...overrides,
})

describe('SnapshotsHub', () => {
  it('is hidden when store.open is false', () => {
    listMock.mockResolvedValue([])
    render(<SnapshotsHub />)
    expect(screen.queryByTestId('snapshots-hub')).not.toBeInTheDocument()
  })

  it('loads snapshots for every tool when opened', async () => {
    listMock.mockResolvedValue([baseSnap()])
    useSnapshotsHubStore.setState({ open: true })
    render(<SnapshotsHub />)
    await waitFor(() => {
      // 7 tools in TOOL_ORDER
      expect(listMock).toHaveBeenCalledTimes(7)
    })
  })

  it('shows restore confirmation dialog (not immediate restore)', async () => {
    listMock.mockImplementation((tool) =>
      tool === 'claude' ? Promise.resolve([baseSnap()]) : Promise.resolve([]),
    )
    useSnapshotsHubStore.setState({ open: true, focusTool: 'claude' })
    render(<SnapshotsHub />)
    await screen.findByText('before redirect')
    fireEvent.click(screen.getByText('恢复'))
    expect(await screen.findByTestId('snapshots-restore-confirm')).toBeInTheDocument()
    expect(restoreMock).not.toHaveBeenCalled()
  })

  it('calls RestoreConfigSnapshot only after confirm click', async () => {
    listMock.mockImplementation((tool) =>
      tool === 'claude' ? Promise.resolve([baseSnap()]) : Promise.resolve([]),
    )
    restoreMock.mockResolvedValue(undefined)
    useSnapshotsHubStore.setState({ open: true, focusTool: 'claude' })
    render(<SnapshotsHub />)
    await screen.findByText('before redirect')
    fireEvent.click(screen.getByText('恢复'))
    await screen.findByTestId('snapshots-restore-confirm')
    fireEvent.click(screen.getByTestId('snapshots-restore-confirm-btn'))
    await waitFor(() => {
      expect(restoreMock).toHaveBeenCalledWith('claude', 'snap-1')
      const t = lastToast()
      expect(t?.type).toBe('success')
    })
  })

  it('takes a new snapshot when label saved', async () => {
    listMock.mockResolvedValue([])
    takeMock.mockResolvedValue(undefined)
    useSnapshotsHubStore.setState({ open: true, focusTool: 'claude' })
    render(<SnapshotsHub />)
    await waitFor(() => expect(listMock).toHaveBeenCalled())
    fireEvent.click(screen.getAllByText(/新建快照/)[0])
    const input = await screen.findByPlaceholderText(/快照标签/)
    fireEvent.change(input, { target: { value: 'pre-codex-switch' } })
    fireEvent.click(screen.getByText('保存'))
    await waitFor(() => {
      expect(takeMock).toHaveBeenCalledWith('claude', 'pre-codex-switch')
      const t = lastToast()
      expect(t?.type).toBe('success')
    })
  })

  it('surfaces error toast when restore fails', async () => {
    listMock.mockImplementation((tool) =>
      tool === 'claude' ? Promise.resolve([baseSnap()]) : Promise.resolve([]),
    )
    restoreMock.mockRejectedValue(new Error('config locked'))
    useSnapshotsHubStore.setState({ open: true, focusTool: 'claude' })
    render(<SnapshotsHub />)
    await screen.findByText('before redirect')
    fireEvent.click(screen.getByText('恢复'))
    await screen.findByTestId('snapshots-restore-confirm')
    fireEvent.click(screen.getByTestId('snapshots-restore-confirm-btn'))
    await waitFor(() => {
      const t = lastToast()
      expect(t?.type).toBe('error')
      expect(t?.message).toMatch(/config locked/)
    })
  })
})
