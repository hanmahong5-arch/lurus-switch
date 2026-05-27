import { type ButtonHTMLAttributes, type ReactNode } from 'react'
import { cn } from '../../../lib/utils'

// Magic UI — ShimmerButton (copy-paste, no extra deps beyond motion)
// Source: magicui.design/docs/components/shimmer-button (adapted)
// Adds a sweeping shimmer highlight on the button border to signal
// primary CTA importance without being loud.

interface ShimmerButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode
  shimmerColor?: string
  shimmerSize?: string
  shimmerDuration?: string
  borderRadius?: string
  background?: string
  className?: string
}

export function ShimmerButton({
  children,
  shimmerColor = 'rgba(255,255,255,0.22)',
  shimmerSize = '0.1em',
  shimmerDuration = '2.2s',
  borderRadius = '6px',
  background = 'hsl(var(--primary))',
  className,
  ...props
}: ShimmerButtonProps) {
  return (
    <button
      className={cn(
        'group relative z-0 flex cursor-pointer items-center justify-center gap-1.5',
        'overflow-hidden whitespace-nowrap px-4 py-2 text-sm font-medium',
        'text-primary-foreground transition-all duration-300',
        'disabled:opacity-50 disabled:cursor-not-allowed',
        className,
      )}
      style={
        {
          '--shimmer-color': shimmerColor,
          '--shimmer-size': shimmerSize,
          '--shimmer-duration': shimmerDuration,
          '--border-radius': borderRadius,
          '--background': background,
          borderRadius: 'var(--border-radius)',
          background: 'var(--background)',
        } as React.CSSProperties
      }
      {...props}
    >
      {/* Sweeping shimmer layer */}
      <div
        className="absolute inset-0 overflow-hidden"
        style={{ borderRadius: 'var(--border-radius)' }}
      >
        <div
          className="absolute inset-0 animate-shimmer-slide"
          style={
            {
              background: `linear-gradient(
                90deg,
                transparent 0%,
                var(--shimmer-color) 48%,
                transparent 52%,
                transparent 100%
              )`,
              backgroundSize: '200% 100%',
              animationDuration: 'var(--shimmer-duration)',
            } as React.CSSProperties
          }
        />
      </div>

      {/* Inner highlight edge */}
      <div
        className="absolute inset-[1px]"
        style={{
          borderRadius: 'calc(var(--border-radius) - 1px)',
          background:
            'linear-gradient(to bottom, rgba(255,255,255,0.08) 0%, transparent 60%)',
        }}
      />

      {/* Content */}
      <span className="relative z-10 flex items-center gap-1.5">{children}</span>
    </button>
  )
}
