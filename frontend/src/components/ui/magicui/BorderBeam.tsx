import { cn } from '../../../lib/utils'

// Magic UI — BorderBeam (copy-paste, no extra deps)
// Source: magicui.design/docs/components/border-beam (adapted)
// Animates a glowing spot travelling along the border of a card.
// Place inside a `position: relative; overflow: hidden` wrapper.
// Uses a CSS conic-gradient that rotates via a CSS @keyframes animation
// (no JS animation loop needed) to keep things smooth and dependency-free.

interface BorderBeamProps {
  /** Animation duration in seconds */
  duration?: number
  /** Inner border radius to match the host element */
  borderRadius?: string
  /** Border width in px */
  borderWidth?: number
  colorFrom?: string
  colorTo?: string
  className?: string
  delay?: number
}

export function BorderBeam({
  duration = 5,
  borderRadius = '0.5rem',
  borderWidth = 1.5,
  colorFrom = 'hsl(var(--primary) / 0.7)',
  colorTo = 'transparent',
  delay = 0,
  className,
}: BorderBeamProps) {
  const id = `bb-${Math.random().toString(36).slice(2, 8)}`

  return (
    <>
      <style>{`
        @keyframes ${id}-spin {
          from { --${id}-a: 0deg; }
          to   { --${id}-a: 360deg; }
        }
        @property --${id}-a {
          syntax: '<angle>';
          initial-value: 0deg;
          inherits: false;
        }
        .${id}-beam {
          animation: ${id}-spin ${duration}s linear ${delay}s infinite;
          background:
            conic-gradient(from var(--${id}-a), ${colorFrom} 0deg, ${colorTo} 60deg, transparent 90deg) border-box,
            transparent padding-box;
          -webkit-mask:
            linear-gradient(#fff 0 0) padding-box,
            linear-gradient(#fff 0 0) border-box;
          -webkit-mask-composite: destination-out;
          mask-composite: exclude;
        }
      `}</style>
      <div
        aria-hidden
        className={cn(`${id}-beam pointer-events-none absolute inset-0 z-[1]`, className)}
        style={{
          borderRadius,
          border: `${borderWidth}px solid transparent`,
        }}
      />
    </>
  )
}
