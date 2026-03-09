import { useEffect, useState } from 'react'

interface ZeroClawConfig {
  provider_type: string
  provider_api_key: string
  provider_model: string
  provider_base_url: string
  gateway_port: number
  memory_backend: string
  security_sandbox: boolean
  security_allow_exec: boolean
}

const DEFAULT_CONFIG: ZeroClawConfig = {
  provider_type: 'anthropic',
  provider_api_key: '',
  provider_model: 'claude-sonnet-4-20250514',
  provider_base_url: '',
  gateway_port: 8765,
  memory_backend: 'sqlite',
  security_sandbox: false,
  security_allow_exec: true,
}

const PROVIDER_OPTIONS = [
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'custom', label: 'Custom' },
]

const MEMORY_OPTIONS = [
  { value: 'sqlite', label: 'SQLite (local)' },
  { value: 'memory', label: 'In-Memory' },
]

function toToml(cfg: ZeroClawConfig): string {
  return [
    '[provider]',
    `type = "${cfg.provider_type}"`,
    `api_key = "${cfg.provider_api_key}"`,
    `model = "${cfg.provider_model}"`,
    cfg.provider_base_url ? `base_url = "${cfg.provider_base_url}"` : '',
    '',
    '[gateway]',
    `port = ${cfg.gateway_port}`,
    '',
    '[memory]',
    `backend = "${cfg.memory_backend}"`,
    '',
    '[security]',
    `sandbox = ${cfg.security_sandbox}`,
    `allow_exec = ${cfg.security_allow_exec}`,
  ].filter((l) => l !== undefined).join('\n')
}

function parseToml(raw: string): Partial<ZeroClawConfig> {
  const result: Partial<ZeroClawConfig> = {}
  const lines = raw.split('\n')
  for (const line of lines) {
    const m = line.match(/^(\w+)\s*=\s*(.+)$/)
    if (!m) continue
    const [, key, val] = m
    const strVal = val.trim().replace(/^"(.*)"$/, '$1')
    if (key === 'type') result.provider_type = strVal
    if (key === 'api_key') result.provider_api_key = strVal
    if (key === 'model') result.provider_model = strVal
    if (key === 'base_url') result.provider_base_url = strVal
    if (key === 'port') result.gateway_port = parseInt(strVal) || DEFAULT_CONFIG.gateway_port
    if (key === 'backend') result.memory_backend = strVal
    if (key === 'sandbox') result.security_sandbox = strVal === 'true'
    if (key === 'allow_exec') result.security_allow_exec = strVal === 'true'
  }
  return result
}

interface ZeroClawConfigFormProps {
  initialContent: string
  onChange: (toml: string) => void
}

export function ZeroClawConfigForm({ initialContent, onChange }: ZeroClawConfigFormProps) {
  const [cfg, setCfg] = useState<ZeroClawConfig>(() => ({
    ...DEFAULT_CONFIG,
    ...parseToml(initialContent),
  }))

  useEffect(() => {
    setCfg({ ...DEFAULT_CONFIG, ...parseToml(initialContent) })
  }, [initialContent])

  const update = (patch: Partial<ZeroClawConfig>) => {
    setCfg((prev) => {
      const next = { ...prev, ...patch }
      onChange(toToml(next))
      return next
    })
  }

  return (
    <div className="space-y-5 p-4">
      {/* Provider */}
      <section id="zeroclaw-section-provider" className="space-y-3">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Provider</h3>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Provider Type</label>
          <select
            value={cfg.provider_type}
            onChange={(e) => update({ provider_type: e.target.value })}
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
          >
            {PROVIDER_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
        </div>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">API Key</label>
          <input
            type="password"
            value={cfg.provider_api_key}
            onChange={(e) => update({ provider_api_key: e.target.value })}
            placeholder="sk-ant-..."
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
          />
        </div>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Model</label>
          <input
            type="text"
            value={cfg.provider_model}
            onChange={(e) => update({ provider_model: e.target.value })}
            placeholder="claude-sonnet-4-20250514"
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
          />
        </div>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Base URL (optional)</label>
          <input
            type="url"
            value={cfg.provider_base_url}
            onChange={(e) => update({ provider_base_url: e.target.value })}
            placeholder="https://proxy.example.com"
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
          />
        </div>
      </section>

      <hr className="border-border" />

      {/* Gateway */}
      <section id="zeroclaw-section-gateway" className="space-y-3">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Gateway</h3>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Port</label>
          <input
            type="number"
            min={1024}
            max={65535}
            value={cfg.gateway_port}
            onChange={(e) => update({ gateway_port: parseInt(e.target.value) || DEFAULT_CONFIG.gateway_port })}
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
      </section>

      <hr className="border-border" />

      {/* Memory */}
      <section id="zeroclaw-section-memory" className="space-y-3">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Memory</h3>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Backend</label>
          <select
            value={cfg.memory_backend}
            onChange={(e) => update({ memory_backend: e.target.value })}
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
          >
            {MEMORY_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
        </div>
      </section>

      <hr className="border-border" />

      {/* Security */}
      <section id="zeroclaw-section-security" className="space-y-3">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Security</h3>

        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={cfg.security_sandbox}
            onChange={(e) => update({ security_sandbox: e.target.checked })}
            className="rounded"
          />
          <span className="text-xs font-medium">Enable Sandbox</span>
        </label>

        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={cfg.security_allow_exec}
            onChange={(e) => update({ security_allow_exec: e.target.checked })}
            className="rounded"
          />
          <span className="text-xs font-medium">Allow Command Execution</span>
        </label>
      </section>
    </div>
  )
}
