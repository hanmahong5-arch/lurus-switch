import { useEffect, useState, useCallback } from 'react'
import { Loader2 } from 'lucide-react'
import { config, validator } from '../../../wailsjs/go/models'
import {
  GetDefaultCodexConfig,
  GenerateCodexConfig,
  ValidateCodexConfig,
} from '../../../wailsjs/go/main/App'
import { BareToggle } from './SwitchField'
import { TagInput } from './TagInput'
import { FieldRow } from './FieldRow'
import { ValidationPanel } from '../ValidationPanel'
import { PresetSelector } from '../PresetSelector'
import { EndpointPresetPicker } from './EndpointPresetPicker'
import { useGatewayStore } from '../../stores/gatewayStore'

interface CodexConfigFormProps {
  initialContent: string
  onChange: (toml: string) => void
  onValidation: (result: validator.ValidationResult | null) => void
}

const MODEL_OPTIONS = [
  { value: 'o4-mini', label: 'o4-mini' },
  { value: 'o3', label: 'o3' },
  { value: 'gpt-4o', label: 'GPT-4o' },
  { value: 'gpt-4o-mini', label: 'GPT-4o mini' },
  { value: 'gpt-4-turbo', label: 'GPT-4 Turbo' },
]

const APPROVAL_OPTIONS = [
  { value: 'on-failure', label: 'On Failure' },
  { value: 'unless-allow-listed', label: 'Unless Allow-Listed' },
  { value: 'never', label: 'Never' },
]

const PROVIDER_OPTIONS = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'azure', label: 'Azure OpenAI' },
  { value: 'custom', label: 'Custom / Proxy' },
]

const NETWORK_OPTIONS = [
  { value: 'full', label: 'Full Access' },
  { value: 'restricted', label: 'Restricted' },
  { value: 'none', label: 'None' },
]

const SANDBOX_TYPE_OPTIONS = [
  { value: 'none', label: 'None' },
  { value: 'docker', label: 'Docker' },
]

const inputCls =
  'w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary'
const monoInputCls = inputCls + ' font-mono'

export function CodexConfigForm({ initialContent, onChange, onValidation }: CodexConfigFormProps) {
  const [cfg, setCfg] = useState<config.CodexConfig | null>(null)
  const [validation, setValidation] = useState<validator.ValidationResult | null>(null)
  const gwStatus = useGatewayStore((s) => s.status)
  const localGatewayURL = gwStatus?.running ? `http://localhost:${gwStatus.port}/v1` : null

  useEffect(() => {
    GetDefaultCodexConfig().then((defaults) => {
      setCfg(defaults)
    }).catch(() => {
      setCfg({} as config.CodexConfig)
    })
  }, [initialContent])

  const syncOutput = useCallback(async (next: config.CodexConfig) => {
    try {
      const [toml, result] = await Promise.all([
        GenerateCodexConfig(next),
        ValidateCodexConfig(next),
      ])
      setValidation(result)
      onValidation(result)
      onChange(toml)
    } catch (err) {
      console.error('CodexConfigForm sync error:', err)
    }
  }, [onChange, onValidation])

  const update = (patch: Partial<config.CodexConfig>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, ...patch } as config.CodexConfig
      syncOutput(next)
      return next
    })
  }

  const updateProvider = (patch: Partial<config.CodexProvider>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, provider: { ...prev.provider, ...patch } } as config.CodexConfig
      syncOutput(next)
      return next
    })
  }

  const updateSecurity = (patch: Partial<config.CodexSecurity>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, security: { ...prev.security, ...patch } } as config.CodexConfig
      syncOutput(next)
      return next
    })
  }

  const updateFileAccess = (patch: Partial<config.CodexFileAccess>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = {
        ...prev,
        security: {
          ...prev.security,
          fileAccess: { ...prev.security?.fileAccess, ...patch },
        },
      } as config.CodexConfig
      syncOutput(next)
      return next
    })
  }

  const updateCommandExec = (patch: Partial<config.CodexCommandExecution>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = {
        ...prev,
        security: {
          ...prev.security,
          commandExecution: { ...prev.security?.commandExecution, ...patch },
        },
      } as config.CodexConfig
      syncOutput(next)
      return next
    })
  }

  const updateMCP = (patch: Partial<config.CodexMCP>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, mcp: { ...prev.mcp, ...patch } } as config.CodexConfig
      syncOutput(next)
      return next
    })
  }

  const updateHistory = (patch: Partial<config.CodexHistory>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, history: { ...prev.history, ...patch } } as config.CodexConfig
      syncOutput(next)
      return next
    })
  }

  const updateSandbox = (patch: Partial<config.CodexSandbox>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, sandbox: { ...prev.sandbox, ...patch } } as config.CodexConfig
      syncOutput(next)
      return next
    })
  }

  if (!cfg) {
    return (
      <div className="flex items-center justify-center h-32">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  const fieldError = (field: string): string | undefined =>
    validation?.errors?.find((e) => e.field === field)?.message

  return (
    <div className="space-y-4 p-4">
      <PresetSelector tool="codex" onApply={(c) => {
        const typed = c as config.CodexConfig
        setCfg(typed)
        syncOutput(typed)
      }} />

      <Section id="codex-section-core" titleZh="核心设置" titleEn="Core">
        <FieldRow metaKey="codex.model" value={cfg.model} errorMessage={fieldError('model')}>
          <select value={cfg.model || ''} onChange={(e) => update({ model: e.target.value })} className={inputCls}>
            {MODEL_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </FieldRow>
        <FieldRow metaKey="codex.apiKey" value={cfg.apiKey} errorMessage={fieldError('apiKey')}>
          <input type="password" value={cfg.apiKey || ''} onChange={(e) => update({ apiKey: e.target.value })}
            placeholder="sk-..." className={monoInputCls} />
        </FieldRow>
        <FieldRow metaKey="codex.approvalMode" value={cfg.approvalMode} errorMessage={fieldError('approvalMode')}>
          <select value={cfg.approvalMode || 'on-failure'} onChange={(e) => update({ approvalMode: e.target.value })} className={inputCls}>
            {APPROVAL_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </FieldRow>
      </Section>

      <Section id="codex-section-provider" titleZh="服务商" titleEn="Provider">
        <FieldRow metaKey="codex.provider.type" value={cfg.provider?.type}>
          <select value={cfg.provider?.type || 'openai'} onChange={(e) => updateProvider({ type: e.target.value })} className={inputCls}>
            {PROVIDER_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </FieldRow>
        {(cfg.provider?.type === 'custom' || cfg.provider?.type === 'azure') && (
          <FieldRow metaKey="codex.provider.baseUrl" value={cfg.provider?.baseUrl} errorMessage={fieldError('provider.baseUrl')}>
            <div className="space-y-1">
              <input type="url" value={cfg.provider?.baseUrl || ''} onChange={(e) => updateProvider({ baseUrl: e.target.value })}
                placeholder="https://proxy.example.com/v1" className={monoInputCls} />
              <EndpointPresetPicker localURL={localGatewayURL} value={cfg.provider?.baseUrl || ''} onChange={(url) => updateProvider({ baseUrl: url })} />
            </div>
          </FieldRow>
        )}
        {cfg.provider?.type === 'azure' && (
          <>
            <FieldRow metaKey="codex.provider.azureDeployment" value={cfg.provider?.azureDeployment}>
              <input type="text" value={cfg.provider?.azureDeployment || ''} onChange={(e) => updateProvider({ azureDeployment: e.target.value })}
                placeholder="my-gpt4o-deployment" className={monoInputCls} />
            </FieldRow>
            <FieldRow metaKey="codex.provider.azureApiVersion" value={cfg.provider?.azureApiVersion}>
              <input type="text" value={cfg.provider?.azureApiVersion || ''} onChange={(e) => updateProvider({ azureApiVersion: e.target.value })}
                placeholder="2024-02-01" className={monoInputCls} />
            </FieldRow>
          </>
        )}
      </Section>

      <Section id="codex-section-security" titleZh="安全" titleEn="Security">
        <FieldRow metaKey="codex.security.networkAccess" value={cfg.security?.networkAccess}>
          <select value={cfg.security?.networkAccess || 'full'} onChange={(e) => updateSecurity({ networkAccess: e.target.value })} className={inputCls}>
            {NETWORK_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </FieldRow>
        <FieldRow metaKey="codex.security.commandExecution.enabled" value={cfg.security?.commandExecution?.enabled ?? true} layout="inline">
          <BareToggle checked={cfg.security?.commandExecution?.enabled ?? true} onChange={(v) => updateCommandExec({ enabled: v })} />
        </FieldRow>
        {cfg.security?.commandExecution?.enabled && (
          <>
            <FieldRow metaKey="codex.security.commandExecution.allowedCommands" value={cfg.security?.commandExecution?.allowedCommands || []}>
              <TagInput label="" values={cfg.security?.commandExecution?.allowedCommands || []}
                onChange={(v) => updateCommandExec({ allowedCommands: v })} placeholder="e.g. git *" />
            </FieldRow>
            <FieldRow metaKey="codex.security.commandExecution.deniedCommands" value={cfg.security?.commandExecution?.deniedCommands || []}>
              <TagInput label="" values={cfg.security?.commandExecution?.deniedCommands || []}
                onChange={(v) => updateCommandExec({ deniedCommands: v })} placeholder="e.g. rm -rf *" />
            </FieldRow>
          </>
        )}
        <FieldRow metaKey="codex.security.fileAccess.allowedDirs" value={cfg.security?.fileAccess?.allowedDirs || []}>
          <TagInput label="" values={cfg.security?.fileAccess?.allowedDirs || []}
            onChange={(v) => updateFileAccess({ allowedDirs: v })} placeholder="e.g. /home/user/projects" />
        </FieldRow>
        <FieldRow metaKey="codex.security.fileAccess.readOnlyDirs" value={cfg.security?.fileAccess?.readOnlyDirs || []}>
          <TagInput label="" values={cfg.security?.fileAccess?.readOnlyDirs || []}
            onChange={(v) => updateFileAccess({ readOnlyDirs: v })} placeholder="e.g. /etc" />
        </FieldRow>
        <FieldRow metaKey="codex.security.fileAccess.deniedPatterns" value={cfg.security?.fileAccess?.deniedPatterns || []}>
          <TagInput label="" values={cfg.security?.fileAccess?.deniedPatterns || []}
            onChange={(v) => updateFileAccess({ deniedPatterns: v })} placeholder="e.g. *.env" />
        </FieldRow>
      </Section>

      <Section id="codex-section-mcp" titleZh="MCP" titleEn="MCP">
        <FieldRow metaKey="codex.mcp.enabled" value={cfg.mcp?.enabled ?? false} layout="inline">
          <BareToggle checked={cfg.mcp?.enabled ?? false} onChange={(v) => updateMCP({ enabled: v })} />
        </FieldRow>
      </Section>

      <Section id="codex-section-history" titleZh="历史" titleEn="History">
        <FieldRow metaKey="codex.history.enabled" value={cfg.history?.enabled ?? true} layout="inline">
          <BareToggle checked={cfg.history?.enabled ?? true} onChange={(v) => updateHistory({ enabled: v })} />
        </FieldRow>
        {cfg.history?.enabled && (
          <FieldRow metaKey="codex.history.maxEntries" value={cfg.history?.maxEntries} errorMessage={fieldError('history.maxEntries')}>
            <input type="number" min={1} max={10000} value={cfg.history?.maxEntries || 1000}
              onChange={(e) => updateHistory({ maxEntries: parseInt(e.target.value) || 1000 })} className={inputCls} />
          </FieldRow>
        )}
      </Section>

      <Section id="codex-section-sandbox" titleZh="沙箱" titleEn="Sandbox">
        <FieldRow metaKey="codex.sandbox.enabled" value={cfg.sandbox?.enabled ?? false} layout="inline">
          <BareToggle checked={cfg.sandbox?.enabled ?? false} onChange={(v) => updateSandbox({ enabled: v })} />
        </FieldRow>
        {cfg.sandbox?.enabled && (
          <FieldRow metaKey="codex.sandbox.type" value={cfg.sandbox?.type} errorMessage={fieldError('sandbox.type')}>
            <select value={cfg.sandbox?.type || 'none'} onChange={(e) => updateSandbox({ type: e.target.value })} className={inputCls}>
              {SANDBOX_TYPE_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
            </select>
          </FieldRow>
        )}
      </Section>

      <ValidationPanel result={validation} showSuccess />
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
