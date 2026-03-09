import { AlertTriangle, CheckCircle2 } from 'lucide-react'
import { validator } from '../../wailsjs/go/models'

interface ValidationPanelProps {
  result: validator.ValidationResult | null
  /** If true, also show a success indicator when result is valid. */
  showSuccess?: boolean
}

/**
 * Renders validation errors from a Go ValidationResult.
 * Shows nothing when result is null or valid (unless showSuccess is set).
 */
export function ValidationPanel({ result, showSuccess }: ValidationPanelProps) {
  if (!result) return null

  if (result.valid) {
    if (!showSuccess) return null
    return (
      <div className="flex items-center gap-1.5 text-xs text-green-500 py-1">
        <CheckCircle2 className="h-3.5 w-3.5 shrink-0" />
        Configuration is valid
      </div>
    )
  }

  const errors = result.errors || []

  return (
    <div className="rounded-md bg-red-500/10 border border-red-500/20 p-3 space-y-1.5">
      <div className="flex items-center gap-1.5 text-xs font-medium text-red-500">
        <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
        {errors.length} validation {errors.length === 1 ? 'error' : 'errors'}
      </div>
      <ul className="space-y-0.5">
        {errors.map((err, i) => (
          <li key={i} className="text-xs text-red-400">
            <span className="font-mono text-red-300">{err.field}</span>
            {' — '}
            {err.message}
          </li>
        ))}
      </ul>
    </div>
  )
}
