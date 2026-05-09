import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Save, RotateCcw, AlertCircle } from 'lucide-react'
import {
  type OptionMeta, type SettingsTab,
  metaForTab, SETTINGS_GROUPS,
} from '../../lib/gatewayOptionMeta'
import { PricingEditor } from './PricingEditor'

interface Props {
  tab: SettingsTab
  options: Record<string, string>
  onSave: (key: string, value: string) => Promise<void>
  /**
   * Render this when the tab has metadata but the corresponding key is
   * absent in the loaded options map. Default: skip the row.
   */
  showMissing?: boolean
}

/**
 * Renders all options that have metadata for the given tab as a labeled,
 * grouped form. Section-level "Save All" button bulk-PUTs only the rows
 * the user touched. Pricing keys delegate to PricingEditor; everything
 * else is a label/widget pair driven by metadata.
 *
 * Tabs without any metadata fall through to the parent's plain
 * OptionEditor — see GatewaySettingsPage.
 */
export function OptionsSectionForm({ tab, options, onSave, showMissing }: Props) {
  const { t } = useTranslation()
  const tabMeta = useMemo(() => metaForTab(tab), [tab])
  const [draft, setDraft] = useState<Record<string, string>>({})
  const [savingKeys, setSavingKeys] = useState<Set<string>>(new Set())
  const [error, setError] = useState<string | null>(null)
  const [bulkSaving, setBulkSaving] = useState(false)

  // Reset draft when the loaded options change (e.g. after refresh).
  useEffect(() => { setDraft({}) }, [options])

  if (tabMeta.length === 0) return null

  const groups = SETTINGS_GROUPS[tab].filter((g) =>
    tabMeta.some((m) => m.group === g.id),
  )
  const dirtyKeys = Object.keys(draft).filter((k) => draft[k] !== (options[k] ?? ''))

  const handleChange = (key: string, value: string) => {
    setDraft((prev) => ({ ...prev, [key]: value }))
  }
  const handleSaveOne = async (key: string) => {
    const next = draft[key]
    if (next === undefined) return
    setSavingKeys((s) => new Set(s).add(key))
    setError(null)
    try {
      await onSave(key, next)
      setDraft((prev) => {
        const { [key]: _drop, ...rest } = prev
        return rest
      })
    } catch (e: any) {
      setError(`${key}: ${e?.message ?? String(e)}`)
    } finally {
      setSavingKeys((s) => {
        const next = new Set(s); next.delete(key); return next
      })
    }
  }
  const handleSaveAll = async () => {
    setBulkSaving(true)
    setError(null)
    try {
      for (const k of dirtyKeys) {
        await onSave(k, draft[k])
      }
      setDraft({})
    } catch (e: any) {
      setError(e?.message ?? String(e))
    } finally {
      setBulkSaving(false)
    }
  }
  const handleRevert = () => setDraft({})

  return (
    <div className="space-y-6">
      {error && (
        <div className="flex items-start gap-2 text-xs text-red-500 bg-red-500/10 border border-red-500/20 rounded-md px-3 py-2">
          <AlertCircle className="h-3.5 w-3.5 shrink-0 mt-0.5" />
          <span className="flex-1 break-all">{error}</span>
        </div>
      )}

      {groups.map((group) => {
        const rows = tabMeta.filter((m) => m.group === group.id)
        if (rows.length === 0) return null
        return (
          <section key={group.id} className="rounded-lg border border-border bg-card">
            <header className="px-4 py-3 border-b border-border/60">
              <h3 className="text-sm font-semibold">{t(group.labelKey, group.labelFallback)}</h3>
            </header>
            <div className="divide-y divide-border/40">
              {rows.map((m) => {
                const present = m.key in options
                if (!present && !showMissing) return null
                const stored = options[m.key] ?? ''
                const drafted = draft[m.key]
                const value = drafted ?? stored
                const dirty = drafted !== undefined && drafted !== stored
                return (
                  <div key={m.key} className="px-4 py-3">
                    <FieldRow
                      meta={m}
                      value={value}
                      stored={stored}
                      dirty={dirty}
                      saving={savingKeys.has(m.key)}
                      onChange={(v) => handleChange(m.key, v)}
                      onSave={() => handleSaveOne(m.key)}
                      onSaveValue={(v) => onSave(m.key, v)}
                    />
                  </div>
                )
              })}
            </div>
          </section>
        )
      })}

      {/* Bulk save bar (sticky at bottom of the scroll container) */}
      {dirtyKeys.length > 0 && (
        <div className="sticky bottom-0 z-10 -mx-6 px-6 py-3 bg-card/95 backdrop-blur-sm border-t border-border flex items-center justify-between">
          <span className="text-xs text-muted-foreground">
            {t('gateway.settings.dirtyCount', { count: dirtyKeys.length })}
          </span>
          <div className="flex items-center gap-2">
            <button
              onClick={handleRevert}
              disabled={bulkSaving}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-md border border-border hover:bg-muted disabled:opacity-50"
            >
              <RotateCcw className="h-3.5 w-3.5" />
              {t('gateway.settings.revert', '撤销')}
            </button>
            <button
              onClick={handleSaveAll}
              disabled={bulkSaving}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-md bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              <Save className="h-3.5 w-3.5" />
              {bulkSaving
                ? t('gateway.settings.saving', '保存中…')
                : t('gateway.settings.saveAll', '保存全部修改')}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

interface FieldRowProps {
  meta: OptionMeta
  value: string
  stored: string
  dirty: boolean
  saving: boolean
  onChange: (v: string) => void
  onSave: () => void | Promise<void>
  onSaveValue: (v: string) => Promise<unknown>
}

function FieldRow({ meta, value, stored, dirty, saving, onChange, onSave, onSaveValue }: FieldRowProps) {
  const { t } = useTranslation()
  const label = t(meta.labelKey, meta.labelFallback)
  const desc = meta.descKey ? t(meta.descKey, meta.descFallback ?? '') : meta.descFallback

  // Pricing-map gets a dedicated full-width editor — no inline label row.
  if (meta.widget === 'pricing-map') {
    return (
      <div className="space-y-2">
        <div>
          <div className="text-sm font-medium">{label}</div>
          {desc && <p className="text-xs text-muted-foreground mt-0.5">{desc}</p>}
        </div>
        <PricingEditor
          optionKey={meta.key}
          rawValue={stored}
          unitHint={meta.unitHint ?? ''}
          onSave={async (k, v) => { await onSaveValue(v); void k }}
        />
      </div>
    )
  }

  // Boolean = toggle pill, full-width row with label on the left.
  if (meta.widget === 'boolean') {
    const on = value === 'true'
    return (
      <label className="flex items-start justify-between gap-4 cursor-pointer group">
        <div className="flex-1 min-w-0">
          <div className="text-sm font-medium">{label}</div>
          {desc && <p className="text-xs text-muted-foreground mt-0.5">{desc}</p>}
        </div>
        <button
          type="button"
          onClick={() => onChange(on ? 'false' : 'true')}
          className={
            'relative inline-flex h-5 w-9 shrink-0 mt-0.5 rounded-full transition-colors ' +
            (on ? 'bg-emerald-500' : 'bg-muted-foreground/30')
          }
          aria-pressed={on}
        >
          <span
            className={
              'absolute top-0.5 h-4 w-4 rounded-full bg-white shadow transition-transform ' +
              (on ? 'translate-x-4' : 'translate-x-0.5')
            }
          />
        </button>
        {dirty && (
          <button
            type="button"
            onClick={(e) => { e.preventDefault(); onSave() }}
            disabled={saving}
            className="text-[10px] text-primary hover:underline whitespace-nowrap mt-0.5 disabled:opacity-50"
          >
            {saving ? '…' : t('gateway.settings.save', '保存')}
          </button>
        )}
      </label>
    )
  }

  // Text/number/textarea/select — label above, widget below.
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between gap-2">
        <div className="flex-1 min-w-0">
          <div className="text-sm font-medium">{label}</div>
          {desc && <p className="text-xs text-muted-foreground mt-0.5">{desc}</p>}
        </div>
        {dirty && (
          <button
            type="button"
            onClick={onSave}
            disabled={saving}
            className="flex items-center gap-1 text-xs text-primary hover:underline disabled:opacity-50 shrink-0"
          >
            <Save className="h-3 w-3" />
            {saving ? t('gateway.settings.saving', '…') : t('gateway.settings.save', '保存')}
          </button>
        )}
      </div>
      {meta.widget === 'textarea' ? (
        <textarea
          value={value}
          onChange={(e) => onChange(e.target.value)}
          rows={4}
          className="w-full px-2 py-1.5 text-sm bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary resize-y font-mono"
        />
      ) : meta.widget === 'select' && meta.choices ? (
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="w-full px-2 py-1.5 text-sm bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
        >
          {meta.choices.map((c) => (
            <option key={c.value} value={c.value}>{t(c.labelKey, c.labelFallback)}</option>
          ))}
        </select>
      ) : (
        <input
          type={meta.widget === 'number' ? 'number' : 'text'}
          value={value}
          min={meta.min}
          max={meta.max}
          placeholder={meta.placeholder}
          onChange={(e) => onChange(e.target.value)}
          className="w-full px-2 py-1.5 text-sm bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
        />
      )}
    </div>
  )
}
