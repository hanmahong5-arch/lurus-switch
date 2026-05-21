import { useEffect, useState, useCallback } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import {
  ShieldCheck, Shield, X, Loader2, AlertTriangle, CheckCircle2,
  Power, FlaskConical, History, ExternalLink,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { formatLocal } from '../lib/formatTime'
import { useToastStore } from '../stores/toastStore'
import {
  BashGuardListRules, BashGuardClaudeStatus, BashGuardInstallClaude,
  BashGuardUninstallClaude, BashGuardTestCommand, BashGuardRecentBlocks,
} from '../../wailsjs/go/main/App'
import type { bashguard } from '../../wailsjs/go/models'

interface BashGuardModalProps {
  open: boolean
  onClose: () => void
}

const SEVERITY_COLOR = {
  critical: 'text-red-400 border-red-500/30 bg-red-500/10',
  high: 'text-orange-400 border-orange-500/30 bg-orange-500/10',
  medium: 'text-amber-400 border-amber-500/30 bg-amber-500/10',
}

type Tab = 'overview' | 'rules' | 'test' | 'log'

export function BashGuardModal({ open, onClose }: BashGuardModalProps) {
  const { t, i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh') ?? true
  const toast = useToastStore((s) => s.addToast)
  const [tab, setTab] = useState<Tab>('overview')
  const [rules, setRules] = useState<bashguard.Rule[]>([])
  const [status, setStatus] = useState<bashguard.HookInstallStatus | null>(null)
  const [blocks, setBlocks] = useState<bashguard.BlockEntry[]>([])
  const [busy, setBusy] = useState(false)

  const refresh = useCallback(async () => {
    try {
      const [rs, st, bl] = await Promise.all([
        BashGuardListRules(),
        BashGuardClaudeStatus(),
        BashGuardRecentBlocks(50),
      ])
      setRules(rs ?? [])
      setStatus(st)
      setBlocks(bl ?? [])
    } catch (e) {
      toast('error', String(e))
    }
  }, [toast])

  useEffect(() => {
    if (open) refresh()
  }, [open, refresh])

  const toggleHook = async () => {
    if (!status) return
    setBusy(true)
    try {
      if (status.installed) {
        await BashGuardUninstallClaude()
        toast('success', isZh ? 'Bash-Guard 已停用' : 'Bash-Guard disabled')
      } else {
        await BashGuardInstallClaude()
        toast('success', isZh ? 'Bash-Guard 已启用，所有 Claude Code 的 Bash 命令将经过审查' : 'Bash-Guard enabled — all Claude Code Bash commands will be reviewed')
      }
      await refresh()
    } catch (e) {
      toast('error', String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) onClose() }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50 animate-in fade-in-0" />
        <Dialog.Content
          className="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-full max-w-3xl max-h-[88vh] flex flex-col bg-card border border-border rounded-xl shadow-2xl z-50 animate-in fade-in-0 zoom-in-95"
          aria-describedby={undefined}
        >
          <div className="flex items-center justify-between px-4 py-3 border-b border-border">
            <Dialog.Title className="flex items-center gap-2 text-sm font-semibold">
              <Shield className="h-4 w-4 text-primary" />
              <span>{isZh ? 'Bash-Guard 危险命令防护' : 'Bash-Guard dangerous-command shield'}</span>
              <span className="text-[10px] text-muted-foreground/70 font-normal">
                {isZh ? '(在 Claude 跑命令前先查危险模式)' : '(intercepts Claude\'s shell calls before execution)'}
              </span>
            </Dialog.Title>
            <button onClick={onClose} className="h-7 w-7 inline-flex items-center justify-center rounded hover:bg-muted text-muted-foreground">
              <X className="h-4 w-4" />
            </button>
          </div>

          <div className="flex border-b border-border px-2 text-xs">
            {([
              { id: 'overview', icon: Power, zh: '概览', en: 'Overview' },
              { id: 'rules', icon: Shield, zh: `规则 (${rules.length})`, en: `Rules (${rules.length})` },
              { id: 'test', icon: FlaskConical, zh: '测命令', en: 'Test command' },
              { id: 'log', icon: History, zh: `拦截日志 (${blocks.length})`, en: `Block log (${blocks.length})` },
            ] as const).map((tabDef) => {
              const Icon = tabDef.icon
              const active = tab === tabDef.id
              return (
                <button
                  key={tabDef.id}
                  onClick={() => setTab(tabDef.id)}
                  className={cn(
                    'inline-flex items-center gap-1.5 px-3 py-2 -mb-px border-b-2 transition-colors',
                    active ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground',
                  )}
                >
                  <Icon className="h-3.5 w-3.5" />
                  {isZh ? tabDef.zh : tabDef.en}
                </button>
              )
            })}
          </div>

          <div className="flex-1 overflow-y-auto p-4">
            {tab === 'overview' && <OverviewTab status={status} busy={busy} onToggle={toggleHook} isZh={isZh} />}
            {tab === 'rules' && <RulesTab rules={rules} isZh={isZh} />}
            {tab === 'test' && <TestTab isZh={isZh} />}
            {tab === 'log' && <LogTab blocks={blocks} isZh={isZh} />}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

function OverviewTab({
  status, busy, onToggle, isZh,
}: {
  status: bashguard.HookInstallStatus | null; busy: boolean; onToggle: () => void; isZh: boolean
}) {
  const installed = status?.installed ?? false
  return (
    <div className="space-y-4">
      <div className={cn(
        'rounded-md border p-4 flex items-start gap-3',
        installed ? 'border-emerald-500/30 bg-emerald-500/10' : 'border-amber-500/30 bg-amber-500/10',
      )}>
        {installed
          ? <ShieldCheck className="h-6 w-6 text-emerald-400 shrink-0" />
          : <AlertTriangle className="h-6 w-6 text-amber-400 shrink-0" />}
        <div className="flex-1 min-w-0">
          <div className={cn('font-semibold text-sm', installed ? 'text-emerald-300' : 'text-amber-300')}>
            {installed
              ? (isZh ? 'Claude Code 已受 Bash-Guard 保护' : 'Claude Code is protected by Bash-Guard')
              : (isZh ? 'Claude Code 未启用 Bash-Guard' : 'Bash-Guard is not enabled for Claude Code')}
          </div>
          <p className="text-xs text-muted-foreground mt-1 leading-relaxed">
            {installed
              ? (isZh
                ? 'Claude Code 每次执行 Bash 工具前会先调用 Switch 比对危险命令规则，命中即拒绝并记录到日志。规则覆盖 rm -rf /、~ 目录技巧、curl|sh、DROP DATABASE、aws s3 rb --force 等 18 类高危模式。'
                : 'Each time Claude Code runs a Bash tool, Switch checks the command against the deny-list and blocks matches. Defaults cover rm -rf /, the ~ directory trick, curl|sh, DROP DATABASE, aws s3 rb --force, and 14 more.')
              : (isZh
                ? '点击下方按钮启用——Switch 会向 ~/.claude/settings.json 写入一条 PreToolUse 钩子，把所有 Bash 工具调用先转给 Switch 审核。可随时关闭，仅修改你自己的 settings.json，不影响其他工具。'
                : 'Click below to enable — Switch will write a PreToolUse hook into ~/.claude/settings.json that routes every Bash tool call through Switch for review. Disable anytime; only your settings.json is touched.')}
          </p>
          {status?.configPath && (
            <p className="text-[10px] text-muted-foreground/60 font-mono mt-1.5 break-all">
              {status.configPath}
            </p>
          )}
        </div>
      </div>

      <button
        onClick={onToggle}
        disabled={busy || !status}
        className={cn(
          'w-full inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-md text-sm font-medium transition-colors disabled:opacity-50',
          installed
            ? 'bg-red-600 hover:bg-red-500 text-white'
            : 'bg-emerald-600 hover:bg-emerald-500 text-white',
        )}
      >
        {busy
          ? <Loader2 className="h-4 w-4 animate-spin" />
          : installed ? <Power className="h-4 w-4" /> : <ShieldCheck className="h-4 w-4" />}
        {installed
          ? (isZh ? '停用 Bash-Guard' : 'Disable Bash-Guard')
          : (isZh ? '启用 Bash-Guard' : 'Enable Bash-Guard')}
      </button>

      <div className="rounded-md border border-border bg-muted/20 p-3 text-[11px] text-muted-foreground leading-relaxed">
        <p className="font-medium text-foreground/80 mb-1">{isZh ? '为什么这是必要的？' : 'Why does this matter?'}</p>
        <ul className="space-y-1 list-disc pl-4">
          <li>{isZh ? '2025-12 一名 Claude 用户用 Reddit 记录了 Claude Code 把家目录清空的事故' : '2025-12: a Reddit user documented Claude Code wiping their home directory'}</li>
          <li>{isZh ? '2025-07 Replit AI 删了 SaaStr 的生产数据库（明确指示了不准动）' : '2025-07: Replit AI wiped SaaStr\'s production DB despite explicit "don\'t touch prod"'}</li>
          <li>{isZh ? 'Claude Code Issue #10077 (Wolak Incident): 没加 --dangerously-skip-permissions 也跑了 rm -rf /' : 'Claude Code issue #10077 (Wolak): rm -rf / executed without --dangerously-skip-permissions'}</li>
        </ul>
      </div>
    </div>
  )
}

function RulesTab({ rules, isZh }: { rules: bashguard.Rule[]; isZh: boolean }) {
  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground mb-3">
        {isZh
          ? `当前内置 ${rules.length} 条规则，按严重度排序。命令命中第一条规则即被拦截。`
          : `${rules.length} built-in rules, ordered by severity. The first matching rule blocks the command.`}
      </p>
      {rules.map((r) => (
        <div key={r.id} className={cn('rounded-md border p-3 text-xs',
          SEVERITY_COLOR[r.severity as keyof typeof SEVERITY_COLOR] ?? 'border-border bg-muted/20',
        )}>
          <div className="flex items-center gap-2">
            <span className="font-mono text-[10px] uppercase tracking-wider opacity-80">{r.severity}</span>
            <span className="font-mono text-[11px] opacity-70">{r.id}</span>
          </div>
          <div className="mt-1 text-foreground">{isZh ? r.reasonZh : r.reasonEn}</div>
          <div className="mt-1 font-mono text-[10px] text-muted-foreground/80 break-all">
            {r.pattern}
          </div>
          {r.reference && (
            <a href={r.reference} target="_blank" rel="noreferrer"
              className="mt-1 inline-flex items-center gap-1 text-[10px] text-cyan-400 hover:underline">
              <ExternalLink className="h-2.5 w-2.5" />
              {r.reference}
            </a>
          )}
        </div>
      ))}
    </div>
  )
}

function TestTab({ isZh }: { isZh: boolean }) {
  const [cmd, setCmd] = useState('')
  const [result, setResult] = useState<bashguard.MatchResult | null>(null)
  const [running, setRunning] = useState(false)

  const run = async () => {
    if (!cmd.trim()) return
    setRunning(true)
    try {
      const r = await BashGuardTestCommand(cmd)
      setResult(r)
    } finally {
      setRunning(false)
    }
  }

  return (
    <div className="space-y-3">
      <p className="text-xs text-muted-foreground">
        {isZh ? '粘贴一条命令，Switch 会按当前规则评估，但不会真正执行。' : 'Paste a command — Switch evaluates against the rules without executing it.'}
      </p>
      <textarea
        value={cmd}
        onChange={(e) => setCmd(e.target.value)}
        placeholder={isZh ? '例如：rm -rf ~/' : 'e.g. rm -rf ~/'}
        rows={3}
        className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono resize-y"
      />
      <button
        onClick={run}
        disabled={running || !cmd.trim()}
        className="inline-flex items-center gap-2 px-3 py-1.5 rounded-md bg-primary text-primary-foreground text-xs font-medium hover:bg-primary/90 disabled:opacity-50"
      >
        {running ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <FlaskConical className="h-3.5 w-3.5" />}
        {isZh ? '评估' : 'Evaluate'}
      </button>

      {result && (
        <div className={cn('rounded-md border p-3 text-xs',
          result.allowed
            ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-200'
            : 'border-red-500/30 bg-red-500/10 text-red-200',
        )}>
          <div className="flex items-center gap-2 font-semibold">
            {result.allowed
              ? <><CheckCircle2 className="h-4 w-4 text-emerald-400" /> {isZh ? '放行' : 'Allowed'}</>
              : <><AlertTriangle className="h-4 w-4 text-red-400" /> {isZh ? `拦截 — ${result.rule?.id}` : `Blocked — ${result.rule?.id}`}</>}
          </div>
          {result.rule && (
            <div className="mt-1.5 space-y-1">
              <div>{isZh ? result.rule.reasonZh : result.rule.reasonEn}</div>
              <div className="font-mono text-[10px] opacity-70">{result.rule.pattern}</div>
            </div>
          )}
          <div className="mt-1.5 font-mono text-[10px] opacity-70 break-all">
            {isZh ? '归一化后' : 'Normalized'}: {result.normalizedCommand}
          </div>
        </div>
      )}
    </div>
  )
}

function LogTab({ blocks, isZh }: { blocks: bashguard.BlockEntry[]; isZh: boolean }) {
  if (blocks.length === 0) {
    return (
      <div className="text-center text-sm text-muted-foreground py-8">
        {isZh ? '还没有拦截记录。一切平静。' : 'Nothing blocked yet — all quiet.'}
      </div>
    )
  }
  return (
    <div className="space-y-2">
      {blocks.map((b, i) => (
        <div key={i} className={cn('rounded-md border p-3 text-xs',
          SEVERITY_COLOR[b.severity as keyof typeof SEVERITY_COLOR] ?? 'border-border bg-muted/20',
        )}>
          <div className="flex items-center gap-2 text-[10px] tabular-nums opacity-80">
            <span>{formatLocal(b.time)}</span>
            <span className="font-mono">·</span>
            <span className="font-mono uppercase tracking-wider">{b.severity}</span>
            <span className="font-mono">·</span>
            <span className="font-mono">{b.ruleId}</span>
            {b.tool && <><span className="font-mono">·</span><span>{b.tool}</span></>}
          </div>
          <div className="mt-1 font-mono text-foreground break-all">{b.command}</div>
          <div className="mt-1 text-muted-foreground">{b.reason}</div>
          {b.cwd && <div className="mt-1 text-[10px] text-muted-foreground/60 font-mono break-all">cwd: {b.cwd}</div>}
        </div>
      ))}
    </div>
  )
}
