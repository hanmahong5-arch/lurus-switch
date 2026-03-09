import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ConfirmModal } from './ConfirmModal'

describe('ConfirmModal', () => {
  it('should not render when open is false', () => {
    const { container } = render(
      <ConfirmModal open={false} title="Delete?" desc="Are you sure?" onConfirm={vi.fn()} onCancel={vi.fn()} />
    )
    expect(container.innerHTML).toBe('')
  })

  it('should render title and description when open', () => {
    render(
      <ConfirmModal open={true} title="Delete Item" desc="This cannot be undone." onConfirm={vi.fn()} onCancel={vi.fn()} />
    )
    expect(screen.getByText('Delete Item')).toBeDefined()
    expect(screen.getByText('This cannot be undone.')).toBeDefined()
  })

  it('should call onConfirm when confirm button clicked', () => {
    const onConfirm = vi.fn()
    render(
      <ConfirmModal open={true} title="Test" desc="Desc" onConfirm={onConfirm} onCancel={vi.fn()} />
    )
    fireEvent.click(screen.getByText('Confirm'))
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it('should call onCancel when cancel button clicked', () => {
    const onCancel = vi.fn()
    render(
      <ConfirmModal open={true} title="Test" desc="Desc" onConfirm={vi.fn()} onCancel={onCancel} />
    )
    fireEvent.click(screen.getByText('Cancel'))
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('should apply danger style when danger prop is true', () => {
    render(
      <ConfirmModal open={true} title="Test" desc="Desc" danger onConfirm={vi.fn()} onCancel={vi.fn()} />
    )
    const confirmBtn = screen.getByText('Confirm')
    expect(confirmBtn.className).toContain('bg-red-600')
  })

  it('should apply normal style when danger prop is false', () => {
    render(
      <ConfirmModal open={true} title="Test" desc="Desc" onConfirm={vi.fn()} onCancel={vi.fn()} />
    )
    const confirmBtn = screen.getByText('Confirm')
    expect(confirmBtn.className).toContain('bg-indigo-600')
  })
})
