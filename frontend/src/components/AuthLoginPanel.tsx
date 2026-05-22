import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { LogIn, LogOut, Loader2, CheckCircle2, AlertCircle, ChevronDown, ChevronRight, Shield, KeyRound } from 'lucide-react'
import { cn } from '../lib/utils'
import { useAuthStore } from '../stores/authStore'
import { GetAppSettings, SaveAppSettings } from '../../wailsjs/go/main/App'
import { appconfig } from '../../wailsjs/go/models'

// Detect the specific "client_id not configured" failure so the panel
// can offer an inline fix instead of dead-ending the user at a Settings
// tab that doesn't actually expose this field.
const isClientIdMissingError = (msg: string | null): boolean =>
  !!msg && /client_id\s+not\s+configured/i.test(msg)

/**
 * AuthLoginPanel — OIDC login panel for the Account > Connection tab.
 *
 * Not logged in: primary "Login with Lurus Account" button + collapsible manual token section.
 * Logged in: user info display + gateway status + logout button.
 * Loading: spinner with "Waiting for browser login..." text.
 */
export function AuthLoginPanel() {
  const { t } = useTranslation()
  const { authState, isLoggingIn, loginError, load, login, logout } = useAuthStore()
  const [showAdvanced, setShowAdvanced] = useState(false)

  // Inline OIDC client_id rescue — opens automatically when login fails
  // on missing config, lets the user fix it without leaving this panel.
  // Personal-mode users have no business poking around Settings for this.
  const [showClientIdForm, setShowClientIdForm] = useState(false)
  const [clientId, setClientId] = useState('')
  const [issuer, setIssuer] = useState('https://auth.lurus.cn')
  const [savingCfg, setSavingCfg] = useState(false)
  const [cfgError, setCfgError] = useState<string | null>(null)

  useEffect(() => {
    load()
  }, [load])

  // When the error indicates client_id is missing, surface the inline
  // form and pre-load whatever's already on disk (if anything).
  useEffect(() => {
    if (!isClientIdMissingError(loginError)) return
    setShowClientIdForm(true)
    GetAppSettings()
      .then((s: any) => {
        if (s?.authClientId) setClientId(String(s.authClientId))
        if (s?.authIssuer) setIssuer(String(s.authIssuer))
      })
      .catch(() => { /* best-effort prefill */ })
  }, [loginError])

  const handleSaveAndLogin = async () => {
    const id = clientId.trim()
    const iss = issuer.trim() || 'https://auth.lurus.cn'
    if (!id) {
      setCfgError(t('auth.clientIdRequired', 'Client ID 不能为空'))
      return
    }
    setSavingCfg(true)
    setCfgError(null)
    try {
      const current = ((await GetAppSettings()) as any) || {}
      const updated = { ...current, authClientId: id, authIssuer: iss }
      await SaveAppSettings(appconfig.AppSettings.createFrom(updated))
      setShowClientIdForm(false)
      await login()
    } catch (e: any) {
      setCfgError(e?.message ?? String(e))
    } finally {
      setSavingCfg(false)
    }
  }

  if (authState.is_logged_in && authState.user) {
    // --- Logged in state ---
    const platform = (authState as any).platform as
      | {
          account_id?: number
          lurus_id?: string
          display_name?: string
          email?: string
          vip_level?: number
          wallet_balance?: number
          wallet_frozen?: number
        }
      | undefined
    const balance = platform?.wallet_balance ?? 0
    const frozen = platform?.wallet_frozen ?? 0
    const displayName =
      platform?.display_name ||
      authState.user.name ||
      authState.user.email ||
      ''

    return (
      <div className="border border-border rounded-lg p-4 bg-card space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium flex items-center gap-2">
            <Shield className="h-4 w-4 text-green-500" />
            {t('auth.lurusAccount', 'Lurus Account')}
          </h3>
          <span className="flex items-center gap-1 text-xs text-green-500">
            <CheckCircle2 className="h-3 w-3" />
            {t('auth.connected', 'Connected')}
          </span>
        </div>

        {/* User info */}
        <div className="flex items-center gap-3">
          {authState.user.picture ? (
            <img
              src={authState.user.picture}
              alt=""
              className="h-8 w-8 rounded-full border border-border"
            />
          ) : (
            <div className="h-8 w-8 rounded-full bg-primary/20 flex items-center justify-center text-xs font-medium text-primary">
              {(displayName || '?').charAt(0).toUpperCase()}
            </div>
          )}
          <div className="flex-1 min-w-0">
            {displayName && (
              <p className="text-sm font-medium truncate">{displayName}</p>
            )}
            {authState.user.email && (
              <p className="text-xs text-muted-foreground truncate">{authState.user.email}</p>
            )}
            {platform?.lurus_id && (
              <p className="text-[10px] text-muted-foreground/70 font-mono mt-0.5">
                {platform.lurus_id}
              </p>
            )}
          </div>
          {platform && platform.vip_level !== undefined && platform.vip_level > 0 && (
            <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/30">
              VIP{platform.vip_level}
            </span>
          )}
        </div>

        {/* Wallet balance — QQ币 style */}
        {platform && (
          <div className="rounded-md border border-border bg-muted/30 px-3 py-2">
            <div className="flex items-baseline justify-between gap-2">
              <span className="text-[11px] text-muted-foreground">
                {t('auth.walletBalance', '钱包余额')}
              </span>
              <span className="text-lg font-semibold tabular-nums">
                ¥{balance.toFixed(2)}
              </span>
            </div>
            {frozen > 0 && (
              <p className="text-[10px] text-muted-foreground mt-0.5">
                {t('auth.walletFrozen', '冻结')}: ¥{frozen.toFixed(2)}
              </p>
            )}
          </div>
        )}

        {/* Gateway status — collapsed when not configured (Personal mode usually) */}
        {authState.has_gateway_token && (
          <div className="flex items-center justify-between text-xs">
            <span className="text-muted-foreground">{t('auth.gatewayToken', 'API Gateway')}</span>
            <span className="flex items-center gap-1 text-green-500">
              <CheckCircle2 className="h-3 w-3" />
              {t('auth.provisioned', 'Provisioned')}
            </span>
          </div>
        )}

        {/* Logout button */}
        <button
          onClick={logout}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors w-full justify-center',
            'border border-border hover:bg-destructive/10 hover:text-destructive hover:border-destructive/30'
          )}
        >
          <LogOut className="h-3.5 w-3.5" />
          {t('auth.logout', 'Logout')}
        </button>
      </div>
    )
  }

  // --- Not logged in state ---
  return (
    <div className="border border-border rounded-lg overflow-hidden bg-card space-y-3">
      {/* Hero gradient band */}
      <div className="px-4 pt-5 pb-4 bg-gradient-to-br from-[hsl(228,99%,65%)] to-[hsl(222,47%,18%)]">
        <div className="flex items-center gap-2 mb-1.5">
          <Shield className="h-4 w-4 text-white/90" />
          <h3 className="text-sm font-semibold text-white">{t('auth.lurusAccount', 'Lurus Account')}</h3>
        </div>
        <p className="text-xs text-white/70">
          {t('auth.loginDescription', 'Login with your Lurus account to access billing, API quotas, and auto-configure all tools.')}
        </p>
      </div>
      <div className="px-4 pb-4 space-y-3">

      {/* Error message */}
      {loginError && (
        <div className="flex items-start gap-2 p-2 rounded-md bg-destructive/10 text-destructive text-xs">
          <AlertCircle className="h-3.5 w-3.5 mt-0.5 shrink-0" />
          <span>{loginError}</span>
        </div>
      )}

      {/* Inline rescue form for the "client_id not configured" case.
          Showing it next to the failure beats sending the user to a
          Settings tab that doesn't actually have this field. */}
      {showClientIdForm && (
        <div className="rounded-md border border-amber-500/30 bg-amber-500/5 p-3 space-y-2">
          <div className="flex items-start gap-2">
            <KeyRound className="h-4 w-4 mt-0.5 shrink-0 text-amber-500" />
            <div className="flex-1 space-y-1">
              <p className="text-xs font-medium text-amber-600 dark:text-amber-400">
                {t('auth.clientIdRescue.title', '需要 OIDC Client ID')}
              </p>
              <p className="text-[11px] text-muted-foreground">
                {t(
                  'auth.clientIdRescue.hint',
                  '不需要登录账号也能用 Switch — 直接配下面的 API key 即可。如果想用 Lurus 账号登录、查计费 / 配额，请向管理员索取 Client ID。',
                )}
              </p>
            </div>
          </div>
          <label className="block">
            <span className="text-[10px] uppercase tracking-wider text-muted-foreground">
              {t('auth.clientId', 'Client ID')}
            </span>
            <input
              value={clientId}
              onChange={(e) => setClientId(e.target.value)}
              placeholder="lurus-switch-cli"
              className="mt-1 w-full px-2 py-1.5 rounded border border-border bg-background text-xs font-mono"
              autoComplete="off"
              spellCheck={false}
            />
          </label>
          <label className="block">
            <span className="text-[10px] uppercase tracking-wider text-muted-foreground">
              {t('auth.issuer', 'Issuer')}
            </span>
            <input
              value={issuer}
              onChange={(e) => setIssuer(e.target.value)}
              placeholder="https://auth.lurus.cn"
              className="mt-1 w-full px-2 py-1.5 rounded border border-border bg-background text-xs font-mono"
              autoComplete="off"
              spellCheck={false}
            />
          </label>
          {cfgError && (
            <p className="text-[11px] text-destructive">{cfgError}</p>
          )}
          <div className="flex items-center gap-2 pt-1">
            <button
              onClick={handleSaveAndLogin}
              disabled={savingCfg || !clientId.trim()}
              className="px-3 py-1.5 rounded-md bg-amber-500 hover:bg-amber-400 text-white text-xs font-medium disabled:opacity-50 inline-flex items-center gap-1"
            >
              {savingCfg ? <Loader2 className="h-3 w-3 animate-spin" /> : <LogIn className="h-3 w-3" />}
              {t('auth.saveAndLogin', '保存并登录')}
            </button>
            <button
              onClick={() => setShowClientIdForm(false)}
              className="px-2 py-1.5 rounded-md border border-border text-xs hover:bg-muted"
            >
              {t('common.dismiss', '关闭')}
            </button>
          </div>
        </div>
      )}

      {/* Login button */}
      <button
        onClick={login}
        disabled={isLoggingIn}
        className={cn(
          'flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors w-full justify-center',
          'bg-primary text-primary-foreground hover:bg-primary/90',
          'disabled:opacity-50 disabled:cursor-not-allowed'
        )}
      >
        {isLoggingIn ? (
          <>
            <Loader2 className="h-4 w-4 animate-spin" />
            {t('auth.waitingForBrowser', 'Waiting for browser login...')}
          </>
        ) : (
          <>
            <LogIn className="h-4 w-4" />
            {t('auth.loginButton', 'Login with Lurus Account')}
          </>
        )}
      </button>

      {/* Collapsible advanced section */}
      <button
        onClick={() => setShowAdvanced(!showAdvanced)}
        className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
      >
        {showAdvanced ? (
          <ChevronDown className="h-3 w-3" />
        ) : (
          <ChevronRight className="h-3 w-3" />
        )}
        {t('auth.advancedManualToken', 'Advanced: Manual Token')}
      </button>

      {showAdvanced && (
        <p className="text-xs text-muted-foreground pl-4 border-l-2 border-border">
          {t('auth.manualTokenHint', 'If you prefer manual setup, configure the API endpoint and token in the section below.')}
        </p>
      )}
      </div>
    </div>
  )
}
