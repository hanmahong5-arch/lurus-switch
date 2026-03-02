import { useEffect, useCallback, useState } from 'react'
import { Download, RefreshCw, Loader2, ArrowUpCircle, Trash2, Wand2, Zap } from 'lucide-react'
import { useTranslation, Trans } from 'react-i18next'
import { cn } from '../lib/utils'
import { useDashboardStore, type ToolStatus, type ProxySettings } from '../stores/dashboardStore'
import { useConfigStore, type ActiveTool } from '../stores/configStore'
import { ToolCard } from '../components/ToolCard'
import { ProxyConfigPanel } from '../components/ProxyConfigPanel'
import { DashboardQuotaWidget } from '../components/DashboardQuotaWidget'
import { DepTreePanel } from '../components/DepTreePanel'
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
} from '../../wailsjs/go/main/App'
import { proxy, appconfig } from '../../wailsjs/go/models'

const TOOL_ORDER = ['claude', 'codex', 'gemini', 'picoclaw', 'nullclaw', 'zeroclaw', 'openclaw'] as const

export function DashboardPage() {
  const { t } = useTranslation()
  const {
    tools, installing, updating, detecting,
    proxySettings, proxySaving, proxyConfiguring,
    appVersion, selfUpdateInfo, checkingUpdates, error,
    toolHealth,
    setTools, setInstalling, setUpdating, setDetecting,
    setProxySettings, setProxySaving, setProxyConfiguring,
    setAppVersion, setSelfUpdateInfo, setCheckingUpdates, setError,
    setToolHealth,
  } = useDashboardStore()

  const { setActiveTool } = useConfigStore()

  // Uninstall state
  const [uninstalling, setUninstalling] = useState<Record<string, boolean>>({})
  const [confirmUninstall, setConfirmUninstall] = useState<string | null>(null)

  // Load fast data immediately (version + proxy settings), then detect tools in background
  useEffect(() => {
    GetAppVersion().then(setAppVersion).catch(() => {})
    GetProxySettings().then((r) => setProxySettings(r)).catch(() => {})
    const timer = setTimeout(() => detectTools(), 100)
    return () => clearTimeout(timer)
  }, [])

  const detectTools = useCallback(async () => {
    setDetecting(true)
    setError(null)
    try {
      const toolStatuses = await DetectAllTools()
      setTools(toolStatuses)
      // Also fetch health data
      try {
        const health = await CheckAllToolHealth()
        setToolHealth(health)
      } catch {
        // Health check is non-critical
      }
    } catch (err) {
      setError(`${t('dashboard.title')}: ${err}`)
    } finally {
      setDetecting(false)
    }
  }, [t, setDetecting, setError, setTools, setToolHealth])

  const loadAll = useCallback(async () => {
    await detectTools()
  }, [detectTools])

  const checkUpdates = async (currentTools?: Record<string, ToolStatus>) => {
    setCheckingUpdates(true)
    setError(null)
    try {
      const [toolUpdates, selfUpdate] = await Promise.all([
        CheckAllUpdates(),
        CheckSelfUpdate(),
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
      setError(`${err}`)
    } finally {
      setCheckingUpdates(false)
    }
  }

  const handleInstall = async (toolName: string) => {
    setInstalling(toolName, true)
    setError(null)
    try {
      await InstallTool(toolName)
      const statuses = await DetectAllTools()
      setTools(statuses)
    } catch (err) {
      setError(`${err}`)
    } finally {
      setInstalling(toolName, false)
    }
  }

  const handleInstallAll = async () => {
    setError(null)
    for (const name of TOOL_ORDER) {
      setInstalling(name, true)
    }
    try {
      await InstallAllTools()
      const statuses = await DetectAllTools()
      setTools(statuses)
    } catch (err) {
      setError(`${err}`)
    } finally {
      for (const name of TOOL_ORDER) {
        setInstalling(name, false)
      }
    }
  }

  const handleUpdate = async (toolName: string) => {
    setUpdating(toolName, true)
    setError(null)
    try {
      await UpdateTool(toolName)
      const statuses = await DetectAllTools()
      setTools(statuses)
    } catch (err) {
      setError(`${err}`)
    } finally {
      setUpdating(toolName, false)
    }
  }

  const handleUpdateAll = async () => {
    setError(null)
    for (const name of TOOL_ORDER) {
      setUpdating(name, true)
    }
    try {
      await UpdateAllTools()
      const statuses = await DetectAllTools()
      setTools(statuses)
    } catch (err) {
      setError(`${err}`)
    } finally {
      for (const name of TOOL_ORDER) {
        setUpdating(name, false)
      }
    }
  }

  const handleConfigure = (toolName: string) => {
    setActiveTool(toolName as ActiveTool)
  }

  const handleUninstallRequest = (toolName: string) => {
    setConfirmUninstall(toolName)
  }

  const handleUninstallConfirm = async () => {
    const toolName = confirmUninstall
    if (!toolName) return
    setConfirmUninstall(null)
    setUninstalling((prev) => ({ ...prev, [toolName]: true }))
    setError(null)
    try {
      await UninstallTool(toolName)
      const statuses = await DetectAllTools()
      setTools(statuses)
    } catch (err) {
      setError(`${err}`)
    } finally {
      setUninstalling((prev) => ({ ...prev, [toolName]: false }))
    }
  }

  const handleSaveProxy = async (settings: ProxySettings) => {
    setProxySaving(true)
    setError(null)
    try {
      await SaveProxySettings(proxy.ProxySettings.createFrom(settings))
      setProxySettings(settings)
    } catch (err) {
      setError(`${err}`)
    } finally {
      setProxySaving(false)
    }
  }

  const handleConfigureAllProxy = async () => {
    setProxyConfiguring(true)
    setError(null)
    try {
      await SaveProxySettings(proxy.ProxySettings.createFrom(proxySettings))
      const errors = await ConfigureAllProxy()
      if (Object.keys(errors).length > 0) {
        const failed = Object.entries(errors).map(([t, e]) => `${t}: ${e}`).join('; ')
        setError(failed)
      }
    } catch (err) {
      setError(`${err}`)
    } finally {
      setProxyConfiguring(false)
    }
  }

  const handleConfigureRelay = async () => {
    setProxyConfiguring(true)
    setError(null)
    try {
      const errors = await ConfigureAllToolsRelay()
      if (Object.keys(errors).length > 0) {
        const failed = Object.entries(errors).map(([t, e]) => `${t}: ${e}`).join('; ')
        setError(failed)
      }
    } catch (err) {
      setError(`${err}`)
    } finally {
      setProxyConfiguring(false)
    }
  }

  const handleSelfUpdate = async () => {
    setError(null)
    try {
      await ApplySelfUpdate()
    } catch (err) {
      setError(`${err}`)
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

        {/* Error banner */}
        {error && (
          <div className="flex items-center justify-between px-4 py-2 bg-red-500/10 text-red-500 text-xs rounded-md border border-red-500/20">
            <span>{error}</span>
            <button onClick={() => setError(null)} className="ml-2 hover:text-red-400 font-medium">
              {t('dashboard.dismiss')}
            </button>
          </div>
        )}

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
              return (
                <ToolCard
                  key={name}
                  tool={tool}
                  installing={installing[name] || false}
                  updating={updating[name] || false}
                  uninstalling={uninstalling[name] || false}
                  health={toolHealth[name]}
                  onInstall={() => handleInstall(name)}
                  onUpdate={() => handleUpdate(name)}
                  onConfigure={() => handleConfigure(name)}
                  onUninstall={tool.installed ? () => handleUninstallRequest(name) : undefined}
                />
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
