// Live-session API helpers. Wraps the raw Wails runtime calls so the rest
// of the frontend has typed access without depending on `wails generate
// module` having run successfully (it has been unreliable in this repo —
// the generator silently no-ops on some Go ASTs and we'd rather not gate
// feature delivery on chasing that).
//
// Backend source of truth: internal/livesession/types.go.

declare global {
  interface Window {
    // The Wails runtime injects `window.go.main.App.<binding>` at boot.
    // We type only the bindings we use here to keep the surface narrow.
    go: {
      main: {
        App: {
          GetLiveSessions: () => Promise<LiveSession[]>
          GetAllLiveSessions: () => Promise<LiveSession[]>
          GetSessionTranscript: (path: string) => Promise<TranscriptEvent[]>
        }
      }
    }
  }
}

export type LiveSessionStatus = 'running' | 'tool_call' | 'awaiting_user' | 'idle'

export interface PendingTool {
  name: string
  preview: string
  startedAt: string
}

export interface EventSummary {
  time: string
  kind: 'user' | 'assistant' | 'tool' | 'result' | 'system'
  label: string
  details?: string
}

export interface FileTouch {
  path: string
  count: number
  kind: 'read' | 'edit' | 'write'
}

export interface LiveSession {
  sessionId: string
  tool: string
  cwd: string
  projectName: string
  startedAt: string
  lastActivity: string
  model?: string
  transcriptPath: string
  status: LiveSessionStatus
  pendingTool?: PendingTool | null
  recent: EventSummary[]
  messageCount: number
  toolCallCount: number
  // Four billable token streams. The cost field already accounts for all
  // four at per-message rates — these are exposed only so the UI can show
  // the breakdown (cache hit ratio is useful diagnostic data).
  inputTokens: number
  outputTokens: number
  cacheCreateTokens: number
  cacheReadTokens: number
  estimatedUsd: number
  // All distinct models the session saw, in order of first appearance.
  // >1 entry means the cost number is a sum of differently-priced spans
  // and the UI flags it so users don't read it as one-rate.
  modelsSeen?: string[]
  bashCommands?: string[]
  filesTouched?: FileTouch[]
}

export async function getLiveSessions(): Promise<LiveSession[]> {
  // Defensive: window.go may not be present in non-Wails dev contexts
  // (e.g. running `bun run test`). Return an empty list there rather than
  // crashing the page render.
  if (typeof window === 'undefined' || !window.go?.main?.App?.GetLiveSessions) {
    return []
  }
  return window.go.main.App.GetLiveSessions()
}

export async function getAllLiveSessions(): Promise<LiveSession[]> {
  if (typeof window === 'undefined' || !window.go?.main?.App?.GetAllLiveSessions) {
    return []
  }
  return window.go.main.App.GetAllLiveSessions()
}

// TranscriptEventType mirrors `conversation.EventType` in Go (parser.go).
// Keep the string values in sync with the backend constants.
export type TranscriptEventType =
  | 'user'
  | 'assistant'
  | 'system'
  | 'tool_use'
  | 'tool_result'
  | 'meta'

// TranscriptEvent mirrors `conversation.Event` in Go. We hand-roll the
// shape rather than depend on `wails generate module` — the generator is
// unreliable on this repo's AST and `wailsjs/go/models.ts` does not
// currently expose the conversation package.
//
// Fields are deliberately permissive (most optional) because the parser
// is permissive too — different CLI vendors disagree on schema.
export interface TranscriptEvent {
  type: TranscriptEventType
  // Both `messageUUID` (omitempty) and the Go zero-time-then-marshalled
  // timestamp can arrive as undefined / empty string; downstream
  // formatters must handle that.
  messageUUID?: string
  parentUUID?: string
  timestamp: string
  content?: string
  toolName?: string
  // toolArgs is `json.RawMessage` server-side — arrives as already-decoded
  // JSON here (object, array, string, etc.).
  toolArgs?: unknown
  model?: string
  inputTokens?: number
  outputTokens?: number
  raw?: unknown
}

// getSessionTranscript fetches the full parsed JSONL for a live session.
// The backend caps the result at the last 500 events and refuses paths
// outside `~/.{claude,codex,gemini}` — see bindings_livesession.go.
export async function getSessionTranscript(path: string): Promise<TranscriptEvent[]> {
  if (typeof window === 'undefined' || !window.go?.main?.App?.GetSessionTranscript) {
    return []
  }
  if (!path) return []
  return window.go.main.App.GetSessionTranscript(path)
}
