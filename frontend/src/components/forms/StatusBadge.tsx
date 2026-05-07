import { CheckCircle2, Circle, AlertTriangle, ShieldCheck, MinusCircle } from 'lucide-react'
import { cn } from '../../lib/utils'
import type { FieldStatus } from '../../lib/fieldMeta'

interface StatusBadgeProps {
  status: FieldStatus
  className?: string
}

// Visual treatment for each status. Colors hand-picked against the dark
// theme — verified against the existing palette (emerald/amber/red/zinc)
// so badges stand out without screaming.
const TREATMENT: Record<FieldStatus, {
  iconColor: string
  bg: string
  border: string
  text: string
  Icon: typeof CheckCircle2
  labelZh: string
  labelEn: string
}> = {
  set: {
    iconColor: 'text-sky-400',
    bg: 'bg-sky-500/10',
    border: 'border-sky-500/30',
    text: 'text-sky-300',
    Icon: CheckCircle2,
    labelZh: '已配置',
    labelEn: 'Set',
  },
  default: {
    iconColor: 'text-zinc-400',
    bg: 'bg-zinc-500/10',
    border: 'border-zinc-500/30',
    text: 'text-zinc-300',
    Icon: MinusCircle,
    labelZh: '默认值',
    labelEn: 'Default',
  },
  unset: {
    iconColor: 'text-amber-400',
    bg: 'bg-amber-500/10',
    border: 'border-amber-500/30',
    text: 'text-amber-300',
    Icon: Circle,
    labelZh: '未配置',
    labelEn: 'Unset',
  },
  risky: {
    iconColor: 'text-red-400',
    bg: 'bg-red-500/10',
    border: 'border-red-500/30',
    text: 'text-red-300',
    Icon: AlertTriangle,
    labelZh: '高风险',
    labelEn: 'Risky',
  },
  safe: {
    iconColor: 'text-emerald-400',
    bg: 'bg-emerald-500/10',
    border: 'border-emerald-500/30',
    text: 'text-emerald-300',
    Icon: ShieldCheck,
    labelZh: '安全',
    labelEn: 'Safe',
  },
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const tx = TREATMENT[status]
  const Icon = tx.Icon
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-medium border',
        tx.bg,
        tx.border,
        tx.text,
        className,
      )}
      title={`${tx.labelZh} / ${tx.labelEn}`}
    >
      <Icon className={cn('h-3 w-3', tx.iconColor)} />
      <span>{tx.labelZh}</span>
      <span className="text-muted-foreground/70">·</span>
      <span>{tx.labelEn}</span>
    </span>
  )
}
