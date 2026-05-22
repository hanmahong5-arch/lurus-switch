import { useEffect, useState, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Button, Card } from '../components/ui'
import {
  Package, Save, RotateCcw, RefreshCw, Loader2, AlertCircle, CheckCircle2,
  Plus, Trash2, Pin, PinOff, ChevronDown, ChevronRight,
} from 'lucide-react'
import { cn } from '../lib/utils'
import { useToastStore } from '../stores/toastStore'
import {
  GetToolManifestAdminView, SaveToolManifestEntry,
  DeleteToolManifestEntry, ResetToolManifestOverrides,
} from '../../wailsjs/go/main/App'
import type { main, toolmanifest } from '../../wailsjs/go/models'

type StatusOpt = 'stable' | 'coming-soon' | 'beta' | ''

const STATUS_OPTIONS: { value: StatusOpt; label: string; tone: string }[] = [
  { value: 'stable',       label: '稳定 (Stable)',      tone: 'text-[var(--lt-ok)]'  },
  { value: 'beta',         label: 'Beta',                tone: 'text-[var(--lt-warn)]' },
  { value: 'coming-soon',  label: '敬请期待',            tone: 'text-muted-foreground' },
  { value: '',             label: '默认 (随上游)',       tone: 'text-muted-foreground/60' },
]

const PLATFORM_OPTIONS = [
  'windows/amd64',
  'darwin/amd64',
  'darwin/arm64',
  'linux/amd64',
  'linux/arm64',
]

interface Draft {
  type: string
  npmPackage: string
  latestVersion: string
  status: StatusOpt
  platforms: Record<string, toolmanifest.PlatformAsset>
}

function rowToDraft(r: main.ToolManifestRow): Draft {
  return {
    type: r.type || '',
    npmPackage: r.npmPackage || '',
    latestVersion: r.latestVersion || '',
    status: (r.status as StatusOpt) || '',
    platforms: { ...(r.platforms || {}) },
  }
}

function draftsEqual(a: Draft, b: Draft): boolean {
  if (a.type !== b.type) return false
  if (a.npmPackage !== b.npmPackage) return false
  if (a.latestVersion !== b.latestVersion) return false
  if (a.status !== b.status) return false
  const ak = Object.keys(a.platforms).sort()
  const bk = Object.keys(b.platforms).sort()
  if (ak.length !== bk.length) return false
  for (let i = 0; i < ak.length; i++) {
    if (ak[i] !== bk[i]) return false
    const av = a.platforms[ak[i]]
    const bv = b.platforms[bk[i]]
    if (av.url !== bv.url || (av.sha256 || '') !== (bv.sha256 || '')) return false
  }
  return true
}

export function ToolReleasePage() {
  const { t } = useTranslation()
  const toast = useToastStore((s) => s.addToast)

  const [view, setView] = useState<main.ToolManifestAdminView | null>(null)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState<string | null>(null)
  const [drafts, setDrafts] = useState<Record<string, Draft>>({})
  const [expanded, setExpanded] = useState<Record<string, boolean>>({})

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const v = await GetToolManifestAdminView()
      setView(v)
      const next: Record<string, Draft> = {}
      for (const r of (v.rows || [])) next[r.name] = rowToDraft(r)
      setDrafts(next)
    } catch (e: any) {
      toast('error', e?.message || String(e))
    } finally {
      setLoading(false)
    }
  }, [toast])

  useEffect(() => { void refresh() }, [refresh])

  const handleSave = async (name: string) => {
    if (saving) return
    const draft = drafts[name]
    if (!draft) return
    setSaving(name)
    try {
      await SaveToolManifestEntry(name, {
        type: draft.type,
        npm_package: draft.npmPackage,
        latest_version: draft.latestVersion,
        status: draft.status,
        platforms: draft.platforms,
      } as any)
      toast('success', t('toolRelease.saved', { name, defaultValue: '已保存 {{name}}（已下发到本地 manifest）' }))
      await refresh()
    } catch (e: any) {
      toast('error', e?.message || String(e))
    } finally {
      setSaving(null)
    }
  }

  const handleRevert = async (name: string) => {
    if (saving) return
    setSaving(name)
    try {
      await DeleteToolManifestEntry(name)
      toast('success', t('toolRelease.reverted', { name, defaultValue: '已恢复 {{name}} 为上游默认' }))
      await refresh()
    } catch (e: any) {
      toast('error', e?.message || String(e))
    } finally {
      setSaving(null)
    }
  }

  const handleResetAll = async () => {
    if (!confirm(t('toolRelease.confirmReset', '清空所有本地覆盖并恢复为上游默认？'))) return
    setLoading(true)
    try {
      await ResetToolManifestOverrides()
      toast('success', t('toolRelease.resetDone', '所有覆盖已清空'))
      await refresh()
    } catch (e: any) {
      toast('error', e?.message || String(e))
    } finally {
      setLoading(false)
    }
  }

  const rows = view?.rows || []

  if (loading && !view) {
    return (
      <div className="h-full flex items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-[1200px] mx-auto p-6 space-y-4">
        <header className="flex items-start justify-between">
          <div>
            <h2 className="text-lg font-semibold flex items-center gap-2">
              <Package className="h-5 w-5 text-primary" />
              {t('toolRelease.title', '工具上架管理')}
            </h2>
            <p className="text-sm text-muted-foreground mt-1">
              {t('toolRelease.subtitle', '编辑 manifest：把工具二进制 URL 指向 Lurus / 你的 CDN，状态切到 stable，前端立即看到一键安装。')}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="secondary"
              size="sm"
              onClick={refresh}
              disabled={loading}
              loading={loading}
              icon={!loading ? <RefreshCw className="h-3.5 w-3.5" /> : undefined}
            >
              {t('toolRelease.refresh', '刷新')}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleResetAll}
              disabled={loading}
              icon={<RotateCcw className="h-3.5 w-3.5" />}
              className="border border-destructive/30 text-destructive hover:bg-destructive/10"
            >
              {t('toolRelease.resetAll', '清空所有覆盖')}
            </Button>
          </div>
        </header>

        <Card variant="default" className="border-primary/30 bg-primary/5 p-3 text-xs text-foreground/80">
          <p className="font-mono text-[10px] uppercase tracking-[0.18em] text-primary mb-1">[ {t('toolRelease.howTo', '上架流程').toUpperCase()} ]</p>
          <ol className="list-decimal list-inside space-y-0.5 text-muted-foreground">
            <li>{t('toolRelease.step1', '把二进制传到稳定的 CDN（如 releases.lurus.cn）')}</li>
            <li>{t('toolRelease.step2', '在下面对应工具的「平台资产」加 URL 和 SHA256')}</li>
            <li>{t('toolRelease.step3', '状态切到「稳定」并保存 — 当前 Switch 立即可用')}</li>
            <li>{t('toolRelease.step4', '（可选）把这份 JSON 推到 hub 让所有 EndUser 客户端同步')}</li>
          </ol>
        </Card>

        {rows.length === 0 && !loading && (
          <div className="text-center text-sm text-muted-foreground py-12">
            {t('toolRelease.empty', '当前 manifest 为空。')}
          </div>
        )}

        <div className="space-y-3">
          {rows.map((row) => (
            <ToolCard
              key={row.name}
              row={row}
              draft={drafts[row.name]}
              setDraft={(d) => setDrafts((prev) => ({ ...prev, [row.name]: d }))}
              expanded={!!expanded[row.name]}
              toggleExpand={() => setExpanded((p) => ({ ...p, [row.name]: !p[row.name] }))}
              saving={saving === row.name}
              onSave={() => handleSave(row.name)}
              onRevert={() => handleRevert(row.name)}
            />
          ))}
        </div>

        {view?.updatedAt && (
          <p className="text-[11px] text-muted-foreground mt-4 text-center">
            {t('toolRelease.lastUpdated', '本地覆盖文件上次更新：{{at}}', { at: new Date(view.updatedAt).toLocaleString() })}
          </p>
        )}
      </div>
    </div>
  )
}

function ToolCard({
  row, draft, setDraft, expanded, toggleExpand, saving, onSave, onRevert,
}: {
  row: main.ToolManifestRow
  draft: Draft
  setDraft: (d: Draft) => void
  expanded: boolean
  toggleExpand: () => void
  saving: boolean
  onSave: () => void
  onRevert: () => void
}) {
  const { t } = useTranslation()
  const original = useMemo(() => rowToDraft(row), [row])
  const dirty = useMemo(() => !draftsEqual(draft, original), [draft, original])

  const statusBadge = useMemo(() => {
    const opt = STATUS_OPTIONS.find((o) => o.value === draft.status) || STATUS_OPTIONS[3]
    return opt
  }, [draft.status])

  const addPlatform = (key: string) => {
    if (draft.platforms[key]) return
    setDraft({ ...draft, platforms: { ...draft.platforms, [key]: { url: '', sha256: '' } as any } })
  }
  const removePlatform = (key: string) => {
    const next = { ...draft.platforms }
    delete next[key]
    setDraft({ ...draft, platforms: next })
  }
  const updatePlatform = (key: string, field: 'url' | 'sha256', value: string) => {
    setDraft({ ...draft, platforms: { ...draft.platforms, [key]: { ...draft.platforms[key], [field]: value } as any } })
  }

  return (
    <div className={cn(
      'rounded-lg border bg-card',
      row.overridden ? 'border-[var(--lt-accent)]/40' : 'border-border',
    )}>
      {/* Header row */}
      <div className="flex items-center gap-3 px-4 py-3">
        <button
          onClick={toggleExpand}
          className="p-1 rounded hover:bg-muted text-muted-foreground shrink-0"
          aria-label="Toggle"
        >
          {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
        </button>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-sm">{row.name}</span>
            <span className={cn('text-[11px]', statusBadge.tone)}>{statusBadge.label}</span>
            {row.overridden && (
              <span className="flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] bg-[var(--lt-accent)]/15 text-[var(--lt-accent)]">
                <Pin className="h-3 w-3" />
                {t('toolRelease.overridden', '已覆盖')}
              </span>
            )}
          </div>
          <p className="text-[11px] text-muted-foreground font-mono mt-0.5">
            type: {draft.type || '—'} · v{draft.latestVersion || '—'}
            {draft.npmPackage && <> · npm: {draft.npmPackage}</>}
            {' · '}{Object.keys(draft.platforms).length} platforms
          </p>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {dirty && (
            <span className="flex items-center gap-1 text-[11px] text-[var(--lt-warn)]">
              <AlertCircle className="h-3 w-3" />
              {t('toolRelease.unsaved', '未保存')}
            </span>
          )}
          {row.overridden && (
            <button
              onClick={onRevert}
              disabled={saving}
              title={t('toolRelease.revertHint', '清除本地覆盖，恢复为上游默认')}
              className="flex items-center gap-1 px-2 py-1 rounded text-xs border border-border hover:bg-muted disabled:opacity-50"
            >
              <PinOff className="h-3 w-3" />
              {t('toolRelease.revert', '恢复默认')}
            </button>
          )}
          <button
            onClick={onSave}
            disabled={saving || !dirty}
            className={cn(
              'flex items-center gap-1 px-3 py-1 rounded text-xs font-medium',
              dirty
                ? 'bg-[var(--lt-accent)] text-white hover:opacity-90'
                : 'bg-muted text-muted-foreground cursor-not-allowed',
            )}
          >
            {saving ? <Loader2 className="h-3 w-3 animate-spin" /> : <Save className="h-3 w-3" />}
            {t('toolRelease.save', '保存')}
          </button>
        </div>
      </div>

      {/* Edit body */}
      {expanded && (
        <div className="border-t border-border px-4 py-4 space-y-4 bg-background/40">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <Field label={t('toolRelease.field.status', '状态')}>
              <select
                value={draft.status}
                onChange={(e) => setDraft({ ...draft, status: e.target.value as StatusOpt })}
                className="w-full px-2 py-1.5 rounded border border-border bg-background text-xs"
              >
                {STATUS_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>{o.label}</option>
                ))}
              </select>
            </Field>
            <Field label={t('toolRelease.field.type', '类型')}>
              <select
                value={draft.type}
                onChange={(e) => setDraft({ ...draft, type: e.target.value })}
                className="w-full px-2 py-1.5 rounded border border-border bg-background text-xs"
              >
                <option value="">—</option>
                <option value="npm">npm</option>
                <option value="binary">binary</option>
                <option value="desktop">desktop</option>
              </select>
            </Field>
            <Field label={t('toolRelease.field.version', '版本号 (latest_version)')}>
              <input
                value={draft.latestVersion}
                onChange={(e) => setDraft({ ...draft, latestVersion: e.target.value })}
                placeholder="1.0.0"
                className="w-full px-2 py-1.5 rounded border border-border bg-background text-xs font-mono"
              />
            </Field>
          </div>

          {draft.type === 'npm' && (
            <Field label={t('toolRelease.field.npm', 'npm 包名')}>
              <input
                value={draft.npmPackage}
                onChange={(e) => setDraft({ ...draft, npmPackage: e.target.value })}
                placeholder="@scope/name"
                className="w-full px-2 py-1.5 rounded border border-border bg-background text-xs font-mono"
              />
            </Field>
          )}

          {/* Platform assets table */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium">{t('toolRelease.field.platforms', '平台资产 (binary 类型)')}</span>
              <PlatformAdder
                existing={Object.keys(draft.platforms)}
                onAdd={addPlatform}
              />
            </div>
            <div className="space-y-1.5">
              {Object.keys(draft.platforms).length === 0 && (
                <p className="text-[11px] text-muted-foreground italic">
                  {t('toolRelease.field.platformsEmpty', '没有平台资产 — 若状态非 npm 类，install 会回退到 GitHub。')}
                </p>
              )}
              {Object.entries(draft.platforms).map(([key, asset]) => (
                <div key={key} className="grid grid-cols-12 gap-2 items-center">
                  <span className="col-span-2 text-[11px] font-mono text-muted-foreground">{key}</span>
                  <input
                    value={asset.url || ''}
                    onChange={(e) => updatePlatform(key, 'url', e.target.value)}
                    placeholder="https://releases.lurus.cn/..."
                    className="col-span-6 px-2 py-1 rounded border border-border bg-background text-[11px] font-mono"
                  />
                  <input
                    value={asset.sha256 || ''}
                    onChange={(e) => updatePlatform(key, 'sha256', e.target.value)}
                    placeholder="sha256 (可选)"
                    className="col-span-3 px-2 py-1 rounded border border-border bg-background text-[11px] font-mono"
                  />
                  <button
                    onClick={() => removePlatform(key)}
                    className="col-span-1 p-1 rounded hover:bg-destructive/10 text-destructive justify-self-end"
                    aria-label="Remove"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
            </div>
          </div>

          {/* Hint footer */}
          <div className="flex items-start gap-2 text-[11px] text-muted-foreground border-t border-border pt-3">
            <CheckCircle2 className="h-3.5 w-3.5 shrink-0 mt-0.5" />
            <span>
              {t('toolRelease.footerHint', '保存后立即影响本机的 install / topology — 不需要重启。要让其他 EndUser 客户端同步，需要把同样的条目推到 hub 的 download-manifest 端点。')}
            </span>
          </div>
        </div>
      )}
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className="text-[11px] uppercase tracking-wider text-muted-foreground">{label}</span>
      <div className="mt-1">{children}</div>
    </label>
  )
}

function PlatformAdder({ existing, onAdd }: { existing: string[]; onAdd: (k: string) => void }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const available = PLATFORM_OPTIONS.filter((p) => !existing.includes(p))
  if (available.length === 0) return null
  return (
    <div className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-1 px-2 py-1 rounded text-[11px] border border-border hover:bg-muted"
      >
        <Plus className="h-3 w-3" />
        {t('toolRelease.addPlatform', '添加平台')}
      </button>
      {open && (
        <div className="absolute right-0 top-7 z-10 min-w-[140px] rounded-md border border-border bg-card shadow-lg">
          {available.map((p) => (
            <button
              key={p}
              onClick={() => { onAdd(p); setOpen(false) }}
              className="block w-full text-left px-3 py-1.5 text-[11px] font-mono hover:bg-muted"
            >
              {p}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
