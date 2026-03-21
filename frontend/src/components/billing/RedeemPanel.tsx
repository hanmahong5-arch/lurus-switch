import { useState } from 'react'
import { Loader2, Gift } from 'lucide-react'
import { cn } from '../../lib/utils'
import { classifyError } from '../../lib/errorClassifier'

interface RedeemPanelProps {
  onRedeem: (code: string) => Promise<number>
}

export function RedeemPanel({ onRedeem }: RedeemPanelProps) {
  const [code, setCode] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [result, setResult] = useState<{ success: boolean; message: string } | null>(null)

  const handleSubmit = async () => {
    if (!code.trim()) return
    setSubmitting(true)
    setResult(null)
    try {
      const amount = await onRedeem(code.trim())
      setResult({ success: true, message: `Redeemed ${amount} credits` })
      setCode('')
    } catch (err) {
      setResult({ success: false, message: classifyError(err).message })
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="border border-border rounded-lg p-4 bg-card">
      <h3 className="text-sm font-medium mb-3 flex items-center gap-2">
        <Gift className="h-4 w-4" />
        Redeem Code
      </h3>

      <div className="flex gap-2">
        <input
          type="text"
          value={code}
          onChange={(e) => setCode(e.target.value)}
          placeholder="Enter redeem code"
          className="flex-1 px-3 py-1.5 rounded-md text-sm border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary"
          onKeyDown={(e) => e.key === 'Enter' && handleSubmit()}
        />
        <button
          onClick={handleSubmit}
          disabled={!code.trim() || submitting}
          className={cn(
            'flex items-center gap-1.5 px-4 py-1.5 rounded-md text-sm font-medium transition-colors',
            'bg-primary text-primary-foreground hover:bg-primary/90',
            'disabled:opacity-50 disabled:cursor-not-allowed'
          )}
        >
          {submitting ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : null}
          Redeem
        </button>
      </div>

      {result && (
        <p className={cn('text-xs mt-2', result.success ? 'text-green-500' : 'text-red-500')}>
          {result.message}
        </p>
      )}
    </div>
  )
}
