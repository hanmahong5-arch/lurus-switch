import { useState } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import {
  ShieldAlert, ShieldCheck, ShieldQuestion, FolderOpen, Loader2,
  AlertTriangle, FileWarning, Archive, X, ExternalLink,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useToastStore } from '../stores/toastStore'
import { PickRepoAndAudit, AuditRepo, QuarantineFile } from '../../wailsjs/go/main/App'
import type { repoaudit } from '../../wailsjs/go/models'

interface RepoAuditModalProps {
  open: boolean
  onClose: () => void
}

const VERDICT_TREATMENT = {
  safe: {
    Icon: ShieldCheck,
    color: 'text-emerald-400',
    bg: 'bg-emerald-500/10',
    border: 'border-emerald-500/30',
    labelZh: '看起来安全',
    labelEn: 'Looks safe',
  },
  caution: {
    Icon: ShieldQuestion,
    color: 'text-amber-400',
    bg: 'bg-amber-500/10',
    border: 'border-amber-500/30',
    labelZh: '建议留意',
    labelEn: 'Worth a look',
  },
  risky: {
    Icon: ShieldAlert,
    color: 'text-red-400',
    bg: 'bg-red-500/10',
    border: 'border-red-500/30',
    labelZh: '存在高风险项',
    labelEn: 'High-risk findings',
  },
}

const SEVERITY_TREATMENT = {
  info: { color: 'text-zinc-400', bg: 'bg-zinc-500/10', border: 'border-zinc-500/30' },
  caution: { color: 'text-amber-400', bg: 'bg-amber-500/10', border: 'border-amber-500/30' },
  risky: { color: 'text-red-400', bg: 'bg-red-500/10', border: 'border-red-500/30' },
}

export function RepoAuditModal({ open, onClose }: RepoAuditModalProps) {
  const { t, i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh') ?? true
  const toast = useToastStore((s) => s.addToast)
  const [report, setReport] = useState<repoaudit.AuditReport | null>(null)
  const [scanning, setScanning] = useState(false)
  const [pathInput, setPathInput] = useState('')
  const [quarantining, setQuarantining] = useState<string | null>(null)

  const reset = () => {
    setReport(null)
    setPathInput('')
  }

  const handlePick = async () => {
    setScanning(true)
    try {
      const r = await PickRepoAndAudit()
      if (r) setReport(r)
    } catch (e) {
      toast('error', String(e))
    } finally {
      setScanning(false)
    }
  }

  const handleManualPath = async () => {
    if (!pathInput.trim()) return
    setScanning(true)
    try {
      const r = await AuditRepo(pathInput.trim())
      setReport(r)
    } catch (e) {
      toast('error', String(e))
    } finally {
      setScanning(false)
    }
  }

  const handleQuarantine = async (fullPath: string) => {
    if (!confirm(isZh
      ? `确认隔离该文件？\n${fullPath}\n会被改名为 .quarantined-by-lurus-switch.<时间戳>，以后可手动恢复。`
      : `Quarantine this file?\n${fullPath}\nIt will be renamed to .quarantined-by-lurus-switch.<timestamp> and can be manually restored later.`)) return
    setQuarantining(fullPath)
    try {
      const newPath = await QuarantineFile(fullPath)
      toast('success', isZh ? `已隔离到 ${newPath}` : `Quarantined to ${newPath}`)
      // Re-run audit to refresh findings.
      if (report) {
        const r = await AuditRepo(report.path)
        setReport(r)
      }
    } catch (e) {
      toast('error', String(e))
    } finally {
      setQuarantining(null)
    }
  }

  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) { reset(); onClose() } }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50 animate-in fade-in-0" />
        <Dialog.Content
          className="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-full max-w-2xl max-h-[85vh] flex flex-col bg-card border border-border rounded-xl shadow-2xl z-50 animate-in fade-in-0 zoom-in-95"
          aria-describedby={undefined}
        >
          <div className="flex items-center justify-between px-4 py-3 border-b border-border">
            <Dialog.Title className="flex items-center gap-2 text-sm font-semibold">
              <ShieldAlert className="h-4 w-4 text-primary" />
              <span>{isZh ? '仓库信任审计' : 'Repo Trust Audit'}</span>
              <span className="text-[10px] text-muted-foreground/70 font-normal">
                {isZh ? '(扫描 .claude/.codex/.gemini 等仓库级覆盖)' : '(scans .claude/.codex/.gemini repo-level overrides)'}
              </span>
            </Dialog.Title>
            <button
              type="button"
              onClick={onClose}
              className="h-7 w-7 inline-flex items-center justify-center rounded hover:bg-muted text-muted-foreground"
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-4 space-y-3">
            {!report && (
              <>
                <div className="rounded-md border border-border bg-muted/30 p-3 text-xs leading-relaxed">
                  <p className="text-foreground/90 mb-1.5 font-medium">
                    {isZh
                      ? '为什么要做这一步？'
                      : 'Why does this matter?'}
                  </p>
                  <p className="text-muted-foreground">
                    {isZh
                      ? '克隆别人仓库时，对方的 .claude/settings.json 可能把你的 ANTHROPIC_BASE_URL 重定向到攻击者地址（CVE-2026-21852），下一次启动 Claude Code 就会泄露你的 API key。建议在打开陌生仓库前先扫一下。'
                      : 'When you clone someone\'s repo, their .claude/settings.json can silently redirect ANTHROPIC_BASE_URL to an attacker host (CVE-2026-21852). The next Claude Code launch leaks your API key. Audit before opening unfamiliar repos.'}
                  </p>
                </div>

                <div className="space-y-2">
                  <button
                    type="button"
                    onClick={handlePick}
                    disabled={scanning}
                    className="w-full inline-flex items-center justify-center gap-2 px-3 py-2 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-50"
                  >
                    {scanning ? <Loader2 className="h-4 w-4 animate-spin" /> : <FolderOpen className="h-4 w-4" />}
                    {isZh ? '选择目录扫描' : 'Pick a directory to audit'}
                  </button>
                  <div className="text-[10px] text-center text-muted-foreground/70 py-1">
                    {isZh ? '或粘贴路径' : 'Or paste a path'}
                  </div>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={pathInput}
                      onChange={(e) => setPathInput(e.target.value)}
                      placeholder={isZh ? '例如 C:\\Users\\you\\projects\\some-repo' : 'e.g. /home/you/projects/some-repo'}
                      className="flex-1 px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
                    />
                    <button
                      type="button"
                      onClick={handleManualPath}
                      disabled={scanning || !pathInput.trim()}
                      className="px-3 py-1.5 rounded-md border border-border text-xs hover:bg-muted disabled:opacity-50"
                    >
                      {isZh ? '扫描' : 'Audit'}
                    </button>
                  </div>
                </div>
              </>
            )}

            {report && <ReportView
              report={report}
              isZh={isZh}
              quarantining={quarantining}
              onQuarantine={handleQuarantine}
              onScanAnother={reset}
            />}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

function ReportView({
  report, isZh, quarantining, onQuarantine, onScanAnother,
}: {
  report: repoaudit.AuditReport
  isZh: boolean
  quarantining: string | null
  onQuarantine: (path: string) => void
  onScanAnother: () => void
}) {
  const verdict = (report.verdict ?? 'safe') as keyof typeof VERDICT_TREATMENT
  const tx = VERDICT_TREATMENT[verdict] ?? VERDICT_TREATMENT.safe
  const VerdictIcon = tx.Icon
  const findings = report.findings ?? []
  const filesFound = report.filesFound ?? []

  return (
    <div className="space-y-3">
      <div className={cn('rounded-md border p-3 flex items-start gap-3', tx.bg, tx.border)}>
        <VerdictIcon className={cn('h-5 w-5 shrink-0 mt-0.5', tx.color)} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className={cn('text-sm font-semibold', tx.color)}>{isZh ? tx.labelZh : tx.labelEn}</span>
            <span className="text-[10px] text-muted-foreground/70 font-mono">
              {findings.length} {isZh ? '项发现' : 'findings'} · {filesFound.length} {isZh ? '个文件' : 'files'}
            </span>
          </div>
          <p className="text-[11px] text-muted-foreground mt-1 font-mono break-all">{report.path}</p>
        </div>
        <button
          type="button"
          onClick={onScanAnother}
          className="text-[11px] text-muted-foreground hover:text-foreground px-2 py-1 rounded border border-border/60"
        >
          {isZh ? '换一个' : 'Scan another'}
        </button>
      </div>

      {filesFound.length === 0 && (
        <div className="rounded-md border border-border bg-muted/20 p-4 text-center text-sm text-muted-foreground">
          {isZh
            ? '该目录没有发现任何 AI CLI 配置文件——可以放心打开。'
            : 'No AI-CLI config files found in this directory — safe to open.'}
        </div>
      )}

      {findings.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
            {isZh ? '发现项' : 'Findings'}
          </h4>
          {findings.map((f, i) => {
            const sev = (f.severity ?? 'info') as keyof typeof SEVERITY_TREATMENT
            const stx = SEVERITY_TREATMENT[sev] ?? SEVERITY_TREATMENT.info
            const isQ = quarantining === f.fullPath
            return (
              <div key={i} className={cn('rounded-md border p-3 text-xs', stx.bg, stx.border)}>
                <div className="flex items-start justify-between gap-3">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <FileWarning className={cn('h-3.5 w-3.5', stx.color)} />
                      <span className={cn('font-medium', stx.color)}>
                        {isZh ? f.issueZh : f.issueEn}
                      </span>
                    </div>
                    <div className="mt-1 text-[10px] text-muted-foreground font-mono break-all">
                      {f.file}
                      {f.field && f.field !== '(file)' && <span className="text-muted-foreground/60"> · {f.field}</span>}
                    </div>
                    {f.detailValue && (
                      <div className="mt-1.5 px-2 py-1 rounded bg-background/50 text-[10px] font-mono break-all">
                        {f.detailValue}
                      </div>
                    )}
                  </div>
                  {f.suggestedAction === 'quarantine' || f.suggestedAction === 'delete' ? (
                    <button
                      type="button"
                      onClick={() => onQuarantine(f.fullPath)}
                      disabled={isQ}
                      className="shrink-0 inline-flex items-center gap-1 px-2 py-1 rounded text-[10px] border border-red-500/30 text-red-300 hover:bg-red-500/10 disabled:opacity-50"
                      title={isZh ? '隔离该文件（重命名加后缀）' : 'Quarantine this file (rename with suffix)'}
                    >
                      {isQ ? <Loader2 className="h-3 w-3 animate-spin" /> : <Archive className="h-3 w-3" />}
                      {isZh ? '隔离' : 'Quarantine'}
                    </button>
                  ) : (
                    <span className="shrink-0 inline-flex items-center gap-1 px-2 py-1 rounded text-[10px] text-muted-foreground border border-border/60">
                      <AlertTriangle className="h-3 w-3" />
                      {isZh ? '人工核实' : 'Review'}
                    </span>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}

      <div className="pt-2 text-[10px] text-muted-foreground/60 leading-relaxed">
        <ExternalLink className="h-3 w-3 inline mr-1 -mt-0.5" />
        {isZh
          ? '参考：CVE-2026-21852 / Reddit "rm -rf ~/" 事故。Switch 不会自动修改任何文件，所有操作以可逆方式进行（隔离即重命名）。'
          : 'Refs: CVE-2026-21852 / Reddit "rm -rf ~/" incident. Switch never modifies files automatically — quarantine is reversible (just renames).'}
      </div>
    </div>
  )
}
