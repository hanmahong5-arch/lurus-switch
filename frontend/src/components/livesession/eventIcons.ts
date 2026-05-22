// Shared icon map for live-session event rendering. The summary rows on
// the Live Sessions cards (`LiveSessionsPage.tsx`) and the full timeline
// in `SessionDetailDrawer.tsx` both render against this so the two views
// stay visually consistent.
//
// Two distinct enums map onto these icons:
//   - `EventSummary.kind` (`user | assistant | tool | result | system`),
//     emitted by the backend watcher for the at-a-glance card recents.
//   - `TranscriptEvent.type` (`user | assistant | tool_use | tool_result
//     | system | meta`), the richer parsed-line type from
//     `internal/conversation`.
//
// Resolve via the helper below — caller-side enum-narrowing keeps the
// indirection cheap and TS-typed.

import {
  Bot, MessageSquare, Wrench, Zap, AlertCircle, ChevronRight,
  type LucideIcon,
} from 'lucide-react'

export const EVENT_ICON = {
  user: MessageSquare,
  assistant: Bot,
  tool: Wrench,
  result: Zap,
  system: AlertCircle,
} as const

export type EventIconKey = keyof typeof EVENT_ICON

export function iconForSummaryKind(kind: string): LucideIcon {
  return (EVENT_ICON as Record<string, LucideIcon>)[kind] ?? ChevronRight
}

// iconForTranscriptType maps the richer `conversation.EventType` set onto
// the same five icons — `tool_use` collapses to the tool icon, `tool_result`
// to the result icon, `meta` falls through to the chevron.
export function iconForTranscriptType(type: string): LucideIcon {
  switch (type) {
    case 'user': return EVENT_ICON.user
    case 'assistant': return EVENT_ICON.assistant
    case 'tool_use': return EVENT_ICON.tool
    case 'tool_result': return EVENT_ICON.result
    case 'system': return EVENT_ICON.system
    default: return ChevronRight
  }
}
