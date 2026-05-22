import { GitBranch, GitCommit } from 'lucide-react'
import type { conversation } from '../../../wailsjs/go/models'

interface Props {
  current: conversation.ConversationMeta
  siblings: conversation.ConversationMeta[]
  onJump?: (tool: string, sessionID: string) => void
}

// ForkTreeView is the small "this session has a parent / children"
// indicator shown in the session detail header. Renders a one-or-two
// hop view — full DAG navigation is overkill for the common case.
export function ForkTreeView({ current, siblings, onJump }: Props) {
  const parent = siblings.find((s) => s.sessionID === current.parentSessionID)
  const children = siblings.filter((s) => s.parentSessionID === current.sessionID)
  if (!parent && children.length === 0) return null

  return (
    <div className="flex items-center gap-2 px-3 py-2 bg-muted/30 border-b border-border text-xs">
      <GitBranch className="h-3.5 w-3.5 text-muted-foreground" />
      {parent && (
        <>
          <span className="text-muted-foreground">forked from</span>
          <button
            onClick={() => onJump?.(parent.tool, parent.sessionID)}
            className="px-1.5 py-0.5 rounded bg-background hover:bg-muted text-foreground font-mono"
            title={parent.sessionID}
          >
            {parent.sessionID.slice(0, 8)}…
          </button>
          {current.forkPointUUID && (
            <span className="text-muted-foreground">@ {current.forkPointUUID.slice(0, 8)}…</span>
          )}
        </>
      )}
      {children.length > 0 && (
        <>
          {parent && <span className="text-muted-foreground">·</span>}
          <span className="text-muted-foreground">forked into {children.length}:</span>
          {children.slice(0, 4).map((c) => (
            <button
              key={c.sessionID}
              onClick={() => onJump?.(c.tool, c.sessionID)}
              className="px-1.5 py-0.5 rounded bg-background hover:bg-muted text-foreground font-mono inline-flex items-center gap-1"
              title={c.sessionID}
            >
              <GitCommit className="h-3 w-3" />
              {c.sessionID.slice(0, 8)}…
            </button>
          ))}
          {children.length > 4 && (
            <span className="text-muted-foreground">+{children.length - 4} more</span>
          )}
        </>
      )}
    </div>
  )
}
