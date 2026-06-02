import { useTranslation } from 'react-i18next'
import { Download, X, ExternalLink } from 'lucide-react'
import { useDeepLinkImportStore, type DeepLinkProviderData } from '../stores/deeplinkImportStore'
import { useToastStore } from '../stores/toastStore'
import { useConfigStore } from '../stores/configStore'
import { GetProxySettings, SaveProxySettings, ApplyDeepLinkImport } from '../../wailsjs/go/main/App'

export function DeepLinkImportModal() {
  const { i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh')
  const { open, payload, close } = useDeepLinkImportStore()
  const addToast = useToastStore((s) => s.addToast)
  const setActiveTool = useConfigStore((s) => s.setActiveTool)

  if (!open || !payload) return null

  const isProvider = payload.type === 'provider'
  const isMCP = payload.type === 'mcp'
  const isPrompt = payload.type === 'prompt'
  const data = (payload.data ?? {}) as DeepLinkProviderData
  const firstModel = (data.models ?? '').split(',').map((m) => m.trim()).filter(Boolean)[0] ?? ''

  const isApplicable = isProvider || isMCP || isPrompt

  const handleApply = async () => {
    if (isProvider) {
      if (!data.baseUrl) {
        addToast('error', isZh ? '导入数据缺少 baseUrl' : 'Missing baseUrl in payload')
        return
      }
      try {
        const current = await GetProxySettings()
        const merged = {
          ...current,
          apiEndpoint: data.baseUrl,
          model: firstModel || current?.model || '',
        }
        await SaveProxySettings(merged as unknown as Parameters<typeof SaveProxySettings>[0])
        addToast(
          'success',
          isZh
            ? `已导入 ${data.name ?? '配置'} · 请在网关页粘贴 API Key`
            : `Imported ${data.name ?? 'preset'} · paste your API key on Gateway page`,
        )
        setActiveTool('gateway')
        close()
      } catch (err) {
        addToast(
          'error',
          isZh
            ? `导入失败：${(err as Error)?.message ?? String(err)}`
            : `Import failed: ${(err as Error)?.message ?? String(err)}`,
        )
      }
      return
    }

    if (isMCP || isPrompt) {
      try {
        const summary = await ApplyDeepLinkImport({ type: payload.type, data: payload.data, raw: payload.raw ?? '' })
        if (!summary) {
          addToast('error', isZh ? '导入返回空结果' : 'Import returned empty result')
          return
        }
        addToast('success', isZh ? `已导入：${summary}` : `Imported: ${summary}`)
        close()
      } catch (err) {
        addToast(
          'error',
          isZh
            ? `导入失败：${(err as Error)?.message ?? String(err)}`
            : `Import failed: ${(err as Error)?.message ?? String(err)}`,
        )
      }
      return
    }

    // Unknown type — surface a clear error, do not silently succeed.
    addToast('error', isZh ? `暂不支持导入类型：${payload.type}` : `Unsupported type: ${payload.type}`)
    close()
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 backdrop-blur-sm"
      onClick={close}
    >
      <div
        className="w-[480px] max-w-[92vw] max-h-[80vh] bg-background border border-border rounded-lg shadow-2xl flex flex-col overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <div className="flex items-center gap-2">
            <Download className="h-4 w-4 text-primary" />
            <h2 className="text-sm font-semibold">
              {isZh ? '导入分享配置' : 'Import shared config'}
            </h2>
          </div>
          <button
            onClick={close}
            className="text-muted-foreground hover:text-foreground transition-colors"
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="px-4 py-4 overflow-y-auto flex-1">
          <div className="text-xs text-muted-foreground mb-3 uppercase tracking-wider">
            {isZh ? `类型：${payload.type}` : `Type: ${payload.type}`}
          </div>

          {isProvider ? (
            <ProviderPreview data={data} isZh={!!isZh} />
          ) : isMCP ? (
            <MCPPreview data={payload.data as Record<string, unknown>} isZh={!!isZh} />
          ) : isPrompt ? (
            <PromptPreview data={payload.data as Record<string, unknown>} isZh={!!isZh} />
          ) : (
            <div className="text-sm text-muted-foreground py-4">
              {isZh
                ? `暂不支持导入类型 "${payload.type}"。完整 URL：`
                : `Type "${payload.type}" is not yet supported. Raw URL:`}
              <pre className="mt-2 text-xs bg-muted p-2 rounded break-all whitespace-pre-wrap">
                {payload.raw}
              </pre>
            </div>
          )}
        </div>

        <div className="px-4 py-3 border-t border-border flex items-center justify-end gap-2">
          <button
            onClick={close}
            className="px-3 py-1.5 text-sm rounded-md border border-border hover:bg-muted transition-colors"
          >
            {isZh ? '取消' : 'Cancel'}
          </button>
          <button
            onClick={handleApply}
            disabled={!isApplicable}
            className="px-3 py-1.5 text-sm rounded-md bg-primary text-primary-foreground hover:opacity-90 transition-opacity disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {isZh ? '应用' : 'Apply'}
          </button>
        </div>
      </div>
    </div>
  )
}

function ProviderPreview({ data, isZh }: { data: DeepLinkProviderData; isZh: boolean }) {
  return (
    <div className="space-y-3">
      <Field label={isZh ? '名称' : 'Name'} value={data.name} mono={false} />
      <Field label={isZh ? '接入端点' : 'Base URL'} value={data.baseUrl} mono />
      {data.models && (
        <Field label={isZh ? '建议模型' : 'Models'} value={data.models} mono />
      )}
      {data.description && (
        <Field label={isZh ? '说明' : 'Description'} value={data.description} mono={false} />
      )}
      {data.docsUrl && (
        <div className="flex items-center gap-1 text-xs">
          <ExternalLink className="h-3 w-3 text-muted-foreground" />
          <a
            href={data.docsUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary hover:underline"
          >
            {data.docsUrl}
          </a>
        </div>
      )}
      <div className="mt-4 pt-3 border-t border-border text-xs text-muted-foreground">
        {isZh
          ? '点击应用后会写入网关 baseURL，请在网关页粘贴你自己的 API Key 启用。'
          : 'Apply writes the baseURL to your gateway settings. Paste your own API key on the Gateway page to activate.'}
      </div>
    </div>
  )
}

function MCPPreview({ data, isZh }: { data: Record<string, unknown>; isZh: boolean }) {
  const server = (data.server ?? {}) as Record<string, unknown>
  return (
    <div className="space-y-3">
      <Field label={isZh ? '名称' : 'Name'} value={data.name as string} mono={false} />
      {Boolean(data.description) && (
        <Field label={isZh ? '说明' : 'Description'} value={data.description as string} mono={false} />
      )}
      <Field label={isZh ? '传输类型' : 'Transport'} value={server.type as string} mono />
      {Boolean(server.command) && (
        <Field label={isZh ? '命令' : 'Command'} value={server.command as string} mono />
      )}
      {Boolean(server.url) && (
        <Field label={isZh ? '地址' : 'URL'} value={server.url as string} mono />
      )}
      {Array.isArray(data.tags) && data.tags.length > 0 && (
        <Field label={isZh ? '标签' : 'Tags'} value={(data.tags as string[]).join(', ')} mono={false} />
      )}
      <div className="mt-4 pt-3 border-t border-border text-xs text-muted-foreground">
        {isZh ? '点击应用后保存至 MCP 预设库。' : 'Apply saves this server to your MCP preset library.'}
      </div>
    </div>
  )
}

function PromptPreview({ data, isZh }: { data: Record<string, unknown>; isZh: boolean }) {
  return (
    <div className="space-y-3">
      <Field label={isZh ? '名称' : 'Name'} value={data.name as string} mono={false} />
      {Boolean(data.category) && (
        <Field label={isZh ? '分类' : 'Category'} value={data.category as string} mono={false} />
      )}
      {Boolean(data.content) && (
        <div>
          <div className="text-xs font-medium text-muted-foreground mb-0.5">
            {isZh ? '内容预览' : 'Content preview'}
          </div>
          <pre className="text-xs bg-muted p-2 rounded break-all whitespace-pre-wrap max-h-32 overflow-y-auto">
            {(data.content as string).slice(0, 300)}
            {(data.content as string).length > 300 ? '…' : ''}
          </pre>
        </div>
      )}
      {Array.isArray(data.tags) && data.tags.length > 0 && (
        <Field label={isZh ? '标签' : 'Tags'} value={(data.tags as string[]).join(', ')} mono={false} />
      )}
      <div className="mt-4 pt-3 border-t border-border text-xs text-muted-foreground">
        {isZh ? '点击应用后保存至提示词库。' : 'Apply saves this prompt to your prompt library.'}
      </div>
    </div>
  )
}

function Field({ label, value, mono }: { label: string; value?: string; mono: boolean }) {
  if (!value) return null
  return (
    <div>
      <div className="text-xs font-medium text-muted-foreground mb-0.5">{label}</div>
      <div className={`text-sm break-all ${mono ? 'font-mono' : ''}`}>{value}</div>
    </div>
  )
}
