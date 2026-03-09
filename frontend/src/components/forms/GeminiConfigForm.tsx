import { useEffect, useState, useCallback } from 'react'
import { Loader2 } from 'lucide-react'
import { config, validator } from '../../../wailsjs/go/models'
import {
  GetDefaultGeminiConfig,
  GenerateGeminiConfig,
  ValidateGeminiConfig,
} from '../../../wailsjs/go/main/App'
import { SwitchField } from './SwitchField'
import { SelectField } from './SelectField'
import { TagInput } from './TagInput'
import { ValidationPanel } from '../ValidationPanel'
import { PresetSelector } from '../PresetSelector'
import { EndpointPresetPicker } from './EndpointPresetPicker'
import { useGatewayStore } from '../../stores/gatewayStore'

interface GeminiConfigFormProps {
  initialContent: string
  onChange: (json: string) => void
  onValidation: (result: validator.ValidationResult | null) => void
}

const MODEL_OPTIONS = [
  { value: 'gemini-2.5-flash', label: 'Gemini 2.5 Flash' },
  { value: 'gemini-2.5-pro', label: 'Gemini 2.5 Pro' },
  { value: 'gemini-2.0-flash', label: 'Gemini 2.0 Flash' },
  { value: 'gemini-1.5-pro', label: 'Gemini 1.5 Pro' },
  { value: 'gemini-1.5-flash', label: 'Gemini 1.5 Flash' },
]

const AUTH_TYPE_OPTIONS = [
  { value: 'api_key', label: 'API Key' },
  { value: 'oauth', label: 'OAuth' },
  { value: 'service_account', label: 'Service Account' },
  { value: 'adc', label: 'Application Default Credentials' },
]

const THEME_OPTIONS = [
  { value: 'dark', label: 'Dark' },
  { value: 'light', label: 'Light' },
  { value: 'system', label: 'System' },
]

export function GeminiConfigForm({ initialContent, onChange, onValidation }: GeminiConfigFormProps) {
  const [cfg, setCfg] = useState<config.GeminiConfig | null>(null)
  const [validation, setValidation] = useState<validator.ValidationResult | null>(null)
  const gwStatus = useGatewayStore((s) => s.status)
  const localGatewayURL = gwStatus?.running ? `http://localhost:${gwStatus.port}/v1` : null

  useEffect(() => {
    let parsed: Partial<config.GeminiConfig> = {}
    try {
      parsed = JSON.parse(initialContent || '{}')
    } catch {
      // ignore
    }
    GetDefaultGeminiConfig().then((defaults) => {
      setCfg({ ...defaults, ...parsed } as config.GeminiConfig)
    }).catch(() => {
      setCfg(parsed as config.GeminiConfig)
    })
  }, [initialContent])

  const syncOutput = useCallback(async (next: config.GeminiConfig) => {
    try {
      const [parts, result] = await Promise.all([
        GenerateGeminiConfig(next),
        ValidateGeminiConfig(next),
      ])
      setValidation(result)
      onValidation(result)
      // GenerateGeminiConfig returns []string (multi-file); join as JSON array for display
      // In form mode we emit the first (settings.json) segment
      onChange(parts?.[0] || '{}')
    } catch (err) {
      console.error('GeminiConfigForm sync error:', err)
    }
  }, [onChange, onValidation])

  const update = (patch: Partial<config.GeminiConfig>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, ...patch } as config.GeminiConfig
      syncOutput(next)
      return next
    })
  }

  const updateAuth = (patch: Partial<config.GeminiAuth>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, auth: { ...prev.auth, ...patch } } as config.GeminiConfig
      syncOutput(next)
      return next
    })
  }

  const updateBehavior = (patch: Partial<config.GeminiBehavior>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, behavior: { ...prev.behavior, ...patch } } as config.GeminiConfig
      syncOutput(next)
      return next
    })
  }

  const updateInstructions = (patch: Partial<config.GeminiInstructions>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, instructions: { ...prev.instructions, ...patch } } as config.GeminiConfig
      syncOutput(next)
      return next
    })
  }

  const updateDisplay = (patch: Partial<config.GeminiDisplay>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, display: { ...prev.display, ...patch } } as config.GeminiConfig
      syncOutput(next)
      return next
    })
  }

  const updateAdvanced = (patch: Partial<config.GeminiAdvanced>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, advanced: { ...prev.advanced, ...patch } } as config.GeminiConfig
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
      <PresetSelector tool="gemini" onApply={(c) => {
        const typed = c as config.GeminiConfig
        setCfg(typed)
        syncOutput(typed)
      }} />

      <hr className="border-border" />

      {/* Core */}
      <section id="gemini-section-core" className="space-y-3">
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
          <label className="text-xs font-medium text-muted-foreground">Project ID</label>
          <input
            type="text"
            value={cfg.projectId || ''}
            onChange={(e) => update({ projectId: e.target.value })}
            placeholder="my-gcp-project"
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
          />
        </div>
      </section>

      <hr className="border-border" />

      {/* Auth */}
      <section id="gemini-section-auth" className="space-y-3">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Authentication</h3>

        <SelectField
          label="Auth Type"
          value={cfg.auth?.type || 'api_key'}
          options={AUTH_TYPE_OPTIONS}
          onChange={(v) => updateAuth({ type: v })}
        />

        {cfg.auth?.type === 'api_key' && (
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground">API Key</label>
            <input
              type="password"
              value={cfg.apiKey || ''}
              onChange={(e) => update({ apiKey: e.target.value })}
              placeholder="AIza..."
              className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
            />
            {fieldError('apiKey') && <p className="text-xs text-red-400">{fieldError('apiKey')}</p>}
          </div>
        )}

        {cfg.auth?.type === 'oauth' && (
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground">OAuth Client ID</label>
            <input
              type="text"
              value={cfg.auth?.oauthClientId || ''}
              onChange={(e) => updateAuth({ oauthClientId: e.target.value })}
              placeholder="123456789-xxxx.apps.googleusercontent.com"
              className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
            />
          </div>
        )}

        {cfg.auth?.type === 'service_account' && (
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground">Service Account Path</label>
            <input
              type="text"
              value={cfg.auth?.serviceAccountPath || ''}
              onChange={(e) => updateAuth({ serviceAccountPath: e.target.value })}
              placeholder="/path/to/service-account.json"
              className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
            />
          </div>
        )}
      </section>

      <hr className="border-border" />

      {/* Behavior */}
      <section id="gemini-section-behavior" className="space-y-2">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Behavior</h3>
        <SwitchField
          label="Sandbox"
          description="Run tools in a sandboxed environment"
          checked={cfg.behavior?.sandbox ?? false}
          onChange={(v) => updateBehavior({ sandbox: v })}
        />
        <SwitchField
          label="YOLO Mode"
          description="Auto-approve all tool executions"
          checked={cfg.behavior?.yoloMode ?? false}
          onChange={(v) => updateBehavior({ yoloMode: v })}
        />
        {!cfg.behavior?.yoloMode && (
          <TagInput
            label="Auto-Approve Patterns"
            values={cfg.behavior?.autoApprove || []}
            onChange={(v) => updateBehavior({ autoApprove: v })}
            placeholder="e.g. read_file"
          />
        )}
        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Max File Size (bytes)</label>
          <input
            type="number"
            min={0}
            value={cfg.behavior?.maxFileSize || 0}
            onChange={(e) => updateBehavior({ maxFileSize: parseInt(e.target.value) || 0 })}
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
        <TagInput
          label="Allowed Extensions"
          values={cfg.behavior?.allowedExtensions || []}
          onChange={(v) => updateBehavior({ allowedExtensions: v })}
          placeholder="e.g. .ts"
        />
      </section>

      <hr className="border-border" />

      {/* Instructions */}
      <section className="space-y-3">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Instructions</h3>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Project Description</label>
          <textarea
            value={cfg.instructions?.projectDescription || ''}
            onChange={(e) => updateInstructions({ projectDescription: e.target.value })}
            placeholder="Describe your project..."
            rows={2}
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary resize-y"
          />
        </div>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Tech Stack</label>
          <input
            type="text"
            value={cfg.instructions?.techStack || ''}
            onChange={(e) => updateInstructions({ techStack: e.target.value })}
            placeholder="e.g. Go, React, PostgreSQL"
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Code Style</label>
          <input
            type="text"
            value={cfg.instructions?.codeStyle || ''}
            onChange={(e) => updateInstructions({ codeStyle: e.target.value })}
            placeholder="e.g. Google style, 2-space indent"
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <TagInput
          label="Custom Rules"
          values={cfg.instructions?.customRules || []}
          onChange={(v) => updateInstructions({ customRules: v })}
          placeholder="e.g. Always write tests"
        />
      </section>

      <hr className="border-border" />

      {/* Display */}
      <section className="space-y-2">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Display</h3>
        <SelectField
          label="Theme"
          value={cfg.display?.theme || 'dark'}
          options={THEME_OPTIONS}
          onChange={(v) => updateDisplay({ theme: v })}
        />
        <SwitchField
          label="Syntax Highlighting"
          checked={cfg.display?.syntaxHighlight ?? true}
          onChange={(v) => updateDisplay({ syntaxHighlight: v })}
        />
        <SwitchField
          label="Markdown Rendering"
          checked={cfg.display?.markdownRender ?? true}
          onChange={(v) => updateDisplay({ markdownRender: v })}
        />
      </section>

      <hr className="border-border" />

      {/* Advanced */}
      <section id="gemini-section-advanced" className="space-y-2">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Advanced</h3>
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
      </section>

      <ValidationPanel result={validation} showSuccess />
    </div>
  )
}
