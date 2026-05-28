import { useTranslation } from 'react-i18next'
import type { conversation } from '../../../wailsjs/go/models'
import { roleKindOf, type RoleKind } from './MessageCard'
import { parseSaneDate } from '../../lib/conversationUtils'
import { cn } from '../../lib/utils'

interface Props {
  events: conversation.Event[]
  activeIdx: number
  onJump: (idx: number) => void
}

const ROLE_DOT: Record<RoleKind, string> = {
  user: 'bg-blue-500',
  assistant: 'bg-amber-500',
  tool_use: 'bg-purple-500',
  tool_result: 'bg-emerald-500',
  system: 'bg-slate-500',
  meta: 'bg-zinc-500',
}

// MiniRail is the 16px vertical navigation column on the left of the
// timeline. Each event becomes a role-colored dot; the dot whose row is
// currently in the viewport gets a ring highlight, and clicking any dot
// scrolls that row into view (handled by parent via onJump).
export function MiniRail({ events, activeIdx, onJump }: Props) {
  const { t } = useTranslation()
  if (!events || events.length === 0) return null

  return (
    <div
      className="w-4 shrink-0 border-r border-border bg-card-recessed/60 overflow-y-auto py-2"
      role="navigation"
      aria-label={t('conversations.miniRail', 'Timeline navigator')}
    >
      <ul className="flex flex-col items-center gap-1">
        {events.map((ev, i) => {
          const role = roleKindOf(ev.type)
          const ts = parseSaneDate(ev.timestamp)
          const snippet = (ev.content || ev.toolName || '').slice(0, 30)
          const title = `${t(`conversations.role.${role}`, role)}${ts ? ` · ${ts.toLocaleTimeString()}` : ''}${snippet ? ` — ${snippet}` : ''}`
          const isActive = i === activeIdx
          return (
            <li key={ev.messageUUID || i}>
              <button
                onClick={() => onJump(i)}
                title={title}
                aria-label={title}
                data-testid={`mini-rail-dot-${i}`}
                data-active={isActive ? 'true' : undefined}
                className={cn(
                  'h-2 w-2 rounded-full transition-all hover:scale-150',
                  ROLE_DOT[role],
                  isActive && 'ring-2 ring-primary scale-125',
                )}
              />
            </li>
          )
        })}
      </ul>
    </div>
  )
}
