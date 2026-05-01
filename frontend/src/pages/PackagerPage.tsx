import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  AlertTriangle,
  Box,
  Check,
  Copy,
  Image as ImageIcon,
  Loader2,
  Package,
} from 'lucide-react'
import {
  BuildWhiteLabelPackage,
  PreviewWhiteLabelLogo,
} from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'

// PackagerPage — Reseller-only. Collects branding inputs and runs the
// white-label packager, showing the resulting binary path + sha256 for
// the operator to verify before distributing.
//
// Out of scope for now: build history (the spec calls for last-10
// records in SQLite — defer until we have a packager_profiles table
// and can show it on this same page without scope creep).
export function PackagerPage() {
  const { t } = useTranslation()
  const [brandName, setBrandName] = useState('')
  const [hubURL, setHubURL] = useState('')
  const [tenantSlug, setTenantSlug] = useState('')
  const [primaryColor, setPrimaryColor] = useState('#9333ea')
  const [supportContact, setSupportContact] = useState('')
  const [logoBase64, setLogoBase64] = useState('')
  const [logoMeta, setLogoMeta] = useState<{ size: number; mime: string; oversized: boolean } | null>(null)
  const [building, setBuilding] = useState(false)
  const [result, setResult] = useState<main.WhiteLabelOutput | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState<'binary' | 'sidecar' | null>(null)

  const onLogoFile = async (file: File | null) => {
    if (!file) {
      setLogoBase64('')
      setLogoMeta(null)
      return
    }
    const buf = await file.arrayBuffer()
    const bytes = new Uint8Array(buf)
    let bin = ''
    for (let i = 0; i < bytes.byteLength; i++) bin += String.fromCharCode(bytes[i])
    const b64 = btoa(bin)
    setLogoBase64(b64)
    try {
      const meta = await PreviewWhiteLabelLogo(b64)
      setLogoMeta({
        size: Number((meta as any).size ?? 0),
        mime: String((meta as any).mime ?? 'unknown'),
        oversized: Boolean((meta as any).oversized),
      })
    } catch (e) {
      setError(String(e))
      setLogoMeta(null)
    }
  }

  const canBuild =
    brandName.trim() !== '' &&
    /^https?:\/\//.test(hubURL.trim()) &&
    !logoMeta?.oversized

  const handleBuild = async () => {
    setBuilding(true)
    setError(null)
    setResult(null)
    try {
      const res = await BuildWhiteLabelPackage({
        brandName: brandName.trim(),
        hubUrl: hubURL.trim(),
        tenantSlug: tenantSlug.trim(),
        primaryColor: primaryColor.trim(),
        supportContact: supportContact.trim(),
        logoBase64,
      })
      setResult(res)
    } catch (e) {
      setError(String(e))
    } finally {
      setBuilding(false)
    }
  }

  const copyToClipboard = async (text: string, label: 'binary' | 'sidecar') => {
    try {
      await navigator.clipboard.writeText(text)
      setCopied(label)
      setTimeout(() => setCopied(null), 1500)
    } catch {
      /* clipboard blocked — surface nothing, user can select manually */
    }
  }

  const formatBytes = (n: number) => {
    if (n < 1024) return `${n} B`
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
    return `${(n / 1024 / 1024).toFixed(2)} MB`
  }

  return (
    <div className="h-full overflow-auto bg-background text-foreground">
      <div className="max-w-3xl mx-auto p-6 space-y-6">
        <header className="flex items-center gap-3">
          <Package className="h-6 w-6 text-purple-400" />
          <div>
            <h1 className="text-xl font-semibold">
              {t('packager.title', '白标打包')}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t('packager.subtitle', '把你的 Hub 接入参数 + 品牌打成一个 EndUser 安装包。')}
            </p>
          </div>
        </header>

        {/* Form */}
        <div className="rounded-lg border border-border bg-card p-5 space-y-4">
          <h2 className="font-medium flex items-center gap-2">
            <Box className="h-4 w-4" />
            {t('packager.section.branding', '品牌信息')}
          </h2>

          <Field label={t('packager.brandName', '品牌名 *')}>
            <input
              className="w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
              value={brandName}
              onChange={(e) => setBrandName(e.target.value)}
              placeholder="Acme Corp"
            />
          </Field>

          <Field label="Hub URL *">
            <input
              className="w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono"
              value={hubURL}
              onChange={(e) => setHubURL(e.target.value)}
              placeholder="https://hub.acme.example"
            />
          </Field>

          <Field label={t('packager.tenantSlug', 'Tenant Slug（V2 多租户可选）')}>
            <input
              className="w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono"
              value={tenantSlug}
              onChange={(e) => setTenantSlug(e.target.value)}
              placeholder="acme"
            />
          </Field>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Field label={t('packager.primaryColor', '主题色')}>
              <div className="flex items-center gap-2">
                <input
                  type="color"
                  className="w-10 h-9 rounded border border-border bg-background cursor-pointer"
                  value={primaryColor}
                  onChange={(e) => setPrimaryColor(e.target.value)}
                />
                <input
                  className="flex-1 rounded border border-border bg-background px-3 py-1.5 text-sm font-mono"
                  value={primaryColor}
                  onChange={(e) => setPrimaryColor(e.target.value)}
                />
              </div>
            </Field>

            <Field label={t('packager.supportContact', '客服联系方式')}>
              <input
                className="w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                value={supportContact}
                onChange={(e) => setSupportContact(e.target.value)}
                placeholder="mailto:support@acme.example"
              />
            </Field>
          </div>

          <Field label={t('packager.logo', '品牌 Logo（PNG/JPG/SVG，≤256KB）')}>
            <label className="block">
              <input
                type="file"
                accept="image/png,image/jpeg,image/svg+xml"
                className="block w-full text-xs text-muted-foreground
                           file:mr-3 file:py-1.5 file:px-3 file:rounded file:border-0
                           file:text-xs file:font-medium
                           file:bg-purple-600 file:text-white hover:file:bg-purple-500"
                onChange={(e) => onLogoFile(e.target.files?.[0] ?? null)}
              />
            </label>
            {logoMeta && (
              <div className="mt-2 text-xs flex items-center gap-2">
                <ImageIcon className="h-3.5 w-3.5 text-muted-foreground" />
                <span className="text-muted-foreground">
                  {logoMeta.mime} · {formatBytes(logoMeta.size)}
                </span>
                {logoMeta.oversized && (
                  <span className="text-red-400 inline-flex items-center gap-1">
                    <AlertTriangle className="h-3.5 w-3.5" />
                    {t('packager.logoOversized', '超过 256KB 限制')}
                  </span>
                )}
              </div>
            )}
          </Field>
        </div>

        {error && (
          <div className="rounded-md border border-red-500/30 bg-red-950/20 px-3 py-2 text-xs text-red-200 flex items-start gap-2">
            <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
            <div>{error}</div>
          </div>
        )}

        <button
          onClick={handleBuild}
          disabled={!canBuild || building}
          className="px-5 py-2 rounded bg-purple-600 hover:bg-purple-500 text-white text-sm
                     disabled:opacity-40 inline-flex items-center gap-2"
        >
          {building ? <Loader2 className="h-4 w-4 animate-spin" /> : <Package className="h-4 w-4" />}
          {t('packager.build', '生成白标包')}
        </button>

        {result && (
          <section className="rounded-lg border border-emerald-500/30 bg-emerald-950/10 p-5 space-y-3">
            <div className="flex items-center gap-2 text-emerald-300">
              <Check className="h-5 w-5" />
              <h3 className="font-medium">{t('packager.success', '打包完成')}</h3>
            </div>
            <ResultRow label="OutputDir" value={result.outputDir} />
            <ResultRow
              label="Binary"
              value={result.binaryPath}
              copyable
              copied={copied === 'binary'}
              onCopy={() => copyToClipboard(result.binaryPath, 'binary')}
            />
            <ResultRow
              label="Sidecar"
              value={result.sidecarPath}
              copyable
              copied={copied === 'sidecar'}
              onCopy={() => copyToClipboard(result.sidecarPath, 'sidecar')}
            />
            <ResultRow label="SHA256 (binary)" value={result.binarySha256} mono />
            <ResultRow label="SHA256 (sidecar)" value={result.sidecarSha256} mono />
            {result.notes && result.notes.length > 0 && (
              <div className="text-xs text-amber-300 space-y-0.5">
                {result.notes.map((n, i) => (
                  <div key={i}>· {n}</div>
                ))}
              </div>
            )}
          </section>
        )}
      </div>
    </div>
  )
}

function Field({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <label className="block">
      <span className="block text-xs text-muted-foreground mb-1">{label}</span>
      {children}
    </label>
  )
}

function ResultRow({
  label,
  value,
  mono,
  copyable,
  copied,
  onCopy,
}: {
  label: string
  value: string
  mono?: boolean
  copyable?: boolean
  copied?: boolean
  onCopy?: () => void
}) {
  return (
    <div className="grid grid-cols-[110px,1fr,auto] gap-2 items-start text-xs">
      <span className="text-muted-foreground pt-0.5">{label}</span>
      <span className={(mono ? 'font-mono ' : '') + 'break-all'}>{value}</span>
      {copyable && (
        <button
          onClick={onCopy}
          className="px-1.5 py-0.5 rounded border border-border hover:bg-muted text-[10px] inline-flex items-center gap-1"
        >
          {copied ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
          {copied ? '已复制' : '复制'}
        </button>
      )}
    </div>
  )
}
