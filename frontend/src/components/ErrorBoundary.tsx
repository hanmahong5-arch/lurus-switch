import { Component, type ErrorInfo, type ReactNode } from 'react'

interface Props {
  children: ReactNode
  /** Optional fallback to render in place of the default UI. */
  fallback?: ReactNode
  /** Stable identifier sent to the backend audit log (e.g. "page:settings"). */
  name?: string
  /** Active page slug so support can correlate crashes to a route. */
  page?: string
}

interface State {
  hasError: boolean
  error: Error | null
  componentStack: string
  copyState: 'idle' | 'copied'
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, error: null, componentStack: '', copyState: 'idle' }
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    const stack = info.componentStack ?? ''
    this.setState({ componentStack: stack })
    console.error('[ErrorBoundary]', error, stack)

    // Persist to the Go-side audit journal so support can recover the
    // crash later without needing devtools open. Lazy-imported to keep
    // the test-suite mock surface minimal — failures here are
    // swallowed because they must not turn a fallback render into a
    // throw loop.
    void this.reportToBackend(error, stack)
  }

  private async reportToBackend(error: Error, componentStack: string) {
    try {
      const mod = await import('../../wailsjs/go/main/App')
      if (typeof mod.LogFrontendError !== 'function') return
      await mod.LogFrontendError(
        this.props.name ?? 'unknown',
        error.message ?? String(error),
        componentStack,
        this.props.page ?? '',
      )
    } catch {
      // Backend logging is best-effort; never re-throw from here.
    }
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null, componentStack: '', copyState: 'idle' })
  }

  handleCopy = async () => {
    const { error, componentStack } = this.state
    const payload = [
      `Boundary: ${this.props.name ?? 'unknown'}`,
      `Page: ${this.props.page ?? '-'}`,
      `Message: ${error?.message ?? ''}`,
      `Stack: ${error?.stack ?? ''}`,
      `Component stack: ${componentStack}`,
    ].join('\n')
    try {
      await navigator.clipboard.writeText(payload)
      this.setState({ copyState: 'copied' })
      setTimeout(() => this.setState({ copyState: 'idle' }), 1500)
    } catch {
      this.setState({ copyState: 'idle' })
    }
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback
      }

      const msg = this.state.error?.message || ''
      const display = msg.length > 150 ? msg.slice(0, 150) + '…' : msg
      const copyLabel = this.state.copyState === 'copied' ? 'Copied ✓' : 'Copy / 复制详情'

      return (
        <div className="h-full flex items-center justify-center p-8">
          <div className="max-w-md text-center space-y-4">
            <div className="text-4xl">:(</div>
            <h2 className="text-lg font-semibold text-foreground">
              Something went wrong / 出了些问题
            </h2>
            {display && (
              <p className="text-sm text-muted-foreground break-words">
                {display}
              </p>
            )}
            <div className="flex flex-wrap gap-3 justify-center">
              <button
                onClick={this.handleRetry}
                className="px-4 py-2 text-sm font-medium bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
              >
                Retry / 重试
              </button>
              <button
                onClick={this.handleCopy}
                className="px-4 py-2 text-sm font-medium border border-border text-foreground rounded-md hover:bg-muted"
              >
                {copyLabel}
              </button>
              <button
                onClick={() => window.location.reload()}
                className="px-4 py-2 text-sm font-medium border border-border text-foreground rounded-md hover:bg-muted"
              >
                Reload / 重载
              </button>
            </div>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
