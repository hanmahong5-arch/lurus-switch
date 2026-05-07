import { AlertTriangle, ShieldCheck } from 'lucide-react'
import { cn } from '../../lib/utils'

interface SecurityAdvisoryProps {
  level: 'risky' | 'safe'
  zh?: string
  en?: string
  className?: string
}

// Bordered callout block shown directly below the field input when the
// current value has a security verdict. Two flavors: red for 'risky',
// emerald for 'safe' (e.g. sandbox enabled).
export function SecurityAdvisory({ level, zh, en, className }: SecurityAdvisoryProps) {
  if (!zh && !en) return null
  const isRisky = level === 'risky'
  const Icon = isRisky ? AlertTriangle : ShieldCheck
  return (
    <div
      className={cn(
        'rounded-md border px-2.5 py-2 text-[11px] leading-relaxed flex gap-2 mt-1.5',
        isRisky
          ? 'border-red-500/30 bg-red-950/20 text-red-200'
          : 'border-emerald-500/30 bg-emerald-950/20 text-emerald-200',
        className,
      )}
    >
      <Icon className={cn('h-3.5 w-3.5 shrink-0 mt-0.5', isRisky ? 'text-red-400' : 'text-emerald-400')} />
      <div className="flex-1 space-y-1">
        {zh && <div>{zh}</div>}
        {en && <div className="text-[10px] opacity-80">{en}</div>}
      </div>
    </div>
  )
}
