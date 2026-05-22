import { type ReactNode } from 'react'
import { cn } from '../../lib/utils'

export type TerminalChipTone = 'default' | 'ok' | 'warn' | 'err' | 'info'

interface TerminalChipProps {
  tone?: TerminalChipTone
  icon?: string
  className?: string
  children: ReactNode
}

const TONE: Record<TerminalChipTone, string> = {
  default: 'text-muted-foreground',
  ok:      'text-emerald-400',
  warn:    'text-amber-400',
  err:     'text-red-400',
  info:    'text-blue-400',
}

export function TerminalChip({ tone = 'default', icon = '▸', className, children }: TerminalChipProps) {
  return (
    <span className={cn('inline-flex items-center gap-1 font-mono text-[11px]', TONE[tone], className)}>
      <span aria-hidden>{icon}</span>
      <span>{children}</span>
    </span>
  )
}
