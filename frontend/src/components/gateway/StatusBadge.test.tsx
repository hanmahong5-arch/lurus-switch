import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StatusBadge } from './StatusBadge'

describe('StatusBadge', () => {
  it('should render enabled status with green styling', () => {
    render(<StatusBadge status="enabled" />)
    const badge = screen.getByText('Enabled')
    expect(badge).toBeDefined()
    expect(badge.className).toContain('text-green-400')
  })

  it('should render disabled status with muted styling', () => {
    render(<StatusBadge status="disabled" />)
    const badge = screen.getByText('Disabled')
    expect(badge).toBeDefined()
    expect(badge.className).toContain('text-muted-foreground')
  })

  it('should render expired status with red styling', () => {
    render(<StatusBadge status="expired" />)
    const badge = screen.getByText('Expired')
    expect(badge).toBeDefined()
    expect(badge.className).toContain('text-red-400')
  })

  it('should render used status with blue styling', () => {
    render(<StatusBadge status="used" />)
    const badge = screen.getByText('Used')
    expect(badge.className).toContain('text-blue-400')
  })

  it('should use custom labels when provided', () => {
    render(<StatusBadge status="enabled" labels={{ enabled: 'Active' }} />)
    expect(screen.getByText('Active')).toBeDefined()
  })
})
