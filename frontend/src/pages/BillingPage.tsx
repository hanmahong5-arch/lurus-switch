import { useEffect, useState, useCallback } from 'react'
import { RefreshCw, Copy, Check } from 'lucide-react'
import { errorToast } from '../lib/errorToast'
import { Button, Card } from '../components/ui'
import { withRetry } from '../lib/withRetry'
import { useBillingStore } from '../stores/billingStore'
import { useToastStore } from '../stores/toastStore'
import { useDashboardStore, type ProxySettings } from '../stores/dashboardStore'
import { AccountPanel } from '../components/AccountPanel'
import { QuotaCard } from '../components/billing/QuotaCard'
import { SubscriptionCard } from '../components/billing/SubscriptionCard'
import { TopUpPanel } from '../components/billing/TopUpPanel'
import { RedeemPanel } from '../components/billing/RedeemPanel'
import { PlanSelector } from '../components/billing/PlanSelector'
import { UsageBreakdown } from '../components/billing/UsageBreakdown'
import {
  BillingGetUserInfo,
  BillingGetPlans,
  BillingGetTopUpInfo,
  BillingGetSubscriptions,
  BillingCreateTopUp,
  BillingSubscribe,
  BillingRedeemCode,
  BillingOpenPaymentURL,
  GetProxySettings,
  SaveProxySettings,
} from '../../wailsjs/go/main/App'
import type { billing } from '../../wailsjs/go/models'
import { proxy } from '../../wailsjs/go/models'

export function BillingPage() {
  const {
    userInfo, plans, subscriptions, topUpInfo,
    loading,
    setUserInfo, setPlans, setSubscriptions, setTopUpInfo,
    setLoading, reset: resetBilling,
  } = useBillingStore()

  const { proxySettings, setProxySettings } = useDashboardStore()
  const toast = useToastStore((s) => s.addToast)

  const [tokenInput, setTokenInput] = useState('')
  const [connecting, setConnecting] = useState(false)
  const [showPlans, setShowPlans] = useState(false)
  const [copiedAff, setCopiedAff] = useState(false)
  const [paymentPending, setPaymentPending] = useState(false)

  const isConnected = !!proxySettings.userToken

  // Load proxy settings on mount
  useEffect(() => {
    GetProxySettings()
      .then((s: proxy.ProxySettings) => {
        const settings: ProxySettings = {
          apiEndpoint: s.apiEndpoint,
          apiKey: s.apiKey,
          registrationUrl: s.registrationUrl,
          tenantSlug: s.tenantSlug,
          userToken: s.userToken,
        }
        setProxySettings(settings)
        if (s.userToken) {
          loadBillingData()
        }
      })
      .catch((err: unknown) => {
        errorToast(toast, err, { currentPage: 'account' })
      })
  }, [])

  const loadBillingData = useCallback(async () => {
    setLoading(true)
    try {
      const [info, planList, topUp, subs] = await Promise.all([
        withRetry(() => BillingGetUserInfo()),
        BillingGetPlans().catch(() => [] as billing.SubscriptionPlan[]),
        BillingGetTopUpInfo().catch(() => null),
        BillingGetSubscriptions().catch(() => [] as billing.SubscriptionInfo[]),
      ])
      setUserInfo(info)
      setPlans(planList || [])
      setTopUpInfo(topUp || null)
      setSubscriptions(subs || [])
    } catch (err) {
      errorToast(toast, err, { currentPage: 'account', retry: () => loadBillingData() })
    } finally {
      setLoading(false)
    }
  }, [])

  const handleConnect = async () => {
    if (!tokenInput.trim()) return
    setConnecting(true)
    try {
      const updated: ProxySettings = { ...proxySettings, userToken: tokenInput.trim() }
      await SaveProxySettings(proxy.ProxySettings.createFrom(updated))
      setProxySettings(updated)
      // Reset stale data from previous account before loading new account
      resetBilling()
      await loadBillingData()
      toast('success', 'Connected successfully')
    } catch (err) {
      errorToast(toast, err, { currentPage: 'account' })
    } finally {
      setConnecting(false)
    }
  }

  const handleTopUp = async (amount: number, method: string) => {
    setPaymentPending(true)
    try {
      const result = await BillingCreateTopUp(amount, method)
      if (result?.payment_url) {
        await BillingOpenPaymentURL(result.payment_url)
        toast('info', 'Payment page opened in browser')
      } else {
        toast('warning', 'Top-up created but no payment URL received')
      }
    } catch (err) {
      errorToast(toast, err, { currentPage: 'account', retry: () => handleTopUp(amount, method) })
    } finally {
      setPaymentPending(false)
    }
  }

  const handleSubscribe = async (planCode: string, method: string) => {
    setPaymentPending(true)
    try {
      const result = await BillingSubscribe(planCode, method)
      if (result?.payment_url) {
        await BillingOpenPaymentURL(result.payment_url)
        toast('info', 'Payment page opened in browser')
      } else {
        toast('warning', 'Subscription created but no payment URL received')
      }
    } catch (err) {
      errorToast(toast, err, { currentPage: 'account', retry: () => handleSubscribe(planCode, method) })
    } finally {
      setPaymentPending(false)
    }
  }

  const handleRedeem = async (code: string): Promise<number> => {
    const amount = await BillingRedeemCode(code)
    // Refresh user info after redeem; swallow errors so a network blip on the
    // follow-up fetch doesn't make a successful redeem appear as a failure.
    // loadBillingData already shows its own error toast, so nothing is lost.
    await loadBillingData().catch(() => {})
    return amount
  }

  const handleCopyAff = () => {
    if (userInfo?.aff_code) {
      navigator.clipboard.writeText(userInfo.aff_code)
      setCopiedAff(true)
      setTimeout(() => setCopiedAff(false), 2000)
    }
  }

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-3xl mx-auto p-6 space-y-6">
        {/* Identity overview: VIP badge + Lubell balance */}
        <AccountPanel />

        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold">Billing</h2>
            <p className="text-sm text-muted-foreground">
              Manage your quota, subscription, and payments
            </p>
          </div>
          {isConnected && (
            <Button
              variant="secondary"
              size="sm"
              onClick={loadBillingData}
              disabled={loading}
              loading={loading}
              icon={!loading ? <RefreshCw className="h-3.5 w-3.5" /> : undefined}
            >
              Refresh
            </Button>
          )}
        </div>

        {/* Token configuration */}
        {!isConnected ? (
          <Card variant="elevated" className="p-6 text-center">
            <h3 className="text-sm font-medium mb-2">Connect to Billing</h3>
            <p className="text-xs text-muted-foreground mb-4">
              Paste your user token from the web portal to access billing features
            </p>
            <div className="flex gap-2 max-w-md mx-auto">
              <input
                type="password"
                value={tokenInput}
                onChange={(e) => setTokenInput(e.target.value)}
                placeholder="Paste your token here"
                className="flex-1 px-3 py-2 rounded-md text-sm border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary font-mono"
                onKeyDown={(e) => e.key === 'Enter' && handleConnect()}
              />
              <Button
                onClick={handleConnect}
                disabled={!tokenInput.trim() || connecting}
                loading={connecting}
              >
                Connect
              </Button>
            </div>
          </Card>
        ) : (
          <>
            {/* Connected user info */}
            {userInfo && (
              <div className="flex items-center gap-3 text-sm">
                <span className="text-muted-foreground">Connected as</span>
                <span className="font-medium">{userInfo.display_name || userInfo.username}</span>
                {userInfo.group && (
                  <span className="px-2 py-0.5 rounded text-xs bg-primary/10 text-primary font-medium">
                    {userInfo.group}
                  </span>
                )}
              </div>
            )}

            {/* Loading state — skeleton blocks */}
            {loading && !userInfo ? (
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="h-24 rounded-lg border border-border bg-muted/30 animate-pulse" />
                  <div className="h-24 rounded-lg border border-border bg-muted/30 animate-pulse" />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="h-28 rounded-lg border border-border bg-muted/30 animate-pulse" />
                  <div className="h-28 rounded-lg border border-border bg-muted/30 animate-pulse" />
                </div>
                <div className="h-32 rounded-lg border border-border bg-muted/30 animate-pulse" />
              </div>
            ) : userInfo ? (
              <>
                {/* Quota overview */}
                <div className="grid grid-cols-2 gap-4">
                  <QuotaCard
                    label="Total Quota"
                    used={userInfo.used_quota}
                    total={userInfo.quota}
                  />
                  <QuotaCard
                    label="Daily Quota"
                    used={userInfo.daily_used}
                    total={userInfo.daily_quota}
                  />
                </div>

                {/* Usage breakdown — by model + by tool (last 30 days) */}
                <UsageBreakdown />

                {/* Subscription */}
                <div className="grid grid-cols-2 gap-4">
                  <SubscriptionCard
                    subscription={userInfo.subscription || subscriptions[0]}
                    onManage={() => setShowPlans(!showPlans)}
                  />

                  {/* Redeem */}
                  <RedeemPanel onRedeem={handleRedeem} />
                </div>

                {/* Plan selector */}
                {showPlans && (
                  <PlanSelector
                    plans={plans}
                    payMethods={topUpInfo?.pay_methods || []}
                    currentPlanCode={userInfo.subscription?.plan_code || subscriptions[0]?.plan_code}
                    onSubscribe={handleSubscribe}
                    loading={paymentPending}
                  />
                )}

                {/* Top Up */}
                <TopUpPanel
                  topUpInfo={topUpInfo}
                  onTopUp={handleTopUp}
                  loading={loading || paymentPending}
                />

                {/* Affiliate code */}
                {userInfo.aff_code && (
                  <Card variant="default" className="p-4">
                    <h3 className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground mb-2">
                      [ AFFILIATE CODE ]
                    </h3>
                    <div className="flex items-center gap-2">
                      <code className="flex-1 px-3 py-1.5 rounded bg-card-recessed text-xs font-mono tabular-nums">
                        {userInfo.aff_code}
                      </code>
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={handleCopyAff}
                        icon={copiedAff ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
                      >
                        {copiedAff ? 'Copied' : 'Copy'}
                      </Button>
                    </div>
                  </Card>
                )}
              </>
            ) : null}
          </>
        )}
      </div>
    </div>
  )
}
