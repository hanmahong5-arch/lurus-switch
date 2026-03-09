import { useEffect, useState, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Settings2, RefreshCw, AlertCircle, Trash } from 'lucide-react'
import { useGatewayStore } from '../stores/gatewayStore'
import { createGatewayClient } from '../lib/gateway-api'
import { OptionEditor } from '../components/gateway/OptionEditor'

// --- Tab Definitions ---

const TAB_KEYS = [
  'operations',
  'pricing',
  'rateLimit',
  'modelConfig',
  'performance',
  'system',
  'other',
] as const

type TabKey = (typeof TAB_KEYS)[number]

const TAB_KEY_PATTERNS: Record<Exclude<TabKey, 'other'>, string[]> = {
  operations: [
    'TopUpLink',
    'ChatLink',
    'QuotaForNewUser',
    'QuotaForInviter',
    'QuotaForInvitee',
    'QuotaRemindThreshold',
    'PreConsumedQuota',
    'DisplayInCurrencyEnabled',
    'DisplayTokenStatEnabled',
    'ApproximateTokenEnabled',
  ],
  pricing: [
    'ModelRatio',
    'CompletionRatio',
    'GroupRatio',
    'ModelPrice',
  ],
  rateLimit: [
    'GlobalApiRateLimitNum',
    'GlobalApiRateLimitDuration',
    'GlobalWebRateLimitNum',
    'GlobalWebRateLimitDuration',
  ],
  modelConfig: [
    'ClaudeOriginalRequest',
    'GeminiSafetySetting',
    'GeminiVersion',
    'GrokOriginalRequest',
  ],
  performance: [
    'ChannelDisableThreshold',
    'AutomaticDisableChannelEnabled',
    'AutomaticEnableChannelEnabled',
    'CacheEnabled',
    'BatchUpdateEnabled',
    'StatisticsEnabled',
    'RetryTimes',
    'SyncFrequency',
  ],
  system: [
    'ServerAddress',
    'EmailDomainRestrictionEnabled',
    'EmailDomainWhitelist',
    'SMTPServer',
    'SMTPPort',
    'SMTPAccount',
    'SMTPToken',
    'Footer',
  ],
}

/** Collect all known patterns into a flat set for "other" tab exclusion. */
const ALL_KNOWN_PATTERNS: string[] = Object.values(TAB_KEY_PATTERNS).flat()

function matchesTab(key: string, patterns: string[]): boolean {
  return patterns.some((p) => key.includes(p))
}

function filterOptions(
  options: Record<string, string>,
  tab: TabKey,
): Record<string, string> {
  const entries = Object.entries(options)

  if (tab === 'other') {
    return Object.fromEntries(
      entries.filter(([k]) => !matchesTab(k, ALL_KNOWN_PATTERNS)),
    )
  }

  const patterns = TAB_KEY_PATTERNS[tab]
  return Object.fromEntries(entries.filter(([k]) => matchesTab(k, patterns)))
}

// --- Tab Label Map ---

const TAB_LABEL_MAP: Record<TabKey, string> = {
  operations: 'gateway.settings.operations',
  pricing: 'gateway.settings.pricing',
  rateLimit: 'gateway.settings.rateLimit',
  modelConfig: 'gateway.settings.modelConfig',
  performance: 'gateway.settings.performance',
  system: 'gateway.settings.system',
  other: 'gateway.settings.other',
}

const TAB_LABEL_FALLBACK: Record<TabKey, string> = {
  operations: 'Operations',
  pricing: 'Pricing',
  rateLimit: 'Rate Limit',
  modelConfig: 'Model Config',
  performance: 'Performance',
  system: 'System',
  other: 'Other',
}

// --- Component ---

export function GatewaySettingsPage() {
  const { t } = useTranslation()
  const { status: serverStatus, adminToken } = useGatewayStore()

  const [tab, setTab] = useState<TabKey>('operations')
  const [options, setOptions] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [confirmAction, setConfirmAction] = useState<'reset' | 'clear' | null>(null)

  const client = serverStatus?.running && adminToken
    ? createGatewayClient(serverStatus.url, adminToken)
    : null

  const load = useCallback(async () => {
    if (!client) return
    setLoading(true)
    setError(null)
    try {
      const res = await client.getOptions()
      setOptions(res.data ?? {})
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }, [serverStatus?.running, adminToken])

  useEffect(() => {
    load()
  }, [load])

  // Reload when tab changes to keep data fresh.
  useEffect(() => {
    load()
  }, [tab])

  const filteredOptions = useMemo(
    () => filterOptions(options, tab),
    [options, tab],
  )

  const handleSaveOption = async (key: string, value: string) => {
    if (!client) return
    await client.updateOption(key, value)
    // Update local state optimistically.
    setOptions((prev) => ({ ...prev, [key]: value }))
  }

  const handleResetModelRatio = async () => {
    if (!client) return
    setConfirmAction(null)
    setError(null)
    try {
      await client.resetModelRatio()
      await load()
    } catch (e) {
      setError(String(e))
    }
  }

  const handleClearCache = async () => {
    if (!client) return
    setConfirmAction(null)
    setError(null)
    try {
      await client.clearCache()
    } catch (e) {
      setError(String(e))
    }
  }

  // --- Stopped State ---

  if (!serverStatus?.running) {
    return (
      <div className="flex flex-col h-full items-center justify-center text-muted-foreground gap-2">
        <AlertCircle className="h-8 w-8" />
        <p>{t('gateway.status.stopped')}</p>
      </div>
    )
  }

  // --- Main Render ---

  return (
    <div className="flex flex-col h-full overflow-y-auto p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold flex items-center gap-2">
          <Settings2 className="h-6 w-6 text-violet-400" />
          {t('gateway.settings')}
        </h2>
        <button
          onClick={load}
          disabled={loading}
          className="flex items-center gap-1 px-3 py-1.5 rounded-md border border-border hover:bg-muted text-sm"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
        </button>
      </div>

      {/* Error */}
      {error && (
        <div className="text-sm text-red-400 bg-red-900/20 rounded px-3 py-2">{error}</div>
      )}

      {/* Tab Bar */}
      <div className="overflow-x-auto">
        <div className="flex gap-1 border-b border-border min-w-max">
          {TAB_KEYS.map((key) => (
            <button
              key={key}
              onClick={() => setTab(key)}
              className={`px-4 py-2 text-sm whitespace-nowrap border-b-2 transition-colors ${
                tab === key
                  ? 'border-indigo-500 text-indigo-400 font-medium'
                  : 'border-transparent text-muted-foreground hover:text-foreground hover:border-muted-foreground/40'
              }`}
            >
              {t(TAB_LABEL_MAP[key], TAB_LABEL_FALLBACK[key])}
            </button>
          ))}
        </div>
      </div>

      {/* Tab Content */}
      <div className="rounded-lg border border-border bg-card p-4 min-h-[200px]">
        {loading ? (
          <p className="text-sm text-muted-foreground py-4">{t('status.loading')}</p>
        ) : (
          <OptionEditor options={filteredOptions} onSave={handleSaveOption} />
        )}
      </div>

      {/* Action Buttons */}
      <div className="flex gap-3 pt-2">
        <button
          onClick={() => setConfirmAction('reset')}
          className="flex items-center gap-2 px-4 py-2 rounded-md border border-border hover:bg-muted text-sm text-amber-400"
        >
          <RefreshCw className="h-4 w-4" />
          {t('gateway.settings.resetModelRatio', 'Reset Model Ratio')}
        </button>
        <button
          onClick={() => setConfirmAction('clear')}
          className="flex items-center gap-2 px-4 py-2 rounded-md border border-border hover:bg-muted text-sm text-red-400"
        >
          <Trash className="h-4 w-4" />
          {t('gateway.settings.clearCache', 'Clear Cache')}
        </button>
      </div>

      {/* Confirm Modal */}
      {confirmAction && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-card border border-border rounded-lg p-6 w-96 space-y-4">
            <h3 className="font-semibold">
              {confirmAction === 'reset'
                ? t('gateway.settings.confirmResetTitle', 'Reset Model Ratio?')
                : t('gateway.settings.confirmClearTitle', 'Clear Cache?')}
            </h3>
            <p className="text-sm text-muted-foreground">
              {confirmAction === 'reset'
                ? t(
                    'gateway.settings.confirmResetDesc',
                    'This will reset all model ratios to their default values. This action cannot be undone.',
                  )
                : t(
                    'gateway.settings.confirmClearDesc',
                    'This will clear all cached data on the gateway server.',
                  )}
            </p>
            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => setConfirmAction(null)}
                className="px-4 py-1.5 rounded border border-border text-sm hover:bg-muted"
              >
                {t('settings.data.cancel')}
              </button>
              <button
                onClick={confirmAction === 'reset' ? handleResetModelRatio : handleClearCache}
                className={`px-4 py-1.5 rounded text-white text-sm ${
                  confirmAction === 'reset'
                    ? 'bg-amber-600 hover:bg-amber-500'
                    : 'bg-red-600 hover:bg-red-500'
                }`}
              >
                {t('settings.confirm', 'Confirm')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
