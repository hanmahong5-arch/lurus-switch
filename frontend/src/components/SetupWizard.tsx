import { useState, useEffect, useRef } from 'react'
import { Loader2, CheckCircle2, XCircle, ArrowRight, ArrowLeft, SkipForward, Zap, User } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { ModelPicker, type Model } from './ModelPicker'
import {
  DetectAllTools,
  InstallAllTools,
  SaveProxySettings,
  GetAppSettings,
  SaveAppSettings,
  BillingValidateToken,
  FetchModelCatalog,
  QuickSetup,
} from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { proxy, appconfig } from '../../wailsjs/go/models'
import type { ToolStatus } from '../stores/dashboardStore'

// Default Lurus relay endpoint — the full-format gateway (OpenAI/Claude/Gemini)
const LURUS_RELAY_ENDPOINT = 'https://api.lurus.cn'

const toolLabels: Record<string, string> = {
  claude: 'Claude Code',
  codex: 'Codex',
  gemini: 'Gemini CLI',
  picoclaw: 'PicoClaw',
  nullclaw: 'NullClaw',
  zeroclaw: 'ZeroClaw',
  openclaw: 'OpenClaw',
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
  const totalSteps = 4

  // Step 0 — language
  const [language, setLanguage] = useState(i18n.language || 'zh')

  // Step 1 — Lurus account
  const [lurusToken, setLurusToken] = useState('')
  const [validatingToken, setValidatingToken] = useState(false)
  const [accountInfo, setAccountInfo] = useState<AccountInfo | null>(null)
  const [accountError, setAccountError] = useState('')
  const [promoCode, setPromoCode] = useState('')
  const [proxySaved, setProxySaved] = useState(false)

  // Step 2 — Model selection
  const [models, setModels] = useState<Model[]>([])
  const [loadingModels, setLoadingModels] = useState(false)
  const [selectedModel, setSelectedModel] = useState('')

  // Step 3 — Done: results of one-click setup
  const [configuring, setConfiguring] = useState(false)
  const [configResults, setConfigResults] = useState<Record<string, string>>({})
  const [tools, setTools] = useState<Record<string, ToolStatus>>({})
  const [setupDone, setSetupDone] = useState(false)

  // Background state
  const [installProgress, setInstallProgress] = useState<Record<string, number>>({})
  const modelsFetchedRef = useRef(false)

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

  // Fetch model catalog on step 2
  useEffect(() => {
    if (step === 2 && !modelsFetchedRef.current) {
      modelsFetchedRef.current = true
      setLoadingModels(true)
      FetchModelCatalog()
        .then((catalog) => {
          if (catalog?.models) {
            setModels(catalog.models)
            // Auto-select default based on language
            if (!selectedModel) {
              const defaultId = language === 'zh' ? 'deepseek-chat' : 'claude-sonnet-4-20250514'
              const exists = catalog.models.some((m: Model) => m.id === defaultId)
              if (exists) setSelectedModel(defaultId)
              else if (catalog.models.length > 0) setSelectedModel(catalog.models[0].id)
            }
          }
        })
        .catch(() => {
          // Fallback models will be empty, user can skip
        })
        .finally(() => setLoadingModels(false))
    }
  }, [step, language, selectedModel])

  // Run one-click setup on step 3 entry
  useEffect(() => {
    if (step === 3 && !setupDone) {
      runSetup()
    }
  }, [step, setupDone])

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
        // Auto-save Lurus relay as the proxy endpoint
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

  // Step 3 — run one-click setup: install all tools + configure endpoint + model
  const runSetup = async () => {
    setConfiguring(true)
    setInstallProgress({})
    try {
      // 1. Install all tools in background
      await InstallAllTools().catch(() => {})
      const statuses = await DetectAllTools()
      setTools(statuses)

      // 2. QuickSetup: configure endpoint + API key + model on all installed tools
      if (proxySaved && selectedModel) {
        const errors = await QuickSetup(selectedModel)
        setConfigResults(errors)
      }
    } catch {
      // Non-critical: user can retry from Dashboard
    } finally {
      setConfiguring(false)
      setSetupDone(true)
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
  const errorCount = Object.keys(configResults).filter(k => k !== 'error').length
  const selectedModelDisplay = models.find(m => m.id === selectedModel)

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
                  <div className="border-t border-border pt-2 mt-1">
                    <label className="block text-xs text-muted-foreground mb-0.5">{t('wizard.promoCode')}</label>
                    <input
                      type="text"
                      value={promoCode}
                      onChange={(e) => setPromoCode(e.target.value)}
                      placeholder="ABCD1234"
                      className="w-full px-3 py-1.5 text-sm rounded-md border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  <p className="text-xs text-muted-foreground">{t('wizard.account.skipHint')}</p>
                </div>
              )}
            </div>
          )}

          {/* Step 2 — Model selection */}
          {step === 2 && (
            <div className="flex-1 flex flex-col gap-3">
              <h2 className="text-lg font-medium">{t('wizard.model.title')}</h2>
              <p className="text-sm text-muted-foreground">{t('wizard.model.desc')}</p>
              <div className="flex-1 overflow-y-auto max-h-[320px]">
                <ModelPicker
                  models={models}
                  selected={selectedModel}
                  onSelect={setSelectedModel}
                  loading={loadingModels}
                />
              </div>
              {selectedModelDisplay && (
                <p className="text-xs text-muted-foreground mt-1">
                  {t('wizard.model.selected', { model: selectedModelDisplay.displayName })}
                </p>
              )}
              <p className="text-[10px] text-muted-foreground">{t('wizard.model.changeLater')}</p>
            </div>
          )}

          {/* Step 3 — Done: setup results */}
          {step === 3 && (
            <div className="flex-1 flex flex-col gap-4 items-center justify-center text-center">
              {configuring ? (
                <>
                  <Loader2 className="h-10 w-10 text-primary animate-spin" />
                  <h2 className="text-lg font-medium">{t('wizard.done.configuring')}</h2>
                  <p className="text-sm text-muted-foreground">{t('wizard.done.configuringDesc')}</p>
                  {/* Per-tool install progress */}
                  <div className="w-full max-w-xs space-y-1 mt-2">
                    {Object.entries(installProgress).map(([tool, pct]) => (
                      <div key={tool} className="flex items-center gap-2 text-xs">
                        <span className="w-20 text-right text-muted-foreground">{toolLabels[tool] || tool}</span>
                        <div className="flex-1 h-1 bg-muted rounded-full overflow-hidden">
                          <div
                            className={cn('h-full transition-all duration-300', pct === -1 ? 'bg-red-500' : 'bg-primary')}
                            style={{ width: `${Math.max(0, pct)}%` }}
                          />
                        </div>
                        <span className="w-8 text-muted-foreground">
                          {pct === -1 ? '!' : pct === 100 ? <CheckCircle2 className="h-3 w-3 text-green-500 inline" /> : `${pct}%`}
                        </span>
                      </div>
                    ))}
                  </div>
                </>
              ) : (
                <>
                  <CheckCircle2 className="h-12 w-12 text-green-500" />
                  <h2 className="text-lg font-medium">{t('wizard.done.title')}</h2>
                  <p className="text-sm text-muted-foreground">{t('wizard.done.desc')}</p>
                  <div className="text-sm space-y-1 mt-2">
                    {accountInfo && (
                      <p className="text-green-500 flex items-center justify-center gap-1">
                        <CheckCircle2 className="h-3.5 w-3.5" />
                        {t('wizard.account.connected')}: {accountInfo.displayName}
                      </p>
                    )}
                    <p>{t('wizard.done.toolsDetected', { count: installedCount })}</p>
                    {selectedModel && (
                      <p className="flex items-center justify-center gap-1">
                        <Zap className="h-3.5 w-3.5 text-primary" />
                        {t('wizard.done.modelConfigured', { model: selectedModelDisplay?.displayName || selectedModel })}
                      </p>
                    )}
                    {errorCount > 0 && (
                      <p className="text-xs text-amber-500 mt-2">
                        {t('wizard.done.someErrors', { count: errorCount })}
                      </p>
                    )}
                  </div>
                </>
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
                disabled={configuring}
                className={cn(
                  'flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium transition-colors',
                  'bg-primary text-primary-foreground hover:bg-primary/90',
                  'disabled:opacity-50 disabled:cursor-not-allowed'
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
