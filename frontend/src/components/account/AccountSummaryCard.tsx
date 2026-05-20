import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { User, ChevronRight, LogIn } from 'lucide-react'
import { cn } from '../../lib/utils'
import { useAuthStore } from '../../stores/authStore'
import { useBillingStore } from '../../stores/billingStore'
import { useConfigStore } from '../../stores/configStore'
import { GetEndUserStatus, GetAppSettings } from '../../../wailsjs/go/main/App'
import { AccountDetailPopover } from './AccountDetailPopover'

export function AccountSummaryCard() {
  const { t } = useTranslation()
  const { authState, login } = useAuthStore()
  const { userInfo, identityOverview } = useBillingStore()
  const appMode = useConfigStore((s) => s.appMode)
  const [open, setOpen] = useState(false)
  const cardRef = useRef<HTMLButtonElement>(null)

  // EndUser activation (read once on mount + when mode is enduser).
  const [endUserHubName, setEndUserHubName] = useState<string>('')
  useEffect(() => {
    if (appMode !== 'enduser') return
    GetAppSettings()
      .then((s: any) => setEndUserHubName(s?.brandName || ''))
      .catch(() => { /* ignore */ })
    GetEndUserStatus().catch(() => { /* ignore */ })
  }, [appMode])

  // Not logged in: show login CTA card.
  if (!authState.is_logged_in && appMode !== 'reseller' && appMode !== 'enduser') {
    return (
      <button
        onClick={() => {
          void login()
        }}
        className={cn(
          'w-full mx-2 my-2 px-3 py-2 rounded-md border border-dashed border-border',
          'hover:border-primary/40 hover:bg-primary/5 transition-colors',
          'flex items-center gap-2 text-left',
        )}
      >
        <LogIn className="h-4 w-4 text-muted-foreground flex-shrink-0" />
        <div className="min-w-0 flex-1">
          <p className="text-xs font-medium">{t('account.summary.signIn', '登录账户')}</p>
          <p className="text-[10px] text-muted-foreground">{t('account.summary.signInHint', '解锁余额 / 订阅 / 用量')}</p>
        </div>
      </button>
    )
  }

  // Derive display fields.
  const user = authState.user
  const wallet = (identityOverview as any)?.wallet
  const vip = (identityOverview as any)?.vip
  const account = (identityOverview as any)?.account
  const balance = wallet?.balance
  const planLabel = vip?.level_name || vip?.level_en

  const pct = (userInfo?.quota != null && userInfo.quota > 0 && userInfo.used_quota != null)
    ? Math.min(100, (userInfo.used_quota / userInfo.quota) * 100)
    : null
  const warn = pct != null && pct >= 80
  const exhausted = pct != null && pct >= 100

  // Mode-specific identity line.
  const isReseller = appMode === 'reseller'
  const isEnduser = appMode === 'enduser'
  const displayName = isEnduser
    ? (endUserHubName || t('account.summary.enduserClient', '激活客户端'))
    : (account?.display_name || user?.name || t('account.summary.guest', '未登录'))

  // Avatar: OIDC picture for Personal/Reseller, generic icon for EndUser.
  const showAvatar = !isEnduser && user?.picture

  return (
    <>
      <button
        ref={cardRef}
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="dialog"
        aria-expanded={open}
        className={cn(
          'w-full mx-2 my-2 px-3 py-2 rounded-md border border-border',
          'hover:bg-muted/40 hover:border-border/80 transition-colors',
          'flex flex-col gap-1.5 text-left',
          open && 'bg-muted/40 border-primary/40',
        )}
      >
        {/* Row 1: avatar + name + chevron */}
        <div className="flex items-center gap-2 min-w-0 w-full">
          {showAvatar ? (
            <img
              src={user.picture}
              alt=""
              className="h-6 w-6 rounded-full border border-border flex-shrink-0"
              referrerPolicy="no-referrer"
            />
          ) : (
            <div className="h-6 w-6 rounded-full bg-primary/15 flex items-center justify-center flex-shrink-0">
              <User className="h-3.5 w-3.5 text-primary" />
            </div>
          )}
          <span className="text-xs font-medium truncate flex-1">{displayName}</span>
          <ChevronRight className={cn('h-3 w-3 text-muted-foreground transition-transform', open && 'rotate-90')} />
        </div>

        {/* Row 2: balance + plan (Personal only) OR mode label */}
        {!isReseller && !isEnduser && (
          <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground">
            {balance != null && (
              <span className="font-mono tabular-nums">¥{balance.toFixed(2)}</span>
            )}
            {balance != null && planLabel && <span className="text-muted-foreground/40">·</span>}
            {planLabel && <span>{planLabel}</span>}
            {balance == null && !planLabel && (
              <span className="text-muted-foreground/60">{t('account.summary.noBilling', '账户已连接')}</span>
            )}
          </div>
        )}
        {isReseller && (
          <div className="text-[10px] text-muted-foreground">
            {t('account.summary.resellerHint', '经销商控制台')}
          </div>
        )}
        {isEnduser && (
          <div className="text-[10px] text-muted-foreground">
            {t('account.summary.enduserHint', '激活码客户端')}
          </div>
        )}

        {/* Row 3: usage bar (when available) */}
        {pct != null && (
          <div className="space-y-0.5 w-full">
            <div className="flex justify-between text-[10px]">
              <span className="text-muted-foreground/70">{t('account.summary.quota', '配额')}</span>
              <span className={cn(
                'tabular-nums font-mono',
                exhausted ? 'text-red-500' : warn ? 'text-amber-500' : 'text-muted-foreground',
              )}>{pct.toFixed(0)}%</span>
            </div>
            <div className="h-1 bg-muted rounded-full overflow-hidden">
              <div
                className={cn(
                  'h-full transition-all',
                  exhausted ? 'bg-red-500' : warn ? 'bg-amber-500' : 'bg-primary',
                )}
                style={{ width: `${pct}%` }}
              />
            </div>
          </div>
        )}
      </button>

      <AccountDetailPopover
        open={open}
        onClose={() => setOpen(false)}
        anchorRef={cardRef}
      />
    </>
  )
}
