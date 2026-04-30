import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Save, Settings2, Loader2, ExternalLink, CheckCircle2, WifiOff, Wifi, Sparkles, ListChecks, Copy, Check } from 'lucide-react'
import { cn } from '../lib/utils'
import { classifyError } from '../lib/errorClassifier'
import type { ProxySettings } from '../stores/dashboardStore'
import { PingEndpoint, FetchProviderModels } from '../../wailsjs/go/main/App'
import { ProviderPicker } from './ProviderPicker'

interface ProxyConfigPanelProps {
  settings: ProxySettings
  saving: boolean
  configuring: boolean
  onSave: (settings: ProxySettings) => void
  onConfigureAll: () => void
}

type PingState = 'idle' | 'pinging' | 'ok' | 'error'

export function ProxyConfigPanel({ settings, saving, configuring, onSave, onConfigureAll }: ProxyConfigPanelProps) {
  const { t } = useTranslation()
  const [endpoint, setEndpoint] = useState(settings.apiEndpoint)
  const [apiKey, setApiKey] = useState(settings.apiKey)
  const [registrationUrl, setRegistrationUrl] = useState(settings.registrationUrl || '')
  const [keyPlaceholder, setKeyPlaceholder] = useState('sk-...')
  const [providerName, setProviderName] = useState('')

  const [pingState, setPingState] = useState<PingState>('idle')
  const [pingMs, setPingMs] = useState(0)
  const [pingError, setPingError] = useState('')

  const [modelsState, setModelsState] = useState<'idle' | 'fetching' | 'ok' | 'error'>('idle')
  const [models, setModels] = useState<string[]>([])
  const [modelsError, setModelsError] = useState('')
  const [copiedModel, setCopiedModel] = useState<string | null>(null)

  const [showPicker, setShowPicker] = useState(false)

  const hasChanges = endpoint !== settings.apiEndpoint || apiKey !== settings.apiKey
  const hasValues = endpoint.trim() !== '' && apiKey.trim() !== ''

  const handlePing = async (url: string) => {
    if (!url.trim()) return
    setPingState('pinging')
    setPingError('')
    try {
      const ms = await PingEndpoint(url.trim())
      if (ms < 0) {
        setPingState('error')
        setPingError(t('proxyPanel.connectFailed'))
      } else {
        setPingMs(ms)
        setPingState('ok')
      }
    } catch (err) {
      setPingState('error')
      setPingError(classifyError(err).message)
    }
  }

  const handleSave = () => {
    onSave({
      apiEndpoint: endpoint.trim(),
      apiKey: apiKey.trim(),
      registrationUrl,
    })
    if (endpoint.trim()) {
      handlePing(endpoint.trim())
    } else {
      setPingState('idle')
    }
  }

  const handleProviderSelect = (preset: { baseUrl: string; keyFormat: string; docsUrl: string; name: string }) => {
    setEndpoint(preset.baseUrl)
    setKeyPlaceholder(preset.keyFormat || 'sk-...')
    setRegistrationUrl(preset.docsUrl || '')
    setProviderName(preset.name)
    setShowPicker(false)
    setPingState('idle')
    setModelsState('idle')
    setModels([])
  }

  const handleFetchModels = async () => {
    if (!endpoint.trim()) return
    setModelsState('fetching')
    setModelsError('')
    try {
      const ids = await FetchProviderModels(endpoint.trim(), apiKey.trim())
      setModels(ids || [])
      setModelsState('ok')
    } catch (err) {
      setModelsError(classifyError(err).message)
      setModelsState('error')
    }
  }

  const handleCopyModel = async (id: string) => {
    try {
      await navigator.clipboard.writeText(id)
      setCopiedModel(id)
      setTimeout(() => setCopiedModel(null), 1500)
    } catch {
      // Clipboard may be unavailable; non-fatal
    }
  }

  return (
    <div className="border border-border rounded-lg p-4 bg-card">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-medium flex items-center gap-2">
          <Settings2 className="h-4 w-4 text-purple-500" />
          {providerName ? `${providerName} — API Configuration` : 'API Provider Configuration'}
        </h3>
        <button
          onClick={() => setShowPicker(true)}
          className={cn(
            'flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-colors',
            'bg-primary/10 text-primary hover:bg-primary/20'
          )}
        >
          <Sparkles className="h-3.5 w-3.5" />
          {t('proxyPanel.chooseProvider', 'Choose Provider')}
        </button>
      </div>

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
            placeholder={keyPlaceholder}
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
            {providerName ? `${providerName} docs` : 'Register for an API key'}
          </a>
        )}

        {/* Actions */}
        <div className="flex gap-2 pt-1 flex-wrap">
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
            onClick={() => handlePing(endpoint)}
            disabled={pingState === 'pinging' || !endpoint.trim()}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
              'border border-border hover:bg-muted',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            {pingState === 'pinging' ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Wifi className="h-3.5 w-3.5" />
            )}
            {t('proxyPanel.testConnection')}
          </button>

          <button
            onClick={handleFetchModels}
            disabled={modelsState === 'fetching' || !endpoint.trim()}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
              'border border-border hover:bg-muted',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
            title={t('proxyPanel.fetchModelsHint', 'Query /v1/models to discover available model IDs')}
          >
            {modelsState === 'fetching' ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <ListChecks className="h-3.5 w-3.5" />
            )}
            {t('proxyPanel.fetchModels', 'Fetch Models')}
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

        {/* Connectivity status badge */}
        {pingState !== 'idle' && (
          <div className={cn(
            'flex items-center gap-1.5 text-xs mt-1',
            pingState === 'pinging' && 'text-muted-foreground',
            pingState === 'ok' && 'text-green-500',
            pingState === 'error' && 'text-red-500',
          )}>
            {pingState === 'pinging' && <Loader2 className="h-3 w-3 animate-spin" />}
            {pingState === 'ok' && <CheckCircle2 className="h-3 w-3" />}
            {pingState === 'error' && <WifiOff className="h-3 w-3" />}
            {pingState === 'pinging' && t('proxyPanel.connecting')}
            {pingState === 'ok' && t('proxyPanel.connectedMs', { ms: pingMs })}
            {pingState === 'error' && pingError}
          </div>
        )}

        {/* Discovered models */}
        {modelsState === 'error' && (
          <div className="flex items-start gap-1.5 text-xs text-red-500 mt-1">
            <WifiOff className="h-3 w-3 mt-0.5 shrink-0" />
            <span className="break-all">{modelsError}</span>
          </div>
        )}
        {modelsState === 'ok' && models.length > 0 && (
          <div className="mt-1 p-2.5 rounded-md border border-border bg-muted/30">
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1.5">
              <ListChecks className="h-3 w-3" />
              <span>
                {t('proxyPanel.modelsFound', { count: models.length, defaultValue: '{{count}} models discovered — click to copy' })}
              </span>
            </div>
            <div className="flex flex-wrap gap-1">
              {models.map((id) => {
                const isCopied = copiedModel === id
                return (
                  <button
                    key={id}
                    onClick={() => handleCopyModel(id)}
                    className={cn(
                      'inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono transition-colors',
                      isCopied
                        ? 'bg-green-500/20 text-green-600'
                        : 'bg-background border border-border hover:border-primary/50 hover:bg-muted'
                    )}
                    title={t('proxyPanel.copyModel', 'Copy model ID')}
                  >
                    {isCopied ? <Check className="h-2.5 w-2.5" /> : <Copy className="h-2.5 w-2.5 opacity-40" />}
                    <span className="break-all">{id}</span>
                  </button>
                )
              })}
            </div>
          </div>
        )}
      </div>

      {/* Provider picker modal */}
      {showPicker && (
        <ProviderPicker
          onSelect={handleProviderSelect}
          onClose={() => setShowPicker(false)}
        />
      )}
    </div>
  )
}
