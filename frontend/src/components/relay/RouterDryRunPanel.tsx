import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Beaker, ArrowRight, Loader2 } from 'lucide-react'
import { DryRunRouter } from '../../../wailsjs/go/main/App'
import { relay } from '../../../wailsjs/go/models'
import { Button } from '../ui'

// RouterDryRunPanel lets users probe what Router.Pick would do for a
// hypothetical request without sending real traffic. Same code path as
// the gateway proxy uses — see internal/gateway/server.go
// buildChainFromRouter — so the verdict is bit-equivalent.
const TOOL_OPTIONS = [
  { value: '', label: '— (auto)' },
  { value: 'claude', label: 'Claude Code' },
  { value: 'codex', label: 'Codex' },
  { value: 'gemini', label: 'Gemini CLI' },
  { value: 'picoclaw', label: 'PicoClaw' },
  { value: 'nullclaw', label: 'NullClaw' },
  { value: 'openclaw', label: 'OpenClaw' },
]

export function RouterDryRunPanel() {
  const { t } = useTranslation()
  const [tool, setTool] = useState('claude')
  const [model, setModel] = useState('claude-sonnet-4-6')
  const [tokens, setTokens] = useState(1000)
  const [hasTools, setHasTools] = useState(false)
  const [result, setResult] = useState<relay.PickResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [running, setRunning] = useState(false)

  const handleRun = async () => {
    setRunning(true)
    setError(null)
    setResult(null)
    try {
      const res = await DryRunRouter(tool, model, tokens, hasTools)
      setResult(res)
    } catch (e: any) {
      setError(e?.message ?? String(e))
    } finally {
      setRunning(false)
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <Beaker className="h-3.5 w-3.5 text-primary" />
        <h4 className="text-sm font-semibold">
          {t('relay.dryRun.title', '路由试运行')}
        </h4>
      </div>
      <p className="text-[11px] text-muted-foreground">
        {t(
          'relay.dryRun.desc',
          '模拟一次请求，看哪个 endpoint 会赢 + 命中哪条规则。不发真实流量。',
        )}
      </p>

      <div className="grid grid-cols-2 gap-2">
        <div>
          <label className="block text-[10px] text-muted-foreground mb-1 font-mono uppercase tracking-wide">
            tool
          </label>
          <select
            value={tool}
            onChange={(e) => setTool(e.target.value)}
            className="w-full px-2 py-1.5 text-xs rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary"
          >
            {TOOL_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-[10px] text-muted-foreground mb-1 font-mono uppercase tracking-wide">
            model
          </label>
          <input
            value={model}
            onChange={(e) => setModel(e.target.value)}
            placeholder="claude-sonnet-4-6"
            className="w-full px-2 py-1.5 text-xs rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary font-mono"
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div>
          <label className="block text-[10px] text-muted-foreground mb-1 font-mono uppercase tracking-wide">
            est. input tokens
          </label>
          <input
            type="number"
            value={tokens}
            min={0}
            onChange={(e) => setTokens(Number(e.target.value) || 0)}
            className="w-full px-2 py-1.5 text-xs rounded border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary font-mono tabular-nums"
          />
        </div>
        <label className="flex items-center gap-2 pt-5 text-xs">
          <input
            type="checkbox"
            checked={hasTools}
            onChange={(e) => setHasTools(e.target.checked)}
            className="h-3.5 w-3.5"
          />
          <span className="text-muted-foreground">{t('relay.dryRun.hasTools', 'has tools')}</span>
        </label>
      </div>

      <Button
        size="sm"
        onClick={handleRun}
        disabled={running || !model.trim()}
        icon={running ? <Loader2 className="h-3 w-3 animate-spin" /> : <Beaker className="h-3 w-3" />}
      >
        {t('relay.dryRun.run', '运行')}
      </Button>

      {error && (
        <div className="text-[11px] text-red-400 bg-red-500/5 px-2 py-1.5 rounded border border-red-500/20">
          {error}
        </div>
      )}

      {result && (
        <div
          data-testid="dry-run-result"
          className="space-y-2 rounded-lg border border-border bg-card-recessed p-3"
        >
          <div className="flex items-baseline gap-2">
            <span className="text-[10px] uppercase tracking-wide text-muted-foreground font-mono">
              {t('relay.dryRun.picked', '中选')}
            </span>
            <span className="text-xs font-medium">{result.Endpoint?.name || result.Endpoint?.id}</span>
            <span className="font-mono text-[10px] text-muted-foreground tabular-nums">
              {result.Endpoint?.latencyMs}ms
            </span>
          </div>
          <div className="text-[11px] text-muted-foreground">
            <span className="font-mono">matchedBy:</span>{' '}
            {result.MatchedBy ? (
              <code className="text-primary">{result.MatchedBy}</code>
            ) : (
              <span className="italic">{t('relay.dryRun.noRule', '无规则匹配（走 tool 映射 / 最低延迟）')}</span>
            )}
          </div>
          {result.Ordered && result.Ordered.length > 1 && (
            <div>
              <p className="text-[10px] uppercase tracking-wide text-muted-foreground font-mono mb-1">
                {t('relay.dryRun.chain', '级联顺序')}
              </p>
              <div className="flex flex-wrap items-center gap-1">
                {result.Ordered.map((ep, i) => (
                  <span key={ep.id} className="inline-flex items-center gap-1">
                    {i > 0 && <ArrowRight className="h-2.5 w-2.5 text-muted-foreground" />}
                    <span
                      className={`font-mono text-[10px] px-1.5 py-0.5 rounded ${
                        i === 0
                          ? 'bg-primary/15 text-primary'
                          : 'bg-card text-muted-foreground border border-border'
                      }`}
                    >
                      {ep.name || ep.id}
                    </span>
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
