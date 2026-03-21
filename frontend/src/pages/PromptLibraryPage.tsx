import { useEffect, useState, useCallback } from 'react'
import { Plus, Trash2, Copy, Search, Loader2, Download, Upload, BookOpen } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useClassifiedError } from '../lib/useClassifiedError'
import { InlineError } from '../components/InlineError'
import {
  ListPrompts, SavePrompt, DeletePrompt, GetBuiltinPrompts,
  ExportPrompts, ImportPrompts,
} from '../../wailsjs/go/main/App'

interface Prompt {
  id: string
  name: string
  category: string
  tags: string[]
  content: string
  targetTools: string[]
  createdAt: string
  updatedAt: string
}

const CATEGORIES = ['all', 'coding', 'writing', 'analysis', 'custom'] as const
type Category = typeof CATEGORIES[number]

const CATEGORY_I18N_KEYS: Record<Category, string> = {
  all: 'promptLib.categories.all',
  coding: 'promptLib.categories.coding',
  writing: 'promptLib.categories.writing',
  analysis: 'promptLib.categories.analysis',
  custom: 'promptLib.categories.custom',
}

export function PromptLibraryPage() {
  const { t } = useTranslation()
  const [prompts, setPrompts] = useState<Prompt[]>([])
  const [builtins, setBuiltins] = useState<Prompt[]>([])
  const [category, setCategory] = useState<Category>('all')
  const [search, setSearch] = useState('')
  const [selected, setSelected] = useState<Prompt | null>(null)
  const [loading, setLoading] = useState(true)
  const { classified: error, setError, clearError } = useClassifiedError()
  const [showEditor, setShowEditor] = useState(false)
  const [editPrompt, setEditPrompt] = useState<Partial<Prompt>>({})
  const [saving, setSaving] = useState(false)
  const [copied, setCopied] = useState(false)

  const loadPrompts = useCallback(async () => {
    setLoading(true)
    try {
      const [userPrompts, builtin] = await Promise.all([
        ListPrompts(''),
        GetBuiltinPrompts(),
      ])
      setPrompts(userPrompts || [])
      setBuiltins(builtin || [])
    } catch (err) {
      setError(err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { loadPrompts() }, [loadPrompts])

  const allPrompts = [...builtins, ...prompts]

  const filtered = allPrompts.filter((p) => {
    const matchCat = category === 'all' || p.category === category
    const matchSearch = search === '' ||
      p.name.toLowerCase().includes(search.toLowerCase()) ||
      p.content.toLowerCase().includes(search.toLowerCase()) ||
      p.tags?.some((t) => t.toLowerCase().includes(search.toLowerCase()))
    return matchCat && matchSearch
  })

  const handleDelete = async (id: string) => {
    try {
      await DeletePrompt(id)
      if (selected?.id === id) setSelected(null)
      await loadPrompts()
    } catch (err) {
      setError(err)
    }
  }

  const handleSave = async () => {
    setSaving(true)
    try {
      await SavePrompt({
        id: editPrompt.id || '',
        name: editPrompt.name || 'Untitled',
        category: editPrompt.category || 'custom',
        tags: editPrompt.tags || [],
        content: editPrompt.content || '',
        targetTools: editPrompt.targetTools || ['all'],
        createdAt: editPrompt.createdAt || '',
        updatedAt: '',
      })
      setShowEditor(false)
      setEditPrompt({})
      await loadPrompts()
    } catch (err) {
      setError(err)
    } finally {
      setSaving(false)
    }
  }

  const handleCopy = async () => {
    if (selected) {
      await navigator.clipboard.writeText(selected.content)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  const handleExport = async () => {
    try {
      await ExportPrompts()
    } catch (err) {
      setError(err)
    }
  }

  const handleImport = async () => {
    try {
      const count = await ImportPrompts()
      await loadPrompts()
      clearError()
      alert(`Successfully imported ${count} prompts.`)
    } catch (err) {
      setError(err)
    }
  }

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="h-full flex overflow-hidden">
      {/* Left: Category + Search */}
      <div className="w-48 border-r border-border bg-muted/30 flex flex-col shrink-0">
        <div className="p-3 border-b border-border">
          <h2 className="text-sm font-semibold flex items-center gap-2">
            <BookOpen className="h-4 w-4 text-purple-400" />
            {t('promptLib.title')}
          </h2>
        </div>
        <div className="p-2 space-y-1">
          {CATEGORIES.map((cat) => (
            <button
              key={cat}
              onClick={() => setCategory(cat)}
              className={cn(
                'w-full text-left px-3 py-2 rounded-md text-sm transition-colors',
                category === cat
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground'
              )}
            >
              {t(CATEGORY_I18N_KEYS[cat])}
            </button>
          ))}
        </div>
        <div className="mt-auto p-2 space-y-1 border-t border-border">
          <button
            onClick={handleExport}
            className="w-full flex items-center gap-2 px-3 py-2 text-xs text-muted-foreground hover:text-foreground hover:bg-muted rounded-md"
          >
            <Download className="h-3.5 w-3.5" /> {t('promptLib.exportBtn')}
          </button>
          <button
            onClick={handleImport}
            className="w-full flex items-center gap-2 px-3 py-2 text-xs text-muted-foreground hover:text-foreground hover:bg-muted rounded-md"
          >
            <Upload className="h-3.5 w-3.5" /> {t('promptLib.importBtn')}
          </button>
        </div>
      </div>

      {/* Middle: Prompt List */}
      <div className="w-64 border-r border-border flex flex-col shrink-0">
        <div className="p-2 border-b border-border space-y-2">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('promptLib.searchPlaceholder')}
              className="w-full pl-8 pr-3 py-1.5 text-xs bg-muted border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
          <button
            onClick={() => {
              setEditPrompt({ category: 'custom', targetTools: ['all'] })
              setShowEditor(true)
            }}
            className="w-full flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-md text-xs bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
          >
            <Plus className="h-3.5 w-3.5" /> {t('promptLib.newPrompt')}
          </button>
        </div>

        {error && (
          <InlineError
            category={error.category}
            message={error.message}
            details={error.details}
            onDismiss={clearError}
          />
        )}

        <div className="flex-1 overflow-y-auto">
          {filtered.map((p) => (
            <button
              key={p.id}
              onClick={() => { setSelected(p); setShowEditor(false) }}
              className={cn(
                'w-full text-left px-3 py-3 border-b border-border hover:bg-muted/50 transition-colors',
                selected?.id === p.id && 'bg-muted'
              )}
            >
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="text-xs font-medium truncate">{p.name}</p>
                  <p className="text-xs text-muted-foreground mt-0.5 line-clamp-2">
                    {p.content.slice(0, 60)}{p.content.length > 60 ? '...' : ''}
                  </p>
                  <div className="flex flex-wrap gap-1 mt-1">
                    {p.tags?.slice(0, 2).map((t) => (
                      <span key={t} className="text-xs bg-muted px-1.5 py-0.5 rounded">
                        {t}
                      </span>
                    ))}
                  </div>
                </div>
                {!p.id.startsWith('builtin-') && (
                  <button
                    onClick={(e) => { e.stopPropagation(); handleDelete(p.id) }}
                    className="p-1 text-muted-foreground hover:text-red-500 transition-colors shrink-0"
                  >
                    <Trash2 className="h-3 w-3" />
                  </button>
                )}
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* Right: Detail / Editor */}
      <div className="flex-1 overflow-y-auto">
        {showEditor ? (
          <div className="p-6 space-y-4">
            <h3 className="font-semibold">{editPrompt.id ? t('promptLib.editPrompt') : t('promptLib.newPrompt')}</h3>
            <div className="space-y-3">
              <input
                type="text"
                value={editPrompt.name || ''}
                onChange={(e) => setEditPrompt({ ...editPrompt, name: e.target.value })}
                placeholder={t('promptLib.namePlaceholder')}
                className="w-full px-3 py-2 text-sm bg-muted border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <select
                value={editPrompt.category || 'custom'}
                onChange={(e) => setEditPrompt({ ...editPrompt, category: e.target.value })}
                className="w-full px-3 py-2 text-sm bg-muted border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
              >
                {CATEGORIES.filter((c) => c !== 'all').map((c) => (
                  <option key={c} value={c}>{t(CATEGORY_I18N_KEYS[c])}</option>
                ))}
              </select>
              <textarea
                value={editPrompt.content || ''}
                onChange={(e) => setEditPrompt({ ...editPrompt, content: e.target.value })}
                placeholder={t('promptLib.contentPlaceholder')}
                rows={12}
                className="w-full px-3 py-2 text-sm bg-muted border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary resize-none font-mono"
              />
              <div className="flex gap-2">
                <button
                  onClick={() => { setShowEditor(false); setEditPrompt({}) }}
                  className="px-4 py-2 text-sm border border-border rounded-md hover:bg-muted transition-colors"
                >
                  {t('promptLib.cancel')}
                </button>
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                  {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : t('promptLib.save')}
                </button>
              </div>
            </div>
          </div>
        ) : selected ? (
          <div className="p-6 space-y-4">
            <div className="flex items-start justify-between">
              <div>
                <h3 className="font-semibold">{selected.name}</h3>
                <div className="flex items-center gap-2 mt-1">
                  <span className="text-xs bg-muted px-2 py-0.5 rounded">{selected.category}</span>
                  {selected.tags?.map((t) => (
                    <span key={t} className="text-xs bg-muted px-2 py-0.5 rounded text-muted-foreground">{t}</span>
                  ))}
                </div>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={handleCopy}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-xs border border-border rounded-md hover:bg-muted transition-colors"
                >
                  <Copy className="h-3.5 w-3.5" />
                  {copied ? t('promptLib.copied') : t('promptLib.copy')}
                </button>
                {!selected.id.startsWith('builtin-') && (
                  <button
                    onClick={() => {
                      setEditPrompt({ ...selected })
                      setShowEditor(true)
                    }}
                    className="flex items-center gap-1.5 px-3 py-1.5 text-xs border border-border rounded-md hover:bg-muted transition-colors"
                  >
                    {t('promptLib.edit')}
                  </button>
                )}
              </div>
            </div>
            <div className="bg-muted/50 rounded-lg p-4">
              <pre className="text-sm whitespace-pre-wrap font-sans">{selected.content}</pre>
            </div>
          </div>
        ) : (
          <div className="h-full flex items-center justify-center text-muted-foreground text-sm">
            {t('promptLib.selectHint')}
          </div>
        )}
      </div>
    </div>
  )
}
