import { useMemo, type MutableRefObject } from 'react'
import { useTranslation } from 'react-i18next'
import type { conversation, audit } from '../../../wailsjs/go/models'
import { MessageCard } from './MessageCard'

interface Props {
  events: conversation.Event[]
  dlpHits: audit.Entry[]
  onFork?: (messageUUID: string) => void
  onOpenAuditEntry?: (entryID: string) => void
  forking?: boolean
  messageRefs?: MutableRefObject<Array<HTMLDivElement | null>>
}

// Timeline renders a session's events top-down as MessageCard rows.
// Per-row refs are written back into messageRefs.current[i] so the
// MiniRail can scrollIntoView() the right row when the user clicks.
export function Timeline({
  events, dlpHits, onFork, onOpenAuditEntry, forking, messageRefs,
}: Props) {
  const { t } = useTranslation()

  const hitsByMessage = useMemo(() => {
    const map = new Map<string, audit.Entry[]>()
    for (const h of dlpHits) {
      const uuid = h.metadata?.conv_message_uuid
      if (!uuid) continue
      const cur = map.get(uuid) || []
      cur.push(h)
      map.set(uuid, cur)
    }
    return map
  }, [dlpHits])

  if (!events || events.length === 0) {
    return (
      <div className="text-sm text-muted-foreground p-6">
        {t('conversations.noMessages', 'No messages in this session.')}
      </div>
    )
  }

  if (messageRefs) {
    messageRefs.current.length = events.length
  }

  return (
    <div className="py-2">
      {events.map((ev, i) => (
        <MessageCard
          key={ev.messageUUID || i}
          ref={(el) => { if (messageRefs) messageRefs.current[i] = el }}
          event={ev}
          hits={hitsByMessage.get(ev.messageUUID || '') || []}
          onFork={onFork}
          onOpenAuditEntry={onOpenAuditEntry}
          forking={forking}
        />
      ))}
    </div>
  )
}
