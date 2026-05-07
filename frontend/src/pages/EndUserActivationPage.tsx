import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, KeyRound, Check, AlertTriangle, Shield } from 'lucide-react'
import { ActivateRedemption, GetDeviceFingerprint, SetAppMode } from '../../wailsjs/go/main/App'

interface Props {
  hubURL?: string
  onActivated: () => void
}

// Maps Hub-classified error kinds (returned suffixed as `[kind=...]` from
// the Go binding) to localized headlines. The hint follows in the
// secondary line — no need for verbose i18n keys per kind.
const KIND_TO_KEY: Record<string, string> = {
  invalid_input: 'enduser.error.invalidInput',
  network: 'enduser.error.network',
  code_not_found: 'enduser.error.notFound',
  code_used: 'enduser.error.used',
  code_expired: 'enduser.error.expired',
  code_disabled: 'enduser.error.disabled',
  endpoint_absent: 'enduser.error.endpointAbsent',
  server: 'enduser.error.server',
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

  useEffect(() => {
    GetDeviceFingerprint().then(setFingerprint).catch(() => setFingerprint(''))
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
          <div className="h-10 w-10 rounded-xl bg-emerald-500/15 flex items-center justify-center">
            <KeyRound className="h-5 w-5 text-emerald-400" />
          </div>
          <div>
            <h1 className="text-xl font-semibold">{t('enduser.activate.title', '激活你的服务')}</h1>
            <p className="text-xs text-muted-foreground">
              {t('enduser.activate.subtitle', '输入经销商提供的激活码')}
            </p>
          </div>
        </header>

        {hubURL && (
          <div className="rounded-md border border-border bg-muted/20 px-3 py-2 text-xs text-muted-foreground mb-4 font-mono">
            <span className="text-foreground/70">Hub:</span> {hubURL}
          </div>
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
          <div className="rounded-md border border-red-500/30 bg-red-950/20 text-red-200 text-xs px-3 py-2 mb-3 flex items-start gap-2">
            <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
            <div>
              <div className="font-medium">{errorHint || error.message}</div>
              {errorHint && errorHint !== error.message && (
                <div className="text-red-300/70 mt-0.5">{error.message}</div>
              )}
            </div>
          </div>
        )}

        <button
          onClick={handleSubmit}
          disabled={!codeOK || submitting}
          className="w-full rounded bg-emerald-600 hover:bg-emerald-500 disabled:opacity-40 text-white text-sm py-2 inline-flex items-center justify-center gap-2"
        >
          {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
          {t('enduser.activate.submit', '激活')}
        </button>

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
