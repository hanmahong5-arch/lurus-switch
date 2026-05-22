import { type LucideIcon } from 'lucide-react'
import { cn } from '../../lib/utils'
import { Card } from './Card'

interface KpiDelta {
  value: number
  label?: string
}

interface KpiCardProps {
  label: string
  value: string | number
  delta?: KpiDelta
  sparkline?: number[]
  icon?: LucideIcon
  accent?: boolean
  className?: string
}

function Sparkline({ points }: { points: number[] }) {
  if (points.length < 2) return null
  const min = Math.min(...points)
  const max = Math.max(...points)
  const span = max - min || 1
  const stepX = 100 / (points.length - 1)
  const path = points
    .map((p, i) => {
      const x = (i * stepX).toFixed(2)
      const y = (18 - ((p - min) / span) * 16).toFixed(2)
      return `${x},${y}`
    })
    .join(' ')
  return (
    <svg viewBox="0 0 100 20" className="w-full h-6 mt-2 text-primary/70" preserveAspectRatio="none">
      <polyline
        points={path}
        fill="none"
        stroke="currentColor"
        strokeWidth="0.9"
        strokeLinecap="round"
        strokeLinejoin="round"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  )
}

export function KpiCard({ label, value, delta, sparkline, icon: Icon, accent, className }: KpiCardProps) {
  const deltaPositive = delta && delta.value >= 0
  return (
    <Card variant="elevated" glow={accent} className={cn('p-3.5 flex flex-col min-w-0', className)}>
      <div className="flex items-center justify-between min-w-0">
        <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground truncate">
          [ {label.toUpperCase()} ]
        </span>
        {Icon && <Icon className="h-3.5 w-3.5 text-muted-foreground/60 shrink-0" />}
      </div>
      <div className="mt-2 font-mono text-2xl md:text-3xl tabular-nums tracking-tight text-foreground">
        {value}
      </div>
      {delta && (
        <div
          className={cn(
            'mt-1 font-mono text-[11px] tabular-nums flex items-center gap-1',
            deltaPositive ? 'text-emerald-400' : 'text-red-400',
          )}
        >
          <span>{deltaPositive ? '▲' : '▼'}</span>
          <span>{deltaPositive ? '+' : ''}{delta.value}%</span>
          {delta.label && <span className="text-muted-foreground/70 ml-1">· {delta.label}</span>}
        </div>
      )}
      {sparkline && sparkline.length >= 2 && <Sparkline points={sparkline} />}
    </Card>
  )
}
