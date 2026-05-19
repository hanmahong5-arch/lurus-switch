import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { DiffEditor } from '@monaco-editor/react'
import { FileDiff, X, AlertTriangle, Loader2, Check } from 'lucide-react'
import type { ChangePlan, ApplyResult } from './types'
import { applyChangePlan } from './api'
import { ApplyResultCard } from './ApplyResultCard'

// Frontend half of F1 + F4 (configuration-rollback-design.md Features 1 & 4).
// Receives a ChangePlan from BuildChangePlan(), shows per-file unified diff in
// Monaco's DiffEditor, and on confirm calls ApplyChangePlan(). The post-apply
// ApplyResult is rendered via ApplyResultCard with 4 self-explaining elements.
//
// Sub-agents wiring call sites: pass the plan from BuildChangePlan, the modal
// handles the apply lifecycle. onApplied callback fires only on success so the
// caller can update its own UI (e.g. refresh tool config display).

interface ChangeReviewModalProps {
  plan: ChangePlan | null
  open: boolean
  onClose: () => void
  onApplied?: (result: ApplyResult) => void
}

export function ChangeReviewModal({ plan, open, onClose, onApplied }: ChangeReviewModalProps) {
  const { i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh')

  const [activeFileIdx, setActiveFileIdx] = useState(0)
  const [applying, setApplying] = useState(false)
  const [result, setResult] = useState<ApplyResult | null>(null)

  useEffect(() => {
    if (open) {
      setActiveFileIdx(0)
      setResult(null)
    }
  }, [open, plan?.id])

  if (!open || !plan) return null

  const active = plan.changes[activeFileIdx]
  const hasChanges = plan.changes.length > 0
  const showResultPane = result !== null

  const handleApply = async () => {
    setApplying(true)
    try {
      const r = await applyChangePlan(plan)
      setResult(r)
      if (r.success && onApplied) onApplied(r)
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      setResult({
        planID: plan.id,
        success: false,
        phase: 'pending',
        startedAt: new Date().toISOString(),
        rollbackDone: false,
        whatHappened: isZh
          ? '调用 ApplyChangePlan 时抛出 JS 异常,可能是 Wails 绑定未就绪'
          : 'Calling ApplyChangePlan threw a JS exception; Wails binding may not be ready',
        whatExpected: isZh
          ? 'Wails App 对象应在 desktop runtime 注入'
          : 'Wails App object should be injected by desktop runtime',
        rawError: message,
        nextSteps: [
          { label: isZh ? '重启 Switch' : 'Restart Switch', action: 'restart_app' },
        ],
      })
    } finally {
      setApplying(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="w-[1000px] max-w-[95vw] max-h-[90vh] bg-background border border-border rounded-lg shadow-2xl flex flex-col overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <div className="flex items-center gap-2 min-w-0">
            <FileDiff className="h-4 w-4 text-primary flex-shrink-0" />
            <div className="min-w-0">
              <h2 className="text-sm font-semibold truncate">
                {isZh ? '配置变更预览' : 'Configuration change preview'}
              </h2>
              <p className="text-xs text-muted-foreground truncate">
                {plan.description || plan.intent}
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {!hasChanges && (
          <div className="flex-1 flex items-center justify-center p-8 text-sm text-muted-foreground">
            {isZh ? '没有改动 — before 和 after 等价。' : 'No changes — before and after are equivalent.'}
          </div>
        )}

        {hasChanges && !showResultPane && (
          <div className="flex flex-1 min-h-0">
            <div className="w-56 flex-shrink-0 border-r border-border overflow-y-auto bg-muted/20">
              <p className="text-xs uppercase tracking-wider text-muted-foreground px-3 pt-3 pb-2">
                {isZh ? `${plan.changes.length} 个文件` : `${plan.changes.length} file(s)`}
              </p>
              {plan.changes.map((ch, i) => (
                <button
                  key={ch.path + i}
                  onClick={() => setActiveFileIdx(i)}
                  className={
                    'w-full text-left px-3 py-2 text-xs border-l-2 transition-colors ' +
                    (i === activeFileIdx
                      ? 'border-primary bg-primary/5'
                      : 'border-transparent hover:bg-muted')
                  }
                >
                  <p className="font-mono truncate" title={ch.path}>
                    {ch.path.split(/[/\\]/).pop()}
                  </p>
                  <p className="text-muted-foreground mt-0.5 flex items-center gap-1.5">
                    <KindBadge kind={ch.kind} />
                    <span>{ch.diffSummary}</span>
                  </p>
                </button>
              ))}
            </div>

            <div className="flex-1 min-w-0 flex flex-col">
              {active && (
                <>
                  <div className="px-3 py-2 border-b border-border bg-muted/10">
                    <p className="text-xs font-mono text-muted-foreground truncate">{active.path}</p>
                  </div>
                  <div className="flex-1 min-h-0">
                    <DiffEditor
                      original={active.before ?? ''}
                      modified={active.after ?? ''}
                      language={detectLanguage(active.path)}
                      theme="vs-dark"
                      options={{
                        readOnly: true,
                        renderSideBySide: true,
                        minimap: { enabled: false },
                        scrollBeyondLastLine: false,
                        fontSize: 12,
                      }}
                    />
                  </div>
                </>
              )}
            </div>
          </div>
        )}

        {showResultPane && (
          <div className="flex-1 overflow-auto p-4">
            <ApplyResultCard result={result!} onDismiss={onClose} />
            {plan.sideEffects && plan.sideEffects.length > 0 && result!.success && (
              <div className="mt-3 text-xs text-muted-foreground">
                <p className="uppercase tracking-wider mb-1">
                  {isZh ? '副作用' : 'Side effects'}
                </p>
                <ul className="list-disc list-inside space-y-0.5">
                  {plan.sideEffects.map((se, i) => <li key={i}>{se}</li>)}
                </ul>
              </div>
            )}
          </div>
        )}

        {hasChanges && !showResultPane && (
          <div className="border-t border-border px-4 py-3 flex items-center justify-between bg-muted/10">
            <div className="flex items-center gap-3 text-xs text-muted-foreground">
              {plan.sideEffects && plan.sideEffects.length > 0 && (
                <div className="flex items-center gap-1">
                  <AlertTriangle className="h-3 w-3 text-amber-500" />
                  <span>
                    {(isZh ? '副作用: ' : 'Side effects: ') + plan.sideEffects.length}
                  </span>
                </div>
              )}
            </div>
            <div className="flex gap-2">
              <button
                onClick={onClose}
                className="px-3 py-1.5 text-sm border border-border rounded hover:bg-muted transition-colors"
                disabled={applying}
              >
                {isZh ? '取消' : 'Cancel'}
              </button>
              <button
                onClick={handleApply}
                disabled={applying}
                className="px-3 py-1.5 text-sm font-medium bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center gap-1.5"
              >
                {applying ? (
                  <>
                    <Loader2 className="h-3 w-3 animate-spin" />
                    <span>{isZh ? '应用中…' : 'Applying…'}</span>
                  </>
                ) : (
                  <>
                    <Check className="h-3 w-3" />
                    <span>{isZh ? '应用变更' : 'Apply'}</span>
                  </>
                )}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function KindBadge({ kind }: { kind: 'create' | 'update' | 'delete' }) {
  const color =
    kind === 'create' ? 'text-green-500' : kind === 'delete' ? 'text-red-500' : 'text-blue-400'
  return <span className={'font-mono uppercase ' + color}>{kind}</span>
}

function detectLanguage(path: string): string {
  const lower = path.toLowerCase()
  if (lower.endsWith('.json')) return 'json'
  if (lower.endsWith('.toml')) return 'toml'
  if (lower.endsWith('.yaml') || lower.endsWith('.yml')) return 'yaml'
  if (lower.endsWith('.md')) return 'markdown'
  if (lower.endsWith('.ts') || lower.endsWith('.tsx')) return 'typescript'
  if (lower.endsWith('.js') || lower.endsWith('.jsx')) return 'javascript'
  return 'plaintext'
}
