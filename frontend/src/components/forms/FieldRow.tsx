import type { ReactNode } from 'react'
import { cn } from '../../lib/utils'
import { getMeta, getStatus, type FieldMeta } from '../../lib/fieldMeta'
import { StatusBadge } from './StatusBadge'
import { SecurityAdvisory } from './SecurityAdvisory'

interface FieldRowProps {
  metaKey: string
  value: unknown
  // 'inline'  — label on the left, control on the right (good for Switch).
  // 'stacked' — label on top, control below at full width (text/select).
  layout?: 'inline' | 'stacked'
  // Optional override: when provided, shown on the right side of the
  // header row in addition to the status badge. Used for "field error".
  errorMessage?: string
  // The actual input element. Should be unlabeled — FieldRow renders the
  // label/desc/status/advisory chrome.
  children: ReactNode
  className?: string
}

// FieldRow wraps any input control with the bilingual label, description,
// status badge, and (when applicable) safety advisory. Looks up metadata
// by metaKey from FIELD_META; if no entry exists the row falls back to a
// plain pass-through so callers can mix-and-match metadata-driven and
// hand-crafted fields without a runtime crash.
export function FieldRow({
  metaKey,
  value,
  layout = 'stacked',
  errorMessage,
  children,
  className,
}: FieldRowProps) {
  const meta = getMeta(metaKey)
  if (!meta) {
    return <div className={className}>{children}</div>
  }
  const status = getStatus(meta, value)
  const showAdvisory = (status === 'risky' || status === 'safe')
  const advisoryZh = status === 'risky' ? meta.advisoryRiskyZh : meta.advisorySafeZh
  const advisoryEn = status === 'risky' ? meta.advisoryRiskyEn : meta.advisorySafeEn

  if (layout === 'inline') {
    return (
      <div className={cn('py-2 border-b border-border/40 last:border-0', className)}>
        <div className="flex items-start justify-between gap-3">
          <div className="flex-1 min-w-0">
            <Header meta={meta} status={status} />
            <Description meta={meta} />
          </div>
          <div className="shrink-0 pt-0.5">{children}</div>
        </div>
        {errorMessage && <p className="text-[11px] text-red-400 mt-1">{errorMessage}</p>}
        {showAdvisory && (
          <SecurityAdvisory level={status as 'risky' | 'safe'} zh={advisoryZh} en={advisoryEn} />
        )}
      </div>
    )
  }

  // Stacked layout
  return (
    <div className={cn('space-y-1.5 py-2 border-b border-border/40 last:border-0', className)}>
      <Header meta={meta} status={status} />
      <Description meta={meta} />
      {children}
      {errorMessage && <p className="text-[11px] text-red-400">{errorMessage}</p>}
      {showAdvisory && (
        <SecurityAdvisory level={status as 'risky' | 'safe'} zh={advisoryZh} en={advisoryEn} />
      )}
    </div>
  )
}

function Header({ meta, status }: { meta: FieldMeta; status: ReturnType<typeof getStatus> }) {
  return (
    <div className="flex items-center gap-2 flex-wrap">
      <span className="text-xs font-medium text-foreground">{meta.labelZh}</span>
      <span className="text-[10px] text-muted-foreground/70 font-mono">{meta.labelEn}</span>
      <StatusBadge status={status} />
    </div>
  )
}

function Description({ meta }: { meta: FieldMeta }) {
  if (!meta.descZh && !meta.descEn) return null
  return (
    <div className="space-y-0.5">
      {meta.descZh && <p className="text-[11px] text-muted-foreground leading-relaxed">{meta.descZh}</p>}
      {meta.descEn && <p className="text-[10px] text-muted-foreground/60 leading-relaxed">{meta.descEn}</p>}
    </div>
  )
}
