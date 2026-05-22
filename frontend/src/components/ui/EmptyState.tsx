import { type ReactNode } from 'react'
import { type LucideIcon } from 'lucide-react'
import { cn } from '../../lib/utils'

interface EmptyStateProps {
  icon: LucideIcon
  title: string
  hint?: string
  action?: ReactNode
  className?: string
}

export function EmptyState({ icon: Icon, title, hint, action, className }: EmptyStateProps) {
  return (
    <div className={cn('flex flex-col items-center justify-center py-12 px-6 text-center', className)}>
      <Icon className="h-10 w-10 text-muted-foreground/60 mb-3" strokeWidth={1.5} />
      <h3 className="text-sm font-medium text-foreground">{title}</h3>
      {hint && (
        <p className="text-xs text-muted-foreground mt-1 max-w-xs leading-relaxed">{hint}</p>
      )}
      {action && <div className="mt-4">{action}</div>}
    </div>
  )
}
