import { useEffect, useState, useCallback } from 'react'
import { Loader2 } from 'lucide-react'
import { config, validator } from '../../../wailsjs/go/models'
import {
  GetDefaultGeminiConfig,
  GenerateGeminiConfig,
  ValidateGeminiConfig,
} from '../../../wailsjs/go/main/App'
import { BareToggle } from './SwitchField'
import { TagInput } from './TagInput'
import { FieldRow } from './FieldRow'
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

const inputCls = 'w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary'
const monoInputCls = inputCls + ' font-mono'

export function GeminiConfigForm({ initialContent, onChange, onValidation }: GeminiConfigFormProps) {
  const [cfg, setCfg] = useState<config.GeminiConfig | null>(null)
  const [validation, setValidation] = useState<validator.ValidationResult | null>(null)
  const gwStatus = useGatewayStore((s) => s.status)
  const localGatewayURL = gwStatus?.running ? `http://localhost:${gwStatus.port}/v1` : null

  useEffect(() => {
    let parsed: Partial<config.GeminiConfig> = {}
    try { parsed = JSON.parse(initialContent || '{}') } catch { /* ignore */ }
    GetDefaultGeminiConfig().then((defaults) => {
      setCfg({ ...defaults, ...parsed } as config.GeminiConfig)
    }).catch(() => setCfg(parsed as config.GeminiConfig))
  }, [initialContent])

  const syncOutput = useCallback(async (next: config.GeminiConfig) => {
    try {
      const [parts, result] = await Promise.all([GenerateGeminiConfig(next), ValidateGeminiConfig(next)])
      setValidation(result)
      onValidation(result)
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
      syncOutput(next); return next
    })
  }
  const updateBehavior = (patch: Partial<config.GeminiBehavior>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, behavior: { ...prev.behavior, ...patch } } as config.GeminiConfig
      syncOutput(next); return next
    })
  }
  const updateInstructions = (patch: Partial<config.GeminiInstructions>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, instructions: { ...prev.instructions, ...patch } } as config.GeminiConfig
      syncOutput(next); return next
    })
  }
  const updateDisplay = (patch: Partial<config.GeminiDisplay>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, display: { ...prev.display, ...patch } } as config.GeminiConfig
      syncOutput(next); return next
    })
  }
  const updateAdvanced = (patch: Partial<config.GeminiAdvanced>) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = { ...prev, advanced: { ...prev.advanced, ...patch } } as config.GeminiConfig
      syncOutput(next); return next
    })
  }

  if (!cfg) return <div className="flex items-center justify-center h-32"><Loader2 className="h-5 w-5 animate-spin text-muted-foreground" /></div>

  const fieldError = (field: string): string | undefined =>
    validation?.errors?.find((e) => e.field === field)?.message

  return (
    <div className="space-y-4 p-4">
      <PresetSelector tool="gemini" onApply={(c) => { const typed = c as config.GeminiConfig; setCfg(typed); syncOutput(typed) }} />

      <Section id="gemini-section-core" titleZh="核心设置" titleEn="Core">
        <FieldRow metaKey="gemini.model" value={cfg.model} errorMessage={fieldError('model')}>
          <select value={cfg.model || ''} onChange={(e) => update({ model: e.target.value })} className={inputCls}>
            {MODEL_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </FieldRow>
        <FieldRow metaKey="gemini.projectId" value={cfg.projectId}>
          <input type="text" value={cfg.projectId || ''} onChange={(e) => update({ projectId: e.target.value })}
            placeholder="my-gcp-project" className={monoInputCls} />
        </FieldRow>
      </Section>

      <Section id="gemini-section-auth" titleZh="鉴权" titleEn="Authentication">
        <FieldRow metaKey="gemini.auth.type" value={cfg.auth?.type}>
          <select value={cfg.auth?.type || 'api_key'} onChange={(e) => updateAuth({ type: e.target.value })} className={inputCls}>
            {AUTH_TYPE_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </FieldRow>
        {cfg.auth?.type === 'api_key' && (
          <FieldRow metaKey="gemini.apiKey" value={cfg.apiKey} errorMessage={fieldError('apiKey')}>
            <input type="password" value={cfg.apiKey || ''} onChange={(e) => update({ apiKey: e.target.value })}
              placeholder="AIza..." className={monoInputCls} />
          </FieldRow>
        )}
        {cfg.auth?.type === 'oauth' && (
          <FieldRow metaKey="gemini.auth.oauthClientId" value={cfg.auth?.oauthClientId}>
            <input type="text" value={cfg.auth?.oauthClientId || ''} onChange={(e) => updateAuth({ oauthClientId: e.target.value })}
              placeholder="123456789-xxxx.apps.googleusercontent.com" className={monoInputCls} />
          </FieldRow>
        )}
        {cfg.auth?.type === 'service_account' && (
          <FieldRow metaKey="gemini.auth.serviceAccountPath" value={cfg.auth?.serviceAccountPath}>
            <input type="text" value={cfg.auth?.serviceAccountPath || ''} onChange={(e) => updateAuth({ serviceAccountPath: e.target.value })}
              placeholder="/path/to/service-account.json" className={monoInputCls} />
          </FieldRow>
        )}
      </Section>

      <Section id="gemini-section-behavior" titleZh="行为" titleEn="Behavior">
        <FieldRow metaKey="gemini.behavior.sandbox" value={cfg.behavior?.sandbox ?? false} layout="inline">
          <BareToggle checked={cfg.behavior?.sandbox ?? false} onChange={(v) => updateBehavior({ sandbox: v })} />
        </FieldRow>
        <FieldRow metaKey="gemini.behavior.yoloMode" value={cfg.behavior?.yoloMode ?? false} layout="inline">
          <BareToggle checked={cfg.behavior?.yoloMode ?? false} onChange={(v) => updateBehavior({ yoloMode: v })} />
        </FieldRow>
        {!cfg.behavior?.yoloMode && (
          <FieldRow metaKey="gemini.behavior.autoApprove" value={cfg.behavior?.autoApprove || []}>
            <TagInput label="" values={cfg.behavior?.autoApprove || []} onChange={(v) => updateBehavior({ autoApprove: v })}
              placeholder="e.g. read_file" />
          </FieldRow>
        )}
        <FieldRow metaKey="gemini.behavior.maxFileSize" value={cfg.behavior?.maxFileSize}>
          <input type="number" min={0} value={cfg.behavior?.maxFileSize || 0}
            onChange={(e) => updateBehavior({ maxFileSize: parseInt(e.target.value) || 0 })} className={inputCls} />
        </FieldRow>
        <FieldRow metaKey="gemini.behavior.allowedExtensions" value={cfg.behavior?.allowedExtensions || []}>
          <TagInput label="" values={cfg.behavior?.allowedExtensions || []}
            onChange={(v) => updateBehavior({ allowedExtensions: v })} placeholder="e.g. .ts" />
        </FieldRow>
      </Section>

      <Section id="gemini-section-instructions" titleZh="指令" titleEn="Instructions">
        <FieldRow metaKey="gemini.instructions.projectDescription" value={cfg.instructions?.projectDescription}>
          <textarea value={cfg.instructions?.projectDescription || ''}
            onChange={(e) => updateInstructions({ projectDescription: e.target.value })}
            placeholder="Describe your project..." rows={2} className={inputCls + ' resize-y'} />
        </FieldRow>
        <FieldRow metaKey="gemini.instructions.techStack" value={cfg.instructions?.techStack}>
          <input type="text" value={cfg.instructions?.techStack || ''}
            onChange={(e) => updateInstructions({ techStack: e.target.value })}
            placeholder="e.g. Go, React, PostgreSQL" className={inputCls} />
        </FieldRow>
        <FieldRow metaKey="gemini.instructions.codeStyle" value={cfg.instructions?.codeStyle}>
          <input type="text" value={cfg.instructions?.codeStyle || ''}
            onChange={(e) => updateInstructions({ codeStyle: e.target.value })}
            placeholder="e.g. Google style, 2-space indent" className={inputCls} />
        </FieldRow>
        <FieldRow metaKey="gemini.instructions.customRules" value={cfg.instructions?.customRules || []}>
          <TagInput label="" values={cfg.instructions?.customRules || []}
            onChange={(v) => updateInstructions({ customRules: v })} placeholder="e.g. Always write tests" />
        </FieldRow>
      </Section>

      <Section id="gemini-section-display" titleZh="显示" titleEn="Display">
        <FieldRow metaKey="gemini.display.theme" value={cfg.display?.theme}>
          <select value={cfg.display?.theme || 'dark'} onChange={(e) => updateDisplay({ theme: e.target.value })} className={inputCls}>
            {THEME_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </FieldRow>
        <FieldRow metaKey="gemini.display.syntaxHighlight" value={cfg.display?.syntaxHighlight ?? true} layout="inline">
          <BareToggle checked={cfg.display?.syntaxHighlight ?? true} onChange={(v) => updateDisplay({ syntaxHighlight: v })} />
        </FieldRow>
        <FieldRow metaKey="gemini.display.markdownRender" value={cfg.display?.markdownRender ?? true} layout="inline">
          <BareToggle checked={cfg.display?.markdownRender ?? true} onChange={(v) => updateDisplay({ markdownRender: v })} />
        </FieldRow>
      </Section>

      <Section id="gemini-section-advanced" titleZh="高级" titleEn="Advanced">
        <FieldRow metaKey="gemini.advanced.apiEndpoint" value={cfg.advanced?.apiEndpoint} errorMessage={fieldError('advanced.apiEndpoint')}>
          <div className="space-y-1">
            <input type="url" value={cfg.advanced?.apiEndpoint || ''}
              onChange={(e) => updateAdvanced({ apiEndpoint: e.target.value })}
              placeholder="https://proxy.example.com" className={monoInputCls} />
            <EndpointPresetPicker localURL={localGatewayURL} value={cfg.advanced?.apiEndpoint || ''}
              onChange={(url) => updateAdvanced({ apiEndpoint: url })} />
          </div>
        </FieldRow>
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
