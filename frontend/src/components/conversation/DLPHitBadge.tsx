import { ShieldAlert } from 'lucide-react'
import type { audit } from '../../../wailsjs/go/models'

interface Props {
  hits: audit.Entry[]
  onOpenEntry?: (entryID: string) => void
}

// DLPHitBadge renders the "this message tripped DLP" pill that lives
// alongside the offending event in the Timeline. Clicking opens the
// audit drawer focused on the matching entry — the actual drawer
// implementation lives elsewhere; we only emit the entryID upward.
export function DLPHitBadge({ hits, onOpenEntry }: Props) {
  if (!hits || hits.length === 0) return null
  const first = hits[0]
  return (
    <button
      onClick={() => onOpenEntry?.(first.id)}
      className="inline-flex items-center gap-1 rounded-full bg-red-500/15 text-red-500 px-2 py-0.5 text-[11px] font-medium hover:bg-red-500/25 transition-colors"
      title={`${hits.length} DLP hit(s) — click to open audit entry`}
    >
      <ShieldAlert className="h-3 w-3" />
      DLP × {hits.length}
    </button>
  )
}
