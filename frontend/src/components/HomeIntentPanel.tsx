import { Plug, Repeat, Shield, Wallet, Package, Bot, ShieldAlert } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useConfigStore } from '../stores/configStore'
import { useToastStore } from '../stores/toastStore'
import { useRepoAuditStore } from '../stores/repoAuditStore'
import { useBashGuardStore } from '../stores/bashGuardStore'
import { useBudgetStore } from '../stores/budgetStore'
import { LaunchTool } from '../../wailsjs/go/main/App'

// Verb-first entry points so users discover features by stating intent
// ("I want to add a relay") rather than navigating by category. Sits at
// the top of HomePage; routes click through to the relevant page +
// sub-tab + (when known) scrolls to the relevant section anchor.
interface IntentCard {
  id: string
  icon: typeof Plug
  // Bilingual label — both shown so the same component works in zh/en
  // without a re-render on language switch.
  zh: { title: string; desc: string }
  en: { title: string; desc: string }
  // Tailwind accent colour used for the icon and ring on hover.
  accent: string
  onClick: () => void | Promise<void>
}

export function HomeIntentPanel() {
  const { t, i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh') ?? true
  const setActiveTool = useConfigStore((s) => s.setActiveTool)
  const setSubTab = useConfigStore((s) => s.setSubTab)
  const toast = useToastStore((s) => s.addToast)
  const openRepoAudit = useRepoAuditStore((s) => s.setOpen)
  const openBashGuard = useBashGuardStore((s) => s.setOpen)
  const openBudget = useBudgetStore((s) => s.setOpen)

  // After navigating, give React one paint to mount the destination page,
  // then scroll the named section into view. 80ms covers the Vite/HMR
  // worst case on a slow machine; faster machines feel instant.
  const scrollToAfterNav = (sectionId: string) => {
    setTimeout(() => {
      document.getElementById(sectionId)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }, 80)
  }

  const cards: IntentCard[] = [
    {
      id: 'add-relay',
      icon: Plug,
      zh: { title: '接中转站', desc: '把你的工具走到第三方代理或本地网关' },
      en: { title: 'Add a relay', desc: 'Route your CLI traffic through a proxy or local gateway' },
      accent: 'text-cyan-400',
      onClick: () => {
        setActiveTool('gateway')
        setSubTab('gateway', 'relay')
      },
    },
    {
      id: 'switch-provider',
      icon: Repeat,
      zh: { title: '换服务商', desc: '一键切换 Anthropic / OpenAI / 第三方供应商' },
      en: { title: 'Switch provider', desc: 'One-click swap between Anthropic / OpenAI / third-party providers' },
      accent: 'text-amber-400',
      onClick: () => {
        setActiveTool('tools')
        // The PresetSelector lives at the top of the form view — landing
        // on Tools is enough; user picks a preset.
      },
    },
    {
      id: 'bash-guard',
      icon: Shield,
      zh: { title: '启用 Bash-Guard', desc: '拦截 Claude 跑 rm -rf / 等危险命令（防 issue #10077 类事故）' },
      en: { title: 'Enable Bash-Guard', desc: 'Intercept dangerous Claude shell calls (rm -rf /, curl|sh, DROP DATABASE)' },
      accent: 'text-red-400',
      onClick: () => openBashGuard(true),
    },
    {
      id: 'audit-repo',
      icon: ShieldAlert,
      zh: { title: '审计仓库', desc: '扫描陌生仓库的 .claude/.codex 配置覆盖（CVE-2026-21852 防护）' },
      en: { title: 'Audit a repo', desc: 'Scan untrusted repos for .claude/.codex config overrides (CVE-2026-21852 defense)' },
      accent: 'text-rose-400',
      onClick: () => openRepoAudit(true),
    },
    {
      id: 'budget-wall',
      icon: Wallet,
      zh: { title: '设置花费上限', desc: '在网关层硬切断 token 用量，防 $1,600 一晚账单' },
      en: { title: 'Set spend cap', desc: 'Hard-cut token usage at the gateway — defends against the $1,600 overnight burn' },
      accent: 'text-emerald-400',
      onClick: () => openBudget(true),
    },
    {
      id: 'install-tool',
      icon: Package,
      zh: { title: '装新工具', desc: '安装 Claude / Codex / Gemini / Pico|Null|Zero|OpenClaw' },
      en: { title: 'Install a tool', desc: 'Install Claude / Codex / Gemini / Pico|Null|Zero|OpenClaw' },
      accent: 'text-violet-400',
      onClick: () => setActiveTool('tools'),
    },
    {
      id: 'launch-claude',
      icon: Bot,
      zh: { title: '跑 Claude Code', desc: '直接启动 CLI（确保已安装并配置）' },
      en: { title: 'Launch Claude Code', desc: 'Start the CLI now (must be installed & configured)' },
      accent: 'text-purple-400',
      onClick: async () => {
        try {
          await LaunchTool('claude', [])
          toast('success', isZh ? 'Claude Code 已启动' : 'Claude Code launched')
        } catch (e) {
          toast('error', String(e))
        }
      },
    },
  ]

  return (
    <section className="rounded-lg border border-border bg-card/40 p-4">
      <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3 flex items-center gap-2">
        <span className="text-foreground">{t('home.intent.title', '你想做什么？')}</span>
        <span className="text-[10px] text-muted-foreground/70 font-normal">
          {t('home.intent.titleEn', 'What do you want to do?')}
        </span>
      </h3>
      <div className="grid grid-cols-2 lg:grid-cols-3 gap-2">
        {cards.map((c) => {
          const Icon = c.icon
          const text = isZh ? c.zh : c.en
          return (
            <button
              key={c.id}
              onClick={c.onClick}
              className={cn(
                'group flex items-start gap-3 p-3 rounded-md border border-border/60 text-left',
                'transition-colors hover:bg-muted/40 hover:border-border',
              )}
            >
              <Icon className={cn('h-5 w-5 shrink-0 mt-0.5', c.accent)} />
              <div className="min-w-0 flex-1">
                <div className="text-sm font-medium truncate">{text.title}</div>
                <div className="text-[11px] text-muted-foreground/80 leading-snug mt-0.5 line-clamp-2">{text.desc}</div>
              </div>
            </button>
          )
        })}
      </div>
    </section>
  )
}
