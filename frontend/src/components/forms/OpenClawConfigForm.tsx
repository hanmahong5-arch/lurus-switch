import { useEffect, useState } from 'react'

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
  const [cfg, setCfg] = useState<OpenClawConfig>(() => ({
    ...DEFAULT_CONFIG,
    ...parseJson(initialContent),
  }))

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
    <div className="space-y-5 p-4">
      {/* Provider */}
      <section id="openclaw-section-provider" className="space-y-3">
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
      <section id="openclaw-section-gateway" className="space-y-3">
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

      {/* Channels */}
      <section id="openclaw-section-channels" className="space-y-3">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Channels</h3>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">DM Policy</label>
          <select
            value={cfg.channels_dm_policy}
            onChange={(e) => update({ channels_dm_policy: e.target.value })}
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
          >
            {DM_POLICY_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
        </div>
      </section>

      <hr className="border-border" />

      {/* Skills */}
      <section id="openclaw-section-skills" className="space-y-3">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Skills</h3>
        <p className="text-xs text-muted-foreground">Select skills to enable for this bot.</p>

        <div className="space-y-1.5">
          {AVAILABLE_SKILLS.map((skill) => (
            <label key={skill} className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={cfg.skills_enabled.includes(skill)}
                onChange={() => toggleSkill(skill)}
                className="rounded"
              />
              <span className="text-xs font-mono">{skill}</span>
            </label>
          ))}
        </div>
      </section>
    </div>
  )
}
