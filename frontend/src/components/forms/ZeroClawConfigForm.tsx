import { useEffect, useState } from 'react'
import { BareToggle } from './SwitchField'
import { FieldRow } from './FieldRow'

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

const inputCls = 'w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary'
const monoInputCls = inputCls + ' font-mono'

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
  const [cfg, setCfg] = useState<ZeroClawConfig>(() => ({ ...DEFAULT_CONFIG, ...parseToml(initialContent) }))

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
    <div className="space-y-4 p-4">
      <Section id="zeroclaw-section-provider" titleZh="服务商" titleEn="Provider">
        <FieldRow metaKey="zeroclaw.provider.type" value={cfg.provider_type}>
          <select value={cfg.provider_type} onChange={(e) => update({ provider_type: e.target.value })} className={inputCls}>
            {PROVIDER_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </FieldRow>
        <FieldRow metaKey="zeroclaw.provider.apiKey" value={cfg.provider_api_key}>
          <input type="password" value={cfg.provider_api_key}
            onChange={(e) => update({ provider_api_key: e.target.value })}
            placeholder="sk-ant-..." className={monoInputCls} />
        </FieldRow>
        <FieldRow metaKey="zeroclaw.provider.model" value={cfg.provider_model}>
          <input type="text" value={cfg.provider_model}
            onChange={(e) => update({ provider_model: e.target.value })}
            placeholder="claude-sonnet-4-20250514" className={monoInputCls} />
        </FieldRow>
        <FieldRow metaKey="zeroclaw.provider.baseUrl" value={cfg.provider_base_url}>
          <input type="url" value={cfg.provider_base_url}
            onChange={(e) => update({ provider_base_url: e.target.value })}
            placeholder="https://proxy.example.com" className={monoInputCls} />
        </FieldRow>
      </Section>

      <Section id="zeroclaw-section-gateway" titleZh="网关" titleEn="Gateway">
        <FieldRow metaKey="zeroclaw.gateway.port" value={cfg.gateway_port}>
          <input type="number" min={1024} max={65535} value={cfg.gateway_port}
            onChange={(e) => update({ gateway_port: parseInt(e.target.value) || DEFAULT_CONFIG.gateway_port })}
            className={inputCls} />
        </FieldRow>
      </Section>

      <Section id="zeroclaw-section-memory" titleZh="记忆" titleEn="Memory">
        <FieldRow metaKey="zeroclaw.memory.backend" value={cfg.memory_backend}>
          <select value={cfg.memory_backend} onChange={(e) => update({ memory_backend: e.target.value })} className={inputCls}>
            {MEMORY_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </FieldRow>
      </Section>

      <Section id="zeroclaw-section-security" titleZh="安全" titleEn="Security">
        <FieldRow metaKey="zeroclaw.security.sandbox" value={cfg.security_sandbox} layout="inline">
          <BareToggle checked={cfg.security_sandbox} onChange={(v) => update({ security_sandbox: v })} />
        </FieldRow>
        <FieldRow metaKey="zeroclaw.security.allowExec" value={cfg.security_allow_exec} layout="inline">
          <BareToggle checked={cfg.security_allow_exec} onChange={(v) => update({ security_allow_exec: v })} />
        </FieldRow>
      </Section>
    </div>
  )
}

function Section({ id, titleZh, titleEn, children }: { id: string; titleZh: string; titleEn: string; children: React.ReactNode }) {
  return (
    <section id={id} className="rounded-lg border border-border/60 bg-card/40 p-3">
      <h3 className="text-xs font-semibold uppercase tracking-wider mb-1 flex items-center gap-2">
        <span className="text-foreground">{titleZh}</span>
        <span className="text-[10px] text-muted-foreground/70 font-normal">{titleEn}</span>
      </h3>
      <div className="space-y-0">{children}</div>
    </section>
  )
}
