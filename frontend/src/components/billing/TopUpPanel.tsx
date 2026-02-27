import { useState } from 'react'
import { Loader2 } from 'lucide-react'
import { cn } from '../../lib/utils'
import type { TopUpInfo } from '../../stores/billingStore'

interface TopUpPanelProps {
  topUpInfo: TopUpInfo | null
  onTopUp: (amount: number, method: string) => Promise<void>
  loading?: boolean
}

export function TopUpPanel({ topUpInfo, onTopUp, loading }: TopUpPanelProps) {
  const [selectedAmount, setSelectedAmount] = useState<number | null>(null)
  const [selectedMethod, setSelectedMethod] = useState<string>('')
  const [submitting, setSubmitting] = useState(false)

  if (!topUpInfo) {
    return (
      <div className="border border-border rounded-lg p-4 bg-card">
        <h3 className="text-sm font-medium mb-2">Top Up</h3>
        <p className="text-xs text-muted-foreground">Loading top-up options...</p>
      </div>
    )
  }

  const handleSubmit = async () => {
    if (!selectedAmount || !selectedMethod) return
    setSubmitting(true)
    try {
      await onTopUp(selectedAmount, selectedMethod)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="border border-border rounded-lg p-4 bg-card">
      <h3 className="text-sm font-medium mb-3">Top Up</h3>

      {/* Amount grid */}
      <div className="grid grid-cols-3 gap-2 mb-3">
        {topUpInfo.amount_options.map((amount) => (
          <button
            key={amount}
            onClick={() => setSelectedAmount(amount)}
            className={cn(
              'px-3 py-2 rounded-md text-sm font-medium border transition-colors',
              selectedAmount === amount
                ? 'border-primary bg-primary/10 text-primary'
                : 'border-border hover:bg-muted'
            )}
          >
            {amount}
          </button>
        ))}
      </div>

      {/* Payment method */}
      {topUpInfo.pay_methods.length > 0 && (
        <div className="mb-3">
          <label className="text-xs text-muted-foreground mb-1.5 block">Payment Method</label>
          <div className="flex gap-2">
            {topUpInfo.pay_methods.map((method) => {
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

      {/* Discount notice */}
      {topUpInfo.discount > 0 && topUpInfo.discount < 1 && (
        <p className="text-xs text-green-500 mb-3">
          Discount: {((1 - topUpInfo.discount) * 100).toFixed(0)}% off
        </p>
      )}

      {/* Submit */}
      <button
        onClick={handleSubmit}
        disabled={!selectedAmount || !selectedMethod || submitting || loading}
        className={cn(
          'w-full flex items-center justify-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium transition-colors',
          'bg-primary text-primary-foreground hover:bg-primary/90',
          'disabled:opacity-50 disabled:cursor-not-allowed'
        )}
      >
        {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
        {submitting ? 'Processing...' : 'Pay'}
      </button>
    </div>
  )
}
