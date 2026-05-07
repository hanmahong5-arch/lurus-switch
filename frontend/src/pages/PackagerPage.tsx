import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  AlertTriangle, Box, Check, Copy, Image as ImageIcon, Loader2, Package,
  ShieldAlert, FolderOpen, Archive, History, X, RefreshCw,
} from 'lucide-react'
import {
  BuildWhiteLabelPackage, PreviewWhiteLabelLogo,
  WhiteLabelPreflight, OpenWhiteLabelOutputDir, ZipWhiteLabelOutputDir,
  ListWhiteLabelBuilds,
} from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'
import { useDirtyGuard } from '../hooks/useDirtyGuard'
import { useToastStore } from '../stores/toastStore'

// PackagerPage — Reseller-only. Collects branding inputs and runs the
// white-label packager, showing the resulting binary path + sha256 for
// the operator to verify before distributing.
//
// Out of scope for now: build history (the spec calls for last-10
// records in SQLite — defer until we have a packager_profiles table
// and can show it on this same page without scope creep).
export function PackagerPage() {
  const { t, i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh') ?? true
  const toast = useToastStore((s) => s.addToast)
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

  // Preflight + post-build extras (audit polish).
  const [preflight, setPreflight] = useState<main.PreflightReport | null>(null)
  const [preflighting, setPreflighting] = useState(false)
  const [zipping, setZipping] = useState(false)
  const [history, setHistory] = useState<main.BuildHistoryEntry[]>([])
  const refreshHistory = async () => {
    try {
      const rows = await ListWhiteLabelBuilds(10)
      setHistory(rows ?? [])
    } catch { /* non-fatal */ }
  }
  useEffect(() => { refreshHistory() }, [])

  // Dirty when the operator has filled anything but hasn't yet seen a
  // successful build — leaving early would discard the inputs.
  const hasInput =
    brandName.trim() !== '' ||
    hubURL.trim() !== '' ||
    tenantSlug.trim() !== '' ||
    supportContact.trim() !== '' ||
    logoBase64 !== ''
  useDirtyGuard('packager-page', hasInput && !result)

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
      refreshHistory()
    } catch (e) {
      setError(String(e))
    } finally {
      setBuilding(false)
    }
  }

  const handlePreflight = async () => {
    if (!hubURL.trim()) {
      toast('error', isZh ? '先填 Hub URL' : 'Fill Hub URL first')
      return
    }
    setPreflighting(true)
    try {
      const r = await WhiteLabelPreflight(hubURL.trim(), tenantSlug.trim())
      setPreflight(r)
    } catch (e) {
      toast('error', String(e))
    } finally {
      setPreflighting(false)
    }
  }

  const handleOpenOutputDir = async () => {
    if (!result?.outputDir) return
    try {
      await OpenWhiteLabelOutputDir(result.outputDir)
    } catch (e) {
      toast('error', String(e))
    }
  }

  const handleZip = async () => {
    if (!result?.outputDir) return
    setZipping(true)
    try {
      const zipPath = await ZipWhiteLabelOutputDir(result.outputDir)
      toast('success', isZh ? `ZIP 已生成：${zipPath}` : `ZIP ready: ${zipPath}`)
    } catch (e) {
      toast('error', String(e))
    } finally {
      setZipping(false)
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

        <div className="flex items-center gap-2">
          <button
            onClick={handlePreflight}
            disabled={!hubURL.trim() || preflighting}
            className="px-3 py-1.5 rounded border border-border hover:bg-muted text-sm disabled:opacity-40 inline-flex items-center gap-1.5"
            title={isZh ? '在打包前先 ping 一下 Hub 的 redeem / heartbeat 端点' : 'Ping the Hub redeem/heartbeat endpoints before building'}
          >
            {preflighting ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <ShieldAlert className="h-3.5 w-3.5" />}
            {isZh ? 'Hub 预检' : 'Hub preflight'}
          </button>
          <button
            onClick={handleBuild}
            disabled={!canBuild || building}
            className="px-5 py-2 rounded bg-purple-600 hover:bg-purple-500 text-white text-sm
                       disabled:opacity-40 inline-flex items-center gap-2"
          >
            {building ? <Loader2 className="h-4 w-4 animate-spin" /> : <Package className="h-4 w-4" />}
            {t('packager.build', '生成白标包')}
          </button>
        </div>

        {preflight && (
          <section className={`rounded-lg border p-3 space-y-1.5 text-xs ${preflight.ok ? 'border-emerald-500/30 bg-emerald-500/10' : 'border-amber-500/30 bg-amber-500/10'}`}>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-1.5 font-semibold">
                {preflight.ok
                  ? <><Check className="h-4 w-4 text-emerald-400" /> {isZh ? 'Hub 全部检查通过' : 'All Hub checks passed'}</>
                  : <><AlertTriangle className="h-4 w-4 text-amber-400" /> {isZh ? '部分检查未通过' : 'Some checks failed'}</>}
              </div>
              <button onClick={() => setPreflight(null)} className="text-muted-foreground hover:text-foreground">
                <X className="h-3.5 w-3.5" />
              </button>
            </div>
            {preflight.checks?.map((c) => (
              <div key={c.id} className="flex items-start gap-2">
                <span className={`mt-0.5 h-1.5 w-1.5 rounded-full ${c.pass ? 'bg-emerald-400' : 'bg-red-400'}`} />
                <div className="flex-1 min-w-0">
                  <div>{isZh ? c.titleZh : c.titleEn}</div>
                  {(c.detailZh || c.detailEn) && (
                    <div className="text-[10px] text-muted-foreground/80 font-mono break-all">
                      {isZh ? (c.detailZh || c.detailEn) : (c.detailEn || c.detailZh)}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </section>
        )}

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
            <div className="flex items-center gap-2 pt-1">
              <button
                onClick={handleOpenOutputDir}
                className="px-3 py-1.5 rounded border border-border hover:bg-muted text-xs inline-flex items-center gap-1.5"
              >
                <FolderOpen className="h-3.5 w-3.5" />
                {isZh ? '在资源管理器中打开' : 'Open in Explorer'}
              </button>
              <button
                onClick={handleZip}
                disabled={zipping}
                className="px-3 py-1.5 rounded border border-border hover:bg-muted text-xs inline-flex items-center gap-1.5 disabled:opacity-50"
              >
                {zipping ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Archive className="h-3.5 w-3.5" />}
                {isZh ? '打 ZIP（一文件分发）' : 'Bundle as ZIP'}
              </button>
            </div>
          </section>
        )}

        {/* Build history */}
        {history.length > 0 && (
          <section className="rounded-lg border border-border bg-card/40 p-4 space-y-2">
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-2">
                <History className="h-3.5 w-3.5" />
                {isZh ? '最近构建' : 'Recent builds'}
                <span className="text-[10px] text-muted-foreground/60 font-normal">{history.length}</span>
              </h3>
              <button onClick={refreshHistory} className="text-muted-foreground hover:text-foreground" title={isZh ? '刷新' : 'Refresh'}>
                <RefreshCw className="h-3 w-3" />
              </button>
            </div>
            <div className="space-y-1">
              {history.map((h, i) => (
                <div key={i} className="rounded border border-border/40 px-2.5 py-1.5 text-[11px] flex items-center justify-between gap-2">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-medium truncate">{h.brandName}</span>
                      <span className="text-[10px] text-muted-foreground/70 tabular-nums">
                        {h.builtAt ? new Date(h.builtAt).toLocaleString() : '—'}
                      </span>
                    </div>
                    <div className="text-[10px] text-muted-foreground font-mono break-all truncate">{h.binaryPath}</div>
                  </div>
                  <button
                    onClick={async () => {
                      try {
                        const dir = h.binaryPath.replace(/[\\/][^\\/]+$/, '')
                        await OpenWhiteLabelOutputDir(dir)
                      } catch (e) { toast('error', String(e)) }
                    }}
                    className="shrink-0 px-2 py-1 rounded border border-border hover:bg-muted text-[10px] inline-flex items-center gap-1"
                  >
                    <FolderOpen className="h-3 w-3" />
                    {isZh ? '打开' : 'Open'}
                  </button>
                </div>
              ))}
            </div>
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
