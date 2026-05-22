import { type ReactNode } from 'react'
import { cn } from '../../lib/utils'

interface ActionRailProps {
  left?: ReactNode
  right?: ReactNode
  sticky?: boolean
  className?: string
}

export function ActionRail({ left, right, sticky = true, className }: ActionRailProps) {
  return (
    <div
      className={cn(
        'flex items-center justify-between gap-2 pt-3 mt-3 border-t border-border',
        sticky && 'sticky bottom-0 -mx-6 px-6 pb-3 bg-background/95 backdrop-blur',
        className,
      )}
    >
      <div className="flex items-center gap-2">{left}</div>
      <div className="flex items-center gap-2">{right}</div>
    </div>
  )
}
