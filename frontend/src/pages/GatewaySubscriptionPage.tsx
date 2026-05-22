import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CreditCard, Plus, Trash2, Edit2, RefreshCw, AlertCircle, Link } from 'lucide-react'
import { Button, Card } from '../components/ui'
import { useGatewayStore } from '../stores/gatewayStore'
import { createGatewayClient, type GatewaySubscriptionPlan } from '../lib/gateway-api'
import { ConfirmModal } from '../components/gateway/ConfirmModal'

type Tab = 'plans' | 'bind'

const STATUS_OPTIONS = [
  { value: 1, label: 'Enabled' },
  { value: 2, label: 'Disabled' },
]

export function GatewaySubscriptionPage() {
  const { t } = useTranslation()
  const { status: serverStatus, adminToken } = useGatewayStore()

  // Shared state
  const [tab, setTab] = useState<Tab>('plans')
  const [plans, setPlans] = useState<GatewaySubscriptionPlan[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Plans tab state
  const [showModal, setShowModal] = useState(false)
  const [editing, setEditing] = useState<Partial<GatewaySubscriptionPlan> | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)

  // Bind tab state
  const [bindUserId, setBindUserId] = useState('')
  const [bindPlanId, setBindPlanId] = useState('')
  const [lookupUserId, setLookupUserId] = useState('')
  const [lookupResults, setLookupResults] = useState<GatewaySubscriptionPlan[]>([])

  const client = serverStatus?.running && adminToken
    ? createGatewayClient(serverStatus.url, adminToken)
    : null

  const loadPlans = async () => {
    if (!client) return
    setLoading(true)
    setError(null)
    try {
      const res = await client.getSubscriptionPlans()
      setPlans(res.data ?? [])
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { loadPlans() }, [serverStatus?.running, adminToken])

  const handleSave = async () => {
    if (!client || !editing) return
    try {
      if (editing.id) {
        await client.updateSubscriptionPlan(editing as GatewaySubscriptionPlan)
      } else {
        await client.createSubscriptionPlan(editing)
      }
      setShowModal(false)
      setEditing(null)
      await loadPlans()
    } catch (e) {
      setError(String(e))
    }
  }

  const handleDelete = async () => {
    if (!client || confirmDelete === null) return
    try {
      await client.deleteSubscriptionPlan(confirmDelete)
      setPlans((prev) => prev.filter((p) => p.id !== confirmDelete))
      setConfirmDelete(null)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleBind = async () => {
    if (!client) return
    const userId = Number(bindUserId)
    const planId = Number(bindPlanId)
    if (!userId || !planId) return
    setError(null)
    try {
      await client.bindSubscription(userId, planId)
      setBindUserId('')
      setBindPlanId('')
    } catch (e) {
      setError(String(e))
    }
  }

  const handleLookup = async () => {
    if (!client) return
    const userId = Number(lookupUserId)
    if (!userId) return
    setError(null)
    try {
      const res = await client.getUserSubscriptions(userId)
      setLookupResults(res.data ?? [])
    } catch (e) {
      setError(String(e))
    }
  }

  const openCreate = () => {
    setEditing({ status: 1, level: 0, pricing: 0, duration: 30, quota: 0 })
    setShowModal(true)
  }

  const openEdit = (plan: GatewaySubscriptionPlan) => {
    setEditing({ ...plan })
    setShowModal(true)
  }

  if (!serverStatus?.running) {
    return (
      <div className="flex flex-col h-full items-center justify-center text-muted-foreground gap-2">
        <AlertCircle className="h-8 w-8" />
        <p>{t('gateway.status.stopped')}</p>
      </div>
    )
  }

  const tabs: { id: Tab; label: string }[] = [
    { id: 'plans', label: t('gateway.subscriptionPlans', 'Plans') },
    { id: 'bind', label: t('gateway.subscriptionBind', 'Bind Subscription') },
  ]

  return (
    <div className="flex flex-col h-full overflow-y-auto p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold flex items-center gap-2">
          <CreditCard className="h-6 w-6 text-primary" />
          {t('gateway.subscriptions', 'Subscriptions')}
        </h2>
        <div className="flex gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={loadPlans}
            disabled={loading}
            loading={loading}
            icon={!loading ? <RefreshCw className="h-4 w-4" /> : undefined}
          />
          {tab === 'plans' && (
            <Button size="sm" onClick={openCreate} icon={<Plus className="h-4 w-4" />}>
              {t('gateway.addPlan', 'Add Plan')}
            </Button>
          )}
        </div>
      </div>

      {error && (
        <div className="text-sm text-red-400 bg-red-500/10 border border-red-500/30 rounded px-3 py-2 font-mono">▸ {error}</div>
      )}

      {/* Tab Bar */}
      <div className="flex border-b border-border">
        {tabs.map((item) => {
          const isActive = tab === item.id
          return (
            <button
              key={item.id}
              onClick={() => setTab(item.id)}
              className={`px-4 py-2 -mb-px border-b-2 transition-all duration-150 ${
                isActive
                  ? 'border-primary text-primary'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              <span className={isActive ? 'font-mono text-[11px] tracking-[0.12em]' : 'text-sm font-medium'}>
                {isActive ? `[ ${item.label.toUpperCase()} ]` : item.label}
              </span>
            </button>
          )
        })}
      </div>

      {/* Plans Tab */}
      {tab === 'plans' && (
        <Card variant="default" className="overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-card-recessed">
              <tr className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
                <th className="text-left px-4 py-2">[ ID ]</th>
                <th className="text-left px-4 py-2">[ {t('gateway.planName', 'Name').toUpperCase()} ]</th>
                <th className="text-left px-4 py-2">[ {t('gateway.planLevel', 'Level').toUpperCase()} ]</th>
                <th className="text-left px-4 py-2">[ {t('gateway.planPricing', 'Pricing').toUpperCase()} ]</th>
                <th className="text-left px-4 py-2">[ {t('gateway.planDuration', 'Duration (days)').toUpperCase()} ]</th>
                <th className="text-left px-4 py-2">[ {t('gateway.planQuota', 'Quota').toUpperCase()} ]</th>
                <th className="text-left px-4 py-2">[ {t('gateway.planGroup', 'Group').toUpperCase()} ]</th>
                <th className="text-left px-4 py-2">[ {t('gateway.planStatus', 'Status').toUpperCase()} ]</th>
                <th className="text-right px-4 py-2">[ {t('gateway.actions', 'Actions').toUpperCase()} ]</th>
              </tr>
            </thead>
            <tbody>
              {plans.length === 0 && (
                <tr>
                  <td colSpan={9} className="text-center py-8 text-muted-foreground font-mono">
                    ▪ {loading ? t('status.loading') : t('gateway.noPlans', 'No plans')}
                  </td>
                </tr>
              )}
              {plans.map((plan) => (
                <tr key={plan.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                  <td className="px-4 py-2 text-muted-foreground font-mono tabular-nums">{plan.id}</td>
                  <td className="px-4 py-2 font-medium">{plan.name}</td>
                  <td className="px-4 py-2 font-mono tabular-nums">{plan.level}</td>
                  <td className="px-4 py-2 font-mono tabular-nums">{plan.pricing}</td>
                  <td className="px-4 py-2 font-mono tabular-nums">{plan.duration}</td>
                  <td className="px-4 py-2 font-mono tabular-nums">{plan.quota}</td>
                  <td className="px-4 py-2 font-mono">{plan.group || '-'}</td>
                  <td className="px-4 py-2">
                    <span className={`font-mono text-[10px] uppercase tracking-[0.08em] rounded px-1.5 py-0.5 ${
                      plan.status === 1
                        ? 'bg-emerald-500/15 text-emerald-400'
                        : 'bg-card-recessed text-muted-foreground'
                    }`}>
                      {plan.status === 1 ? '▸ Enabled' : '▪ Disabled'}
                    </span>
                  </td>
                  <td className="px-4 py-2 text-right">
                    <div className="flex justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => openEdit(plan)}
                        title="Edit"
                        icon={<Edit2 className="h-3.5 w-3.5" />}
                        className="hover:text-primary"
                      />
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setConfirmDelete(plan.id)}
                        title="Delete"
                        icon={<Trash2 className="h-3.5 w-3.5" />}
                        className="hover:text-red-400 hover:bg-red-500/10"
                      />
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      )}

      {/* Bind Tab */}
      {tab === 'bind' && (
        <div className="space-y-6">
          {/* Bind Form */}
          <Card variant="default" className="p-4 space-y-4">
            <h3 className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground flex items-center gap-2">
              <Link className="h-3.5 w-3.5 text-primary" />
              [ {t('gateway.bindSubscription', 'Bind Subscription').toUpperCase()} ]
            </h3>
            <div className="flex items-end gap-3">
              <label className="block text-sm flex-1">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.userId', 'User ID')}</span>
                <input
                  type="number"
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono tabular-nums focus:outline-none focus:ring-1 focus:ring-primary"
                  value={bindUserId}
                  onChange={(e) => setBindUserId(e.target.value)}
                  placeholder="1"
                />
              </label>
              <label className="block text-sm flex-1">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.plan', 'Plan')}</span>
                <select
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                  value={bindPlanId}
                  onChange={(e) => setBindPlanId(e.target.value)}
                >
                  <option value="">{t('gateway.selectPlan', '-- Select Plan --')}</option>
                  {plans.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name} (ID: {p.id})
                    </option>
                  ))}
                </select>
              </label>
              <Button
                size="sm"
                onClick={handleBind}
                disabled={!bindUserId || !bindPlanId}
                className="whitespace-nowrap"
              >
                {t('gateway.bind', 'Bind')}
              </Button>
            </div>
          </Card>

          {/* Lookup User Subscriptions */}
          <Card variant="default" className="p-4 space-y-4">
            <h3 className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground">
              [ {t('gateway.lookupSubscriptions', 'Lookup User Subscriptions').toUpperCase()} ]
            </h3>
            <div className="flex items-end gap-3">
              <label className="block text-sm flex-1">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.userId', 'User ID')}</span>
                <input
                  type="number"
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono tabular-nums focus:outline-none focus:ring-1 focus:ring-primary"
                  value={lookupUserId}
                  onChange={(e) => setLookupUserId(e.target.value)}
                  placeholder="1"
                />
              </label>
              <Button
                variant="secondary"
                size="sm"
                onClick={handleLookup}
                disabled={!lookupUserId}
                className="whitespace-nowrap"
              >
                {t('gateway.search', 'Search')}
              </Button>
            </div>

            {lookupResults.length > 0 && (
              <Card variant="default" className="overflow-hidden">
                <table className="w-full text-sm">
                  <thead className="bg-card-recessed">
                    <tr className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
                      <th className="text-left px-4 py-2">[ ID ]</th>
                      <th className="text-left px-4 py-2">[ {t('gateway.planName', 'Name').toUpperCase()} ]</th>
                      <th className="text-left px-4 py-2">[ {t('gateway.planLevel', 'Level').toUpperCase()} ]</th>
                      <th className="text-left px-4 py-2">[ {t('gateway.planDuration', 'Duration (days)').toUpperCase()} ]</th>
                      <th className="text-left px-4 py-2">[ {t('gateway.planQuota', 'Quota').toUpperCase()} ]</th>
                      <th className="text-left px-4 py-2">[ {t('gateway.planStatus', 'Status').toUpperCase()} ]</th>
                    </tr>
                  </thead>
                  <tbody>
                    {lookupResults.map((sub) => (
                      <tr key={sub.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                        <td className="px-4 py-2 text-muted-foreground font-mono tabular-nums">{sub.id}</td>
                        <td className="px-4 py-2 font-medium">{sub.name}</td>
                        <td className="px-4 py-2 font-mono tabular-nums">{sub.level}</td>
                        <td className="px-4 py-2 font-mono tabular-nums">{sub.duration}</td>
                        <td className="px-4 py-2 font-mono tabular-nums">{sub.quota}</td>
                        <td className="px-4 py-2">
                          <span className={`font-mono text-[10px] uppercase tracking-[0.08em] rounded px-1.5 py-0.5 ${
                            sub.status === 1
                              ? 'bg-emerald-500/15 text-emerald-400'
                              : 'bg-card-recessed text-muted-foreground'
                          }`}>
                            {sub.status === 1 ? '▸ Enabled' : '▪ Disabled'}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </Card>
            )}

            {lookupResults.length === 0 && lookupUserId && (
              <p className="text-sm text-muted-foreground font-mono">
                ▪ {t('gateway.noSubscriptions', 'No subscriptions found')}
              </p>
            )}
          </Card>
        </div>
      )}

      {/* Create/Edit Modal */}
      {showModal && editing && (
        <div className="fixed inset-0 bg-black/40 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="bg-card border border-border rounded-lg p-6 w-[28rem] space-y-4 max-h-[90vh] overflow-y-auto">
            <h3 className="font-semibold">
              {editing.id
                ? t('gateway.editPlan', 'Edit Plan')
                : t('gateway.addPlan', 'Add Plan')}
            </h3>

            <div className="space-y-3">
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.planName', 'Name')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.name ?? ''}
                  onChange={(e) => setEditing({ ...editing, name: e.target.value })}
                />
              </label>

              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.planDescription', 'Description')}</span>
                <textarea
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm resize-none"
                  rows={3}
                  value={editing.description ?? ''}
                  onChange={(e) => setEditing({ ...editing, description: e.target.value })}
                />
              </label>

              <div className="grid grid-cols-2 gap-3">
                <label className="block text-sm">
                  <span className="text-muted-foreground">{t('gateway.planLevel', 'Level')}</span>
                  <input
                    type="number"
                    className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                    value={editing.level ?? 0}
                    onChange={(e) => setEditing({ ...editing, level: Number(e.target.value) })}
                  />
                </label>

                <label className="block text-sm">
                  <span className="text-muted-foreground">{t('gateway.planPricing', 'Pricing')}</span>
                  <input
                    type="number"
                    className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                    value={editing.pricing ?? 0}
                    onChange={(e) => setEditing({ ...editing, pricing: Number(e.target.value) })}
                  />
                </label>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <label className="block text-sm">
                  <span className="text-muted-foreground">{t('gateway.planDuration', 'Duration (days)')}</span>
                  <input
                    type="number"
                    className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                    value={editing.duration ?? 30}
                    onChange={(e) => setEditing({ ...editing, duration: Number(e.target.value) })}
                  />
                </label>

                <label className="block text-sm">
                  <span className="text-muted-foreground">{t('gateway.planQuota', 'Quota')}</span>
                  <input
                    type="number"
                    className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                    value={editing.quota ?? 0}
                    onChange={(e) => setEditing({ ...editing, quota: Number(e.target.value) })}
                  />
                </label>
              </div>

              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.planGroup', 'Group')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.group ?? ''}
                  onChange={(e) => setEditing({ ...editing, group: e.target.value })}
                />
              </label>

              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.planFeatures', 'Features')}</span>
                <textarea
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm resize-none"
                  rows={3}
                  value={editing.features ?? ''}
                  onChange={(e) => setEditing({ ...editing, features: e.target.value })}
                />
              </label>

              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.planStatus', 'Status')}</span>
                <select
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.status ?? 1}
                  onChange={(e) => setEditing({ ...editing, status: Number(e.target.value) })}
                >
                  {STATUS_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>{opt.label}</option>
                  ))}
                </select>
              </label>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => { setShowModal(false); setEditing(null) }}
                className="px-4 py-1.5 rounded border border-border text-sm hover:bg-muted"
              >
                {t('settings.data.cancel')}
              </button>
              <button
                onClick={handleSave}
                className="px-4 py-1.5 rounded bg-primary hover:bg-primary/90 text-primary-foreground text-sm"
              >
                {t('settings.save')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation */}
      <ConfirmModal
        open={confirmDelete !== null}
        title={t('gateway.deletePlan', 'Delete Plan')}
        desc={t('gateway.deletePlanConfirm', 'Are you sure you want to delete this subscription plan? This action cannot be undone.')}
        danger
        onConfirm={handleDelete}
        onCancel={() => setConfirmDelete(null)}
      />
    </div>
  )
}
