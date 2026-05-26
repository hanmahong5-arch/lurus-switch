import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Users, Plus, Trash2, RefreshCw, AlertCircle, Edit2, ArrowUpDown, AlertTriangle } from 'lucide-react'
import { Button, Card, Modal } from '../components/ui'
import { useGatewayStore } from '../stores/gatewayStore'
import { createGatewayClient, type GatewayUser } from '../lib/gateway-api'
import { SearchBar } from '../components/gateway/SearchBar'
import { Pagination } from '../components/gateway/Pagination'
import { StatusBadge } from '../components/gateway/StatusBadge'
import { ConfirmModal } from '../components/gateway/ConfirmModal'

const PER_PAGE = 50
const ROLE_LABELS: Record<number, string> = { 1: 'User', 10: 'Admin', 100: 'Root' }
const ROLE_OPTIONS = [
  { value: 1, label: 'User' },
  { value: 10, label: 'Admin' },
  { value: 100, label: 'Root' },
]

// Wave 5 W5.2 — usage threshold for the "即将耗尽" badge. used_quota / quota
// ≥ 0.9 surfaces a red chip so resellers can DM the user before they hit the
// wall. Unlimited (quota=0) users are excluded.
const QUOTA_WARN_RATIO = 0.9

type UserSort = 'id' | 'used_quota' | 'request_count'

function usageRatio(u: GatewayUser): number {
  if (u.quota <= 0) return 0
  return u.used_quota / u.quota
}

export function GatewayUserPage() {
  const { t } = useTranslation()
  const { status: serverStatus, adminToken } = useGatewayStore()

  const [users, setUsers] = useState<GatewayUser[]>([])
  const [page, setPage] = useState(0)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [sortBy, setSortBy] = useState<UserSort>('id')
  const [sortDesc, setSortDesc] = useState(true)

  // Modal state
  const [showModal, setShowModal] = useState(false)
  const [editing, setEditing] = useState<Partial<GatewayUser> & { password?: string } | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)

  const client = serverStatus?.running && adminToken
    ? createGatewayClient(serverStatus.url, adminToken)
    : null

  // Wave 5 W5.2 — client-side sort. Hub's user list endpoint doesn't accept
  // a sort param, so we sort the in-memory page after fetch. For the typical
  // <100 users/page that's cheap.
  const sortedUsers = useMemo(() => {
    const dir = sortDesc ? -1 : 1
    return [...users].sort((a, b) => {
      const va = (a[sortBy] as number) ?? 0
      const vb = (b[sortBy] as number) ?? 0
      return va === vb ? 0 : va < vb ? -dir : dir
    })
  }, [users, sortBy, sortDesc])

  const cycleSort = (key: UserSort) => {
    if (sortBy === key) {
      setSortDesc((d) => !d)
    } else {
      setSortBy(key)
      setSortDesc(true)
    }
  }

  const load = async (p = page) => {
    if (!client) return
    setLoading(true)
    setError(null)
    try {
      const res = keyword.trim()
        ? await client.searchUsers(keyword.trim(), p, PER_PAGE)
        : await client.getUsers(p, PER_PAGE)
      setUsers(res.data ?? [])
      setTotal(res.data?.length === PER_PAGE ? (p + 2) * PER_PAGE : (p * PER_PAGE) + (res.data?.length ?? 0))
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [serverStatus?.running, adminToken])

  const handlePageChange = (p: number) => {
    setPage(p)
    load(p)
  }

  const handleSearch = () => {
    setPage(0)
    load(0)
  }

  const handleDelete = async () => {
    if (!client || confirmDelete === null) return
    try {
      await client.deleteUser(confirmDelete)
      setUsers((prev) => prev.filter((u) => u.id !== confirmDelete))
      setConfirmDelete(null)
    } catch (e) {
      setError(String(e))
    }
  }

  const handleSave = async () => {
    if (!client || !editing) return
    try {
      if (editing.id) {
        await client.updateUser(editing as GatewayUser)
      } else {
        await client.createUser(editing)
      }
      setShowModal(false)
      setEditing(null)
      await load()
    } catch (e) {
      setError(String(e))
    }
  }

  const handleToggleStatus = async (user: GatewayUser) => {
    if (!client) return
    const action = user.status === 1 ? 'disable' : 'enable'
    try {
      await client.manageUser({ id: user.id, action })
      setUsers((prev) => prev.map((u) => u.id === user.id ? { ...u, status: u.status === 1 ? 2 : 1 } : u))
    } catch (e) {
      setError(String(e))
    }
  }

  if (!serverStatus?.running) {
    return (
      <div className="flex flex-col h-full items-center justify-center text-muted-foreground gap-2">
        <AlertCircle className="h-8 w-8" />
        <p>{t('gateway.status.stopped')}</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold flex items-center gap-2">
          <Users className="h-6 w-6 text-primary" />
          {t('gateway.users')}
        </h2>
        <div className="flex gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => load()}
            disabled={loading}
            loading={loading}
            icon={!loading ? <RefreshCw className="h-4 w-4" /> : undefined}
          />
          <Button
            size="sm"
            onClick={() => { setEditing({ status: 1, role: 1, quota: 0 }); setShowModal(true) }}
            icon={<Plus className="h-4 w-4" />}
          >
            {t('gateway.createUser')}
          </Button>
        </div>
      </div>

      <SearchBar value={keyword} onChange={setKeyword} onSearch={handleSearch} placeholder={t('gateway.search')} />

      {error && (
        <div className="text-sm text-red-400 bg-red-500/10 border border-red-500/30 rounded px-3 py-2 font-mono">▸ {error}</div>
      )}

      <Card variant="default" className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-card-recessed">
            <tr className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
              <SortableTh sortKey="id" current={sortBy} desc={sortDesc} onClick={cycleSort}>ID</SortableTh>
              <th className="text-left px-4 py-2">[ {t('gateway.userUsername').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.userDisplayName').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.userEmail').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.userRole').toUpperCase()} ]</th>
              <SortableTh sortKey="used_quota" current={sortBy} desc={sortDesc} onClick={cycleSort}>
                {t('gateway.userQuota').toUpperCase()}
              </SortableTh>
              <SortableTh sortKey="request_count" current={sortBy} desc={sortDesc} onClick={cycleSort}>
                {t('gateway.userRequests', '请求数').toUpperCase()}
              </SortableTh>
              <th className="text-left px-4 py-2">[ {t('gateway.userGroup').toUpperCase()} ]</th>
              <th className="text-left px-4 py-2">[ {t('gateway.userStatus').toUpperCase()} ]</th>
              <th className="text-right px-4 py-2">[ {t('gateway.actions').toUpperCase()} ]</th>
            </tr>
          </thead>
          <tbody>
            {sortedUsers.length === 0 && (
              <tr>
                <td colSpan={10} className="text-center py-8 text-muted-foreground font-mono">
                  ▪ {loading ? t('status.loading') : t('gateway.noUsers')}
                </td>
              </tr>
            )}
            {sortedUsers.map((u) => {
              const ratio = usageRatio(u)
              const nearLimit = u.quota > 0 && ratio >= QUOTA_WARN_RATIO
              return (
              <tr key={u.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                <td className="px-4 py-2 text-muted-foreground font-mono tabular-nums">{u.id}</td>
                <td className="px-4 py-2 font-medium">{u.username}</td>
                <td className="px-4 py-2">{u.display_name || '-'}</td>
                <td className="px-4 py-2 text-xs text-muted-foreground font-mono">{u.email || '-'}</td>
                <td className="px-4 py-2">
                  <span className="font-mono text-[10px] uppercase tracking-[0.08em] rounded px-1.5 py-0.5 bg-card-recessed text-muted-foreground">
                    {ROLE_LABELS[u.role] ?? `Role ${u.role}`}
                  </span>
                </td>
                <td className="px-4 py-2 font-mono tabular-nums">
                  <div className="flex items-center gap-2">
                    <span>{u.used_quota} / {u.quota || '∞'}</span>
                    {nearLimit && (
                      <span
                        className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-red-500/15 text-red-400 text-[10px]"
                        title={t('gateway.userNearLimitHint', '使用率 ≥ 90%')}
                      >
                        <AlertTriangle className="h-2.5 w-2.5" />
                        {t('gateway.userNearLimit', '即将耗尽')}
                      </span>
                    )}
                  </div>
                </td>
                <td className="px-4 py-2 font-mono tabular-nums">{u.request_count ?? 0}</td>
                <td className="px-4 py-2 text-xs font-mono">{u.group || '-'}</td>
                <td className="px-4 py-2">
                  <button onClick={() => handleToggleStatus(u)} className="transition-opacity hover:opacity-80">
                    <StatusBadge status={u.status === 1 ? 'enabled' : 'disabled'} />
                  </button>
                </td>
                <td className="px-4 py-2 text-right">
                  <div className="flex justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => { setEditing(u); setShowModal(true) }}
                      title={t('gateway.edit')}
                      icon={<Edit2 className="h-3.5 w-3.5" />}
                      className="hover:text-primary"
                    />
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setConfirmDelete(u.id)}
                      title={t('gateway.delete')}
                      icon={<Trash2 className="h-3.5 w-3.5" />}
                      className="hover:text-red-400 hover:bg-red-500/10"
                    />
                  </div>
                </td>
              </tr>
            )})}
          </tbody>
        </table>
      </Card>

      <Pagination page={page} total={total} perPage={PER_PAGE} onPageChange={handlePageChange} />

      {/* Delete confirm */}
      <ConfirmModal
        open={confirmDelete !== null}
        title={t('gateway.deleteConfirmTitle')}
        desc={t('gateway.deleteConfirm')}
        danger
        onConfirm={handleDelete}
        onCancel={() => setConfirmDelete(null)}
      />

      {/* Create/Edit Modal */}
      <Modal
        open={showModal && !!editing}
        onClose={() => { setShowModal(false); setEditing(null) }}
        title={editing?.id ? t('gateway.editUser') : t('gateway.createUser')}
        icon={editing?.id ? Edit2 : Plus}
        size="md"
        footer={
          <>
            <Button variant="secondary" size="sm" onClick={() => { setShowModal(false); setEditing(null) }}>
              {t('gateway.cancel')}
            </Button>
            <Button size="sm" onClick={handleSave}>
              {t('gateway.save')}
            </Button>
          </>
        }
      >
        {editing && (
          <div className="space-y-3">
            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.userUsername')}</span>
              <input
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                value={editing.username ?? ''}
                onChange={(e) => setEditing({ ...editing, username: e.target.value })}
              />
            </label>
            {!editing.id && (
              <label className="block text-sm">
                <span className="text-muted-foreground font-mono text-xs">{t('gateway.userPassword')}</span>
                <input
                  type="password"
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                  value={editing.password ?? ''}
                  onChange={(e) => setEditing({ ...editing, password: e.target.value })}
                />
              </label>
            )}
            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.userDisplayName')}</span>
              <input
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                value={editing.display_name ?? ''}
                onChange={(e) => setEditing({ ...editing, display_name: e.target.value })}
              />
            </label>
            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.userEmail')}</span>
              <input
                type="email"
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                value={editing.email ?? ''}
                onChange={(e) => setEditing({ ...editing, email: e.target.value })}
              />
            </label>
            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.userRole')}</span>
              <select
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                value={editing.role ?? 1}
                onChange={(e) => setEditing({ ...editing, role: parseInt(e.target.value) })}
              >
                {ROLE_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>{o.label}</option>
                ))}
              </select>
            </label>
            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.userQuota')}</span>
              <input
                type="number"
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono tabular-nums focus:outline-none focus:ring-1 focus:ring-primary"
                value={editing.quota ?? 0}
                onChange={(e) => setEditing({ ...editing, quota: parseInt(e.target.value) || 0 })}
              />
            </label>
            <label className="block text-sm">
              <span className="text-muted-foreground font-mono text-xs">{t('gateway.userGroup')}</span>
              <input
                className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
                value={editing.group ?? ''}
                onChange={(e) => setEditing({ ...editing, group: e.target.value })}
              />
            </label>
          </div>
        )}
      </Modal>
    </div>
  )
}

interface SortableThProps {
  sortKey: UserSort
  current: UserSort
  desc: boolean
  onClick: (key: UserSort) => void
  children: React.ReactNode
}

function SortableTh({ sortKey, current, desc, onClick, children }: SortableThProps) {
  const active = current === sortKey
  return (
    <th className="text-left px-4 py-2">
      <button
        type="button"
        onClick={() => onClick(sortKey)}
        className={`inline-flex items-center gap-1 hover:text-foreground transition-colors ${active ? 'text-primary' : ''}`}
      >
        <span>[ {children} ]</span>
        {active ? <span className="text-[10px]">{desc ? '↓' : '↑'}</span> : <ArrowUpDown className="h-2.5 w-2.5 opacity-50" />}
      </button>
    </th>
  )
}
