import { useState } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import {
  Shield, ShieldAlert, Wallet, Activity, Sparkles, ChevronLeft, ChevronRight,
  X, Rocket,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useBashGuardStore } from '../stores/bashGuardStore'
import { useRepoAuditStore } from '../stores/repoAuditStore'
import { useBudgetStore } from '../stores/budgetStore'
import { GetAppSettings, SaveAppSettings } from '../../wailsjs/go/main/App'
import { appconfig } from '../../wailsjs/go/models'

interface FeatureTourModalProps {
  open: boolean
  onClose: () => void
}

interface Slide {
  id: string
  icon: typeof Shield
  accent: string
  zh: { title: string; tagline: string; body: string; cta?: string }
  en: { title: string; tagline: string; body: string; cta?: string }
  // What clicking the CTA does (open the relevant feature). Optional —
  // the welcome / closing slides have no CTA.
  ctaAction?: () => void
}

export function FeatureTourModal({ open, onClose }: FeatureTourModalProps) {
  const { i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh') ?? true
  const [step, setStep] = useState(0)

  const openBashGuard = useBashGuardStore((s) => s.setOpen)
  const openRepoAudit = useRepoAuditStore((s) => s.setOpen)
  const openBudget = useBudgetStore((s) => s.setOpen)

  const slides: Slide[] = [
    {
      id: 'welcome',
      icon: Sparkles,
      accent: 'text-violet-400 bg-violet-500/10 border-violet-500/30',
      zh: {
        title: '欢迎使用 Lurus Switch',
        tagline: '本地 AI 网关 · 守在你和 CLI 之间',
        body: '它不是又一个配置切换器。Switch 站在你的 AI CLI 和外网之间，做一次它们做不到的事——主动拦危险命令、主动锁花费、主动审陌生仓库的隐藏配置。这一轮带你看 4 个直接解决 2026 年高频痛点的能力。',
      },
      en: {
        title: 'Welcome to Lurus Switch',
        tagline: 'Local AI gateway · Stands between you and your CLI',
        body: 'Not just another config swapper. Switch sits between your AI CLI and the outside world, doing things they cannot — actively blocking dangerous commands, actively capping spend, actively auditing unknown repos. The next 4 slides show capabilities that directly defuse 2026\'s top horror stories.',
      },
    },
    {
      id: 'bashguard',
      icon: Shield,
      accent: 'text-red-400 bg-red-500/10 border-red-500/30',
      zh: {
        title: 'Bash-Guard 危险命令防护',
        tagline: '在 Claude 跑命令前先查',
        body: '内置 18 条规则覆盖 rm -rf / · ~ 目录技巧 · curl|sh · DROP DATABASE · aws s3 rb --force · format C: 等场景。事故案例：Reddit 的 byteiota 家目录被清空、Claude Issue #10077 跑了 rm -rf /、Replit AI 删了 SaaStr 生产库。一键启用后，Claude 触发危险命令时 Switch 直接拒绝并记录到日志。',
        cta: '现在启用',
      },
      en: {
        title: 'Bash-Guard',
        tagline: 'Block dangerous shell commands BEFORE they execute',
        body: '18 built-in rules covering rm -rf /, the ~ directory trick, curl|sh, DROP DATABASE, aws s3 rb --force, format C:. Real incidents: Reddit byteiota home wipe, Claude #10077 (rm -rf /), Replit AI wiped SaaStr\'s prod DB. One click enables a PreToolUse hook in ~/.claude/settings.json — Switch blocks at exit code 2 and logs every attempt.',
        cta: 'Enable now',
      },
      ctaAction: () => openBashGuard(true),
    },
    {
      id: 'budget',
      icon: Wallet,
      accent: 'text-emerald-400 bg-emerald-500/10 border-emerald-500/30',
      zh: {
        title: 'Budget Wall 花费硬上限',
        tagline: '在网关层硬切断 token 用量',
        body: 'r/ClaudeCode 上"$1,600 一晚账单"是高频帖。Claude Code 平均比 Codex 多用 4× tokens，2026 年 3 月一次缓存 bug 还把账单乘了 10 倍。Switch 网关层每次调用前先查累计 token：超 daily 或 session 上限，直接 429 拒绝并提示重置。Dashboard 是事后看，Switch 是事前拦。',
        cta: '设置上限',
      },
      en: {
        title: 'Budget Wall',
        tagline: 'Hard-cut token spend at the gateway',
        body: 'r/ClaudeCode is full of "$1,600 overnight" posts. Claude Code burns ~4× tokens of Codex; the March 2026 caching bug 10×\'d bills. Switch checks cumulative tokens BEFORE forwarding each request — over daily or session cap = instant 429 with a reset hint. Dashboards see post-hoc; Switch blocks before the spend.',
        cta: 'Set a cap',
      },
      ctaAction: () => openBudget(true),
    },
    {
      id: 'repoaudit',
      icon: ShieldAlert,
      accent: 'text-rose-400 bg-rose-500/10 border-rose-500/30',
      zh: {
        title: '仓库信任审计',
        tagline: '克隆陌生仓库前先扫一眼',
        body: 'CVE-2026-21852：恶意仓库的 .claude/settings.json 把 ANTHROPIC_BASE_URL 重定向到攻击者地址，下次启动 Claude Code 就泄露你的 API key。Switch 扫描 6 类配置文件 + 项目级 MCP 服务器 + AGENTS.md / CLAUDE.md 上下文文件，逐项给出 risky/caution/info 评级，可一键隔离（重命名加 sentinel 后缀，可手动恢复）。',
        cta: '审计一个仓库',
      },
      en: {
        title: 'Repo Trust Audit',
        tagline: 'Scan before you cd into an unknown repo',
        body: 'CVE-2026-21852: a malicious repo\'s .claude/settings.json silently redirects ANTHROPIC_BASE_URL to an attacker host — the next Claude Code launch leaks your key. Switch audits 6 config-file types + repo-level MCP servers + AGENTS.md/CLAUDE.md context, rates each finding risky/caution/info, and offers one-click quarantine (rename with sentinel — reversible).',
        cta: 'Audit a repo',
      },
      ctaAction: () => openRepoAudit(true),
    },
    {
      id: 'transparency',
      icon: Activity,
      accent: 'text-cyan-400 bg-cyan-500/10 border-cyan-500/30',
      zh: {
        title: '实时透明度',
        tagline: '看得见 Switch 在干嘛',
        body: '右下角的 Activity 面板会实时显示长操作的进度——"安装 Claude Code 25%"、"启动网关"、"修复配置 3/7"，不再是空转的 spinner。Home 页还有 Runtime Status 卡片：每个 CLI 的端点、模型、连接状态（HTTP HEAD 探活）+ 进程是否在跑，30 秒/次自动刷新。',
      },
      en: {
        title: 'Live transparency',
        tagline: 'See what Switch is doing — in real time',
        body: 'The bottom-right Activity pane streams progress for long ops — "Installing Claude Code 25%", "Starting gateway", "Apply optimizations 3/7" — no more black-box spinners. Home also has a Runtime Status card: per-CLI endpoint, model, reachability probe (HTTP HEAD), and process state, refreshing every 30s.',
      },
    },
    {
      id: 'closing',
      icon: Rocket,
      accent: 'text-primary bg-primary/10 border-primary/30',
      zh: {
        title: '准备好了',
        tagline: '随时按 Ctrl+K 召唤命令面板',
        body: '本指引以后通过命令面板搜 "tour" 重看。所有功能都在 Home 页 7 张意图卡里，按你的意图（"接中转站"、"启用 Bash-Guard"、"设置花费上限"...）一键直达对应流程。开始用吧。',
      },
      en: {
        title: 'You\'re set',
        tagline: 'Press Ctrl+K anytime for the command palette',
        body: 'Re-run this tour later via the command palette ("tour"). All features live on the Home page as 7 intent cards — describe what you want ("add relay", "enable bash-guard", "set spend cap"...) and click straight through. Have at it.',
      },
    },
  ]

  const current = slides[step]
  const Icon = current.icon
  const isLast = step === slides.length - 1
  const text = isZh ? current.zh : current.en

  const close = async () => {
    // Mark the tour as seen so the auto-launch on next start doesn't fire.
    // Best-effort: we don't want a transient SaveAppSettings failure to
    // block the close, so swallow errors (the user can re-trigger via
    // the command palette).
    try {
      const cur = await GetAppSettings()
      const merged = appconfig.AppSettings.createFrom({ ...cur, featureTourSeen: true })
      await SaveAppSettings(merged)
    } catch { /* ignore */ }
    onClose()
  }

  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) close() }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/70 z-50 animate-in fade-in-0" />
        <Dialog.Content
          className="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-full max-w-xl bg-card border border-border rounded-xl shadow-2xl z-50 animate-in fade-in-0 zoom-in-95"
          aria-describedby={undefined}
        >
          <button
            onClick={close}
            className="absolute top-3 right-3 h-7 w-7 inline-flex items-center justify-center rounded hover:bg-muted text-muted-foreground"
            title={isZh ? '关闭（以后命令面板搜 tour 重看）' : 'Close (re-run via command palette > tour)'}
          >
            <X className="h-4 w-4" />
          </button>

          <div className="p-8">
            <div className={cn('h-14 w-14 rounded-full border-2 flex items-center justify-center mb-5', current.accent)}>
              <Icon className="h-7 w-7" />
            </div>
            <Dialog.Title className="text-xl font-semibold mb-1">{text.title}</Dialog.Title>
            <p className="text-sm text-muted-foreground mb-4">{text.tagline}</p>
            <p className="text-sm text-foreground/90 leading-relaxed">{text.body}</p>

            {text.cta && current.ctaAction && (
              <button
                onClick={() => { current.ctaAction!() }}
                className={cn(
                  'mt-5 px-4 py-2 rounded-md text-sm font-medium border transition-colors',
                  current.accent,
                  'hover:bg-opacity-20',
                )}
              >
                {text.cta} →
              </button>
            )}
          </div>

          {/* Footer: progress dots + nav */}
          <div className="flex items-center justify-between px-6 py-3 border-t border-border bg-muted/20">
            <div className="flex items-center gap-1.5">
              {slides.map((_, i) => (
                <span
                  key={i}
                  className={cn(
                    'h-1.5 rounded-full transition-all cursor-pointer',
                    i === step ? 'w-6 bg-primary' : 'w-1.5 bg-muted hover:bg-muted-foreground/40',
                  )}
                  onClick={() => setStep(i)}
                />
              ))}
            </div>
            <div className="flex items-center gap-2">
              {step > 0 && (
                <button
                  onClick={() => setStep(step - 1)}
                  className="inline-flex items-center gap-1 px-3 py-1.5 rounded-md text-xs hover:bg-muted text-muted-foreground"
                >
                  <ChevronLeft className="h-3.5 w-3.5" />
                  {isZh ? '上一页' : 'Back'}
                </button>
              )}
              {!isLast ? (
                <button
                  onClick={() => setStep(step + 1)}
                  className="inline-flex items-center gap-1 px-4 py-1.5 rounded-md text-xs font-medium bg-primary text-primary-foreground hover:bg-primary/90"
                >
                  {isZh ? '下一页' : 'Next'}
                  <ChevronRight className="h-3.5 w-3.5" />
                </button>
              ) : (
                <button
                  onClick={close}
                  className="inline-flex items-center gap-1 px-4 py-1.5 rounded-md text-xs font-medium bg-primary text-primary-foreground hover:bg-primary/90"
                >
                  {isZh ? '完成' : 'Done'}
                </button>
              )}
            </div>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
