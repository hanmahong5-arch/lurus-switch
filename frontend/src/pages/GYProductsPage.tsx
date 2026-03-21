import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, RefreshCw, ExternalLink, Download, Play } from 'lucide-react'
import { cn } from '../lib/utils'
import { useClassifiedError } from '../lib/useClassifiedError'
import { InlineError } from '../components/InlineError'
import { useGYStore } from '../stores/gyStore'
import { GetGYProducts, CheckGYStatus, LaunchGYProduct, DownloadCreator } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import type { gy } from '../../wailsjs/go/models'

const PRODUCT_ICONS: Record<string, string> = {
  'lurus-lucrum': '🔮',
  'lurus-creator': '🎨',
  'lurus-memorus': '🧠',
}

const KIND_BADGES: Record<string, { label: string; color: string }> = {
  web: { label: 'gyProducts.categories.web', color: 'bg-blue-500/10 text-blue-500' },
  desktop: { label: 'gyProducts.categories.desktop', color: 'bg-violet-500/10 text-violet-500' },
  service: { label: 'gyProducts.categories.service', color: 'bg-teal-500/10 text-teal-500' },
}

function StatusDot({ status }: { status: gy.GYStatus | undefined }) {
  const { t } = useTranslation()
  if (!status) return <span className="text-xs text-muted-foreground/60">{t('gyProducts.checking')}</span>
  if (status.available) {
    return (
      <span className="flex items-center gap-1 text-xs text-green-500">
        <span className="h-1.5 w-1.5 rounded-full bg-green-500 inline-block" />
        {t('gyProducts.online')}
        {status.latencyMs > 0 && <span className="text-muted-foreground">({status.latencyMs}ms)</span>}
      </span>
    )
  }
  return (
    <span className="flex items-center gap-1 text-xs text-red-500">
      <span className="h-1.5 w-1.5 rounded-full bg-red-500 inline-block" />
      {t('gyProducts.unreachable')}
    </span>
  )
}

function InstalledBadge({ status }: { status: gy.GYStatus | undefined }) {
  const { t } = useTranslation()
  if (!status) return null
  if (status.version) {
    return <span className="text-xs text-green-500">{t('gyProducts.installedVersion', { version: status.version })}</span>
  }
  return <span className="text-xs text-muted-foreground">{t('gyProducts.notInstalled')}</span>
}

export function GYProductsPage() {
  const { t } = useTranslation()
  const { products, statuses, loading, checking, setProducts, setStatuses, setLoading, setChecking } = useGYStore()
  const [launching, setLaunching] = useState<string | null>(null)
  const [downloading, setDownloading] = useState(false)
  const [creatorProgress, setCreatorProgress] = useState(-1) // -1=idle, 0-100=downloading
  const { classified: error, setError, clearError } = useClassifiedError()

  // Subscribe to Creator download-progress events from the Go backend.
  useEffect(() => {
    const off = EventsOn('gy:creator:progress', (d: { percent: number }) => {
      setCreatorProgress(d.percent ?? 0)
    })
    return () => { off() }
  }, [])

  const loadProducts = async () => {
    setLoading(true)
    try {
      const ps = await GetGYProducts()
      setProducts(ps || [])
    } catch (err) {
      setError(err)
    } finally {
      setLoading(false)
    }
  }

  const checkStatus = async () => {
    setChecking(true)
    clearError()
    try {
      const ss = await CheckGYStatus()
      const map: Record<string, gy.GYStatus> = {}
      for (const s of ss || []) {
        map[s.productId] = s
      }
      setStatuses(map)
    } catch (err) {
      setError(err)
    } finally {
      setChecking(false)
    }
  }

  useEffect(() => {
    loadProducts().then(checkStatus)
  }, [])

  const handleLaunch = async (productId: string) => {
    setLaunching(productId)
    clearError()
    try {
      await LaunchGYProduct(productId)
    } catch (err) {
      setError(err)
    } finally {
      setLaunching(null)
    }
  }

  const handleDownloadCreator = async () => {
    setDownloading(true)
    setCreatorProgress(0)
    clearError()
    try {
      await DownloadCreator()
      setCreatorProgress(100)
    } catch (err) {
      setCreatorProgress(-1)
      setError(err)
    } finally {
      setDownloading(false)
    }
  }

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-3xl mx-auto p-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold">{t('gyProducts.title')}</h2>
            <p className="text-sm text-muted-foreground mt-0.5">{t('gyProducts.subtitle')}</p>
          </div>
          <button
            onClick={checkStatus}
            disabled={checking}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
              'border border-border hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            {checking ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
            {t('gyProducts.refreshStatus')}
          </button>
        </div>

        {/* Error */}
        {error && (
          <InlineError
            category={error.category}
            message={error.message}
            details={error.details}
            onDismiss={clearError}
          />
        )}

        {/* Product cards */}
        {loading ? (
          <div className="flex items-center gap-2 py-8">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            <span className="text-sm text-muted-foreground">{t('gyProducts.loading')}</span>
          </div>
        ) : (
          <div className="space-y-4">
            {products.map((product) => {
              const status = statuses[product.id]
              const badge = KIND_BADGES[product.kind] || KIND_BADGES.web
              const icon = PRODUCT_ICONS[product.id] || '📦'
              const isLaunching = launching === product.id

              return (
                <div key={product.id} className="border border-border rounded-xl p-5 bg-card space-y-3">
                  {/* Card header */}
                  <div className="flex items-start gap-4">
                    <div className="text-3xl">{icon}</div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <h3 className="text-sm font-semibold">{product.name}</h3>
                        <span className={cn('text-[10px] px-1.5 py-0.5 rounded font-medium', badge.color)}>
                          {t(badge.label)}
                        </span>
                        {product.kind === 'desktop' ? (
                          <InstalledBadge status={status} />
                        ) : (
                          <StatusDot status={status} />
                        )}
                      </div>
                      <p className="text-xs text-muted-foreground mt-1">{product.description}</p>
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="flex gap-2 flex-wrap">
                    {product.kind === 'web' && (
                      <button
                        onClick={() => handleLaunch(product.id)}
                        disabled={isLaunching}
                        className={cn(
                          'flex items-center gap-1.5 px-4 py-1.5 rounded-md text-xs font-medium transition-colors',
                          'bg-primary text-primary-foreground hover:bg-primary/90',
                          'disabled:opacity-50 disabled:cursor-not-allowed'
                        )}
                      >
                        {isLaunching ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <ExternalLink className="h-3.5 w-3.5" />}
                        {t('gyProducts.open', { name: product.name })}
                      </button>
                    )}

                    {product.kind === 'desktop' && (
                      <>
                        <button
                          onClick={() => handleLaunch(product.id)}
                          disabled={isLaunching}
                          className={cn(
                            'flex items-center gap-1.5 px-4 py-1.5 rounded-md text-xs font-medium transition-colors',
                            'bg-primary text-primary-foreground hover:bg-primary/90',
                            'disabled:opacity-50 disabled:cursor-not-allowed'
                          )}
                        >
                          {isLaunching ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Play className="h-3.5 w-3.5" />}
                          {t('gyProducts.launch', { name: product.name })}
                        </button>
                        <div className="flex flex-col gap-1">
                          <button
                            onClick={handleDownloadCreator}
                            disabled={downloading}
                            className={cn(
                              'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
                              'border border-border hover:bg-muted',
                              'disabled:opacity-50 disabled:cursor-not-allowed'
                            )}
                          >
                            {downloading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Download className="h-3.5 w-3.5" />}
                            {downloading ? t('gyProducts.downloading', { progress: creatorProgress >= 0 ? creatorProgress + '%' : '' }) : t('gyProducts.redownload')}
                          </button>
                          {downloading && creatorProgress >= 0 && (
                            <div className="w-full h-1 bg-muted rounded-full overflow-hidden">
                              <div
                                className="h-full bg-primary transition-all duration-300"
                                style={{ width: `${creatorProgress}%` }}
                              />
                            </div>
                          )}
                        </div>
                      </>
                    )}

                    {product.kind === 'service' && (
                      <>
                        <button
                          onClick={() => handleLaunch(product.id)}
                          disabled={isLaunching}
                          className={cn(
                            'flex items-center gap-1.5 px-4 py-1.5 rounded-md text-xs font-medium transition-colors',
                            'bg-primary text-primary-foreground hover:bg-primary/90',
                            'disabled:opacity-50 disabled:cursor-not-allowed'
                          )}
                        >
                          {isLaunching ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <ExternalLink className="h-3.5 w-3.5" />}
                          {t('gyProducts.openConsole')}
                        </button>
                      </>
                    )}
                  </div>

                  {/* Error from status */}
                  {status?.error && (
                    <p className="text-xs text-red-400/70">{status.error}</p>
                  )}
                </div>
              )
            })}

            {products.length === 0 && !loading && (
              <div className="border border-dashed border-border rounded-lg p-8 text-center">
                <p className="text-sm text-muted-foreground">{t('gyProducts.noProducts')}</p>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
