import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, ServerCog, Check, AlertTriangle, ArrowRight, ArrowLeft, LogOut } from 'lucide-react'
import {
  ListResellerDeployKinds,
  TestHubConnection,
  ProvisionResellerHub,
  SetAppMode,
} from '../../wailsjs/go/main/App'
import { useConfigStore } from '../stores/configStore'
import { useDirtyGuard } from '../hooks/useDirtyGuard'

// The Wails-generated type for ListResellerDeployKinds is `main.resellerKindEntry`
// (unexported in Go → no clean export from models.ts). Mirror the shape locally.
interface DeployKindEntry {
  kind: string
  implemented: boolean
  labelZh: string
  labelEn: string
  descriptionZh: string
  descriptionEn: string
}

type Step = 'pick' | 'manual' | 'test' | 'done'

interface Props {
  onComplete: () => void
}

export function ResellerSetupWizard({ onComplete }: Props) {
  const { t, i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh') ?? true
  const setAppModeLocal = useConfigStore((s) => s.setAppMode)

  const [kinds, setKinds] = useState<DeployKindEntry[]>([])
  const [pickedKind, setPickedKind] = useState<string>('manual')
  const [step, setStep] = useState<Step>('pick')
  const [switchingMode, setSwitchingMode] = useState(false)

  // Manual entry form
  const [hubURL, setHubURL] = useState('')
  const [adminToken, setAdminToken] = useState('')
  const [tenantSlug, setTenantSlug] = useState('')
  const [displayName, setDisplayName] = useState('')

  // Connection test state
  const [testing, setTesting] = useState(false)
  const [testMessage, setTestMessage] = useState('')
  const [testError, setTestError] = useState('')

  // Provisioning state
  const [provisioning, setProvisioning] = useState(false)
  const [provisionError, setProvisionError] = useState('')

  // Mark wizard dirty if any field has input and we haven't reached 'done'.
  // Note: Reseller wizard is a gate (replaces the shell), so today the only
  // path that triggers the discard prompt is the Switch-mode button — but
  // the registration also future-proofs us for any in-shell back integration.
  const hasInput =
    hubURL.trim() !== '' ||
    adminToken.trim() !== '' ||
    tenantSlug.trim() !== '' ||
    displayName.trim() !== ''
  useDirtyGuard('reseller-setup-wizard', hasInput && step !== 'done')

  useEffect(() => {
    ListResellerDeployKinds()
      .then((rows) => setKinds(rows as unknown as DeployKindEntry[]))
      .catch(() => setKinds([
        { kind: 'manual', implemented: true, labelZh: '手动接入', labelEn: 'Manual', descriptionZh: '', descriptionEn: '' },
      ]))
  }, [])

  // ESC steps backwards in the flow (manual←test, pick←manual). On 'pick'
  // it does nothing; users wanting out should use the "Switch mode" link.
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      if (step === 'manual') setStep('pick')
      else if (step === 'test') setStep('manual')
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [step])

  // Escape hatch: drop back to Personal mode. The user can re-pick Reseller
  // later from Settings → Mode without losing any data they already entered
  // here (the form state is local, not persisted).
  const handleSwitchMode = async () => {
    if (switchingMode) return
    const ok = window.confirm(
      t(
        'reseller.setup.switchModeConfirm',
        '切换到 Personal 模式将放弃当前向导中未保存的输入，确定继续？',
      ),
    )
    if (!ok) return
    setSwitchingMode(true)
    try {
      await SetAppMode('personal')
      setAppModeLocal('personal')
      // Do NOT call onComplete — the parent's mode check will re-route on
      // the next render because appMode is no longer 'reseller'.
    } catch (e) {
      window.alert(String(e))
    } finally {
      setSwitchingMode(false)
    }
  }

  const labelOf = (k: DeployKindEntry) => isZh ? k.labelZh : k.labelEn
  const descOf = (k: DeployKindEntry) => isZh ? k.descriptionZh : k.descriptionEn

  const handlePickKind = (kind: string, implemented: boolean) => {
    setPickedKind(kind)
    if (!implemented) {
      // Stub providers — collect intent but route through manual entry.
      setStep('manual')
      return
    }
    setStep('manual')
  }

  const handleTestConnection = async () => {
    setTesting(true)
    setTestError('')
    setTestMessage('')
    try {
      const msg = await TestHubConnection(hubURL.trim(), adminToken.trim())
      setTestMessage(msg)
    } catch (e) {
      setTestError(String(e))
    } finally {
      setTesting(false)
    }
  }

  const handleProvision = async () => {
    setProvisioning(true)
    setProvisionError('')
    try {
      // Force kind=manual at the backend boundary — stub providers reject.
      await ProvisionResellerHub(
        'manual',
        displayName.trim(),
        hubURL.trim(),
        adminToken.trim(),
        tenantSlug.trim(),
      )
      setStep('done')
    } catch (e) {
      setProvisionError(String(e))
    } finally {
      setProvisioning(false)
    }
  }

  const canTest = hubURL.trim() !== '' && adminToken.trim() !== ''
  const canProvision = canTest && testMessage !== '' && !testError

  return (
    <div className="h-screen flex flex-col items-center justify-center bg-background text-foreground p-6">
      <div className="w-full max-w-2xl">
        <header className="flex items-start justify-between gap-3 mb-6">
          <div className="flex items-center gap-3">
            <ServerCog className="h-7 w-7 text-purple-400" />
            <div>
              <h1 className="text-2xl font-semibold">{t('reseller.setup.title', '配置 Reseller Hub')}</h1>
              <p className="text-sm text-muted-foreground">
                {t('reseller.setup.subtitle', '为你的经销商业务接入或部署一台 lurus-newhub')}
              </p>
            </div>
          </div>
          <button
            onClick={handleSwitchMode}
            disabled={switchingMode}
            className="text-xs text-muted-foreground hover:text-foreground inline-flex items-center gap-1 px-2 py-1 rounded hover:bg-muted disabled:opacity-50"
            title={t('reseller.setup.switchModeHint', '切回 Personal 模式（不影响已填写的连接信息）')}
          >
            {switchingMode ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <LogOut className="h-3.5 w-3.5" />
            )}
            {t('reseller.setup.switchMode', '切换模式')}
          </button>
        </header>

        {/* Step indicator */}
        <ol className="flex items-center gap-2 mb-8 text-xs text-muted-foreground">
          {(['pick', 'manual', 'test', 'done'] as Step[]).map((s, i) => (
            <li key={s} className="flex items-center gap-2">
              <span className={
                'h-6 w-6 inline-flex items-center justify-center rounded-full font-medium ' +
                (step === s
                  ? 'bg-purple-600 text-white'
                  : i < (['pick', 'manual', 'test', 'done'] as Step[]).indexOf(step)
                  ? 'bg-emerald-600/30 text-emerald-300'
                  : 'bg-muted text-muted-foreground')
              }>
                {i + 1}
              </span>
              <span className="capitalize hidden sm:inline">
                {t(`reseller.setup.step.${s}`, s)}
              </span>
              {i < 3 && <span className="mx-2 text-muted-foreground/40">/</span>}
            </li>
          ))}
        </ol>

        {/* Step 1: pick provider */}
        {step === 'pick' && (
          <section className="space-y-3">
            <h2 className="font-medium">{t('reseller.setup.pickKind', '选择部署方式')}</h2>
            <div className="grid gap-3">
              {kinds.map((k) => (
                <button
                  key={k.kind}
                  onClick={() => handlePickKind(k.kind, k.implemented)}
                  className={
                    'text-left rounded-lg border p-4 transition-colors ' +
                    (pickedKind === k.kind
                      ? 'border-purple-500 bg-purple-950/20'
                      : 'border-border hover:bg-muted/30')
                  }
                >
                  <div className="flex items-center justify-between">
                    <span className="font-medium">{labelOf(k)}</span>
                    {!k.implemented && (
                      <span className="text-[10px] uppercase tracking-wide rounded bg-amber-500/15 text-amber-300 px-1.5 py-0.5">
                        {t('reseller.setup.comingSoon', 'coming soon')}
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground mt-1">{descOf(k)}</p>
                </button>
              ))}
            </div>
          </section>
        )}

        {/* Step 2: manual entry */}
        {step === 'manual' && (
          <section className="space-y-3">
            <h2 className="font-medium">{t('reseller.setup.manualTitle', '填写 Hub 连接信息')}</h2>
            <p className="text-xs text-muted-foreground">
              {t('reseller.setup.manualHint', '在 lurus-newhub 控制台「我的账户 → API 令牌」页面创建一个 root 角色的访问令牌。')}
            </p>
            <div className="space-y-2 mt-3">
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('reseller.setup.displayName', '展示名（可选）')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm"
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                  placeholder="Acme Corp"
                />
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground">Hub URL *</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono"
                  value={hubURL}
                  onChange={(e) => setHubURL(e.target.value)}
                  placeholder="https://hub.acme.example"
                />
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('reseller.setup.adminToken', 'Admin Token *')}</span>
                <input
                  type="password"
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono"
                  value={adminToken}
                  onChange={(e) => setAdminToken(e.target.value)}
                  placeholder="••••••••"
                />
              </label>
              <label className="block text-sm">
                <span className="text-muted-foreground">{t('reseller.setup.tenantSlug', 'Tenant Slug（V2 多租户可选）')}</span>
                <input
                  className="mt-1 w-full rounded border border-border bg-background px-3 py-1.5 text-sm font-mono"
                  value={tenantSlug}
                  onChange={(e) => setTenantSlug(e.target.value)}
                  placeholder="acme"
                />
              </label>
            </div>

            <div className="flex justify-between pt-2">
              <button
                onClick={() => setStep('pick')}
                className="px-3 py-1.5 rounded border border-border text-sm hover:bg-muted inline-flex items-center gap-1"
              >
                <ArrowLeft className="h-4 w-4" />
                {t('common.back', '上一步')}
              </button>
              <button
                onClick={() => setStep('test')}
                disabled={!canTest}
                className="px-4 py-1.5 rounded bg-purple-600 hover:bg-purple-500 text-white text-sm disabled:opacity-40 inline-flex items-center gap-1"
              >
                {t('common.next', '下一步')}
                <ArrowRight className="h-4 w-4" />
              </button>
            </div>
          </section>
        )}

        {/* Step 3: test + save */}
        {step === 'test' && (
          <section className="space-y-3">
            <h2 className="font-medium">{t('reseller.setup.verifyTitle', '验证连接并保存')}</h2>
            <div className="rounded border border-border p-3 text-xs space-y-1 font-mono">
              <div><span className="text-muted-foreground">URL:</span> {hubURL}</div>
              <div><span className="text-muted-foreground">Token:</span> {'•'.repeat(Math.min(8, adminToken.length))}</div>
              {tenantSlug && <div><span className="text-muted-foreground">Tenant:</span> {tenantSlug}</div>}
              {displayName && <div><span className="text-muted-foreground">Name:</span> {displayName}</div>}
            </div>

            <div className="flex items-center gap-2">
              <button
                onClick={handleTestConnection}
                disabled={testing}
                className="px-3 py-1.5 rounded border border-border text-sm hover:bg-muted inline-flex items-center gap-2"
              >
                {testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
                {t('reseller.setup.testConnection', '测试连接')}
              </button>
              {testMessage && (
                <span className="text-xs text-emerald-400 flex items-center gap-1">
                  <Check className="h-3.5 w-3.5" />
                  {testMessage}
                </span>
              )}
              {testError && (
                <span className="text-xs text-red-400 flex items-center gap-1">
                  <AlertTriangle className="h-3.5 w-3.5" />
                  {testError}
                </span>
              )}
            </div>

            {provisionError && (
              <div className="text-xs text-red-400 bg-red-900/20 rounded px-3 py-2">
                {provisionError}
              </div>
            )}

            <div className="flex justify-between pt-2">
              <button
                onClick={() => setStep('manual')}
                className="px-3 py-1.5 rounded border border-border text-sm hover:bg-muted inline-flex items-center gap-1"
              >
                <ArrowLeft className="h-4 w-4" />
                {t('common.back', '上一步')}
              </button>
              <button
                onClick={handleProvision}
                disabled={!canProvision || provisioning}
                className="px-4 py-1.5 rounded bg-emerald-600 hover:bg-emerald-500 text-white text-sm disabled:opacity-40 inline-flex items-center gap-2"
              >
                {provisioning && <Loader2 className="h-4 w-4 animate-spin" />}
                {t('reseller.setup.save', '保存配置')}
              </button>
            </div>
          </section>
        )}

        {/* Step 4: done */}
        {step === 'done' && (
          <section className="space-y-4 text-center">
            <div className="flex justify-center">
              <div className="h-12 w-12 rounded-full bg-emerald-500/20 flex items-center justify-center">
                <Check className="h-7 w-7 text-emerald-400" />
              </div>
            </div>
            <div>
              <h2 className="text-lg font-semibold">{t('reseller.setup.doneTitle', 'Hub 已就绪')}</h2>
              <p className="text-sm text-muted-foreground mt-1">
                {t('reseller.setup.doneDesc', '你现在可以在「Gateway 管理」页配置 channel、生成激活码、查看日志。')}
              </p>
            </div>
            <button
              onClick={onComplete}
              className="px-5 py-2 rounded bg-purple-600 hover:bg-purple-500 text-white text-sm"
            >
              {t('reseller.setup.enter', '进入 Reseller 控制台')}
            </button>
          </section>
        )}
      </div>
    </div>
  )
}
