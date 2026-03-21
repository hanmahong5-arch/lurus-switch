import { useEffect, useState } from 'react'
import { Loader2, Copy, CheckCircle2, Users, DollarSign, Clock, Globe } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useClassifiedError } from '../lib/useClassifiedError'
import { InlineError } from '../components/InlineError'
import { PromoterGetInfo } from '../../wailsjs/go/main/App'
import { usePromoterStore, type PromoterInfo } from '../stores/promoterStore'

export function PromoterHubPage() {
  const { t } = useTranslation()
  const { info, loading, setInfo, setLoading } = usePromoterStore()
  const [copied, setCopied] = useState(false)
  const { classified: error, setError, clearError } = useClassifiedError()

  useEffect(() => {
    setLoading(true)
    clearError()
    PromoterGetInfo()
      .then((data: PromoterInfo) => setInfo(data))
      .catch((err: unknown) => setError(err))
      .finally(() => setLoading(false))
  }, [setInfo, setLoading])

  const handleCopyLink = () => {
    if (!info?.share_link) return
    navigator.clipboard.writeText(info.share_link)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleCopyCode = () => {
    if (!info?.aff_code) return
    navigator.clipboard.writeText(info.aff_code)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="h-full flex items-center justify-center p-6">
        <div className="max-w-md w-full space-y-3">
          <InlineError
            category={error.category}
            message={error.message}
            details={error.details}
            action={{ label: t('error.action.retry'), onClick: () => { clearError(); setLoading(true); PromoterGetInfo().then((data: PromoterInfo) => setInfo(data)).catch((err: unknown) => setError(err)).finally(() => setLoading(false)) } }}
            onDismiss={clearError}
          />
          <p className="text-xs text-muted-foreground text-center">{t('promoter.connectHint')}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-2xl mx-auto p-6 space-y-6">
        {/* Header */}
        <div>
          <h2 className="text-lg font-semibold">{t('promoter.title')}</h2>
          <p className="text-sm text-muted-foreground">{t('promoter.subtitle')}</p>
        </div>

        {/* Promo Code Card */}
        <div className="border border-border rounded-lg p-5 bg-card space-y-3">
          <h3 className="text-sm font-medium">{t('promoter.affCode')}</h3>
          <div className="flex items-center gap-3">
            <code className="flex-1 px-4 py-2.5 rounded-md bg-muted text-lg font-mono tracking-widest">
              {info?.aff_code || '—'}
            </code>
            <button
              onClick={handleCopyCode}
              className={cn(
                'flex items-center gap-1.5 px-3 py-2 rounded-md text-sm font-medium transition-colors',
                'border border-border hover:bg-muted'
              )}
            >
              {copied ? <CheckCircle2 className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
            </button>
          </div>
          <button
            onClick={handleCopyLink}
            className={cn(
              'w-full flex items-center justify-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium transition-colors',
              'bg-primary text-primary-foreground hover:bg-primary/90'
            )}
          >
            <Copy className="h-4 w-4" />
            {t('promoter.copyLink')}
          </button>
          {info?.share_link && (
            <p className="text-xs text-muted-foreground truncate">{info.share_link}</p>
          )}
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-3 gap-4">
          <StatCard
            icon={Users}
            label={t('promoter.totalReferrals')}
            value={String(info?.total_referrals ?? 0)}
            color="text-blue-500"
          />
          <StatCard
            icon={DollarSign}
            label={t('promoter.totalEarned')}
            value={`${(info?.total_earned ?? 0).toFixed(2)}`}
            color="text-green-500"
          />
          <StatCard
            icon={Clock}
            label={t('promoter.pendingEarned')}
            value={`${(info?.pending_earned ?? 0).toFixed(2)}`}
            color="text-yellow-500"
          />
        </div>

        {/* Gateway URL */}
        <div className="border border-border rounded-lg p-5 bg-card space-y-3">
          <h3 className="text-sm font-medium flex items-center gap-2">
            <Globe className="h-4 w-4 text-muted-foreground" />
            {t('promoter.gatewayUrl')}
          </h3>
          <p className="text-xs text-muted-foreground">{t('promoter.gatewayUrlDesc')}</p>
          <p className="text-xs text-muted-foreground">{t('promoter.gatewayUrlHint')}</p>
        </div>
      </div>
    </div>
  )
}

function StatCard({ icon: Icon, label, value, color }: {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: string
  color: string
}) {
  return (
    <div className="border border-border rounded-lg p-4 bg-card">
      <div className="flex items-center gap-2 mb-2">
        <Icon className={cn('h-4 w-4', color)} />
        <span className="text-xs text-muted-foreground">{label}</span>
      </div>
      <p className="text-xl font-semibold">{value}</p>
    </div>
  )
}
