import { useEffect, useState, useCallback } from 'react'
import { Save, Loader2, CheckCircle2, Stethoscope, FolderOpen, Bell, Send } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { Button, Card } from '../components/ui'
import { useClassifiedError } from '../lib/useClassifiedError'
import { InlineError } from '../components/InlineError'
import { classifyError } from '../lib/errorClassifier'
import { GetAppSettings, SaveAppSettings, ClearAllSnapshots, ClearAllUserPrompts, SetAppMode, IsModeLocked, GetSystemInfo, GetConfigDir, OpenConfigDir } from '../../wailsjs/go/main/App'
import { appconfig } from '../../wailsjs/go/models'
import { useConfigStore, type AppMode, type UserLevel } from '../stores/configStore'
import { RESELLER_ONLY_PAGES, PERSONAL_ONLY_PAGES } from '../components/Sidebar'
import { DiagnosticsModal } from '../components/DiagnosticsModal'
import { CompetingInstallBanner } from '../components/CompetingInstallBanner'
import { UpstreamProxySection } from '../components/UpstreamProxySection'
import {
  DEFAULT_NOTIFY_CONFIG,
  getNotifyConfig,
  getRecentNotifications,
  saveNotifyConfig,
  testNotify,
  type NotifyConfig,
  type NotifyEvent,
} from '../lib/notifyApi'
import { useToastStore } from '../stores/toastStore'
import { StartupPerformanceCard } from '../components/StartupPerformanceCard'
import { CustomProvidersSection } from '../components/CustomProvidersSection'
import { BackupRestoreCard } from '../components/BackupRestoreCard'
import { ModelHealthMatrix } from '../components/ModelHealthMatrix'

type Tab = 'appearance' | 'providers' | 'proxy' | 'notify' | 'update' | 'backup' | 'data'

interface AppSettings {
  theme: string
  language: string
  autoUpdate: boolean
  editorFontSize: number
  startupPage: string
  onboardingCompleted: boolean
  appMode: string
}

const DEFAULT: AppSettings = {
  theme: 'dark',
  language: 'zh',
  autoUpdate: true,
  editorFontSize: 13,
  startupPage: 'home',
  onboardingCompleted: true,
  appMode: 'user',
}

export function SettingsPage() {
  const [activeTab, setActiveTab] = useState<Tab>('appearance')
  const [settings, setSettings] = useState<AppSettings>(DEFAULT)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const { classified: error, setError, clearError } = useClassifiedError()
  // modeConfirm holds the proposed target when the user clicks a different
  // mode pill — null means no pending switch. Excludes 'unset' since that's
  // the bootstrap-only state.
  const [modeConfirm, setModeConfirm] = useState<Exclude<AppMode, 'unset'> | null>(null)
  const [modeLocked, setModeLocked] = useState(false)
  const [diagnosticsOpen, setDiagnosticsOpen] = useState(false)
  const [sysInfo, setSysInfo] = useState<{ appVersion: string; goos: string; goarch: string } | null>(null)
  const [configDir, setConfigDir] = useState<string>('')
  const { t, i18n } = useTranslation()
  const { setAppMode, setUserLevel, activeTool, setActiveTool } = useConfigStore()

  // Apply theme to document root
  const applyTheme = useCallback((theme: string) => {
    const root = document.documentElement
    if (theme === 'auto') {
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
      root.classList.toggle('dark', prefersDark)
    } else {
      root.classList.toggle('dark', theme === 'dark')
    }
  }, [])

  useEffect(() => {
    GetAppSettings().then((s) => {
      if (s) {
        const loaded = s as AppSettings
        setSettings(loaded)
        applyTheme(loaded.theme)
        i18n.changeLanguage(loaded.language)
      }
    }).catch(() => {}).finally(() => setLoading(false))
    IsModeLocked().then(setModeLocked).catch(() => setModeLocked(false))
    GetSystemInfo().then((info) => { if (info) setSysInfo(info as any) }).catch(() => {})
    GetConfigDir().then(setConfigDir).catch(() => {})
  }, [])

  // Listen for system theme changes when in auto mode
  useEffect(() => {
    if (settings.theme !== 'auto') return
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const handler = () => applyTheme('auto')
    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [settings.theme, applyTheme])

  const handleSave = async () => {
    setSaving(true)
    clearError()
    try {
      await SaveAppSettings(appconfig.AppSettings.createFrom(settings))
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch (err) {
      setError(err)
    } finally {
      setSaving(false)
    }
  }

  const handleLanguageChange = (lang: string) => {
    setSettings({ ...settings, language: lang })
    i18n.changeLanguage(lang)
  }

  const tabs: { id: Tab; label: string }[] = [
    { id: 'appearance', label: t('settings.tabs.appearance') },
    { id: 'providers', label: t('settings.tabs.providers', '供应商') },
    { id: 'proxy', label: t('settings.tabs.proxy') },
    { id: 'notify', label: t('settings.tabs.notify', '通知') },
    { id: 'update', label: t('settings.tabs.update') },
    { id: 'backup', label: t('settings.tabs.backup', '备份') },
    { id: 'data', label: t('settings.tabs.data') },
  ]

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-2xl mx-auto p-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold">{t('settings.title')}</h2>
            <p className="text-sm text-muted-foreground">{t('settings.subtitle')}</p>
          </div>
          <Button
            size="lg"
            onClick={handleSave}
            disabled={saving}
            loading={saving}
            icon={!saving ? (saved ? <CheckCircle2 className="h-4 w-4" /> : <Save className="h-4 w-4" />) : undefined}
          >
            {saved ? t('settings.saved') : t('settings.save')}
          </Button>
        </div>

        {error && (
          <InlineError
            category={error.category}
            message={error.message}
            details={error.details}
            onDismiss={clearError}
          />
        )}

        {/* Hermes-style info strip — version + paths + Diagnose at a glance.
            Replaces the "About" tab that used to be buried elsewhere. */}
        <Card variant="recessed" className="p-3 space-y-2">
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-[11px]">
            <InfoBadge
              label={t('settings.info.version', '版本')}
              value={sysInfo ? `v${sysInfo.appVersion}` : '—'}
            />
            <InfoBadge
              label={t('settings.info.platform', '平台')}
              value={sysInfo ? `${sysInfo.goos}/${sysInfo.goarch}` : '—'}
            />
            <InfoBadge
              label={t('settings.info.mode', '模式')}
              value={t(`mode.${settings.appMode}.label`, settings.appMode)}
            />
            <InfoBadge
              label={t('settings.info.configDir', '配置目录')}
              value={configDir ? configDir.split(/[\\/]/).pop() ?? '' : '—'}
              title={configDir}
            />
          </div>
          <div className="flex items-center gap-2 pt-1 border-t border-border">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setDiagnosticsOpen(true)}
              icon={<Stethoscope className="h-3 w-3" />}
            >
              {t('settings.info.diagnose', '运行诊断')}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => OpenConfigDir().catch(() => {})}
              icon={<FolderOpen className="h-3 w-3" />}
            >
              {t('settings.info.openConfigDir', '打开配置目录')}
            </Button>
          </div>
        </Card>

        <CompetingInstallBanner onJumpToTools={() => setActiveTool('tools')} />

        <DiagnosticsModal open={diagnosticsOpen} onClose={() => setDiagnosticsOpen(false)} />

        {/* Tabs */}
        <div className="border-b border-border">
          <nav className="flex gap-1 -mb-px">
            {tabs.map((tab) => {
              const isActive = activeTab === tab.id
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={cn(
                    'px-4 py-2 border-b-2 transition-all duration-150',
                    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary rounded-t-sm',
                    isActive
                      ? 'border-primary text-primary'
                      : 'border-transparent text-muted-foreground hover:text-foreground',
                  )}
                >
                  <span className={isActive ? 'font-mono text-[11px] tracking-[0.12em]' : 'text-sm font-medium'}>
                    {isActive ? `[ ${tab.label.toUpperCase()} ]` : tab.label}
                  </span>
                </button>
              )
            })}
          </nav>
        </div>

        {/* Tab Content */}
        {activeTab === 'appearance' && (
          <div className="space-y-6">
            <SettingRow label={t('settings.appearance.theme')} description={t('settings.appearance.themeDesc')}>
              <select
                value={settings.theme}
                onChange={(e) => {
                  const theme = e.target.value
                  setSettings({ ...settings, theme })
                  applyTheme(theme)
                }}
                className="px-3 py-1.5 text-sm bg-muted border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
              >
                <option value="dark">{t('settings.appearance.themeDark')}</option>
                <option value="light">{t('settings.appearance.themeLight')}</option>
                <option value="auto">{t('settings.appearance.themeSystem')}</option>
              </select>
            </SettingRow>

            <SettingRow label={t('settings.appearance.language')} description={t('settings.appearance.languageDesc')}>
              <select
                value={settings.language}
                onChange={(e) => handleLanguageChange(e.target.value)}
                className="px-3 py-1.5 text-sm bg-muted border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
              >
                <option value="zh">中文</option>
                <option value="en">English</option>
              </select>
            </SettingRow>

            <SettingRow label={t('settings.appearance.fontSize')} description={t('settings.appearance.fontSizeDesc', { size: settings.editorFontSize })}>
              <input
                type="range"
                min={10}
                max={24}
                value={settings.editorFontSize}
                onChange={(e) => setSettings({ ...settings, editorFontSize: Number(e.target.value) })}
                className="w-32"
              />
              <span className="text-sm text-muted-foreground w-8 text-right">{settings.editorFontSize}</span>
            </SettingRow>

            <SettingRow label={t('settings.appearance.startupPage')} description={t('settings.appearance.startupPageDesc')}>
              <select
                value={settings.startupPage}
                onChange={(e) => setSettings({ ...settings, startupPage: e.target.value })}
                className="px-3 py-1.5 text-sm bg-muted border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
              >
                <option value="home">{t('nav.home')}</option>
                <option value="tools">{t('nav.tools')}</option>
                <option value="gateway">{t('nav.gateway')}</option>
                <option value="workspace">{t('nav.workspace')}</option>
                <option value="account">{t('nav.account')}</option>
                <option value="settings">{t('nav.settings')}</option>
              </select>
            </SettingRow>

            <div className="border-t border-border pt-4">
              <SettingRow label={t('settings.appMode')} description={t('settings.appModeDesc')}>
                <div className="flex rounded-md border border-border overflow-hidden">
                  {(['personal', 'reseller', 'enduser'] as const).map((m) => (
                    <button
                      key={m}
                      disabled={modeLocked && settings.appMode !== m}
                      onClick={() => {
                        if (modeLocked) return
                        if (settings.appMode !== m) setModeConfirm(m)
                      }}
                      className={cn(
                        'px-3 py-1.5 text-sm font-medium transition-colors',
                        settings.appMode === m
                          ? 'bg-primary text-primary-foreground'
                          : 'bg-muted text-muted-foreground hover:text-foreground',
                        modeLocked && settings.appMode !== m && 'opacity-50 cursor-not-allowed'
                      )}
                      title={modeLocked ? t('settings.modeLockedHint', '已被白标包锁定') : ''}
                    >
                      {t(`mode.${m}.label`, m)}
                    </button>
                  ))}
                </div>
              </SettingRow>

              {/* Mode switch confirmation dialog */}
              {modeConfirm && (
                <div className="mt-3 p-3 rounded-md border border-primary/30 bg-primary/5">
                  <p className="text-sm font-medium">
                    {t('settings.modeSwitchConfirm', {
                      mode: t(`mode.${modeConfirm}.label`, modeConfirm),
                    })}
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">
                    {t(`mode.${modeConfirm}.desc`, '')}
                  </p>
                  <div className="flex gap-2 mt-2">
                    <button
                      onClick={() => setModeConfirm(null)}
                      className="px-3 py-1 text-xs border border-border rounded hover:bg-muted"
                    >
                      {t('settings.data.cancel')}
                    </button>
                    <button
                      onClick={async () => {
                        const newMode = modeConfirm
                        try {
                          await SetAppMode(newMode)
                        } catch (e) {
                          console.error('SetAppMode failed:', e)
                          setModeConfirm(null)
                          return
                        }
                        const updated = { ...settings, appMode: newMode }
                        setSettings(updated)
                        setAppMode(newMode)
                        setModeConfirm(null)
                        // Switch away from now-hidden pages.
                        if (newMode !== 'reseller' && RESELLER_ONLY_PAGES.has(activeTool)) {
                          setActiveTool('home')
                        }
                        if (newMode !== 'personal' && PERSONAL_ONLY_PAGES.has(activeTool)) {
                          setActiveTool('home')
                        }
                      }}
                      className="px-3 py-1 text-xs bg-primary text-primary-foreground rounded hover:bg-primary/90"
                    >
                      {t('settings.data.confirm')}
                    </button>
                  </div>
                </div>
              )}
            </div>

            <div className="border-t border-border pt-4">
              <SettingRow label={t('settings.userLevel')} description={t('settings.userLevelDesc')}>
                <div className="flex rounded-md border border-border overflow-hidden">
                  {(['beginner', 'regular', 'power'] as const).map((level) => (
                    <button
                      key={level}
                      onClick={async () => {
                        const updated = { ...settings, userLevel: level }
                        setSettings(updated)
                        setUserLevel(level)
                        try {
                          await SaveAppSettings(appconfig.AppSettings.createFrom(updated))
                        } catch { /* ignore */ }
                      }}
                      className={cn(
                        'px-3 py-1.5 text-sm font-medium transition-colors',
                        (settings as any).userLevel === level
                          ? 'bg-primary text-primary-foreground'
                          : 'bg-muted text-muted-foreground hover:text-foreground'
                      )}
                    >
                      {t(`settings.level.${level}`)}
                    </button>
                  ))}
                </div>
              </SettingRow>
            </div>

            <StartupPerformanceCard />
          </div>
        )}

        {activeTab === 'providers' && (
          <div className="space-y-6">
            <CustomProvidersSection />
            <div className="border-t border-border pt-4">
              <ModelHealthMatrix includeCustom />
            </div>
          </div>
        )}

        {activeTab === 'proxy' && (
          <div className="space-y-6">
            <UpstreamProxySection />
            <div className="pt-4 border-t border-border">
              <p className="text-xs text-muted-foreground">
                {t('settings.proxy.movedNotice')}
              </p>
            </div>
          </div>
        )}

        {activeTab === 'notify' && <NotifyTab />}

        {activeTab === 'update' && (
          <div className="space-y-6">
            <SettingRow label={t('settings.update.autoCheck')} description={t('settings.update.autoCheckDesc')}>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={settings.autoUpdate}
                  onChange={(e) => setSettings({ ...settings, autoUpdate: e.target.checked })}
                  className="w-4 h-4 accent-primary"
                />
                <span className="text-sm">{settings.autoUpdate ? t('settings.update.enabled') : t('settings.update.disabled')}</span>
              </label>
            </SettingRow>
            <p className="text-xs text-muted-foreground">
              {t('settings.update.toolUpdateHint')}
            </p>
          </div>
        )}

        {activeTab === 'backup' && <BackupRestoreCard />}

        {activeTab === 'data' && (
          <div className="space-y-6">
            <div className="p-4 border border-border rounded-md space-y-3">
              <h3 className="text-sm font-medium">{t('settings.data.title')}</h3>
              <p className="text-xs text-muted-foreground">
                {t('settings.data.warning')}
              </p>
              <div className="space-y-2 pt-2 border-t border-border">
                <DangerButton
                  label={t('settings.data.clearSnapshots')}
                  description={t('settings.data.clearSnapshotsDesc')}
                  onConfirm={async () => {
                    const count = await ClearAllSnapshots()
                    return t('settings.data.snapshotsCleared', { count })
                  }}
                />
                <DangerButton
                  label={t('settings.data.clearPrompts')}
                  description={t('settings.data.clearPromptsDesc')}
                  onConfirm={async () => {
                    const count = await ClearAllUserPrompts()
                    return t('settings.data.promptsCleared', { count })
                  }}
                />
              </div>
            </div>
            <div className="p-4 border border-border rounded-md space-y-3">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium">{t('settings.data.rerunSetup')}</p>
                  <p className="text-xs text-muted-foreground">{t('settings.data.rerunSetupDesc')}</p>
                </div>
                <button
                  onClick={async () => {
                    await SaveAppSettings(appconfig.AppSettings.createFrom({ ...settings, onboardingCompleted: false }))
                    window.location.reload()
                  }}
                  className="px-3 py-1.5 text-xs border border-border rounded hover:bg-muted transition-colors"
                >
                  {t('settings.data.rerunSetup')}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function InfoBadge({ label, value, title }: { label: string; value: string; title?: string }) {
  return (
    <div className="min-w-0" title={title}>
      <p className="uppercase tracking-wider text-muted-foreground text-[10px]">{label}</p>
      <p className="font-mono truncate">{value}</p>
    </div>
  )
}

function SettingRow({ label, description, children }: {
  label: string
  description: string
  children: React.ReactNode
}) {
  return (
    <div className="flex items-center justify-between">
      <div>
        <p className="text-sm font-medium">{label}</p>
        <p className="text-xs text-muted-foreground">{description}</p>
      </div>
      <div className="flex items-center gap-2">{children}</div>
    </div>
  )
}

// NotifyTab renders the per-transport settings for outbound push (currently
// Feishu only — Telegram / Slack stub options stay disabled until those
// transports land). Loads + saves through `notifyApi` direct-bridge calls
// so we don't depend on `wails generate module` having picked up the new
// bindings, which has been unreliable on this repo.
function NotifyTab() {
  const { t } = useTranslation()
  const addToast = useToastStore((s) => s.addToast)
  const [cfg, setCfg] = useState<NotifyConfig>(DEFAULT_NOTIFY_CONFIG)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [recent, setRecent] = useState<NotifyEvent[]>([])
  const [recentLoading, setRecentLoading] = useState(false)

  const reloadRecent = useCallback(async () => {
    setRecentLoading(true)
    try {
      setRecent(await getRecentNotifications())
    } catch {
      // Recent list is best-effort — silent on failure.
    } finally {
      setRecentLoading(false)
    }
  }, [])

  useEffect(() => {
    getNotifyConfig()
      .then((loaded) => {
        // Backfill nested fields so a partially-populated server response
        // doesn't render the form with undefined values.
        setCfg({
          ...DEFAULT_NOTIFY_CONFIG,
          ...loaded,
          feishu: { ...DEFAULT_NOTIFY_CONFIG.feishu, ...(loaded.feishu ?? {}) },
          rules: { ...DEFAULT_NOTIFY_CONFIG.rules, ...(loaded.rules ?? {}) },
        })
      })
      .catch(() => {})
      .finally(() => setLoading(false))
    reloadRecent()
  }, [reloadRecent])

  const handleSave = async () => {
    setSaving(true)
    try {
      await saveNotifyConfig(cfg)
      addToast('success', t('settings.notify.saved', '通知设置已保存'))
      reloadRecent()
    } catch (err) {
      addToast('error', t('settings.notify.saveFailed', '保存失败') + ': ' + classifyError(err).message)
    } finally {
      setSaving(false)
    }
  }

  const handleTest = async () => {
    setTesting(true)
    try {
      await testNotify()
      addToast('success', t('settings.notify.testSent', '已推送测试卡片,请在 Feishu 群确认'))
      reloadRecent()
    } catch (err) {
      addToast('error', t('settings.notify.testFailed', '测试失败') + ': ' + classifyError(err).message)
    } finally {
      setTesting(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  const webhookEmpty = cfg.feishu.webhookUrl.trim() === ''
  const httpsBad = !webhookEmpty && !/^https:\/\//i.test(cfg.feishu.webhookUrl.trim())

  return (
    <div className="space-y-6">
      {/* Master toggle */}
      <div className="flex items-start justify-between p-3 rounded-md border border-border bg-muted/30">
        <div className="flex gap-2">
          <Bell className="h-4 w-4 mt-0.5 text-primary" />
          <div>
            <p className="text-sm font-medium">{t('settings.notify.enable', '启用远程推送')}</p>
            <p className="text-xs text-muted-foreground">
              {t('settings.notify.enableDesc', '工具卡住 / 任务完成 / 危险命令时推送到聊天软件')}
            </p>
          </div>
        </div>
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            className="w-4 h-4 accent-primary"
            checked={cfg.enabled}
            onChange={(e) => setCfg({ ...cfg, enabled: e.target.checked })}
          />
        </label>
      </div>

      {/* Transport picker */}
      <div className="space-y-2">
        <p className="text-sm font-medium">{t('settings.notify.transport', '推送渠道')}</p>
        <div className="grid grid-cols-2 gap-2">
          <TransportPill name="Feishu / 飞书" selected disabled={!cfg.enabled} />
          <TransportPill name={'Telegram (' + t('settings.notify.comingSoon', '即将支持') + ')'} selected={false} disabled />
          <TransportPill name={'Slack (' + t('settings.notify.comingSoon', '即将支持') + ')'} selected={false} disabled />
          <TransportPill name={'DingTalk (' + t('settings.notify.comingSoon', '即将支持') + ')'} selected={false} disabled />
        </div>
      </div>

      {/* Feishu config block */}
      <div className="space-y-3 p-3 border border-border rounded-md">
        <div>
          <label className="text-xs font-medium block mb-1">
            {t('settings.notify.feishu.webhookUrl', 'Webhook URL')}
            <span className="text-red-500 ml-0.5">*</span>
          </label>
          <input
            type="text"
            value={cfg.feishu.webhookUrl}
            placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/…"
            disabled={!cfg.enabled}
            onChange={(e) =>
              setCfg({ ...cfg, feishu: { ...cfg.feishu, webhookUrl: e.target.value } })
            }
            className="w-full px-2 py-1.5 text-xs font-mono bg-muted border border-border rounded focus:outline-none focus:ring-1 focus:ring-primary disabled:opacity-50"
          />
          {httpsBad && (
            <p className="text-[11px] text-red-500 mt-1">
              {t('settings.notify.feishu.httpsRequired', 'Webhook URL 必须是 https://')}
            </p>
          )}
          <p className="text-[11px] text-muted-foreground mt-1">
            {t('settings.notify.feishu.urlHint', '在飞书群里 → 设置 → 群机器人 → 添加机器人 → 自定义机器人 → 复制 webhook 地址')}
          </p>
        </div>
        <div>
          <label className="text-xs font-medium block mb-1">
            {t('settings.notify.feishu.secret', '签名 Secret (可选)')}
          </label>
          <input
            type="password"
            value={cfg.feishu.secret ?? ''}
            disabled={!cfg.enabled}
            onChange={(e) =>
              setCfg({ ...cfg, feishu: { ...cfg.feishu, secret: e.target.value } })
            }
            className="w-full px-2 py-1.5 text-xs font-mono bg-muted border border-border rounded focus:outline-none focus:ring-1 focus:ring-primary disabled:opacity-50"
          />
          <p className="text-[11px] text-muted-foreground mt-1">
            {t('settings.notify.feishu.secretHint', '若机器人开启了"签名校验",在此填入对应 Secret')}
          </p>
        </div>
      </div>

      {/* Rule toggles */}
      <div className="space-y-2 p-3 border border-border rounded-md">
        <p className="text-sm font-medium">{t('settings.notify.rules', '推送时机')}</p>
        <RuleRow
          label={t('settings.notify.rules.stuck', '工具调用偏长 / 卡住')}
          desc={t('settings.notify.rules.stuckDesc', '单个 tool_use 超过 60 秒先发橙色卡;超过 5 分钟升级为红色')}
          checked={cfg.rules.notifyStuck}
          disabled={!cfg.enabled}
          onChange={(v) => setCfg({ ...cfg, rules: { ...cfg.rules, notifyStuck: v } })}
        />
        <RuleRow
          label={t('settings.notify.rules.done', '任务完成 / Claude 等你')}
          desc={t('settings.notify.rules.doneDesc', '会话从活动转为静默超过 5 分钟时推送绿色卡片')}
          checked={cfg.rules.notifyDone}
          disabled={!cfg.enabled}
          onChange={(v) => setCfg({ ...cfg, rules: { ...cfg.rules, notifyDone: v } })}
        />
      </div>

      {/* Action buttons */}
      <div className="flex gap-2">
        <button
          onClick={handleSave}
          disabled={saving || (cfg.enabled && !webhookEmpty && httpsBad)}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          {saving ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Save className="h-3.5 w-3.5" />}
          {t('settings.save')}
        </button>
        <button
          onClick={handleTest}
          disabled={testing || !cfg.enabled || webhookEmpty}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-md border border-border hover:bg-muted disabled:opacity-50"
          title={!cfg.enabled || webhookEmpty ? t('settings.notify.testDisabledHint', '请先开启并填写 Webhook URL,保存后再测试') : ''}
        >
          {testing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Send className="h-3.5 w-3.5" />}
          {t('settings.notify.testButton', '测试推送')}
        </button>
      </div>

      {/* Recent push log */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <p className="text-sm font-medium">{t('settings.notify.recent', '最近推送')}</p>
          <button
            onClick={reloadRecent}
            disabled={recentLoading}
            className="text-xs text-muted-foreground hover:text-foreground"
          >
            {recentLoading ? '…' : t('settings.notify.refresh', '刷新')}
          </button>
        </div>
        {recent.length === 0 ? (
          <p className="text-xs text-muted-foreground italic">
            {t('settings.notify.recentEmpty', '尚无推送记录 — 启用后会显示最近 30 条')}
          </p>
        ) : (
          <div className="border border-border rounded-md overflow-hidden">
            <table className="w-full text-xs">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-2 py-1.5 text-left font-medium">{t('settings.notify.col.time', '时间')}</th>
                  <th className="px-2 py-1.5 text-left font-medium">{t('settings.notify.col.severity', '级别')}</th>
                  <th className="px-2 py-1.5 text-left font-medium">{t('settings.notify.col.title', '标题')}</th>
                </tr>
              </thead>
              <tbody>
                {[...recent].reverse().map((ev) => (
                  <tr key={ev.id} className="border-t border-border">
                    <td className="px-2 py-1 font-mono text-[11px] text-muted-foreground">
                      {new Date(ev.time).toLocaleString()}
                    </td>
                    <td className="px-2 py-1">
                      <SeverityChip sev={ev.severity} />
                    </td>
                    <td className="px-2 py-1 truncate max-w-md" title={ev.body}>{ev.title}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}

function TransportPill({ name, selected, disabled }: { name: string; selected: boolean; disabled?: boolean }) {
  return (
    <div
      className={cn(
        'px-3 py-2 text-xs rounded-md border flex items-center gap-2',
        selected ? 'border-primary bg-primary/10 text-primary' : 'border-border text-muted-foreground',
        disabled && 'opacity-50 cursor-not-allowed'
      )}
    >
      <input type="radio" checked={selected} readOnly disabled={disabled} className="accent-primary" />
      {name}
    </div>
  )
}

function RuleRow({ label, desc, checked, disabled, onChange }: {
  label: string
  desc: string
  checked: boolean
  disabled?: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <label className={cn('flex items-start gap-2 cursor-pointer', disabled && 'opacity-50 cursor-not-allowed')}>
      <input
        type="checkbox"
        className="mt-0.5 w-4 h-4 accent-primary"
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange(e.target.checked)}
      />
      <div>
        <p className="text-xs font-medium">{label}</p>
        <p className="text-[11px] text-muted-foreground">{desc}</p>
      </div>
    </label>
  )
}

function SeverityChip({ sev }: { sev: NotifyEvent['severity'] }) {
  const styles: Record<NotifyEvent['severity'], string> = {
    info: 'bg-blue-500/15 text-blue-500',
    success: 'bg-green-500/15 text-green-500',
    warning: 'bg-orange-500/15 text-orange-500',
    error: 'bg-red-500/15 text-red-500',
  }
  return <span className={cn('px-1.5 py-0.5 rounded text-[10px] font-medium', styles[sev])}>{sev}</span>
}

function DangerButton({ label, description, onConfirm }: {
  label: string
  description: string
  onConfirm?: () => Promise<string>
}) {
  const [confirming, setConfirming] = useState(false)
  const [executing, setExecuting] = useState(false)
  const [result, setResult] = useState('')
  const { t } = useTranslation()

  const handleConfirm = async () => {
    if (!onConfirm) {
      setConfirming(false)
      return
    }
    setExecuting(true)
    try {
      const msg = await onConfirm()
      setResult(msg)
      setTimeout(() => setResult(''), 3000)
    } catch (err) {
      setResult(classifyError(err).message)
    } finally {
      setExecuting(false)
      setConfirming(false)
    }
  }

  return (
    <div className="flex items-center justify-between">
      <div>
        <p className="text-sm font-medium">{label}</p>
        <p className="text-xs text-muted-foreground">
          {result || description}
        </p>
      </div>
      {confirming ? (
        <div className="flex gap-2">
          <button
            onClick={() => setConfirming(false)}
            disabled={executing}
            className="px-3 py-1 text-xs border border-border rounded hover:bg-muted disabled:opacity-50"
          >
            {t('settings.data.cancel')}
          </button>
          <button
            onClick={handleConfirm}
            disabled={executing}
            className="px-3 py-1 text-xs bg-red-500 text-white rounded hover:bg-red-600 disabled:opacity-50"
          >
            {executing ? '...' : t('settings.data.confirm')}
          </button>
        </div>
      ) : (
        <button
          onClick={() => setConfirming(true)}
          className="px-3 py-1.5 text-xs border border-red-500/30 text-red-500 rounded hover:bg-red-500/10 transition-colors"
        >
          {label}
        </button>
      )}
    </div>
  )
}
