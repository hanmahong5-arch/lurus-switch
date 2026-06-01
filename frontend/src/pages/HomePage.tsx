import { useEffect, useCallback, useState } from 'react'
import { Loader2, Download, ArrowUpCircle, Wand2, Zap, Repeat, Wrench, AlertTriangle } from 'lucide-react'
import { useTranslation, Trans } from 'react-i18next'
import { TOOL_ORDER, TOOL_DISPLAY } from '../lib/toolMeta'
import { Button, Card, EmptyState, Modal } from '../components/ui'
import { errorToast } from '../lib/errorToast'
import { withRetry } from '../lib/withRetry'
import { useHomeStore, type Suggestion } from '../stores/homeStore'
import { useConfigStore } from '../stores/configStore'
import { useToastStore } from '../stores/toastStore'
import { useGYStore } from '../stores/gyStore'
import { usePageActionsStore } from '../stores/pageActionsStore'
import { ToolCard } from '../components/ToolCard'
import { QuickActionCards } from '../components/QuickActionCards'
import { StatusOverview } from '../components/StatusOverview'
import { DashboardQuotaWidget } from '../components/DashboardQuotaWidget'
import { DepTreePanel } from '../components/DepTreePanel'
import { HealthScoreGauge } from '../components/HealthScoreGauge'
import { QuickActions } from '../components/QuickActions'
import { OptimizationPanel } from '../components/OptimizationPanel'
import { HomeIntentPanel } from '../components/HomeIntentPanel'
import { RuntimeStatusPanel } from '../components/RuntimeStatusPanel'
import { TopologyView } from '../components/TopologyView'
import { HomeAccountHero } from '../components/account/HomeAccountHero'
import { AgentFleetCard } from '../components/AgentFleetCard'
import { ModelPicker, type Model } from '../components/ModelPicker'
import { useDashboardStore, type ToolStatus, type ProxySettings } from '../stores/dashboardStore'
import type { gy } from '../../wailsjs/go/models'
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
  GetAppVersion,
  CheckSelfUpdate,
  ApplySelfUpdate,
  FetchModelCatalog,
  SwitchModel,
  FullSetupForGateway,
  AutoConfigureToolsForGateway,
  StartGateway,
  GetGatewayStatus,
  GetGYProducts,
  CheckGYStatus,
  LaunchGYProduct,
  InstallDependency,
  AutoFixToolConfig,
  AutoConfigureToolForGateway,
  ApplyAllOptimizations,
} from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

// Lazy-resolve ComputeHealthScore — the Wails binding may not exist yet
let _resolvedHealthScore: (() => Promise<any>) | null | undefined = undefined
async function getComputeHealthScore(): Promise<(() => Promise<any>) | null> {
  if (_resolvedHealthScore !== undefined) return _resolvedHealthScore
  try {
    const mod = await import('../../wailsjs/go/main/App')
    _resolvedHealthScore = ('ComputeHealthScore' in mod)
      ? (mod as any).ComputeHealthScore as () => Promise<any>
      : null
  } catch {
    _resolvedHealthScore = null
  }
  return _resolvedHealthScore
}

// Many Wails bindings return a Result-shaped object: { success, message, ... }.
// When success===false the binding did NOT throw — but the operation failed
// (e.g. GitHub release API blocked in CN). Without this check the UI shows a
// fake success toast while the underlying tool stays broken. Mirror the
// TopologyView pattern: throw so the outer try/catch surfaces via errorToast.
type WailsResult = { success?: boolean; message?: string } | null | undefined
function ensureSuccess(result: WailsResult, fallback: string): void {
  if (result && result.success === false) {
    throw new Error(result.message || fallback)
  }
}


export function HomePage() {
  const { t } = useTranslation()
  const {
    scoreReport, scoreLoading,
    tools, installing, updating, detecting, toolHealth,
    appVersion, selfUpdateInfo, checkingUpdates,
    configuring,
    setScoreReport, setScoreLoading,
    setTools, setInstalling, setUpdating, setDetecting,
    setToolHealth, setAppVersion, setSelfUpdateInfo,
    setCheckingUpdates, setConfiguring, setError,
  } = useHomeStore()

  const { setActiveTool, setSubTab, appMode } = useConfigStore()
  const toast = useToastStore((s) => s.addToast)
  const { products, setProducts, setStatuses } = useGYStore()

  const [uninstalling, setUninstalling] = useState<Record<string, boolean>>({})
  const [confirmUninstall, setConfirmUninstall] = useState<string | null>(null)
  const [installProgress, setInstallProgress] = useState<Record<string, number>>({})
  const [executingActions, setExecutingActions] = useState<Record<string, boolean>>({})
  const [currentModel, setCurrentModel] = useState('')
  const [showModelPicker, setShowModelPicker] = useState(false)
  const [catalogModels, setCatalogModels] = useState<Model[]>([])
  const [switchingModel, setSwitchingModel] = useState(false)
  const [quickStarting, setQuickStarting] = useState<Record<string, boolean>>({})
  const [fixingAll, setFixingAll] = useState(false)

  // Subscribe to install progress events
  useEffect(() => {
    const offProgress = EventsOn('tool:install:progress', (d: { tool: string; percent: number }) => {
      setInstallProgress(p => ({ ...p, [d.tool]: d.percent }))
    })
    const offDone = EventsOn('tool:install:done', (d: { tool: string; success: boolean }) => {
      setInstallProgress(p => ({ ...p, [d.tool]: d.success ? 100 : -1 }))
    })
    return () => { offProgress(); offDone() }
  }, [])

  // Load initial data — also sync to dashboardStore so components reading
  // from the old store (StatusBar, NewToolsPage, AccountStatusBadge, etc.) stay up-to-date.
  useEffect(() => {
    GetAppVersion().then((v) => {
      setAppVersion(v)
      useDashboardStore.getState().setAppVersion(v)
    }).catch(() => {})
    GetProxySettings().then((r) => {
      useHomeStore.getState().setProxySettings(r)
      useDashboardStore.getState().setProxySettings(r)
      if (r.model) setCurrentModel(r.model)
    }).catch(() => {})
    FetchModelCatalog().then((cat) => {
      if (cat?.models) setCatalogModels(cat.models)
    }).catch(() => {})
    // Load GY products
    GetGYProducts().then((ps) => setProducts(ps || [])).catch(() => {})
    CheckGYStatus().then((ss) => {
      const map: Record<string, gy.GYStatus> = {}
      for (const s of ss || []) map[s.productId] = s
      setStatuses(map)
    }).catch(() => {})

    const timer = setTimeout(() => {
      detectTools()
      loadHealthScore()
    }, 100)
    return () => clearTimeout(timer)
  }, [])

  // Register the page-level refresh with the global header. The header
  // renders the refresh icon only when a handler is registered, so other
  // pages that haven't opted in stay clean.
  const setRefreshHandler = usePageActionsStore((s) => s.setRefreshHandler)

  const loadHealthScore = useCallback(async () => {
    const fn = await getComputeHealthScore()
    if (!fn) return
    setScoreLoading(true)
    try {
      const report = await fn()
      setScoreReport(report)
    } catch {
      // Health score is non-critical
    } finally {
      setScoreLoading(false)
    }
  }, [setScoreReport, setScoreLoading])

  const detectTools = useCallback(async () => {
    setDetecting(true)
    try {
      const toolStatuses = await withRetry(() => DetectAllTools())
      setTools(toolStatuses)
      useDashboardStore.getState().setTools(toolStatuses)
      try {
        const health = await CheckAllToolHealth()
        setToolHealth(health)
        useDashboardStore.getState().setToolHealth(health)
      } catch (e) {
        // Tool health is best-effort but failure shouldn't be invisible —
        // surface a low-priority toast so the user knows the health panel
        // is stale instead of silently showing stale data.
        toast('warning', t('home.healthCheckFailed', '健康检查失败') + ': ' + String((e as Error)?.message ?? e))
      }
    } catch (err) {
      errorToast(toast, err, { navigate: (p: string) => setActiveTool(p as any), currentPage: 'home', retry: () => detectTools(), t })
    } finally {
      setDetecting(false)
    }
  }, [t, setDetecting, setTools, setToolHealth, toast, setActiveTool])

  useEffect(() => {
    setRefreshHandler(async () => {
      await detectTools()
      await loadHealthScore()
    })
    return () => setRefreshHandler(null)
  }, [detectTools, loadHealthScore, setRefreshHandler])

  const handleOptimize = useCallback(async () => {
    setConfiguring(true)
    try {
      const result = await FullSetupForGateway()
      if (result.errors?.length > 0) {
        toast('warning', result.errors.join('; '))
      } else {
        toast('success', t('home.optimizeSuccess'))
      }
      // Refresh
      await detectTools()
      await loadHealthScore()
    } catch (err) {
      errorToast(toast, err, { navigate: (p: string) => setActiveTool(p as any), currentPage: 'home', t })
    } finally {
      setConfiguring(false)
    }
  }, [t, toast, setActiveTool, setConfiguring, detectTools, loadHealthScore])

  const handleQuickAction = useCallback(async (suggestion: Suggestion) => {
    setExecutingActions(prev => ({ ...prev, [suggestion.id]: true }))
    try {
      switch (suggestion.action) {
        case 'install-tool':
          ensureSuccess(await InstallTool(suggestion.target), t('home.installFailed', '安装失败'))
          break
        case 'install-runtime':
          ensureSuccess(await InstallDependency(suggestion.target), t('home.installFailed', '安装失败'))
          break
        case 'update-tool':
          ensureSuccess(await UpdateTool(suggestion.target), t('home.updateFailed', '更新失败'))
          break
        case 'start-gateway':
          await StartGateway()
          break
        case 'connect-gateway':
          await AutoConfigureToolsForGateway()
          break
        case 'fix-config':
          ensureSuccess(await AutoFixToolConfig(suggestion.target), t('home.fixFailed', '修复失败'))
          break
        default:
          break
      }
      // Refresh after action
      await detectTools()
      await loadHealthScore()
      toast('success', suggestion.title + ' - done')
    } catch (err) {
      errorToast(toast, err, { navigate: (p: string) => setActiveTool(p as any), currentPage: 'home', t })
    } finally {
      setExecutingActions(prev => ({ ...prev, [suggestion.id]: false }))
    }
  }, [t, toast, setActiveTool, detectTools, loadHealthScore])

  const handleInstall = async (toolName: string) => {
    // Guard: skip if any install/update is already running
    const { installing: curInstalling, updating: curUpdating } = useHomeStore.getState()
    if (curInstalling[toolName] || Object.values(curInstalling).some(Boolean) || Object.values(curUpdating).some(Boolean)) return
    setInstalling(toolName, true)
    try {
      ensureSuccess(await InstallTool(toolName), t('home.installFailed', '安装失败'))
      const statuses = await DetectAllTools()
      setTools(statuses)
      toast('success', `${TOOL_DISPLAY[toolName] || toolName} ${t('dashboard.installSuccess')}`, {
        label: t('dashboard.configure'),
        onClick: () => handleConfigure(toolName),
      })
      loadHealthScore()
    } catch (err) {
      errorToast(toast, err, { navigate: (p: string) => setActiveTool(p as any), currentPage: 'home', retry: () => handleInstall(toolName), t })
    } finally {
      setInstalling(toolName, false)
    }
  }

  const handleInstallAll = async () => {
    // Guard: skip if any install is already running
    const { installing: curInstalling } = useHomeStore.getState()
    if (Object.values(curInstalling).some(Boolean)) return
    for (const name of TOOL_ORDER) setInstalling(name, true)
    try {
      const results = await InstallAllTools()
      const statuses = await DetectAllTools()
      setTools(statuses)
      const failed = Array.isArray(results) ? results.filter(r => r && r.success === false) : []
      if (failed.length > 0) {
        const detail = failed.map(r => `${r.tool}: ${r.message || t('home.installFailed', '安装失败')}`).join('; ')
        toast('warning', detail)
      } else {
        toast('success', t('dashboard.installAllSuccess'))
      }
      loadHealthScore()
    } catch (err) {
      errorToast(toast, err, { navigate: (p: string) => setActiveTool(p as any), currentPage: 'home', retry: () => handleInstallAll(), t })
    } finally {
      for (const name of TOOL_ORDER) setInstalling(name, false)
    }
  }

  const handleUpdate = async (toolName: string) => {
    setUpdating(toolName, true)
    try {
      ensureSuccess(await UpdateTool(toolName), t('home.updateFailed', '更新失败'))
      const statuses = await DetectAllTools()
      setTools(statuses)
      toast('success', `${TOOL_DISPLAY[toolName] || toolName} ${t('dashboard.updateSuccess')}`)
      loadHealthScore()
    } catch (err) {
      errorToast(toast, err, { navigate: (p: string) => setActiveTool(p as any), currentPage: 'home', retry: () => handleUpdate(toolName), t })
    } finally {
      setUpdating(toolName, false)
    }
  }

  const handleUpdateAll = async () => {
    for (const name of TOOL_ORDER) setUpdating(name, true)
    try {
      const results = await UpdateAllTools()
      const statuses = await DetectAllTools()
      setTools(statuses)
      const failed = Array.isArray(results) ? results.filter(r => r && r.success === false) : []
      if (failed.length > 0) {
        const detail = failed.map(r => `${r.tool}: ${r.message || t('home.updateFailed', '更新失败')}`).join('; ')
        toast('warning', detail)
      } else {
        toast('success', t('dashboard.updateAllSuccess'))
      }
      loadHealthScore()
    } catch (err) {
      errorToast(toast, err, { navigate: (p: string) => setActiveTool(p as any), currentPage: 'home', retry: () => handleUpdateAll(), t })
    } finally {
      for (const name of TOOL_ORDER) setUpdating(name, false)
    }
  }

  const handleConfigure = (toolName: string) => {
    setActiveTool('tools')
    setSubTab('tools', toolName)
  }

  const handleUninstallRequest = (toolName: string) => setConfirmUninstall(toolName)

  const handleUninstallConfirm = async () => {
    const toolName = confirmUninstall
    if (!toolName) return
    setConfirmUninstall(null)
    setUninstalling((prev) => ({ ...prev, [toolName]: true }))
    try {
      ensureSuccess(await UninstallTool(toolName), t('home.uninstallFailed', '卸载失败'))
      const statuses = await DetectAllTools()
      setTools(statuses)
      toast('success', `${TOOL_DISPLAY[toolName] || toolName} ${t('dashboard.uninstallSuccess')}`)
      loadHealthScore()
    } catch (err) {
      errorToast(toast, err, { navigate: (p: string) => setActiveTool(p as any), currentPage: 'home', t })
    } finally {
      setUninstalling((prev) => ({ ...prev, [toolName]: false }))
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
      errorToast(toast, err, { navigate: (p: string) => setActiveTool(p as any), currentPage: 'home', t })
    } finally {
      setSwitchingModel(false)
    }
  }

  const handleQuickStart = async (toolName: string) => {
    setQuickStarting((prev) => ({ ...prev, [toolName]: true }))
    try {
      ensureSuccess(await InstallTool(toolName), t('home.installFailed', '安装失败'))
      try {
        const gwStatus = await GetGatewayStatus()
        if (gwStatus?.running) {
          ensureSuccess(
            await AutoConfigureToolForGateway(toolName),
            t('home.configureFailed', '网关配置失败'),
          )
        }
      } catch (e) {
        // Gateway may not be running — degrade to a warning so the rest of
        // quick-start still completes; don't fail the whole flow.
        toast('warning', t('home.configureFailed', '网关配置失败') + ': ' + String((e as Error)?.message ?? e))
      }
      try {
        ensureSuccess(await AutoFixToolConfig(toolName), t('home.fixFailed', '修复失败'))
      } catch (e) {
        // Auto-fix is best-effort, but a silent failure leaves the tool
        // broken while showing success. Surface as warning, continue.
        toast('warning', t('home.fixFailed', '修复失败') + ': ' + String((e as Error)?.message ?? e))
      }
      await detectTools()
      await loadHealthScore()
      toast('success', t('home.quickStartSuccess', { tool: TOOL_DISPLAY[toolName] || toolName }))
    } catch (err) {
      errorToast(toast, err, { navigate: (p: string) => setActiveTool(p as any), currentPage: 'home', t })
    } finally {
      setQuickStarting((prev) => ({ ...prev, [toolName]: false }))
    }
  }

  const handleFixAll = async () => {
    setFixingAll(true)
    try {
      const results = await ApplyAllOptimizations()
      await FullSetupForGateway()
      await detectTools()
      await loadHealthScore()
      const fixed = Array.isArray(results) ? results.filter((r: any) => r.success).length : 0
      const total = Array.isArray(results) ? results.length : 0
      if (total === 0 || fixed === total) {
        toast('success', t('home.fixAllSuccess'))
      } else {
        toast('warning', t('home.fixPartial', { fixed, total }))
      }
    } catch (err) {
      errorToast(toast, err, { navigate: (p: string) => setActiveTool(p as any), currentPage: 'home', t })
    } finally {
      setFixingAll(false)
    }
  }

  const handleStartGateway = async () => {
    try {
      await StartGateway()
      toast('success', t('switch.startSuccess'))
    } catch (err) {
      errorToast(toast, err, { navigate: (p: string) => setActiveTool(p as any), currentPage: 'home', t })
    }
  }

  const handleConnectAll = async () => {
    try {
      await AutoConfigureToolsForGateway()
      await detectTools()
      await loadHealthScore()
      toast('success', t('home.optimizeSuccess'))
    } catch (err) {
      errorToast(toast, err, { navigate: (p: string) => setActiveTool(p as any), currentPage: 'home', t })
    }
  }

  const checkUpdates = async () => {
    setCheckingUpdates(true)
    try {
      const [toolUpdates, selfUpdate] = await Promise.all([
        withRetry(() => CheckAllUpdates()),
        withRetry(() => CheckSelfUpdate()),
      ])
      const merged: Record<string, ToolStatus> = { ...tools }
      for (const [name, update] of Object.entries(toolUpdates)) {
        if (merged[name]) {
          merged[name] = { ...merged[name], latestVersion: update.latestVersion, updateAvailable: update.updateAvailable }
        }
      }
      setTools(merged)
      setSelfUpdateInfo(selfUpdate)
    } catch (err) {
      errorToast(toast, err, { navigate: (p: string) => setActiveTool(p as any), currentPage: 'home', retry: () => checkUpdates(), t })
    } finally {
      setCheckingUpdates(false)
    }
  }

  const handleSelfUpdate = async () => {
    try {
      await ApplySelfUpdate()
      toast('success', t('dashboard.selfUpdateSuccess'))
    } catch (err) {
      errorToast(toast, err, { navigate: (p: string) => setActiveTool(p as any), currentPage: 'home', retry: () => handleSelfUpdate(), t })
    }
  }

  const anyInstalling = Object.values(installing).some(Boolean)
  const anyUpdating = Object.values(updating).some(Boolean)
  const hasUpdates = TOOL_ORDER.some((name) => tools[name]?.updateAvailable)
  const anyInstalled = TOOL_ORDER.some((name) => tools[name]?.installed)

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-[1600px] mx-auto p-6 space-y-6">
        {/* Account summary hero — welcome + wallet/usage/subscription at a
            glance. Triggers initial refresh + 60s background polling on
            mount; hidden in EndUser mode (its own dashboard handles it). */}
        <HomeAccountHero />

        {/* Architecture topology — first thing users see, click-through to
            the page that fixes whichever entity is red. The user's "360-grade
            GUI" directive: every branch state visible with an actionable next
            step. */}
        <TopologyView />

        {/* Intent panel — verb-first entry points */}
        <HomeIntentPanel />

        {/* Live runtime status — endpoint reachability + process state */}
        <RuntimeStatusPanel />

        {/* Agent fleet entry — only Personal mode has the Agents page. */}
        {appMode === 'personal' && <AgentFleetCard />}

        {/* Section A: Health Score Gauge */}
        <HealthScoreGauge
          report={scoreReport}
          loading={scoreLoading}
          onOptimize={handleOptimize}
          optimizing={configuring}
        />

        {/* Quick Action Cards */}
        <QuickActionCards
          onInstallAll={handleInstallAll}
          onStartGateway={handleStartGateway}
          onConnectAll={handleConnectAll}
          onFixAll={handleFixAll}
          onDiagnostics={() => { detectTools(); loadHealthScore() }}
          installingAll={anyInstalling}
          fixing={fixingAll}
        />

        {/* Section B: Quick Actions / Suggestions */}
        {scoreReport && (
          <QuickActions
            suggestions={scoreReport.suggestions || []}
            onAction={handleQuickAction}
            executing={executingActions}
          />
        )}

        {/* Section C: Optimization Suggestions (regular+ users) */}
        <OptimizationPanel onRefresh={() => { detectTools(); loadHealthScore() }} />

        {/* Uninstall Confirmation Modal */}
        <Modal
          open={!!confirmUninstall}
          onClose={() => setConfirmUninstall(null)}
          title={confirmUninstall ? t('dashboard.uninstallTitle', { tool: confirmUninstall }) : ''}
          icon={AlertTriangle}
          size="sm"
          footer={
            <>
              <Button variant="secondary" size="sm" onClick={() => setConfirmUninstall(null)}>
                {t('dashboard.uninstallCancel')}
              </Button>
              <Button variant="danger" size="sm" onClick={handleUninstallConfirm}>
                {t('dashboard.uninstallConfirm')}
              </Button>
            </>
          }
        >
          <p className="text-sm text-muted-foreground leading-relaxed">
            <Trans
              i18nKey="dashboard.uninstallDesc"
              values={{ tool: confirmUninstall ?? '' }}
              components={{ bold: <strong /> }}
            />
          </p>
        </Modal>

        {/* Quota Widget */}
        <DashboardQuotaWidget />

        {/* Current Model */}
        {currentModel && (
          <Card variant="elevated" className="flex items-center justify-between p-4">
            <div className="flex items-center gap-3">
              <Zap className="h-5 w-5 text-primary" />
              <div>
                <p className="text-sm font-medium">{t('dashboard.currentModel')}</p>
                <p className="text-xs text-muted-foreground font-mono">
                  {catalogModels.find(m => m.id === currentModel)?.displayName || currentModel}
                </p>
              </div>
            </div>
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setShowModelPicker(!showModelPicker)}
              loading={switchingModel}
              icon={!switchingModel ? <Repeat className="h-3.5 w-3.5" /> : undefined}
            >
              {t('dashboard.switchModel')}
            </Button>
          </Card>
        )}

        {showModelPicker && (
          <Card variant="elevated" className="p-4">
            <ModelPicker models={catalogModels} selected={currentModel} onSelect={handleSwitchModel} loading={switchingModel} />
          </Card>
        )}

        {/* Runtime Dependencies */}
        <DepTreePanel />

        {/* Status Overview */}
        <StatusOverview onRefresh={() => { detectTools(); loadHealthScore() }} refreshing={detecting} />

        {/* Section C: Tool Cards Grid */}
        {detecting && Object.keys(tools).length === 0 ? (
          <div className="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4">
            {TOOL_ORDER.map((name) => (
              <div key={name} className="h-28 rounded-lg border border-border bg-muted/30 animate-pulse" />
            ))}
          </div>
        ) : !detecting && !anyInstalled && Object.keys(tools).length > 0 ? (
          <Card variant="default" className="border-dashed">
            <EmptyState
              icon={Wrench}
              title={t('dashboard.noToolsTitle')}
              hint={t('dashboard.noToolsDesc')}
              action={
                <Button
                  size="lg"
                  onClick={handleInstallAll}
                  loading={anyInstalling}
                  icon={!anyInstalling ? <Download className="h-4 w-4" /> : undefined}
                >
                  {t('dashboard.installAll')}
                </Button>
              }
            />
          </Card>
        ) : (
          <div className="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4">
            {TOOL_ORDER.map((name) => {
              const tool = tools[name] || { name, installed: false, version: '', latestVersion: '', updateAvailable: false, path: '' }
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
                      if (issues?.length) useConfigStore.getState().setHighlightField(issues[0])
                      handleConfigure(name)
                    } : undefined}
                    onQuickStart={!tool.installed ? () => handleQuickStart(name) : undefined}
                    quickStarting={quickStarting[name] || false}
                  />
                  {isActiveInstall && (
                    <div className="px-4 pb-1">
                      <div className="w-full h-1 bg-muted rounded-full overflow-hidden">
                        <div className="h-full bg-primary transition-all duration-300" style={{ width: `${pct}%` }} />
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
          <Button
            size="lg"
            onClick={handleInstallAll}
            disabled={anyInstalling || detecting}
            loading={anyInstalling}
            icon={!anyInstalling ? <Download className="h-4 w-4" /> : undefined}
          >
            {t('dashboard.installAll')}
          </Button>
          {hasUpdates && (
            <Button
              size="lg"
              variant="secondary"
              onClick={handleUpdateAll}
              disabled={anyUpdating || detecting}
              loading={anyUpdating}
              icon={!anyUpdating ? <ArrowUpCircle className="h-4 w-4" /> : undefined}
              className="bg-amber-500/15 border-amber-500/40 text-amber-300 hover:bg-amber-500/25"
            >
              {t('dashboard.updateAll')}
            </Button>
          )}
        </div>

        {/* Lurus Ecosystem (moved from GYProductsPage) */}
        {products.length > 0 && (
          <div className="space-y-3">
            <h3 className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground">
              [ {t('home.ecosystem').toUpperCase()} ]
            </h3>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
              {products.map((product) => (
                <Card
                  key={product.id}
                  as="button"
                  variant="default"
                  onClick={() => LaunchGYProduct(product.id).catch(() => {})}
                  className="flex items-center gap-3 p-3 hover:bg-card/60 hover:border-rule-strong text-left"
                >
                  <span className="text-xl">{product.id === 'lurus-lucrum' ? '🔮' : product.id === 'lurus-creator' ? '🎨' : '🧠'}</span>
                  <div className="min-w-0">
                    <p className="text-sm font-medium truncate">{product.name}</p>
                    <p className="text-xs text-muted-foreground truncate">{product.description}</p>
                  </div>
                </Card>
              ))}
            </div>
          </div>
        )}

        {/* App version */}
        <div className="flex items-center justify-between text-xs text-muted-foreground border-t border-border pt-4">
          <span>{t('dashboard.version', { version: appVersion || '...' })}</span>
          <div className="flex items-center gap-2">
            {selfUpdateInfo?.updateAvailable ? (
              <button onClick={handleSelfUpdate} className="flex items-center gap-1 text-primary hover:underline">
                <ArrowUpCircle className="h-3.5 w-3.5" />
                {t('dashboard.updateTo', { version: selfUpdateInfo.latestVersion })}
              </button>
            ) : checkingUpdates ? (
              <span className="flex items-center gap-1">
                <Loader2 className="h-3 w-3 animate-spin" />
                {t('dashboard.checking')}
              </span>
            ) : (
              <button onClick={() => checkUpdates()} className="hover:text-foreground transition-colors">
                {t('dashboard.checkUpdates')}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
