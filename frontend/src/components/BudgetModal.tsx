import { useEffect, useState, useCallback } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import {
  Wallet, X, Loader2, AlertTriangle, CheckCircle2, RotateCcw, Save,
  ShieldCheck, TrendingUp,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useToastStore } from '../stores/toastStore'
import {
  BudgetGetConfig, BudgetSetConfig, BudgetGetStatus, BudgetResetSession,
} from '../../wailsjs/go/main/App'
import type { budget } from '../../wailsjs/go/models'

interface BudgetModalProps {
  open: boolean
  onClose: () => void
}

const REFRESH_MS = 5_000

// Quick-pick presets so users don't have to know token-budget math.
// Numbers picked to map roughly to common Anthropic / OpenAI spend
// ceilings — 100K tokens ≈ $0.50-2 depending on model + cache.
const DAILY_PRESETS = [
  { label: '50K', value: 50_000 },
  { label: '200K', value: 200_000 },
  { label: '1M', value: 1_000_000 },
  { label: '5M', value: 5_000_000 },
]

const SESSION_PRESETS = [
  { label: '20K', value: 20_000 },
  { label: '100K', value: 100_000 },
  { label: '500K', value: 500_000 },
]

export function BudgetModal({ open, onClose }: BudgetModalProps) {
  const { i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh') ?? true
  const toast = useToastStore((s) => s.addToast)

  const [cfg, setCfg] = useState<budget.Config | null>(null)
  const [status, setStatus] = useState<budget.Status | null>(null)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [resetting, setResetting] = useState(false)
  const [dirty, setDirty] = useState(false)

  const refresh = useCallback(async () => {
    try {
      const [c, s] = await Promise.all([BudgetGetConfig(), BudgetGetStatus()])
      setCfg((prev) => (dirty && prev ? prev : c))
      setStatus(s)
    } catch (e) {
      toast('error', String(e))
    }
  }, [toast, dirty])

  useEffect(() => {
    if (open) {
      setLoading(true)
      refresh().finally(() => setLoading(false))
      const id = setInterval(refresh, REFRESH_MS)
      return () => clearInterval(id)
    }
  }, [open, refresh])

  const updateCfg = (patch: Partial<budget.Config>) => {
    setCfg((prev) => (prev ? { ...prev, ...patch } : prev))
    setDirty(true)
  }

  const save = async () => {
    if (!cfg) return
    setSaving(true)
    try {
      await BudgetSetConfig(cfg)
      setDirty(false)
      toast('success', isZh ? 'Budget Wall 配置已保存' : 'Budget Wall config saved')
      await refresh()
    } catch (e) {
      toast('error', String(e))
    } finally {
      setSaving(false)
    }
  }

  const resetSession = async () => {
    if (!confirm(isZh
      ? '确认重置 session 计数？已用 token 会归零，daily 计数不受影响。'
      : 'Reset the session counter? Tokens used in this session zero out; daily counter is unaffected.')) return
    setResetting(true)
    try {
      await BudgetResetSession()
      toast('success', isZh ? '本次 session 已重置' : 'Session reset')
      await refresh()
    } catch (e) {
      toast('error', String(e))
    } finally {
      setResetting(false)
    }
  }

  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) onClose() }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/30 backdrop-blur-sm z-50 animate-in fade-in-0" />
        <Dialog.Content
          className="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-full max-w-xl max-h-[88vh] flex flex-col bg-card border border-border rounded-xl shadow-2xl z-50 animate-in fade-in-0 zoom-in-95"
          aria-describedby={undefined}
        >
          <div className="flex items-center justify-between px-4 py-3 border-b border-border">
            <Dialog.Title className="flex items-center gap-2 text-sm font-semibold">
              <Wallet className="h-4 w-4 text-primary" />
              <span>{isZh ? 'Budget Wall 花费上限' : 'Budget Wall — Active spend cap'}</span>
            </Dialog.Title>
            <button onClick={onClose} className="h-7 w-7 inline-flex items-center justify-center rounded hover:bg-muted text-muted-foreground">
              <X className="h-4 w-4" />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {loading && !cfg && (
              <div className="flex items-center justify-center py-12">
                <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
              </div>
            )}

            {cfg && status && (
              <>
                <div className={cn(
                  'rounded-md border p-3 flex items-start gap-3',
                  cfg.enabled ? 'border-emerald-500/30 bg-emerald-500/10' : 'border-amber-500/30 bg-amber-500/10',
                )}>
                  {cfg.enabled
                    ? <ShieldCheck className="h-5 w-5 text-emerald-400 shrink-0 mt-0.5" />
                    : <AlertTriangle className="h-5 w-5 text-amber-400 shrink-0 mt-0.5" />}
                  <div className="flex-1 min-w-0">
                    <div className={cn('text-sm font-semibold',
                      cfg.enabled ? 'text-emerald-300' : 'text-amber-300')}>
                      {cfg.enabled
                        ? (isZh ? 'Budget Wall 已启用' : 'Budget Wall is active')
                        : (isZh ? 'Budget Wall 未启用' : 'Budget Wall is off')}
                    </div>
                    <p className="text-[11px] text-muted-foreground mt-1 leading-relaxed">
                      {cfg.enabled
                        ? (isZh
                          ? '当 token 用量越过上限，Switch 网关会立刻返回 429 拒绝后续请求，直到你提高上限或重置 session。'
                          : 'When token usage crosses the cap, the Switch gateway returns 429 to block further requests until you raise the limit or reset the session.')
                        : (isZh
                          ? '打开下方开关并设置 daily / session 上限——能阻止 Claude 一晚烧 $1,600 的事故。'
                          : 'Toggle below and set daily / session caps — defends against the $1,600 overnight burn pattern.')}
                    </p>
                  </div>
                  <button
                    onClick={() => updateCfg({ enabled: !cfg.enabled })}
                    className={cn(
                      'shrink-0 px-3 py-1 rounded-md text-xs font-medium transition-colors',
                      cfg.enabled
                        ? 'bg-red-600 hover:bg-red-500 text-white'
                        : 'bg-emerald-600 hover:bg-emerald-500 text-white',
                    )}
                  >
                    {cfg.enabled
                      ? (isZh ? '关闭' : 'Disable')
                      : (isZh ? '启用' : 'Enable')}
                  </button>
                </div>

                {/* Live gauges */}
                <div className="space-y-3">
                  <h3 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                    {isZh ? '实时用量' : 'Live usage'}
                  </h3>
                  <Gauge
                    isZh={isZh}
                    titleZh="今日总用量" titleEn="Today"
                    used={status.dailyUsed}
                    limit={status.dailyTokens}
                    pct={status.dailyPct}
                    hit={status.hitDaily}
                    warn={status.warnDaily}
                  />
                  <Gauge
                    isZh={isZh}
                    titleZh="本次 session" titleEn="This session"
                    used={status.sessionUsed}
                    limit={status.sessionTokens}
                    pct={status.sessionPct}
                    hit={status.hitSession}
                    warn={status.warnSession}
                  />
                </div>

                {/* Limits */}
                <div className="space-y-3">
                  <h3 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                    {isZh ? '上限设置' : 'Limits'}
                  </h3>

                  <LimitField
                    isZh={isZh}
                    labelZh="每日 token 上限" labelEn="Daily token cap"
                    helperZh="0 = 不限。一天累计超过即拦截。" helperEn="0 = unlimited. Resets at local midnight."
                    value={cfg.dailyTokens}
                    presets={DAILY_PRESETS}
                    onChange={(v) => updateCfg({ dailyTokens: v })}
                  />
                  <LimitField
                    isZh={isZh}
                    labelZh="单 session token 上限" labelEn="Session token cap"
                    helperZh="0 = 不限。可手动重置归零。" helperEn="0 = unlimited. Reset manually below."
                    value={cfg.sessionTokens}
                    presets={SESSION_PRESETS}
                    onChange={(v) => updateCfg({ sessionTokens: v })}
                  />

                  <div className="space-y-1">
                    <label className="block text-xs text-muted-foreground">
                      {isZh ? '软警告阈值（%）' : 'Soft-warn threshold (%)'}
                      <span className="ml-1 text-[10px] text-muted-foreground/60">
                        {isZh ? '到达后弹通知但不拦截' : 'Notifies but does not block'}
                      </span>
                    </label>
                    <input
                      type="number"
                      min={0} max={100}
                      value={cfg.softWarnPct}
                      onChange={(e) => updateCfg({ softWarnPct: parseInt(e.target.value) || 0 })}
                      className="w-full px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary tabular-nums"
                    />
                  </div>
                </div>

                {/* Actions */}
                <div className="flex items-center justify-between gap-2 pt-2 border-t border-border">
                  <button
                    onClick={resetSession}
                    disabled={resetting}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md border border-border text-xs hover:bg-muted disabled:opacity-50"
                  >
                    {resetting ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RotateCcw className="h-3.5 w-3.5" />}
                    {isZh ? '重置 session' : 'Reset session'}
                  </button>
                  <button
                    onClick={save}
                    disabled={saving || !dirty}
                    className={cn(
                      'inline-flex items-center gap-1.5 px-4 py-1.5 rounded-md text-xs font-medium transition-colors',
                      dirty
                        ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                        : 'bg-muted text-muted-foreground cursor-not-allowed',
                    )}
                  >
                    {saving ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Save className="h-3.5 w-3.5" />}
                    {isZh ? '保存' : 'Save'}
                    {dirty && <span className="h-1.5 w-1.5 rounded-full bg-amber-400" />}
                  </button>
                </div>

                <div className="rounded-md border border-border bg-muted/20 p-3 text-[11px] text-muted-foreground leading-relaxed">
                  <p className="font-medium text-foreground/80 mb-1 flex items-center gap-1.5">
                    <TrendingUp className="h-3 w-3" />
                    {isZh ? '为什么需要这个？' : 'Why this matters'}
                  </p>
                  <ul className="space-y-1 list-disc pl-4">
                    <li>{isZh ? 'r/ClaudeCode 大量"$1,600 一晚账单"帖子' : 'r/ClaudeCode is full of "$1,600 overnight bill" reports'}</li>
                    <li>{isZh ? 'Claude Code 比 Codex 平均多用 4× tokens；缓存 bug 会再放大 10×' : 'Claude Code burns ~4× tokens of Codex; the March 2026 caching bug multiplied bills 10×'}</li>
                    <li>{isZh ? '此功能在网关层硬切断，比"事后看仪表盘"早一步' : 'Switch hard-cuts at the gateway layer — earlier than any post-hoc dashboard can'}</li>
                  </ul>
                </div>
              </>
            )}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

function Gauge({
  isZh, titleZh, titleEn, used, limit, pct, hit, warn,
}: {
  isZh: boolean; titleZh: string; titleEn: string
  used: number; limit: number; pct: number; hit: boolean; warn: boolean
}) {
  const noLimit = !limit || limit <= 0
  const barColor = hit
    ? 'bg-red-500'
    : warn
      ? 'bg-amber-500'
      : 'bg-emerald-500'

  return (
    <div className="rounded-md border border-border/60 bg-muted/20 p-3">
      <div className="flex items-center justify-between text-xs mb-1.5">
        <span className="font-medium">{isZh ? titleZh : titleEn}</span>
        <span className="tabular-nums text-muted-foreground">
          {fmtTokens(used)}{!noLimit && <> <span className="opacity-50">/</span> {fmtTokens(limit)}</>}
          {!noLimit && <span className="ml-2 text-foreground tabular-nums">{pct}%</span>}
        </span>
      </div>
      {noLimit ? (
        <div className="text-[10px] text-muted-foreground/60">
          {isZh ? '未设置上限' : 'No cap set'}
        </div>
      ) : (
        <div className="h-1.5 bg-muted rounded-full overflow-hidden">
          <div className={cn('h-full transition-all', barColor)} style={{ width: `${Math.min(100, pct)}%` }} />
        </div>
      )}
      {hit && (
        <div className="mt-1.5 flex items-center gap-1 text-[11px] text-red-400">
          <AlertTriangle className="h-3 w-3" />
          {isZh ? '已超上限：所有新请求会被拦截' : 'Cap reached — new requests blocked'}
        </div>
      )}
      {warn && !hit && (
        <div className="mt-1.5 flex items-center gap-1 text-[11px] text-amber-400">
          <CheckCircle2 className="h-3 w-3" />
          {isZh ? '接近上限，请留意' : 'Approaching the cap'}
        </div>
      )}
    </div>
  )
}

function LimitField({
  isZh, labelZh, labelEn, helperZh, helperEn, value, presets, onChange,
}: {
  isZh: boolean; labelZh: string; labelEn: string
  helperZh: string; helperEn: string
  value: number
  presets: { label: string; value: number }[]
  onChange: (v: number) => void
}) {
  return (
    <div className="space-y-1">
      <label className="block text-xs text-muted-foreground">
        {isZh ? labelZh : labelEn}
        <span className="ml-1 text-[10px] text-muted-foreground/60">{isZh ? helperZh : helperEn}</span>
      </label>
      <div className="flex gap-1">
        <input
          type="number"
          min={0}
          step={1000}
          value={value}
          onChange={(e) => onChange(parseInt(e.target.value) || 0)}
          className="flex-1 px-2 py-1.5 text-xs bg-muted/30 border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary tabular-nums font-mono"
        />
        {presets.map((p) => (
          <button
            key={p.value}
            type="button"
            onClick={() => onChange(p.value)}
            className={cn(
              'px-2 py-1 rounded text-[10px] font-medium border transition-colors',
              value === p.value
                ? 'border-primary text-primary bg-primary/10'
                : 'border-border text-muted-foreground hover:bg-muted hover:text-foreground',
            )}
          >
            {p.label}
          </button>
        ))}
        <button
          type="button"
          onClick={() => onChange(0)}
          className={cn(
            'px-2 py-1 rounded text-[10px] font-medium border transition-colors',
            value === 0
              ? 'border-primary text-primary bg-primary/10'
              : 'border-border text-muted-foreground hover:bg-muted hover:text-foreground',
          )}
        >
          {isZh ? '不限' : '∞'}
        </button>
      </div>
    </div>
  )
}

function fmtTokens(n: number): string {
  if (n < 1000) return n.toLocaleString()
  if (n < 1_000_000) return (n / 1000).toFixed(n < 10_000 ? 1 : 0) + 'K'
  return (n / 1_000_000).toFixed(n < 10_000_000 ? 1 : 0) + 'M'
}
