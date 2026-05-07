import { useEffect, useState, useCallback } from 'react'
import { Loader2 } from 'lucide-react'
import { config, validator } from '../../../wailsjs/go/models'
import {
  GetDefaultClaudeConfig,
  GenerateClaudeConfig,
  ValidateClaudeConfig,
} from '../../../wailsjs/go/main/App'
import { BareToggle } from './SwitchField'
import { TagInput } from './TagInput'
import { FieldRow } from './FieldRow'
import { ValidationPanel } from '../ValidationPanel'
import { PresetSelector } from '../PresetSelector'
import { EndpointPresetPicker } from './EndpointPresetPicker'
import { useGatewayStore } from '../../stores/gatewayStore'

interface ClaudeConfigFormProps {
  initialContent: string
  onChange: (json: string) => void
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

const inputCls =
  'w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary'
const monoInputCls = inputCls + ' font-mono'

export function ClaudeConfigForm({ initialContent, onChange, onValidation }: ClaudeConfigFormProps) {
  const [cfg, setCfg] = useState<config.ClaudeConfig | null>(null)
  const [validation, setValidation] = useState<validator.ValidationResult | null>(null)
  const gwStatus = useGatewayStore((s) => s.status)
  const localGatewayURL = gwStatus?.running ? `http://localhost:${gwStatus.port}/v1` : null

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
    <div className="space-y-4 p-4">
      <PresetSelector tool="claude" onApply={(c) => {
        const typed = c as config.ClaudeConfig
        setCfg(typed)
        syncOutput(typed)
      }} />

      <Section id="claude-section-core" titleZh="核心设置" titleEn="Core">
        <FieldRow metaKey="claude.model" value={cfg.model} errorMessage={fieldError('model')}>
          <select
            value={cfg.model || ''}
            onChange={(e) => update({ model: e.target.value })}
            className={inputCls}
          >
            {MODEL_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
        </FieldRow>

        <FieldRow metaKey="claude.apiKey" value={cfg.apiKey} errorMessage={fieldError('apiKey')}>
          <input
            type="password"
            value={cfg.apiKey || ''}
            onChange={(e) => update({ apiKey: e.target.value })}
            placeholder="sk-ant-..."
            className={monoInputCls}
          />
        </FieldRow>

        <FieldRow metaKey="claude.maxTokens" value={cfg.maxTokens} errorMessage={fieldError('maxTokens')}>
          <input
            type="number"
            min={0}
            max={200000}
            value={cfg.maxTokens || 0}
            onChange={(e) => update({ maxTokens: parseInt(e.target.value) || 0 })}
            className={inputCls}
          />
        </FieldRow>

        <FieldRow metaKey="claude.customInstructions" value={cfg.customInstructions}>
          <textarea
            value={cfg.customInstructions || ''}
            onChange={(e) => update({ customInstructions: e.target.value })}
            placeholder="Instructions applied to all conversations..."
            rows={3}
            className={inputCls + ' resize-y'}
          />
        </FieldRow>
      </Section>

      <Section id="claude-section-permissions" titleZh="权限" titleEn="Permissions">
        <FieldRow
          metaKey="claude.permissions.allowBash"
          value={cfg.permissions?.allowBash ?? true}
          layout="inline"
        >
          <BareToggle
            checked={cfg.permissions?.allowBash ?? true}
            onChange={(v) => updatePermissions({ allowBash: v })}
          />
        </FieldRow>
        <FieldRow
          metaKey="claude.permissions.allowRead"
          value={cfg.permissions?.allowRead ?? true}
          layout="inline"
        >
          <BareToggle
            checked={cfg.permissions?.allowRead ?? true}
            onChange={(v) => updatePermissions({ allowRead: v })}
          />
        </FieldRow>
        <FieldRow
          metaKey="claude.permissions.allowWrite"
          value={cfg.permissions?.allowWrite ?? true}
          layout="inline"
        >
          <BareToggle
            checked={cfg.permissions?.allowWrite ?? true}
            onChange={(v) => updatePermissions({ allowWrite: v })}
          />
        </FieldRow>
        <FieldRow
          metaKey="claude.permissions.allowWebFetch"
          value={cfg.permissions?.allowWebFetch ?? false}
          layout="inline"
        >
          <BareToggle
            checked={cfg.permissions?.allowWebFetch ?? false}
            onChange={(v) => updatePermissions({ allowWebFetch: v })}
          />
        </FieldRow>
        <FieldRow
          metaKey="claude.permissions.trustedDirectories"
          value={cfg.permissions?.trustedDirectories || []}
        >
          <TagInput
            label=""
            values={cfg.permissions?.trustedDirectories || []}
            onChange={(v) => updatePermissions({ trustedDirectories: v })}
            placeholder="e.g. /home/user/projects"
          />
        </FieldRow>
        <FieldRow
          metaKey="claude.permissions.allowedBashCommands"
          value={cfg.permissions?.allowedBashCommands || []}
        >
          <TagInput
            label=""
            values={cfg.permissions?.allowedBashCommands || []}
            onChange={(v) => updatePermissions({ allowedBashCommands: v })}
            placeholder="e.g. git *"
          />
        </FieldRow>
        <FieldRow
          metaKey="claude.permissions.deniedBashCommands"
          value={cfg.permissions?.deniedBashCommands || []}
        >
          <TagInput
            label=""
            values={cfg.permissions?.deniedBashCommands || []}
            onChange={(v) => updatePermissions({ deniedBashCommands: v })}
            placeholder="e.g. rm -rf *"
          />
        </FieldRow>
      </Section>

      <Section id="claude-section-sandbox" titleZh="沙箱" titleEn="Sandbox">
        <FieldRow
          metaKey="claude.sandbox.enabled"
          value={cfg.sandbox?.enabled ?? false}
          layout="inline"
        >
          <BareToggle
            checked={cfg.sandbox?.enabled ?? false}
            onChange={(v) => updateSandbox({ enabled: v })}
          />
        </FieldRow>
        {cfg.sandbox?.enabled && (
          <>
            <FieldRow
              metaKey="claude.sandbox.type"
              value={cfg.sandbox?.type}
              errorMessage={fieldError('sandbox.type')}
            >
              <select
                value={cfg.sandbox?.type || 'none'}
                onChange={(e) => updateSandbox({ type: e.target.value })}
                className={inputCls}
              >
                {SANDBOX_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>{o.label}</option>
                ))}
              </select>
            </FieldRow>
            {cfg.sandbox?.type === 'docker' && (
              <FieldRow metaKey="claude.sandbox.dockerImage" value={cfg.sandbox?.dockerImage}>
                <input
                  type="text"
                  value={cfg.sandbox?.dockerImage || ''}
                  onChange={(e) => updateSandbox({ dockerImage: e.target.value })}
                  placeholder="e.g. ubuntu:22.04"
                  className={monoInputCls}
                />
              </FieldRow>
            )}
          </>
        )}
      </Section>

      <Section id="claude-section-advanced" titleZh="高级设置" titleEn="Advanced">
        <FieldRow
          metaKey="claude.advanced.verbose"
          value={cfg.advanced?.verbose ?? false}
          layout="inline"
        >
          <BareToggle
            checked={cfg.advanced?.verbose ?? false}
            onChange={(v) => updateAdvanced({ verbose: v })}
          />
        </FieldRow>
        <FieldRow
          metaKey="claude.advanced.disableTelemetry"
          value={cfg.advanced?.disableTelemetry ?? false}
          layout="inline"
        >
          <BareToggle
            checked={cfg.advanced?.disableTelemetry ?? false}
            onChange={(v) => updateAdvanced({ disableTelemetry: v })}
          />
        </FieldRow>
        <FieldRow
          metaKey="claude.advanced.experimentalFeatures"
          value={cfg.advanced?.experimentalFeatures ?? false}
          layout="inline"
        >
          <BareToggle
            checked={cfg.advanced?.experimentalFeatures ?? false}
            onChange={(v) => updateAdvanced({ experimentalFeatures: v })}
          />
        </FieldRow>
        <FieldRow
          metaKey="claude.advanced.apiEndpoint"
          value={cfg.advanced?.apiEndpoint}
          errorMessage={fieldError('advanced.apiEndpoint')}
        >
          <div className="space-y-1">
            <input
              type="url"
              value={cfg.advanced?.apiEndpoint || ''}
              onChange={(e) => updateAdvanced({ apiEndpoint: e.target.value })}
              placeholder="https://proxy.example.com"
              className={monoInputCls}
            />
            <EndpointPresetPicker
              localURL={localGatewayURL}
              value={cfg.advanced?.apiEndpoint || ''}
              onChange={(url) => updateAdvanced({ apiEndpoint: url })}
            />
          </div>
        </FieldRow>
        <FieldRow
          metaKey="claude.advanced.timeout"
          value={cfg.advanced?.timeout}
          errorMessage={fieldError('advanced.timeout')}
        >
          <input
            type="number"
            min={0}
            max={3600}
            value={cfg.advanced?.timeout || 0}
            onChange={(e) => updateAdvanced({ timeout: parseInt(e.target.value) || 0 })}
            className={inputCls}
          />
        </FieldRow>
      </Section>

      <ValidationPanel result={validation} showSuccess />
    </div>
  )
}

function Section({
  id,
  titleZh,
  titleEn,
  children,
}: {
  id: string
  titleZh: string
  titleEn: string
  children: React.ReactNode
}) {
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
