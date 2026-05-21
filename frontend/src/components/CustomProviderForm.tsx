import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, CheckCircle2, AlertTriangle, Plug, Save, X } from 'lucide-react'
import { cn } from '../lib/utils'
import { SaveCustomProvider, TestCustomProvider } from '../../wailsjs/go/main/App'

// Mirror of provider.CustomProvider on the Go side.
export interface CustomProvider {
  id: string
  name: string
  baseUrl: string
  apiKey: string
  defaultModels: string[]
  headers?: Record<string, string>
  docsUrl?: string
  description?: string
  createdAt?: string
}

interface TestResult {
  ok: boolean
  models: string[]
  latencyMs: number
  error?: string
}

interface Props {
  initial?: CustomProvider | null
  onSaved: (p: CustomProvider) => void
  onCancel: () => void
}

function blank(): CustomProvider {
  return { id: '', name: '', baseUrl: '', apiKey: '', defaultModels: [] }
}

export function CustomProviderForm({ initial, onSaved, onCancel }: Props) {
  const { t } = useTranslation()
  const [form, setForm] = useState<CustomProvider>(initial ?? blank())
  const [modelsText, setModelsText] = useState((initial?.defaultModels ?? []).join(', '))
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<TestResult | null>(null)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const set = (patch: Partial<CustomProvider>) => setForm((f) => ({ ...f, ...patch }))

  const parsedModels = () =>
    modelsText.split(',').map((s) => s.trim()).filter(Boolean)

  const handleTest = async () => {
    if (!form.baseUrl.trim()) {
      setError(t('customProvider.errBaseUrl', '请填写 Base URL'))
      return
    }
    setTesting(true)
    setError(null)
    setTestResult(null)
    try {
      const r = (await TestCustomProvider({
        ...form,
        defaultModels: parsedModels(),
      } as any)) as TestResult
      setTestResult(r)
    } catch (e: any) {
      setError(e?.message ?? String(e))
    } finally {
      setTesting(false)
    }
  }

  const handleSave = async () => {
    if (!form.baseUrl.trim()) {
      setError(t('customProvider.errBaseUrl', '请填写 Base URL'))
      return
    }
    setSaving(true)
    setError(null)
    try {
      const saved = (await SaveCustomProvider({
        ...form,
        defaultModels: parsedModels(),
      } as any)) as CustomProvider
      onSaved(saved)
    } catch (e: any) {
      setError(e?.message ?? String(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="rounded-lg border border-border bg-card p-4 space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">
          {initial ? t('customProvider.editTitle', '编辑自定义供应商') : t('customProvider.addTitle', '添加自定义供应商')}
        </h3>
        <button onClick={onCancel} className="p-1 rounded hover:bg-muted" title={t('common.close', '关闭')}>
          <X className="h-4 w-4" />
        </button>
      </div>

      <Field label={t('customProvider.name', '名称')}>
        <input
          value={form.name}
          onChange={(e) => set({ name: e.target.value })}
          placeholder={t('customProvider.namePlaceholder', '例如 内部 LLM 网关')}
          className="w-full px-2.5 py-1.5 text-sm bg-background border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
        />
      </Field>

      <Field label={t('customProvider.baseUrl', 'Base URL')} required>
        <input
          value={form.baseUrl}
          onChange={(e) => set({ baseUrl: e.target.value })}
          placeholder="https://api.example.com/v1"
          className="w-full px-2.5 py-1.5 text-sm bg-background border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
        />
      </Field>

      <Field label={t('customProvider.apiKey', 'API Key')}>
        <input
          type="password"
          value={form.apiKey}
          onChange={(e) => set({ apiKey: e.target.value })}
          placeholder="sk-..."
          className="w-full px-2.5 py-1.5 text-sm bg-background border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
        />
      </Field>

      <Field label={t('customProvider.models', '默认模型（逗号分隔）')}>
        <input
          value={modelsText}
          onChange={(e) => setModelsText(e.target.value)}
          placeholder="gpt-4o, claude-3-5-sonnet"
          className="w-full px-2.5 py-1.5 text-sm bg-background border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
        />
      </Field>

      <Field label={t('customProvider.docsUrl', '文档链接')}>
        <input
          value={form.docsUrl ?? ''}
          onChange={(e) => set({ docsUrl: e.target.value })}
          placeholder="https://docs.example.com"
          className="w-full px-2.5 py-1.5 text-sm bg-background border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
        />
      </Field>

      {error && (
        <div className="text-xs text-red-500 bg-red-500/10 border border-red-500/20 rounded px-2 py-1.5">
          {error}
        </div>
      )}

      {testResult && (
        <div
          className={cn(
            'text-xs rounded px-2 py-1.5 border flex items-center gap-2',
            testResult.ok
              ? 'text-emerald-500 bg-emerald-500/10 border-emerald-500/20'
              : 'text-red-500 bg-red-500/10 border-red-500/20',
          )}
        >
          {testResult.ok ? <CheckCircle2 className="h-3.5 w-3.5" /> : <AlertTriangle className="h-3.5 w-3.5" />}
          {testResult.ok
            ? t('customProvider.testOk', '连接成功：{{count}} 个模型 · {{ms}}ms', {
                count: testResult.models.length,
                ms: testResult.latencyMs,
              })
            : t('customProvider.testFail', '连接失败：{{err}}', { err: testResult.error ?? '' })}
        </div>
      )}

      <div className="flex items-center gap-2 pt-1">
        <button
          onClick={handleTest}
          disabled={testing}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded border border-border hover:bg-muted disabled:opacity-50"
        >
          {testing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Plug className="h-3.5 w-3.5" />}
          {t('customProvider.test', '测试连接')}
        </button>
        <button
          onClick={handleSave}
          disabled={saving}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 ml-auto"
        >
          {saving ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Save className="h-3.5 w-3.5" />}
          {t('common.save', '保存')}
        </button>
      </div>
    </div>
  )
}

function Field({ label, required, children }: { label: string; required?: boolean; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className="text-xs text-muted-foreground mb-1 inline-block">
        {label}
        {required && <span className="text-red-500 ml-0.5">*</span>}
      </span>
      {children}
    </label>
  )
}
