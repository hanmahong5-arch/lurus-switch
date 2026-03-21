import { useEffect, useState } from 'react'
import { Shield, Activity, RefreshCw, Loader2, ExternalLink, Server, Monitor } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { errorToast } from '../lib/errorToast'
import { useToastStore } from '../stores/toastStore'
import { PingLurusAPI, GetSystemInfo, GetAppVersion, CheckSelfUpdate, ApplySelfUpdate, DetectAllTools } from '../../wailsjs/go/main/App'

interface SystemInfo {
  appVersion: string
  goos: string
  goarch: string
}

interface ToolStatus {
  name: string
  installed: boolean
  version: string
  path: string
}

export function AdminPage() {
  const { t } = useTranslation()
  const [apiOnline, setApiOnline] = useState<boolean | null>(null)
  const [pingLoading, setPingLoading] = useState(false)
  const [sysInfo, setSysInfo] = useState<SystemInfo | null>(null)
  const [appVersion, setAppVersion] = useState('')
  const [tools, setTools] = useState<Record<string, ToolStatus>>({})
  const [toolsLoading, setToolsLoading] = useState(false)
  const [updateInfo, setUpdateInfo] = useState<{ updateAvailable: boolean; latestVersion: string } | null>(null)
  const [updating, setUpdating] = useState(false)
  const toast = useToastStore((s) => s.addToast)

  useEffect(() => {
    loadAll()
  }, [])

  const loadAll = async () => {
    pingAPI()
    try {
      const [info, ver] = await Promise.all([
        GetSystemInfo(),
        GetAppVersion(),
      ])
      setSysInfo(info as unknown as SystemInfo)
      setAppVersion(ver)
    } catch (err) {
      errorToast(toast, err, { currentPage: 'api-admin' })
    }
    loadTools()
  }

  const pingAPI = async () => {
    setPingLoading(true)
    try {
      const ok = await PingLurusAPI()
      setApiOnline(ok)
    } catch {
      setApiOnline(false)
    } finally {
      setPingLoading(false)
    }
  }

  const loadTools = async () => {
    setToolsLoading(true)
    try {
      const statuses = await DetectAllTools()
      setTools(statuses || {})
    } catch (err) {
      errorToast(toast, err, { currentPage: 'api-admin' })
    } finally {
      setToolsLoading(false)
    }
  }

  const checkUpdate = async () => {
    try {
      const info = await CheckSelfUpdate()
      setUpdateInfo(info as { updateAvailable: boolean; latestVersion: string })
    } catch (err) {
      errorToast(toast, err, { currentPage: 'api-admin' })
    }
  }

  const applyUpdate = async () => {
    setUpdating(true)
    try {
      await ApplySelfUpdate()
    } catch (err) {
      errorToast(toast, err, { currentPage: 'api-admin' })
    } finally {
      setUpdating(false)
    }
  }

  const TOOL_ORDER = ['claude', 'codex', 'gemini', 'picoclaw', 'nullclaw']

  const QUICK_LINKS = [
    { labelKey: 'admin.links.anthropicDocs', url: 'https://docs.anthropic.com' },
    { labelKey: 'admin.links.codexDocs', url: 'https://platform.openai.com/docs' },
    { labelKey: 'admin.links.geminiDocs', url: 'https://github.com/google-gemini/gemini-cli' },
    { labelKey: 'admin.links.githubIssues', url: 'https://github.com' },
  ]

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-2xl mx-auto p-6 space-y-6">
        {/* Header */}
        <div>
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <Shield className="h-5 w-5 text-red-400" />
            {t('admin.title')}
          </h2>
          <p className="text-sm text-muted-foreground">{t('admin.subtitle')}</p>
        </div>

        {/* API Health */}
        <div className="border border-border rounded-lg p-4 space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold flex items-center gap-2">
              <Activity className="h-4 w-4 text-muted-foreground" />
              {t('admin.serviceStatus')}
            </h3>
            <button
              onClick={pingAPI}
              disabled={pingLoading}
              className="flex items-center gap-1 px-2 py-1 text-xs border border-border rounded hover:bg-muted transition-colors disabled:opacity-50"
            >
              {pingLoading ? <Loader2 className="h-3 w-3 animate-spin" /> : <RefreshCw className="h-3 w-3" />}
              {t('dashboard.refresh')}
            </button>
          </div>
          <div className="flex items-center gap-3">
            <div className={cn(
              'h-2.5 w-2.5 rounded-full',
              apiOnline === null ? 'bg-muted' : apiOnline ? 'bg-green-500' : 'bg-red-500'
            )} />
            <span className="text-sm">
              Lurus API —{' '}
              {apiOnline === null
                ? t('admin.checking')
                : apiOnline
                ? <span className="text-green-500">{t('admin.online')}</span>
                : <span className="text-red-500">{t('admin.offline')}</span>
              }
            </span>
          </div>
        </div>

        {/* System Info */}
        <div className="border border-border rounded-lg p-4 space-y-3">
          <h3 className="text-sm font-semibold flex items-center gap-2">
            <Monitor className="h-4 w-4 text-muted-foreground" />
            {t('admin.systemInfo')}
          </h3>
          {sysInfo ? (
            <div className="grid grid-cols-2 gap-2 text-sm">
              {[
                { label: t('admin.appVersion'), value: `v${appVersion || sysInfo.appVersion || '...'}` },
                { label: t('admin.os'), value: sysInfo.goos },
                { label: t('admin.arch'), value: sysInfo.goarch },
              ].map(({ label, value }) => (
                <div key={label}>
                  <p className="text-xs text-muted-foreground">{label}</p>
                  <p className="font-mono text-xs">{value}</p>
                </div>
              ))}
            </div>
          ) : (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              {t('status.loading')}
            </div>
          )}
        </div>

        {/* Self Update */}
        <div className="border border-border rounded-lg p-4 space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold">{t('admin.appUpdate')}</h3>
            {updateInfo?.updateAvailable ? (
              <button
                onClick={applyUpdate}
                disabled={updating}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {updating ? <Loader2 className="h-3 w-3 animate-spin" /> : null}
                {t('dashboard.updateTo', { version: updateInfo.latestVersion })}
              </button>
            ) : (
              <button
                onClick={checkUpdate}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs border border-border rounded hover:bg-muted transition-colors"
              >
                {t('dashboard.checkUpdates')}
              </button>
            )}
          </div>
          <p className="text-xs text-muted-foreground">
            {updateInfo
              ? updateInfo.updateAvailable
                ? t('admin.newVersionFound', { version: updateInfo.latestVersion })
                : t('admin.upToDate')
              : t('admin.checkUpdateHint')}
          </p>
        </div>

        {/* Tool Summary */}
        <div className="border border-border rounded-lg overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-muted/30">
            <h3 className="text-sm font-semibold flex items-center gap-2">
              <Server className="h-4 w-4 text-muted-foreground" />
              {t('admin.toolSummary')}
            </h3>
            <button
              onClick={loadTools}
              disabled={toolsLoading}
              className="p-1 text-muted-foreground hover:text-foreground disabled:opacity-50"
            >
              {toolsLoading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
            </button>
          </div>
          {TOOL_ORDER.map((name) => {
            const tool = tools[name]
            return (
              <div key={name} className="flex items-center justify-between px-4 py-2.5 border-b border-border last:border-0 text-sm">
                <span className="font-medium capitalize">{name}</span>
                <div className="flex items-center gap-3">
                  {tool?.installed ? (
                    <>
                      <span className="text-xs text-muted-foreground font-mono">{tool.version || 'n/a'}</span>
                      <span className="text-xs bg-green-500/10 text-green-500 px-1.5 py-0.5 rounded">{t('admin.installed')}</span>
                    </>
                  ) : (
                    <span className="text-xs bg-muted text-muted-foreground px-1.5 py-0.5 rounded">{t('admin.notInstalled')}</span>
                  )}
                </div>
              </div>
            )
          })}
        </div>

        {/* Quick Links */}
        <div className="border border-border rounded-lg p-4 space-y-3">
          <h3 className="text-sm font-semibold">{t('admin.quickLinks')}</h3>
          <div className="space-y-2">
            {QUICK_LINKS.map(({ labelKey, url }) => (
              <a
                key={labelKey}
                href={url}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center justify-between px-3 py-2 text-sm border border-border rounded hover:bg-muted transition-colors"
              >
                <span>{t(labelKey)}</span>
                <ExternalLink className="h-3.5 w-3.5 text-muted-foreground" />
              </a>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
