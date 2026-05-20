import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { RefreshCw } from 'lucide-react'
import { cn } from '../../lib/utils'
import { useAuthStore } from '../../stores/authStore'
import { useBillingStore } from '../../stores/billingStore'
import { useConfigStore } from '../../stores/configStore'
import { AccountWalletBlock } from './AccountWalletBlock'
import { AccountUsageBlock } from './AccountUsageBlock'
import { AccountSubscriptionBlock } from './AccountSubscriptionBlock'
import { BillingGetUserInfo, BillingGetIdentityOverview, BillingOpenTopup } from '../../../wailsjs/go/main/App'

// Frontend estimation for this month's cost. Pulled out to a named export so
// tests can assert the month-start and month-end edge values. Uses local
// time getDate() — newapi quota / 500000 ≈ $1 ≈ ¥7.2.
export function estimateMonthCost(dailyUsed?: number, now: Date = new Date()): number | undefined {
  if (dailyUsed == null) return undefined
  const days = now.getDate()
  const dollars = (dailyUsed / 500000) * days
  return dollars * 7.2
}

export function HomeAccountHero() {
  const { t } = useTranslation()
  const { authState } = useAuthStore()
  const {
    userInfo,
    identityOverview,
    lastRefreshedAt,
    setUserInfo,
    setIdentityOverview,
    setLastRefreshedAt,
    startPolling,
    stopPolling,
  } = useBillingStore()
  const { appMode, setActiveTool, setSubTab } = useConfigStore()

  const isReseller = appMode === 'reseller'
  const isEnduser = appMode === 'enduser'

  // Hero is hidden for EndUser (their own dashboard handles it) and shown
  // for Personal + Reseller. For Reseller we drop the wallet/subscription
  // blocks (they live in the dedicated console).
  useEffect(() => {
    if (isEnduser) return
    // Trigger initial refresh + start polling on mount.
    void refresh()
    startPolling()
    return () => stopPolling()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isEnduser])

  const refresh = async () => {
    try {
      const info = await BillingGetUserInfo()
      if (info) setUserInfo(info)
    } catch { /* swallow */ }
    try {
      const ov = await BillingGetIdentityOverview('lurus-switch')
      if (ov) setIdentityOverview(ov)
    } catch { /* swallow */ }
    setLastRefreshedAt(new Date().toISOString())
  }

  if (isEnduser) return null
  if (!authState.is_logged_in && !isReseller) return null

  const wallet = (identityOverview as any)?.wallet
  const vip = (identityOverview as any)?.vip
  const account = (identityOverview as any)?.account
  const subscription = (identityOverview as any)?.subscription
  const userName = account?.display_name || authState.user?.name || ''

  const handleTopUp = async () => {
    const target = (identityOverview as any)?.topup_url
    if (target) {
      try { await BillingOpenTopup(target) } catch { /* ignore */ }
      return
    }
    setActiveTool('account')
    setSubTab('account', 'billing')
  }
  const goAccountBilling = () => {
    setActiveTool('account')
    setSubTab('account', 'billing')
  }

  const lastRefreshLabel = lastRefreshedAt
    ? new Date(lastRefreshedAt).toLocaleTimeString()
    : '—'

  return (
    <section
      aria-label={t('home.hero.label', '账户摘要')}
      className="bg-card border border-border rounded-lg p-5"
    >
      <div className="flex items-baseline justify-between mb-4">
        <h2 className="text-base font-semibold">
          {userName
            ? t('home.hero.welcome', '欢迎回来, {{name}}', { name: userName })
            : t('home.hero.welcomeGuest', '欢迎')}
        </h2>
        <div className="flex items-center gap-2 text-[10px] text-muted-foreground">
          <span>{t('home.hero.lastUpdated', '最后更新: {{time}}', { time: lastRefreshLabel })}</span>
          <button
            onClick={refresh}
            className="p-1 rounded-sm hover:bg-muted text-muted-foreground inline-flex items-center gap-1"
            aria-label={t('account.detail.refresh', '刷新')}
          >
            <RefreshCw className="h-3 w-3" />
          </button>
        </div>
      </div>
      <div className={cn(
        'grid gap-4',
        isReseller ? 'grid-cols-1' : 'sm:grid-cols-2 lg:grid-cols-3',
      )}>
        {!isReseller && (
          <AccountWalletBlock
            balance={wallet?.balance}
            frozen={wallet?.frozen}
            onTopUp={handleTopUp}
            onHistory={goAccountBilling}
          />
        )}
        <AccountUsageBlock
          quota={userInfo?.quota}
          usedQuota={userInfo?.used_quota}
          dailyUsed={userInfo?.daily_used}
          estimatedMonthCost={estimateMonthCost(userInfo?.daily_used)}
          onByModel={goAccountBilling}
          onByChannel={goAccountBilling}
        />
        {!isReseller && (
          <AccountSubscriptionBlock
            planName={vip?.level_name}
            planCode={subscription?.plan_code}
            status={subscription?.status || (vip?.level > 0 ? 'active' : 'free')}
            expiresAt={subscription?.expires_at}
            autoRenew={subscription?.auto_renew}
            onRenew={goAccountBilling}
            onUpgrade={goAccountBilling}
          />
        )}
      </div>
    </section>
  )
}
