// TypeScript types mirroring lurus-switch/internal/configapply/{plan,result}.go.
// Kept hand-maintained because Wails 2 binding generation does not always pick
// up sub-package structs reliably; sub-agents wiring more call sites should
// reuse these types instead of redefining locally.

export type ChangeKind = 'create' | 'update' | 'delete'

export type ApplyPhase = 'pending' | 'validate' | 'snapshot' | 'write' | 'verify' | 'done'

export interface FileChange {
  path: string
  kind: ChangeKind
  before?: string
  after?: string
  mode: number
  diffSummary: string
  unifiedDiff?: string
}

export interface ChangePlan {
  id: string
  intent: string
  description: string
  createdAt: string
  changes: FileChange[]
  sideEffects?: string[]
}

export interface NextStep {
  label: string
  action: string
  url?: string
  params?: Record<string, string>
}

export interface ApplyResult {
  planID: string
  success: boolean
  phase: ApplyPhase
  startedAt: string
  finishedAt?: string
  whatHappened?: string
  whatExpected?: string
  rollbackDone: boolean
  rollbackNote?: string
  nextSteps?: NextStep[]
  filesWritten?: string[]
  filesRolled?: string[]
  rawError?: string
}
