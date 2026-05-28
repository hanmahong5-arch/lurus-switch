import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, cleanup, fireEvent } from '@testing-library/react'

import { DLPHitBadge } from './DLPHitBadge'
import type { audit } from '../../../wailsjs/go/models'

function mkHit(id: string): audit.Entry {
  return { id, operation: 'dlp.redact', target: 'message' } as audit.Entry
}

describe('DLPHitBadge', () => {
  afterEach(() => cleanup())

  it('renders nothing when there are no hits', () => {
    const { container } = render(<DLPHitBadge hits={[]} />)
    expect(container.firstChild).toBeNull()
  })

  it('renders a clickable button that emits the first entry id when wired', () => {
    const onOpen = vi.fn()
    render(<DLPHitBadge hits={[mkHit('entry-1'), mkHit('entry-2')]} onOpenEntry={onOpen} />)
    const btn = screen.getByRole('button')
    expect(btn).toHaveTextContent('DLP × 2')
    fireEvent.click(btn)
    expect(onOpen).toHaveBeenCalledWith('entry-1')
  })

  it('renders a non-interactive indicator (no button) when no handler is wired', () => {
    render(<DLPHitBadge hits={[mkHit('entry-1')]} />)
    expect(screen.queryByRole('button')).toBeNull()
    expect(screen.getByText('DLP × 1')).toBeInTheDocument()
  })
})
