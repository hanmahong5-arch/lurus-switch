import { useEffect, useCallback } from 'react'
import { Download, RefreshCw, Loader2, ArrowUpCircle } from 'lucide-react'
import { cn } from '../lib/utils'
import { useDashboardStore } from '../stores/dashboardStore'
import { useConfigStore } from '../stores/configStore'
import { ToolCard } from '../components/ToolCard'
import { ProxyConfigPanel } from '../components/ProxyConfigPanel'
import {
  DetectAllTools,
  InstallTool,
  InstallAllTools,
  UpdateTool,
  UpdateAllTools,
  CheckAllUpdates,
  GetProxySettings,
  SaveProxySettings,
  ConfigureAllProxy,
  GetAppVersion,
  CheckSelfUpdate,
  ApplySelfUpdate,
} from '../../wailsjs/go/main/App'
import type { ProxySettings } from '../stores/dashboardStore'

const TOOL_ORDER = ['claude', 'codex', 'gemini'] as const

export function DashboardPage() {
  const {
    tools, installing, updating, detecting,
    proxySettings, proxySaving, proxyConfiguring,
    appVersion, selfUpdateInfo, checkingUpdates,
    setTools, setInstalling, setUpdating, setDetecting,
    setProxySettings, setProxySaving, setProxyConfiguring,
    setAppVersion, setSelfUpdateInfo, setCheckingUpdates,
  } = useDashboardStore()

  const { setActiveTool } = useConfigStore()

  // Load fast data immediately (version + proxy settings), then detect tools in background
  useEffect(() => {
    // These are instant — no subprocess calls
    GetAppVersion().then(setAppVersion).catch(() => {})
    GetProxySettings().then((r) => setProxySettings(r as any)).catch(() => {})
    // Tool detection runs subprocesses — defer slightly so UI renders first
    const timer = setTimeout(() => detectTools(), 100)
    return () => clearTimeout(timer)
  }, [])

  const detectTools = useCallback(async () => {
    setDetecting(true)
    try {
      const toolStatuses = await DetectAllTools()
      setTools(toolStatuses as any)
    } catch (err) {
      console.error('Failed to detect tools:', err)
    } finally {
      setDetecting(false)
    }
  }, [])

  const loadAll = useCallback(async () => {
    await detectTools()
  }, [])

  const checkUpdates = async (currentTools?: Record<string, any>) => {
    setCheckingUpdates(true)
    try {
      const [toolUpdates, selfUpdate] = await Promise.all([
        CheckAllUpdates(),
        CheckSelfUpdate(),
      ])

      // Merge update info into tool statuses
      const merged = { ...(currentTools || tools) }
      for (const [name, update] of Object.entries(toolUpdates as Record<string, any>)) {
        if (merged[name]) {
          merged[name] = {
            ...merged[name],
            latestVersion: update.latestVersion,
            updateAvailable: update.updateAvailable,
          }
        }
      }
      setTools(merged as any)
      setSelfUpdateInfo(selfUpdate as any)
    } catch (err) {
      console.error('Failed to check updates:', err)
    } finally {
      setCheckingUpdates(false)
    }
  }

  const handleInstall = async (toolName: string) => {
    setInstalling(toolName, true)
    try {
      await InstallTool(toolName)
      // Refresh detection after install
      const statuses = await DetectAllTools()
      setTools(statuses as any)
    } catch (err) {
      console.error(`Failed to install ${toolName}:`, err)
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
      setTools(statuses as any)
    } catch (err) {
      console.error('Failed to install all tools:', err)
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
      setTools(statuses as any)
    } catch (err) {
      console.error(`Failed to update ${toolName}:`, err)
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
      setTools(statuses as any)
    } catch (err) {
      console.error('Failed to update all tools:', err)
    } finally {
      for (const name of TOOL_ORDER) {
        setUpdating(name, false)
      }
    }
  }

  const handleConfigure = (toolName: string) => {
    setActiveTool(toolName as any)
  }

  const handleSaveProxy = async (settings: ProxySettings) => {
    setProxySaving(true)
    try {
      await SaveProxySettings(settings as any)
      setProxySettings(settings)
    } catch (err) {
      console.error('Failed to save proxy settings:', err)
    } finally {
      setProxySaving(false)
    }
  }

  const handleConfigureAllProxy = async () => {
    setProxyConfiguring(true)
    try {
      // Save first, then apply
      await SaveProxySettings(proxySettings as any)
      const errors = await ConfigureAllProxy()
      if (Object.keys(errors).length > 0) {
        console.warn('Some tools failed proxy configuration:', errors)
      }
    } catch (err) {
      console.error('Failed to configure proxy:', err)
    } finally {
      setProxyConfiguring(false)
    }
  }

  const handleSelfUpdate = async () => {
    try {
      await ApplySelfUpdate()
    } catch (err) {
      console.error('Failed to apply self-update:', err)
    }
  }

  const anyInstalling = Object.values(installing).some(Boolean)
  const anyUpdating = Object.values(updating).some(Boolean)
  const hasUpdates = TOOL_ORDER.some((name) => tools[name]?.updateAvailable)

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-3xl mx-auto p-6 space-y-6">
        {/* Header */}
        <div>
          <h2 className="text-lg font-semibold">Dashboard</h2>
          <p className="text-sm text-muted-foreground">
            Manage AI CLI tools installation and configuration
          </p>
        </div>

        {/* Tool Cards */}
        <div className="grid grid-cols-3 gap-4">
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
                onInstall={() => handleInstall(name)}
                onUpdate={() => handleUpdate(name)}
                onConfigure={() => handleConfigure(name)}
              />
            )
          })}
        </div>

        {/* Bulk actions */}
        <div className="flex gap-2">
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
            Install All
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
              Update All
            </button>
          )}

          <button
            onClick={loadAll}
            disabled={detecting}
            className={cn(
              'flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium transition-colors',
              'border border-border hover:bg-muted',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            {detecting ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="h-4 w-4" />
            )}
            Refresh
          </button>
        </div>

        {/* Proxy Configuration */}
        <ProxyConfigPanel
          settings={proxySettings}
          saving={proxySaving}
          configuring={proxyConfiguring}
          onSave={handleSaveProxy}
          onConfigureAll={handleConfigureAllProxy}
        />

        {/* App version and self-update */}
        <div className="flex items-center justify-between text-xs text-muted-foreground border-t border-border pt-4">
          <span>Lurus Switch v{appVersion || '...'}</span>
          <div className="flex items-center gap-2">
            {selfUpdateInfo?.updateAvailable ? (
              <button
                onClick={handleSelfUpdate}
                className="flex items-center gap-1 text-primary hover:underline"
              >
                <ArrowUpCircle className="h-3.5 w-3.5" />
                Update to v{selfUpdateInfo.latestVersion}
              </button>
            ) : checkingUpdates ? (
              <span className="flex items-center gap-1">
                <Loader2 className="h-3 w-3 animate-spin" />
                Checking...
              </span>
            ) : (
              <button
                onClick={() => checkUpdates()}
                className="hover:text-foreground transition-colors"
              >
                Check for updates
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
