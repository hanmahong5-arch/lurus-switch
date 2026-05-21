import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallbackOrOpts?: string | Record<string, unknown>) =>
      typeof fallbackOrOpts === 'string' ? fallbackOrOpts : key,
    i18n: { language: 'zh', changeLanguage: vi.fn() },
  }),
}))

const SaveCustomProvider = vi.fn()
const TestCustomProvider = vi.fn()

vi.mock('../../wailsjs/go/main/App', () => ({
  SaveCustomProvider: (...args: unknown[]) => SaveCustomProvider(...args),
  TestCustomProvider: (...args: unknown[]) => TestCustomProvider(...args),
}))

import { CustomProviderForm } from './CustomProviderForm'

beforeEach(() => {
  SaveCustomProvider.mockReset()
  TestCustomProvider.mockReset()
})

describe('CustomProviderForm', () => {
  it('blocks save when Base URL is empty', async () => {
    const onSaved = vi.fn()
    render(<CustomProviderForm onSaved={onSaved} onCancel={() => {}} />)
    fireEvent.click(screen.getByText('保存'))
    await waitFor(() => {
      expect(screen.getByText('请填写 Base URL')).toBeTruthy()
    })
    expect(SaveCustomProvider).not.toHaveBeenCalled()
    expect(onSaved).not.toHaveBeenCalled()
  })

  it('calls SaveCustomProvider with parsed models and fires onSaved', async () => {
    const saved = { id: 'custom-x-1', name: 'X', baseUrl: 'https://x.test', apiKey: 'k', defaultModels: ['a', 'b'] }
    SaveCustomProvider.mockResolvedValue(saved)
    const onSaved = vi.fn()
    render(<CustomProviderForm onSaved={onSaved} onCancel={() => {}} />)

    fireEvent.change(screen.getByPlaceholderText('https://api.example.com/v1'), {
      target: { value: 'https://x.test' },
    })
    fireEvent.change(screen.getByPlaceholderText('gpt-4o, claude-3-5-sonnet'), {
      target: { value: 'a, b ,' },
    })
    fireEvent.click(screen.getByText('保存'))

    await waitFor(() => expect(onSaved).toHaveBeenCalledWith(saved))
    const arg = SaveCustomProvider.mock.calls[0][0]
    expect(arg.baseUrl).toBe('https://x.test')
    expect(arg.defaultModels).toEqual(['a', 'b'])
  })

  it('shows model count + latency on a successful test', async () => {
    TestCustomProvider.mockResolvedValue({ ok: true, models: ['m1', 'm2', 'm3'], latencyMs: 42 })
    render(<CustomProviderForm onSaved={() => {}} onCancel={() => {}} />)
    fireEvent.change(screen.getByPlaceholderText('https://api.example.com/v1'), {
      target: { value: 'https://x.test' },
    })
    fireEvent.click(screen.getByText('测试连接'))
    await waitFor(() => {
      expect(screen.getByText(/连接成功/)).toBeTruthy()
    })
  })

  it('surfaces a failed test', async () => {
    TestCustomProvider.mockResolvedValue({ ok: false, models: [], latencyMs: 5000, error: 'timeout' })
    render(<CustomProviderForm onSaved={() => {}} onCancel={() => {}} />)
    fireEvent.change(screen.getByPlaceholderText('https://api.example.com/v1'), {
      target: { value: 'https://x.test' },
    })
    fireEvent.click(screen.getByText('测试连接'))
    await waitFor(() => {
      expect(screen.getByText(/连接失败/)).toBeTruthy()
    })
  })
})
