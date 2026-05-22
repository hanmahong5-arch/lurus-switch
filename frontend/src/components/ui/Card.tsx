import { forwardRef, type HTMLAttributes, type ElementType, type ReactNode } from 'react'
import { cn } from '../../lib/utils'

export type CardVariant = 'default' | 'elevated' | 'recessed'

interface CardProps extends HTMLAttributes<HTMLDivElement> {
  variant?: CardVariant
  glow?: boolean
  as?: ElementType
  children?: ReactNode
  // Allow button-as cases (`as="button"`) to pass through standard button attrs
  // without needing a generic-polymorphic Card. Surgical: enumerate what we use.
  disabled?: boolean
  type?: 'button' | 'submit' | 'reset'
}

const VARIANT: Record<CardVariant, string> = {
  default:  'bg-card/40 border border-border',
  elevated: 'bg-card-elevated border border-rule-strong shadow-card-elevated',
  recessed: 'bg-card-recessed border border-border/60',
}

export const Card = forwardRef<HTMLDivElement, CardProps>(function Card(
  { variant = 'default', glow, as, className, children, ...rest },
  ref,
) {
  const Comp = (as ?? 'div') as ElementType
  return (
    <Comp
      ref={ref as never}
      className={cn(
        'rounded-md transition-all duration-150',
        VARIANT[variant],
        glow && 'ring-1 ring-primary/60 shadow-glow-orange',
        className,
      )}
      {...rest}
    >
      {children}
    </Comp>
  )
})
