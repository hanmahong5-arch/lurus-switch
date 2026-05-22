import { useState } from 'react'
import { FileText, X } from 'lucide-react'
import type { conversation } from '../../../wailsjs/go/models'

interface Props {
  files: conversation.ContextFile[]
  onClose?: () => void
}

// ProjectContextDrawer is the read-only viewer for CLAUDE.md / AGENTS.md
// / .cursorrules files that live next to the session's cwd. v1 ships
// read-only by design — letting users edit these from inside a session
// browser is a bigger UX question we defer.
export function ProjectContextDrawer({ files, onClose }: Props) {
  const [active, setActive] = useState(0)
  if (!files || files.length === 0) {
    return (
      <div className="p-4 text-sm text-muted-foreground">
        No project context files (CLAUDE.md / AGENTS.md / .cursorrules) found.
      </div>
    )
  }
  const file = files[active]
  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-2 border-b border-border px-3 py-2">
        <FileText className="h-4 w-4 text-muted-foreground" />
        <span className="text-sm font-medium">Project context</span>
        {onClose && (
          <button onClick={onClose} className="ml-auto p-1 rounded hover:bg-muted">
            <X className="h-4 w-4" />
          </button>
        )}
      </div>
      <div className="flex gap-1 border-b border-border px-2 py-1.5 overflow-x-auto">
        {files.map((f, i) => (
          <button
            key={f.path}
            onClick={() => setActive(i)}
            className={
              'px-2 py-1 rounded text-xs font-medium whitespace-nowrap ' +
              (i === active
                ? 'bg-primary/15 text-primary'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted')
            }
          >
            {f.name}
          </button>
        ))}
      </div>
      <div className="flex-1 overflow-auto p-3">
        <pre className="whitespace-pre-wrap break-words text-xs font-mono text-foreground/90">
          {file.content}
        </pre>
        {file.truncated && (
          <div className="mt-3 text-[11px] text-amber-500">
            File truncated at 256 KB. Open in editor for the full content.
          </div>
        )}
      </div>
    </div>
  )
}
