import { useEffect, useState } from 'react'
import { CheckCircle2, Loader2, AlertCircle, Save, Network, Play, Wand2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import {
  GetProxySettings,
  SaveProxySettings,
  GetUpstreamProxy,
  TestUpstreamProxy,
  DetectLocalProxies,
} from '../../wailsjs/go/main/App'
import { proxy, netproxy } from '../../wailsjs/go/models'
import { ConnectivityDoctor } from './ConnectivityDoctor'

/**
 * Upstream HTTP/HTTPS/SOCKS5 proxy settings.
 *
 * This is the "BYO VPN" hook — users running V2Ray / Clash / a corporate
 * HTTP proxy point Switch at it and every outbound request (Anthropic
 * relay, GitHub updater, npm checker, …) flows through it. Switch ships
 * no censorship-evasion logic of its own.
 */
export function UpstreamProxySection() {
  const { t } = useTranslation()

  const [enabled, setEnabled] = useState(false)
  const [url, setUrl] = useState('')
  const [noProxy, setNoProxy] = useState('')
  const [testUrl, setTestUrl] = useState('')

  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [detecting, setDetecting] = useState(false)
  const [detectMsg, setDetectMsg] = useState<string | null>(null)
  const [savedFlash, setSavedFlash] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [testResult, setTestResult] = useState<netproxy.TestResult | null>(null)

  useEffect(() => {
    GetUpstreamProxy()
      .then((s) => {
        if (s) {
          setEnabled(Boolean(s.enabled))
          setUrl(s.url ?? '')
          setNoProxy(s.noProxy ?? '')
          setTestUrl(s.testUrl ?? '')
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const buildPayload = (): netproxy.Settings =>
    netproxy.Settings.createFrom({
      enabled,
      url: url.trim(),
      noProxy: noProxy.trim(),
      testUrl: testUrl.trim(),
    })

  const handleAutoDetect = async () => {
    setDetecting(true)
    setDetectMsg(null)
    try {
      const found = await DetectLocalProxies()
      if (!found || found.length === 0) {
        setDetectMsg(t('settings.upstreamProxy.detectNone', '未检测到本地代理。请先启动 V2Ray / Clash 或 plugins/dnstunnel 客户端。'))
      } else {
        const pick = found[0]
        setUrl(pick.url)
        setEnabled(true)
        setDetectMsg(
          t('settings.upstreamProxy.detectHit', '已检测到 {{name}}：{{url}}', {
            name: pick.guessedName || 'proxy',
            url: pick.url,
          }),
        )
      }
    } catch (err) {
      setDetectMsg((err as Error)?.message ?? String(err))
    } finally {
      setDetecting(false)
    }
  }

  const handleTest = async () => {
    setTesting(true)
    setTestResult(null)
    try {
      const r = await TestUpstreamProxy(buildPayload())
      setTestResult(r)
    } catch (err) {
      setTestResult(
        netproxy.TestResult.createFrom({
          ok: false,
          error: (err as Error)?.message ?? String(err),
          probedUrl: testUrl.trim() || 'https://www.google.com/generate_204',
        }),
      )
    } finally {
      setTesting(false)
    }
  }

  const handleSave = async () => {
    setSaving(true)
    setSaveError(null)
    try {
      const current = await GetProxySettings()
      const merged = proxy.ProxySettings.createFrom({
        ...(current ?? {}),
        upstreamProxy: buildPayload(),
      })
      await SaveProxySettings(merged)
      setSavedFlash(true)
      setTimeout(() => setSavedFlash(false), 2000)
    } catch (err) {
      setSaveError((err as Error)?.message ?? String(err))
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-6">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  const exampleHints = [
    'http://127.0.0.1:7890',
    'socks5://127.0.0.1:1080',
    'socks5h://user:pass@proxy.example.com:1080',
  ]

  return (
    <div className="space-y-5">
      <div className="flex items-start gap-2">
        <Network className="h-4 w-4 mt-0.5 text-muted-foreground" />
        <div>
          <h3 className="text-sm font-semibold">
            {t('settings.upstreamProxy.title', '上游代理')}
          </h3>
          <p className="text-xs text-muted-foreground mt-0.5">
            {t(
              'settings.upstreamProxy.subtitle',
              '所有出站 HTTP 请求（AI 中转、自动更新、版本检查）将通过此代理转发。Switch 自身不提供任何翻墙能力，仅支持您接入自有 V2Ray / Clash / 公司代理。',
            )}
          </p>
        </div>
      </div>

      <div className="rounded-md border border-border bg-muted/30 p-4 space-y-4">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(e) => setEnabled(e.target.checked)}
            className="w-4 h-4 accent-primary"
          />
          <span className="text-sm font-medium">
            {t('settings.upstreamProxy.enable', '启用上游代理')}
          </span>
        </label>

        <div className={cn(!enabled && 'opacity-50 pointer-events-none')}>
          <div className="space-y-3">
            <div>
              <label className="text-xs font-medium block mb-1">
                {t('settings.upstreamProxy.url', '代理地址')}
              </label>
              <input
                type="text"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="socks5://127.0.0.1:1080"
                spellCheck={false}
                className="w-full px-3 py-1.5 text-sm font-mono bg-background border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <p className="text-[11px] text-muted-foreground mt-1">
                {t(
                  'settings.upstreamProxy.urlHint',
                  '支持 http / https / socks5 / socks5h（推荐 socks5h 由代理解析 DNS）',
                )}
              </p>
              <div className="flex flex-wrap gap-1.5 mt-1.5">
                {exampleHints.map((ex) => (
                  <button
                    key={ex}
                    type="button"
                    onClick={() => setUrl(ex)}
                    className="text-[10px] font-mono px-2 py-0.5 rounded border border-border bg-background hover:bg-muted"
                  >
                    {ex}
                  </button>
                ))}
              </div>
            </div>

            <div>
              <label className="text-xs font-medium block mb-1">
                {t('settings.upstreamProxy.noProxy', '绕过列表')}
              </label>
              <input
                type="text"
                value={noProxy}
                onChange={(e) => setNoProxy(e.target.value)}
                placeholder="lurus.cn, internal.corp"
                spellCheck={false}
                className="w-full px-3 py-1.5 text-sm font-mono bg-background border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <p className="text-[11px] text-muted-foreground mt-1">
                {t(
                  'settings.upstreamProxy.noProxyHint',
                  '逗号分隔的域名后缀。localhost / 127.0.0.0/8 / ::1 始终绕过。',
                )}
              </p>
            </div>

            <div>
              <label className="text-xs font-medium block mb-1">
                {t('settings.upstreamProxy.testUrl', '测试目标')}
              </label>
              <input
                type="text"
                value={testUrl}
                onChange={(e) => setTestUrl(e.target.value)}
                placeholder="https://www.google.com/generate_204"
                spellCheck={false}
                className="w-full px-3 py-1.5 text-sm font-mono bg-background border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <p className="text-[11px] text-muted-foreground mt-1">
                {t(
                  'settings.upstreamProxy.testUrlHint',
                  '点"测试"时探测此 URL。空则使用默认（Google 204）。',
                )}
              </p>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2 pt-2 border-t border-border flex-wrap">
          <button
            onClick={handleAutoDetect}
            disabled={detecting}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md transition-colors',
              'border border-border hover:bg-muted',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
          >
            {detecting ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Wand2 className="h-3.5 w-3.5" />
            )}
            {t('settings.upstreamProxy.autoDetect', '自动检测')}
          </button>

          <button
            onClick={handleTest}
            disabled={testing || !enabled || !url.trim()}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md transition-colors',
              'border border-border hover:bg-muted',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
          >
            {testing ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Play className="h-3.5 w-3.5" />
            )}
            {t('settings.upstreamProxy.test', '测试连接')}
          </button>

          <button
            onClick={handleSave}
            disabled={saving}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md transition-colors',
              'bg-primary text-primary-foreground hover:bg-primary/90',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
          >
            {saving ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : savedFlash ? (
              <CheckCircle2 className="h-3.5 w-3.5" />
            ) : (
              <Save className="h-3.5 w-3.5" />
            )}
            {savedFlash
              ? t('settings.saved', '已保存')
              : t('settings.upstreamProxy.save', '保存并立即生效')}
          </button>
        </div>

        {saveError && (
          <div className="flex items-start gap-2 px-3 py-2 rounded-md bg-red-500/10 border border-red-500/30 text-red-500 text-xs">
            <AlertCircle className="h-3.5 w-3.5 mt-0.5 shrink-0" />
            <span className="break-all">{saveError}</span>
          </div>
        )}

        {detectMsg && (
          <div className="text-xs px-3 py-2 rounded-md bg-muted/50 border border-border break-all">
            {detectMsg}
          </div>
        )}

        {testResult && (
          <TestResultLine result={testResult} />
        )}
      </div>

      <div className="pt-4 border-t border-border">
        <ConnectivityDoctor />
      </div>

      <details className="text-xs text-muted-foreground">
        <summary className="cursor-pointer hover:text-foreground">
          {t('settings.upstreamProxy.helpTitle', '什么是上游代理？')}
        </summary>
        <div className="mt-2 space-y-2 pl-4 border-l border-border">
          <p>
            {t(
              'settings.upstreamProxy.help1',
              '如果您所在的网络环境无法直接访问 Anthropic / GitHub / npm 等服务，可在本地运行 V2Ray、Clash、Shadowsocks 等独立代理客户端，然后在此填入其本地监听地址。',
            )}
          </p>
          <p>
            {t(
              'settings.upstreamProxy.help2',
              'Switch 仅做一件事：把自己的所有 HTTP 请求转交给您填的代理。不内置任何 VPN / DNS 隧道 / 翻墙逻辑。',
            )}
          </p>
          <p>
            {t(
              'settings.upstreamProxy.help3',
              '修改保存后立即生效，无需重启。新发起的请求会走新代理；已在传输中的连接保持原状直到完成。',
            )}
          </p>
        </div>
      </details>
    </div>
  )
}

function TestResultLine({ result }: { result: netproxy.TestResult }) {
  const { t } = useTranslation()
  const ok = result.ok
  return (
    <div
      className={cn(
        'flex items-start gap-2 px-3 py-2 rounded-md text-xs border',
        ok
          ? 'bg-green-500/10 border-green-500/30 text-green-600 dark:text-green-400'
          : 'bg-red-500/10 border-red-500/30 text-red-500',
      )}
    >
      {ok ? (
        <CheckCircle2 className="h-3.5 w-3.5 mt-0.5 shrink-0" />
      ) : (
        <AlertCircle className="h-3.5 w-3.5 mt-0.5 shrink-0" />
      )}
      <div className="min-w-0 flex-1">
        {ok ? (
          <span>
            {t('settings.upstreamProxy.testOK', '连接成功')} · HTTP {result.statusCode} ·{' '}
            {result.latencyMs}ms
          </span>
        ) : (
          <>
            <div className="font-medium">{t('settings.upstreamProxy.testFail', '连接失败')}</div>
            <div className="break-all opacity-80">{result.error || 'unknown error'}</div>
          </>
        )}
        <div className="opacity-60 mt-0.5 break-all">
          → {result.probedUrl}
        </div>
      </div>
    </div>
  )
}
