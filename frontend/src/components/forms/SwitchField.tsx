import { cn } from '../../lib/utils'

interface SwitchFieldProps {
  label: string
  description?: string
  checked: boolean
  onChange: (checked: boolean) => void
  disabled?: boolean
  className?: string
}

/** Toggle switch with label and optional description. */
export function SwitchField({ label, description, checked, onChange, disabled, className }: SwitchFieldProps) {
  return (
    <div className={cn('flex items-center justify-between py-1', className)}>
      <div>
        <div className="text-xs font-medium">{label}</div>
        {description && (
          <div className="text-xs text-muted-foreground mt-0.5">{description}</div>
        )}
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className={cn(
          'relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent',
          'transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-1',
          checked ? 'bg-primary' : 'bg-muted',
          disabled && 'cursor-not-allowed opacity-50'
        )}
      >
        <span
          className={cn(
            'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow transition duration-200 ease-in-out',
            checked ? 'translate-x-4' : 'translate-x-0'
          )}
        />
      </button>
    </div>
  )
}
