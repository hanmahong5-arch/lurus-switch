import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { LogIn, LogOut, Loader2, CheckCircle2, AlertCircle, ChevronDown, ChevronRight, Shield } from 'lucide-react'
import { cn } from '../lib/utils'
import { useAuthStore } from '../stores/authStore'

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

  useEffect(() => {
    load()
  }, [load])

  if (authState.is_logged_in && authState.user) {
    // --- Logged in state ---
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
              {(authState.user.name || authState.user.email || '?').charAt(0).toUpperCase()}
            </div>
          )}
          <div className="flex-1 min-w-0">
            {authState.user.name && (
              <p className="text-sm font-medium truncate">{authState.user.name}</p>
            )}
            {authState.user.email && (
              <p className="text-xs text-muted-foreground truncate">{authState.user.email}</p>
            )}
          </div>
        </div>

        {/* Gateway status */}
        <div className="flex items-center justify-between text-xs">
          <span className="text-muted-foreground">{t('auth.gatewayToken', 'API Gateway')}</span>
          {authState.has_gateway_token ? (
            <span className="flex items-center gap-1 text-green-500">
              <CheckCircle2 className="h-3 w-3" />
              {t('auth.provisioned', 'Provisioned')}
            </span>
          ) : (
            <span className="flex items-center gap-1 text-amber-500">
              <Loader2 className="h-3 w-3 animate-spin" />
              {t('auth.provisioning', 'Provisioning...')}
            </span>
          )}
        </div>

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
