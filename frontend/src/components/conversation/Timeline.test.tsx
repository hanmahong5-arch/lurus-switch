import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, cleanup, fireEvent } from '@testing-library/react'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: string) => fallback ?? key,
    i18n: { language: 'en' },
  }),
}))

import { Timeline } from './Timeline'
import type { conversation, audit } from '../../../wailsjs/go/models'

function mkEvent(p: Partial<conversation.Event>): conversation.Event {
  return {
    type: 'user',
    messageUUID: 'u-' + Math.random().toString(36).slice(2, 8),
    timestamp: '2026-05-26T10:00:00Z',
    content: 'hi',
    ...p,
  } as conversation.Event
}

describe('Timeline → MessageCard', () => {
  afterEach(() => cleanup())

  it('renders one MessageCard per event with role attribute', () => {
    const events = [
      mkEvent({ type: 'user', content: 'hello' }),
      mkEvent({ type: 'assistant', content: 'hi back' }),
      mkEvent({ type: 'tool_use', toolName: 'Read', content: 'reading' }),
      mkEvent({ type: 'tool_result', content: 'file contents' }),
      mkEvent({ type: 'system', content: 'sys note' }),
    ]
    render(<Timeline events={events} dlpHits={[]} />)
    const cards = screen.getAllByTestId('message-card')
    expect(cards.length).toBe(5)
    expect(cards[0].getAttribute('data-role')).toBe('user')
    expect(cards[1].getAttribute('data-role')).toBe('assistant')
    expect(cards[2].getAttribute('data-role')).toBe('tool_use')
    expect(cards[3].getAttribute('data-role')).toBe('tool_result')
    expect(cards[4].getAttribute('data-role')).toBe('system')
  })

  it('applies the role color class to the left border', () => {
    const events = [
      mkEvent({ type: 'user' }),
      mkEvent({ type: 'assistant' }),
      mkEvent({ type: 'tool_use', toolName: 'Bash' }),
      mkEvent({ type: 'tool_result' }),
    ]
    render(<Timeline events={events} dlpHits={[]} />)
    const cards = screen.getAllByTestId('message-card')
    expect(cards[0].className).toContain('border-l-blue-500')
    expect(cards[1].className).toContain('border-l-amber-500')
    expect(cards[2].className).toContain('border-l-purple-500')
    expect(cards[3].className).toContain('border-l-emerald-500')
  })

  it('collapses tool rows by default and expands on click', () => {
    const events = [mkEvent({ type: 'tool_use', toolName: 'Read', content: 'secret-body-text' })]
    render(<Timeline events={events} dlpHits={[]} />)
    expect(screen.queryByText('secret-body-text')).toBeNull()
    const btn = screen.getByLabelText('Expand')
    fireEvent.click(btn)
    expect(screen.getByText('secret-body-text')).toBeInTheDocument()
  })

  it('renders markdown bold as <strong>', () => {
    const events = [mkEvent({ type: 'assistant', content: 'this is **important** stuff' })]
    render(<Timeline events={events} dlpHits={[]} />)
    const strong = screen.getByText('important')
    expect(strong.tagName).toBe('STRONG')
  })

  it('renders fenced code block with hljs wrapper and copy button', () => {
    const md = ['```js', 'const x = 1', '```'].join('\n')
    const events = [mkEvent({ type: 'assistant', content: md })]
    render(<Timeline events={events} dlpHits={[]} />)
    const code = document.querySelector('code.hljs')
    expect(code).not.toBeNull()
    expect(screen.getByTitle('Copy code')).toBeInTheDocument()
  })

  it('shows the DLP badge when hits match the message uuid', () => {
    const events = [mkEvent({ type: 'user', messageUUID: 'm-1', content: 'leaked' })]
    const hits: audit.Entry[] = [
      { id: 'a-1', metadata: { conv_message_uuid: 'm-1' } } as unknown as audit.Entry,
    ]
    render(<Timeline events={events} dlpHits={hits} />)
    expect(screen.getByText(/DLP × 1/)).toBeInTheDocument()
  })

  it('renders empty-state when events is empty', () => {
    render(<Timeline events={[]} dlpHits={[]} />)
    expect(screen.getByText(/No messages/)).toBeInTheDocument()
  })
})
