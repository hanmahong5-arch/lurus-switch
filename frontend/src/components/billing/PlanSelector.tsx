import { useState } from 'react'
import { Loader2, Check } from 'lucide-react'
import { cn } from '../../lib/utils'
import type { SubscriptionPlan, TopUpInfo } from '../../stores/billingStore'

interface PlanSelectorProps {
  plans: SubscriptionPlan[]
  payMethods: TopUpInfo['pay_methods']
  currentPlanCode?: string
  onSubscribe: (planCode: string, method: string) => Promise<void>
  loading?: boolean
}

export function PlanSelector({ plans, payMethods, currentPlanCode, onSubscribe, loading }: PlanSelectorProps) {
  const [selectedPlan, setSelectedPlan] = useState<string>('')
  const [selectedMethod, setSelectedMethod] = useState<string>('')
  const [submitting, setSubmitting] = useState(false)

  if (plans.length === 0) {
    return null
  }

  const handleSubscribe = async () => {
    if (!selectedPlan || !selectedMethod) return
    setSubmitting(true)
    try {
      await onSubscribe(selectedPlan, selectedMethod)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="border border-border rounded-lg p-4 bg-card">
      <h3 className="text-sm font-medium mb-3">Available Plans</h3>

      <div className="space-y-2 mb-3">
        {plans.map((plan) => (
          <button
            key={plan.code}
            onClick={() => setSelectedPlan(plan.code)}
            className={cn(
              'w-full flex items-center justify-between p-3 rounded-md border text-left transition-colors',
              selectedPlan === plan.code
                ? 'border-primary bg-primary/5'
                : 'border-border hover:bg-muted',
              currentPlanCode === plan.code && 'ring-1 ring-green-500'
            )}
          >
            <div>
              <div className="text-sm font-medium flex items-center gap-1.5">
                {plan.name}
                {currentPlanCode === plan.code && (
                  <span className="text-xs bg-green-500/10 text-green-600 px-1.5 py-0.5 rounded">Current</span>
                )}
              </div>
              <div className="text-xs text-muted-foreground mt-0.5">
                {plan.duration} &middot; Daily: {plan.daily_quota} &middot; Total: {plan.total_quota}
              </div>
              {plan.features.length > 0 && (
                <div className="flex flex-wrap gap-1 mt-1">
                  {plan.features.map((f) => (
                    <span key={f} className="text-xs bg-muted px-1.5 py-0.5 rounded">{f}</span>
                  ))}
                </div>
              )}
            </div>
            <div className="text-right shrink-0 ml-4">
              <div className="text-sm font-semibold">
                {plan.currency} {plan.price}
              </div>
              {selectedPlan === plan.code && <Check className="h-4 w-4 text-primary mt-1 ml-auto" />}
            </div>
          </button>
        ))}
      </div>

      {/* Payment method selection */}
      {selectedPlan && payMethods.length > 0 && (
        <div className="mb-3">
          <label className="text-xs text-muted-foreground mb-1.5 block">Payment Method</label>
          <div className="flex gap-2">
            {payMethods.map((method) => {
              const key = Object.keys(method)[0]
              const label = method[key]
              return (
                <button
                  key={key}
                  onClick={() => setSelectedMethod(key)}
                  className={cn(
                    'flex-1 px-3 py-1.5 rounded-md text-xs font-medium border transition-colors',
                    selectedMethod === key
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'border-border hover:bg-muted'
                  )}
                >
                  {label}
                </button>
              )
            })}
          </div>
        </div>
      )}

      {/* Subscribe button */}
      {selectedPlan && (
        <button
          onClick={handleSubscribe}
          disabled={!selectedMethod || submitting || loading}
          className={cn(
            'w-full flex items-center justify-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium transition-colors',
            'bg-primary text-primary-foreground hover:bg-primary/90',
            'disabled:opacity-50 disabled:cursor-not-allowed'
          )}
        >
          {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
          {submitting ? 'Processing...' : 'Subscribe'}
        </button>
      )}
    </div>
  )
}
