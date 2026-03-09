import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SearchBar } from './SearchBar'

describe('SearchBar', () => {
  it('should render with placeholder', () => {
    render(
      <SearchBar value="" onChange={vi.fn()} onSearch={vi.fn()} placeholder="Search here" />
    )
    expect(screen.getByPlaceholderText('Search here')).toBeDefined()
  })

  it('should call onChange when typing', () => {
    const onChange = vi.fn()
    render(
      <SearchBar value="" onChange={onChange} onSearch={vi.fn()} />
    )
    const input = screen.getByRole('textbox')
    fireEvent.change(input, { target: { value: 'test' } })
    expect(onChange).toHaveBeenCalledWith('test')
  })

  it('should trigger onSearch when Enter is pressed', () => {
    const onSearch = vi.fn()
    render(
      <SearchBar value="query" onChange={vi.fn()} onSearch={onSearch} />
    )
    const input = screen.getByRole('textbox')
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onSearch).toHaveBeenCalledTimes(1)
  })

  it('should trigger onSearch when button is clicked', () => {
    const onSearch = vi.fn()
    render(
      <SearchBar value="query" onChange={vi.fn()} onSearch={onSearch} />
    )
    fireEvent.click(screen.getByText('Search'))
    expect(onSearch).toHaveBeenCalledTimes(1)
  })

  it('should render children', () => {
    render(
      <SearchBar value="" onChange={vi.fn()} onSearch={vi.fn()}>
        <span data-testid="child">Extra</span>
      </SearchBar>
    )
    expect(screen.getByTestId('child')).toBeDefined()
  })
})
