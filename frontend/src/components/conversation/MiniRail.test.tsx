import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, cleanup, fireEvent } from '@testing-library/react'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: string) => fallback ?? key,
    i18n: { language: 'en' },
  }),
}))

import { MiniRail } from './MiniRail'
import type { conversation } from '../../../wailsjs/go/models'

function mkEvent(p: Partial<conversation.Event>): conversation.Event {
  return {
    type: 'user',
    timestamp: '2026-05-26T10:00:00Z',
    content: 'msg',
    ...p,
  } as conversation.Event
}

describe('MiniRail', () => {
  afterEach(() => cleanup())

  it('renders one dot per event', () => {
    const events = [
      mkEvent({ type: 'user' }),
      mkEvent({ type: 'assistant' }),
      mkEvent({ type: 'tool_use', toolName: 'Read' }),
    ]
    render(<MiniRail events={events} activeIdx={-1} onJump={() => {}} />)
    expect(screen.getByTestId('mini-rail-dot-0')).toBeInTheDocument()
    expect(screen.getByTestId('mini-rail-dot-1')).toBeInTheDocument()
    expect(screen.getByTestId('mini-rail-dot-2')).toBeInTheDocument()
  })

  it('calls onJump with the clicked index', () => {
    const events = [mkEvent({}), mkEvent({}), mkEvent({})]
    const onJump = vi.fn()
    render(<MiniRail events={events} activeIdx={0} onJump={onJump} />)
    fireEvent.click(screen.getByTestId('mini-rail-dot-2'))
    expect(onJump).toHaveBeenCalledWith(2)
  })

  it('marks the active dot with data-active', () => {
    const events = [mkEvent({}), mkEvent({}), mkEvent({})]
    render(<MiniRail events={events} activeIdx={1} onJump={() => {}} />)
    expect(screen.getByTestId('mini-rail-dot-1').getAttribute('data-active')).toBe('true')
    expect(screen.getByTestId('mini-rail-dot-0').getAttribute('data-active')).toBeNull()
  })
})
