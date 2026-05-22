import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, RefreshCw, ExternalLink, Download, Play } from 'lucide-react'
import { cn } from '../lib/utils'
import { useClassifiedError } from '../lib/useClassifiedError'
import { InlineError } from '../components/InlineError'
import { Button, Card } from '../components/ui'
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
  web: { label: 'gyProducts.categories.web', color: 'bg-blue-500/15 text-blue-400' },
  desktop: { label: 'gyProducts.categories.desktop', color: 'bg-violet-500/15 text-violet-400' },
  service: { label: 'gyProducts.categories.service', color: 'bg-teal-500/15 text-teal-400' },
}

function StatusDot({ status }: { status: gy.GYStatus | undefined }) {
  const { t } = useTranslation()
  if (!status) return <span className="text-xs text-muted-foreground/60 font-mono">▾ {t('gyProducts.checking')}</span>
  if (status.available) {
    return (
      <span className="flex items-center gap-1 text-xs text-emerald-400 font-mono">
        <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse inline-block" />
        ▸ {t('gyProducts.online')}
        {status.latencyMs > 0 && <span className="text-muted-foreground tabular-nums">({status.latencyMs}ms)</span>}
      </span>
    )
  }
  return (
    <span className="flex items-center gap-1 text-xs text-red-400 font-mono">
      <span className="h-1.5 w-1.5 rounded-full bg-red-400 inline-block" />
      ▪ {t('gyProducts.unreachable')}
    </span>
  )
}

function InstalledBadge({ status }: { status: gy.GYStatus | undefined }) {
  const { t } = useTranslation()
  if (!status) return null
  if (status.version) {
    return <span className="text-xs text-emerald-400 font-mono tabular-nums">▸ {t('gyProducts.installedVersion', { version: status.version })}</span>
  }
  return <span className="text-xs text-muted-foreground font-mono">▪ {t('gyProducts.notInstalled')}</span>
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
          <Button
            variant="secondary"
            size="sm"
            onClick={checkStatus}
            disabled={checking}
            loading={checking}
            icon={!checking ? <RefreshCw className="h-3.5 w-3.5" /> : undefined}
          >
            {t('gyProducts.refreshStatus')}
          </Button>
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
                <Card key={product.id} variant="elevated" className="p-5 space-y-3">
                  {/* Card header */}
                  <div className="flex items-start gap-4">
                    <div className="text-3xl">{icon}</div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <h3 className="text-sm font-semibold">{product.name}</h3>
                        <span className={cn('font-mono text-[10px] uppercase tracking-[0.12em] px-1.5 py-0.5 rounded', badge.color)}>
                          [ {t(badge.label).toUpperCase()} ]
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
                      <Button
                        size="sm"
                        onClick={() => handleLaunch(product.id)}
                        disabled={isLaunching}
                        loading={isLaunching}
                        icon={!isLaunching ? <ExternalLink className="h-3.5 w-3.5" /> : undefined}
                      >
                        {t('gyProducts.open', { name: product.name })}
                      </Button>
                    )}

                    {product.kind === 'desktop' && (
                      <>
                        <Button
                          size="sm"
                          onClick={() => handleLaunch(product.id)}
                          disabled={isLaunching}
                          loading={isLaunching}
                          icon={!isLaunching ? <Play className="h-3.5 w-3.5" /> : undefined}
                        >
                          {t('gyProducts.launch', { name: product.name })}
                        </Button>
                        <div className="flex flex-col gap-1">
                          <Button
                            variant="secondary"
                            size="sm"
                            onClick={handleDownloadCreator}
                            disabled={downloading}
                            loading={downloading}
                            icon={!downloading ? <Download className="h-3.5 w-3.5" /> : undefined}
                          >
                            {downloading ? t('gyProducts.downloading', { progress: creatorProgress >= 0 ? creatorProgress + '%' : '' }) : t('gyProducts.redownload')}
                          </Button>
                          {downloading && creatorProgress >= 0 && (
                            <div className="w-full h-1 bg-card-recessed rounded-full overflow-hidden">
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
                      <Button
                        size="sm"
                        onClick={() => handleLaunch(product.id)}
                        disabled={isLaunching}
                        loading={isLaunching}
                        icon={!isLaunching ? <ExternalLink className="h-3.5 w-3.5" /> : undefined}
                      >
                        {t('gyProducts.openConsole')}
                      </Button>
                    )}
                  </div>

                  {/* Error from status */}
                  {status?.error && (
                    <p className="text-xs text-red-400/70 font-mono">▸ {status.error}</p>
                  )}
                </Card>
              )
            })}

            {products.length === 0 && !loading && (
              <Card variant="default" className="border-dashed p-8 text-center">
                <p className="text-sm text-muted-foreground">{t('gyProducts.noProducts')}</p>
              </Card>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
