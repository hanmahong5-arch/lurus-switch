import { type ReactNode } from 'react'
import { cn } from '../../lib/utils'

interface PageShellProps {
  children: ReactNode
  className?: string
}

export function PageShell({ children, className }: PageShellProps) {
  return (
    <div className={cn('h-full overflow-y-auto px-6 py-5 space-y-5', className)}>
      {children}
    </div>
  )
}
