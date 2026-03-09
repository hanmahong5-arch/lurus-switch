import { useEffect, useState, useCallback } from 'react'
import { Loader2 } from 'lucide-react'
import { config, validator } from '../../../wailsjs/go/models'
import {
  GetDefaultCodexConfig,
  GenerateCodexConfig,
  ValidateCodexConfig,
} from '../../../wailsjs/go/main/App'
import { SwitchField } from './SwitchField'
import { SelectField } from './SelectField'
import { TagInput } from './TagInput'
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

export function CodexConfigForm({ initialContent, onChange, onValidation }: CodexConfigFormProps) {
  const [cfg, setCfg] = useState<config.CodexConfig | null>(null)
  const [validation, setValidation] = useState<validator.ValidationResult | null>(null)
  const gwStatus = useGatewayStore((s) => s.status)
  const localGatewayURL = gwStatus?.running ? `http://localhost:${gwStatus.port}/v1` : null

  useEffect(() => {
    GetDefaultCodexConfig().then((defaults) => {
      // TOML can't be merged on frontend — just use defaults and overlay
      // known scalar fields that appear commonly in TOML content
      // The backend GenerateCodexConfig will produce correct TOML from the struct
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
    <div className="space-y-5 p-4">
      {/* Presets */}
      <PresetSelector tool="codex" onApply={(c) => {
        const typed = c as config.CodexConfig
        setCfg(typed)
        syncOutput(typed)
      }} />

      <hr className="border-border" />

      {/* Core */}
      <section id="codex-section-core" className="space-y-3">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Core</h3>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Model</label>
          <select
            value={cfg.model || ''}
            onChange={(e) => update({ model: e.target.value })}
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
          >
            {MODEL_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
          {fieldError('model') && <p className="text-xs text-red-400">{fieldError('model')}</p>}
        </div>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">API Key</label>
          <input
            type="password"
            value={cfg.apiKey || ''}
            onChange={(e) => update({ apiKey: e.target.value })}
            placeholder="sk-..."
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
          />
          {fieldError('apiKey') && <p className="text-xs text-red-400">{fieldError('apiKey')}</p>}
        </div>

        <SelectField
          label="Approval Mode"
          value={cfg.approvalMode || 'on-failure'}
          options={APPROVAL_OPTIONS}
          onChange={(v) => update({ approvalMode: v })}
        />
        {fieldError('approvalMode') && <p className="text-xs text-red-400">{fieldError('approvalMode')}</p>}
      </section>

      <hr className="border-border" />

      {/* Provider */}
      <section id="codex-section-provider" className="space-y-3">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Provider</h3>

        <SelectField
          label="Provider Type"
          value={cfg.provider?.type || 'openai'}
          options={PROVIDER_OPTIONS}
          onChange={(v) => updateProvider({ type: v })}
        />

        {(cfg.provider?.type === 'custom' || cfg.provider?.type === 'azure') && (
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground">Base URL</label>
            <input
              type="url"
              value={cfg.provider?.baseUrl || ''}
              onChange={(e) => updateProvider({ baseUrl: e.target.value })}
              placeholder="https://proxy.example.com/v1"
              className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
            />
            <EndpointPresetPicker
              localURL={localGatewayURL}
              value={cfg.provider?.baseUrl || ''}
              onChange={(url) => updateProvider({ baseUrl: url })}
            />
            {fieldError('provider.baseUrl') && <p className="text-xs text-red-400">{fieldError('provider.baseUrl')}</p>}
          </div>
        )}

        {cfg.provider?.type === 'azure' && (
          <>
            <div className="space-y-1">
              <label className="text-xs font-medium text-muted-foreground">Azure Deployment</label>
              <input
                type="text"
                value={cfg.provider?.azureDeployment || ''}
                onChange={(e) => updateProvider({ azureDeployment: e.target.value })}
                placeholder="my-gpt4o-deployment"
                className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
              />
            </div>
            <div className="space-y-1">
              <label className="text-xs font-medium text-muted-foreground">Azure API Version</label>
              <input
                type="text"
                value={cfg.provider?.azureApiVersion || ''}
                onChange={(e) => updateProvider({ azureApiVersion: e.target.value })}
                placeholder="2024-02-01"
                className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
              />
            </div>
          </>
        )}
      </section>

      <hr className="border-border" />

      {/* Security */}
      <section id="codex-section-security" className="space-y-2">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Security</h3>

        <SelectField
          label="Network Access"
          value={cfg.security?.networkAccess || 'full'}
          options={NETWORK_OPTIONS}
          onChange={(v) => updateSecurity({ networkAccess: v })}
        />

        <SwitchField
          label="Command Execution"
          description="Allow shell command execution"
          checked={cfg.security?.commandExecution?.enabled ?? true}
          onChange={(v) => updateCommandExec({ enabled: v })}
        />

        {cfg.security?.commandExecution?.enabled && (
          <>
            <TagInput
              label="Allowed Commands"
              values={cfg.security?.commandExecution?.allowedCommands || []}
              onChange={(v) => updateCommandExec({ allowedCommands: v })}
              placeholder="e.g. git *"
            />
            <TagInput
              label="Denied Commands"
              values={cfg.security?.commandExecution?.deniedCommands || []}
              onChange={(v) => updateCommandExec({ deniedCommands: v })}
              placeholder="e.g. rm -rf *"
            />
          </>
        )}

        <TagInput
          label="Allowed Directories"
          values={cfg.security?.fileAccess?.allowedDirs || []}
          onChange={(v) => updateFileAccess({ allowedDirs: v })}
          placeholder="e.g. /home/user/projects"
        />
        <TagInput
          label="Read-Only Directories"
          values={cfg.security?.fileAccess?.readOnlyDirs || []}
          onChange={(v) => updateFileAccess({ readOnlyDirs: v })}
          placeholder="e.g. /etc"
        />
        <TagInput
          label="Denied Patterns"
          values={cfg.security?.fileAccess?.deniedPatterns || []}
          onChange={(v) => updateFileAccess({ deniedPatterns: v })}
          placeholder="e.g. *.env"
        />
      </section>

      <hr className="border-border" />

      {/* MCP */}
      <section className="space-y-2">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">MCP</h3>
        <SwitchField
          label="Enable MCP"
          description="Model Context Protocol server support"
          checked={cfg.mcp?.enabled ?? false}
          onChange={(v) => updateMCP({ enabled: v })}
        />
      </section>

      <hr className="border-border" />

      {/* History */}
      <section className="space-y-2">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">History</h3>
        <SwitchField
          label="Enable History"
          description="Save conversation history to file"
          checked={cfg.history?.enabled ?? true}
          onChange={(v) => updateHistory({ enabled: v })}
        />
        {cfg.history?.enabled && (
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground">Max Entries</label>
            <input
              type="number"
              min={1}
              max={10000}
              value={cfg.history?.maxEntries || 1000}
              onChange={(e) => updateHistory({ maxEntries: parseInt(e.target.value) || 1000 })}
              className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
            />
            {fieldError('history.maxEntries') && <p className="text-xs text-red-400">{fieldError('history.maxEntries')}</p>}
          </div>
        )}
      </section>

      <hr className="border-border" />

      {/* Sandbox */}
      <section id="codex-section-sandbox" className="space-y-2">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Sandbox</h3>
        <SwitchField
          label="Enable Sandbox"
          checked={cfg.sandbox?.enabled ?? false}
          onChange={(v) => updateSandbox({ enabled: v })}
        />
        {cfg.sandbox?.enabled && (
          <SelectField
            label="Sandbox Type"
            value={cfg.sandbox?.type || 'none'}
            options={SANDBOX_TYPE_OPTIONS}
            onChange={(v) => updateSandbox({ type: v })}
          />
        )}
        {fieldError('sandbox.type') && <p className="text-xs text-red-400">{fieldError('sandbox.type')}</p>}
      </section>

      <ValidationPanel result={validation} showSuccess />
    </div>
  )
}
