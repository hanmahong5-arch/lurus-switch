import { useEffect, useState, useCallback, useRef } from 'react'
import Editor from '@monaco-editor/react'
import {
  Save, FolderOpen, RotateCcw, Loader2, CheckCircle2,
  AlertTriangle, FileText, ArrowLeft,
} from 'lucide-react'
import { cn } from '../lib/utils'
import { useConfigStore } from '../stores/configStore'
import {
  ReadToolConfig,
  SaveToolConfig,
  OpenToolConfigDir,
} from '../../wailsjs/go/main/App'

const TOOL_LABELS: Record<string, string> = {
  claude: 'Claude Code',
  codex: 'Codex CLI',
  gemini: 'Gemini CLI',
  picoclaw: 'PicoClaw',
}

const TOOL_DESCRIPTIONS: Record<string, string> = {
  claude: '~/.claude/settings.json',
  codex: '~/.codex/config.toml',
  gemini: '~/.gemini/settings.json',
  picoclaw: '~/.picoclaw/config.json',
}

// Map our language names to Monaco language IDs
const MONACO_LANGUAGE: Record<string, string> = {
  json: 'json',
  toml: 'ini', // Monaco has no native TOML; ini is close enough for syntax highlighting
  markdown: 'markdown',
}

// Quick reference documentation for each tool's config options
const QUICK_REFERENCE: Record<string, Array<{ key: string; description: string; example: string }>> = {
  claude: [
    { key: 'env.ANTHROPIC_API_KEY', description: 'API key for authentication', example: '"sk-ant-..."' },
    { key: 'env.ANTHROPIC_BASE_URL', description: 'Custom API endpoint (proxy)', example: '"https://proxy.example.com"' },
    { key: 'permissions.allow', description: 'Auto-approved tool patterns', example: '["Bash(git *)", "Read"]' },
    { key: 'permissions.deny', description: 'Blocked tool patterns', example: '["Bash(rm -rf *)"]' },
    { key: 'hooks.preToolUse', description: 'Commands to run before tool execution', example: '[{"matcher":"Bash","hooks":[{"type":"command","command":"echo pre"}]}]' },
    { key: 'mcpServers', description: 'MCP server configurations', example: '{"server1":{"command":"npx","args":["-y","@mcp/server"]}}' },
  ],
  codex: [
    { key: 'model', description: 'Default model for completions', example: '"o4-mini"' },
    { key: 'approval_policy', description: 'When to ask for approval', example: '"on-failure" | "unless-allow-listed" | "never"' },
    { key: 'sandbox_mode', description: 'Sandbox restrictions', example: '"workspace-write" | "workspace-read" | "off"' },
    { key: 'model_providers.custom', description: 'Custom proxy provider', example: 'name = "Proxy"\nbase_url = "https://..."\nenv_key = "OPENAI_API_KEY"' },
    { key: 'features', description: 'Feature flags', example: 'stream = true\nautosuggest = false' },
  ],
  gemini: [
    { key: 'model.name', description: 'Default model name', example: '"gemini-2.5-flash"' },
    { key: 'general.defaultApprovalMode', description: 'Approval mode', example: '"default" | "yolo"' },
    { key: 'tools.sandbox', description: 'Enable sandboxed tool execution', example: 'false' },
    { key: 'security', description: 'Security settings', example: '{"allowedDomains": [], "blockedDomains": []}' },
    { key: 'mcpServers', description: 'MCP server configurations', example: '{"name":{"command":"npx","args":["server"]}}' },
  ],
  picoclaw: [
    { key: 'model_list[].name', description: 'Unique name for the model endpoint', example: '"default"' },
    { key: 'model_list[].api_base', description: 'API base URL', example: '"https://api.example.com/v1"' },
    { key: 'model_list[].api_key', description: 'API key for this endpoint', example: '"sk-..."' },
    { key: 'model_list[].model_name', description: 'Model identifier to use', example: '"claude-sonnet-4-20250514"' },
    { key: 'agents.defaults.model_name', description: 'Default model for all agents', example: '"claude-sonnet-4-20250514"' },
  ],
}

type SaveStatus = 'idle' | 'saving' | 'saved' | 'error'

export function ToolConfigPage() {
  const { activeTool, setActiveTool } = useConfigStore()
  const tool = activeTool as string

  const [content, setContent] = useState('')
  const [originalContent, setOriginalContent] = useState('')
  const [configPath, setConfigPath] = useState('')
  const [configExists, setConfigExists] = useState(false)
  const [language, setLanguage] = useState('json')
  const [loading, setLoading] = useState(true)
  const [saveStatus, setSaveStatus] = useState<SaveStatus>('idle')
  const [errorMsg, setErrorMsg] = useState('')
  const editorRef = useRef<any>(null)

  const loadConfig = useCallback(async () => {
    setLoading(true)
    setErrorMsg('')
    try {
      const info = await ReadToolConfig(tool)
      setContent(info.content)
      setOriginalContent(info.content)
      setConfigPath(info.path)
      setConfigExists(info.exists)
      setLanguage(info.language)
    } catch (err) {
      setErrorMsg(`Failed to load config: ${err}`)
    } finally {
      setLoading(false)
    }
  }, [tool])

  useEffect(() => {
    if (tool && tool !== 'dashboard') {
      loadConfig()
    }
  }, [tool, loadConfig])

  const handleSave = async () => {
    setSaveStatus('saving')
    setErrorMsg('')
    try {
      await SaveToolConfig(tool, content)
      setOriginalContent(content)
      setConfigExists(true)
      setSaveStatus('saved')
      setTimeout(() => setSaveStatus('idle'), 2000)
    } catch (err) {
      setSaveStatus('error')
      setErrorMsg(`Failed to save: ${err}`)
    }
  }

  const handleRevert = () => {
    setContent(originalContent)
    setSaveStatus('idle')
    setErrorMsg('')
  }

  const handleOpenDir = async () => {
    try {
      await OpenToolConfigDir(tool)
    } catch (err) {
      console.error('Failed to open directory:', err)
    }
  }

  const hasChanges = content !== originalContent
  const label = TOOL_LABELS[tool] || tool
  const desc = TOOL_DESCRIPTIONS[tool] || ''
  const quickRef = QUICK_REFERENCE[tool] || []

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-card shrink-0">
        <div className="flex items-center gap-3">
          <button
            onClick={() => setActiveTool('dashboard')}
            className="p-1 rounded hover:bg-muted transition-colors"
            title="Back to Dashboard"
          >
            <ArrowLeft className="h-4 w-4" />
          </button>
          <div>
            <h2 className="text-sm font-semibold">{label} Configuration</h2>
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <FileText className="h-3 w-3" />
              {configPath || desc}
              {!configExists && (
                <span className="text-amber-500 ml-1">(new file - will be created on save)</span>
              )}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {/* Status indicator */}
          {saveStatus === 'saved' && (
            <span className="flex items-center gap-1 text-xs text-green-500">
              <CheckCircle2 className="h-3.5 w-3.5" />
              Saved
            </span>
          )}
          {saveStatus === 'error' && (
            <span className="flex items-center gap-1 text-xs text-red-500">
              <AlertTriangle className="h-3.5 w-3.5" />
              Error
            </span>
          )}

          <button
            onClick={handleOpenDir}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border border-border hover:bg-muted transition-colors"
            title="Open config directory in file explorer"
          >
            <FolderOpen className="h-3.5 w-3.5" />
            Open Folder
          </button>

          <button
            onClick={handleRevert}
            disabled={!hasChanges}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border border-border hover:bg-muted transition-colors',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            <RotateCcw className="h-3.5 w-3.5" />
            Revert
          </button>

          <button
            onClick={handleSave}
            disabled={!hasChanges || saveStatus === 'saving'}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
              'bg-primary text-primary-foreground hover:bg-primary/90',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            {saveStatus === 'saving' ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Save className="h-3.5 w-3.5" />
            )}
            Save
          </button>
        </div>
      </div>

      {/* Error message */}
      {errorMsg && (
        <div className="px-4 py-2 bg-red-500/10 text-red-500 text-xs border-b border-red-500/20 shrink-0">
          {errorMsg}
        </div>
      )}

      {/* Main content: Editor + Quick Reference sidebar */}
      <div className="flex-1 flex overflow-hidden">
        {/* Monaco Editor */}
        <div className="flex-1 overflow-hidden">
          <Editor
            height="100%"
            language={MONACO_LANGUAGE[language] || 'json'}
            value={content}
            onChange={(value) => setContent(value || '')}
            onMount={(editor) => { editorRef.current = editor }}
            theme="vs-dark"
            options={{
              fontSize: 13,
              lineNumbers: 'on',
              minimap: { enabled: false },
              scrollBeyondLastLine: false,
              wordWrap: 'on',
              tabSize: 2,
              formatOnPaste: true,
              automaticLayout: true,
              renderWhitespace: 'selection',
              bracketPairColorization: { enabled: true },
            }}
          />
        </div>

        {/* Quick Reference Sidebar */}
        <div className="w-72 border-l border-border bg-muted/30 overflow-y-auto shrink-0">
          <div className="p-3">
            <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">
              Quick Reference
            </h3>
            <div className="space-y-3">
              {quickRef.map((item) => (
                <div
                  key={item.key}
                  className="group cursor-pointer"
                  onClick={() => {
                    // Copy the key to help user find it in the editor
                    const editor = editorRef.current
                    if (editor) {
                      // Try to find and highlight the key in the editor
                      const model = editor.getModel()
                      if (model) {
                        const searchKey = item.key.split('.').pop() || item.key
                        const matches = model.findMatches(searchKey, true, false, true, null, true)
                        if (matches.length > 0) {
                          editor.setSelection(matches[0].range)
                          editor.revealLineInCenter(matches[0].range.startLineNumber)
                        }
                      }
                      editor.focus()
                    }
                  }}
                >
                  <div className="text-xs font-mono text-primary group-hover:underline">
                    {item.key}
                  </div>
                  <div className="text-xs text-muted-foreground mt-0.5">
                    {item.description}
                  </div>
                  <div className="text-xs font-mono text-muted-foreground/70 mt-0.5 bg-muted/50 rounded px-1.5 py-0.5 whitespace-pre-wrap">
                    {item.example}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
