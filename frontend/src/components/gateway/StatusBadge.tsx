import { cn } from '../../lib/utils'

type StatusType = 'enabled' | 'disabled' | 'used' | 'expired' | 'active' | 'pending'

const STATUS_STYLES: Record<StatusType, string> = {
  enabled: 'bg-green-900/40 text-green-400',
  active: 'bg-green-900/40 text-green-400',
  disabled: 'bg-muted text-muted-foreground',
  used: 'bg-blue-900/40 text-blue-400',
  expired: 'bg-red-900/40 text-red-400',
  pending: 'bg-yellow-900/40 text-yellow-400',
}

const DEFAULT_LABELS: Record<StatusType, string> = {
  enabled: 'Enabled',
  active: 'Active',
  disabled: 'Disabled',
  used: 'Used',
  expired: 'Expired',
  pending: 'Pending',
}

interface StatusBadgeProps {
  status: StatusType
  labels?: Partial<Record<StatusType, string>>
}

export function StatusBadge({ status, labels }: StatusBadgeProps) {
  const label = labels?.[status] ?? DEFAULT_LABELS[status] ?? status
  const style = STATUS_STYLES[status] ?? 'bg-muted text-muted-foreground'

  return (
    <span className={cn('text-xs rounded px-1.5 py-0.5 inline-block', style)}>
      {label}
    </span>
  )
}
