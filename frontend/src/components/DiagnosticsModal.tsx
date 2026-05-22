import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, CheckCircle2, AlertCircle, XCircle, X, RefreshCw, FileText, FolderOpen } from 'lucide-react'
import { RunDiagnostics, WriteDebugDump, OpenDebugDumpDir } from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'

interface Props {
  open: boolean
  onClose: () => void
}

// DiagnosticsModal — Hermes-style "Run Diagnostics" panel.
// Auto-runs on open so the user sees results immediately; re-run is one click.
// The "Generate Debug Dump" path writes a redacted JSON the user can attach
// to a support email without auditing every field by hand.
export function DiagnosticsModal({ open, onClose }: Props) {
  const { t } = useTranslation()
  const [report, setReport] = useState<main.DiagnosticsReport | null>(null)
  const [running, setRunning] = useState(false)
  const [dumpPath, setDumpPath] = useState<string | null>(null)
  const [dumping, setDumping] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const run = async () => {
    setRunning(true)
    setError(null)
    try {
      const r = await RunDiagnostics()
      setReport(r)
    } catch (e: any) {
      setError(e?.message ?? String(e))
    } finally {
      setRunning(false)
    }
  }

  useEffect(() => {
    if (open && !report) run()
    // Reset on close so reopen always reflects current state.
    if (!open) {
      setDumpPath(null)
      setError(null)
    }
  }, [open])

  const handleDump = async () => {
    setDumping(true)
    setError(null)
    try {
      const path = await WriteDebugDump()
      setDumpPath(path)
    } catch (e: any) {
      setError(e?.message ?? String(e))
    } finally {
      setDumping(false)
    }
  }

  if (!open) return null

  const counts = (report?.checks ?? []).reduce(
    (acc, c) => {
      if (c.status === 'ok') acc.ok++
      else if (c.status === 'warn') acc.warn++
      else acc.fail++
      return acc
    },
    { ok: 0, warn: 0, fail: 0 },
  )

  return (
    <div
      className="fixed inset-0 z-50 bg-black/30 backdrop-blur-sm flex items-center justify-center p-4"
      onClick={onClose}
    >
      <div
        className="bg-card border border-border rounded-lg w-full max-w-2xl max-h-[80vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <div>
            <h2 className="text-sm font-semibold">{t('diagnostics.title', '系统诊断')}</h2>
            {report && (
              <p className="text-[11px] text-muted-foreground">
                {t('diagnostics.summary', '{{ok}} 正常 · {{warn}} 警告 · {{fail}} 失败', counts)}
              </p>
            )}
          </div>
          <button onClick={onClose} className="p-1 hover:bg-muted rounded">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto p-4 space-y-2">
          {running && !report && (
            <div className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              {t('diagnostics.running', '正在收集…')}
            </div>
          )}

          {error && (
            <div className="flex items-start gap-2 p-2 rounded-md bg-destructive/10 text-destructive text-xs">
              <AlertCircle className="h-3.5 w-3.5 mt-0.5 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {report?.checks.map((c) => (
            <div
              key={c.id}
              className="flex items-start gap-2 px-3 py-2 rounded-md border border-border"
            >
              <StatusIcon status={c.status} />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium">{c.label}</p>
                <p className="text-xs text-muted-foreground break-all">{c.detail}</p>
              </div>
            </div>
          ))}

          {report && (
            <p className="text-[11px] text-muted-foreground pt-2 border-t border-border mt-2">
              {t('diagnostics.meta', '生成时间 {{ts}} · v{{ver}} · {{os}}/{{arch}}', {
                ts: report.generatedAt,
                ver: report.appVersion,
                os: report.os,
                arch: report.arch,
              })}
            </p>
          )}

          {dumpPath && (
            <div className="flex items-start gap-2 p-2 rounded-md bg-green-500/10 text-green-600 dark:text-green-400 text-xs">
              <CheckCircle2 className="h-3.5 w-3.5 mt-0.5 shrink-0" />
              <span className="break-all">
                {t('diagnostics.dumpSaved', 'Debug dump 已保存到：')}
                <br />
                <code className="font-mono">{dumpPath}</code>
              </span>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between gap-2 px-4 py-3 border-t border-border">
          <button
            onClick={() => OpenDebugDumpDir().catch(() => {})}
            className="px-2 py-1.5 text-xs border border-border rounded hover:bg-muted inline-flex items-center gap-1"
            title={t('diagnostics.openDumpsDir', '打开 dump 目录')}
          >
            <FolderOpen className="h-3 w-3" />
            {t('diagnostics.openDumpsDir', '打开 dump 目录')}
          </button>
          <div className="flex items-center gap-2">
            <button
              onClick={handleDump}
              disabled={dumping}
              className="px-3 py-1.5 text-xs border border-border rounded hover:bg-muted inline-flex items-center gap-1 disabled:opacity-50"
            >
              {dumping ? <Loader2 className="h-3 w-3 animate-spin" /> : <FileText className="h-3 w-3" />}
              {t('diagnostics.dump', '生成 Debug Dump')}
            </button>
            <button
              onClick={run}
              disabled={running}
              className="px-3 py-1.5 text-xs bg-primary text-primary-foreground rounded hover:bg-primary/90 inline-flex items-center gap-1 disabled:opacity-50"
            >
              {running ? <Loader2 className="h-3 w-3 animate-spin" /> : <RefreshCw className="h-3 w-3" />}
              {t('diagnostics.rerun', '重新检测')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function StatusIcon({ status }: { status: string }) {
  if (status === 'ok') return <CheckCircle2 className="h-4 w-4 mt-0.5 text-green-500 shrink-0" />
  if (status === 'warn') return <AlertCircle className="h-4 w-4 mt-0.5 text-amber-500 shrink-0" />
  return <XCircle className="h-4 w-4 mt-0.5 text-destructive shrink-0" />
}
