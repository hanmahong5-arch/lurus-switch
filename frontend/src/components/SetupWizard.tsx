import { useState, useEffect, useRef } from 'react'
import { Loader2, CheckCircle2, XCircle, ArrowRight, ArrowLeft, SkipForward, Wifi, Zap, User } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import {
  DetectAllTools,
  InstallAllTools,
  DetectSystemProxy,
  SaveProxySettings,
  GetAppSettings,
  SaveAppSettings,
  BillingValidateToken,
  ConfigureAllToolsRelay,
} from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { proxy, appconfig } from '../../wailsjs/go/models'
import type { ToolStatus } from '../stores/dashboardStore'

// Default Lurus relay endpoint — the newapi gateway
const LURUS_RELAY_ENDPOINT = 'https://newapi.lurus.cn'

const TOOL_ORDER = ['claude', 'codex', 'gemini', 'picoclaw', 'nullclaw', 'zeroclaw', 'openclaw'] as const

const toolLabels: Record<string, string> = {
  claude: 'Claude Code',
  codex: 'Codex',
  gemini: 'Gemini CLI',
  picoclaw: 'PicoClaw',
  nullclaw: 'NullClaw',
  zeroclaw: 'ZeroClaw',
  openclaw: 'OpenClaw',
}

interface DetectedProxy {
  source: string
  host: string
  port: number
  type: string
  url: string
}

interface AccountInfo {
  displayName: string
  balance: number
  planCode?: string
}

interface SetupWizardProps {
  onComplete: () => void
}

export function SetupWizard({ onComplete }: SetupWizardProps) {
  const { t, i18n } = useTranslation()
  const [step, setStep] = useState(0)
  const totalSteps = 5

  // Step 0 — language
  const [language, setLanguage] = useState(i18n.language || 'zh')

  // Step 1 — Lurus account
  const [lurusToken, setLurusToken] = useState('')
  const [validatingToken, setValidatingToken] = useState(false)
  const [accountInfo, setAccountInfo] = useState<AccountInfo | null>(null)
  const [accountError, setAccountError] = useState('')

  // Step 2 — tools
  const [tools, setTools] = useState<Record<string, ToolStatus>>({})
  const [detectingTools, setDetectingTools] = useState(false)
  const [detectError, setDetectError] = useState('')
  const [installingAll, setInstallingAll] = useState(false)
  // Per-tool install progress: 0-99 = downloading, 100 = done, -1 = failed
  const [installProgress, setInstallProgress] = useState<Record<string, number>>({})

  // Step 3 — proxy
  const [proxies, setProxies] = useState<DetectedProxy[]>([])
  const [detectingProxy, setDetectingProxy] = useState(false)
  const [selectedProxy, setSelectedProxy] = useState<string>('')
  const [manualEndpoint, setManualEndpoint] = useState('')
  const [manualKey, setManualKey] = useState('')
  const [proxySaved, setProxySaved] = useState(false)

  // Step 4 — done: relay config
  const [configuringRelay, setConfiguringRelay] = useState(false)
  const [relayConfigured, setRelayConfigured] = useState(false)

  // Prevent re-triggering on back/forward navigation
  const toolsDetectedRef = useRef(false)
  const proxyDetectedRef = useRef(false)

  // Subscribe to per-tool install progress events from the Go backend.
  useEffect(() => {
    const offProgress = EventsOn('tool:install:progress', (d: { tool: string; percent: number }) => {
      setInstallProgress(p => ({ ...p, [d.tool]: d.percent }))
    })
    const offDone = EventsOn('tool:install:done', (d: { tool: string; success: boolean }) => {
      setInstallProgress(p => ({ ...p, [d.tool]: d.success ? 100 : -1 }))
    })
    return () => { offProgress(); offDone() }
  }, [])

  // Detect tools on step 2
  useEffect(() => {
    if (step === 2 && !toolsDetectedRef.current) {
      toolsDetectedRef.current = true
      setDetectingTools(true)
      setDetectError('')
      DetectAllTools()
        .then((r) => setTools(r))
        .catch((err) => setDetectError(`${err}`))
        .finally(() => setDetectingTools(false))
    }
  }, [step])

  // Detect proxy on step 3
  useEffect(() => {
    if (step === 3 && !proxyDetectedRef.current) {
      proxyDetectedRef.current = true

      // Auto-prefill endpoint with Lurus relay if account was connected
      if (accountInfo && !manualEndpoint) {
        setManualEndpoint(LURUS_RELAY_ENDPOINT)
        setManualKey(lurusToken)
      }

      setDetectingProxy(true)
      DetectSystemProxy()
        .then((r) => {
          setProxies(r || [])
          if (r && r.length > 0 && !accountInfo) setSelectedProxy(r[0].url)
        })
        .catch(() => {})
        .finally(() => setDetectingProxy(false))
    }
  }, [step, accountInfo, lurusToken, manualEndpoint])

  const handleLanguageChange = (lang: string) => {
    setLanguage(lang)
    i18n.changeLanguage(lang)
  }

  // Step 1 — validate Lurus token
  const handleValidateToken = async () => {
    if (!lurusToken.trim()) return
    setValidatingToken(true)
    setAccountError('')
    setAccountInfo(null)
    try {
      const overview = await BillingValidateToken(LURUS_RELAY_ENDPOINT, lurusToken.trim())
      if (overview) {
        setAccountInfo({
          displayName: overview.account?.displayName || overview.account?.lurusId || 'Lurus User',
          balance: overview.wallet?.balance ?? 0,
          planCode: overview.subscription?.planCode,
        })
        // Auto-save Lurus relay as the proxy endpoint so Step 3 can be skipped.
        await SaveProxySettings(proxy.ProxySettings.createFrom({
          apiEndpoint: LURUS_RELAY_ENDPOINT,
          apiKey: '',
          userToken: lurusToken.trim(),
        })).catch(() => {/* non-critical */})
        setProxySaved(true)
      }
    } catch (err) {
      setAccountError(`${err}`)
    } finally {
      setValidatingToken(false)
    }
  }

  const handleInstallAll = async () => {
    setInstallingAll(true)
    setInstallProgress({})
    try {
      await InstallAllTools()
      const statuses = await DetectAllTools()
      setTools(statuses)

      // For Lurus members: auto-configure relay and skip manual proxy step.
      if (accountInfo) {
        await ConfigureAllToolsRelay().catch(() => {/* best-effort */})
        await handleFinish()
      }
    } catch {
      // Ignore errors in wizard — user can retry from Dashboard
    } finally {
      setInstallingAll(false)
    }
  }

  const handleUseLurusPlatform = () => {
    setManualEndpoint(LURUS_RELAY_ENDPOINT)
    if (lurusToken) setManualKey(lurusToken)
    setSelectedProxy('')
  }

  const handleSaveProxy = async () => {
    const endpoint = selectedProxy || manualEndpoint.trim()
    const key = manualKey.trim() || lurusToken.trim()
    if (!endpoint) return
    try {
      await SaveProxySettings(proxy.ProxySettings.createFrom({
        apiEndpoint: endpoint,
        apiKey: key,
        userToken: lurusToken.trim(),
      }))
      setProxySaved(true)
    } catch {
      // Ignore
    }
  }

  const handleConfigureAllRelay = async () => {
    setConfiguringRelay(true)
    try {
      await ConfigureAllToolsRelay()
      setRelayConfigured(true)
    } catch {
      // Non-critical
    } finally {
      setConfiguringRelay(false)
    }
  }

  const handleFinish = async () => {
    try {
      const settings = await GetAppSettings()
      await SaveAppSettings(appconfig.AppSettings.createFrom({
        ...settings,
        language,
        onboardingCompleted: true,
      }))
    } catch {
      // Ignore
    }
    onComplete()
  }

  const handleSkip = () => {
    setStep(totalSteps - 1)
  }

  const installedCount = Object.values(tools).filter((t) => t.installed).length
  const hasProxy = proxySaved || selectedProxy !== '' || manualEndpoint.trim() !== ''
  const canConfigureRelay = proxySaved && (installedCount > 0) && !relayConfigured

  return (
    <div className="h-screen flex flex-col items-center justify-center bg-background text-foreground p-6">
      <div className="w-full max-w-lg space-y-6">
        {/* Progress */}
        <div className="text-center space-y-2">
          <h1 className="text-xl font-semibold">{t('wizard.title')}</h1>
          <p className="text-sm text-muted-foreground">{t('wizard.subtitle')}</p>
          <div className="flex justify-center gap-2 pt-2">
            {Array.from({ length: totalSteps }).map((_, i) => (
              <div
                key={i}
                className={cn(
                  'h-1.5 w-8 rounded-full transition-colors',
                  i <= step ? 'bg-primary' : 'bg-muted'
                )}
              />
            ))}
          </div>
          <p className="text-xs text-muted-foreground">
            {t('wizard.step', { current: step + 1, total: totalSteps })}
          </p>
        </div>

        {/* Step content */}
        <div className="border border-border rounded-lg p-6 bg-card min-h-[300px] flex flex-col">
          {/* Step 0 — Welcome + language */}
          {step === 0 && (
            <div className="flex-1 flex flex-col gap-4">
              <h2 className="text-lg font-medium">{t('wizard.welcome.title')}</h2>
              <p className="text-sm text-muted-foreground">{t('wizard.welcome.desc')}</p>
              <div className="mt-4">
                <label className="block text-xs text-muted-foreground mb-1">{t('settings.appearance.language')}</label>
                <select
                  value={language}
                  onChange={(e) => handleLanguageChange(e.target.value)}
                  className="px-3 py-1.5 text-sm bg-muted border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
                >
                  <option value="zh">中文</option>
                  <option value="en">English</option>
                </select>
              </div>
            </div>
          )}

          {/* Step 1 — Lurus account connection */}
          {step === 1 && (
            <div className="flex-1 flex flex-col gap-4">
              <h2 className="text-lg font-medium">{t('wizard.account.title')}</h2>
              <p className="text-sm text-muted-foreground">{t('wizard.account.desc')}</p>

              {accountInfo ? (
                <div className="flex items-start gap-3 p-3 rounded-md bg-green-500/10 border border-green-500/20 mt-2">
                  <CheckCircle2 className="h-5 w-5 text-green-500 shrink-0 mt-0.5" />
                  <div className="text-sm space-y-1">
                    <p className="font-medium text-green-600">{t('wizard.account.connected')}</p>
                    <p className="text-muted-foreground">{accountInfo.displayName}</p>
                    <p className="text-muted-foreground">
                      {t('wizard.account.balance', { balance: accountInfo.balance.toFixed(2) })}
                      {accountInfo.planCode && ` · ${accountInfo.planCode}`}
                    </p>
                  </div>
                </div>
              ) : (
                <div className="space-y-3 mt-2">
                  <div>
                    <label className="block text-xs text-muted-foreground mb-0.5">{t('wizard.account.tokenLabel')}</label>
                    <div className="flex gap-2">
                      <input
                        type="password"
                        value={lurusToken}
                        onChange={(e) => { setLurusToken(e.target.value); setAccountError('') }}
                        placeholder="sk-..."
                        className="flex-1 px-3 py-1.5 text-sm rounded-md border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary"
                        onKeyDown={(e) => e.key === 'Enter' && handleValidateToken()}
                      />
                      <button
                        onClick={handleValidateToken}
                        disabled={validatingToken || !lurusToken.trim()}
                        className={cn(
                          'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-colors',
                          'bg-primary text-primary-foreground hover:bg-primary/90',
                          'disabled:opacity-50 disabled:cursor-not-allowed'
                        )}
                      >
                        {validatingToken ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <User className="h-3.5 w-3.5" />}
                        {t('wizard.account.verify')}
                      </button>
                    </div>
                  </div>
                  {accountError && (
                    <p className="text-xs text-red-500 flex items-center gap-1">
                      <XCircle className="h-3.5 w-3.5" />
                      {t('wizard.account.verifyFailed')}
                    </p>
                  )}
                  <p className="text-xs text-muted-foreground">{t('wizard.account.skipHint')}</p>
                </div>
              )}
            </div>
          )}

          {/* Step 2 — Tools */}
          {step === 2 && (
            <div className="flex-1 flex flex-col gap-3">
              <h2 className="text-lg font-medium">{t('wizard.tools.title')}</h2>
              <p className="text-sm text-muted-foreground">{t('wizard.tools.desc')}</p>
              {detectError && (
                <div className="flex items-center gap-2 px-3 py-2 rounded-md bg-red-500/10 border border-red-500/20 text-red-500 text-xs mt-1">
                  <XCircle className="h-3.5 w-3.5 shrink-0" />
                  <span>{detectError}</span>
                  <button
                    onClick={() => {
                      toolsDetectedRef.current = false
                      setDetectError('')
                      setStep(2)
                    }}
                    className="ml-auto underline hover:no-underline"
                  >
                    {t('wizard.tools.retry')}
                  </button>
                </div>
              )}
              {detectingTools ? (
                <div className="flex items-center gap-2 py-4">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span className="text-sm">{t('wizard.tools.detecting')}</span>
                </div>
              ) : (
                <div className="space-y-2 mt-2">
                  {TOOL_ORDER.map((name) => {
                    const tool = tools[name]
                    const installed = tool?.installed
                    const pct = installProgress[name]
                    const isDownloading = installingAll && pct !== undefined && pct >= 0 && pct < 100
                    return (
                      <div key={name} className="flex flex-col gap-0.5 py-1.5 px-3 rounded-md bg-muted/50">
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-medium">{toolLabels[name]}</span>
                          <span className={cn('flex items-center gap-1 text-xs',
                            isDownloading ? 'text-primary' :
                            installed ? 'text-green-500' : 'text-muted-foreground'
                          )}>
                            {isDownloading ? (
                              <>
                                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                                {pct}%
                              </>
                            ) : installed ? (
                              <>
                                <CheckCircle2 className="h-3.5 w-3.5" />
                                {t('wizard.tools.installed')} {tool?.version && `v${tool.version}`}
                              </>
                            ) : (
                              <>
                                <XCircle className="h-3.5 w-3.5" />
                                {t('wizard.tools.notInstalled')}
                              </>
                            )}
                          </span>
                        </div>
                        {isDownloading && (
                          <div className="w-full h-1 bg-muted rounded-full overflow-hidden">
                            <div
                              className="h-full bg-primary transition-all duration-300"
                              style={{ width: `${pct}%` }}
                            />
                          </div>
                        )}
                      </div>
                    )
                  })}
                  <button
                    onClick={handleInstallAll}
                    disabled={installingAll}
                    className={cn(
                      'mt-2 w-full flex items-center justify-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium transition-colors',
                      'bg-primary text-primary-foreground hover:bg-primary/90',
                      'disabled:opacity-50 disabled:cursor-not-allowed'
                    )}
                  >
                    {installingAll && <Loader2 className="h-4 w-4 animate-spin" />}
                    {t('wizard.tools.installAll')}
                  </button>
                </div>
              )}
            </div>
          )}

          {/* Step 3 — Proxy */}
          {step === 3 && (
            <div className="flex-1 flex flex-col gap-3">
              <h2 className="text-lg font-medium">{t('wizard.proxy.title')}</h2>
              <p className="text-sm text-muted-foreground">{t('wizard.proxy.desc')}</p>

              {/* Lurus Platform shortcut — highlight if account connected */}
              <div className={cn(
                'flex items-center justify-between p-3 rounded-md border',
                accountInfo
                  ? 'border-primary/40 bg-primary/5'
                  : 'border-border bg-muted/30'
              )}>
                <div className="text-sm">
                  <p className="font-medium">{t('wizard.proxy.lurusPlatform')}</p>
                  <p className="text-xs text-muted-foreground">{LURUS_RELAY_ENDPOINT}</p>
                </div>
                <button
                  onClick={handleUseLurusPlatform}
                  className={cn(
                    'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors shrink-0',
                    accountInfo
                      ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                      : 'border border-border hover:bg-muted'
                  )}
                >
                  <Zap className="h-3.5 w-3.5" />
                  {t('dashboard.useLurusPlatform')}
                </button>
              </div>

              {detectingProxy ? (
                <div className="flex items-center gap-2 py-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span className="text-sm">{t('wizard.proxy.detecting')}</span>
                </div>
              ) : (
                <div className="space-y-3">
                  {proxies.length > 0 && (
                    <>
                      <p className="text-xs text-muted-foreground">{t('wizard.proxy.detected')}</p>
                      <div className="space-y-1.5">
                        {proxies.map((p) => (
                          <button
                            key={p.url}
                            onClick={() => { setSelectedProxy(p.url); setManualEndpoint('') }}
                            className={cn(
                              'w-full text-left px-3 py-2 rounded-md border text-sm transition-colors',
                              selectedProxy === p.url
                                ? 'border-primary bg-primary/10'
                                : 'border-border hover:bg-muted'
                            )}
                          >
                            <div className="flex items-center gap-2">
                              <Wifi className="h-3.5 w-3.5 text-primary" />
                              <span className="font-medium capitalize">{p.source}</span>
                              <span className="text-muted-foreground">{p.url}</span>
                            </div>
                          </button>
                        ))}
                      </div>
                    </>
                  )}

                  {/* Manual config */}
                  <div className="border-t border-border pt-3 space-y-2">
                    <p className="text-xs font-medium">{t('wizard.proxy.manualConfig')}</p>
                    <div>
                      <label className="block text-xs text-muted-foreground mb-0.5">{t('wizard.proxy.apiEndpoint')}</label>
                      <input
                        type="url"
                        value={manualEndpoint}
                        onChange={(e) => { setManualEndpoint(e.target.value); setSelectedProxy('') }}
                        placeholder="https://api.example.com/v1"
                        className="w-full px-3 py-1.5 text-sm rounded-md border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary"
                      />
                    </div>
                    <div>
                      <label className="block text-xs text-muted-foreground mb-0.5">{t('wizard.proxy.apiKey')}</label>
                      <input
                        type="password"
                        value={manualKey}
                        onChange={(e) => setManualKey(e.target.value)}
                        placeholder="sk-..."
                        className="w-full px-3 py-1.5 text-sm rounded-md border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary"
                      />
                    </div>
                    {(selectedProxy || manualEndpoint.trim()) && (
                      <button
                        onClick={handleSaveProxy}
                        disabled={proxySaved}
                        className={cn(
                          'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
                          proxySaved
                            ? 'bg-green-500/10 text-green-500 border border-green-500/20'
                            : 'bg-primary text-primary-foreground hover:bg-primary/90'
                        )}
                      >
                        {proxySaved ? <CheckCircle2 className="h-3.5 w-3.5" /> : null}
                        {proxySaved ? t('wizard.proxy.saved') : t('settings.save')}
                      </button>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Step 4 — Done */}
          {step === 4 && (
            <div className="flex-1 flex flex-col gap-4 items-center justify-center text-center">
              <CheckCircle2 className="h-12 w-12 text-green-500" />
              <h2 className="text-lg font-medium">{t('wizard.done.title')}</h2>
              <p className="text-sm text-muted-foreground">{t('wizard.done.desc')}</p>
              <div className="text-sm space-y-1 mt-2">
                <p className="text-muted-foreground">{t('wizard.done.summary')}</p>
                {accountInfo && (
                  <p className="text-green-500 flex items-center justify-center gap-1">
                    <CheckCircle2 className="h-3.5 w-3.5" />
                    {t('wizard.account.connected')}: {accountInfo.displayName}
                  </p>
                )}
                <p>{t('wizard.done.toolsDetected', { count: installedCount })}</p>
                <p>{hasProxy ? t('wizard.done.proxyConfigured') : t('wizard.done.proxyNotConfigured')}</p>
              </div>

              {/* One-click relay config button */}
              {canConfigureRelay && (
                <button
                  onClick={handleConfigureAllRelay}
                  disabled={configuringRelay}
                  className={cn(
                    'mt-2 flex items-center gap-2 px-5 py-2.5 rounded-md text-sm font-medium transition-colors',
                    'bg-primary text-primary-foreground hover:bg-primary/90',
                    'disabled:opacity-50 disabled:cursor-not-allowed'
                  )}
                >
                  {configuringRelay ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Zap className="h-4 w-4" />
                  )}
                  {t('dashboard.configureLurusRelay')}
                </button>
              )}
              {relayConfigured && (
                <p className="text-xs text-green-500 flex items-center gap-1">
                  <CheckCircle2 className="h-3.5 w-3.5" />
                  {t('dashboard.relayConfigured', { count: installedCount })}
                </p>
              )}
            </div>
          )}
        </div>

        {/* Navigation */}
        <div className="flex justify-between">
          <div>
            {step > 0 && step < totalSteps - 1 && (
              <button
                onClick={() => setStep(step - 1)}
                className="flex items-center gap-1.5 px-4 py-2 rounded-md text-sm border border-border hover:bg-muted transition-colors"
              >
                <ArrowLeft className="h-4 w-4" />
                {t('wizard.back')}
              </button>
            )}
          </div>
          <div className="flex gap-2">
            {step < totalSteps - 1 && (
              <button
                onClick={handleSkip}
                className="flex items-center gap-1.5 px-4 py-2 rounded-md text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                <SkipForward className="h-4 w-4" />
                {t('wizard.skip')}
              </button>
            )}
            {step < totalSteps - 1 ? (
              <button
                onClick={() => setStep(step + 1)}
                className={cn(
                  'flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium transition-colors',
                  'bg-primary text-primary-foreground hover:bg-primary/90'
                )}
              >
                {step === 0 ? t('wizard.getStarted') : t('wizard.next')}
                <ArrowRight className="h-4 w-4" />
              </button>
            ) : (
              <button
                onClick={handleFinish}
                className={cn(
                  'flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium transition-colors',
                  'bg-primary text-primary-foreground hover:bg-primary/90'
                )}
              >
                {t('wizard.done.goToDashboard')}
                <ArrowRight className="h-4 w-4" />
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
