import { cn } from '../../lib/utils'

interface SelectOption {
  value: string
  label: string
}

interface SelectFieldProps {
  label: string
  value: string
  options: SelectOption[]
  onChange: (value: string) => void
  description?: string
  disabled?: boolean
  className?: string
}

/** Labeled select dropdown. */
export function SelectField({ label, value, options, onChange, description, disabled, className }: SelectFieldProps) {
  return (
    <div className={cn('space-y-1', className)}>
      <label className="text-xs font-medium text-muted-foreground">{label}</label>
      {description && (
        <p className="text-xs text-muted-foreground/70">{description}</p>
      )}
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        className={cn(
          'w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md',
          'focus:outline-none focus:ring-1 focus:ring-primary',
          'disabled:cursor-not-allowed disabled:opacity-50'
        )}
      >
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
    </div>
  )
}
