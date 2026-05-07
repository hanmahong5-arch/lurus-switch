import { useEffect, useState } from 'react'
import { FieldRow } from './FieldRow'

interface OpenClawConfig {
  provider_type: string
  provider_api_key: string
  provider_model: string
  provider_base_url: string
  gateway_port: number
  channels_dm_policy: string
  skills_enabled: string[]
}

const DEFAULT_CONFIG: OpenClawConfig = {
  provider_type: 'anthropic',
  provider_api_key: '',
  provider_model: 'claude-sonnet-4-20250514',
  provider_base_url: '',
  gateway_port: 18789,
  channels_dm_policy: 'all',
  skills_enabled: [],
}

const PROVIDER_OPTIONS = [
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'custom', label: 'Custom' },
]

const DM_POLICY_OPTIONS = [
  { value: 'all', label: 'Allow all users' },
  { value: 'registered', label: 'Registered users only' },
  { value: 'none', label: 'Nobody' },
]

const AVAILABLE_SKILLS = ['web-search', 'code-exec', 'file-read', 'memory', 'calendar']

const inputCls = 'w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary'
const monoInputCls = inputCls + ' font-mono'

function toJson(cfg: OpenClawConfig): string {
  return JSON.stringify(
    {
      provider: {
        type: cfg.provider_type,
        api_key: cfg.provider_api_key,
        model: cfg.provider_model,
        ...(cfg.provider_base_url ? { base_url: cfg.provider_base_url } : {}),
      },
      gateway: { port: cfg.gateway_port },
      channels: { dm_policy: cfg.channels_dm_policy },
      skills: { enabled: cfg.skills_enabled },
    },
    null,
    2,
  )
}

function parseJson(raw: string): Partial<OpenClawConfig> {
  try {
    const parsed = JSON.parse(raw || '{}')
    return {
      provider_type: parsed.provider?.type ?? DEFAULT_CONFIG.provider_type,
      provider_api_key: parsed.provider?.api_key ?? '',
      provider_model: parsed.provider?.model ?? DEFAULT_CONFIG.provider_model,
      provider_base_url: parsed.provider?.base_url ?? '',
      gateway_port: parsed.gateway?.port ?? DEFAULT_CONFIG.gateway_port,
      channels_dm_policy: parsed.channels?.dm_policy ?? DEFAULT_CONFIG.channels_dm_policy,
      skills_enabled: parsed.skills?.enabled ?? [],
    }
  } catch {
    return {}
  }
}

interface OpenClawConfigFormProps {
  initialContent: string
  onChange: (json: string) => void
}

export function OpenClawConfigForm({ initialContent, onChange }: OpenClawConfigFormProps) {
  const [cfg, setCfg] = useState<OpenClawConfig>(() => ({ ...DEFAULT_CONFIG, ...parseJson(initialContent) }))

  useEffect(() => {
    setCfg({ ...DEFAULT_CONFIG, ...parseJson(initialContent) })
  }, [initialContent])

  const update = (patch: Partial<OpenClawConfig>) => {
    setCfg((prev) => {
      const next = { ...prev, ...patch }
      onChange(toJson(next))
      return next
    })
  }

  const toggleSkill = (skill: string) => {
    const next = cfg.skills_enabled.includes(skill)
      ? cfg.skills_enabled.filter((s) => s !== skill)
      : [...cfg.skills_enabled, skill]
    update({ skills_enabled: next })
  }

  return (
    <div className="space-y-4 p-4">
      <Section id="openclaw-section-provider" titleZh="服务商" titleEn="Provider">
        <FieldRow metaKey="openclaw.provider.type" value={cfg.provider_type}>
          <select value={cfg.provider_type} onChange={(e) => update({ provider_type: e.target.value })} className={inputCls}>
            {PROVIDER_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </FieldRow>
        <FieldRow metaKey="openclaw.provider.apiKey" value={cfg.provider_api_key}>
          <input type="password" value={cfg.provider_api_key}
            onChange={(e) => update({ provider_api_key: e.target.value })}
            placeholder="sk-ant-..." className={monoInputCls} />
        </FieldRow>
        <FieldRow metaKey="openclaw.provider.model" value={cfg.provider_model}>
          <input type="text" value={cfg.provider_model}
            onChange={(e) => update({ provider_model: e.target.value })}
            placeholder="claude-sonnet-4-20250514" className={monoInputCls} />
        </FieldRow>
        <FieldRow metaKey="openclaw.provider.baseUrl" value={cfg.provider_base_url}>
          <input type="url" value={cfg.provider_base_url}
            onChange={(e) => update({ provider_base_url: e.target.value })}
            placeholder="https://proxy.example.com" className={monoInputCls} />
        </FieldRow>
      </Section>

      <Section id="openclaw-section-gateway" titleZh="网关" titleEn="Gateway">
        <FieldRow metaKey="openclaw.gateway.port" value={cfg.gateway_port}>
          <input type="number" min={1024} max={65535} value={cfg.gateway_port}
            onChange={(e) => update({ gateway_port: parseInt(e.target.value) || DEFAULT_CONFIG.gateway_port })}
            className={inputCls} />
        </FieldRow>
      </Section>

      <Section id="openclaw-section-channels" titleZh="频道" titleEn="Channels">
        <FieldRow metaKey="openclaw.channels.dmPolicy" value={cfg.channels_dm_policy}>
          <select value={cfg.channels_dm_policy} onChange={(e) => update({ channels_dm_policy: e.target.value })} className={inputCls}>
            {DM_POLICY_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </FieldRow>
      </Section>

      <Section id="openclaw-section-skills" titleZh="能力" titleEn="Skills">
        <FieldRow metaKey="openclaw.skills.enabled" value={cfg.skills_enabled}>
          <div className="space-y-1.5 pt-1">
            {AVAILABLE_SKILLS.map((skill) => (
              <label key={skill} className="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" checked={cfg.skills_enabled.includes(skill)}
                  onChange={() => toggleSkill(skill)} className="rounded" />
                <span className="text-xs font-mono">{skill}</span>
              </label>
            ))}
          </div>
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
