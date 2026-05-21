import { useEffect, useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2, Pencil, Server, ExternalLink, Loader2 } from 'lucide-react'
import { ListCustomProviders, DeleteCustomProvider } from '../../wailsjs/go/main/App'
import { CustomProviderForm, type CustomProvider } from './CustomProviderForm'

// Settings → Providers tab body. Lists user-defined providers with edit /
// delete, and an "Add" flow backed by CustomProviderForm.
export function CustomProvidersSection() {
  const { t } = useTranslation()
  const [providers, setProviders] = useState<CustomProvider[]>([])
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState<CustomProvider | null>(null)
  const [adding, setAdding] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const list = (await ListCustomProviders()) as CustomProvider[]
      setProviders(list || [])
    } catch {
      setProviders([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleDelete = async (id: string) => {
    setDeletingId(id)
    try {
      await DeleteCustomProvider(id)
      await load()
    } finally {
      setDeletingId(null)
    }
  }

  const showForm = adding || editing !== null

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-medium">{t('customProvider.sectionTitle', '自定义供应商')}</h3>
          <p className="text-xs text-muted-foreground">
            {t('customProvider.sectionDesc', '接入任意 OpenAI 兼容端点（私有部署、企业内网等）。')}
          </p>
        </div>
        {!showForm && (
          <button
            onClick={() => setAdding(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-md bg-primary text-primary-foreground hover:bg-primary/90"
          >
            <Plus className="h-3.5 w-3.5" />
            {t('customProvider.add', '添加')}
          </button>
        )}
      </div>

      {showForm && (
        <CustomProviderForm
          initial={editing}
          onSaved={() => {
            setAdding(false)
            setEditing(null)
            load()
          }}
          onCancel={() => {
            setAdding(false)
            setEditing(null)
          }}
        />
      )}

      {loading ? (
        <div className="flex items-center justify-center py-8 text-muted-foreground">
          <Loader2 className="h-5 w-5 animate-spin" />
        </div>
      ) : providers.length === 0 && !showForm ? (
        <div className="text-center py-8 text-xs text-muted-foreground border border-dashed border-border rounded-lg">
          {t('customProvider.empty', '尚未添加自定义供应商')}
        </div>
      ) : (
        <div className="space-y-2">
          {providers.map((p) => (
            <div
              key={p.id}
              className="flex items-center gap-3 p-3 rounded-lg border border-border bg-card"
            >
              <div className="w-8 h-8 rounded-lg flex items-center justify-center bg-muted shrink-0">
                <Server className="h-4 w-4 text-muted-foreground" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium truncate">{p.name}</div>
                <div className="text-[11px] text-muted-foreground font-mono truncate">{p.baseUrl}</div>
                {p.defaultModels?.length > 0 && (
                  <div className="text-[10px] text-muted-foreground/70 truncate">
                    {p.defaultModels.join(', ')}
                  </div>
                )}
              </div>
              {p.docsUrl && (
                <a
                  href={p.docsUrl}
                  target="_blank"
                  rel="noreferrer"
                  className="p-1 rounded hover:bg-muted text-muted-foreground"
                  title={t('customProvider.docs', '文档')}
                >
                  <ExternalLink className="h-3.5 w-3.5" />
                </a>
              )}
              <button
                onClick={() => { setEditing(p); setAdding(false) }}
                className="p-1 rounded hover:bg-muted text-muted-foreground"
                title={t('common.edit', '编辑')}
              >
                <Pencil className="h-3.5 w-3.5" />
              </button>
              <button
                onClick={() => handleDelete(p.id)}
                disabled={deletingId === p.id}
                className="p-1 rounded hover:bg-red-500/10 text-red-500 disabled:opacity-50"
                title={t('common.delete', '删除')}
              >
                {deletingId === p.id
                  ? <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  : <Trash2 className="h-3.5 w-3.5" />}
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
