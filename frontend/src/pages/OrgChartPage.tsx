import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Building2, Users, Plus, Trash2, UserX, RefreshCw, AlertTriangle, ChevronRight, Upload, X } from 'lucide-react'
import { useOrgChartStore, type DepartmentTreeNode, type Employee, type OrgRole, type CSVImportResult } from '../stores/orgChartStore'
import { Button, Card } from '../components/ui'

const ROLES: OrgRole[] = ['employee', 'team_lead', 'dept_admin', 'it_admin', 'compliance', 'finance']

export function OrgChartPage() {
  const { t } = useTranslation()
  const {
    tree, employees, selectedDeptId, showInactive, loading, error,
    load, selectDept, toggleInactive,
    createDept, deleteDept, createEmployee, updateEmployee, deactivateEmployee,
  } = useOrgChartStore()

  const [newDeptName, setNewDeptName] = useState('')
  const [newEmpDraft, setNewEmpDraft] = useState({ email: '', displayName: '', role: 'employee' as OrgRole })
  const [showImport, setShowImport] = useState(false)

  useEffect(() => {
    void load()
  }, [load])

  const handleAddDept = async () => {
    if (!newDeptName.trim()) return
    const r = await createDept({ name: newDeptName.trim(), parentId: selectedDeptId || '' })
    if (r) {
      setNewDeptName('')
      selectDept(r.id)
    }
  }

  const handleAddEmployee = async () => {
    if (!selectedDeptId || !newEmpDraft.email.trim()) return
    const r = await createEmployee({
      email: newEmpDraft.email.trim(),
      displayName: newEmpDraft.displayName.trim(),
      departmentId: selectedDeptId,
      role: newEmpDraft.role,
    })
    if (r) {
      setNewEmpDraft({ email: '', displayName: '', role: 'employee' })
    }
  }

  return (
    <div className="h-full overflow-auto p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-xl font-semibold flex items-center gap-2">
            <Building2 className="h-5 w-5 text-primary" />
            {t('orgchart.title', '组织架构 — 部门与员工')}
          </h1>
          <p className="text-xs text-muted-foreground mt-1">
            {t('orgchart.subtitle', 'Enterprise 模式下用于成本中心归集、SSO 绑定、按部门授权 token。')}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => setShowImport(true)}
            icon={<Upload className="h-3.5 w-3.5" />}
          >
            {t('orgchart.import.button', '批量导入 CSV')}
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => void load()}
            disabled={loading}
            loading={loading}
            icon={!loading ? <RefreshCw className="h-3.5 w-3.5" /> : undefined}
          >
            {t('common.refresh', '刷新')}
          </Button>
        </div>
      </div>

      {showImport && <CSVImportModal onClose={() => setShowImport(false)} defaultDeptId={selectedDeptId} />}

      {error && (
        <Card variant="default" className="mb-3 p-2 border-red-500/30 bg-red-500/10 text-red-400 text-xs flex items-center gap-2 font-mono">
          <AlertTriangle className="h-3.5 w-3.5" />
          ▸ {error}
        </Card>
      )}

      <div className="grid grid-cols-12 gap-4">
        {/* Tree */}
        <section className="col-span-12 lg:col-span-4 rounded-lg border border-border bg-card">
          <header className="p-3 border-b border-border flex items-center justify-between">
            <h2 className="text-sm font-medium">{t('orgchart.tree.title', '部门树')}</h2>
            <span className="text-[10px] text-muted-foreground">{tree.length} {t('orgchart.tree.roots', '根')}</span>
          </header>
          <div className="p-2 max-h-[420px] overflow-auto">
            {tree.length === 0 ? (
              <div className="p-4 text-center text-xs text-muted-foreground">{t('orgchart.tree.empty', '尚未创建部门')}</div>
            ) : (
              <ul className="text-sm">
                {tree.map(n => (
                  <DeptNode
                    key={n.department.id}
                    node={n}
                    selectedId={selectedDeptId}
                    onSelect={selectDept}
                    onDelete={deleteDept}
                    depth={0}
                  />
                ))}
              </ul>
            )}
          </div>
          <footer className="p-3 border-t border-border flex gap-2">
            <input
              value={newDeptName}
              onChange={(e) => setNewDeptName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleAddDept()}
              placeholder={selectedDeptId ? t('orgchart.tree.addChildPh', '+ 子部门名称') : t('orgchart.tree.addRootPh', '+ 根部门名称')}
              className="flex-1 px-2 py-1 rounded border border-border bg-background text-xs"
            />
            <button
              onClick={handleAddDept}
              className="px-2 py-1 rounded bg-primary text-primary-foreground text-xs flex items-center gap-1"
            >
              <Plus className="h-3.5 w-3.5" />
              {t('common.add', '添加')}
            </button>
          </footer>
        </section>

        {/* Employees */}
        <section className="col-span-12 lg:col-span-8 rounded-lg border border-border bg-card">
          <header className="p-3 border-b border-border flex items-center justify-between">
            <div>
              <h2 className="text-sm font-medium flex items-center gap-2">
                <Users className="h-4 w-4" />
                {t('orgchart.employees.title', '员工')}
              </h2>
              <p className="text-[10px] text-muted-foreground mt-0.5">
                {selectedDeptId
                  ? t('orgchart.employees.deptSel', '当前部门 · {{count}} 人', { count: employees.length })
                  : t('orgchart.employees.noDept', '请先在左侧选择部门')}
              </p>
            </div>
            <label className="text-[11px] flex items-center gap-1 cursor-pointer select-none">
              <input
                type="checkbox"
                checked={showInactive}
                onChange={toggleInactive}
                className="h-3 w-3"
              />
              {t('orgchart.employees.includeInactive', '含已停用')}
            </label>
          </header>

          {/* Add employee inline */}
          {selectedDeptId && (
            <div className="p-3 border-b border-border bg-muted/20 grid grid-cols-1 md:grid-cols-4 gap-2">
              <input
                value={newEmpDraft.email}
                onChange={(e) => setNewEmpDraft({ ...newEmpDraft, email: e.target.value })}
                placeholder={t('orgchart.employees.emailPh', 'email')}
                className="px-2 py-1 rounded border border-border bg-background text-xs"
              />
              <input
                value={newEmpDraft.displayName}
                onChange={(e) => setNewEmpDraft({ ...newEmpDraft, displayName: e.target.value })}
                placeholder={t('orgchart.employees.namePh', '姓名（选填）')}
                className="px-2 py-1 rounded border border-border bg-background text-xs"
              />
              <select
                value={newEmpDraft.role}
                onChange={(e) => setNewEmpDraft({ ...newEmpDraft, role: e.target.value as OrgRole })}
                className="px-2 py-1 rounded border border-border bg-background text-xs"
              >
                {ROLES.map(r => <option key={r} value={r}>{r}</option>)}
              </select>
              <button
                onClick={handleAddEmployee}
                className="px-2 py-1 rounded bg-primary text-primary-foreground text-xs flex items-center justify-center gap-1"
              >
                <Plus className="h-3.5 w-3.5" />
                {t('common.add', '添加')}
              </button>
            </div>
          )}

          {/* Employee table */}
          <div className="overflow-x-auto max-h-[400px] overflow-y-auto">
            <table className="w-full text-xs">
              <thead className="text-[10px] uppercase tracking-wider text-muted-foreground bg-muted/30 sticky top-0">
                <tr>
                  <th className="text-left px-3 py-2">{t('orgchart.col.email', 'email')}</th>
                  <th className="text-left px-3 py-2">{t('orgchart.col.name', '姓名')}</th>
                  <th className="text-left px-3 py-2">{t('orgchart.col.role', '角色')}</th>
                  <th className="text-left px-3 py-2">SSO</th>
                  <th className="text-left px-3 py-2">{t('orgchart.col.status', '状态')}</th>
                  <th className="text-right px-3 py-2"></th>
                </tr>
              </thead>
              <tbody>
                {employees.length === 0 && (
                  <tr><td colSpan={6} className="px-3 py-6 text-center text-muted-foreground">{t('common.empty', '暂无数据')}</td></tr>
                )}
                {employees.map(e => (
                  <EmployeeRow
                    key={e.id}
                    employee={e}
                    onRoleChange={(role) => void updateEmployee({ id: e.id, role })}
                    onDeactivate={() => void deactivateEmployee(e.id)}
                  />
                ))}
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </div>
  )
}

function DeptNode({
  node, selectedId, onSelect, onDelete, depth,
}: {
  node: DepartmentTreeNode
  selectedId: string
  onSelect: (id: string) => void
  onDelete: (id: string) => void
  depth: number
}) {
  const [open, setOpen] = useState(true)
  const isSel = node.department.id === selectedId
  return (
    <li>
      <div
        className={`flex items-center gap-1 px-2 py-1.5 rounded cursor-pointer hover:bg-muted/40 ${isSel ? 'bg-primary/10' : ''}`}
        style={{ paddingLeft: `${0.5 + depth * 0.75}rem` }}
        onClick={() => onSelect(node.department.id)}
      >
        {node.children.length > 0 ? (
          <button onClick={(e) => { e.stopPropagation(); setOpen(!open) }} className="text-muted-foreground hover:text-foreground">
            <ChevronRight className={`h-3.5 w-3.5 transition-transform ${open ? 'rotate-90' : ''}`} />
          </button>
        ) : (
          <span className="inline-block w-3.5" />
        )}
        <span className="flex-1 truncate text-xs font-medium">{node.department.name}</span>
        <span className="text-[10px] text-muted-foreground">{node.employeeCount}</span>
        <button
          onClick={(e) => {
            e.stopPropagation()
            if (confirm(`Delete department "${node.department.name}"?`)) onDelete(node.department.id)
          }}
          className="text-muted-foreground hover:text-red-400 p-0.5"
          title="delete"
        >
          <Trash2 className="h-3 w-3" />
        </button>
      </div>
      {open && node.children.length > 0 && (
        <ul>
          {node.children.map(c => (
            <DeptNode key={c.department.id} node={c} selectedId={selectedId} onSelect={onSelect} onDelete={onDelete} depth={depth + 1} />
          ))}
        </ul>
      )}
    </li>
  )
}

function CSVImportModal({ onClose, defaultDeptId }: { onClose: () => void; defaultDeptId: string }) {
  const { t } = useTranslation()
  const importCSV = useOrgChartStore((s) => s.importCSV)
  const [content, setContent] = useState('')
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState<CSVImportResult | null>(null)

  const handleImport = async () => {
    setBusy(true)
    setResult(null)
    const r = await importCSV(content, defaultDeptId)
    setBusy(false)
    if (r) setResult(r)
  }

  return (
    <div
      className="fixed inset-0 bg-black/30 backdrop-blur-sm z-50 flex items-center justify-center p-4"
      onClick={onClose}
    >
      <div
        className="rounded-lg border border-border bg-card max-w-2xl w-full max-h-[85vh] overflow-hidden flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <header className="p-4 border-b border-border flex items-center justify-between">
          <h2 className="text-base font-semibold flex items-center gap-2">
            <Upload className="h-4 w-4" />
            {t('orgchart.import.title', '批量导入员工 (SCIM-lite)')}
          </h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="h-4 w-4" />
          </button>
        </header>

        <div className="p-4 space-y-3 overflow-auto flex-1">
          <div className="text-xs text-muted-foreground space-y-1">
            <p>{t('orgchart.import.hint1', '粘贴 CSV，列顺序：email, display_name, department, role。可选 header 行（含 "email" 关键字会被识别为标题）。')}</p>
            <p>{t('orgchart.import.hint2', 'department 列可填部门名（不区分大小写）或部门 ID。空时使用左侧选中的部门作为默认。')}</p>
            <p>{t('orgchart.import.hint3', '已存在 email 会更新（DisplayName / Department / Role 覆盖；SSO subject 不动），不存在则新建。')}</p>
          </div>

          <textarea
            value={content}
            onChange={(e) => setContent(e.target.value)}
            placeholder={`email,display_name,department,role\nalice@acme.com,Alice Smith,Engineering,team_lead\nbob@acme.com,Bob,Sales,employee`}
            className="w-full h-48 px-2 py-1.5 rounded border border-border bg-background text-xs font-mono resize-y"
          />

          {defaultDeptId && (
            <div className="text-[11px] text-muted-foreground">
              {t('orgchart.import.usingDefault', '默认部门 (列空时使用): ')}<code className="font-mono">{defaultDeptId}</code>
            </div>
          )}

          {result && (
            <div className="rounded border border-border bg-background p-3 space-y-2">
              <div className="grid grid-cols-3 gap-2 text-xs">
                <Stat label={t('orgchart.import.created', '新建')} value={result.created} accent="text-emerald-400" />
                <Stat label={t('orgchart.import.updated', '更新')} value={result.updated} accent="text-blue-400" />
                <Stat label={t('orgchart.import.errors', '失败')} value={result.errorRows.length} accent={result.errorRows.length > 0 ? 'text-red-400' : 'text-muted-foreground'} />
              </div>
              {result.errorRows.length > 0 && (
                <div className="text-[11px] space-y-1">
                  <div className="text-red-400 font-medium">{t('orgchart.import.errorList', '失败的行')}:</div>
                  <ul className="space-y-0.5 max-h-32 overflow-auto">
                    {result.errorRows.map((er, i) => (
                      <li key={i} className="font-mono">
                        line {er.lineNumber}: <span className="text-foreground">{er.email}</span> — <span className="text-red-400">{er.reason}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}
        </div>

        <footer className="p-3 border-t border-border flex justify-end gap-2">
          <button
            onClick={onClose}
            className="px-3 py-1.5 rounded border border-border text-xs hover:bg-muted"
          >
            {t('common.dismiss', '关闭')}
          </button>
          <button
            onClick={handleImport}
            disabled={busy || !content.trim()}
            className="px-3 py-1.5 rounded bg-primary text-primary-foreground text-xs disabled:opacity-50 flex items-center gap-1"
          >
            {busy ? <RefreshCw className="h-3.5 w-3.5 animate-spin" /> : <Upload className="h-3.5 w-3.5" />}
            {t('orgchart.import.run', '导入')}
          </button>
        </footer>
      </div>
    </div>
  )
}

function Stat({ label, value, accent }: { label: string; value: number; accent: string }) {
  return (
    <div>
      <div className="text-[10px] uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className={`text-xl font-semibold ${accent}`}>{value}</div>
    </div>
  )
}

function EmployeeRow({
  employee, onRoleChange, onDeactivate,
}: {
  employee: Employee
  onRoleChange: (role: OrgRole) => void
  onDeactivate: () => void
}) {
  return (
    <tr className="border-t border-border/50 hover:bg-muted/20">
      <td className="px-3 py-2 font-mono">{employee.email}</td>
      <td className="px-3 py-2">{employee.displayName || '—'}</td>
      <td className="px-3 py-2">
        <select
          value={employee.role}
          onChange={(e) => onRoleChange(e.target.value as OrgRole)}
          disabled={!employee.active}
          className="px-1.5 py-0.5 rounded border border-border bg-background text-[10px] font-mono"
        >
          {ROLES.map(r => <option key={r} value={r}>{r}</option>)}
        </select>
      </td>
      <td className="px-3 py-2 text-[10px] font-mono text-muted-foreground truncate max-w-[14ch]" title={employee.ssoSubject}>
        {employee.ssoSubject ? '🔗 ' + employee.ssoSubject.slice(0, 8) + '…' : '—'}
      </td>
      <td className="px-3 py-2">
        {employee.active ? (
          <span className="px-1.5 py-0.5 rounded bg-emerald-500/15 text-emerald-400 text-[10px]">active</span>
        ) : (
          <span className="px-1.5 py-0.5 rounded bg-muted text-muted-foreground text-[10px]">inactive</span>
        )}
      </td>
      <td className="px-3 py-2 text-right">
        {employee.active && (
          <button
            onClick={onDeactivate}
            title="deactivate"
            className="text-muted-foreground hover:text-red-400 p-1"
          >
            <UserX className="h-3.5 w-3.5" />
          </button>
        )}
      </td>
    </tr>
  )
}
