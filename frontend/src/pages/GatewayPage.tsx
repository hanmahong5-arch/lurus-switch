import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Server, Download, Play, Square, ExternalLink, RefreshCw, ToggleLeft, ToggleRight } from 'lucide-react'
import { useGatewayStore } from '../stores/gatewayStore'
import {
  GetServerStatus,
  StartServer,
  StopServer,
  EnsureServerBinary,
  GetServerAdminToken,
  OpenServerAdminPanel,
  GetServerConfig,
  SaveServerConfig,
} from '../../wailsjs/go/main/App'
import type { serverctl } from '../../wailsjs/go/models'

export function GatewayPage() {
  const { t } = useTranslation()
  const { status, setStatus, setAdminToken, startPolling, stopPolling } = useGatewayStore()

  const [actionLoading, setActionLoading] = useState(false)
  const [downloadProgress, setDownloadProgress] = useState<{ pct: number } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [config, setConfig] = useState<serverctl.ServerConfig | null>(null)

  useEffect(() => {
    startPolling(
      () => GetServerStatus() as Promise<ReturnType<typeof useGatewayStore.getState>['status'] & object>,
      () => GetServerAdminToken(),
    )
    GetServerConfig().then(setConfig)
    return () => stopPolling()
  }, [])

  const handleStartStop = async () => {
    setError(null)
    setActionLoading(true)
    try {
      if (status?.running) {
        await StopServer()
      } else {
        await StartServer()
      }
      const s = await GetServerStatus()
      setStatus(s as Parameters<typeof setStatus>[0])
    } catch (e) {
      setError(String(e))
    } finally {
      setActionLoading(false)
    }
  }

  const handleDownload = async () => {
    setError(null)
    setActionLoading(true)
    setDownloadProgress({ pct: 0 })
    try {
      await EnsureServerBinary()
      setDownloadProgress(null)
      const s = await GetServerStatus()
      setStatus(s as Parameters<typeof setStatus>[0])
    } catch (e) {
      setError(String(e))
      setDownloadProgress(null)
    } finally {
      setActionLoading(false)
    }
  }

  const handleAutoStartToggle = async () => {
    if (!config) return
    const next = { ...config, auto_start: !config.auto_start }
    try {
      await SaveServerConfig(next)
      setConfig(next)
    } catch (e) {
      setError(String(e))
    }
  }

  const formatUptime = (seconds: number) => {
    if (seconds < 60) return `${seconds}s`
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
    return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto p-6 space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-2xl font-bold flex items-center gap-2">
          <Server className="h-6 w-6 text-indigo-400" />
          {t('gateway.server')}
        </h2>
        <p className="text-sm text-muted-foreground mt-1">{t('gateway.subtitle')}</p>
      </div>

      {/* Status Card */}
      <div className="rounded-lg border border-border bg-card p-5 space-y-4">
        {/* Status row */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span
              className={`h-3 w-3 rounded-full ${status?.running ? 'bg-green-500' : 'bg-muted-foreground'}`}
            />
            <span className="font-semibold text-lg">
              {status?.running ? t('gateway.status.running') : t('gateway.status.stopped')}
            </span>
          </div>
          {status && (
            <span className="text-xs text-muted-foreground">
              {t('common.port', 'Port')}: {status.port}
            </span>
          )}
        </div>

        {/* Uptime */}
        {status?.running && status.uptime > 0 && (
          <div className="text-sm text-muted-foreground">
            {t('common.uptime', 'Uptime')}: {formatUptime(status.uptime)}
          </div>
        )}

        {/* Error */}
        {error && (
          <div className="text-sm text-red-400 bg-red-900/20 rounded px-3 py-2">{error}</div>
        )}

        {/* Download progress */}
        {downloadProgress && (
          <div className="space-y-1">
            <div className="text-xs text-muted-foreground">{t('gateway.downloadBinary')}...</div>
            <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
              <div
                className="h-full bg-indigo-500 transition-all"
                style={{ width: `${downloadProgress.pct}%` }}
              />
            </div>
          </div>
        )}

        {/* Action buttons */}
        <div className="flex flex-wrap gap-3 pt-1">
          {/* Download binary if missing */}
          {status && !status.binaryOk && (
            <button
              onClick={handleDownload}
              disabled={actionLoading}
              className="flex items-center gap-2 px-4 py-2 rounded-md bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium disabled:opacity-50"
            >
              <Download className="h-4 w-4" />
              {t('gateway.downloadBinary')}
            </button>
          )}

          {/* Start / Stop */}
          {(status?.binaryOk || status?.running) && (
            <button
              onClick={handleStartStop}
              disabled={actionLoading}
              className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium disabled:opacity-50 ${
                status?.running
                  ? 'bg-red-700 hover:bg-red-600 text-white'
                  : 'bg-green-700 hover:bg-green-600 text-white'
              }`}
            >
              {actionLoading ? (
                <RefreshCw className="h-4 w-4 animate-spin" />
              ) : status?.running ? (
                <Square className="h-4 w-4" />
              ) : (
                <Play className="h-4 w-4" />
              )}
              {status?.running ? t('gateway.stopServer') : t('gateway.startServer')}
            </button>
          )}

          {/* Open admin panel */}
          {status?.running && (
            <button
              onClick={() => OpenServerAdminPanel()}
              className="flex items-center gap-2 px-4 py-2 rounded-md border border-border hover:bg-muted text-sm font-medium"
            >
              <ExternalLink className="h-4 w-4" />
              {t('gateway.openAdminPanel')}
            </button>
          )}
        </div>
      </div>

      {/* Config Card */}
      {config && (
        <div className="rounded-lg border border-border bg-card p-5 space-y-4">
          <h3 className="font-semibold text-sm text-muted-foreground uppercase tracking-wide">
            {t('settings.title')}
          </h3>

          {/* Auto-start toggle */}
          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm font-medium">{t('gateway.autoStart')}</div>
            </div>
            <button onClick={handleAutoStartToggle} className="text-indigo-400 hover:text-indigo-300">
              {config.auto_start ? (
                <ToggleRight className="h-6 w-6" />
              ) : (
                <ToggleLeft className="h-6 w-6" />
              )}
            </button>
          </div>

          {/* Port display */}
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">{t('common.port', 'Port')}</span>
            <span className="font-mono">{config.port}</span>
          </div>
        </div>
      )}
    </div>
  )
}
