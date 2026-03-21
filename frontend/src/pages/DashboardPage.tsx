import { useEffect, useCallback, useState } from 'react'
import { Download, RefreshCw, Loader2, ArrowUpCircle, Trash2, Wand2, Zap, Repeat } from 'lucide-react'
import { useTranslation, Trans } from 'react-i18next'
import { cn } from '../lib/utils'
import { errorToast } from '../lib/errorToast'
import { withRetry } from '../lib/withRetry'
import { useDashboardStore, type ToolStatus, type ProxySettings } from '../stores/dashboardStore'
import { useConfigStore, type ActiveTool, type ToolsSubTab } from '../stores/configStore'
import { useToastStore } from '../stores/toastStore'
import { ToolCard } from '../components/ToolCard'
import { ProxyConfigPanel } from '../components/ProxyConfigPanel'
import { DashboardQuotaWidget } from '../components/DashboardQuotaWidget'
import { DepTreePanel } from '../components/DepTreePanel'
import { ModelPicker, type Model } from '../components/ModelPicker'
import {
  DetectAllTools,
  InstallTool,
  InstallAllTools,
  UpdateTool,
  UpdateAllTools,
  UninstallTool,
  CheckAllUpdates,
  CheckAllToolHealth,
  GetProxySettings,
  SaveProxySettings,
  ConfigureAllProxy,
  ConfigureAllToolsRelay,
  GetAppVersion,
  CheckSelfUpdate,
  ApplySelfUpdate,
  SaveAppSettings,
  GetAppSettings,
  FetchModelCatalog,
  SwitchModel,
} from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { proxy, appconfig } from '../../wailsjs/go/models'

const TOOL_ORDER = ['claude', 'codex', 'gemini', 'picoclaw', 'nullclaw', 'zeroclaw', 'openclaw'] as const

export function DashboardPage() {
  const { t } = useTranslation()
  const {
    tools, installing, updating, detecting,
    proxySettings, proxySaving, proxyConfiguring,
    appVersion, selfUpdateInfo, checkingUpdates,
    toolHealth,
    setTools, setInstalling, setUpdating, setDetecting,
    setProxySettings, setProxySaving, setProxyConfiguring,
    setAppVersion, setSelfUpdateInfo, setCheckingUpdates,
    setToolHealth,
  } = useDashboardStore()

  const { setActiveTool, setHighlightField } = useConfigStore()
  const toast = useToastStore((s) => s.addToast)

  // Uninstall state
  const [uninstalling, setUninstalling] = useState<Record<string, boolean>>({})
  const [confirmUninstall, setConfirmUninstall] = useState<string | null>(null)

  // Per-tool install progress: 0-99 = in-progress, 100 = done, -1 = failed
  const [installProgress, setInstallProgress] = useState<Record<string, number>>({})

  // Model selection state
  const [currentModel, setCurrentModel] = useState('')
  const [showModelPicker, setShowModelPicker] = useState(false)
  const [catalogModels, setCatalogModels] = useState<Model[]>([])
  const [switchingModel, setSwitchingModel] = useState(false)

  // Subscribe to install progress events emitted by the Go backend.
  useEffect(() => {
    const offProgress = EventsOn('tool:install:progress', (d: { tool: string; percent: number }) => {
      setInstallProgress(p => ({ ...p, [d.tool]: d.percent }))
    })
    const offDone = EventsOn('tool:install:done', (d: { tool: string; success: boolean }) => {
      setInstallProgress(p => ({ ...p, [d.tool]: d.success ? 100 : -1 }))
    })
    return () => { offProgress(); offDone() }
  }, [])

  const TOOL_DISPLAY: Record<string, string> = {
    claude: 'Claude Code', codex: 'Codex', gemini: 'Gemini CLI',
    picoclaw: 'PicoClaw', nullclaw: 'NullClaw', zeroclaw: 'ZeroClaw', openclaw: 'OpenClaw',
  }

  // Load fast data immediately (version + proxy settings + model), then detect tools in background
  useEffect(() => {
    GetAppVersion().then(setAppVersion).catch(() => {})
    GetProxySettings().then((r) => {
      setProxySettings(r)
      if (r.model) setCurrentModel(r.model)
    }).catch(() => {})
    FetchModelCatalog().then((cat) => {
      if (cat?.models) setCatalogModels(cat.models)
    }).catch(() => {})
    const timer = setTimeout(() => detectTools(), 100)
    return () => clearTimeout(timer)
  }, [])

  const detectTools = useCallback(async () => {
    setDetecting(true)
    try {
      const toolStatuses = await withRetry(() => DetectAllTools())
      setTools(toolStatuses)
      // Also fetch health data
      try {
        const health = await CheckAllToolHealth()
        setToolHealth(health)
      } catch {
        // Health check is non-critical
      }
    } catch (err) {
      errorToast(toast, err, { navigate: setActiveTool, currentPage: 'home', retry: () => detectTools(), t })
    } finally {
      setDetecting(false)
    }
  }, [t, setDetecting, setTools, setToolHealth, toast, setActiveTool])

  const loadAll = useCallback(async () => {
    await detectTools()
  }, [detectTools])

  const checkUpdates = async (currentTools?: Record<string, ToolStatus>) => {
    setCheckingUpdates(true)
    try {
      const [toolUpdates, selfUpdate] = await Promise.all([
        withRetry(() => CheckAllUpdates()),
        withRetry(() => CheckSelfUpdate()),
      ])

      const merged: Record<string, ToolStatus> = { ...(currentTools || tools) }
      for (const [name, update] of Object.entries(toolUpdates)) {
        if (merged[name]) {
          merged[name] = {
            ...merged[name],
            latestVersion: update.latestVersion,
            updateAvailable: update.updateAvailable,
          }
        }
      }
      setTools(merged)
      setSelfUpdateInfo(selfUpdate)
    } catch (err) {
      errorToast(toast, err, { navigate: setActiveTool, currentPage: 'home', retry: () => checkUpdates(), t })
    } finally {
      setCheckingUpdates(false)
    }
  }

  const handleInstall = async (toolName: string) => {
    setInstalling(toolName, true)
    try {
      await InstallTool(toolName)
      const statuses = await DetectAllTools()
      setTools(statuses)
      toast('success', `${TOOL_DISPLAY[toolName] || toolName} ${t('dashboard.installSuccess')}`, {
        label: t('dashboard.configure'),
        onClick: () => handleConfigure(toolName),
      })
    } catch (err) {
      errorToast(toast, err, { navigate: setActiveTool, currentPage: 'home', retry: () => handleInstall(toolName), t })
    } finally {
      setInstalling(toolName, false)
    }
  }

  const handleInstallAll = async () => {
    for (const name of TOOL_ORDER) {
      setInstalling(name, true)
    }
    try {
      await InstallAllTools()
      const statuses = await DetectAllTools()
      setTools(statuses)
      toast('success', t('dashboard.installAllSuccess'))
    } catch (err) {
      errorToast(toast, err, { navigate: setActiveTool, currentPage: 'home', retry: () => handleInstallAll(), t })
    } finally {
      for (const name of TOOL_ORDER) {
        setInstalling(name, false)
      }
    }
  }

  const handleUpdate = async (toolName: string) => {
    setUpdating(toolName, true)
    try {
      await UpdateTool(toolName)
      const statuses = await DetectAllTools()
      setTools(statuses)
      toast('success', `${TOOL_DISPLAY[toolName] || toolName} ${t('dashboard.updateSuccess')}`)
    } catch (err) {
      errorToast(toast, err, { navigate: setActiveTool, currentPage: 'home', retry: () => handleUpdate(toolName), t })
    } finally {
      setUpdating(toolName, false)
    }
  }

  const handleUpdateAll = async () => {
    for (const name of TOOL_ORDER) {
      setUpdating(name, true)
    }
    try {
      await UpdateAllTools()
      const statuses = await DetectAllTools()
      setTools(statuses)
      toast('success', t('dashboard.updateAllSuccess'))
    } catch (err) {
      errorToast(toast, err, { navigate: setActiveTool, currentPage: 'home', retry: () => handleUpdateAll(), t })
    } finally {
      for (const name of TOOL_ORDER) {
        setUpdating(name, false)
      }
    }
  }

  const handleConfigure = (toolName: string) => {
    useConfigStore.getState().setSubTab('tools', toolName)
    useConfigStore.getState().setLastActiveTool(toolName as ToolsSubTab)
    setActiveTool('tools')
  }

  const handleUninstallRequest = (toolName: string) => {
    setConfirmUninstall(toolName)
  }

  const handleUninstallConfirm = async () => {
    const toolName = confirmUninstall
    if (!toolName) return
    setConfirmUninstall(null)
    setUninstalling((prev) => ({ ...prev, [toolName]: true }))
    try {
      await UninstallTool(toolName)
      const statuses = await DetectAllTools()
      setTools(statuses)
      toast('success', `${TOOL_DISPLAY[toolName] || toolName} ${t('dashboard.uninstallSuccess')}`)
    } catch (err) {
      errorToast(toast, err, { navigate: setActiveTool, currentPage: 'home', t })
    } finally {
      setUninstalling((prev) => ({ ...prev, [toolName]: false }))
    }
  }

  const handleSaveProxy = async (settings: ProxySettings) => {
    setProxySaving(true)
    try {
      await SaveProxySettings(proxy.ProxySettings.createFrom(settings))
      setProxySettings(settings)
      toast('success', t('dashboard.proxySaved'))
    } catch (err) {
      errorToast(toast, err, { navigate: setActiveTool, currentPage: 'home', t })
    } finally {
      setProxySaving(false)
    }
  }

  const handleConfigureAllProxy = async () => {
    setProxyConfiguring(true)
    try {
      await SaveProxySettings(proxy.ProxySettings.createFrom(proxySettings))
      const errors = await ConfigureAllProxy()
      if (Object.keys(errors).length > 0) {
        const failed = Object.entries(errors).map(([tool, e]) => `${tool}: ${e}`).join('; ')
        toast('warning', failed)
      } else {
        toast('success', t('dashboard.proxyConfigured'))
      }
    } catch (err) {
      errorToast(toast, err, { navigate: setActiveTool, currentPage: 'home', t })
    } finally {
      setProxyConfiguring(false)
    }
  }

  const handleConfigureRelay = async () => {
    setProxyConfiguring(true)
    try {
      const errors = await ConfigureAllToolsRelay()
      if (Object.keys(errors).length > 0) {
        const failed = Object.entries(errors).map(([tool, e]) => `${tool}: ${e}`).join('; ')
        toast('warning', failed)
      } else {
        toast('success', t('dashboard.relayConfigured'))
      }
    } catch (err) {
      errorToast(toast, err, { navigate: setActiveTool, currentPage: 'home', t })
    } finally {
      setProxyConfiguring(false)
    }
  }

  const handleSelfUpdate = async () => {
    try {
      await ApplySelfUpdate()
      toast('success', t('dashboard.selfUpdateSuccess'))
    } catch (err) {
      errorToast(toast, err, { navigate: setActiveTool, currentPage: 'home', retry: () => handleSelfUpdate(), t })
    }
  }

  const handleRunWizard = async () => {
    try {
      const settings = await GetAppSettings()
      await SaveAppSettings(appconfig.AppSettings.createFrom({ ...settings, onboardingCompleted: false }))
      window.location.reload()
    } catch {
      // Ignore
    }
  }

  const handleSwitchModel = async (modelId: string) => {
    setSwitchingModel(true)
    try {
      const errors = await SwitchModel(modelId)
      setCurrentModel(modelId)
      setShowModelPicker(false)
      if (Object.keys(errors).length > 0) {
        const failed = Object.entries(errors).map(([tool, e]) => `${tool}: ${e}`).join('; ')
        toast('warning', failed)
      } else {
        const display = catalogModels.find(m => m.id === modelId)?.displayName || modelId
        toast('success', t('dashboard.modelSwitched', { model: display }))
      }
    } catch (err) {
      errorToast(toast, err, { navigate: setActiveTool, currentPage: 'home', t })
    } finally {
      setSwitchingModel(false)
    }
  }

  const anyInstalling = Object.values(installing).some(Boolean)
  const anyUpdating = Object.values(updating).some(Boolean)
  const hasUpdates = TOOL_ORDER.some((name) => tools[name]?.updateAvailable)
  const anyInstalled = TOOL_ORDER.some((name) => tools[name]?.installed)

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-4xl mx-auto p-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold">{t('dashboard.title')}</h2>
            <p className="text-sm text-muted-foreground">
              {t('dashboard.subtitle')}
            </p>
          </div>
          <button
            onClick={loadAll}
            disabled={detecting}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-colors',
              'border border-border hover:bg-muted',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            {detecting ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="h-4 w-4" />
            )}
            {t('dashboard.refresh')}
          </button>
        </div>

        {/* Uninstall Confirmation Modal */}
        {confirmUninstall && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-card border border-border rounded-lg p-6 max-w-sm w-full mx-4 shadow-xl">
              <div className="flex items-center gap-3 mb-4">
                <Trash2 className="h-5 w-5 text-red-500" />
                <h3 className="font-semibold">{t('dashboard.uninstallTitle', { tool: confirmUninstall })}</h3>
              </div>
              <p className="text-sm text-muted-foreground mb-6">
                <Trans
                  i18nKey="dashboard.uninstallDesc"
                  values={{ tool: confirmUninstall }}
                  components={{ bold: <strong /> }}
                />
              </p>
              <div className="flex gap-3">
                <button
                  onClick={() => setConfirmUninstall(null)}
                  className="flex-1 px-4 py-2 rounded-md text-sm border border-border hover:bg-muted transition-colors"
                >
                  {t('dashboard.uninstallCancel')}
                </button>
                <button
                  onClick={handleUninstallConfirm}
                  className="flex-1 px-4 py-2 rounded-md text-sm bg-red-500 text-white hover:bg-red-600 transition-colors"
                >
                  {t('dashboard.uninstallConfirm')}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Quota Widget */}
        <DashboardQuotaWidget />

        {/* Current Model */}
        {currentModel && (
          <div className="flex items-center justify-between p-4 rounded-lg border border-border bg-card">
            <div className="flex items-center gap-3">
              <Zap className="h-5 w-5 text-primary" />
              <div>
                <p className="text-sm font-medium">{t('dashboard.currentModel')}</p>
                <p className="text-xs text-muted-foreground">
                  {catalogModels.find(m => m.id === currentModel)?.displayName || currentModel}
                  <span className="ml-2 font-mono text-[10px]">{currentModel}</span>
                </p>
              </div>
            </div>
            <button
              onClick={() => setShowModelPicker(!showModelPicker)}
              disabled={switchingModel}
              className={cn(
                'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
                'border border-border hover:bg-muted',
                'disabled:opacity-50 disabled:cursor-not-allowed'
              )}
            >
              {switchingModel ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Repeat className="h-3.5 w-3.5" />
              )}
              {t('dashboard.switchModel')}
            </button>
          </div>
        )}

        {/* Model Picker Modal */}
        {showModelPicker && (
          <div className="border border-border rounded-lg p-4 bg-card">
            <ModelPicker
              models={catalogModels}
              selected={currentModel}
              onSelect={handleSwitchModel}
              loading={switchingModel}
            />
          </div>
        )}

        {/* Runtime Dependencies */}
        <DepTreePanel />

        {/* Tool Cards or Empty State */}
        {!detecting && !anyInstalled && Object.keys(tools).length > 0 ? (
          <div className="border border-dashed border-border rounded-lg p-8 flex flex-col items-center gap-3 text-center">
            <p className="text-sm font-medium">{t('dashboard.noToolsTitle')}</p>
            <p className="text-xs text-muted-foreground">{t('dashboard.noToolsDesc')}</p>
            <button
              onClick={handleRunWizard}
              className={cn(
                'flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium transition-colors',
                'bg-primary text-primary-foreground hover:bg-primary/90'
              )}
            >
              <Wand2 className="h-4 w-4" />
              {t('dashboard.runWizard')}
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4">
            {TOOL_ORDER.map((name) => {
              const tool = tools[name] || {
                name,
                installed: false,
                version: '',
                latestVersion: '',
                updateAvailable: false,
                path: '',
              }
              const pct = installProgress[name]
              const isActiveInstall = installing[name] && pct !== undefined && pct >= 0 && pct < 100
              return (
                <div key={name} className="flex flex-col gap-0.5">
                  <ToolCard
                    tool={tool}
                    installing={installing[name] || false}
                    updating={updating[name] || false}
                    uninstalling={uninstalling[name] || false}
                    health={toolHealth[name]}
                    onInstall={() => handleInstall(name)}
                    onUpdate={() => handleUpdate(name)}
                    onConfigure={() => handleConfigure(name)}
                    onUninstall={tool.installed ? () => handleUninstallRequest(name) : undefined}
                    onViewIssues={tool.installed && toolHealth[name]?.status === 'red' ? () => {
                      const issues = toolHealth[name]?.issues
                      if (issues?.length) setHighlightField(issues[0])
                      handleConfigure(name)
                    } : undefined}
                  />
                  {isActiveInstall && (
                    <div className="px-4 pb-1">
                      <div className="w-full h-1 bg-muted rounded-full overflow-hidden">
                        <div
                          className="h-full bg-primary transition-all duration-300"
                          style={{ width: `${pct}%` }}
                        />
                      </div>
                      <p className="text-[10px] text-muted-foreground mt-0.5">{pct}%</p>
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}

        {/* Bulk actions */}
        <div className="flex gap-2 flex-wrap">
          <button
            onClick={handleInstallAll}
            disabled={anyInstalling || detecting}
            className={cn(
              'flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium transition-colors',
              'bg-primary text-primary-foreground hover:bg-primary/90',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            {anyInstalling ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Download className="h-4 w-4" />
            )}
            {t('dashboard.installAll')}
          </button>

          {hasUpdates && (
            <button
              onClick={handleUpdateAll}
              disabled={anyUpdating || detecting}
              className={cn(
                'flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium transition-colors',
                'bg-amber-500 text-white hover:bg-amber-600',
                'disabled:opacity-50 disabled:cursor-not-allowed'
              )}
            >
              {anyUpdating ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <ArrowUpCircle className="h-4 w-4" />
              )}
              {t('dashboard.updateAll')}
            </button>
          )}
        </div>

        {/* Proxy Configuration (collapsible) */}
        <details className="group">
          <summary className="cursor-pointer text-sm font-medium text-muted-foreground hover:text-foreground transition-colors list-none flex items-center gap-2">
            <span className="text-xs transition-transform group-open:rotate-90">&#9654;</span>
            {t('dashboard.proxyConfig')}
          </summary>
          <div className="mt-3 space-y-3">
            <ProxyConfigPanel
              settings={proxySettings}
              saving={proxySaving}
              configuring={proxyConfiguring}
              onSave={handleSaveProxy}
              onConfigureAll={handleConfigureAllProxy}
            />
            {proxySettings.apiEndpoint && (proxySettings.userToken || proxySettings.apiKey) && (
              <button
                onClick={handleConfigureRelay}
                disabled={proxyConfiguring}
                className={cn(
                  'flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium transition-colors',
                  'bg-primary text-primary-foreground hover:bg-primary/90',
                  'disabled:opacity-50 disabled:cursor-not-allowed'
                )}
              >
                {proxyConfiguring ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Zap className="h-4 w-4" />
                )}
                {t('dashboard.configureLurusRelay')}
              </button>
            )}
          </div>
        </details>

        {/* App version and self-update */}
        <div className="flex items-center justify-between text-xs text-muted-foreground border-t border-border pt-4">
          <span>{t('dashboard.version', { version: appVersion || '...' })}</span>
          <div className="flex items-center gap-2">
            {selfUpdateInfo?.updateAvailable ? (
              <button
                onClick={handleSelfUpdate}
                className="flex items-center gap-1 text-primary hover:underline"
              >
                <ArrowUpCircle className="h-3.5 w-3.5" />
                {t('dashboard.updateTo', { version: selfUpdateInfo.latestVersion })}
              </button>
            ) : checkingUpdates ? (
              <span className="flex items-center gap-1">
                <Loader2 className="h-3 w-3 animate-spin" />
                {t('dashboard.checking')}
              </span>
            ) : (
              <button
                onClick={() => checkUpdates()}
                className="hover:text-foreground transition-colors"
              >
                {t('dashboard.checkUpdates')}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
