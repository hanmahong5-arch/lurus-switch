import { cn } from '../../lib/utils'

interface QuotaCardProps {
  label: string
  used: number
  total: number
  className?: string
}

function formatQuota(value: number): string {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`
  return value.toString()
}

export function QuotaCard({ label, used, total, className }: QuotaCardProps) {
  const percentage = total > 0 ? Math.min((used / total) * 100, 100) : 0
  const remaining = Math.max(total - used, 0)

  return (
    <div className={cn('border border-border rounded-lg p-4 bg-card', className)}>
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm font-medium">{label}</span>
        <span className="text-xs text-muted-foreground">
          {formatQuota(used)} / {formatQuota(total)}
        </span>
      </div>

      {/* Progress bar */}
      <div className="w-full bg-muted rounded-full h-2 mb-2">
        <div
          className={cn(
            'h-2 rounded-full transition-all',
            percentage > 90 ? 'bg-red-500' : percentage > 70 ? 'bg-amber-500' : 'bg-primary'
          )}
          style={{ width: `${percentage}%` }}
        />
      </div>

      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span>{percentage.toFixed(1)}% used</span>
        <span>{formatQuota(remaining)} remaining</span>
      </div>
    </div>
  )
}
