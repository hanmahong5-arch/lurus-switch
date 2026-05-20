import { useTranslation } from 'react-i18next'
import { Radio } from 'lucide-react'
import { cn } from '../../lib/utils'

interface Props {
  gatewayRunning?: boolean
  gatewayPort?: number
  gatewayUrl?: string
  toolsConnected?: number
  toolsTotal?: number
  /** Optional Hub URL for EndUser mode display. */
  hubUrl?: string
  /** Optional tenant slug for Reseller mode display. */
  tenantSlug?: string
  loading?: boolean
  compact?: boolean
}

export function AccountServiceBlock({
  gatewayRunning,
  gatewayPort,
  toolsConnected,
  toolsTotal,
  hubUrl,
  tenantSlug,
  loading = false,
}: Props) {
  const { t } = useTranslation()

  if (loading) {
    return (
      <div className="space-y-2">
        <div className="h-3 w-16 bg-muted rounded animate-pulse" />
        <div className="h-4 w-32 bg-muted rounded animate-pulse" />
      </div>
    )
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5">
        <Radio className="h-3.5 w-3.5 text-cyan-500" />
        <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground/70">
          {t('account.detail.service.title', '服务')}
        </span>
      </div>
      <div className="space-y-1 text-xs">
        {gatewayPort != null && (
          <div className="flex justify-between items-center">
            <span className="text-muted-foreground/70">{t('account.detail.service.gateway', 'Gateway')}</span>
            <span className="font-mono tabular-nums flex items-center gap-1.5">
              :{gatewayPort}
              <span className={cn(
                'h-1.5 w-1.5 rounded-full',
                gatewayRunning ? 'bg-emerald-400 animate-pulse' : 'bg-muted-foreground/30',
              )} />
            </span>
          </div>
        )}
        {toolsTotal != null && toolsTotal > 0 && (
          <div className="flex justify-between items-center">
            <span className="text-muted-foreground/70">{t('account.detail.service.tools', '工具')}</span>
            <span className="font-mono tabular-nums">
              {toolsConnected ?? 0}/{toolsTotal} {t('account.detail.service.connected', 'connected')}
            </span>
          </div>
        )}
        {hubUrl && (
          <div className="flex justify-between items-center">
            <span className="text-muted-foreground/70">Hub</span>
            <span className="font-mono text-[10px] truncate max-w-[200px]" title={hubUrl}>{hubUrl}</span>
          </div>
        )}
        {tenantSlug && (
          <div className="flex justify-between items-center">
            <span className="text-muted-foreground/70">{t('account.detail.service.tenant', 'Tenant')}</span>
            <span className="font-mono">{tenantSlug}</span>
          </div>
        )}
      </div>
    </div>
  )
}
