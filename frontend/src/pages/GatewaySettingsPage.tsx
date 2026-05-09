import { useEffect, useMemo, useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Settings2, RefreshCw, AlertCircle, Trash, Search,
} from 'lucide-react'
import { useGatewayStore } from '../stores/gatewayStore'
import { createGatewayClient } from '../lib/gateway-api'
import { OptionEditor } from '../components/gateway/OptionEditor'
import { OptionsSectionForm } from '../components/gateway/OptionsSectionForm'
import {
  type SettingsTab, OPTION_META, metaForTab, buildKeyToTabIndex,
} from '../lib/gatewayOptionMeta'

// Tab order matches newapi's web admin (12 tabs total). Each tab can be
// rendered either through the metadata-driven OptionsSectionForm (rich
// labels + descriptions + grouping) or, for tabs without metadata yet,
// the legacy OptionEditor flat-list.
const TAB_KEYS: SettingsTab[] = [
  'operations', 'dashboard', 'chat', 'drawing', 'payment', 'pricing',
  'rateLimit', 'modelConfig', 'modelDeploy', 'performance', 'system', 'other',
]

const TAB_LABEL: Record<SettingsTab, { key: string; fallback: string }> = {
  operations:  { key: 'gateway.settings.tab.operations',  fallback: '运营设置' },
  dashboard:   { key: 'gateway.settings.tab.dashboard',   fallback: '仪表盘' },
  chat:        { key: 'gateway.settings.tab.chat',        fallback: '聊天设置' },
  drawing:     { key: 'gateway.settings.tab.drawing',     fallback: '绘图设置' },
  payment:     { key: 'gateway.settings.tab.payment',     fallback: '支付设置' },
  pricing:     { key: 'gateway.settings.tab.pricing',     fallback: '分组与模型定价' },
  rateLimit:   { key: 'gateway.settings.tab.rateLimit',   fallback: '速率限制' },
  modelConfig: { key: 'gateway.settings.tab.modelConfig', fallback: '模型相关' },
  modelDeploy: { key: 'gateway.settings.tab.modelDeploy', fallback: '模型部署' },
  performance: { key: 'gateway.settings.tab.performance', fallback: '性能设置' },
  system:      { key: 'gateway.settings.tab.system',      fallback: '系统设置' },
  other:       { key: 'gateway.settings.tab.other',       fallback: '其他设置' },
}

// One-line description rendered under the active tab title — tells users
// at a glance what kind of options live here.
const TAB_DESC: Record<SettingsTab, { key: string; fallback: string }> = {
  operations:  { key: 'gateway.settings.desc.operations',  fallback: '充值入口、配额、品牌、注册策略 — 影响所有用户的全局策略。' },
  dashboard:   { key: 'gateway.settings.desc.dashboard',   fallback: '数据导出与统计开关。' },
  chat:        { key: 'gateway.settings.desc.chat',        fallback: '聊天页与流式缓存相关。' },
  drawing:     { key: 'gateway.settings.desc.drawing',     fallback: '绘图、Midjourney、图片生成相关。' },
  payment:     { key: 'gateway.settings.desc.payment',     fallback: 'Epay / Stripe / Creem / Waffo 等支付通道密钥与回调。' },
  pricing:     { key: 'gateway.settings.desc.pricing',     fallback: '模型倍率、分组折扣、按次定价 — 用可视化编辑或粘贴 JSON。' },
  rateLimit:   { key: 'gateway.settings.desc.rateLimit',   fallback: '全局 API / Web 速率，按模型 / 分组的限流。' },
  modelConfig: { key: 'gateway.settings.desc.modelConfig', fallback: '模型行为开关：原始请求、安全设置、敏感词检查。' },
  modelDeploy: { key: 'gateway.settings.desc.modelDeploy', fallback: '渠道自动启停、可用分组、状态码策略。' },
  performance: { key: 'gateway.settings.desc.performance', fallback: '重试、缓存、批量更新、统计汇总。' },
  system:      { key: 'gateway.settings.desc.system',      fallback: 'SMTP、第三方 OAuth、人机验证、文件权限。' },
  other:       { key: 'gateway.settings.desc.other',       fallback: '尚未归类到上述 tab 的所有 newapi 选项。' },
}

const KEY_TO_TAB = buildKeyToTabIndex()

/**
 * Decide which tab an option key belongs to. Metadata wins; otherwise we
 * use a fallback prefix-match heuristic so the unknown 70% of newapi
 * options still land in the right place instead of all dumping into
 * "other".
 */
function inferTab(key: string): SettingsTab {
  if (KEY_TO_TAB[key]) return KEY_TO_TAB[key]
  // Heuristic prefix routing for keys that don't have explicit metadata yet.
  if (key.startsWith('Mj') || key.startsWith('Drawing') || key.startsWith('Worker') || key.startsWith('Task')) return 'drawing'
  if (key.startsWith('Stripe') || key.startsWith('Creem') || key.startsWith('Epay') || key.startsWith('Waffo') || key.startsWith('Pay') || key.includes('TopUp') || key === 'MinTopUp' || key === 'USDExchangeRate' || key === 'QuotaPerUnit' || key === 'CustomCallbackAddress') return 'payment'
  if (key.includes('Ratio') || key.includes('Price') || key === 'AutoGroups' || key === 'UserUsableGroups') return key === 'UserUsableGroups' ? 'modelDeploy' : 'pricing'
  if (key.includes('RateLimit')) return 'rateLimit'
  if (key.startsWith('SMTP') || key.includes('Email') || key.includes('OAuth') || key.startsWith('GitHub') || key.startsWith('Telegram') || key.startsWith('WeChat') || key.startsWith('LinuxDO') || key.startsWith('Turnstile') || key === 'ServerAddress' || key.includes('Permission')) return 'system'
  if (key.startsWith('DataExport')) return 'dashboard'
  if (key.startsWith('CheckSensitive') || key === 'StopOnSensitiveEnabled' || key === 'SensitiveWords' || key.includes('OriginalRequest') || key.includes('Gemini')) return 'modelConfig'
  if (key.startsWith('Automatic') || key === 'ChannelDisableThreshold' || key === 'CacheEnabled' || key === 'BatchUpdateEnabled' || key === 'StatisticsEnabled' || key === 'SyncFrequency' || key === 'LogConsumeEnabled' || key === 'StreamCacheQueueLength') return 'performance'
  if (key === 'Chats' || key === 'ChatLink2') return 'chat'
  return 'other'
}

export function GatewaySettingsPage() {
  const { t } = useTranslation()
  const { status: serverStatus, adminToken } = useGatewayStore()

  const [tab, setTab] = useState<SettingsTab>('operations')
  const [options, setOptions] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [confirmAction, setConfirmAction] = useState<'reset' | 'clear' | null>(null)
  const [search, setSearch] = useState('')

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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [serverStatus?.running, adminToken])

  useEffect(() => { load() }, [load])

  // Group options by tab once per load — used for both the rich form and
  // the fallback OptionEditor on tabs without metadata.
  const optionsByTab = useMemo(() => {
    const out: Record<SettingsTab, Record<string, string>> = {
      operations: {}, dashboard: {}, chat: {}, drawing: {}, payment: {},
      pricing: {}, rateLimit: {}, modelConfig: {}, modelDeploy: {},
      performance: {}, system: {}, other: {},
    }
    for (const [k, v] of Object.entries(options)) {
      out[inferTab(k)][k] = v
    }
    return out
  }, [options])

  // Search index — flat list of (tab, key, value) for cross-tab keyword filter.
  const searchHits = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return null
    const hits: Array<{ tab: SettingsTab; key: string; value: string }> = []
    for (const [tabId, tabOpts] of Object.entries(optionsByTab) as Array<[SettingsTab, Record<string, string>]>) {
      for (const [k, v] of Object.entries(tabOpts)) {
        if (k.toLowerCase().includes(q) || v.toLowerCase().includes(q)) {
          hits.push({ tab: tabId, key: k, value: v })
        }
      }
    }
    return hits
  }, [search, optionsByTab])

  const handleSaveOption = useCallback(async (key: string, value: string) => {
    if (!client) throw new Error('No gateway connection')
    await client.updateOption(key, value)
    setOptions((prev) => ({ ...prev, [key]: value }))
  }, [client])

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

  if (!serverStatus?.running) {
    return (
      <div className="flex flex-col h-full items-center justify-center text-muted-foreground gap-2">
        <AlertCircle className="h-8 w-8" />
        <p>{t('gateway.status.stopped')}</p>
      </div>
    )
  }

  // Whether the active tab has metadata-driven rows.
  const hasMetadata = metaForTab(tab).length > 0
  const activeTabOpts = optionsByTab[tab]

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="px-6 py-4 border-b border-border flex items-center justify-between gap-4 flex-wrap">
        <div className="flex items-center gap-2 min-w-0">
          <Settings2 className="h-5 w-5 text-violet-400 shrink-0" />
          <div className="min-w-0">
            <h2 className="text-lg font-semibold truncate">{t('gateway.settings.title', '设置')}</h2>
            <p className="text-xs text-muted-foreground truncate">
              {t(TAB_DESC[tab].key, TAB_DESC[tab].fallback)}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('gateway.settings.searchAll', '搜索所有选项…')}
              className="pl-7 pr-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary w-56"
            />
          </div>
          <button
            onClick={load}
            disabled={loading}
            className="flex items-center gap-1 px-3 py-1.5 rounded-md border border-border hover:bg-muted text-sm disabled:opacity-50"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      {/* Error banner */}
      {error && (
        <div className="mx-6 mt-3 text-sm text-red-400 bg-red-900/20 rounded px-3 py-2">{error}</div>
      )}

      {/* Search results override the tab UI when there's a query */}
      {searchHits ? (
        <div className="flex-1 overflow-y-auto p-6">
          <div className="text-xs text-muted-foreground mb-3">
            {t('gateway.settings.searchResults', { count: searchHits.length })}
          </div>
          <SearchResults hits={searchHits} onSave={handleSaveOption} tabLabels={TAB_LABEL} />
        </div>
      ) : (
        <>
          {/* Tab Bar */}
          <div className="overflow-x-auto border-b border-border">
            <div className="flex gap-1 min-w-max px-6 pt-2">
              {TAB_KEYS.map((key) => {
                const count = Object.keys(optionsByTab[key]).length
                const isActive = tab === key
                return (
                  <button
                    key={key}
                    onClick={() => setTab(key)}
                    className={`flex items-center gap-1.5 px-3 py-2 text-sm whitespace-nowrap rounded-t-md border-b-2 transition-colors ${
                      isActive
                        ? 'border-primary text-foreground bg-background'
                        : 'border-transparent text-muted-foreground hover:text-foreground hover:bg-muted/30'
                    }`}
                  >
                    {t(TAB_LABEL[key].key, TAB_LABEL[key].fallback)}
                    {count > 0 && (
                      <span className="text-[10px] tabular-nums opacity-60">({count})</span>
                    )}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Tab Content */}
          <div className="flex-1 overflow-y-auto p-6">
            {loading ? (
              <p className="text-sm text-muted-foreground py-4">{t('status.loading')}</p>
            ) : Object.keys(activeTabOpts).length === 0 ? (
              <p className="text-sm text-muted-foreground py-4">{t('gateway.settings.tabEmpty', '此分组无选项。')}</p>
            ) : hasMetadata ? (
              <OptionsSectionForm
                tab={tab}
                options={activeTabOpts}
                onSave={handleSaveOption}
              />
            ) : (
              <div className="rounded-lg border border-border bg-card p-4">
                <OptionEditor options={activeTabOpts} onSave={handleSaveOption} />
              </div>
            )}

            {/* Page-level Reset / Clear actions — only show on Pricing + Performance respectively */}
            {(tab === 'pricing' || tab === 'performance') && (
              <div className="flex gap-3 pt-6 mt-6 border-t border-border">
                {tab === 'pricing' && (
                  <button
                    onClick={() => setConfirmAction('reset')}
                    className="flex items-center gap-2 px-4 py-2 rounded-md border border-amber-500/30 hover:bg-amber-500/10 text-sm text-amber-500"
                  >
                    <RefreshCw className="h-4 w-4" />
                    {t('gateway.settings.resetModelRatio', '重置模型倍率')}
                  </button>
                )}
                {tab === 'performance' && (
                  <button
                    onClick={() => setConfirmAction('clear')}
                    className="flex items-center gap-2 px-4 py-2 rounded-md border border-red-500/30 hover:bg-red-500/10 text-sm text-red-500"
                  >
                    <Trash className="h-4 w-4" />
                    {t('gateway.settings.clearCache', '清空缓存')}
                  </button>
                )}
              </div>
            )}
          </div>
        </>
      )}

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
                    'This will clear the channel-affinity cache on the gateway server.',
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

function SearchResults({
  hits,
  onSave,
  tabLabels,
}: {
  hits: Array<{ tab: SettingsTab; key: string; value: string }>
  onSave: (k: string, v: string) => Promise<void>
  tabLabels: Record<SettingsTab, { key: string; fallback: string }>
}) {
  const { t } = useTranslation()
  if (hits.length === 0) {
    return <p className="text-sm text-muted-foreground">{t('gateway.settings.noMatch', '没有匹配的选项。')}</p>
  }
  // Group hits by tab for readability.
  const byTab: Record<string, Array<{ key: string; value: string }>> = {}
  for (const h of hits) {
    if (!byTab[h.tab]) byTab[h.tab] = []
    byTab[h.tab].push({ key: h.key, value: h.value })
  }
  return (
    <div className="space-y-4">
      {Object.entries(byTab).map(([tab, rows]) => (
        <section key={tab} className="rounded-lg border border-border bg-card p-4">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-2">
            {t(tabLabels[tab as SettingsTab].key, tabLabels[tab as SettingsTab].fallback)}
          </h3>
          <OptionEditor options={Object.fromEntries(rows.map((r) => [r.key, r.value]))} onSave={onSave} />
        </section>
      ))}
    </div>
  )
}

// Re-export so that someone who happens to read this file knows the
// metadata source of truth — the registry lives in one place.
export { OPTION_META }
