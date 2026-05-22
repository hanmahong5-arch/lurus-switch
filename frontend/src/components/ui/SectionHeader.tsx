import { type ReactNode } from 'react'
import { cn } from '../../lib/utils'

interface SectionHeaderProps {
  title: string
  action?: ReactNode
  mono?: boolean
  className?: string
}

export function SectionHeader({ title, action, mono = true, className }: SectionHeaderProps) {
  return (
    <div className={cn('flex items-center justify-between', className)}>
      <h3
        className={cn(
          mono
            ? 'font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground'
            : 'text-xs font-semibold uppercase tracking-wider text-muted-foreground',
        )}
      >
        {mono ? `[ ${title.toUpperCase()} ]` : title}
      </h3>
      {action && <div className="flex items-center gap-1.5">{action}</div>}
    </div>
  )
}
