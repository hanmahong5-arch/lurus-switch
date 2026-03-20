import { useState } from 'react'
import { Copy, Check, ExternalLink, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../../lib/utils'

interface ConnectGuideProps {
  appId: string
  appName: string
  token: string
  gatewayUrl: string
  tier: number
  onClose: () => void
}

// Per-tool configuration instructions
const TOOL_GUIDES: Record<string, {
  steps: { label: string; code?: string; note?: string }[]
  configFile?: string
}> = {
  claude: {
    configFile: '~/.claude/settings.json',
    steps: [
      {
        label: 'Open or create the settings file',
        code: '~/.claude/settings.json',
      },
      {
        label: 'Add the API base URL',
        code: '"apiBaseUrl": "{{URL}}/v1"',
      },
      {
        label: 'Set the API key to your Switch token',
        code: 'export ANTHROPIC_API_KEY={{TOKEN}}',
        note: 'Or set it in the Claude Code config',
      },
    ],
  },
  codex: {
    configFile: '~/.codex/config.toml',
    steps: [
      {
        label: 'Open or create the config file',
        code: '~/.codex/config.toml',
      },
      {
        label: 'Add the provider section',
        code: '[provider]\napi_base_url = "{{URL}}/v1"',
      },
      {
        label: 'Set the API key',
        code: 'export OPENAI_API_KEY={{TOKEN}}',
      },
    ],
  },
  gemini: {
    steps: [
      {
        label: 'Set environment variables',
        code: 'export GEMINI_API_BASE={{URL}}/v1\nexport GEMINI_API_KEY={{TOKEN}}',
      },
    ],
  },
  aider: {
    steps: [
      {
        label: 'Run Aider with OpenAI-compatible endpoint',
        code: 'aider --openai-api-base {{URL}}/v1 --openai-api-key {{TOKEN}}',
      },
    ],
  },
  cursor: {
    steps: [
      {
        label: 'Open Cursor settings',
        note: 'Settings → Models → OpenAI API Key',
      },
      {
        label: 'Set Override OpenAI Base URL',
        code: '{{URL}}/v1',
      },
      {
        label: 'Set API Key to your Switch token',
        code: '{{TOKEN}}',
      },
    ],
  },
  windsurf: {
    steps: [
      {
        label: 'Open Windsurf settings',
        note: 'Settings → AI Provider → Custom OpenAI',
      },
      {
        label: 'Set the API Base URL',
        code: '{{URL}}/v1',
      },
      {
        label: 'Set the API Key',
        code: '{{TOKEN}}',
      },
    ],
  },
  continue: {
    configFile: '~/.continue/config.json',
    steps: [
      {
        label: 'Edit the config file',
        code: '~/.continue/config.json',
      },
      {
        label: 'Add or update the model config',
        code: '{\n  "models": [{\n    "provider": "openai",\n    "apiBase": "{{URL}}/v1",\n    "apiKey": "{{TOKEN}}",\n    "model": "claude-sonnet-4-20250514"\n  }]\n}',
      },
    ],
  },
  cline: {
    steps: [
      {
        label: 'Open VS Code → Cline extension settings',
        note: 'Or edit settings.json',
      },
      {
        label: 'Set API provider to OpenAI Compatible',
        note: 'Select "OpenAI Compatible" from the provider dropdown',
      },
      {
        label: 'Set the Base URL and API Key',
        code: 'Base URL: {{URL}}/v1\nAPI Key: {{TOKEN}}',
      },
    ],
  },
  trae: {
    steps: [
      {
        label: 'Open Trae settings → AI Model Configuration',
        note: 'Settings → AI → Custom Provider',
      },
      {
        label: 'Set API Base URL and Key',
        code: 'Base URL: {{URL}}/v1\nAPI Key: {{TOKEN}}',
      },
    ],
  },
  'zed-ai': {
    configFile: '~/.config/zed/settings.json',
    steps: [
      {
        label: 'Edit Zed settings',
        code: '~/.config/zed/settings.json',
      },
      {
        label: 'Add OpenAI compatible provider',
        code: '{\n  "language_models": {\n    "openai": {\n      "api_url": "{{URL}}/v1",\n      "api_key": "{{TOKEN}}"\n    }\n  }\n}',
      },
    ],
  },
}

// Generic guide for unknown tools
const GENERIC_GUIDE = {
  steps: [
    {
      label: 'Configure your app to use the OpenAI-compatible API',
      code: 'Base URL: {{URL}}/v1\nAPI Key:   {{TOKEN}}',
      note: 'Most AI tools support OpenAI-compatible endpoints',
    },
  ],
}

function CopyBlock({ text }: { text: string }) {
  const [copied, setCopied] = useState(false)
  const handleCopy = () => {
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }
  return (
    <div className="relative group">
      <pre className="bg-muted/80 border border-border rounded-md px-3 py-2 text-xs font-mono whitespace-pre-wrap select-all overflow-x-auto">
        {text}
      </pre>
      <button
        onClick={handleCopy}
        className="absolute top-1.5 right-1.5 p-1 rounded bg-background/80 border border-border/50 opacity-0 group-hover:opacity-100 transition-opacity"
        title="Copy"
      >
        {copied ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3 text-muted-foreground" />}
      </button>
    </div>
  )
}

export function ConnectGuide({ appId, appName, token, gatewayUrl, tier, onClose }: ConnectGuideProps) {
  const { t } = useTranslation()
  const guide = TOOL_GUIDES[appId] || GENERIC_GUIDE

  const replaceVars = (s: string) =>
    s.replace(/\{\{URL\}\}/g, gatewayUrl).replace(/\{\{TOKEN\}\}/g, token)

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-card border border-border rounded-lg shadow-xl max-w-lg w-full mx-4 max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-border">
          <div>
            <h3 className="font-semibold text-sm">{t('switch.connectGuide')}: {appName}</h3>
            <p className="text-xs text-muted-foreground mt-0.5">{t('switch.connectGuideDesc')}</p>
          </div>
          <button onClick={onClose} className="p-1 rounded hover:bg-muted">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Content */}
        <div className="p-4 space-y-4 overflow-y-auto flex-1">
          {/* Quick copy: endpoint + token */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">{t('switch.endpoint')}</p>
              <CopyBlock text={`${gatewayUrl}/v1`} />
            </div>
            <div>
              <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">{t('switch.apiToken')}</p>
              <CopyBlock text={token} />
            </div>
          </div>

          {/* Steps */}
          <div className="space-y-3">
            {guide.steps.map((step, i) => (
              <div key={i} className="space-y-1.5">
                <div className="flex items-start gap-2">
                  <span className={cn(
                    'flex-shrink-0 w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-semibold',
                    'bg-primary/10 text-primary'
                  )}>
                    {i + 1}
                  </span>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium">{step.label}</p>
                    {step.note && (
                      <p className="text-xs text-muted-foreground mt-0.5">{step.note}</p>
                    )}
                  </div>
                </div>
                {step.code && (
                  <div className="ml-7">
                    <CopyBlock text={replaceVars(step.code)} />
                  </div>
                )}
              </div>
            ))}
          </div>

          {/* Config file hint */}
          {guide.configFile && (
            <p className="text-xs text-muted-foreground border-t border-border pt-3">
              Config file: <code className="bg-muted px-1 rounded">{guide.configFile}</code>
            </p>
          )}
        </div>

        {/* Footer */}
        <div className="p-3 border-t border-border flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-1.5 rounded-md text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90"
          >
            Done
          </button>
        </div>
      </div>
    </div>
  )
}
