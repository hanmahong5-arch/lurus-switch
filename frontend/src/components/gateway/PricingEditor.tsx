import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, Plus, Trash2, Save, AlertCircle } from 'lucide-react'

interface PricingEditorProps {
  /** newapi option key, e.g. "ModelRatio" / "CompletionRatio" / "GroupRatio". */
  optionKey: string
  /** Raw value from /api/option/ — JSON string mapping model→ratio. */
  rawValue: string
  /** Hint about what the row means in this map (input price multiplier, etc.). */
  unitHint: string
  onSave: (key: string, value: string) => Promise<void>
}

/**
 * Per-row editor for newapi's JSON-map pricing options. The on-disk value is
 * a JSON string `{ "model-id": ratio_number, ... }`. Editing as raw text is
 * error-prone — one misplaced comma blanks the whole pricing surface — so
 * we parse, render rows, and re-serialize on save.
 *
 * Falls back to a textarea when the value isn't a flat string→number map
 * so callers can still edit malformed pricing without losing data.
 */
export function PricingEditor({ optionKey, rawValue, unitHint, onSave }: PricingEditorProps) {
  const { t } = useTranslation()
  const initialMap = useMemo(() => parseMap(rawValue), [rawValue])
  const [rows, setRows] = useState<Array<{ model: string; ratio: string }>>(
    () => initialMap === null ? [] : Object.entries(initialMap).map(([m, r]) => ({ model: m, ratio: String(r) })),
  )
  const [search, setSearch] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  // Track raw-mode fallback for malformed JSON.
  const [rawMode, setRawMode] = useState(initialMap === null)
  const [rawText, setRawText] = useState(rawValue)

  const dirty = useMemo(() => {
    if (rawMode) return rawText !== rawValue
    const current = Object.fromEntries(rows.filter((r) => r.model.trim()).map((r) => [r.model.trim(), Number(r.ratio)]))
    const original = initialMap ?? {}
    if (Object.keys(current).length !== Object.keys(original).length) return true
    for (const [k, v] of Object.entries(current)) {
      if (original[k] !== v) return true
    }
    return false
  }, [rows, initialMap, rawMode, rawText, rawValue])

  const filteredRows = useMemo(() => {
    if (!search.trim()) return rows
    const q = search.trim().toLowerCase()
    return rows.filter((r) => r.model.toLowerCase().includes(q))
  }, [rows, search])

  const handleAdd = () => {
    setRows((prev) => [...prev, { model: '', ratio: '1' }])
  }
  const handleRemove = (idx: number) => {
    setRows((prev) => prev.filter((_, i) => i !== idx))
  }
  const handleEdit = (idx: number, field: 'model' | 'ratio', val: string) => {
    setRows((prev) => prev.map((r, i) => (i === idx ? { ...r, [field]: val } : r)))
  }

  const handleSave = async () => {
    setSaving(true)
    setError(null)
    try {
      let serialized: string
      if (rawMode) {
        // Validate the raw JSON before send so we don't push garbage upstream.
        try {
          JSON.parse(rawText)
        } catch (e: any) {
          throw new Error('Invalid JSON: ' + (e?.message ?? String(e)))
        }
        serialized = rawText
      } else {
        const map: Record<string, number> = {}
        for (const r of rows) {
          const k = r.model.trim()
          if (!k) continue
          const n = Number(r.ratio)
          if (!Number.isFinite(n)) throw new Error(`Invalid ratio for "${k}": ${r.ratio}`)
          map[k] = n
        }
        serialized = JSON.stringify(map)
      }
      await onSave(optionKey, serialized)
    } catch (e: any) {
      setError(e?.message ?? String(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-2">
        <div className="text-xs text-muted-foreground">{unitHint}</div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setRawMode((v) => !v)}
            className="text-xs text-muted-foreground hover:text-foreground underline-offset-2 hover:underline"
          >
            {rawMode ? t('gateway.settings.editor.visual', '可视化编辑') : t('gateway.settings.editor.raw', '原始 JSON')}
          </button>
          {dirty && (
            <button
              onClick={handleSave}
              disabled={saving}
              className="flex items-center gap-1 px-3 py-1 text-xs rounded-md bg-primary text-primary-foreground disabled:opacity-50"
            >
              <Save className="h-3 w-3" />
              {t('gateway.settings.save', 'Save')}
            </button>
          )}
        </div>
      </div>

      {error && (
        <div className="flex items-start gap-2 text-xs text-red-500 bg-red-500/10 border border-red-500/20 rounded-md px-3 py-2">
          <AlertCircle className="h-3.5 w-3.5 shrink-0 mt-0.5" />
          <span className="flex-1 break-all">{error}</span>
        </div>
      )}

      {rawMode ? (
        <textarea
          value={rawText}
          onChange={(e) => setRawText(e.target.value)}
          rows={10}
          className="w-full px-2 py-1.5 text-xs font-mono bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
        />
      ) : (
        <>
          <div className="flex items-center gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={t('gateway.settings.editor.searchModel', '搜索模型名…')}
                className="w-full pl-7 pr-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
            <button
              onClick={handleAdd}
              className="flex items-center gap-1 px-2.5 py-1.5 text-xs rounded-md border border-border hover:bg-muted"
            >
              <Plus className="h-3 w-3" />
              {t('gateway.settings.editor.addModel', '添加模型')}
            </button>
            <span className="text-[10px] text-muted-foreground tabular-nums">
              {filteredRows.length} / {rows.length}
            </span>
          </div>

          <div className="rounded-md border border-border bg-background/30 max-h-96 overflow-y-auto">
            {filteredRows.length === 0 ? (
              <div className="text-xs text-muted-foreground py-6 text-center">
                {rows.length === 0
                  ? t('gateway.settings.editor.noEntries', '暂无条目，点击"添加模型"开始')
                  : t('gateway.settings.editor.noMatch', '没有匹配的模型')}
              </div>
            ) : (
              <div className="divide-y divide-border/50">
                {filteredRows.map((row, _filtIdx) => {
                  const realIdx = rows.indexOf(row)
                  return (
                    <div key={realIdx} className="flex items-center gap-2 px-2 py-1.5">
                      <input
                        type="text"
                        value={row.model}
                        onChange={(e) => handleEdit(realIdx, 'model', e.target.value)}
                        placeholder="model-id"
                        className="flex-1 px-2 py-1 text-xs font-mono bg-muted/30 border border-border rounded focus:outline-none focus:ring-1 focus:ring-primary"
                      />
                      <input
                        type="text"
                        value={row.ratio}
                        onChange={(e) => handleEdit(realIdx, 'ratio', e.target.value)}
                        className="w-24 px-2 py-1 text-xs font-mono tabular-nums bg-muted/30 border border-border rounded focus:outline-none focus:ring-1 focus:ring-primary text-right"
                      />
                      <button
                        onClick={() => handleRemove(realIdx)}
                        className="p-1 rounded hover:bg-destructive/10 text-destructive"
                        title={t('gateway.settings.editor.removeRow', 'Remove')}
                      >
                        <Trash2 className="h-3 w-3" />
                      </button>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  )
}

function parseMap(raw: string): Record<string, number> | null {
  if (!raw || !raw.trim()) return {}
  try {
    const parsed = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return null
    const out: Record<string, number> = {}
    for (const [k, v] of Object.entries(parsed)) {
      if (typeof v === 'number' && Number.isFinite(v)) out[k] = v
      else if (typeof v === 'string' && Number.isFinite(Number(v))) out[k] = Number(v)
      else return null // non-numeric value — fall back to raw mode
    }
    return out
  } catch {
    return null
  }
}
