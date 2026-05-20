import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { RefreshCw, ExternalLink, X } from 'lucide-react'
import { cn } from '../../lib/utils'
import { useAuthStore } from '../../stores/authStore'
import { useBillingStore } from '../../stores/billingStore'
import { useGatewayStore } from '../../stores/gatewayStore'
import { useDashboardStore } from '../../stores/dashboardStore'
import { useConfigStore } from '../../stores/configStore'
import { AccountWalletBlock } from './AccountWalletBlock'
import { AccountUsageBlock } from './AccountUsageBlock'
import { AccountSubscriptionBlock } from './AccountSubscriptionBlock'
import { AccountServiceBlock } from './AccountServiceBlock'
import { BillingGetUserInfo, BillingGetIdentityOverview, BillingOpenTopup, GetEndUserStatus, GetAppSettings } from '../../../wailsjs/go/main/App'

interface Props {
  open: boolean
  onClose: () => void
  /** Anchor element (the summary card). Popover positions to its right. */
  anchorRef: React.RefObject<HTMLElement>
}

// Estimate this month's cost by extrapolating today's daily_used over the
// days elapsed so far this month. Uses Date.getDate() (local time) rather
// than a flat ×30 — gives a better signal early in the month.
// Newapi unit: quota / 500000 ≈ $1 ≈ ¥7.2.
function estimateMonthCost(dailyUsed?: number): number | undefined {
  if (dailyUsed == null) return undefined
  const today = new Date().getDate()
  const dollars = (dailyUsed / 500000) * today
  return dollars * 7.2
}

export function AccountDetailPopover({ open, onClose, anchorRef }: Props) {
  const { t } = useTranslation()
  const { authState, refresh: refreshAuth } = useAuthStore()
  const { userInfo, identityOverview, setUserInfo, setIdentityOverview } = useBillingStore()
  const gatewayStatus = useGatewayStore((s) => s.status)
  const proxySettings = useDashboardStore((s) => s.proxySettings)
  const { appMode, setActiveTool, setSubTab } = useConfigStore()

  const [refreshing, setRefreshing] = useState(false)
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null)
  const popoverRef = useRef<HTMLDivElement>(null)

  // EndUser activation snapshot — pulled on open (small payload, no store).
  const [endUserStatus, setEndUserStatus] = useState<{
    hubUrl?: string
    tenantSlug?: string
    quota?: number
    expiresAt?: string
    lastHeartbeat?: string
    state?: string
  } | null>(null)
  // Reseller config snapshot.
  const [reseller, setReseller] = useState<{
    hubUrl?: string
    tenantSlug?: string
    displayName?: string
  } | null>(null)

  // Position relative to anchor + viewport.
  useEffect(() => {
    if (!open || !anchorRef.current) return
    const rect = anchorRef.current.getBoundingClientRect()
    const POP_WIDTH = 380
    const POP_HEIGHT = 540
    const MARGIN = 8
    let left = rect.right + MARGIN
    // If overflows right, flip to left of anchor.
    if (left + POP_WIDTH > window.innerWidth) {
      left = Math.max(MARGIN, rect.left - POP_WIDTH - MARGIN)
    }
    let top = rect.top
    if (top + POP_HEIGHT > window.innerHeight) {
      top = Math.max(MARGIN, window.innerHeight - POP_HEIGHT - MARGIN)
    }
    setPos({ top, left })
  }, [open, anchorRef])

  // Click-outside + Esc.
  useEffect(() => {
    if (!open) return
    const onClick = (e: MouseEvent) => {
      const target = e.target as Node
      if (popoverRef.current?.contains(target)) return
      if (anchorRef.current?.contains(target)) return
      onClose()
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('mousedown', onClick)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onClick)
      document.removeEventListener('keydown', onKey)
    }
  }, [open, onClose, anchorRef])

  // On open: pull mode-specific snapshots.
  useEffect(() => {
    if (!open) return
    if (appMode === 'enduser') {
      GetEndUserStatus()
        .then((s: any) => setEndUserStatus({
          hubUrl: s?.hubUrl,
          tenantSlug: s?.tenantSlug,
          quota: s?.quota,
          expiresAt: s?.expiresAt,
          lastHeartbeat: s?.lastHeartbeat,
          state: s?.state,
        }))
        .catch(() => setEndUserStatus(null))
    }
    if (appMode === 'reseller') {
      GetAppSettings()
        .then((s: any) => setReseller({
          hubUrl: s?.reseller?.hubUrl,
          tenantSlug: s?.reseller?.tenantSlug,
          displayName: s?.reseller?.displayName,
        }))
        .catch(() => setReseller(null))
    }
  }, [open, appMode])

  const handleRefresh = async () => {
    setRefreshing(true)
    try {
      // Refresh in parallel — auth, user info, identity overview.
      await Promise.allSettled([
        refreshAuth(),
        (async () => {
          try {
            const info = await BillingGetUserInfo()
            if (info) setUserInfo(info)
          } catch { /* swallow */ }
        })(),
        (async () => {
          try {
            const ov = await BillingGetIdentityOverview('lurus-switch')
            if (ov) setIdentityOverview(ov)
          } catch { /* swallow */ }
        })(),
      ])
    } finally {
      setRefreshing(false)
    }
  }

  const handleTopUp = async () => {
    const target = (identityOverview as any)?.topup_url
    if (target) {
      try { await BillingOpenTopup(target) } catch { /* ignore */ }
      return
    }
    setActiveTool('account')
    setSubTab('account', 'billing')
    onClose()
  }

  const handleOpenAccountPage = () => {
    setActiveTool('account')
    onClose()
  }

  if (!open || !pos) return null

  const user = authState.user
  const wallet = (identityOverview as any)?.wallet
  const account = (identityOverview as any)?.account
  const vip = (identityOverview as any)?.vip
  const subscription = (identityOverview as any)?.subscription
  const isEnduser = appMode === 'enduser'
  const isReseller = appMode === 'reseller'

  return (
    <div
      ref={popoverRef}
      role="dialog"
      aria-label={t('account.detail.title', 'Account · 详细信息')}
      className={cn(
        'fixed z-50 bg-card border border-border rounded-md shadow-2xl',
        'w-[380px] max-h-[540px] overflow-y-auto',
        'animate-in fade-in slide-in-from-left-2 duration-150',
      )}
      style={{ top: pos.top, left: pos.left }}
    >
      {/* Header */}
      <div className="sticky top-0 bg-card border-b border-border px-4 py-3 flex items-start justify-between gap-2">
        <div className="flex items-center gap-3 min-w-0 flex-1">
          {user?.picture && (
            <img
              src={user.picture}
              alt={user.name || ''}
              className="h-9 w-9 rounded-full border border-border flex-shrink-0"
              referrerPolicy="no-referrer"
            />
          )}
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium truncate">
              {account?.display_name || user?.name || t('account.detail.guest', '未登录')}
            </p>
            {user?.email && (
              <p className="text-[10px] text-muted-foreground truncate">{user.email}</p>
            )}
            {account?.lurus_id && (
              <p className="text-[10px] text-muted-foreground/70 font-mono truncate">
                LurusID: {account.lurus_id}
              </p>
            )}
          </div>
        </div>
        <button
          onClick={onClose}
          className="p-1 rounded-sm hover:bg-muted text-muted-foreground"
          aria-label={t('common.close', '关闭')}
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Body — mode-specific blocks */}
      <div className="p-4 space-y-4 divide-y divide-border/40 [&>*]:pt-3 [&>*:first-child]:pt-0">
        {/* Personal: full set */}
        {!isReseller && !isEnduser && (
          <>
            <AccountWalletBlock
              balance={wallet?.balance}
              frozen={wallet?.frozen}
              onTopUp={handleTopUp}
              onHistory={handleOpenAccountPage}
            />
            <AccountUsageBlock
              quota={userInfo?.quota}
              usedQuota={userInfo?.used_quota}
              dailyUsed={userInfo?.daily_used}
              estimatedMonthCost={estimateMonthCost(userInfo?.daily_used)}
              onByModel={handleOpenAccountPage}
              onByChannel={handleOpenAccountPage}
            />
            <AccountSubscriptionBlock
              planName={vip?.level_name}
              planCode={subscription?.plan_code}
              status={subscription?.status || (vip?.level > 0 ? 'active' : 'free')}
              expiresAt={subscription?.expires_at}
              autoRenew={subscription?.auto_renew}
              onRenew={handleOpenAccountPage}
              onUpgrade={handleOpenAccountPage}
            />
            <AccountServiceBlock
              gatewayRunning={gatewayStatus?.running}
              gatewayPort={gatewayStatus?.port}
              gatewayUrl={gatewayStatus?.url}
            />
          </>
        )}

        {/* Reseller: no personal wallet — show tenant + hub + gateway */}
        {isReseller && (
          <>
            <AccountServiceBlock
              gatewayRunning={gatewayStatus?.running}
              gatewayPort={gatewayStatus?.port}
              hubUrl={reseller?.hubUrl || proxySettings?.apiEndpoint}
              tenantSlug={reseller?.tenantSlug || proxySettings?.tenantSlug}
            />
            {reseller?.displayName && (
              <div className="text-xs">
                <p className="text-muted-foreground/70">{t('account.detail.reseller.name', '经销商名称')}</p>
                <p className="font-medium">{reseller.displayName}</p>
              </div>
            )}
            <div className="text-[11px] text-muted-foreground/70">
              {t('account.detail.reseller.hint', '经销商财务信息请在专属控制台查看')}
            </div>
          </>
        )}

        {/* EndUser: hub + quota + expiry + heartbeat */}
        {isEnduser && (
          <>
            <AccountServiceBlock
              gatewayRunning={gatewayStatus?.running}
              gatewayPort={gatewayStatus?.port}
              hubUrl={endUserStatus?.hubUrl}
            />
            {endUserStatus?.quota != null && (
              <AccountUsageBlock
                quota={endUserStatus.quota}
                usedQuota={userInfo?.used_quota}
                dailyUsed={userInfo?.daily_used}
              />
            )}
            {endUserStatus?.expiresAt && (
              <div className="text-xs">
                <p className="text-muted-foreground/70">{t('account.detail.enduser.expires', '激活到期')}</p>
                <p className="font-medium tabular-nums">
                  {new Date(endUserStatus.expiresAt).toISOString().slice(0, 10)}
                </p>
              </div>
            )}
            {endUserStatus?.lastHeartbeat && (
              <div className="text-xs">
                <p className="text-muted-foreground/70">{t('account.detail.enduser.heartbeat', '最后心跳')}</p>
                <p className="font-mono text-[10px]">
                  {new Date(endUserStatus.lastHeartbeat).toLocaleString()}
                </p>
              </div>
            )}
          </>
        )}
      </div>

      {/* Footer actions */}
      <div className="sticky bottom-0 bg-card border-t border-border px-4 py-2 flex items-center justify-between">
        <button
          onClick={handleRefresh}
          disabled={refreshing}
          className="text-xs px-2 py-1 rounded-sm hover:bg-muted text-muted-foreground inline-flex items-center gap-1.5 disabled:opacity-50"
        >
          <RefreshCw className={cn('h-3 w-3', refreshing && 'animate-spin')} />
          {t('account.detail.refresh', '刷新')}
        </button>
        <button
          onClick={handleOpenAccountPage}
          className="text-xs px-2 py-1 rounded-sm bg-primary/10 text-primary hover:bg-primary/15 inline-flex items-center gap-1"
        >
          {t('account.detail.openFull', '完整账户页')}
          <ExternalLink className="h-3 w-3" />
        </button>
      </div>
    </div>
  )
}
