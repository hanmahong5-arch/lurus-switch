import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Wallet,
  RefreshCw,
  AlertCircle,
  Send,
  Copy,
  Check,
  ExternalLink,
  TrendingUp,
  TrendingDown,
} from 'lucide-react'
import { Button, Card, KpiCard, Modal } from '../components/ui'
import { useConfigStore } from '../stores/configStore'
import { makeWalletSource, type WalletSource } from '../lib/walletSource'
import { Pagination } from '../components/gateway/Pagination'
import { formatLocal } from '../lib/formatTime'
import type { admin } from '../../wailsjs/go/models'
import { cn } from '../lib/utils'

const PER_PAGE = 20

// Transaction types we surface badges for. Hub forwards platform's enum
// verbatim so the mapping stays in this one file rather than in i18n —
// labels are short and tone-driven, not free-form copy.
const TX_TONE: Record<string, 'credit' | 'debit' | 'neutral'> = {
  topup: 'credit',
  credit: 'credit',
  refund: 'credit',
  grant: 'credit',
  debit: 'debit',
  spend: 'debit',
  consume: 'debit',
  product_purchase: 'debit',
  pre_authorize: 'neutral',
  settle_pre_auth: 'debit',
  release_pre_auth: 'credit',
}

function txTone(type: string): 'credit' | 'debit' | 'neutral' {
  return TX_TONE[type.toLowerCase()] ?? 'neutral'
}

function formatAmount(amount: number, tone: 'credit' | 'debit' | 'neutral'): string {
  const abs = Math.abs(amount).toFixed(2)
  if (tone === 'credit') return `+${abs}`
  if (tone === 'debit') return `-${abs}`
  return abs
}

// netMonth aggregates the first PER_PAGE rows on the current page as a
// rough "this period" indicator. The accurate per-month rollup would require
// a server-side aggregate endpoint — Wave 5 v1 ships with a transparent
// proxy of what's loaded, so the UI doesn't lie about being authoritative.
function netByTone(rows: admin.WalletTransaction[], tone: 'credit' | 'debit'): number {
  return rows
    .filter((r) => txTone(r.type) === tone)
    .reduce((sum, r) => sum + Math.abs(r.amount), 0)
}

export function GatewayWalletPage() {
  const { t } = useTranslation()
  const appMode = useConfigStore((s) => s.appMode)
  const isReseller = appMode === 'reseller'

  const [info, setInfo] = useState<admin.WalletInfo | null>(null)
  const [rows, setRows] = useState<admin.WalletTransaction[]>([])
  const [page, setPage] = useState(0)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showWithdraw, setShowWithdraw] = useState(false)
  const [copiedKey, setCopiedKey] = useState<string | null>(null)

  const source: WalletSource | null = useMemo(() => {
    if (!isReseller) return null
    return makeWalletSource('hub')
  }, [isReseller])

  const load = useCallback(
    async (p = page) => {
      if (!source) return
      setLoading(true)
      setError(null)
      try {
        const [walletInfo, txPage] = await Promise.all([
          source.getInfo(),
          source.listTransactions(p + 1, PER_PAGE),
        ])
        setInfo(walletInfo)
        setRows(txPage.items ?? [])
        setTotal(txPage.total ?? 0)
      } catch (e) {
        setError(String(e instanceof Error ? e.message : e))
      } finally {
        setLoading(false)
      }
    },
    [source, page],
  )

  useEffect(() => {
    load(0)
    setPage(0)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [source])

  const onPageChange = (p: number) => {
    setPage(p)
    load(p)
  }

  const copyToClipboard = async (text: string, key: string) => {
    await navigator.clipboard.writeText(text)
    setCopiedKey(key)
    setTimeout(() => setCopiedKey(null), 1500)
  }

  const periodCredit = netByTone(rows, 'credit')
  const periodDebit = netByTone(rows, 'debit')

  if (!source) {
    return (
      <div className="flex flex-col h-full items-center justify-center text-muted-foreground gap-2">
        <AlertCircle className="h-8 w-8" />
        <p>{t('wallet.resellerOnly', '钱包功能仅在 Reseller 模式可用')}</p>
      </div>
    )
  }

  const isPlatform = info?.source === 'platform'

  return (
    <div className="h-full overflow-y-auto p-4 md:p-6 space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div className="flex items-center gap-2">
          <Wallet className="h-5 w-5 text-primary" />
          <h2 className="text-lg font-semibold">{t('wallet.title', '经销商钱包')}</h2>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => load(page)} disabled={loading}>
            <RefreshCw className={cn('h-3.5 w-3.5 mr-1', loading && 'animate-spin')} />
            {t('common.refresh', '刷新')}
          </Button>
          <Button variant="primary" size="sm" onClick={() => setShowWithdraw(true)} disabled={!info || !isPlatform}>
            <Send className="h-3.5 w-3.5 mr-1" />
            {t('wallet.withdrawRequest', '申请提现')}
          </Button>
        </div>
      </div>

      {/* Source warning */}
      {info && !isPlatform && (
        <Card variant="elevated" className="p-3 border-amber-500/40 bg-amber-500/5">
          <div className="flex items-start gap-2 text-sm">
            <AlertCircle className="h-4 w-4 text-amber-500 mt-0.5 shrink-0" />
            <div>
              <p className="font-medium">{t('wallet.notLinkedTitle', '当前账号未绑定 lurus-platform 钱包')}</p>
              <p className="text-xs text-muted-foreground mt-1">
                {t('wallet.notLinkedDesc', '显示的是 Hub 本地配额，非平台真实钱包。请联系平台管理员完成账号绑定。')}
              </p>
            </div>
          </div>
        </Card>
      )}

      {/* KPI grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <KpiCard
          label={t('wallet.balance', '账户余额')}
          value={`¥ ${(info?.balance ?? 0).toFixed(2)}`}
          icon={Wallet}
          accent
        />
        <KpiCard
          label={t('wallet.available', '可用余额')}
          value={`¥ ${(info?.available ?? 0).toFixed(2)}`}
        />
        <KpiCard
          label={t('wallet.periodCredit', '本页入账')}
          value={`+¥ ${periodCredit.toFixed(2)}`}
          icon={TrendingUp}
        />
        <KpiCard
          label={t('wallet.periodDebit', '本页出账')}
          value={`-¥ ${periodDebit.toFixed(2)}`}
          icon={TrendingDown}
        />
      </div>

      {/* Error */}
      {error && (
        <Card variant="elevated" className="p-3 border-red-500/40 bg-red-500/5">
          <div className="flex items-center gap-2 text-sm text-red-400">
            <AlertCircle className="h-4 w-4" />
            <span>{error}</span>
          </div>
        </Card>
      )}

      {/* Transactions table */}
      <Card variant="elevated" className="overflow-hidden">
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <h3 className="text-sm font-medium">{t('wallet.transactions', '交易明细')}</h3>
          <Pagination page={page} total={total} perPage={PER_PAGE} onPageChange={onPageChange} />
        </div>

        {rows.length === 0 && !loading ? (
          <div className="px-4 py-12 text-center text-sm text-muted-foreground">
            {t('wallet.empty', '暂无交易记录')}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs text-muted-foreground border-b border-border">
                  <th className="px-4 py-2 font-medium">{t('wallet.col.time', '时间')}</th>
                  <th className="px-4 py-2 font-medium">{t('wallet.col.type', '类型')}</th>
                  <th className="px-4 py-2 font-medium text-right">{t('wallet.col.amount', '金额 (¥)')}</th>
                  <th className="px-4 py-2 font-medium text-right">{t('wallet.col.balance', '余额 (¥)')}</th>
                  <th className="px-4 py-2 font-medium">{t('wallet.col.product', '来源')}</th>
                  <th className="px-4 py-2 font-medium">{t('wallet.col.desc', '备注')}</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((r) => {
                  const tone = txTone(r.type)
                  return (
                    <tr key={r.id} className="border-b border-border/40 hover:bg-muted/30">
                      <td className="px-4 py-2 text-xs text-muted-foreground tabular-nums whitespace-nowrap">
                        {formatLocal(r.created_at)}
                      </td>
                      <td className="px-4 py-2">
                        <span
                          className={cn(
                            'inline-flex items-center px-2 py-0.5 rounded text-[10px] font-mono uppercase tracking-wider',
                            tone === 'credit' && 'bg-emerald-500/10 text-emerald-400',
                            tone === 'debit' && 'bg-red-500/10 text-red-400',
                            tone === 'neutral' && 'bg-muted text-muted-foreground',
                          )}
                        >
                          {r.type}
                        </span>
                      </td>
                      <td
                        className={cn(
                          'px-4 py-2 font-mono tabular-nums text-right',
                          tone === 'credit' && 'text-emerald-400',
                          tone === 'debit' && 'text-red-400',
                        )}
                      >
                        {formatAmount(r.amount, tone)}
                      </td>
                      <td className="px-4 py-2 font-mono tabular-nums text-right text-muted-foreground">
                        {r.balance_after.toFixed(2)}
                      </td>
                      <td className="px-4 py-2 text-xs text-muted-foreground">{r.product_id || '—'}</td>
                      <td className="px-4 py-2 text-xs truncate max-w-[280px]" title={r.description}>
                        {r.description || '—'}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      {/* Withdraw modal */}
      <Modal open={showWithdraw} onClose={() => setShowWithdraw(false)} title={t('wallet.withdrawTitle', '申请提现')}>
        <WithdrawTemplate
          info={info}
          copyToClipboard={copyToClipboard}
          copiedKey={copiedKey}
          topupUrl={info?.topup_url}
        />
      </Modal>
    </div>
  )
}

interface WithdrawTemplateProps {
  info: admin.WalletInfo | null
  copyToClipboard: (text: string, key: string) => void
  copiedKey: string | null
  topupUrl?: string
}

function WithdrawTemplate({ info, copyToClipboard, copiedKey, topupUrl }: WithdrawTemplateProps) {
  const { t } = useTranslation()
  const supportEmail = 'support@lurus.cn'
  const subject = t('wallet.mailSubject', '【经销商提现】Switch 钱包提现申请')
  const balance = info?.available ?? 0
  const body = t('wallet.mailBody', {
    defaultValue:
      '您好，\n\n我是 Lurus Switch 经销商，希望申请钱包提现。\n\n当前可用余额：¥ {{balance}}\n提现金额：¥ ___\n收款方式：（支付宝 / 微信 / 银行卡，请填写账号）\n\n谢谢。',
    balance: balance.toFixed(2),
  })
  const mailto = `mailto:${supportEmail}?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(body)}`

  return (
    <div className="space-y-4 text-sm">
      <div>
        <p className="text-muted-foreground">
          {t(
            'wallet.withdrawNote',
            'v1 阶段提现走客服邮件审批 — 复制下方模板发送至客服邮箱，或点击「打开邮件客户端」自动填充。',
          )}
        </p>
      </div>

      <div className="grid grid-cols-2 gap-3 p-3 bg-muted/40 rounded">
        <div>
          <p className="text-xs text-muted-foreground">{t('wallet.available', '可用余额')}</p>
          <p className="font-mono text-lg tabular-nums">¥ {balance.toFixed(2)}</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">{t('wallet.frozen', '冻结金额')}</p>
          <p className="font-mono text-lg tabular-nums">¥ {(info?.frozen ?? 0).toFixed(2)}</p>
        </div>
      </div>

      <div>
        <div className="flex items-center justify-between mb-1">
          <p className="text-xs text-muted-foreground">{t('wallet.supportEmail', '客服邮箱')}</p>
          <button
            onClick={() => copyToClipboard(supportEmail, 'email')}
            className="text-xs text-primary hover:underline flex items-center gap-1"
          >
            {copiedKey === 'email' ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
            {copiedKey === 'email' ? t('common.copied', '已复制') : t('common.copy', '复制')}
          </button>
        </div>
        <p className="font-mono text-sm bg-muted/40 px-3 py-2 rounded">{supportEmail}</p>
      </div>

      <div>
        <div className="flex items-center justify-between mb-1">
          <p className="text-xs text-muted-foreground">{t('wallet.mailTemplate', '邮件模板')}</p>
          <button
            onClick={() => copyToClipboard(body, 'body')}
            className="text-xs text-primary hover:underline flex items-center gap-1"
          >
            {copiedKey === 'body' ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
            {copiedKey === 'body' ? t('common.copied', '已复制') : t('common.copyTemplate', '复制模板')}
          </button>
        </div>
        <textarea
          readOnly
          className="w-full h-32 text-xs font-mono bg-muted/40 px-3 py-2 rounded resize-none"
          value={body}
        />
      </div>

      <div className="flex gap-2 justify-end">
        {topupUrl && (
          <a
            href={topupUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 px-3 py-1.5 rounded text-sm border border-border hover:bg-muted"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            {t('wallet.openTopup', '打开充值页')}
          </a>
        )}
        <a
          href={mailto}
          className="inline-flex items-center gap-1 px-3 py-1.5 rounded text-sm bg-primary text-primary-foreground hover:opacity-90"
        >
          <Send className="h-3.5 w-3.5" />
          {t('wallet.openMail', '打开邮件客户端')}
        </a>
      </div>
    </div>
  )
}
