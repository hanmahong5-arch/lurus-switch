import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { Pagination } from './Pagination'

describe('Pagination', () => {
  it('should not render when totalPages <= 1', () => {
    const { container } = render(
      <Pagination page={0} total={10} perPage={50} onPageChange={vi.fn()} />
    )
    expect(container.innerHTML).toBe('')
  })

  it('should render page buttons for multiple pages', () => {
    render(
      <Pagination page={0} total={150} perPage={50} onPageChange={vi.fn()} />
    )
    // Should show 3 pages: 1, 2, 3
    expect(screen.getByText('1')).toBeDefined()
    expect(screen.getByText('2')).toBeDefined()
    expect(screen.getByText('3')).toBeDefined()
  })

  it('should call onPageChange with previous page', () => {
    const onChange = vi.fn()
    render(
      <Pagination page={1} total={150} perPage={50} onPageChange={onChange} />
    )
    // Click the prev button (first button)
    const buttons = screen.getAllByRole('button')
    fireEvent.click(buttons[0])
    expect(onChange).toHaveBeenCalledWith(0)
  })

  it('should call onPageChange with next page', () => {
    const onChange = vi.fn()
    render(
      <Pagination page={0} total={150} perPage={50} onPageChange={onChange} />
    )
    // Click the next button (last button)
    const buttons = screen.getAllByRole('button')
    fireEvent.click(buttons[buttons.length - 1])
    expect(onChange).toHaveBeenCalledWith(1)
  })

  it('should display total items count', () => {
    render(
      <Pagination page={0} total={150} perPage={50} onPageChange={vi.fn()} />
    )
    expect(screen.getByText('150 items')).toBeDefined()
  })
})
