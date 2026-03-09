import { useState, useEffect } from 'react'
import { Loader2, Wand2 } from 'lucide-react'
import { cn } from '../lib/utils'
import { preset } from '../../wailsjs/go/models'
import {
  GetClaudePresets, ApplyClaudePreset,
  GetCodexPresets, ApplyCodexPreset,
  GetGeminiPresets, ApplyGeminiPreset,
} from '../../wailsjs/go/main/App'

type SupportedTool = 'claude' | 'codex' | 'gemini'

interface PresetSelectorProps {
  tool: SupportedTool
  /** Called with the preset config object after the backend applies the preset. */
  onApply: (cfg: any) => void
  disabled?: boolean
}

/**
 * Displays preset cards for claude/codex/gemini.
 * Clicking a card fetches the preset from Go and calls onApply with the filled config.
 */
export function PresetSelector({ tool, onApply, disabled }: PresetSelectorProps) {
  const [presets, setPresets] = useState<preset.Preset[]>([])
  const [applying, setApplying] = useState<string | null>(null)

  useEffect(() => {
    const fetchers: Record<SupportedTool, () => Promise<preset.Preset[]>> = {
      claude: GetClaudePresets,
      codex: GetCodexPresets,
      gemini: GetGeminiPresets,
    }
    fetchers[tool]().then(setPresets).catch(() => setPresets([]))
  }, [tool])

  const handleApply = async (id: string) => {
    if (disabled || applying) return
    setApplying(id)
    try {
      let cfg: any
      if (tool === 'claude') cfg = await ApplyClaudePreset(id)
      else if (tool === 'codex') cfg = await ApplyCodexPreset(id)
      else cfg = await ApplyGeminiPreset(id)
      onApply(cfg)
    } catch (err) {
      console.error('Failed to apply preset:', err)
    } finally {
      setApplying(null)
    }
  }

  if (presets.length === 0) return null

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
        <Wand2 className="h-3.5 w-3.5" />
        Presets
      </div>
      <div className="grid grid-cols-2 gap-2">
        {presets.map((p) => (
          <button
            key={p.id}
            type="button"
            onClick={() => handleApply(p.id)}
            disabled={disabled || applying !== null}
            className={cn(
              'text-left px-3 py-2 rounded-md border border-border bg-muted/30',
              'hover:bg-muted hover:border-primary/50 transition-colors',
              'disabled:opacity-50 disabled:cursor-not-allowed',
              'focus:outline-none focus:ring-1 focus:ring-primary'
            )}
          >
            <div className="flex items-center gap-1.5">
              {applying === p.id && (
                <Loader2 className="h-3 w-3 animate-spin shrink-0" />
              )}
              <span className="text-xs font-medium">{p.name}</span>
            </div>
            <p className="text-xs text-muted-foreground mt-0.5 leading-relaxed">{p.description}</p>
          </button>
        ))}
      </div>
    </div>
  )
}
