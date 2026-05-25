import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, cleanup, fireEvent, waitFor } from '@testing-library/react'

const logFrontendErrorMock = vi.fn()
vi.mock('../../wailsjs/go/main/App', () => ({
  LogFrontendError: (...args: unknown[]) => logFrontendErrorMock(...args),
}))

import { ErrorBoundary } from './ErrorBoundary'

function Boom({ throwError = true }: { throwError?: boolean }) {
  if (throwError) {
    throw new Error('kaboom render')
  }
  return <div data-testid="ok-child">ok</div>
}

describe('ErrorBoundary', () => {
  // Silence the noisy React error log so failed-render tests stay readable.
  let consoleErrSpy: ReturnType<typeof vi.spyOn>
  beforeEach(() => {
    consoleErrSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    logFrontendErrorMock.mockReset()
    logFrontendErrorMock.mockResolvedValue(undefined)
  })
  afterEach(() => {
    cleanup()
    consoleErrSpy.mockRestore()
  })

  it('renders children when no error', () => {
    render(
      <ErrorBoundary>
        <Boom throwError={false} />
      </ErrorBoundary>,
    )
    expect(screen.getByTestId('ok-child')).toBeInTheDocument()
  })

  it('catches a thrown error and renders the fallback with Retry / Copy / Reload', () => {
    render(
      <ErrorBoundary name="page:test" page="test">
        <Boom />
      </ErrorBoundary>,
    )
    expect(screen.getByText(/Something went wrong/i)).toBeInTheDocument()
    expect(screen.getByText(/kaboom render/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Retry/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Copy/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Reload/i })).toBeInTheDocument()
  })

  it('calls LogFrontendError with boundary + page metadata', async () => {
    render(
      <ErrorBoundary name="page:settings" page="settings">
        <Boom />
      </ErrorBoundary>,
    )
    await waitFor(() => {
      expect(logFrontendErrorMock).toHaveBeenCalledTimes(1)
    })
    const [boundary, message, , page] = logFrontendErrorMock.mock.calls[0]
    expect(boundary).toBe('page:settings')
    expect(message).toBe('kaboom render')
    expect(page).toBe('settings')
  })

  it('Retry resets state and re-renders children when the throw stops', () => {
    let shouldThrow = true
    function Toggler() {
      if (shouldThrow) throw new Error('first time')
      return <div data-testid="recovered">recovered</div>
    }

    render(
      <ErrorBoundary>
        <Toggler />
      </ErrorBoundary>,
    )
    expect(screen.getByText(/first time/)).toBeInTheDocument()

    shouldThrow = false
    fireEvent.click(screen.getByRole('button', { name: /Retry/i }))
    expect(screen.getByTestId('recovered')).toBeInTheDocument()
  })
})
