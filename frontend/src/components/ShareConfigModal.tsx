import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Share2, X, Copy, Check, AlertTriangle, Loader2 } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import { GenerateImportLink } from '../../wailsjs/go/main/App'
import { useToastStore } from '../stores/toastStore'

export interface ShareConfigModalProps {
  /** Import type: "provider" | "mcp" | "prompt" | "skill" */
  type: string
  /** JSON-serialisable object that describes the config to share */
  data: Record<string, unknown>
  onClose: () => void
}

// ShareConfigModal generates a switch:// deep-link URL from the provided
// type + data, renders the URL, a copy-to-clipboard button, and a QR code.
// Visual placement / styling needs human review.
export function ShareConfigModal({ type, data, onClose }: ShareConfigModalProps) {
  const { t } = useTranslation()
  const addToast = useToastStore((s) => s.addToast)

  const [url, setUrl] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [copied, setCopied] = useState(false)

  // Serialize once per render; the string value is stable across re-renders with
  // the same content, so the effect below does not re-fire (and re-call the Go
  // binding) when the parent passes a fresh inline object literal each render.
  const dataJSON = JSON.stringify(data)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    setUrl(null)

    GenerateImportLink(type, dataJSON)
      .then((link) => {
        if (!cancelled) {
          setUrl(link)
          setLoading(false)
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          const msg = (err as Error)?.message ?? String(err)
          setError(msg)
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [type, dataJSON])

  const handleCopy = async () => {
    if (!url) return
    try {
      await navigator.clipboard.writeText(url)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
      addToast('success', t('share.copySuccess', '链接已复制'))
    } catch {
      addToast('error', t('share.copyFailed', '复制失败，请手动选择文本'))
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="w-[480px] max-w-[92vw] max-h-[80vh] bg-background border border-border rounded-lg shadow-2xl flex flex-col overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <div className="flex items-center gap-2">
            <Share2 className="h-4 w-4 text-primary" />
            <h2 className="text-sm font-semibold">
              {t('share.title', '分享配置')}
            </h2>
          </div>
          <button
            onClick={onClose}
            className="text-muted-foreground hover:text-foreground transition-colors"
            aria-label={t('common.close', '关闭')}
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Body */}
        <div className="px-4 py-4 overflow-y-auto flex-1 space-y-4">
          <div className="text-xs text-muted-foreground uppercase tracking-wider">
            {t('share.typeLabel', '类型')}: {type}
          </div>

          {loading && (
            <div className="flex items-center justify-center py-8 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin" />
            </div>
          )}

          {error && (
            <div className="flex items-start gap-2 text-xs text-red-500 bg-red-500/10 border border-red-500/20 rounded px-3 py-2.5">
              <AlertTriangle className="h-3.5 w-3.5 flex-shrink-0 mt-0.5" />
              <span>{error}</span>
            </div>
          )}

          {url && !loading && (
            <>
              {/* URL display + copy button */}
              <div className="space-y-1.5">
                <p className="text-xs font-medium text-muted-foreground">
                  {t('share.urlLabel', '分享链接')}
                </p>
                <div className="flex items-start gap-2">
                  <code
                    className="flex-1 text-xs font-mono bg-muted px-3 py-2 rounded-md break-all select-all border border-border leading-relaxed"
                    data-testid="share-url"
                  >
                    {url}
                  </code>
                  <button
                    onClick={handleCopy}
                    className="flex-shrink-0 p-2 rounded-md border border-border hover:bg-muted transition-colors"
                    aria-label={t('share.copy', '复制链接')}
                    title={t('share.copy', '复制链接')}
                    data-testid="copy-btn"
                  >
                    {copied
                      ? <Check className="h-3.5 w-3.5 text-green-500" />
                      : <Copy className="h-3.5 w-3.5 text-muted-foreground" />}
                  </button>
                </div>
              </div>

              {/* QR code */}
              <div className="space-y-1.5">
                <p className="text-xs font-medium text-muted-foreground">
                  {t('share.qrLabel', '扫码导入')}
                </p>
                <div className="flex justify-center p-4 bg-white rounded-md border border-border" data-testid="qr-container">
                  <QRCodeSVG value={url} size={160} />
                </div>
              </div>

              <p className="text-xs text-muted-foreground">
                {t(
                  'share.hint',
                  '对方在装有 Switch 的设备上点击链接，或扫描二维码即可导入此配置。',
                )}
              </p>
            </>
          )}
        </div>

        {/* Footer */}
        <div className="px-4 py-3 border-t border-border flex items-center justify-end">
          <button
            onClick={onClose}
            className="px-3 py-1.5 text-sm rounded-md border border-border hover:bg-muted transition-colors"
          >
            {t('common.close', '关闭')}
          </button>
        </div>
      </div>
    </div>
  )
}
