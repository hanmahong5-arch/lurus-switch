import { useState, useEffect, useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, ChevronRight, Star, Globe, Server, Cloud, Laptop, X, Trash2, Wrench } from 'lucide-react'
import { cn } from '../lib/utils'
import { GetProviderPresets, ListCustomProviders, DeleteCustomProvider } from '../../wailsjs/go/main/App'
import type { CustomProvider } from './CustomProviderForm'

interface ProviderPreset {
  id: string
  name: string
  icon: string
  iconColor: string
  category: string
  baseUrl: string
  keyFormat: string
  docsUrl: string
  models: string
  description: string
  // True only for user-defined providers — drives the delete affordance.
  custom?: boolean
}

interface ProviderPickerProps {
  onSelect: (preset: ProviderPreset) => void
  onClose: () => void
}

const CATEGORY_ORDER = ['official', 'china', 'proxy', 'cloud', 'self-hosted', 'custom'] as const

const CATEGORY_META: Record<string, { label: string; labelZh: string; icon: typeof Star }> = {
  official:      { label: 'Official',    labelZh: '官方',     icon: Star },
  china:         { label: 'China',       labelZh: '国内',     icon: Globe },
  proxy:         { label: 'Aggregator',  labelZh: '聚合平台', icon: Server },
  cloud:         { label: 'Cloud',       labelZh: '云平台',   icon: Cloud },
  'self-hosted': { label: 'Self-Hosted', labelZh: '本地部署', icon: Laptop },
  custom:        { label: 'Custom',      labelZh: '自定义',   icon: Wrench },
}

// Map a user-defined provider into the preset shape the picker renders.
function customToPreset(c: CustomProvider): ProviderPreset {
  return {
    id: c.id,
    name: c.name,
    icon: 'custom',
    iconColor: '#6B7280',
    category: 'custom',
    baseUrl: c.baseUrl,
    keyFormat: 'sk-...',
    docsUrl: c.docsUrl ?? '',
    models: (c.defaultModels ?? []).join(', '),
    description: c.description ?? c.baseUrl,
    custom: true,
  }
}

export function ProviderPicker({ onSelect, onClose }: ProviderPickerProps) {
  const { t, i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh')
  const [presets, setPresets] = useState<ProviderPreset[]>([])
  const [customs, setCustoms] = useState<ProviderPreset[]>([])
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)

  const loadCustoms = useCallback(async () => {
    try {
      const list = (await ListCustomProviders()) as CustomProvider[]
      setCustoms((list || []).map(customToPreset))
    } catch {
      setCustoms([])
    }
  }, [])

  useEffect(() => {
    Promise.all([
      GetProviderPresets()
        .then((data: ProviderPreset[]) => setPresets(data || []))
        .catch(() => {}),
      loadCustoms(),
    ]).finally(() => setLoading(false))
  }, [loadCustoms])

  const handleDeleteCustom = async (id: string) => {
    try {
      await DeleteCustomProvider(id)
      await loadCustoms()
    } catch {
      // best-effort; surface nothing — list reloads on next open
    }
  }

  const allPresets = useMemo(() => [...presets, ...customs], [presets, customs])

  const filtered = useMemo(() => {
    if (!search.trim()) return allPresets
    const q = search.toLowerCase()
    return allPresets.filter(
      p => p.name.toLowerCase().includes(q) ||
           p.id.toLowerCase().includes(q) ||
           p.description.toLowerCase().includes(q) ||
           p.models.toLowerCase().includes(q)
    )
  }, [allPresets, search])

  const grouped = useMemo(() => {
    const map = new Map<string, ProviderPreset[]>()
    for (const cat of CATEGORY_ORDER) map.set(cat, [])
    for (const p of filtered) {
      const arr = map.get(p.category)
      if (arr) arr.push(p)
    }
    return map
  }, [filtered])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm" onClick={onClose}>
      <div
        className="w-full max-w-2xl max-h-[80vh] bg-card border border-border rounded-xl shadow-2xl flex flex-col overflow-hidden"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <h2 className="text-base font-semibold">
            {isZh ? '选择 API 供应商' : 'Choose API Provider'}
          </h2>
          <button onClick={onClose} className="p-1 rounded hover:bg-muted">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Search */}
        <div className="px-4 py-2 border-b border-border">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <input
              autoFocus
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder={isZh ? '搜索供应商、模型...' : 'Search providers, models...'}
              className="w-full pl-8 pr-3 py-1.5 text-sm rounded-md border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
        </div>

        {/* Provider list */}
        <div className="flex-1 overflow-y-auto px-4 py-2">
          {loading ? (
            <div className="flex items-center justify-center py-12 text-muted-foreground text-sm">
              {isZh ? '加载中...' : 'Loading...'}
            </div>
          ) : filtered.length === 0 ? (
            <div className="flex items-center justify-center py-12 text-muted-foreground text-sm">
              {isZh ? '无匹配结果' : 'No matches'}
            </div>
          ) : (
            CATEGORY_ORDER.map(cat => {
              const items = grouped.get(cat) || []
              if (items.length === 0) return null
              const meta = CATEGORY_META[cat]
              const CatIcon = meta.icon
              return (
                <div key={cat} className="mb-4">
                  <div className="flex items-center gap-1.5 mb-2">
                    <CatIcon className="h-3.5 w-3.5 text-muted-foreground" />
                    <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                      {isZh ? meta.labelZh : meta.label}
                    </span>
                    <span className="text-xs text-muted-foreground/50">({items.length})</span>
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                    {items.map(preset => (
                      <div
                        key={preset.id}
                        className={cn(
                          'relative flex items-center gap-3 p-3 rounded-lg border border-border',
                          'hover:border-primary/50 hover:bg-muted/50 transition-colors group'
                        )}
                      >
                        <button
                          onClick={() => onSelect(preset)}
                          className="flex items-center gap-3 flex-1 min-w-0 text-left"
                        >
                          {/* Icon dot */}
                          <div
                            className="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 text-xs font-bold text-white"
                            style={{ backgroundColor: preset.iconColor || '#6B7280' }}
                          >
                            {preset.name.charAt(0)}
                          </div>

                          {/* Info */}
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-1.5">
                              <span className="text-sm font-medium truncate">{preset.name}</span>
                              {preset.id === 'lurus' && (
                                <span className="px-1 py-0.5 text-[10px] font-medium rounded bg-primary/10 text-primary">
                                  {isZh ? '推荐' : 'Recommended'}
                                </span>
                              )}
                            </div>
                            <p className="text-xs text-muted-foreground truncate">{preset.description}</p>
                          </div>
                        </button>

                        {preset.custom ? (
                          <button
                            onClick={(e) => { e.stopPropagation(); handleDeleteCustom(preset.id) }}
                            className="p-1 rounded hover:bg-red-500/10 text-red-500 flex-shrink-0"
                            title={isZh ? '删除' : 'Delete'}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        ) : (
                          <ChevronRight className="h-4 w-4 text-muted-foreground/30 group-hover:text-primary transition-colors flex-shrink-0" />
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )
            })
          )}
        </div>

        {/* Footer hint */}
        <div className="px-4 py-2 border-t border-border text-xs text-muted-foreground text-center">
          {isZh
            ? '选择供应商后自动填入 API 地址，只需输入 API Key 即可使用'
            : 'Select a provider to auto-fill the API endpoint — just enter your API key'}
        </div>
      </div>
    </div>
  )
}
