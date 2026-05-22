import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Bot, ShieldCheck, Coins, Clock, Wrench, RefreshCw, AlertTriangle } from 'lucide-react'
import { useAgentTemplateStore, type AgentTemplate } from '../stores/agentTemplateStore'
import { Button, Card } from '../components/ui'

export function AgentTemplateGalleryPage() {
  const { t } = useTranslation()
  const { templates, loading, error, selectedId, load, select } = useAgentTemplateStore()

  useEffect(() => {
    void load()
  }, [load])

  const selected = templates.find(tpl => tpl.id === selectedId) ?? null

  return (
    <div className="h-full overflow-auto p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-xl font-semibold flex items-center gap-2">
            <Bot className="h-5 w-5 text-primary" />
            {t('agentGallery.title', 'Agent 模板库')}
          </h1>
          <p className="text-xs text-muted-foreground mt-1">
            {t('agentGallery.subtitle', '内置 5 类岗位 agent — 销售 / 支持 / 运维 / 财务 / 合规。点击查看 system prompt、capability 白名单、预算与 guardrails。')}
          </p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => void load()}
          disabled={loading}
          loading={loading}
          icon={!loading ? <RefreshCw className="h-3.5 w-3.5" /> : undefined}
        >
          {t('common.refresh', '刷新')}
        </Button>
      </div>

      {error && (
        <Card variant="default" className="mb-3 p-2 border-red-500/30 bg-red-500/10 text-red-400 text-xs flex items-center gap-2 font-mono">
          <AlertTriangle className="h-3.5 w-3.5" />
          {error}
        </Card>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 mb-4">
        {templates.map(tpl => {
          const isSelected = selectedId === tpl.id
          return (
            <Card
              key={tpl.id}
              as="button"
              variant={isSelected ? 'elevated' : 'default'}
              glow={isSelected}
              onClick={() => select(tpl.id)}
              className="text-left p-4"
            >
              <div className="flex items-start gap-3">
                <div className="text-2xl leading-none">{tpl.icon}</div>
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-semibold truncate">{tpl.displayName}</div>
                  <div className="text-[11px] font-mono text-muted-foreground mt-0.5 truncate tabular-nums">{tpl.id}</div>
                </div>
              </div>
              <div className="mt-3 flex flex-wrap gap-1">
                {(tpl.tags ?? []).slice(0, 3).map(tg => (
                  <span key={tg} className="px-1.5 py-0.5 rounded bg-card-recessed text-[10px] font-mono text-muted-foreground">{tg}</span>
                ))}
              </div>
              <div className="mt-3 grid grid-cols-2 gap-1.5 text-[11px] text-muted-foreground font-mono">
                <span className="flex items-center gap-1"><Wrench className="h-3 w-3" /> {tpl.toolType}</span>
                <span className="flex items-center gap-1 tabular-nums"><Coins className="h-3 w-3" /> ${tpl.budgetUsd}/{tpl.budgetPeriod}</span>
                <span className="flex items-center gap-1"><Clock className="h-3 w-3" /> {tpl.budgetPolicy}</span>
                <span className="flex items-center gap-1 tabular-nums"><ShieldCheck className="h-3 w-3" /> {tpl.capabilities.length} caps</span>
              </div>
            </Card>
          )
        })}
      </div>

      {selected && <TemplateDetail template={selected} />}
    </div>
  )
}

function TemplateDetail({ template }: { template: AgentTemplate }) {
  const { t } = useTranslation()
  return (
    <Card as="section" variant="elevated">
      <header className="p-4 border-b border-border flex items-center gap-3">
        <span className="text-3xl leading-none">{template.icon}</span>
        <div className="flex-1">
          <h2 className="text-base font-semibold">{template.displayName}</h2>
          <div className="text-[11px] font-mono text-muted-foreground mt-0.5 tabular-nums">
            {template.id} · {template.toolType} · {template.modelId}
          </div>
        </div>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 p-4">
        {/* Caps + Budget */}
        <div className="space-y-3">
          <Block title={t('agentGallery.section.caps', 'Capabilities (allowlist)')}>
            <div className="flex flex-wrap gap-1">
              {template.capabilities.length === 0 && <span className="text-[11px] text-muted-foreground">{t('agentGallery.noCaps', '没有授权（deny-by-default）')}</span>}
              {template.capabilities.map(c => (
                <span key={c} className="px-1.5 py-0.5 rounded bg-blue-500/15 text-blue-400 text-[10px] font-mono">{c}</span>
              ))}
            </div>
          </Block>

          <Block title={t('agentGallery.section.budget', 'Budget')}>
            <dl className="grid grid-cols-2 gap-1 text-[11px]">
              <dt className="text-muted-foreground font-mono">tokens</dt><dd className="font-mono tabular-nums">{template.budgetTokens.toLocaleString()}</dd>
              <dt className="text-muted-foreground font-mono">usd</dt><dd className="font-mono tabular-nums">${template.budgetUsd}</dd>
              <dt className="text-muted-foreground font-mono">period</dt><dd className="font-mono">{template.budgetPeriod}</dd>
              <dt className="text-muted-foreground font-mono">policy</dt><dd className="font-mono">{template.budgetPolicy}</dd>
            </dl>
          </Block>

          <Block title={t('agentGallery.section.guardrails', 'Guardrails')}>
            <ul className="space-y-1 text-[11px] list-disc list-inside marker:text-amber-400">
              {template.guardrails.length === 0 && <li className="text-muted-foreground">{t('common.empty', '暂无')}</li>}
              {template.guardrails.map((g, i) => (
                <li key={i}>{g}</li>
              ))}
            </ul>
          </Block>

          {template.useCases.length > 0 && (
            <Block title={t('agentGallery.section.useCases', 'Use cases')}>
              <ul className="space-y-1 text-[11px] list-disc list-inside marker:text-emerald-400">
                {template.useCases.map((u, i) => <li key={i}>{u}</li>)}
              </ul>
            </Block>
          )}
        </div>

        {/* System prompt + MCP */}
        <div className="space-y-3">
          <Block title={t('agentGallery.section.mcp', 'MCP servers required')}>
            <div className="flex flex-wrap gap-1">
              {template.mcpServers.length === 0 && <span className="text-[11px] text-muted-foreground">—</span>}
              {template.mcpServers.map(s => (
                <span key={s} className="px-1.5 py-0.5 rounded bg-emerald-500/15 text-emerald-400 text-[10px] font-mono">{s}</span>
              ))}
            </div>
          </Block>

          <Block title={t('agentGallery.section.prompt', 'System prompt')}>
            <pre className="text-[10px] font-mono whitespace-pre-wrap break-all max-h-72 overflow-auto p-2 rounded bg-card-recessed border border-border">{template.systemPrompt}</pre>
          </Block>

          {template.notes && (
            <Block title={t('agentGallery.section.notes', 'Deployment notes')}>
              <pre className="text-[10px] whitespace-pre-wrap p-2 rounded bg-card-recessed border border-border">{template.notes}</pre>
            </Block>
          )}
        </div>
      </div>
    </Card>
  )
}

function Block({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Card variant="recessed" className="p-3">
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground mb-2">[ {title.toUpperCase()} ]</div>
      {children}
    </Card>
  )
}
