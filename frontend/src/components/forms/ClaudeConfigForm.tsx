import { useEffect, useState, useCallback } from 'react'
import { Loader2 } from 'lucide-react'
import { config, validator } from '../../../wailsjs/go/models'
import {
  GetDefaultClaudeConfig,
  GenerateClaudeConfig,
  ValidateClaudeConfig,
} from '../../../wailsjs/go/main/App'
import { SwitchField } from './SwitchField'
import { SelectField } from './SelectField'
import { TagInput } from './TagInput'
import { ValidationPanel } from '../ValidationPanel'
import { PresetSelector } from '../PresetSelector'
import { EndpointPresetPicker } from './EndpointPresetPicker'
import { useGatewayStore } from '../../stores/gatewayStore'

interface ClaudeConfigFormProps {
  /** Current raw JSON content from disk (used to initialise form state). */
  initialContent: string
  /** Called with newly generated JSON whenever the form changes. */
  onChange: (json: string) => void
  /** Expose current validation result to parent. */
  onValidation: (result: validator.ValidationResult | null) => void
}

const MODEL_OPTIONS = [
  { value: 'claude-sonnet-4-20250514', label: 'Claude Sonnet 4' },
  { value: 'claude-opus-4-20250514', label: 'Claude Opus 4' },
  { value: 'claude-haiku-4-20250514', label: 'Claude Haiku 4' },
  { value: 'claude-3-5-sonnet-20241022', label: 'Claude 3.5 Sonnet' },
  { value: 'claude-3-5-haiku-20241022', label: 'Claude 3.5 Haiku' },
]

const SANDBOX_OPTIONS = [
  { value: 'none', label: 'None' },
  { value: 'docker', label: 'Docker' },
  { value: 'wsl', label: 'WSL' },
]

export function ClaudeConfigForm({ initialContent, onChange, onValidation }: ClaudeConfigFormProps) {
  const [cfg, setCfg] = useState<config.ClaudeConfig | null>(null)
  const [validation, setValidation] = useState<validator.ValidationResult | null>(null)
  const gwStatus = useGatewayStore((s) => s.status)
  const localGatewayURL = gwStatus?.running ? `http://localhost:${gwStatus.port}/v1` : null

  // Parse initial JSON → form state
  useEffect(() => {
    let parsed: Partial<config.ClaudeConfig> = {}
    try {
      parsed = JSON.parse(initialContent || '{}')
    } catch {
      // ignore — will use defaults
    }
    GetDefaultClaudeConfig().then((defaults) => {
      setCfg({ ...defaults, ...parsed } as config.ClaudeConfig)
    }).catch(() => {
      setCfg(parsed as config.ClaudeConfig)
    })
  }, [initialContent])

  // Validate + regenerate JSON on every cfg change
  const syncOutput = useCallback(async (next: config.ClaudeConfig) => {
    try {
      const [json, result] = await Promise.all([
        GenerateClaudeConfig(next),
        ValidateClaudeConfig(next),
      ])
      setValidation(result)
      onValidation(result)
      onChange(json)
    } catch (err) {
      console.error('ClaudeConfigForm sync error:', err)
    }
  }, [onChange, onValidation])

  const update = (patch: Partial<config.ClaudeConfig>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, ...patch } as config.ClaudeConfig
      syncOutput(next)
      return next
    })
  }

  const updatePermissions = (patch: Partial<config.ClaudePermissions>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, permissions: { ...prev.permissions, ...patch } } as config.ClaudeConfig
      syncOutput(next)
      return next
    })
  }

  const updateSandbox = (patch: Partial<config.ClaudeSandbox>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, sandbox: { ...prev.sandbox, ...patch } } as config.ClaudeConfig
      syncOutput(next)
      return next
    })
  }

  const updateAdvanced = (patch: Partial<config.ClaudeAdvanced>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, advanced: { ...prev.advanced, ...patch } } as config.ClaudeConfig
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
      <PresetSelector tool="claude" onApply={(c) => {
        const typed = c as config.ClaudeConfig
        setCfg(typed)
        syncOutput(typed)
      }} />

      <hr className="border-border" />

      {/* Core */}
      <section id="claude-section-core" className="space-y-3">
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
          {fieldError('model') && (
            <p className="text-xs text-red-400">{fieldError('model')}</p>
          )}
        </div>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">API Key</label>
          <input
            type="password"
            value={cfg.apiKey || ''}
            onChange={(e) => update({ apiKey: e.target.value })}
            placeholder="sk-ant-..."
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
          />
          {fieldError('apiKey') && (
            <p className="text-xs text-red-400">{fieldError('apiKey')}</p>
          )}
        </div>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Max Tokens</label>
          <input
            type="number"
            min={0}
            max={200000}
            value={cfg.maxTokens || 0}
            onChange={(e) => update({ maxTokens: parseInt(e.target.value) || 0 })}
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
          />
          {fieldError('maxTokens') && (
            <p className="text-xs text-red-400">{fieldError('maxTokens')}</p>
          )}
        </div>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Custom Instructions</label>
          <textarea
            value={cfg.customInstructions || ''}
            onChange={(e) => update({ customInstructions: e.target.value })}
            placeholder="Instructions applied to all conversations..."
            rows={3}
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary resize-y"
          />
        </div>
      </section>

      <hr className="border-border" />

      {/* Permissions */}
      <section id="claude-section-permissions" className="space-y-2">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Permissions</h3>
        <SwitchField
          label="Allow Bash"
          description="Permit shell command execution"
          checked={cfg.permissions?.allowBash ?? true}
          onChange={(v) => updatePermissions({ allowBash: v })}
        />
        <SwitchField
          label="Allow Read"
          description="Permit file read operations"
          checked={cfg.permissions?.allowRead ?? true}
          onChange={(v) => updatePermissions({ allowRead: v })}
        />
        <SwitchField
          label="Allow Write"
          description="Permit file write operations"
          checked={cfg.permissions?.allowWrite ?? true}
          onChange={(v) => updatePermissions({ allowWrite: v })}
        />
        <SwitchField
          label="Allow Web Fetch"
          description="Permit outbound HTTP requests"
          checked={cfg.permissions?.allowWebFetch ?? false}
          onChange={(v) => updatePermissions({ allowWebFetch: v })}
        />
        <TagInput
          label="Trusted Directories"
          values={cfg.permissions?.trustedDirectories || []}
          onChange={(v) => updatePermissions({ trustedDirectories: v })}
          placeholder="e.g. /home/user/projects"
        />
        <TagInput
          label="Allowed Bash Commands"
          values={cfg.permissions?.allowedBashCommands || []}
          onChange={(v) => updatePermissions({ allowedBashCommands: v })}
          placeholder="e.g. git *"
        />
        <TagInput
          label="Denied Bash Commands"
          values={cfg.permissions?.deniedBashCommands || []}
          onChange={(v) => updatePermissions({ deniedBashCommands: v })}
          placeholder="e.g. rm -rf *"
        />
      </section>

      <hr className="border-border" />

      {/* Sandbox */}
      <section id="claude-section-sandbox" className="space-y-2">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Sandbox</h3>
        <SwitchField
          label="Enable Sandbox"
          checked={cfg.sandbox?.enabled ?? false}
          onChange={(v) => updateSandbox({ enabled: v })}
        />
        {cfg.sandbox?.enabled && (
          <>
            <SelectField
              label="Sandbox Type"
              value={cfg.sandbox?.type || 'none'}
              options={SANDBOX_OPTIONS}
              onChange={(v) => updateSandbox({ type: v })}
            />
            {fieldError('sandbox.type') && (
              <p className="text-xs text-red-400">{fieldError('sandbox.type')}</p>
            )}
            {cfg.sandbox?.type === 'docker' && (
              <div className="space-y-1">
                <label className="text-xs font-medium text-muted-foreground">Docker Image</label>
                <input
                  type="text"
                  value={cfg.sandbox?.dockerImage || ''}
                  onChange={(e) => updateSandbox({ dockerImage: e.target.value })}
                  placeholder="e.g. ubuntu:22.04"
                  className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
                />
              </div>
            )}
          </>
        )}
      </section>

      <hr className="border-border" />

      {/* Advanced */}
      <section id="claude-section-advanced" className="space-y-2">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Advanced</h3>
        <SwitchField
          label="Verbose Logging"
          checked={cfg.advanced?.verbose ?? false}
          onChange={(v) => updateAdvanced({ verbose: v })}
        />
        <SwitchField
          label="Disable Telemetry"
          checked={cfg.advanced?.disableTelemetry ?? false}
          onChange={(v) => updateAdvanced({ disableTelemetry: v })}
        />
        <SwitchField
          label="Experimental Features"
          checked={cfg.advanced?.experimentalFeatures ?? false}
          onChange={(v) => updateAdvanced({ experimentalFeatures: v })}
        />

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">API Endpoint</label>
          <input
            type="url"
            value={cfg.advanced?.apiEndpoint || ''}
            onChange={(e) => updateAdvanced({ apiEndpoint: e.target.value })}
            placeholder="https://proxy.example.com"
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
          />
          <EndpointPresetPicker
            localURL={localGatewayURL}
            value={cfg.advanced?.apiEndpoint || ''}
            onChange={(url) => updateAdvanced({ apiEndpoint: url })}
          />
          {fieldError('advanced.apiEndpoint') && (
            <p className="text-xs text-red-400">{fieldError('advanced.apiEndpoint')}</p>
          )}
        </div>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Timeout (seconds)</label>
          <input
            type="number"
            min={0}
            max={3600}
            value={cfg.advanced?.timeout || 0}
            onChange={(e) => updateAdvanced({ timeout: parseInt(e.target.value) || 0 })}
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
          />
          {fieldError('advanced.timeout') && (
            <p className="text-xs text-red-400">{fieldError('advanced.timeout')}</p>
          )}
        </div>
      </section>

      {/* Validation summary */}
      <ValidationPanel result={validation} showSuccess />
    </div>
  )
}
