import { cn } from '../../lib/utils'
import type { SubscriptionInfo } from '../../stores/billingStore'

interface SubscriptionCardProps {
  subscription?: SubscriptionInfo
  onManage?: () => void
}

const statusColors: Record<string, string> = {
  active: 'bg-green-500/10 text-green-600',
  expired: 'bg-red-500/10 text-red-600',
  cancelled: 'bg-muted text-muted-foreground',
  pending: 'bg-amber-500/10 text-amber-600',
}

export function SubscriptionCard({ subscription, onManage }: SubscriptionCardProps) {
  if (!subscription) {
    return (
      <div className="border border-border rounded-lg p-4 bg-card">
        <h3 className="text-sm font-medium mb-2">Subscription</h3>
        <p className="text-xs text-muted-foreground mb-3">No active subscription</p>
        {onManage && (
          <button
            onClick={onManage}
            className="px-3 py-1.5 rounded-md text-xs font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
          >
            Subscribe
          </button>
        )}
      </div>
    )
  }

  const colorClass = statusColors[subscription.status] || statusColors.pending

  return (
    <div className="border border-border rounded-lg p-4 bg-card">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-medium">Subscription</h3>
        <span className={cn('px-2 py-0.5 rounded text-xs font-medium', colorClass)}>
          {subscription.status}
        </span>
      </div>

      <div className="space-y-1.5 text-xs">
        <div className="flex justify-between">
          <span className="text-muted-foreground">Plan</span>
          <span className="font-medium">{subscription.plan_name}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-muted-foreground">Expires</span>
          <span>{subscription.expires_at || '-'}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-muted-foreground">Auto Renew</span>
          <span>{subscription.auto_renew ? 'Yes' : 'No'}</span>
        </div>
      </div>

      {onManage && (
        <button
          onClick={onManage}
          className="mt-3 w-full px-3 py-1.5 rounded-md text-xs font-medium border border-border hover:bg-muted transition-colors"
        >
          Manage
        </button>
      )}
    </div>
  )
}
