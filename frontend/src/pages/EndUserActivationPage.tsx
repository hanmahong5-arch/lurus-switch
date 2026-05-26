import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, KeyRound, Check, AlertTriangle, Shield, RefreshCw, Mail, Eraser } from 'lucide-react'
import { Button, Card } from '../components/ui'
import { ActivateRedemption, GetDeviceFingerprint, GetAppSettings, SetAppMode } from '../../wailsjs/go/main/App'

interface Props {
  hubURL?: string
  onActivated: () => void
}

// Maps Hub-classified error kinds (returned suffixed as `[kind=...]` from
// the Go binding) to localized headlines. The hint follows in the
// secondary line — no need for verbose i18n keys per kind. Kinds added
// in Wave 5 W5.2 (tenant_/version_/clock_skew/...) extend the catalogue
// so the EndUser sees actionable copy instead of a generic "server".
const KIND_TO_KEY: Record<string, string> = {
  // Pre-existing — keep IDs stable for upstream Hub error contract.
  invalid_input: 'enduser.error.invalidInput',
  network: 'enduser.error.network',
  code_not_found: 'enduser.error.notFound',
  code_used: 'enduser.error.used',
  code_expired: 'enduser.error.expired',
  code_disabled: 'enduser.error.disabled',
  endpoint_absent: 'enduser.error.endpointAbsent',
  server: 'enduser.error.server',
  // Wave 5 W5.2 — protocol-level / transport-level / device-context
  // failures that previously collapsed into "server".
  rate_limit: 'enduser.error.rateLimit',
  timeout: 'enduser.error.timeout',
  mismatched_user: 'enduser.error.mismatchedUser',
  tenant_disabled: 'enduser.error.tenantDisabled',
  tenant_quota_exhausted: 'enduser.error.tenantQuotaExhausted',
  activation_paused: 'enduser.error.activationPaused',
  version_too_old: 'enduser.error.versionTooOld',
  clock_skew: 'enduser.error.clockSkew',
  unsupported_region: 'enduser.error.unsupportedRegion',
  multiple_redemptions: 'enduser.error.multipleRedemptions',
  unknown: 'enduser.error.unknown',
}

// EndUser activation page — the first thing users see when launching a
// white-label build that hasn't been activated yet. The Hub URL is
// embedded at packaging time, so this page only collects the redemption
// code; we display the URL read-only so the user can verify they were
// given the right installer.
export function EndUserActivationPage({ hubURL, onActivated }: Props) {
  const { t } = useTranslation()
  const [code, setCode] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<{ kind: string; message: string } | null>(null)
  const [fingerprint, setFingerprint] = useState('')
  // Distributor support email — read from white-label settings so the
  // "联系经销商" CTA goes to the right inbox, not stock Lurus support.
  const [supportEmail, setSupportEmail] = useState('')

  useEffect(() => {
    GetDeviceFingerprint().then(setFingerprint).catch(() => setFingerprint(''))
    GetAppSettings()
      .then((s) => setSupportEmail((s as any)?.brandSupportEmail ?? ''))
      .catch(() => setSupportEmail(''))
  }, [])

  const handleSubmit = async () => {
    setSubmitting(true)
    setError(null)
    try {
      await ActivateRedemption(code.trim())
      onActivated()
    } catch (e) {
      const raw = String(e)
      // The Go binding suffixes the kind so the UI doesn't have to parse
      // localized error text. Format: "<message> [kind=<id>]"
      const m = raw.match(/^(.*)\s\[kind=([a-z_]+)\]$/i)
      if (m) {
        setError({ message: m[1], kind: m[2] })
      } else {
        setError({ message: raw, kind: 'server' })
      }
    } finally {
      setSubmitting(false)
    }
  }

  const errorHint = error ? t(KIND_TO_KEY[error.kind] ?? 'enduser.error.server', error.message) : ''

  // Restrict: 4-32 chars, alnum + dash. Hub keys are typically 16-24 hex
  // but we leave headroom for future formats.
  const codeOK = /^[A-Za-z0-9-]{4,32}$/.test(code.trim())

  return (
    <div className="h-screen flex flex-col items-center justify-center bg-background text-foreground p-6">
      <div className="w-full max-w-md">
        <header className="flex items-center gap-3 mb-6">
          <div className="h-10 w-10 rounded-xl bg-primary/15 border border-primary/40 flex items-center justify-center shadow-glow-orange">
            <KeyRound className="h-5 w-5 text-primary" />
          </div>
          <div>
            <p className="font-mono text-[10px] uppercase tracking-[0.18em] text-primary mb-0.5">
              [ ENDUSER · ACTIVATION ]
            </p>
            <h1 className="text-xl font-semibold">{t('enduser.activate.title', '激活你的服务')}</h1>
            <p className="text-xs text-muted-foreground">
              {t('enduser.activate.subtitle', '输入经销商提供的激活码')}
            </p>
          </div>
        </header>

        {hubURL && (
          <Card variant="recessed" className="px-3 py-2 text-xs text-muted-foreground mb-4 font-mono tabular-nums">
            <span className="text-foreground/70">Hub:</span> {hubURL}
          </Card>
        )}

        <label className="block text-sm mb-1">
          <span className="text-muted-foreground">{t('enduser.activate.code', '激活码')}</span>
        </label>
        <input
          autoFocus
          spellCheck={false}
          autoComplete="off"
          className="w-full rounded border border-border bg-background px-3 py-2 text-sm font-mono mb-3"
          value={code}
          onChange={(e) => setCode(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && codeOK && !submitting) handleSubmit()
          }}
          placeholder="ABCD-1234-EFGH-5678"
          disabled={submitting}
        />

        {error && (
          <Card variant="default" className="border-red-500/30 bg-red-500/10 text-red-300 text-xs px-3 py-2 mb-3">
            <div className="flex items-start gap-2">
              <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
              <div className="flex-1">
                <div className="font-mono text-red-400">▸ {errorHint || error.message}</div>
                {errorHint && errorHint !== error.message && (
                  <div className="text-red-300/70 mt-0.5 font-mono">{error.message}</div>
                )}
              </div>
            </div>
            <ErrorActions
              kind={error.kind}
              supportEmail={supportEmail}
              code={code}
              hubURL={hubURL}
              onRetry={handleSubmit}
              onClear={() => { setCode(''); setError(null) }}
            />
          </Card>
        )}

        <Button
          onClick={handleSubmit}
          disabled={!codeOK || submitting}
          loading={submitting}
          icon={!submitting ? <Check className="h-4 w-4" /> : undefined}
          className="w-full justify-center bg-emerald-600 hover:bg-emerald-500 ring-emerald-500/40"
        >
          {t('enduser.activate.submit', '激活')}
        </Button>

        <div
          className="mt-6 pt-4 border-t border-border/60 flex items-center gap-2 text-[11px] text-muted-foreground/80"
          onClick={async (e) => {
            // Dev-only escape hatch: Shift+Click on the fingerprint row drops
            // the app back to Personal mode. The white-label premise forbids
            // an in-product mode switch in production builds, hence the
            // import.meta.env.DEV gate.
            if (!import.meta.env.DEV || !e.shiftKey) return
            if (!window.confirm('[DEV] Switch back to Personal mode?')) return
            try {
              await SetAppMode('personal')
              window.location.reload()
            } catch (err) {
              window.alert(String(err))
            }
          }}
          title={import.meta.env.DEV ? 'Shift+Click → switch to Personal (dev only)' : undefined}
        >
          <Shield className="h-3.5 w-3.5" />
          <span>
            {t('enduser.activate.deviceLabel', '设备指纹')}:{' '}
            <span className="font-mono">{fingerprint || '...'}</span>
          </span>
        </div>
        <p className="mt-2 text-[11px] text-muted-foreground/60 leading-relaxed">
          {t(
            'enduser.activate.deviceNote',
            '激活码与本机指纹绑定，更换设备需重新申请。',
          )}
        </p>

        <p className="mt-4 text-[11px] text-muted-foreground/60 leading-relaxed">
          {t(
            'enduser.activate.help',
            '没有激活码？请联系销售给您安装包的经销商。',
          )}
        </p>
      </div>
    </div>
  )
}

// ErrorActions renders one or two CTAs tailored to the failure kind so
// the user has somewhere to go from a red box. Kinds without a sensible
// recovery (e.g. server) only get a generic retry.
export function ErrorActions({
  kind, supportEmail, code, hubURL, onRetry, onClear,
}: {
  kind: string
  supportEmail: string
  code: string
  hubURL?: string
  onRetry: () => void
  onClear: () => void
}) {
  const { t } = useTranslation()
  const mailHref = supportEmail
    ? `mailto:${supportEmail}?subject=${encodeURIComponent(t('enduser.error.action.mailSubject', '激活码无法使用'))}&body=${encodeURIComponent(
        t('enduser.error.action.mailBody', '激活码：{{code}}\nHub：{{hub}}\n失败原因（kind）：{{kind}}\n', {
          code, hub: hubURL ?? '', kind,
        }),
      )}`
    : ''

  // Map each kind to its preferred recovery actions.
  // Wave 5: extended kinds (rate_limit / timeout / clock_skew → retry;
  // tenant_* / activation_paused / multiple_redemptions → contact;
  // version_too_old / unsupported_region → contact + custom hint).
  const showRetry =
    kind === 'network' ||
    kind === 'server' ||
    kind === 'endpoint_absent' ||
    kind === 'rate_limit' ||
    kind === 'timeout' ||
    kind === 'clock_skew' ||
    kind === 'unknown'
  const showContact =
    kind === 'code_used' ||
    kind === 'code_expired' ||
    kind === 'code_disabled' ||
    kind === 'code_not_found' ||
    kind === 'tenant_disabled' ||
    kind === 'tenant_quota_exhausted' ||
    kind === 'activation_paused' ||
    kind === 'mismatched_user' ||
    kind === 'multiple_redemptions' ||
    kind === 'version_too_old' ||
    kind === 'unsupported_region'
  const showClear = kind === 'invalid_input' || kind === 'code_not_found'

  if (!showRetry && !showContact && !showClear) return null

  return (
    <div className="mt-2 ml-6 flex items-center gap-2 flex-wrap">
      {showRetry && (
        <button
          onClick={onRetry}
          className="inline-flex items-center gap-1 px-2 py-1 rounded border border-red-400/40 text-red-200 hover:bg-red-500/10 text-[11px]"
        >
          <RefreshCw className="h-3 w-3" />
          {t('enduser.error.action.retry', '重试')}
        </button>
      )}
      {showClear && (
        <button
          onClick={onClear}
          className="inline-flex items-center gap-1 px-2 py-1 rounded border border-red-400/40 text-red-200 hover:bg-red-500/10 text-[11px]"
        >
          <Eraser className="h-3 w-3" />
          {t('enduser.error.action.clear', '清空重输')}
        </button>
      )}
      {showContact && (
        mailHref ? (
          <a
            href={mailHref}
            className="inline-flex items-center gap-1 px-2 py-1 rounded border border-red-400/40 text-red-200 hover:bg-red-500/10 text-[11px]"
          >
            <Mail className="h-3 w-3" />
            {t('enduser.error.action.contact', '联系经销商')}
          </a>
        ) : (
          <span className="text-[11px] text-red-300/70 italic">
            {t('enduser.error.action.noSupport', '请联系您拿到这份安装包的经销商')}
          </span>
        )
      )}
    </div>
  )
}
