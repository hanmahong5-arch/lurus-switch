import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Users, Plus, Trash2, RefreshCw, AlertCircle, Edit2 } from 'lucide-react'
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

export function GatewayUserPage() {
  const { t } = useTranslation()
  const { status: serverStatus, adminToken } = useGatewayStore()

  const [users, setUsers] = useState<GatewayUser[]>([])
  const [page, setPage] = useState(0)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Modal state
  const [showModal, setShowModal] = useState(false)
  const [editing, setEditing] = useState<Partial<GatewayUser> & { password?: string } | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)

  const client = serverStatus?.running && adminToken
    ? createGatewayClient(serverStatus.url, adminToken)
    : null

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
          <Users className="h-6 w-6 text-purple-400" />
          {t('gateway.users')}
        </h2>
        <div className="flex gap-2">
          <button
            onClick={() => load()}
            disabled={loading}
            className="flex items-center gap-1 px-3 py-1.5 rounded-md border border-border hover:bg-muted text-sm"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
          <button
            onClick={() => { setEditing({ status: 1, role: 1, quota: 0 }); setShowModal(true) }}
            className="flex items-center gap-1 px-3 py-1.5 rounded-md bg-indigo-600 hover:bg-indigo-500 text-white text-sm"
          >
            <Plus className="h-4 w-4" />
            {t('gateway.createUser')}
          </button>
        </div>
      </div>

      <SearchBar value={keyword} onChange={setKeyword} onSearch={handleSearch} placeholder={t('gateway.search')} />

      {error && (
        <div className="text-sm text-red-400 bg-red-900/20 rounded px-3 py-2">{error}</div>
      )}

      <div className="rounded-lg border border-border overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-muted-foreground">
            <tr>
              <th className="text-left px-4 py-2">ID</th>
              <th className="text-left px-4 py-2">{t('gateway.userUsername')}</th>
              <th className="text-left px-4 py-2">{t('gateway.userDisplayName')}</th>
              <th className="text-left px-4 py-2">{t('gateway.userEmail')}</th>
              <th className="text-left px-4 py-2">{t('gateway.userRole')}</th>
              <th className="text-left px-4 py-2">{t('gateway.userQuota')}</th>
              <th className="text-left px-4 py-2">{t('gateway.userGroup')}</th>
              <th className="text-left px-4 py-2">{t('gateway.userStatus')}</th>
              <th className="text-right px-4 py-2">{t('gateway.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {users.length === 0 && (
              <tr>
                <td colSpan={9} className="text-center py-8 text-muted-foreground">
                  {loading ? t('status.loading') : t('gateway.noUsers')}
                </td>
              </tr>
            )}
            {users.map((u) => (
              <tr key={u.id} className="border-t border-border hover:bg-muted/30">
                <td className="px-4 py-2 text-muted-foreground">{u.id}</td>
                <td className="px-4 py-2 font-medium">{u.username}</td>
                <td className="px-4 py-2">{u.display_name || '-'}</td>
                <td className="px-4 py-2 text-xs text-muted-foreground">{u.email || '-'}</td>
                <td className="px-4 py-2">
                  <span className="text-xs rounded px-1.5 py-0.5 bg-muted/60">
                    {ROLE_LABELS[u.role] ?? `Role ${u.role}`}
                  </span>
                </td>
                <td className="px-4 py-2">{u.used_quota} / {u.quota}</td>
                <td className="px-4 py-2 text-xs">{u.group || '-'}</td>
                <td className="px-4 py-2">
                  <button onClick={() => handleToggleStatus(u)}>
                    <StatusBadge status={u.status === 1 ? 'enabled' : 'disabled'} />
                  </button>
                </td>
                <td className="px-4 py-2 text-right">
                  <div className="flex justify-end gap-1">
                    <button
                      onClick={() => { setEditing(u); setShowModal(true) }}
                      title={t('gateway.edit')}
                      className="p-1 hover:text-indigo-400"
                    >
                      <Edit2 className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => setConfirmDelete(u.id)}
                      title={t('gateway.delete')}
                      className="p-1 hover:text-red-400"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

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
      {showModal && editing && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-card border border-border rounded-lg p-6 w-[28rem] max-h-[80vh] overflow-y-auto space-y-4">
            <h3 className="font-semibold">{editing.id ? t('gateway.editUser') : t('gateway.createUser')}</h3>
            <div className="space-y-3">
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.userUsername')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.username ?? ''}
                  onChange={(e) => setEditing({ ...editing, username: e.target.value })}
                />
              </label>
              {!editing.id && (
                <label className="block text-sm">
                  <span className="text-muted-foreground">{t('gateway.userPassword')}</span>
                  <input
                    type="password"
                    className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                    value={editing.password ?? ''}
                    onChange={(e) => setEditing({ ...editing, password: e.target.value })}
                  />
                </label>
              )}
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.userDisplayName')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.display_name ?? ''}
                  onChange={(e) => setEditing({ ...editing, display_name: e.target.value })}
                />
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.userEmail')}</span>
                <input
                  type="email"
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.email ?? ''}
                  onChange={(e) => setEditing({ ...editing, email: e.target.value })}
                />
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.userRole')}</span>
                <select
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.role ?? 1}
                  onChange={(e) => setEditing({ ...editing, role: parseInt(e.target.value) })}
                >
                  {ROLE_OPTIONS.map((o) => (
                    <option key={o.value} value={o.value}>{o.label}</option>
                  ))}
                </select>
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.userQuota')}</span>
                <input
                  type="number"
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.quota ?? 0}
                  onChange={(e) => setEditing({ ...editing, quota: parseInt(e.target.value) || 0 })}
                />
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('gateway.userGroup')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={editing.group ?? ''}
                  onChange={(e) => setEditing({ ...editing, group: e.target.value })}
                />
              </label>
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => { setShowModal(false); setEditing(null) }}
                className="px-4 py-1.5 rounded border border-border text-sm hover:bg-muted"
              >
                {t('gateway.cancel')}
              </button>
              <button
                onClick={handleSave}
                className="px-4 py-1.5 rounded bg-indigo-600 hover:bg-indigo-500 text-white text-sm"
              >
                {t('gateway.save')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
