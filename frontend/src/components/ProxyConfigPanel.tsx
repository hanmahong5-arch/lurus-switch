import { useState } from 'react'
import { Save, Settings2, Loader2, ExternalLink } from 'lucide-react'
import { cn } from '../lib/utils'
import type { ProxySettings } from '../stores/dashboardStore'

interface ProxyConfigPanelProps {
  settings: ProxySettings
  saving: boolean
  configuring: boolean
  onSave: (settings: ProxySettings) => void
  onConfigureAll: () => void
}

export function ProxyConfigPanel({ settings, saving, configuring, onSave, onConfigureAll }: ProxyConfigPanelProps) {
  const [endpoint, setEndpoint] = useState(settings.apiEndpoint)
  const [apiKey, setApiKey] = useState(settings.apiKey)
  const [registrationUrl] = useState(settings.registrationUrl || '')

  const hasChanges = endpoint !== settings.apiEndpoint || apiKey !== settings.apiKey
  const hasValues = endpoint.trim() !== '' && apiKey.trim() !== ''

  const handleSave = () => {
    onSave({
      apiEndpoint: endpoint.trim(),
      apiKey: apiKey.trim(),
      registrationUrl,
    })
  }

  return (
    <div className="border border-border rounded-lg p-4 bg-card">
      <h3 className="text-sm font-medium mb-3 flex items-center gap-2">
        <Settings2 className="h-4 w-4 text-purple-500" />
        NewAPI Proxy Configuration
      </h3>

      <div className="space-y-3">
        {/* API Endpoint */}
        <div>
          <label className="block text-xs text-muted-foreground mb-1">API Endpoint</label>
          <input
            type="url"
            value={endpoint}
            onChange={(e) => setEndpoint(e.target.value)}
            placeholder="https://api.example.com/v1"
            className="w-full px-3 py-1.5 text-sm rounded-md border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        {/* API Key */}
        <div>
          <label className="block text-xs text-muted-foreground mb-1">API Key</label>
          <input
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder="sk-..."
            className="w-full px-3 py-1.5 text-sm rounded-md border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        {/* Registration link */}
        {registrationUrl && (
          <a
            href={registrationUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
          >
            <ExternalLink className="h-3 w-3" />
            Register for an API key
          </a>
        )}

        {/* Actions */}
        <div className="flex gap-2 pt-1">
          <button
            onClick={handleSave}
            disabled={saving || !hasChanges}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
              'bg-primary text-primary-foreground hover:bg-primary/90',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            {saving ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Save className="h-3.5 w-3.5" />
            )}
            Save Settings
          </button>

          <button
            onClick={onConfigureAll}
            disabled={configuring || !hasValues}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
              'border border-border hover:bg-muted',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            {configuring ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Settings2 className="h-3.5 w-3.5" />
            )}
            Apply to All Tools
          </button>
        </div>
      </div>
    </div>
  )
}
