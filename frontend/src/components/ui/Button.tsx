import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from 'react'
import { Loader2 } from 'lucide-react'
import { cn } from '../../lib/utils'

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger'
export type ButtonSize = 'sm' | 'md' | 'lg'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ButtonSize
  loading?: boolean
  icon?: ReactNode
  iconPos?: 'left' | 'right'
}

const SIZE: Record<ButtonSize, string> = {
  sm: 'h-7 px-2 text-[11px] gap-1',
  md: 'h-8 px-3 text-xs gap-1.5',
  lg: 'h-10 px-4 text-sm gap-2',
}

const VARIANT: Record<ButtonVariant, string> = {
  primary:
    'bg-primary text-primary-foreground hover:bg-primary/90 ring-1 ring-primary/40 focus-visible:ring-2 focus-visible:ring-primary',
  secondary:
    'bg-muted text-foreground border border-border hover:bg-muted/70 focus-visible:ring-2 focus-visible:ring-primary',
  ghost:
    'text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:ring-2 focus-visible:ring-primary',
  danger:
    'bg-destructive text-destructive-foreground hover:bg-destructive/90 focus-visible:ring-2 focus-visible:ring-destructive',
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = 'primary', size = 'md', loading, icon, iconPos = 'left', className, children, disabled, ...rest },
  ref,
) {
  const isDisabled = disabled || loading
  const iconNode = loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : icon
  return (
    <button
      ref={ref}
      disabled={isDisabled}
      className={cn(
        'inline-flex items-center justify-center rounded-md font-medium transition-all duration-150',
        'focus-visible:outline-none disabled:opacity-50 disabled:cursor-not-allowed',
        SIZE[size],
        VARIANT[variant],
        className,
      )}
      {...rest}
    >
      {iconNode && iconPos === 'left' && iconNode}
      {children}
      {iconNode && iconPos === 'right' && iconNode}
    </button>
  )
})
