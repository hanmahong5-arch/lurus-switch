import { useEffect, useState, useCallback } from 'react'
import { Save, Loader2, CheckCircle2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useClassifiedError } from '../lib/useClassifiedError'
import { InlineError } from '../components/InlineError'
import { classifyError } from '../lib/errorClassifier'
import { GetAppSettings, SaveAppSettings, ClearAllSnapshots, ClearAllUserPrompts } from '../../wailsjs/go/main/App'
import { appconfig } from '../../wailsjs/go/models'
import { useConfigStore, type AppMode, type UserLevel } from '../stores/configStore'
import { PROMOTER_ONLY_PAGES } from '../components/Sidebar'

type Tab = 'appearance' | 'proxy' | 'update' | 'data'

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
  const [modeConfirm, setModeConfirm] = useState<AppMode | null>(null)
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
    { id: 'proxy', label: t('settings.tabs.proxy') },
    { id: 'update', label: t('settings.tabs.update') },
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
          <button
            onClick={handleSave}
            disabled={saving}
            className={cn(
              'flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium transition-colors',
              'bg-primary text-primary-foreground hover:bg-primary/90',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : saved ? <CheckCircle2 className="h-4 w-4" /> : <Save className="h-4 w-4" />}
            {saved ? t('settings.saved') : t('settings.save')}
          </button>
        </div>

        {error && (
          <InlineError
            category={error.category}
            message={error.message}
            details={error.details}
            onDismiss={clearError}
          />
        )}

        {/* Tabs */}
        <div className="border-b border-border">
          <nav className="flex gap-1">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={cn(
                  'px-4 py-2 text-sm font-medium border-b-2 transition-colors',
                  activeTab === tab.id
                    ? 'border-primary text-primary'
                    : 'border-transparent text-muted-foreground hover:text-foreground'
                )}
              >
                {tab.label}
              </button>
            ))}
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
                  <button
                    onClick={() => {
                      if (settings.appMode !== 'user') setModeConfirm('user')
                    }}
                    className={cn(
                      'px-3 py-1.5 text-sm font-medium transition-colors',
                      settings.appMode === 'user'
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-muted text-muted-foreground hover:text-foreground'
                    )}
                  >
                    {t('settings.modeUser')}
                  </button>
                  <button
                    onClick={() => {
                      if (settings.appMode !== 'promoter') setModeConfirm('promoter')
                    }}
                    className={cn(
                      'px-3 py-1.5 text-sm font-medium transition-colors',
                      settings.appMode === 'promoter'
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-muted text-muted-foreground hover:text-foreground'
                    )}
                  >
                    {t('settings.modePromoter')}
                  </button>
                </div>
              </SettingRow>

              {/* Mode switch confirmation dialog */}
              {modeConfirm && (
                <div className="mt-3 p-3 rounded-md border border-primary/30 bg-primary/5">
                  <p className="text-sm font-medium">
                    {t('settings.modeSwitchConfirm', {
                      mode: modeConfirm === 'user' ? t('settings.modeUser') : t('settings.modePromoter'),
                    })}
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">
                    {modeConfirm === 'user'
                      ? t('settings.modeSwitchUserBody')
                      : t('settings.modeSwitchPromoterBody')}
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
                        const updated = { ...settings, appMode: newMode }
                        setSettings(updated)
                        setAppMode(newMode)
                        setModeConfirm(null)
                        // Navigate away from promoter-only pages when switching to user mode
                        if (newMode === 'user' && PROMOTER_ONLY_PAGES.has(activeTool)) {
                          setActiveTool('home')
                        }
                        try {
                          await SaveAppSettings(appconfig.AppSettings.createFrom(updated))
                        } catch { /* ignore */ }
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
          </div>
        )}

        {activeTab === 'proxy' && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {t('settings.proxy.movedNotice')}
            </p>
            <div className="p-4 bg-muted/30 rounded-md border border-border">
              <p className="text-xs text-muted-foreground">
                {t('settings.proxy.hint')}
              </p>
            </div>
          </div>
        )}

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
