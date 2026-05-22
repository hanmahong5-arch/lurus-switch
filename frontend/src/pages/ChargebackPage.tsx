import { useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Coins, RefreshCw, Download, Building2, User, AlertTriangle } from 'lucide-react'
import { useChargebackStore, type ChargebackRow } from '../stores/chargebackStore'
import { useConfigStore } from '../stores/configStore'
import { Button, Card } from '../components/ui'

const DAY = 24 * 60 * 60 * 1000

export function ChargebackPage() {
  const { t } = useTranslation()
  const {
    fromMs, toMs, view, report, loading, error,
    setRange, setView, load,
  } = useChargebackStore()

  useEffect(() => { void load() }, [load])

  const rows = view === 'department' ? (report?.byDepartment ?? []) : (report?.byEmployee ?? [])
  const totals = useMemo(() => rows.reduce((acc, r) => ({
    calls: acc.calls + r.totalCalls,
    tokens: acc.tokens + r.tokensIn + r.tokensOut,
  }), { calls: 0, tokens: 0 }), [rows])

  const handleQuickRange = (days: number) => {
    const to = Date.now()
    setRange(to - days * DAY, to)
  }

  const handleExportCSV = () => {
    const headers = view === 'department'
      ? ['department', 'cost_center', 'unique_employees', 'total_calls', 'tokens_in', 'tokens_out']
      : ['email', 'display_name', 'department', 'cost_center', 'total_calls', 'tokens_in', 'tokens_out']
    const lines = [headers.join(',')]
    for (const r of rows) {
      const cells = view === 'department'
        ? [r.deptName ?? '', r.costCenter ?? '', String(r.uniqueEmployees ?? 0), String(r.totalCalls), String(r.tokensIn), String(r.tokensOut)]
        : [r.email ?? '', r.displayName ?? '', r.deptName ?? '', r.costCenter ?? '', String(r.totalCalls), String(r.tokensIn), String(r.tokensOut)]
      lines.push(cells.map(escapeCSV).join(','))
    }
    const blob = new Blob([lines.join('\n')], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `chargeback_${view}_${dateStr(fromMs)}_to_${dateStr(toMs)}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="h-full overflow-auto p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-xl font-semibold flex items-center gap-2">
            <Coins className="h-5 w-5 text-primary" />
            {t('chargeback.title', '成本归集 — Chargeback')}
          </h1>
          <p className="text-xs text-muted-foreground mt-1">
            {t('chargeback.subtitle', '按部门和员工汇总 token 用量。要让数据落到对的桶里，先在 Connected Apps 把每个 app 绑定到员工。')}
          </p>
        </div>
        <div className="flex items-center gap-2">
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
          <Button
            size="sm"
            onClick={handleExportCSV}
            disabled={rows.length === 0}
            icon={<Download className="h-3.5 w-3.5" />}
          >
            {t('chargeback.exportCsv', '导出 CSV')}
          </Button>
        </div>
      </div>

      {error && (
        <Card variant="default" className="mb-3 p-2 border-red-500/30 bg-red-500/10 text-red-400 text-xs flex items-center gap-2 font-mono">
          <AlertTriangle className="h-3.5 w-3.5" />
          ▸ {error}
        </Card>
      )}

      {/* Range + view controls */}
      <Card variant="default" className="p-3 mb-4 grid grid-cols-1 md:grid-cols-3 gap-3 items-end">
        <div>
          <label className="block text-[10px] uppercase tracking-wider text-muted-foreground mb-1">
            {t('chargeback.from', '起')}
          </label>
          <input
            type="date"
            value={dateInput(fromMs)}
            onChange={(e) => setRange(parseDateInput(e.target.value), toMs)}
            className="w-full px-2 py-1 rounded border border-border bg-background text-xs"
          />
        </div>
        <div>
          <label className="block text-[10px] uppercase tracking-wider text-muted-foreground mb-1">
            {t('chargeback.to', '止')}
          </label>
          <input
            type="date"
            value={dateInput(toMs)}
            onChange={(e) => setRange(fromMs, parseDateInput(e.target.value))}
            className="w-full px-2 py-1 rounded border border-border bg-background text-xs"
          />
        </div>
        <div className="flex flex-wrap gap-1">
          <QuickRange label="今" onClick={() => handleQuickRange(1)} />
          <QuickRange label="7d" onClick={() => handleQuickRange(7)} />
          <QuickRange label="30d" onClick={() => handleQuickRange(30)} />
          <QuickRange label="90d" onClick={() => handleQuickRange(90)} />
        </div>
      </Card>

      {/* Tabs */}
      <div className="flex items-center gap-1 mb-3 border-b border-border">
        <TabButton
          active={view === 'department'}
          onClick={() => setView('department')}
          icon={<Building2 className="h-3.5 w-3.5" />}
          label={t('chargeback.byDept', '按部门')}
          count={report?.byDepartment?.length ?? 0}
        />
        <TabButton
          active={view === 'employee'}
          onClick={() => setView('employee')}
          icon={<User className="h-3.5 w-3.5" />}
          label={t('chargeback.byEmployee', '按员工')}
          count={report?.byEmployee?.length ?? 0}
        />
        <div className="ml-auto text-[11px] text-muted-foreground pb-2">
          {t('chargeback.totals', '合计')}: {totals.calls.toLocaleString()} {t('chargeback.calls', 'calls')} · {totals.tokens.toLocaleString()} {t('chargeback.tokens', 'tokens')}
        </div>
      </div>

      {/* Table */}
      <Card as="section" variant="default" className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground bg-card-recessed">
              <tr>
                {view === 'department' ? (
                  <>
                    <th className="text-left px-3 py-2">{t('chargeback.col.dept', '部门')}</th>
                    <th className="text-left px-3 py-2">{t('chargeback.col.cc', '成本中心')}</th>
                    <th className="text-right px-3 py-2">{t('chargeback.col.headcount', '人数')}</th>
                  </>
                ) : (
                  <>
                    <th className="text-left px-3 py-2">{t('chargeback.col.employee', '员工')}</th>
                    <th className="text-left px-3 py-2">{t('chargeback.col.dept', '部门')}</th>
                    <th className="text-left px-3 py-2">{t('chargeback.col.cc', '成本中心')}</th>
                  </>
                )}
                <th className="text-right px-3 py-2">{t('chargeback.col.calls', '调用数')}</th>
                <th className="text-right px-3 py-2">{t('chargeback.col.in', 'token in')}</th>
                <th className="text-right px-3 py-2">{t('chargeback.col.out', 'token out')}</th>
                <th className="text-right px-3 py-2">{t('chargeback.col.total', '合计')}</th>
              </tr>
            </thead>
            <tbody>
              {rows.length === 0 && !loading && (
                <tr><td colSpan={7} className="px-3 py-8 text-center">
                  <div className="text-muted-foreground mb-3">
                    {t('chargeback.empty', '该时间段无记录。先把 app 绑定到员工 / 部门，再让流量过本机网关。')}
                  </div>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => {
                      const cs = useConfigStore.getState()
                      cs.setActiveTool('gateway')
                      cs.setSubTab('gateway', 'apps')
                    }}
                    icon={<User className="h-3.5 w-3.5" />}
                    className="border-primary/40 text-primary hover:bg-primary/10"
                  >
                    {t('chargeback.gotoBinding', '去 Connected Apps 绑定归属 →')}
                  </Button>
                </td></tr>
              )}
              {rows.map((r, i) => <Row key={i} r={r} view={view} />)}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  )
}

function Row({ r, view }: { r: ChargebackRow; view: 'department' | 'employee' }) {
  const total = r.tokensIn + r.tokensOut
  const isUnattributed = view === 'department' ? !r.deptId : !r.employeeId
  return (
    <tr className={`border-t border-border/50 hover:bg-muted/20 ${isUnattributed ? 'text-muted-foreground italic' : ''}`}>
      {view === 'department' ? (
        <>
          <td className="px-3 py-2">{r.deptName || '—'}</td>
          <td className="px-3 py-2 font-mono text-[11px]">{r.costCenter || '—'}</td>
          <td className="px-3 py-2 text-right">{r.uniqueEmployees ?? 0}</td>
        </>
      ) : (
        <>
          <td className="px-3 py-2">
            <div className="font-medium">{r.displayName || r.email || '—'}</div>
            {r.email && r.displayName && (
              <div className="text-[10px] text-muted-foreground/70 font-mono">{r.email}</div>
            )}
          </td>
          <td className="px-3 py-2">{r.deptName || '—'}</td>
          <td className="px-3 py-2 font-mono text-[11px]">{r.costCenter || '—'}</td>
        </>
      )}
      <td className="px-3 py-2 text-right tabular-nums">{r.totalCalls.toLocaleString()}</td>
      <td className="px-3 py-2 text-right tabular-nums">{r.tokensIn.toLocaleString()}</td>
      <td className="px-3 py-2 text-right tabular-nums">{r.tokensOut.toLocaleString()}</td>
      <td className="px-3 py-2 text-right tabular-nums font-semibold">{total.toLocaleString()}</td>
    </tr>
  )
}

function TabButton({ active, onClick, icon, label, count }: { active: boolean; onClick: () => void; icon: React.ReactNode; label: string; count: number }) {
  return (
    <button
      onClick={onClick}
      className={`px-3 py-1.5 -mb-px border-b-2 flex items-center gap-1.5 transition-all duration-150 ${
        active ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'
      }`}
    >
      {icon}
      <span className={active ? 'font-mono text-[11px] tracking-[0.12em]' : 'text-xs font-medium'}>
        {active ? `[ ${label.toUpperCase()} ]` : label}
      </span>
      <span className="font-mono text-[10px] text-muted-foreground/70 tabular-nums">({count})</span>
    </button>
  )
}

function QuickRange({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="px-2 py-1 rounded border border-border text-[11px] hover:bg-muted"
    >
      {label}
    </button>
  )
}

function dateInput(ms: number): string {
  const d = new Date(ms)
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd}`
}

function parseDateInput(s: string): number {
  return new Date(s + 'T00:00:00').getTime()
}

function dateStr(ms: number): string {
  return dateInput(ms).replace(/-/g, '')
}

function escapeCSV(s: string): string {
  if (/[",\n]/.test(s)) {
    return '"' + s.replace(/"/g, '""') + '"'
  }
  return s
}
