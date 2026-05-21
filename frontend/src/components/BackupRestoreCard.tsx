import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Download, Upload, Loader2, AlertTriangle, CheckCircle2, X,
} from 'lucide-react'
import { cn } from '../lib/utils'
import {
  PickExportBundlePath, PickImportBundlePath,
  ExportConfigBundle, PreviewImportBundle, ApplyImportBundle,
} from '../../wailsjs/go/main/App'

interface ComponentPreview {
  key: string
  inBundle: boolean
  action: string // overwrite | create | skip
  detail?: string
}

interface BundlePreview {
  manifest: { schemaVersion: number; exportedAt: string; appVersion: string; includesKeys: boolean; components: string[] }
  components: ComponentPreview[]
}

const COMPONENT_LABEL: Record<string, { zh: string; en: string }> = {
  'app-settings': { zh: '应用设置', en: 'App settings' },
  'custom-providers': { zh: '自定义供应商', en: 'Custom providers' },
  'tool-configs': { zh: 'CLI 工具配置', en: 'Tool configs' },
  'mcp-presets': { zh: 'MCP 预设', en: 'MCP presets' },
  'prompts': { zh: '提示词库', en: 'Prompts' },
  'snapshots': { zh: '配置快照', en: 'Snapshots' },
}

export function BackupRestoreCard() {
  const { t, i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh') ?? true
  const [includeKeys, setIncludeKeys] = useState(false)
  const [busy, setBusy] = useState(false)
  const [msg, setMsg] = useState<{ kind: 'ok' | 'err'; text: string } | null>(null)

  // Import flow state.
  const [importPath, setImportPath] = useState('')
  const [preview, setPreview] = useState<BundlePreview | null>(null)
  const [accepted, setAccepted] = useState<Record<string, boolean>>({})

  const label = (key: string) => {
    const l = COMPONENT_LABEL[key]
    return l ? (isZh ? l.zh : l.en) : key
  }

  const handleExport = async () => {
    setMsg(null)
    const path = await PickExportBundlePath()
    if (!path) return
    setBusy(true)
    try {
      const mf = await ExportConfigBundle(path, includeKeys)
      setMsg({
        kind: 'ok',
        text: t('backup.exportOk', '已导出 {{count}} 个组件到 {{path}}', {
          count: (mf as any)?.components?.length ?? 0,
          path,
        }),
      })
    } catch (e: any) {
      setMsg({ kind: 'err', text: e?.message ?? String(e) })
    } finally {
      setBusy(false)
    }
  }

  const handlePickImport = async () => {
    setMsg(null)
    setPreview(null)
    const path = await PickImportBundlePath()
    if (!path) return
    setImportPath(path)
    setBusy(true)
    try {
      const pv = (await PreviewImportBundle(path)) as BundlePreview
      setPreview(pv)
      // Default-accept every component that's actually in the bundle.
      const acc: Record<string, boolean> = {}
      for (const c of pv.components) acc[c.key] = c.inBundle
      setAccepted(acc)
    } catch (e: any) {
      setMsg({ kind: 'err', text: e?.message ?? String(e) })
    } finally {
      setBusy(false)
    }
  }

  const handleApply = async () => {
    setBusy(true)
    setMsg(null)
    try {
      const written = (await ApplyImportBundle(importPath, accepted)) as string[]
      setMsg({
        kind: 'ok',
        text: t('backup.importOk', '已恢复 {{count}} 个组件，原文件已备份。重启后生效。', {
          count: written.length,
        }),
      })
      setPreview(null)
      setImportPath('')
    } catch (e: any) {
      setMsg({ kind: 'err', text: e?.message ?? String(e) })
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Export */}
      <div className="p-4 border border-border rounded-md space-y-3">
        <div>
          <h3 className="text-sm font-medium">{t('backup.exportTitle', '导出配置')}</h3>
          <p className="text-xs text-muted-foreground">
            {t('backup.exportDesc', '把所有本地配置打包成一个 zip，便于换机迁移。')}
          </p>
        </div>
        <label className="flex items-start gap-2 text-xs">
          <input
            type="checkbox"
            checked={includeKeys}
            onChange={(e) => setIncludeKeys(e.target.checked)}
            className="mt-0.5 accent-red-500"
          />
          <span>
            <span className="text-red-500 font-medium">{t('backup.includeKeys', '包含 API Key')}</span>
            <span className="text-muted-foreground ml-1">
              {t('backup.includeKeysWarn', '（导出的文件将含明文密钥，请妥善保管）')}
            </span>
          </span>
        </label>
        <button
          onClick={handleExport}
          disabled={busy}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-md bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Download className="h-3.5 w-3.5" />}
          {t('backup.export', '导出')}
        </button>
      </div>

      {/* Import */}
      <div className="p-4 border border-border rounded-md space-y-3">
        <div>
          <h3 className="text-sm font-medium">{t('backup.importTitle', '导入配置')}</h3>
          <p className="text-xs text-muted-foreground">
            {t('backup.importDesc', '从一个导出的 zip 恢复配置。覆盖前会自动备份现有文件。')}
          </p>
        </div>

        {!preview ? (
          <button
            onClick={handlePickImport}
            disabled={busy}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-md border border-border hover:bg-muted disabled:opacity-50"
          >
            {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Upload className="h-3.5 w-3.5" />}
            {t('backup.chooseFile', '选择文件…')}
          </button>
        ) : (
          <div className="space-y-2">
            <div className="text-[11px] text-muted-foreground">
              {t('backup.bundleInfo', '来自 v{{ver}} · {{keys}}', {
                ver: preview.manifest.appVersion || '?',
                keys: preview.manifest.includesKeys
                  ? t('backup.withKeys', '含密钥')
                  : t('backup.noKeys', '不含密钥'),
              })}
            </div>
            <div className="space-y-1">
              {preview.components.filter((c) => c.inBundle).map((c) => (
                <label key={c.key} className="flex items-center gap-2 text-xs">
                  <input
                    type="checkbox"
                    checked={accepted[c.key] ?? false}
                    onChange={(e) => setAccepted((a) => ({ ...a, [c.key]: e.target.checked }))}
                    className="accent-primary"
                  />
                  <span className="flex-1">{label(c.key)}</span>
                  <span
                    className={cn(
                      'text-[10px] px-1.5 py-0.5 rounded',
                      c.action === 'overwrite'
                        ? 'text-amber-500 bg-amber-500/10'
                        : 'text-emerald-500 bg-emerald-500/10',
                    )}
                  >
                    {c.action === 'overwrite'
                      ? t('backup.willOverwrite', '覆盖')
                      : t('backup.willCreate', '新建')}
                  </span>
                </label>
              ))}
            </div>
            <div className="flex items-center gap-2 pt-1">
              <button
                onClick={() => { setPreview(null); setImportPath('') }}
                className="flex items-center gap-1 px-3 py-1.5 text-xs rounded border border-border hover:bg-muted"
              >
                <X className="h-3.5 w-3.5" />
                {t('common.cancel', '取消')}
              </button>
              <button
                onClick={handleApply}
                disabled={busy || !Object.values(accepted).some(Boolean)}
                className="flex items-center gap-1 px-3 py-1.5 text-xs rounded bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 ml-auto"
              >
                {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Upload className="h-3.5 w-3.5" />}
                {t('backup.applyImport', '确认导入')}
              </button>
            </div>
          </div>
        )}
      </div>

      {msg && (
        <div
          className={cn(
            'text-xs rounded px-3 py-2 border flex items-start gap-2',
            msg.kind === 'ok'
              ? 'text-emerald-600 bg-emerald-500/10 border-emerald-500/20'
              : 'text-red-500 bg-red-500/10 border-red-500/20',
          )}
        >
          {msg.kind === 'ok' ? <CheckCircle2 className="h-4 w-4 shrink-0" /> : <AlertTriangle className="h-4 w-4 shrink-0" />}
          <span className="break-all">{msg.text}</span>
        </div>
      )}
    </div>
  )
}
