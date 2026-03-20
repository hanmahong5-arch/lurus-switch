import { useTranslation } from 'react-i18next'
import { Server, Globe } from 'lucide-react'
import { cn } from '../../lib/utils'

interface EndpointPreset {
  label: string
  url: string
  icon: React.ReactNode
  description: string
}

interface EndpointPresetPickerProps {
  localURL: string | null        // null when local server not running
  platformURL?: string
  value: string
  onChange: (url: string) => void
}

export function EndpointPresetPicker({ localURL, platformURL = 'https://api.lurus.cn', value, onChange }: EndpointPresetPickerProps) {
  const { t } = useTranslation()

  const presets: EndpointPreset[] = []

  if (localURL) {
    presets.push({
      label: t('gateway.localServer', 'Local Server'),
      url: localURL,
      icon: <Server className="h-3.5 w-3.5" />,
      description: t('gateway.localServerDesc', 'Use your embedded local gateway'),
    })
  }

  presets.push({
    label: t('gateway.lurusPlatform', 'Lurus Platform'),
    url: platformURL,
    icon: <Globe className="h-3.5 w-3.5" />,
    description: t('gateway.lurusPlatformDesc', 'Use the Lurus cloud relay'),
  })

  if (presets.length === 0) return null

  return (
    <div className="flex flex-wrap gap-2 mt-1.5">
      {presets.map((preset) => (
        <button
          key={preset.url}
          type="button"
          title={preset.description}
          onClick={() => onChange(preset.url)}
          className={cn(
            'flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs border transition-colors',
            value === preset.url
              ? 'border-indigo-500 bg-indigo-500/20 text-indigo-300'
              : 'border-border hover:border-indigo-400 hover:bg-muted/60 text-muted-foreground'
          )}
        >
          {preset.icon}
          {preset.label}
        </button>
      ))}
    </div>
  )
}
