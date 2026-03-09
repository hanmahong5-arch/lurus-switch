import { useState, KeyboardEvent } from 'react'
import { X, Plus } from 'lucide-react'
import { cn } from '../../lib/utils'

interface TagInputProps {
  label: string
  values: string[]
  onChange: (values: string[]) => void
  placeholder?: string
  className?: string
}

/** Multi-value string input that renders each entry as a removable tag (pill). */
export function TagInput({ label, values, onChange, placeholder, className }: TagInputProps) {
  const [draft, setDraft] = useState('')

  const addTag = () => {
    const trimmed = draft.trim()
    if (trimmed && !values.includes(trimmed)) {
      onChange([...values, trimmed])
    }
    setDraft('')
  }

  const removeTag = (index: number) => {
    onChange(values.filter((_, i) => i !== index))
  }

  const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      addTag()
    } else if (e.key === 'Backspace' && draft === '' && values.length > 0) {
      removeTag(values.length - 1)
    }
  }

  return (
    <div className={cn('space-y-1', className)}>
      <label className="text-xs font-medium text-muted-foreground">{label}</label>
      <div className="min-h-[32px] flex flex-wrap gap-1 p-1.5 bg-muted/30 border border-border rounded-md focus-within:ring-1 focus-within:ring-primary">
        {values.map((tag, i) => (
          <span
            key={i}
            className="inline-flex items-center gap-1 px-2 py-0.5 bg-primary/10 text-primary text-xs rounded-full"
          >
            {tag}
            <button
              type="button"
              onClick={() => removeTag(i)}
              className="hover:text-destructive transition-colors"
              aria-label={`Remove ${tag}`}
            >
              <X className="h-2.5 w-2.5" />
            </button>
          </span>
        ))}
        <input
          type="text"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={onKeyDown}
          onBlur={addTag}
          placeholder={values.length === 0 ? (placeholder || 'Type and press Enter') : ''}
          className="flex-1 min-w-[100px] bg-transparent text-xs outline-none placeholder:text-muted-foreground/50"
        />
        {draft && (
          <button
            type="button"
            onClick={addTag}
            className="p-0.5 hover:text-primary text-muted-foreground transition-colors"
          >
            <Plus className="h-3 w-3" />
          </button>
        )}
      </div>
    </div>
  )
}
