import { useState } from 'react'
import { Save } from 'lucide-react'

interface OptionEditorProps {
  options: Record<string, string>
  onSave: (key: string, value: string) => Promise<void>
}

function detectType(value: string): 'boolean' | 'number' | 'json' | 'text' {
  if (value === 'true' || value === 'false') return 'boolean'
  if (value !== '' && !isNaN(Number(value))) return 'number'
  try {
    const parsed = JSON.parse(value)
    if (typeof parsed === 'object') return 'json'
  } catch { /* not json */ }
  return 'text'
}

function OptionRow({ k, v, onSave }: { k: string; v: string; onSave: (key: string, val: string) => Promise<void> }) {
  const [value, setValue] = useState(v)
  const [saving, setSaving] = useState(false)
  const changed = value !== v
  const type = detectType(v)

  const handleSave = async () => {
    setSaving(true)
    try {
      await onSave(k, value)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex items-start gap-3 py-2">
      <div className="min-w-[200px] text-xs font-mono text-muted-foreground pt-1.5 break-all">{k}</div>
      <div className="flex-1">
        {type === 'boolean' ? (
          <button
            onClick={() => setValue(value === 'true' ? 'false' : 'true')}
            className={`px-3 py-1 rounded text-xs font-medium ${
              value === 'true' ? 'bg-green-900/40 text-green-400' : 'bg-muted text-muted-foreground'
            }`}
          >
            {value}
          </button>
        ) : type === 'json' ? (
          <textarea
            value={value}
            onChange={(e) => setValue(e.target.value)}
            rows={3}
            className="w-full px-2 py-1.5 text-xs font-mono bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary resize-y"
          />
        ) : (
          <input
            type={type === 'number' ? 'number' : 'text'}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
          />
        )}
      </div>
      {changed && (
        <button
          onClick={handleSave}
          disabled={saving}
          className="p-1.5 rounded hover:bg-muted text-indigo-400 disabled:opacity-50"
          title="Save"
        >
          <Save className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  )
}

export function OptionEditor({ options, onSave }: OptionEditorProps) {
  const entries = Object.entries(options).sort(([a], [b]) => a.localeCompare(b))

  if (entries.length === 0) {
    return <p className="text-sm text-muted-foreground py-4">No options available.</p>
  }

  return (
    <div className="divide-y divide-border">
      {entries.map(([k, v]) => (
        <OptionRow key={k} k={k} v={v} onSave={onSave} />
      ))}
    </div>
  )
}
