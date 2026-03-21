import { Component, type ErrorInfo, type ReactNode } from 'react'

interface Props {
  children: ReactNode
  fallback?: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('[ErrorBoundary]', error, info.componentStack)
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null })
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback
      }

      const msg = this.state.error?.message || ''
      // Truncate long error messages for display.
      const display = msg.length > 150 ? msg.slice(0, 150) + '…' : msg

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
            <div className="flex gap-3 justify-center">
              <button
                onClick={this.handleRetry}
                className="px-4 py-2 text-sm font-medium bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
              >
                Retry / 重试
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
