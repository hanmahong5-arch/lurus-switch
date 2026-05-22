import { useEffect, useState } from 'react'
import { Shield, Activity, RefreshCw, Loader2, ExternalLink, Server, Monitor } from 'lucide-react'
import { Button, Card } from '../components/ui'
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
      errorToast(toast, err, { currentPage: 'gateway' })
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
      errorToast(toast, err, { currentPage: 'gateway' })
    } finally {
      setToolsLoading(false)
    }
  }

  const checkUpdate = async () => {
    try {
      const info = await CheckSelfUpdate()
      setUpdateInfo(info as { updateAvailable: boolean; latestVersion: string })
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway' })
    }
  }

  const applyUpdate = async () => {
    setUpdating(true)
    try {
      await ApplySelfUpdate()
    } catch (err) {
      errorToast(toast, err, { currentPage: 'gateway' })
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
            <Shield className="h-5 w-5 text-primary" />
            {t('admin.title')}
          </h2>
          <p className="text-sm text-muted-foreground">{t('admin.subtitle')}</p>
        </div>

        {/* API Health */}
        <Card variant="default" className="p-4 space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground flex items-center gap-2">
              <Activity className="h-3.5 w-3.5" />
              [ {t('admin.serviceStatus').toUpperCase()} ]
            </h3>
            <Button
              variant="secondary"
              size="sm"
              onClick={pingAPI}
              disabled={pingLoading}
              loading={pingLoading}
              icon={!pingLoading ? <RefreshCw className="h-3 w-3" /> : undefined}
            >
              {t('dashboard.refresh')}
            </Button>
          </div>
          <div className="flex items-center gap-3">
            <div className={cn(
              'h-2.5 w-2.5 rounded-full',
              apiOnline === null ? 'bg-muted' : apiOnline ? 'bg-emerald-400 animate-pulse' : 'bg-red-400'
            )} />
            <span className="text-sm font-mono">
              Lurus API —{' '}
              {apiOnline === null
                ? t('admin.checking')
                : apiOnline
                ? <span className="text-emerald-400">▸ {t('admin.online')}</span>
                : <span className="text-red-400">▪ {t('admin.offline')}</span>
              }
            </span>
          </div>
        </Card>

        {/* System Info */}
        <Card variant="default" className="p-4 space-y-3">
          <h3 className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground flex items-center gap-2">
            <Monitor className="h-3.5 w-3.5" />
            [ {t('admin.systemInfo').toUpperCase()} ]
          </h3>
          {sysInfo ? (
            <div className="grid grid-cols-2 gap-2 text-sm">
              {[
                { label: t('admin.appVersion'), value: `v${appVersion || sysInfo.appVersion || '...'}` },
                { label: t('admin.os'), value: sysInfo.goos },
                { label: t('admin.arch'), value: sysInfo.goarch },
              ].map(({ label, value }) => (
                <div key={label}>
                  <p className="text-xs text-muted-foreground font-mono">{label}</p>
                  <p className="font-mono text-xs tabular-nums">{value}</p>
                </div>
              ))}
            </div>
          ) : (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              {t('status.loading')}
            </div>
          )}
        </Card>

        {/* Self Update */}
        <Card variant="default" className="p-4 space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground">[ {t('admin.appUpdate').toUpperCase()} ]</h3>
            {updateInfo?.updateAvailable ? (
              <Button
                size="sm"
                onClick={applyUpdate}
                disabled={updating}
                loading={updating}
              >
                {t('dashboard.updateTo', { version: updateInfo.latestVersion })}
              </Button>
            ) : (
              <Button variant="secondary" size="sm" onClick={checkUpdate}>
                {t('dashboard.checkUpdates')}
              </Button>
            )}
          </div>
          <p className="text-xs text-muted-foreground">
            {updateInfo
              ? updateInfo.updateAvailable
                ? t('admin.newVersionFound', { version: updateInfo.latestVersion })
                : t('admin.upToDate')
              : t('admin.checkUpdateHint')}
          </p>
        </Card>

        {/* Tool Summary */}
        <Card variant="default" className="overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-card-recessed">
            <h3 className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground flex items-center gap-2">
              <Server className="h-3.5 w-3.5" />
              [ {t('admin.toolSummary').toUpperCase()} ]
            </h3>
            <Button
              variant="ghost"
              size="sm"
              onClick={loadTools}
              disabled={toolsLoading}
              loading={toolsLoading}
              icon={!toolsLoading ? <RefreshCw className="h-3.5 w-3.5" /> : undefined}
            />
          </div>
          {TOOL_ORDER.map((name) => {
            const tool = tools[name]
            return (
              <div key={name} className="flex items-center justify-between px-4 py-2.5 border-b border-border last:border-0 text-sm">
                <span className="font-medium capitalize">{name}</span>
                <div className="flex items-center gap-3">
                  {tool?.installed ? (
                    <>
                      <span className="text-xs text-muted-foreground font-mono tabular-nums">{tool.version || 'n/a'}</span>
                      <span className="text-xs bg-emerald-500/15 text-emerald-400 px-1.5 py-0.5 rounded font-mono">▸ {t('admin.installed')}</span>
                    </>
                  ) : (
                    <span className="text-xs bg-card-recessed text-muted-foreground px-1.5 py-0.5 rounded font-mono">▪ {t('admin.notInstalled')}</span>
                  )}
                </div>
              </div>
            )
          })}
        </Card>

        {/* Quick Links */}
        <Card variant="default" className="p-4 space-y-3">
          <h3 className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground">[ {t('admin.quickLinks').toUpperCase()} ]</h3>
          <div className="space-y-2">
            {QUICK_LINKS.map(({ labelKey, url }) => (
              <a
                key={labelKey}
                href={url}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center justify-between px-3 py-2 text-sm border border-border rounded hover:bg-muted hover:border-rule-strong transition-all duration-150"
              >
                <span>{t(labelKey)}</span>
                <ExternalLink className="h-3.5 w-3.5 text-muted-foreground" />
              </a>
            ))}
          </div>
        </Card>
      </div>
    </div>
  )
}
