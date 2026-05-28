import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: string) => fallback ?? key,
    i18n: { language: 'en' },
  }),
}))

import { SessionKpiStrip } from './SessionKpiStrip'
import type { conversation } from '../../../wailsjs/go/models'

function mkEvent(p: Partial<conversation.Event>): conversation.Event {
  return {
    type: 'user',
    timestamp: '2026-05-26T10:00:00Z',
    content: 'x',
    ...p,
  } as conversation.Event
}

describe('SessionKpiStrip', () => {
  afterEach(() => cleanup())

  it('counts messages and sums tokens', () => {
    const events = [
      mkEvent({ type: 'user', inputTokens: 100 }),
      mkEvent({ type: 'assistant', outputTokens: 250 }),
      mkEvent({ type: 'tool_use', inputTokens: 50 }),
    ]
    const { container } = render(<SessionKpiStrip events={events} />)
    expect(container.textContent).toContain('3') // message count
    expect(container.textContent).toContain('400') // token sum
  })

  it('shows +N more when multiple models are mixed', () => {
    const events = [
      mkEvent({ type: 'assistant', model: 'claude-opus-4-7' }),
      mkEvent({ type: 'assistant', model: 'claude-sonnet-4-6' }),
      mkEvent({ type: 'assistant', model: 'claude-haiku-4-5' }),
    ]
    const { container } = render(<SessionKpiStrip events={events} />)
    expect(container.textContent).toMatch(/\+2/)
  })

  it('does not crash on empty events', () => {
    expect(() => render(<SessionKpiStrip events={[]} />)).not.toThrow()
    expect(screen.getByText(/MESSAGES/i)).toBeInTheDocument()
  })
})
