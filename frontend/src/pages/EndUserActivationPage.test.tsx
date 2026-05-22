import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ErrorActions } from './EndUserActivationPage'

// Minimal i18next stub — return the fallback string and substitute
// {{vars}} so the rendered output matches what the user actually sees.
// Mocking the module is cleaner than wrapping every render in an
// I18nextProvider just for this test.
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (_key: string, fallbackOrOpts?: string | Record<string, unknown>, opts?: Record<string, unknown>) => {
      const fallback = typeof fallbackOrOpts === 'string' ? fallbackOrOpts : _key
      const vars = typeof fallbackOrOpts === 'object' ? fallbackOrOpts : opts
      if (vars && typeof vars === 'object') {
        return Object.entries(vars).reduce<string>(
          (s, [k, v]) => s.replace(new RegExp(`{{\\s*${k}\\s*}}`, 'g'), String(v)),
          fallback,
        )
      }
      return fallback
    },
  }),
}))

const baseProps = {
  supportEmail: 'support@reseller.example',
  code: 'ABCD-1234',
  hubURL: 'https://hub.example',
  onRetry: vi.fn(),
  onClear: vi.fn(),
}

describe('ErrorActions — kind → CTA mapping', () => {
  beforeEach(() => {
    baseProps.onRetry = vi.fn()
    baseProps.onClear = vi.fn()
  })

  // Per-kind expectations live in one place so it's obvious when the
  // mapping changes — each row is a self-contained spec.
  type Spec = {
    kind: string
    retry: boolean
    clear: boolean
    contact: boolean
  }
  const specs: Spec[] = [
    { kind: 'network',          retry: true,  clear: false, contact: false },
    { kind: 'server',           retry: true,  clear: false, contact: false },
    { kind: 'endpoint_absent',  retry: true,  clear: false, contact: false },
    { kind: 'invalid_input',    retry: false, clear: true,  contact: false },
    { kind: 'code_not_found',   retry: false, clear: true,  contact: true },
    { kind: 'code_used',        retry: false, clear: false, contact: true },
    { kind: 'code_expired',     retry: false, clear: false, contact: true },
    { kind: 'code_disabled',    retry: false, clear: false, contact: true },
  ]

  for (const s of specs) {
    it(`kind=${s.kind} shows retry=${s.retry} clear=${s.clear} contact=${s.contact}`, () => {
      render(<ErrorActions {...baseProps} kind={s.kind} />)
      expect(!!screen.queryByText('重试')).toBe(s.retry)
      expect(!!screen.queryByText('清空重输')).toBe(s.clear)
      expect(!!screen.queryByText('联系经销商')).toBe(s.contact)
    })
  }

  it('returns null (renders nothing) for unknown kind', () => {
    const { container } = render(<ErrorActions {...baseProps} kind="totally_unknown_kind" />)
    expect(container.innerHTML).toBe('')
  })

  it('retry button calls onRetry', () => {
    const onRetry = vi.fn()
    render(<ErrorActions {...baseProps} kind="network" onRetry={onRetry} />)
    fireEvent.click(screen.getByText('重试'))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it('clear button calls onClear', () => {
    const onClear = vi.fn()
    render(<ErrorActions {...baseProps} kind="invalid_input" onClear={onClear} />)
    fireEvent.click(screen.getByText('清空重输'))
    expect(onClear).toHaveBeenCalledTimes(1)
  })
})

describe('ErrorActions — mailto composition', () => {
  it('composes mailto with code, hub, and kind in body', () => {
    render(<ErrorActions {...baseProps} kind="code_used" />)
    const link = screen.getByText('联系经销商').closest('a') as HTMLAnchorElement
    expect(link).not.toBeNull()
    expect(link.href).toMatch(/^mailto:support@reseller\.example/)
    // Decode the body so assertions read against plaintext.
    const body = decodeURIComponent(link.href.split('body=')[1] ?? '')
    expect(body).toContain('ABCD-1234')
    expect(body).toContain('hub.example')
    expect(body).toContain('code_used')
  })

  it('subject is i18n-driven, URL-encoded', () => {
    render(<ErrorActions {...baseProps} kind="code_expired" />)
    const link = screen.getByText('联系经销商').closest('a') as HTMLAnchorElement
    const subject = decodeURIComponent(link.href.match(/subject=([^&]+)/)?.[1] ?? '')
    expect(subject).toBe('激活码无法使用')
  })

  it('falls back to a plain message when supportEmail is empty', () => {
    render(<ErrorActions {...baseProps} kind="code_used" supportEmail="" />)
    // No anchor — the contact slot degrades to a static hint instead.
    expect(screen.queryByText('联系经销商')).toBeNull()
    expect(screen.getByText('请联系您拿到这份安装包的经销商')).toBeDefined()
  })

  it('handles undefined hubURL without producing the literal "undefined"', () => {
    render(<ErrorActions {...baseProps} kind="code_used" hubURL={undefined} />)
    const link = screen.getByText('联系经销商').closest('a') as HTMLAnchorElement
    const body = decodeURIComponent(link.href.split('body=')[1] ?? '')
    // Hub line should be empty after the colon, not the literal word
    // "undefined" — that's the kind of leak users actually email about.
    expect(body).not.toContain('undefined')
  })
})
