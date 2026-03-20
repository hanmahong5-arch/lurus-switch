import { useState, useMemo } from 'react'
import { Search, Star, Zap, Globe2, Check } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'

export interface Model {
  id: string
  displayName: string
  provider: string
  inputRatio: number
  outputRatio: number
  tags: string[]
  recommended: boolean
}

interface ModelPickerProps {
  models: Model[]
  selected: string
  onSelect: (modelId: string) => void
  loading?: boolean
  compact?: boolean
}

type FilterTab = 'recommended' | 'domestic' | 'international' | 'all'

const PROVIDER_COLORS: Record<string, string> = {
  DeepSeek: 'bg-blue-500/20 text-blue-400',
  Alibaba: 'bg-orange-500/20 text-orange-400',
  Zhipu: 'bg-purple-500/20 text-purple-400',
  Anthropic: 'bg-amber-500/20 text-amber-400',
  OpenAI: 'bg-green-500/20 text-green-400',
  Google: 'bg-cyan-500/20 text-cyan-400',
  Moonshot: 'bg-pink-500/20 text-pink-400',
  '01.AI': 'bg-red-500/20 text-red-400',
}

function priceBadge(ratio: number): { label: string; color: string } {
  if (ratio <= 0.15) return { label: '$', color: 'text-green-400' }
  if (ratio <= 0.5) return { label: '$$', color: 'text-yellow-400' }
  return { label: '$$$', color: 'text-red-400' }
}

export function ModelPicker({ models, selected, onSelect, loading, compact }: ModelPickerProps) {
  const { t } = useTranslation()
  const [tab, setTab] = useState<FilterTab>('recommended')
  const [search, setSearch] = useState('')

  const filtered = useMemo(() => {
    let list = models
    if (tab === 'recommended') list = list.filter(m => m.recommended)
    else if (tab === 'domestic') list = list.filter(m => m.tags.includes('domestic'))
    else if (tab === 'international') list = list.filter(m => m.tags.includes('international'))

    if (search.trim()) {
      const q = search.toLowerCase()
      list = list.filter(m =>
        m.id.toLowerCase().includes(q) ||
        m.displayName.toLowerCase().includes(q) ||
        m.provider.toLowerCase().includes(q)
      )
    }
    return list
  }, [models, tab, search])

  const tabs: { key: FilterTab; label: string; icon: React.ReactNode }[] = [
    { key: 'recommended', label: t('model.tabs.recommended', 'Recommended'), icon: <Star className="h-3 w-3" /> },
    { key: 'domestic', label: t('model.tabs.domestic', 'Domestic'), icon: <Zap className="h-3 w-3" /> },
    { key: 'international', label: t('model.tabs.international', 'International'), icon: <Globe2 className="h-3 w-3" /> },
    { key: 'all', label: t('model.tabs.all', 'All'), icon: null },
  ]

  return (
    <div className="space-y-3">
      {/* Tabs */}
      <div className="flex gap-1.5">
        {tabs.map(t => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={cn(
              'flex items-center gap-1 px-2.5 py-1 rounded-full text-xs transition-colors',
              tab === t.key
                ? 'bg-primary text-primary-foreground'
                : 'bg-muted text-muted-foreground hover:text-foreground'
            )}
          >
            {t.icon}
            {t.label}
          </button>
        ))}
      </div>

      {/* Search (only in "all" tab) */}
      {tab === 'all' && (
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
          <input
            type="text"
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder={t('model.search', 'Search models...')}
            className="w-full pl-8 pr-3 py-1.5 text-sm rounded-md border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
      )}

      {/* Model grid */}
      {loading ? (
        <div className="py-6 text-center text-sm text-muted-foreground">
          {t('status.loading')}
        </div>
      ) : filtered.length === 0 ? (
        <div className="py-6 text-center text-sm text-muted-foreground">
          {t('model.noResults', 'No models found')}
        </div>
      ) : (
        <div className={cn('grid gap-2', compact ? 'grid-cols-1' : 'grid-cols-2')}>
          {filtered.map(model => {
            const isSelected = model.id === selected
            const price = priceBadge(model.inputRatio)
            const provColor = PROVIDER_COLORS[model.provider] || 'bg-muted text-muted-foreground'

            return (
              <button
                key={model.id}
                onClick={() => onSelect(model.id)}
                className={cn(
                  'relative flex flex-col gap-1 p-3 rounded-lg border text-left transition-all',
                  isSelected
                    ? 'border-primary bg-primary/5 ring-1 ring-primary/30'
                    : 'border-border hover:border-primary/40 hover:bg-muted/30'
                )}
              >
                {isSelected && (
                  <div className="absolute top-2 right-2">
                    <Check className="h-4 w-4 text-primary" />
                  </div>
                )}
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">{model.displayName}</span>
                  {model.recommended && <Star className="h-3 w-3 text-amber-400 fill-amber-400" />}
                </div>
                <div className="flex items-center gap-1.5 flex-wrap">
                  <span className={cn('px-1.5 py-0.5 rounded text-[10px] font-medium', provColor)}>
                    {model.provider}
                  </span>
                  <span className={cn('text-[10px] font-mono', price.color)}>
                    {price.label}
                  </span>
                </div>
                <span className="text-[10px] text-muted-foreground font-mono">{model.id}</span>
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}
