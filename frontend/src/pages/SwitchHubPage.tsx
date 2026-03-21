import { useEffect, useCallback, useState, useRef } from 'react'
import {
  Power, PowerOff, RefreshCw, Loader2, Copy, Check, Plus, Trash2, RotateCw,
  Activity, BarChart3, ChevronDown, ChevronRight, DollarSign, Clock, ShieldAlert,
  Eye, EyeOff, Wifi, WifiOff, BookOpen, Zap, AlertTriangle, CheckCircle2,
  Circle, CircleDot, Link2, Unlink, ClipboardCopy, Wrench, FolderOpen, Signal, Database,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn, safeArray } from '../lib/utils'
import { errorToast } from '../lib/errorToast'
import { exportErrorLog } from '../lib/errorLog'
import { useConnectivityStore } from '../stores/connectivityStore'
import {
  useSwitchStore,
  type RegisteredApp, type GatewayLocalConfig,
  type ToolDiagnostic, type ToolConfigResult, type ToolSnapshotInfo,
  type UpstreamHealthResult, type UsageInsight, type RequestLogEntry,
} from '../stores/switchStore'
import { useToastStore } from '../stores/toastStore'
import { ConnectGuide } from '../components/switch/ConnectGuide'
import {
  GetGatewayStatus,
  GetGatewayConfig,
  SaveGatewayConfig,
  StartGateway,
  StopGateway,
  GetRegisteredApps,
  RegisterApp,
  DeleteApp,
  ResetAppToken,
  GetTodaySummary,
  GetDaySummaries,
  GetAppSummaries,
  GetModelSummaries,
  GetRecentActivity,
  RunEnvironmentCheck,
  AutoConfigureToolsForGateway,
  AutoConfigureToolForGateway,
  FullSetupForGateway,
  DisconnectToolFromGateway,
  DisconnectAllToolsFromGateway,
  ListToolSnapshots,
  RestoreToolSnapshot,
  ExportDiagnostics,
  AutoFixToolConfig,
  PingGatewayUpstream,
  OpenToolConfigDir,
  SyncToolConnectionStatus,
  InstallTool,
  InstallDependency,
  FetchModelCatalog,
  SwitchModel,
  GetProxySettings,
  GetUsageInsights,
  GetRequestLog,
} from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { gateway } from '../../wailsjs/go/models'

// --- Utility helpers ---

const TIER_KEYS: Record<number, string> = { 1: 'tierAuto', 2: 'tierGuided', 3: 'tierEnvVar', 4: 'tierManual' }

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return String(n)
}

function formatCost(usd: number): string {
  if (usd < 0.01) return `$${usd.toFixed(4)}`
  if (usd < 1) return `$${usd.toFixed(3)}`
  return `$${usd.toFixed(2)}`
}

function formatUptime(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  return `${h}h ${m}m`
}

// --- Small reusable components ---

function CopyBtn({ text }: { text: string }) {
  const [copied, setCopied] = useState(false)
  const handleCopy = () => {
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }
  return (
    <button onClick={handleCopy} className="p-1 rounded hover:bg-muted transition-colors" title="Copy">
      {copied ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5 text-muted-foreground" />}
    </button>
  )
}

function TokenDisplay({ token }: { token: string }) {
  const [visible, setVisible] = useState(false)
  const masked = token.slice(0, 12) + '••••••••'
  return (
    <div className="flex items-center gap-1">
      <code className="text-xs font-mono bg-muted px-1.5 py-0.5 rounded select-all">
        {visible ? token : masked}
      </code>
      <button onClick={() => setVisible(!visible)} className="p-0.5 rounded hover:bg-muted">
        {visible ? <EyeOff className="h-3 w-3 text-muted-foreground" /> : <Eye className="h-3 w-3 text-muted-foreground" />}
      </button>
      <CopyBtn text={token} />
    </div>
  )
}

// --- App Row ---

function AppRow({
  app,
  gatewayUrl,
  onResetToken,
  onDelete,
  resetting,
}: {
  app: RegisteredApp
  gatewayUrl: string
  onResetToken: (id: string) => void
  onDelete?: (id: string) => void
  resetting: string | null
}) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const [showGuide, setShowGuide] = useState(false)

  return (
    <>
      <div className={cn(
        'rounded-md border bg-card transition-colors',
        app.connected ? 'border-green-500/20' : 'border-border'
      )}>
        <div
          className="flex items-center justify-between px-3 py-2 cursor-pointer hover:bg-muted/30"
          onClick={() => setExpanded(!expanded)}
        >
          <div className="flex items-center gap-2">
            {app.connected ? (
              <Wifi className="h-3.5 w-3.5 text-green-500" />
            ) : (
              <WifiOff className="h-3.5 w-3.5 text-muted-foreground/40" />
            )}
            <span className="text-sm font-medium">{app.name}</span>
            <span className={cn(
              'text-[10px] px-1.5 py-0.5 rounded-full',
              app.tier === 1 ? 'bg-green-500/10 text-green-500' :
              app.tier === 2 ? 'bg-blue-500/10 text-blue-500' :
              app.tier === 3 ? 'bg-amber-500/10 text-amber-500' :
              'bg-muted text-muted-foreground'
            )}>
              {t(`switch.${TIER_KEYS[app.tier] || 'tierManual'}`)}
            </span>
          </div>
          <div className="flex items-center gap-1">
            {/* Connect guide button */}
            {gatewayUrl && (
              <button
                onClick={(e) => { e.stopPropagation(); setShowGuide(true) }}
                className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-primary transition-colors"
                title={t('switch.connectGuide')}
              >
                <BookOpen className="h-3.5 w-3.5" />
              </button>
            )}
            <ChevronRight className={cn('h-3.5 w-3.5 text-muted-foreground transition-transform', expanded && 'rotate-90')} />
          </div>
        </div>

        {expanded && (
          <div className="px-3 pb-3 space-y-2 border-t border-border/50 pt-2">
            {app.description && (
              <p className="text-xs text-muted-foreground">{app.description}</p>
            )}
            <div className="space-y-1">
              <p className="text-[10px] text-muted-foreground uppercase tracking-wider">{t('switch.apiToken')}</p>
              <TokenDisplay token={app.token} />
            </div>
            <div className="flex gap-2 mt-2">
              <button
                onClick={(e) => { e.stopPropagation(); onResetToken(app.id) }}
                disabled={resetting === app.id}
                className="flex items-center gap-1 px-2 py-1 rounded text-xs border border-border hover:bg-muted disabled:opacity-50"
              >
                {resetting === app.id ? <Loader2 className="h-3 w-3 animate-spin" /> : <RotateCw className="h-3 w-3" />}
                {t('switch.resetToken')}
              </button>
              {gatewayUrl && (
                <button
                  onClick={(e) => { e.stopPropagation(); setShowGuide(true) }}
                  className="flex items-center gap-1 px-2 py-1 rounded text-xs border border-primary/20 text-primary hover:bg-primary/10"
                >
                  <BookOpen className="h-3 w-3" />
                  {t('switch.connectGuide')}
                </button>
              )}
              {onDelete && (
                <button
                  onClick={(e) => { e.stopPropagation(); onDelete(app.id) }}
                  className="flex items-center gap-1 px-2 py-1 rounded text-xs border border-red-500/20 text-red-500 hover:bg-red-500/10"
                >
                  <Trash2 className="h-3 w-3" />
                  {t('switch.deleteConfirm')}
                </button>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Connect guide modal */}
      {showGuide && (
        <ConnectGuide
          appId={app.id}
          appName={app.name}
          token={app.token}
          gatewayUrl={gatewayUrl}
          tier={app.tier}
          onClose={() => setShowGuide(false)}
        />
      )}
    </>
  )
}

// --- Tool Diagnostic Row ---

const HEALTH_ICON: Record<string, React.ReactNode> = {
  green: <CheckCircle2 className="h-3.5 w-3.5 text-green-500" />,
  yellow: <AlertTriangle className="h-3.5 w-3.5 text-amber-500" />,
  red: <AlertTriangle className="h-3.5 w-3.5 text-red-500" />,
  unknown: <Circle className="h-3.5 w-3.5 text-muted-foreground/40" />,
}

function ToolDiagRow({
  diag, onConnect, onDisconnect, onInstall, onShowSnapshots, onAutoFix, onOpenConfig, connecting, disconnecting, installing, fixing, installProgress,
}: {
  diag: ToolDiagnostic
  onConnect: (tool: string) => void
  onDisconnect: (tool: string) => void
  onInstall: (tool: string) => void
  onShowSnapshots: (tool: string) => void
  onAutoFix: (tool: string) => void
  onOpenConfig: (tool: string) => void
  connecting: string | null
  disconnecting: string | null
  installing: string | null
  fixing: string | null
  installProgress: Record<string, number>
}) {
  const { t } = useTranslation()

  return (
    <div className={cn(
      'rounded-md border py-2.5 px-3',
      diag.installed ? 'bg-card border-border' : 'bg-muted/30 border-border/50'
    )}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3 min-w-0">
          {diag.installed ? (
            <CircleDot className="h-3.5 w-3.5 text-green-500 flex-shrink-0" />
          ) : (
            <Circle className="h-3.5 w-3.5 text-muted-foreground/30 flex-shrink-0" />
          )}
          <div className="min-w-0">
            <p className="text-sm font-medium truncate">{diag.tool}</p>
            {diag.installed && (
              <p className="text-[10px] text-muted-foreground truncate">
                v{diag.version || '?'}
              </p>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2 flex-shrink-0">
          {/* Config health + auto-fix */}
          {diag.installed && (
            <span className="flex items-center gap-1" title={diag.healthIssues?.join(', ') || t('switch.configOk')}>
              {HEALTH_ICON[diag.healthStatus] || HEALTH_ICON.unknown}
              {(diag.healthStatus === 'red' || diag.healthStatus === 'yellow') && (
                <button
                  onClick={() => onAutoFix(diag.tool)}
                  disabled={fixing === diag.tool}
                  className="p-0.5 rounded hover:bg-amber-500/10 text-amber-500 disabled:opacity-50"
                  title={t('switch.autoFix')}
                >
                  {fixing === diag.tool ? <Loader2 className="h-3 w-3 animate-spin" /> : <Wrench className="h-3 w-3" />}
                </button>
              )}
            </span>
          )}

          {/* Gateway binding status + disconnect */}
          {diag.installed && diag.gatewayBound && (
            <>
              <span className="flex items-center gap-1 text-[10px] text-green-600 dark:text-green-400">
                <Link2 className="h-3 w-3" />
                {t('switch.bound')}
              </span>
              <button
                onClick={() => onDisconnect(diag.tool)}
                disabled={disconnecting === diag.tool}
                className="flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-medium border border-red-500/20 text-red-500 hover:bg-red-500/10 disabled:opacity-50"
                title={t('switch.disconnectDesc')}
              >
                {disconnecting === diag.tool ? (
                  <Loader2 className="h-3 w-3 animate-spin" />
                ) : (
                  <Unlink className="h-3 w-3" />
                )}
                {t('switch.disconnect')}
              </button>
            </>
          )}

          {/* Connect button for installed but unbound tools */}
          {diag.installed && !diag.gatewayBound && (
            <button
              onClick={() => onConnect(diag.tool)}
              disabled={connecting === diag.tool}
              className="flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-medium border border-primary/30 text-primary hover:bg-primary/10 disabled:opacity-50"
            >
              {connecting === diag.tool ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                <Link2 className="h-3 w-3" />
              )}
              {t('switch.connectTool')}
            </button>
          )}

          {/* Snapshot + Open Config buttons for installed tools */}
          {diag.installed && (
            <>
              <button
                onClick={() => onOpenConfig(diag.tool)}
                className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-primary transition-colors"
                title={t('switch.openConfig')}
              >
                <FolderOpen className="h-3 w-3" />
              </button>
              <button
                onClick={() => onShowSnapshots(diag.tool)}
                className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-primary transition-colors"
                title={t('switch.snapshots')}
              >
                <RotateCw className="h-3 w-3" />
              </button>
            </>
          )}

          {/* Install button for not-installed tools */}
          {!diag.installed && (
            <button
              onClick={() => onInstall(diag.tool)}
              disabled={installing === diag.tool}
              className="flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-medium border border-border hover:bg-muted disabled:opacity-50"
            >
              {installing === diag.tool ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                <Plus className="h-3 w-3" />
              )}
              {installing === diag.tool && installProgress[diag.tool] > 0
                ? `${installProgress[diag.tool]}%`
                : t('switch.install')
              }
            </button>
          )}
        </div>
      </div>

      {/* Current config details for installed tools */}
      {diag.installed && diag.currentEndpoint && (
        <div className="mt-1.5 ml-[26px] flex items-center gap-3 text-[10px] text-muted-foreground">
          <span className="truncate max-w-[200px]" title={diag.currentEndpoint}>
            {diag.currentEndpoint}
          </span>
          {diag.currentModel && (
            <>
              <span className="text-border">|</span>
              <span className="font-mono">{diag.currentModel}</span>
            </>
          )}
        </div>
      )}
    </div>
  )
}

// --- Main Page ---

export function SwitchHubPage() {
  const { t } = useTranslation()
  const {
    status, config, loading, starting, stopping,
    apps,
    todaySummary, daySummaries, appSummaries, modelSummaries, recentActivity,
    meteringPeriod,
    envCheck, envLoading, configResults, configuring,
    setStatus, setConfig, setLoading, setStarting, setStopping,
    setApps,
    setTodaySummary, setDaySummaries, setAppSummaries, setModelSummaries, setRecentActivity,
    setMeteringPeriod,
    setPollHandle,
    setEnvCheck, setEnvLoading, setConfigResults, setConfiguring,
  } = useSwitchStore()
  const toast = useToastStore((s) => s.addToast)
  const recordConnSuccess = useConnectivityStore((s) => s.recordSuccess)

  const [showRegister, setShowRegister] = useState(false)
  const [newAppName, setNewAppName] = useState('')
  const [newAppDesc, setNewAppDesc] = useState('')
  const [registering, setRegistering] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)
  const [resetting, setResetting] = useState<string | null>(null)
  const [showApps, setShowApps] = useState(true)
  const [showUsage, setShowUsage] = useState(true)
  const [connectingSingle, setConnectingSingle] = useState<string | null>(null)
  const [installingSingle, setInstallingSingle] = useState<string | null>(null)
  const [settingUp, setSettingUp] = useState(false)
  const [currentModel, setCurrentModel] = useState('')
  const [modelCatalog, setModelCatalog] = useState<{ id: string; name: string }[]>([])
  const [switchingModel, setSwitchingModel] = useState(false)
  const [disconnectingSingle, setDisconnectingSingle] = useState<string | null>(null)
  const [disconnectingAll, setDisconnectingAll] = useState(false)
  const [snapshotTool, setSnapshotTool] = useState<string | null>(null)
  const [snapshots, setSnapshots] = useState<ToolSnapshotInfo[]>([])
  const [restoringSnapshot, setRestoringSnapshot] = useState<string | null>(null)
  const [fixingTool, setFixingTool] = useState<string | null>(null)
  const [upstreamHealth, setUpstreamHealth] = useState<UpstreamHealthResult | null>(null)
  const [installProgress, setInstallProgress] = useState<Record<string, number>>({})
  const [usageInsights, setUsageInsights] = useState<UsageInsight | null>(null)
  const [installingRuntime, setInstallingRuntime] = useState<string | null>(null)
  const [requestLog, setRequestLog] = useState<RequestLogEntry[]>([])
  const [logFilterApp, setLogFilterApp] = useState('')
  const [logFilterModel, setLogFilterModel] = useState('')
  const [backendStale, setBackendStale] = useState(false)
  const envRefreshCount = useRef(0)
  const pollFailCount = useRef(0)

  const loadAll = useCallback(async () => {
    setLoading(true)
    try {
      const [st, cfg, appsList, summary, daySums, activity] = await Promise.all([
        GetGatewayStatus(),
        GetGatewayConfig(),
        GetRegisteredApps(),
        GetTodaySummary(),
        GetDaySummaries(7),
        GetRecentActivity(20),
      ])
      setStatus(st)
      setConfig(cfg)
      setApps(safeArray(appsList))
      setTodaySummary(summary)
      setDaySummaries(safeArray(daySums))
      setRecentActivity(safeArray(activity))

      const [appSums, modelSums] = await Promise.all([
        GetAppSummaries(meteringPeriod),
        GetModelSummaries(meteringPeriod),
      ])
      setAppSummaries(safeArray(appSums))
      setModelSummaries(safeArray(modelSums))

      setBackendStale(false)
      pollFailCount.current = 0

      // Run environment check in the background.
      RunEnvironmentCheck().then(setEnvCheck).catch(() => {})

      // Load model catalog and current model (non-critical, silent on failure).
      FetchModelCatalog().then((cat) => {
        if (cat?.models) setModelCatalog(cat.models.map((m: any) => ({ id: m.id, name: m.name || m.id })))
      }).catch(() => {})
      GetProxySettings().then((s) => {
        if (s?.model) setCurrentModel(s.model)
      }).catch(() => {})
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway', t })
    } finally {
      setLoading(false)
    }
  }, [meteringPeriod])

  const refreshStatus = useCallback(async () => {
    try {
      const [st, summary, activity] = await Promise.all([
        GetGatewayStatus(),
        GetTodaySummary(),
        GetRecentActivity(20),
      ])
      setStatus(st)
      setTodaySummary(summary)
      setRecentActivity(safeArray(activity))
      // Reset failure tracking on success.
      if (pollFailCount.current > 0) {
        pollFailCount.current = 0
        setBackendStale(false)
      }
      recordConnSuccess()
    } catch {
      // Track consecutive poll failures — show stale indicator after 3 in a row.
      pollFailCount.current++
      if (pollFailCount.current >= 3 && !backendStale) {
        setBackendStale(true)
      }
    }
  }, [backendStale])

  useEffect(() => {
    loadAll()

    // Upstream health + request log on load.
    PingGatewayUpstream().then(setUpstreamHealth).catch(() => {})
    GetRequestLog(50, '', '').then(r => setRequestLog(safeArray(r))).catch(() => {})

    // Clear any stale interval from previous mount (React StrictMode double-mount).
    const prevHandle = useSwitchStore.getState().pollHandle
    if (prevHandle !== null) clearInterval(prevHandle)

    const h = setInterval(() => {
      refreshStatus()
      // Refresh env check every 3rd poll cycle (~15s).
      envRefreshCount.current++
      if (envRefreshCount.current % 3 === 0) {
        RunEnvironmentCheck().then(setEnvCheck).catch(() => {})
        SyncToolConnectionStatus().catch(() => {})
      }
      // Refresh request log every 2nd cycle (~10s).
      if (envRefreshCount.current % 2 === 0) {
        GetRequestLog(50, '', '').then(r => setRequestLog(safeArray(r))).catch(() => {})
      }
      // Refresh upstream health every 6th cycle (~30s).
      if (envRefreshCount.current % 6 === 0) {
        PingGatewayUpstream().then(setUpstreamHealth).catch(() => {})
      }
    }, 5000)
    setPollHandle(h)

    // Subscribe to install progress events from backend.
    const offProgress = EventsOn('tool:install:progress', (d: { tool: string; percent: number }) => {
      setInstallProgress(p => ({ ...p, [d.tool]: d.percent }))
    })
    const offDone = EventsOn('tool:install:done', (d: { tool: string; success: boolean }) => {
      setInstallProgress(p => ({ ...p, [d.tool]: d.success ? 100 : -1 }))
      // Refresh env check after install completes.
      RunEnvironmentCheck().then(setEnvCheck).catch(() => {})
    })

    // Subscribe to gateway crash/restart events.
    const offCrash = EventsOn('gateway:crash', (d: { attempt: number; error: string }) => {
      toast('error', t('switch.gatewayCrashRestart', { attempt: d.attempt }), { persistent: true })
      refreshStatus()
    })

    return () => {
      clearInterval(h)
      setPollHandle(null)
      offProgress()
      offDone()
      offCrash()
    }
  }, [])

  useEffect(() => {
    Promise.all([
      GetAppSummaries(meteringPeriod),
      GetModelSummaries(meteringPeriod),
      GetUsageInsights(meteringPeriod),
    ]).then(([a, m, ins]) => {
      setAppSummaries(safeArray(a))
      setModelSummaries(safeArray(m))
      setUsageInsights(ins ?? null)
    }).catch(() => {})
  }, [meteringPeriod])

  // --- Handlers ---

  const handleStart = async () => {
    setStarting(true)
    try {
      await StartGateway()
      toast('success', t('switch.startSuccess'))
      await refreshStatus()
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway', t })
    } finally { setStarting(false) }
  }

  const handleStop = async () => {
    setStopping(true)
    try {
      await StopGateway()
      toast('success', t('switch.stopSuccess'))
      await refreshStatus()
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway', t })
    } finally { setStopping(false) }
  }

  const handleRegister = async () => {
    if (!newAppName.trim()) return
    setRegistering(true)
    try {
      const app = await RegisterApp(newAppName.trim(), '', newAppDesc.trim())
      toast('success', t('switch.registerSuccess', { name: app.name }))
      setNewAppName('')
      setNewAppDesc('')
      setShowRegister(false)
      setApps(await GetRegisteredApps() || [])
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway', t })
    } finally { setRegistering(false) }
  }

  const handleDelete = async (id: string) => {
    try {
      await DeleteApp(id)
      toast('success', t('switch.deleteSuccess'))
      setConfirmDelete(null)
      setApps(await GetRegisteredApps() || [])
    } catch (err) { errorToast(toast, err, { currentPage: 'gateway', t }) }
  }

  const handleResetToken = async (id: string) => {
    setResetting(id)
    try {
      await ResetAppToken(id)
      toast('success', t('switch.tokenReset'))
      setApps(await GetRegisteredApps() || [])
    } catch (err) { errorToast(toast, err, { currentPage: 'gateway', t }) }
    finally { setResetting(null) }
  }

  // --- One-click connect handlers ---

  const handleConnectAll = async () => {
    setConfiguring(true)
    try {
      const results = safeArray(await AutoConfigureToolsForGateway())
      setConfigResults(results)
      const successes = results.filter(r => r.success).length
      const failures = results.filter(r => !r.success).length
      if (failures === 0 && successes > 0) {
        toast('success', t('switch.connectAllSuccess', { count: successes }))
      } else if (successes > 0) {
        toast('info', t('switch.connectPartial', { ok: successes, fail: failures }))
      } else {
        toast('error', t('switch.connectAllFailed'), { persistent: true })
      }
      // Refresh env check + apps after configuring.
      RunEnvironmentCheck().then(setEnvCheck).catch(() => {})
      GetRegisteredApps().then(a => setApps(safeArray(a))).catch(() => {})
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway', t })
    } finally {
      setConfiguring(false)
    }
  }

  const handleConnectSingle = async (tool: string) => {
    setConnectingSingle(tool)
    try {
      const result = await AutoConfigureToolForGateway(tool)
      if (result?.success) {
        toast('success', `${tool}: ${result.message}`)
      } else {
        errorToast(toast, result?.message || 'Configuration failed', { currentPage: 'gateway', t })
      }
      RunEnvironmentCheck().then(setEnvCheck).catch(() => {})
      GetRegisteredApps().then(a => setApps(safeArray(a))).catch(() => {})
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway', t })
    } finally {
      setConnectingSingle(null)
    }
  }

  const handleInstallSingle = async (tool: string) => {
    setInstallingSingle(tool)
    try {
      const result = await InstallTool(tool)
      if (result?.success) {
        toast('success', `${tool} v${result.version || '?'} ${t('switch.installSuccess')}`)
      } else {
        errorToast(toast, result?.message || 'Installation failed', { currentPage: 'gateway', t })
      }
      RunEnvironmentCheck().then(setEnvCheck).catch(() => {})
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway', t })
    } finally {
      setInstallingSingle(null)
    }
  }

  const handleFullSetup = async () => {
    setSettingUp(true)
    try {
      const result = await FullSetupForGateway()
      if (!result) { errorToast(toast, 'Setup returned empty result', { currentPage: 'gateway', t }); return }
      const cfgResults = safeArray(result.configResults)
      const successes = cfgResults.filter(r => r.success).length
      setConfigResults(cfgResults)

      let msg = ''
      if (result.gatewayStarted) msg += t('switch.gwAutoStarted') + ' '
      if (result.snapshotsTaken > 0) msg += t('switch.snapshotsTaken', { count: result.snapshotsTaken }) + ' '
      msg += t('switch.connectAllSuccess', { count: successes })

      const errors = safeArray(result.errors)
      if (errors.length > 0) {
        toast('info', msg + ` (${errors.join('; ')})`)
      } else {
        toast('success', msg)
      }

      // Refresh everything.
      await refreshStatus()
      RunEnvironmentCheck().then(setEnvCheck).catch(() => {})
      GetRegisteredApps().then(a => setApps(safeArray(a))).catch(() => {})
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway', t })
    } finally {
      setSettingUp(false)
    }
  }

  const handleSwitchModel = async (modelId: string) => {
    setCurrentModel(modelId)
    setSwitchingModel(true)
    try {
      const errs = await SwitchModel(modelId)
      const errKeys = Object.keys(errs).filter(k => k !== 'error')
      if (errs['error']) {
        errorToast(toast, errs['error'], { currentPage: 'gateway', t })
      } else if (errKeys.length === 0) {
        toast('success', t('switch.modelSwitched', { model: modelId }))
      } else {
        toast('info', `${t('switch.modelSwitched', { model: modelId })} (${errKeys.length} ${t('switch.toolErrors')})`)
      }
      RunEnvironmentCheck().then(setEnvCheck).catch(() => {})
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway', t })
    } finally {
      setSwitchingModel(false)
    }
  }

  const handleRefreshEnv = async () => {
    setEnvLoading(true)
    try {
      const check = await RunEnvironmentCheck()
      setEnvCheck(check)
    } catch { /* silent */ }
    finally { setEnvLoading(false) }
  }

  const handleDisconnectSingle = async (tool: string) => {
    setDisconnectingSingle(tool)
    try {
      const result = await DisconnectToolFromGateway(tool)
      if (result?.success) {
        toast('success', t('switch.disconnectSuccess', { tool }))
      } else {
        errorToast(toast, result?.message || 'Disconnect failed', { currentPage: 'gateway', t })
      }
      RunEnvironmentCheck().then(setEnvCheck).catch(() => {})
      GetRegisteredApps().then(a => setApps(safeArray(a))).catch(() => {})
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway', t })
    } finally {
      setDisconnectingSingle(null)
    }
  }

  const handleDisconnectAll = async () => {
    setDisconnectingAll(true)
    try {
      const results = safeArray(await DisconnectAllToolsFromGateway())
      const successes = results.filter(r => r.success).length
      if (successes > 0) {
        toast('success', t('switch.disconnectAllSuccess', { count: successes }))
      } else {
        toast('info', t('switch.disconnectAllFailed'))
      }
      RunEnvironmentCheck().then(setEnvCheck).catch(() => {})
      GetRegisteredApps().then(a => setApps(safeArray(a))).catch(() => {})
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway', t })
    } finally {
      setDisconnectingAll(false)
    }
  }

  const handleShowSnapshots = async (tool: string) => {
    setSnapshotTool(tool)
    try {
      const list = await ListToolSnapshots(tool)
      setSnapshots(safeArray(list))
    } catch {
      setSnapshots([])
    }
  }

  const handleRestoreSnapshot = async (tool: string, snapshotId: string) => {
    setRestoringSnapshot(snapshotId)
    try {
      const result = await RestoreToolSnapshot(tool, snapshotId)
      if (result?.success) {
        toast('success', t('switch.snapshotRestored', { tool }))
        setSnapshotTool(null)
      } else {
        errorToast(toast, result?.message || 'Restore failed', { currentPage: 'gateway', t })
      }
      RunEnvironmentCheck().then(setEnvCheck).catch(() => {})
      GetRegisteredApps().then(a => setApps(safeArray(a))).catch(() => {})
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway', t })
    } finally {
      setRestoringSnapshot(null)
    }
  }

  const handleOpenConfig = (tool: string) => {
    OpenToolConfigDir(tool).catch((err) => errorToast(toast, err, { currentPage: 'gateway', t }))
  }

  const handleAutoFix = async (tool: string) => {
    setFixingTool(tool)
    try {
      const result = await AutoFixToolConfig(tool)
      if (result?.success) {
        toast('success', `${tool}: ${result.message}`)
      } else {
        errorToast(toast, result?.message || 'Auto-fix failed', { currentPage: 'gateway', t })
      }
      RunEnvironmentCheck().then(setEnvCheck).catch(() => {})
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway', t })
    } finally {
      setFixingTool(null)
    }
  }

  const handleInstallRuntime = async (runtimeId: string) => {
    setInstallingRuntime(runtimeId)
    try {
      const result = await InstallDependency(runtimeId)
      if (result?.success) {
        toast('success', `${runtimeId}: ${result.message || 'installed'}`)
      } else {
        errorToast(toast, result?.message || 'Installation failed', { currentPage: 'gateway', t })
      }
      RunEnvironmentCheck().then(setEnvCheck).catch(() => {})
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway', t })
    } finally {
      setInstallingRuntime(null)
    }
  }

  const handleExportDiagnostics = async () => {
    try {
      const report = await ExportDiagnostics()
      const errorHistory = exportErrorLog()
      const fullReport = report + '\n\n--- Recent Errors ---\n' + errorHistory
      await navigator.clipboard.writeText(fullReport)
      toast('success', t('switch.diagnosticsCopied'))
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway', t })
    }
  }

  const handleSaveConfig = async (cfg: GatewayLocalConfig) => {
    try {
      await SaveGatewayConfig(gateway.Config.createFrom(cfg))
      setConfig(cfg)
      toast('success', t('switch.configSaved'))
    } catch (err) { errorToast(toast, err, { currentPage: 'gateway', t }) }
  }

  const running = status?.running ?? false
  const gwUrl = running && status?.url ? status.url : ''
  const connectedCount = apps.filter(a => a.connected).length
  const builtinApps = apps.filter(a => a.kind === 'builtin')
  const userApps = apps.filter(a => a.kind === 'user')

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-4xl mx-auto p-6 space-y-6">
        {/* ── Header ── */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold">{t('switch.title')}</h2>
            <p className="text-sm text-muted-foreground">{t('switch.subtitle')}</p>
          </div>
          <button
            onClick={loadAll}
            disabled={loading}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-colors',
              'border border-border hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
            {t('switch.refresh')}
          </button>
        </div>

        {/* ── Stale data warning ── */}
        {backendStale && (
          <div className="flex items-center justify-between rounded-md border border-amber-500/30 bg-amber-500/5 px-4 py-2.5">
            <div className="flex items-center gap-2 text-sm text-amber-700 dark:text-amber-400">
              <WifiOff className="h-4 w-4 flex-shrink-0" />
              <span>{t('switch.backendStale')}</span>
            </div>
            <button
              onClick={() => { pollFailCount.current = 0; setBackendStale(false); loadAll() }}
              className="px-2.5 py-1 rounded text-xs font-medium border border-amber-500/30 text-amber-600 hover:bg-amber-500/10"
            >
              {t('switch.backendRetry')}
            </button>
          </div>
        )}

        {/* ── Gateway Status Card ── */}
        <div className={cn(
          'rounded-lg border p-5',
          running ? 'border-green-500/30 bg-green-500/5' : 'border-border bg-card'
        )}>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className={cn(
                'h-3 w-3 rounded-full',
                running ? 'bg-green-500 animate-pulse' : 'bg-muted-foreground/30'
              )} />
              <div>
                <p className="text-sm font-medium">
                  {running ? t('switch.running') : t('switch.stopped')}
                </p>
                {running && status && (
                  <p className="text-xs text-muted-foreground">
                    {status.url} — {t('switch.uptime')} {formatUptime(status.uptime)} — {status.totalRequests} {t('switch.requests')}
                  </p>
                )}
              </div>
            </div>
            {running ? (
              <button
                onClick={handleStop}
                disabled={stopping}
                className={cn(
                  'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium',
                  'bg-red-500/10 text-red-500 border border-red-500/20 hover:bg-red-500/20 disabled:opacity-50'
                )}
              >
                {stopping ? <Loader2 className="h-4 w-4 animate-spin" /> : <PowerOff className="h-4 w-4" />}
                {t('switch.stop')}
              </button>
            ) : (
              <button
                onClick={handleStart}
                disabled={starting}
                className={cn(
                  'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium',
                  'bg-green-500 text-white hover:bg-green-600 disabled:opacity-50'
                )}
              >
                {starting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Power className="h-4 w-4" />}
                {t('switch.start')}
              </button>
            )}
          </div>

          {/* Stats row */}
          {running && todaySummary && (
            <div className="mt-4 grid grid-cols-4 gap-3">
              {[
                { label: t('switch.callsToday'), value: todaySummary.totalCalls },
                { label: t('switch.tokensIn'), value: formatTokens(todaySummary.tokensIn) },
                { label: t('switch.tokensOut'), value: formatTokens(todaySummary.tokensOut) },
                { label: t('switch.connected'), value: connectedCount },
              ].map(({ label, value }) => (
                <div key={label} className="rounded-md bg-background/50 border border-border/50 p-3">
                  <p className="text-[10px] uppercase tracking-wider text-muted-foreground">{label}</p>
                  <p className="text-lg font-semibold">{value}</p>
                </div>
              ))}
            </div>
          )}

          {/* Upstream unreachable warning */}
          {running && upstreamHealth && !upstreamHealth.reachable && (
            <div className="mt-3 flex items-start gap-2.5 rounded-md border border-red-500/30 bg-red-500/5 p-3">
              <WifiOff className="h-4 w-4 text-red-500 flex-shrink-0 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-red-600 dark:text-red-400">{t('switch.upstreamWarning')}</p>
                <p className="text-xs text-red-500/70 mt-0.5">{t('switch.upstreamWarningHint')}</p>
                {upstreamHealth.error && (
                  <p className="text-[10px] font-mono text-red-500/50 mt-1 truncate">{upstreamHealth.error}</p>
                )}
              </div>
            </div>
          )}

          {/* Endpoint copy helper */}
          {running && gwUrl && (
            <div className="mt-3 flex items-center gap-2 text-xs text-muted-foreground">
              <span>{t('switch.endpoint')}:</span>
              <code className="bg-muted px-2 py-0.5 rounded font-mono select-all">{gwUrl}/v1</code>
              <CopyBtn text={`${gwUrl}/v1`} />
            </div>
          )}
        </div>

        {/* ── Environment & Connect ── */}
        {envCheck && (
          <div className="rounded-lg border border-border bg-card p-5 space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Zap className="h-4 w-4 text-primary" />
                <h3 className="text-sm font-semibold">{t('switch.envTitle')}</h3>
              </div>
              <div className="flex items-center gap-1">
                <button
                  onClick={handleExportDiagnostics}
                  className="p-1.5 rounded hover:bg-muted text-muted-foreground"
                  title={t('switch.exportDiagnostics')}
                >
                  <ClipboardCopy className="h-3.5 w-3.5" />
                </button>
                <button
                  onClick={handleRefreshEnv}
                  disabled={envLoading}
                  className="p-1.5 rounded hover:bg-muted text-muted-foreground disabled:opacity-50"
                  title={t('switch.refresh')}
                >
                  {envLoading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
                </button>
              </div>
            </div>

            {/* Status summary */}
            <div className="flex items-center gap-3 text-xs text-muted-foreground flex-wrap">
              <span>{envCheck.installedCount} {t('switch.toolsInstalled')}</span>
              <span className="text-border">|</span>
              <span>{envCheck.boundCount}/{envCheck.installedCount} {t('switch.toolsBound')}</span>
              {envCheck.gatewayRunning && (
                <>
                  <span className="text-border">|</span>
                  <span className="text-green-600 dark:text-green-400">{t('switch.gwOnline')}</span>
                </>
              )}
              {/* Upstream health indicator */}
              {upstreamHealth && (
                <>
                  <span className="text-border">|</span>
                  <span className="flex items-center gap-1">
                    <Signal className={cn('h-3 w-3', upstreamHealth.reachable ? 'text-green-500' : 'text-red-500')} />
                    {upstreamHealth.reachable ? (
                      <span className="text-green-600 dark:text-green-400">
                        {t('switch.upstreamOk')} ({upstreamHealth.latencyMs}ms)
                      </span>
                    ) : (
                      <span className="text-red-500">
                        {t('switch.upstreamDown')}
                      </span>
                    )}
                  </span>
                </>
              )}
            </div>

            {/* Full Setup or Connect All */}
            {!envCheck.allToolsBound && envCheck.tools.some(t => t.installed) && (
              <div className={cn(
                'flex items-center justify-between rounded-md p-3 border',
                'bg-primary/5 border-primary/20'
              )}>
                <div className="min-w-0">
                  <p className="text-sm font-medium">{t('switch.connectBannerTitle')}</p>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {t('switch.connectBannerDesc')}
                  </p>
                </div>
                <div className="flex items-center gap-2 flex-shrink-0 ml-3">
                  {!envCheck.gatewayRunning ? (
                    <button
                      onClick={handleFullSetup}
                      disabled={settingUp}
                      className={cn(
                        'flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium',
                        'bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50'
                      )}
                    >
                      {settingUp ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />}
                      {t('switch.fullSetup')}
                    </button>
                  ) : (
                    <button
                      onClick={handleConnectAll}
                      disabled={configuring}
                      className={cn(
                        'flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium',
                        'bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50'
                      )}
                    >
                      {configuring ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />}
                      {t('switch.connectAll')}
                    </button>
                  )}
                </div>
              </div>
            )}

            {/* All connected badge + disconnect all */}
            {envCheck.allToolsBound && envCheck.tools.some(t => t.installed) && (
              <div className="flex items-center justify-between rounded-md p-3 border border-green-500/20 bg-green-500/5">
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <p className="text-sm text-green-700 dark:text-green-400">{t('switch.allBound')}</p>
                </div>
                <button
                  onClick={handleDisconnectAll}
                  disabled={disconnectingAll}
                  className="flex items-center gap-1 px-2.5 py-1 rounded text-xs font-medium border border-red-500/20 text-red-500 hover:bg-red-500/10 disabled:opacity-50"
                >
                  {disconnectingAll ? <Loader2 className="h-3 w-3 animate-spin" /> : <Unlink className="h-3.5 w-3.5" />}
                  {t('switch.disconnectAll')}
                </button>
              </div>
            )}

            {/* Model selector */}
            {modelCatalog.length > 0 && (
              <div className="flex items-center gap-3">
                <label className="text-xs text-muted-foreground whitespace-nowrap">{t('switch.globalModel')}:</label>
                <select
                  value={currentModel}
                  onChange={(e) => handleSwitchModel(e.target.value)}
                  disabled={switchingModel}
                  className="flex-1 px-2 py-1 rounded-md border border-border bg-background text-xs font-mono disabled:opacity-50"
                >
                  <option value="">{t('switch.modelDefault')}</option>
                  {modelCatalog.map((m) => (
                    <option key={m.id} value={m.id}>{m.name}</option>
                  ))}
                </select>
                {switchingModel && <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" />}
              </div>
            )}

            {/* Tool list */}
            <div className="space-y-1">
              {envCheck.tools.map((diag) => (
                <ToolDiagRow
                  key={diag.tool}
                  diag={diag}
                  onConnect={handleConnectSingle}
                  onDisconnect={handleDisconnectSingle}
                  onInstall={handleInstallSingle}
                  onShowSnapshots={handleShowSnapshots}
                  onAutoFix={handleAutoFix}
                  onOpenConfig={handleOpenConfig}
                  connecting={connectingSingle}
                  disconnecting={disconnectingSingle}
                  installing={installingSingle}
                  fixing={fixingTool}
                  installProgress={installProgress}
                />
              ))}
            </div>

            {/* Runtime deps */}
            {envCheck.runtimes && envCheck.runtimes.length > 0 && (
              <div className="border-t border-border pt-3">
                <p className="text-[10px] text-muted-foreground uppercase tracking-wider mb-2">{t('switch.runtimes')}</p>
                <div className="flex flex-wrap gap-2">
                  {envCheck.runtimes.map((rt) => (
                    <div
                      key={rt.id}
                      className={cn(
                        'inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs border',
                        rt.installed
                          ? 'border-green-500/20 bg-green-500/5 text-green-700 dark:text-green-400'
                          : rt.required
                          ? 'border-red-500/20 bg-red-500/5 text-red-600'
                          : 'border-border bg-muted text-muted-foreground'
                      )}
                    >
                      {rt.installed ? <CheckCircle2 className="h-3 w-3" /> : <AlertTriangle className="h-3 w-3" />}
                      <span>{rt.name} {rt.version ? `v${rt.version}` : ''}</span>
                      {!rt.installed && rt.required && rt.id !== 'none' && (
                        <button
                          onClick={() => handleInstallRuntime(rt.id)}
                          disabled={installingRuntime === rt.id}
                          className="ml-0.5 px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-primary/10 text-primary hover:bg-primary/20 disabled:opacity-50"
                        >
                          {installingRuntime === rt.id ? (
                            <Loader2 className="h-3 w-3 animate-spin inline" />
                          ) : (
                            t('switch.installRuntime')
                          )}
                        </button>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Config results feedback */}
            {configResults.length > 0 && (
              <div className="border-t border-border pt-3">
                <p className="text-[10px] text-muted-foreground uppercase tracking-wider mb-2">{t('switch.lastConfigResult')}</p>
                <div className="space-y-1">
                  {configResults.map((r, i) => (
                    <div key={i} className="flex items-center gap-2 text-xs">
                      {r.success ? (
                        <Check className="h-3 w-3 text-green-500" />
                      ) : (
                        <AlertTriangle className="h-3 w-3 text-red-500" />
                      )}
                      <span className="font-medium">{r.tool}</span>
                      <span className="text-muted-foreground truncate">{r.message}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {/* ── Gateway Settings ── */}
        <details className="group">
          <summary className="cursor-pointer text-sm font-medium text-muted-foreground hover:text-foreground transition-colors list-none flex items-center gap-2">
            <span className="text-xs transition-transform group-open:rotate-90">&#9654;</span>
            {t('switch.settings')}
          </summary>
          {config && (
            <div className="mt-3 rounded-lg border border-border bg-card p-4 space-y-3">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-xs text-muted-foreground block mb-1">{t('switch.port')}</label>
                  <input
                    type="number"
                    value={config.port}
                    onChange={(e) => setConfig({ ...config, port: parseInt(e.target.value) || 19090 })}
                    className="w-full px-3 py-1.5 rounded-md border border-border bg-background text-sm"
                  />
                </div>
                <div className="flex items-end">
                  <label className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={config.autoStart}
                      onChange={(e) => setConfig({ ...config, autoStart: e.target.checked })}
                      className="rounded"
                    />
                    {t('switch.autoStart')}
                  </label>
                </div>
              </div>
              <button
                onClick={() => handleSaveConfig(config)}
                className="px-4 py-1.5 rounded-md text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                {t('switch.save')}
              </button>
            </div>
          )}
        </details>

        {/* ── Connected Apps ── */}
        <div>
          <button
            onClick={() => setShowApps(!showApps)}
            className="flex items-center gap-2 text-sm font-semibold mb-3 hover:text-primary transition-colors"
          >
            {showApps ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            <Wifi className="h-4 w-4" />
            {t('switch.connectedApps')} ({apps.length})
          </button>

          {showApps && (
            <div className="space-y-2">
              {/* Register button */}
              {!showRegister ? (
                <button
                  onClick={() => setShowRegister(true)}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm border border-dashed border-border hover:bg-muted transition-colors w-full justify-center text-muted-foreground"
                >
                  <Plus className="h-4 w-4" />
                  {t('switch.registerApp')}
                </button>
              ) : (
                <div className="rounded-lg border border-border bg-card p-4 space-y-3">
                  <div className="grid grid-cols-2 gap-3">
                    <input
                      value={newAppName}
                      onChange={(e) => setNewAppName(e.target.value)}
                      placeholder={t('switch.appName')}
                      className="px-3 py-1.5 rounded-md border border-border bg-background text-sm"
                      onKeyDown={(e) => e.key === 'Enter' && handleRegister()}
                    />
                    <input
                      value={newAppDesc}
                      onChange={(e) => setNewAppDesc(e.target.value)}
                      placeholder={t('switch.appDesc')}
                      className="px-3 py-1.5 rounded-md border border-border bg-background text-sm"
                      onKeyDown={(e) => e.key === 'Enter' && handleRegister()}
                    />
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={handleRegister}
                      disabled={registering || !newAppName.trim()}
                      className={cn(
                        'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium',
                        'bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50'
                      )}
                    >
                      {registering ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                      {t('switch.register')}
                    </button>
                    <button
                      onClick={() => setShowRegister(false)}
                      className="px-3 py-1.5 rounded-md text-sm border border-border hover:bg-muted"
                    >
                      {t('switch.cancel')}
                    </button>
                  </div>
                </div>
              )}

              {/* Builtin tools */}
              {builtinApps.length > 0 && (
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider px-1">{t('switch.aiTools')}</p>
                  <div className="grid grid-cols-1 gap-1">
                    {builtinApps.map((app) => (
                      <AppRow key={app.id} app={app} gatewayUrl={gwUrl} onResetToken={handleResetToken} resetting={resetting} />
                    ))}
                  </div>
                </div>
              )}

              {/* User apps */}
              {userApps.length > 0 && (
                <div className="space-y-1 mt-3">
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider px-1">{t('switch.customApps')}</p>
                  <div className="grid grid-cols-1 gap-1">
                    {userApps.map((app) => (
                      <AppRow
                        key={app.id}
                        app={app}
                        gatewayUrl={gwUrl}
                        onResetToken={handleResetToken}
                        onDelete={(id) => setConfirmDelete(id)}
                        resetting={resetting}
                      />
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* ── Usage Analytics ── */}
        <div>
          <button
            onClick={() => setShowUsage(!showUsage)}
            className="flex items-center gap-2 text-sm font-semibold mb-3 hover:text-primary transition-colors"
          >
            {showUsage ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            <BarChart3 className="h-4 w-4" />
            {t('switch.usage')}
          </button>

          {showUsage && (
            <div className="space-y-4">
              {/* Period selector */}
              <div className="flex gap-1">
                {(['today', 'week', 'month'] as const).map((p) => (
                  <button
                    key={p}
                    onClick={() => setMeteringPeriod(p)}
                    className={cn(
                      'px-3 py-1 rounded-md text-xs font-medium transition-colors',
                      meteringPeriod === p ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground hover:text-foreground'
                    )}
                  >
                    {p === 'today' ? t('switch.periodToday') : p === 'week' ? t('switch.period7d') : t('switch.period30d')}
                  </button>
                ))}
              </div>

              {/* Insight summary cards */}
              {usageInsights && usageInsights.totalCalls > 0 && (
                <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                  <div className="rounded-lg border border-border bg-card p-3">
                    <div className="flex items-center gap-1.5 text-muted-foreground mb-1">
                      <DollarSign className="h-3 w-3" />
                      <span className="text-[10px] uppercase tracking-wider">{t('switch.costEstimate')}</span>
                    </div>
                    <p className="text-lg font-semibold">{formatCost(usageInsights.totalCostUSD)}</p>
                  </div>
                  <div className="rounded-lg border border-border bg-card p-3">
                    <div className="flex items-center gap-1.5 text-muted-foreground mb-1">
                      <Clock className="h-3 w-3" />
                      <span className="text-[10px] uppercase tracking-wider">{t('switch.upstreamLatency')}</span>
                    </div>
                    <p className="text-lg font-semibold">{usageInsights.avgLatencyMs}ms</p>
                  </div>
                  <div className="rounded-lg border border-border bg-card p-3">
                    <div className="flex items-center gap-1.5 text-muted-foreground mb-1">
                      <Database className="h-3 w-3" />
                      <span className="text-[10px] uppercase tracking-wider">{t('switch.cacheHitRate')}</span>
                    </div>
                    <p className="text-lg font-semibold">{(usageInsights.cacheHitRate * 100).toFixed(1)}%</p>
                  </div>
                  <div className={cn(
                    'rounded-lg border p-3',
                    usageInsights.rateLimitEvents > 0
                      ? 'border-red-500/30 bg-red-500/5'
                      : 'border-border bg-card'
                  )}>
                    <div className="flex items-center gap-1.5 text-muted-foreground mb-1">
                      <ShieldAlert className="h-3 w-3" />
                      <span className="text-[10px] uppercase tracking-wider">{t('switch.rateLimitTitle')}</span>
                    </div>
                    <p className={cn(
                      'text-lg font-semibold',
                      usageInsights.rateLimitEvents > 0 ? 'text-red-500' : ''
                    )}>
                      {usageInsights.rateLimitEvents > 0
                        ? `${usageInsights.rateLimitEvents} ${t('switch.throttled')}`
                        : '0'
                      }
                    </p>
                  </div>
                </div>
              )}

              {/* Cost breakdown by model */}
              {usageInsights && usageInsights.modelCosts && usageInsights.modelCosts.length > 0 && (
                <div className="rounded-lg border border-border bg-card p-4">
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                    <DollarSign className="h-3 w-3 inline mr-1" />
                    {t('switch.topModelCost')}
                  </p>
                  <div className="space-y-2">
                    {usageInsights.modelCosts.slice(0, 5).map((mc) => {
                      const maxCost = usageInsights.modelCosts[0]?.costUSD || 1
                      const pct = (mc.costUSD / maxCost) * 100
                      return (
                        <div key={mc.model} className="space-y-1">
                          <div className="flex items-center justify-between text-xs">
                            <span className="font-medium font-mono">{mc.model}</span>
                            <span className="text-muted-foreground">
                              {formatCost(mc.costUSD)} — {formatTokens(mc.tokensIn + mc.tokensOut)} {t('switch.tokens')}
                            </span>
                          </div>
                          <div className="h-1.5 bg-muted rounded-full overflow-hidden">
                            <div className="h-full bg-emerald-500 rounded-full transition-all" style={{ width: `${pct}%` }} />
                          </div>
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}

              {/* Per-app */}
              {appSummaries.length > 0 && (
                <div className="rounded-lg border border-border bg-card p-4">
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">{t('switch.byApp')}</p>
                  <div className="space-y-2">
                    {appSummaries.map((as) => {
                      const totalTokens = as.tokensIn + as.tokensOut
                      const maxTokens = Math.max(...appSummaries.map(a => a.tokensIn + a.tokensOut), 1)
                      const pct = (totalTokens / maxTokens) * 100
                      const appInfo = apps.find(a => a.id === as.appId)
                      return (
                        <div key={as.appId} className="space-y-1">
                          <div className="flex items-center justify-between text-xs">
                            <span className="font-medium">{appInfo?.name || as.appId}</span>
                            <span className="text-muted-foreground">
                              {as.totalCalls} {t('switch.calls')} — {formatTokens(totalTokens)} {t('switch.tokens')}
                            </span>
                          </div>
                          <div className="h-1.5 bg-muted rounded-full overflow-hidden">
                            <div className="h-full bg-primary rounded-full transition-all" style={{ width: `${pct}%` }} />
                          </div>
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}

              {/* Per-model */}
              {modelSummaries.length > 0 && (
                <div className="rounded-lg border border-border bg-card p-4">
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">{t('switch.byModel')}</p>
                  <div className="space-y-2">
                    {modelSummaries.map((ms) => {
                      const totalTokens = ms.tokensIn + ms.tokensOut
                      const maxTokens = Math.max(...modelSummaries.map(m => m.tokensIn + m.tokensOut), 1)
                      const pct = (totalTokens / maxTokens) * 100
                      return (
                        <div key={ms.model} className="space-y-1">
                          <div className="flex items-center justify-between text-xs">
                            <span className="font-medium font-mono">{ms.model}</span>
                            <span className="text-muted-foreground">
                              {ms.totalCalls} {t('switch.calls')} — {formatTokens(totalTokens)} {t('switch.tokens')}
                            </span>
                          </div>
                          <div className="h-1.5 bg-muted rounded-full overflow-hidden">
                            <div className="h-full bg-amber-500 rounded-full transition-all" style={{ width: `${pct}%` }} />
                          </div>
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}

              {/* 7-day sparkline */}
              {daySummaries.length > 0 && (
                <div className="rounded-lg border border-border bg-card p-4">
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">{t('switch.trend7d')}</p>
                  <div className="flex items-end gap-1 h-16">
                    {daySummaries.map((d) => {
                      const maxCalls = Math.max(...daySummaries.map(s => s.totalCalls), 1)
                      const pct = (d.totalCalls / maxCalls) * 100
                      return (
                        <div key={d.date} className="flex-1 flex flex-col items-center gap-0.5">
                          <div
                            className="w-full bg-primary/60 rounded-sm transition-all min-h-[2px]"
                            style={{ height: `${Math.max(pct, 3)}%` }}
                            title={`${d.date}: ${d.totalCalls} ${t('switch.calls')}`}
                          />
                          <span className="text-[8px] text-muted-foreground">{d.date.slice(5)}</span>
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}

              {/* Request Log */}
              {requestLog.length > 0 && (
                <div className="rounded-lg border border-border bg-card p-4">
                  <div className="flex items-center justify-between mb-3">
                    <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('switch.requestLog')}</p>
                    <div className="flex items-center gap-2">
                      <select
                        value={logFilterApp}
                        onChange={(e) => {
                          setLogFilterApp(e.target.value)
                          GetRequestLog(50, e.target.value, logFilterModel).then(r => setRequestLog(safeArray(r))).catch(() => {})
                        }}
                        className="px-1.5 py-0.5 rounded border border-border bg-background text-[10px]"
                      >
                        <option value="">{t('switch.allApps')}</option>
                        {apps.map(a => <option key={a.id} value={a.id}>{a.name}</option>)}
                      </select>
                      <select
                        value={logFilterModel}
                        onChange={(e) => {
                          setLogFilterModel(e.target.value)
                          GetRequestLog(50, logFilterApp, e.target.value).then(r => setRequestLog(safeArray(r))).catch(() => {})
                        }}
                        className="px-1.5 py-0.5 rounded border border-border bg-background text-[10px]"
                      >
                        <option value="">{t('switch.allModels')}</option>
                        {[...new Set(requestLog.map(r => r.model))].filter(Boolean).map(m => (
                          <option key={m} value={m}>{m}</option>
                        ))}
                      </select>
                    </div>
                  </div>
                  <div className="space-y-0.5 max-h-64 overflow-y-auto">
                    {requestLog.map((r) => {
                      const appInfo = apps.find(a => a.id === r.appId)
                      const isErr = r.statusCode >= 400
                      const is429 = r.statusCode === 429
                      return (
                        <div
                          key={r.id}
                          className={cn(
                            'flex items-center justify-between text-xs py-1.5 px-2 rounded border-b border-border/30 last:border-0',
                            is429 ? 'bg-amber-500/5' : isErr ? 'bg-red-500/5' : ''
                          )}
                        >
                          <div className="flex items-center gap-2 min-w-0">
                            <span className="text-muted-foreground font-mono text-[10px] w-12 flex-shrink-0">
                              {r.timestamp.slice(11, 16)}
                            </span>
                            <span className={cn(
                              'px-1 py-0.5 rounded text-[9px] font-mono font-bold flex-shrink-0',
                              r.statusCode < 300 ? 'bg-green-500/10 text-green-600' :
                              r.statusCode < 400 ? 'bg-blue-500/10 text-blue-500' :
                              is429 ? 'bg-amber-500/10 text-amber-600' :
                              'bg-red-500/10 text-red-500'
                            )}>
                              {r.statusCode}
                            </span>
                            <span className="font-medium truncate">{appInfo?.name || r.appId}</span>
                            {r.cached && (
                              <span className="text-[9px] px-1 py-0.5 rounded bg-blue-500/10 text-blue-500 flex-shrink-0">
                                {t('switch.cached')}
                              </span>
                            )}
                          </div>
                          <div className="flex items-center gap-3 text-muted-foreground flex-shrink-0">
                            <span className="font-mono text-[10px] max-w-[120px] truncate">{r.model}</span>
                            <span className="text-[10px] w-14 text-right">{r.latencyMs}ms</span>
                            <span className="text-[10px] w-12 text-right">{formatTokens(r.tokensIn + r.tokensOut)}</span>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}

              {/* Empty state */}
              {appSummaries.length === 0 && modelSummaries.length === 0 && requestLog.length === 0 && (
                <div className="rounded-lg border border-dashed border-border p-8 text-center">
                  <Activity className="h-8 w-8 mx-auto text-muted-foreground/50 mb-2" />
                  <p className="text-sm font-medium text-muted-foreground">{t('switch.noUsage')}</p>
                  <p className="text-xs text-muted-foreground/70 mt-1">{t('switch.noUsageDesc')}</p>
                </div>
              )}
            </div>
          )}
        </div>

        {/* ── Snapshot restore modal ── */}
        {snapshotTool && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-card border border-border rounded-lg p-6 max-w-md w-full mx-4 shadow-xl">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                  <RotateCw className="h-4 w-4 text-primary" />
                  <h3 className="font-semibold">{t('switch.snapshots')} — {snapshotTool}</h3>
                </div>
                <button
                  onClick={() => setSnapshotTool(null)}
                  className="p-1 rounded hover:bg-muted text-muted-foreground"
                >
                  &times;
                </button>
              </div>
              {snapshots.length === 0 ? (
                <p className="text-sm text-muted-foreground py-4 text-center">{t('switch.snapshotEmpty')}</p>
              ) : (
                <div className="space-y-1.5 max-h-64 overflow-y-auto">
                  {snapshots.map((snap) => (
                    <div key={snap.id} className="flex items-center justify-between rounded-md border border-border px-3 py-2">
                      <div className="min-w-0">
                        <p className="text-xs font-medium truncate">{snap.label}</p>
                        <p className="text-[10px] text-muted-foreground">
                          {new Date(snap.createdAt).toLocaleString()} — {snap.size} bytes
                        </p>
                      </div>
                      <button
                        onClick={() => handleRestoreSnapshot(snapshotTool, snap.id)}
                        disabled={restoringSnapshot === snap.id}
                        className="flex items-center gap-1 px-2 py-1 rounded text-xs font-medium border border-primary/30 text-primary hover:bg-primary/10 disabled:opacity-50 flex-shrink-0 ml-2"
                      >
                        {restoringSnapshot === snap.id ? (
                          <Loader2 className="h-3 w-3 animate-spin" />
                        ) : (
                          <RotateCw className="h-3 w-3" />
                        )}
                        {t('switch.snapshotRestore')}
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {/* ── Delete confirmation modal ── */}
        {confirmDelete && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-card border border-border rounded-lg p-6 max-w-sm w-full mx-4 shadow-xl">
              <div className="flex items-center gap-3 mb-4">
                <Trash2 className="h-5 w-5 text-red-500" />
                <h3 className="font-semibold">{t('switch.deleteApp')}</h3>
              </div>
              <p className="text-sm text-muted-foreground mb-6">{t('switch.deleteAppDesc')}</p>
              <div className="flex gap-3">
                <button
                  onClick={() => setConfirmDelete(null)}
                  className="flex-1 px-4 py-2 rounded-md text-sm border border-border hover:bg-muted"
                >
                  {t('switch.cancel')}
                </button>
                <button
                  onClick={() => handleDelete(confirmDelete)}
                  className="flex-1 px-4 py-2 rounded-md text-sm bg-red-500 text-white hover:bg-red-600"
                >
                  {t('switch.deleteConfirm')}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
