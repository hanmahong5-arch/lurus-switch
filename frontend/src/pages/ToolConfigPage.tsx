import { useEffect, useState, useCallback, useRef } from 'react'
import Editor from '@monaco-editor/react'
import {
  Save, FolderOpen, RotateCcw, Loader2, CheckCircle2,
  AlertTriangle, FileText, Camera, Clock, RotateCw, X,
  FormInput, Code2, Cloud, ChevronDown, ChevronUp, Tag,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { errorToast } from '../lib/errorToast'
import { useConfigStore, type ConfigPreset, type ToolsSubTab } from '../stores/configStore'
import { useToastStore } from '../stores/toastStore'
import { useDashboardStore } from '../stores/dashboardStore'
import {
  ReadToolConfig,
  SaveToolConfig,
  OpenToolConfigDir,
  TakeConfigSnapshot,
  ListConfigSnapshots,
  RestoreConfigSnapshot,
  DeleteConfigSnapshot,
  FetchCloudPresets,
} from '../../wailsjs/go/main/App'
import { ClaudeConfigForm } from '../components/forms/ClaudeConfigForm'
import { CodexConfigForm } from '../components/forms/CodexConfigForm'
import { GeminiConfigForm } from '../components/forms/GeminiConfigForm'
import { ZeroClawConfigForm } from '../components/forms/ZeroClawConfigForm'
import { OpenClawConfigForm } from '../components/forms/OpenClawConfigForm'
import { ProductTabBar } from '../components/ProductTabBar'
import { ContextSidebar } from '../components/ContextSidebar'
import { SectionDescriptionBanner } from '../components/SectionDescriptionBanner'
import { useFormSectionSync } from '../hooks/useFormSectionSync'
import bundledSchemas from '../assets/tool-schemas.json'
import { getToolSections } from '../lib/toolSchema'
import type { ToolSchema } from '../lib/toolSchema'

/** Tools that support the visual form editor. */
const TOOLS_WITH_FORM = new Set(['claude', 'codex', 'gemini', 'zeroclaw', 'openclaw'])

const TOOL_DESCRIPTIONS: Record<string, string> = {
  claude:   '~/.claude/settings.json',
  codex:    '~/.codex/config.toml',
  gemini:   '~/.gemini/settings.json',
  picoclaw: '~/.picoclaw/config.json',
  nullclaw: '~/.nullclaw/config.json',
  zeroclaw: '~/.zeroclaw/config.toml',
  openclaw: '~/.openclaw/openclaw.json',
}

const TOOL_LABELS: Record<string, string> = {
  claude:   'Claude Code',
  codex:    'Codex CLI',
  gemini:   'Gemini CLI',
  picoclaw: 'PicoClaw',
  nullclaw: 'NullClaw',
  zeroclaw: 'ZeroClaw',
  openclaw: 'OpenClaw',
}

const MONACO_LANGUAGE: Record<string, string> = {
  json:     'json',
  toml:     'ini',
  markdown: 'markdown',
}

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
  nullclaw: [
    { key: 'model_list[].name', description: 'Unique name for the model endpoint', example: '"code-switch"' },
    { key: 'model_list[].api_base', description: 'API base URL', example: '"https://api.example.com/v1"' },
    { key: 'model_list[].api_key', description: 'API key for this endpoint', example: '"sk-..."' },
    { key: 'model_list[].model_name', description: 'Model identifier to use', example: '"claude-sonnet-4-20250514"' },
    { key: 'agents.defaults.model_name', description: 'Default model for all agents', example: '"claude-sonnet-4-20250514"' },
  ],
  zeroclaw: [
    { key: 'provider.type',     description: 'AI provider type',              example: '"anthropic"' },
    { key: 'provider.api_key',  description: 'API key',                       example: '"sk-..."' },
    { key: 'provider.model',    description: 'Model name',                    example: '"claude-sonnet-4-20250514"' },
    { key: 'provider.base_url', description: 'Custom base URL (proxy)',       example: '"https://api.example.com"' },
    { key: 'gateway.port',      description: 'Local gateway port',            example: '8765' },
    { key: 'memory.backend',    description: 'Memory storage backend',        example: '"sqlite"' },
  ],
  openclaw: [
    { key: 'provider.type',      description: 'AI provider type',             example: '"anthropic"' },
    { key: 'provider.api_key',   description: 'API key',                      example: '"sk-..."' },
    { key: 'provider.model',     description: 'Model name',                   example: '"claude-sonnet-4-20250514"' },
    { key: 'gateway.port',       description: 'Gateway port',                 example: '18789' },
    { key: 'channels.dm_policy', description: 'Who can DM the bot',           example: '"all"' },
    { key: 'skills.enabled',     description: 'Active skill list',            example: '["web-search"]' },
  ],
}

type SaveStatus = 'idle' | 'saving' | 'saved' | 'error'
type ViewMode = 'form' | 'text'

interface SnapshotMeta {
  id: string
  tool: string
  label: string
  createdAt: string
  size: number
}

const TOOL_NAMES: ToolsSubTab[] = [
  'claude', 'codex', 'gemini', 'picoclaw', 'nullclaw', 'zeroclaw', 'openclaw',
]

export function ToolConfigPage() {
  const { t } = useTranslation()
  const {
    lastActiveTool, setLastActiveTool,
    getSubTab, setSubTab,
    activeSection, setActiveSection,
    cloudPresets, setCloudPresets,
    highlightField, setHighlightField,
  } = useConfigStore()

  const { tools } = useDashboardStore()
  const toast = useToastStore((s) => s.addToast)
  const [dismissedBanner, setDismissedBanner] = useState(false)

  // Determine active tool from sub-tab state (new navigation model)
  const activeToolSubTab = getSubTab('tools', lastActiveTool || 'claude') as ToolsSubTab
  const tool = TOOL_NAMES.includes(activeToolSubTab) ? activeToolSubTab : lastActiveTool || 'claude'
  const toolKnownNotInstalled = Object.keys(tools).length > 0 && tools[tool]?.installed === false

  // Schemas from bundled JSON (type-cast; remote updates applied via schemaCache in App.tsx if needed)
  const schemas = bundledSchemas as ToolSchema[]
  const currentSections = getToolSections(tool, schemas)

  // Section scroll sync (IntersectionObserver)
  const { scrollToSection } = useFormSectionSync(tool, currentSections, setActiveSection)

  const [content, setContent] = useState('')
  const [originalContent, setOriginalContent] = useState('')
  const [configPath, setConfigPath] = useState('')
  const [configExists, setConfigExists] = useState(false)
  const [language, setLanguage] = useState('json')
  const [loading, setLoading] = useState(true)
  const [saveStatus, setSaveStatus] = useState<SaveStatus>('idle')
  const [viewMode, setViewMode] = useState<ViewMode>('text')
  const editorRef = useRef<any>(null)

  // Snapshot panel state
  const [snapshotPanelOpen, setSnapshotPanelOpen] = useState(false)
  const [snapshots, setSnapshots] = useState<SnapshotMeta[]>([])
  const [snapshotLabel, setSnapshotLabel] = useState('')
  const [snapshotBusy, setSnapshotBusy] = useState(false)

  // Cloud presets panel state
  const [presetsOpen, setPresetsOpen] = useState(false)
  const [presetsLoading, setPresetsLoading] = useState(false)
  const [presetDiffPreview, setPresetDiffPreview] = useState<ConfigPreset | null>(null)

  const loadConfig = useCallback(async () => {
    setLoading(true)
    try {
      const info = await ReadToolConfig(tool)
      setContent(info.content)
      setOriginalContent(info.content)
      setConfigPath(info.path)
      setConfigExists(info.exists)
      setLanguage(info.language)
    } catch (err) {
      errorToast(toast, err, { retry: () => loadConfig() })
    } finally {
      setLoading(false)
    }
  }, [tool, toast])

  const loadSnapshots = useCallback(async () => {
    try {
      const list = await ListConfigSnapshots(tool)
      setSnapshots(list || [])
    } catch {
      setSnapshots([])
    }
  }, [tool])

  const loadCloudPresets = useCallback(async () => {
    if (cloudPresets[tool]?.length > 0) return
    setPresetsLoading(true)
    try {
      const presets = await FetchCloudPresets(tool)
      setCloudPresets(tool, presets || [])
    } catch {
      setCloudPresets(tool, [])
    } finally {
      setPresetsLoading(false)
    }
  }, [tool, cloudPresets, setCloudPresets])

  useEffect(() => {
    if (tool) {
      loadConfig()
    }
  }, [tool, loadConfig])

  useEffect(() => {
    if (snapshotPanelOpen) {
      loadSnapshots()
    }
  }, [snapshotPanelOpen, loadSnapshots])

  // Track last active tool sub-tab
  useEffect(() => {
    if (TOOL_NAMES.includes(tool as ToolsSubTab)) {
      setLastActiveTool(tool as ToolsSubTab)
    }
  }, [tool, setLastActiveTool])

  // Reset active section when switching tools
  useEffect(() => {
    setActiveSection('core')
    setDismissedBanner(false)
  }, [tool, setActiveSection])

  // Highlight a field in Monaco when navigated here from health check
  useEffect(() => {
    if (!highlightField || viewMode !== 'text') return
    const editor = editorRef.current
    if (!editor) return
    const model = editor.getModel()
    if (!model) return
    const searchTerm = highlightField.split('.').pop() || highlightField
    const matches = model.findMatches(searchTerm, true, false, false, null, true)
    if (matches.length > 0) {
      editor.setSelection(matches[0].range)
      editor.revealLineInCenter(matches[0].range.startLineNumber)
      editor.focus()
    }
    setHighlightField('')
  }, [highlightField, viewMode, setHighlightField])

  const handleSave = async () => {
    setSaveStatus('saving')
    try {
      await SaveToolConfig(tool, content)
      setOriginalContent(content)
      setConfigExists(true)
      setSaveStatus('saved')
      setTimeout(() => setSaveStatus('idle'), 2000)
      toast('success', 'Configuration saved')
    } catch (err) {
      setSaveStatus('error')
      errorToast(toast, err, { retry: () => handleSave() })
    }
  }

  const handleRevert = () => {
    setContent(originalContent)
    setSaveStatus('idle')
  }

  const handleOpenDir = async () => {
    try {
      await OpenToolConfigDir(tool)
    } catch (err) {
      console.error('Failed to open directory:', err)
    }
  }

  const handleTakeSnapshot = async () => {
    setSnapshotBusy(true)
    try {
      await TakeConfigSnapshot(tool, snapshotLabel || new Date().toLocaleString())
      setSnapshotLabel('')
      await loadSnapshots()
      toast('success', 'Snapshot saved')
    } catch (err) {
      errorToast(toast, err)
    } finally {
      setSnapshotBusy(false)
    }
  }

  const handleRestoreSnapshot = async (id: string) => {
    setSnapshotBusy(true)
    try {
      await RestoreConfigSnapshot(tool, id)
      await loadConfig()
      setSnapshotPanelOpen(false)
      toast('success', 'Snapshot restored')
    } catch (err) {
      errorToast(toast, err)
    } finally {
      setSnapshotBusy(false)
    }
  }

  const handleDeleteSnapshot = async (id: string) => {
    try {
      await DeleteConfigSnapshot(tool, id)
      await loadSnapshots()
    } catch (err) {
      errorToast(toast, err)
    }
  }

  const hasChanges = content !== originalContent
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
          <div>
            <h2 className="text-sm font-semibold">{TOOL_LABELS[tool] || tool} Configuration</h2>
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

          {/* Form / Text mode toggle (only for tools with form support) */}
          {TOOLS_WITH_FORM.has(tool) && (
            <div className="flex items-center border border-border rounded-md overflow-hidden">
              <button
                onClick={() => setViewMode('form')}
                className={cn(
                  'flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium transition-colors',
                  viewMode === 'form'
                    ? 'bg-primary text-primary-foreground'
                    : 'hover:bg-muted text-muted-foreground'
                )}
                title="Visual form editor"
              >
                <FormInput className="h-3.5 w-3.5" />
                Form
              </button>
              <button
                onClick={() => setViewMode('text')}
                className={cn(
                  'flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium transition-colors',
                  viewMode === 'text'
                    ? 'bg-primary text-primary-foreground'
                    : 'hover:bg-muted text-muted-foreground'
                )}
                title="Raw text editor"
              >
                <Code2 className="h-3.5 w-3.5" />
                Text
              </button>
            </div>
          )}

          <button
            onClick={() => {
              const next = !presetsOpen
              setPresetsOpen(next)
              setSnapshotPanelOpen(false)
              if (next) loadCloudPresets()
            }}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border border-border hover:bg-muted transition-colors"
            title="Cloud config presets"
          >
            <Cloud className="h-3.5 w-3.5" />
            Cloud Presets
            {presetsOpen ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
          </button>

          <button
            onClick={() => setSnapshotPanelOpen(!snapshotPanelOpen)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border border-border hover:bg-muted transition-colors"
            title="Manage config snapshots"
          >
            <Camera className="h-3.5 w-3.5" />
            Snapshots
          </button>

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
            data-shortcut="save"
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

      {/* Not-installed banner */}
      {!dismissedBanner && toolKnownNotInstalled && (
        <div className="flex items-center gap-3 px-4 py-2.5 bg-amber-500/10 border-b border-amber-500/20 text-xs shrink-0">
          <AlertTriangle className="h-4 w-4 text-amber-500 shrink-0" />
          <div className="flex-1 min-w-0">
            <span className="font-medium text-amber-600">{TOOL_LABELS[tool] || tool} {t('dashboard.notInstalled')}</span>
            <span className="text-muted-foreground ml-1.5">{t('toolConfig.notInstalledHint')}</span>
          </div>
          <button
            onClick={() => useConfigStore.getState().setActiveTool('home')}
            className="flex items-center gap-1 px-2.5 py-1 rounded text-xs font-medium bg-amber-500/20 hover:bg-amber-500/30 text-amber-600 transition-colors whitespace-nowrap"
          >
            {t('toolConfig.goInstall')}
          </button>
          <button
            onClick={() => setDismissedBanner(true)}
            className="p-1 hover:bg-amber-500/20 rounded text-amber-500/70 hover:text-amber-500 transition-colors"
            title={t('dashboard.dismiss')}
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      )}

      {/* ProductTabBar — tool switcher */}
      <ProductTabBar activeTool={tool} onSelect={(t) => { setSubTab('tools', t); setLastActiveTool(t) }} />

      {/* Main content: ContextSidebar + Editor + Sidebars */}
      <div className="flex-1 flex overflow-hidden">
        {/* ContextSidebar — only shown in form mode */}
        {viewMode === 'form' && currentSections.length > 0 && (
          <ContextSidebar
            toolId={tool}
            sections={currentSections}
            activeSection={activeSection}
            onSectionClick={scrollToSection}
          />
        )}

        {/* Editor area (form or Monaco) */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {/* Section description banner — only in form mode */}
          {viewMode === 'form' && currentSections.length > 0 && (
            <SectionDescriptionBanner
              toolId={tool}
              activeSection={activeSection}
              sections={currentSections}
            />
          )}

          <div className="flex-1 overflow-hidden">
            {viewMode === 'form' && TOOLS_WITH_FORM.has(tool) ? (
              <div className="h-full overflow-y-auto">
                {tool === 'claude' && (
                  <ClaudeConfigForm
                    initialContent={content}
                    onChange={setContent}
                    onValidation={() => {}}
                  />
                )}
                {tool === 'codex' && (
                  <CodexConfigForm
                    initialContent={content}
                    onChange={setContent}
                    onValidation={() => {}}
                  />
                )}
                {tool === 'gemini' && (
                  <GeminiConfigForm
                    initialContent={content}
                    onChange={setContent}
                    onValidation={() => {}}
                  />
                )}
                {tool === 'zeroclaw' && (
                  <ZeroClawConfigForm
                    initialContent={content}
                    onChange={setContent}
                  />
                )}
                {tool === 'openclaw' && (
                  <OpenClawConfigForm
                    initialContent={content}
                    onChange={setContent}
                  />
                )}
              </div>
            ) : (
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
            )}
          </div>
        </div>

        {/* Cloud Presets Panel (slide-in) */}
        {presetsOpen && (
          <div className="w-72 border-l border-border bg-card overflow-y-auto shrink-0 flex flex-col">
            <div className="p-3 border-b border-border flex items-center justify-between shrink-0">
              <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                Cloud Presets
              </h3>
              <button onClick={() => setPresetsOpen(false)} className="p-1 hover:bg-muted rounded">
                <X className="h-3.5 w-3.5" />
              </button>
            </div>

            <div className="flex-1 overflow-y-auto p-2 space-y-2">
              {presetsLoading ? (
                <div className="flex items-center justify-center gap-2 py-6">
                  <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                </div>
              ) : (cloudPresets[tool] || []).length === 0 ? (
                <p className="text-xs text-muted-foreground text-center py-4">No cloud presets available</p>
              ) : (
                (cloudPresets[tool] || []).map((preset) => (
                  <div
                    key={preset.id}
                    className="border border-border rounded-md p-2.5 space-y-1.5 bg-muted/30 cursor-pointer hover:bg-muted/50 transition-colors"
                    onClick={() => setPresetDiffPreview(preset)}
                  >
                    <div className="flex items-start justify-between gap-1">
                      <span className="text-xs font-medium truncate" title={preset.name}>
                        {preset.name}
                      </span>
                      {preset.category && (
                        <span className="flex items-center gap-0.5 text-xs bg-primary/10 text-primary rounded px-1.5 py-0.5 shrink-0">
                          <Tag className="h-2.5 w-2.5" />
                          {preset.category}
                        </span>
                      )}
                    </div>
                    {preset.description && (
                      <p className="text-xs text-muted-foreground leading-relaxed">{preset.description}</p>
                    )}
                  </div>
                ))
              )}
            </div>
          </div>
        )}

        {/* Preset Diff Preview Dialog */}
        {presetDiffPreview && (
          <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
            <div className="bg-card border border-border rounded-lg w-full max-w-3xl shadow-2xl flex flex-col max-h-[80vh]">
              <div className="flex items-center justify-between px-4 py-3 border-b border-border shrink-0">
                <div>
                  <h3 className="text-sm font-semibold">Apply Preset: {presetDiffPreview.name}</h3>
                  <p className="text-xs text-muted-foreground mt-0.5">{presetDiffPreview.description}</p>
                </div>
                <button
                  onClick={() => setPresetDiffPreview(null)}
                  className="p-1 hover:bg-muted rounded transition-colors"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
              <div className="flex flex-1 overflow-hidden min-h-0">
                <div className="flex-1 border-r border-border overflow-y-auto">
                  <div className="px-3 py-2 text-xs font-semibold text-muted-foreground bg-muted/30 border-b border-border">
                    Current
                  </div>
                  <pre className="p-3 text-xs font-mono text-foreground whitespace-pre-wrap break-all">{content}</pre>
                </div>
                <div className="flex-1 overflow-y-auto">
                  <div className="px-3 py-2 text-xs font-semibold text-primary bg-primary/5 border-b border-border">
                    Preset Content
                  </div>
                  <pre className="p-3 text-xs font-mono text-foreground whitespace-pre-wrap break-all">
                    {JSON.stringify(presetDiffPreview.config_json, null, 2)}
                  </pre>
                </div>
              </div>
              <div className="flex justify-end gap-2 px-4 py-3 border-t border-border shrink-0">
                <button
                  onClick={() => setPresetDiffPreview(null)}
                  className="px-4 py-1.5 text-xs rounded-md border border-border hover:bg-muted transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={() => {
                    setContent(JSON.stringify(presetDiffPreview.config_json, null, 2))
                    setPresetDiffPreview(null)
                    setPresetsOpen(false)
                  }}
                  className="px-4 py-1.5 text-xs rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
                >
                  Apply Preset
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Snapshot Panel (slide-in) */}
        {snapshotPanelOpen && (
          <div className="w-72 border-l border-border bg-card overflow-y-auto shrink-0 flex flex-col">
            <div className="p-3 border-b border-border flex items-center justify-between shrink-0">
              <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                Snapshots
              </h3>
              <button
                onClick={() => setSnapshotPanelOpen(false)}
                className="p-1 hover:bg-muted rounded"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </div>

            {/* Take new snapshot */}
            <div className="p-3 border-b border-border shrink-0 space-y-2">
              <input
                type="text"
                value={snapshotLabel}
                onChange={(e) => setSnapshotLabel(e.target.value)}
                placeholder="Snapshot label (optional)"
                className="w-full px-2 py-1.5 text-xs bg-muted border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <button
                onClick={handleTakeSnapshot}
                disabled={snapshotBusy}
                className={cn(
                  'w-full flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
                  'bg-primary text-primary-foreground hover:bg-primary/90',
                  'disabled:opacity-50 disabled:cursor-not-allowed'
                )}
              >
                {snapshotBusy ? <Loader2 className="h-3 w-3 animate-spin" /> : <Camera className="h-3 w-3" />}
                Take Snapshot
              </button>
            </div>

            {/* Snapshot list */}
            <div className="flex-1 overflow-y-auto p-2 space-y-2">
              {snapshots.length === 0 ? (
                <p className="text-xs text-muted-foreground text-center py-4">No snapshots yet</p>
              ) : (
                snapshots.map((snap) => (
                  <div
                    key={snap.id}
                    className="border border-border rounded-md p-2 space-y-1 bg-muted/30"
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-medium truncate max-w-[140px]" title={snap.label}>
                        {snap.label || snap.id}
                      </span>
                      <button
                        onClick={() => handleDeleteSnapshot(snap.id)}
                        className="p-0.5 hover:text-red-500 text-muted-foreground transition-colors"
                        title="Delete snapshot"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </div>
                    <div className="flex items-center gap-1 text-xs text-muted-foreground">
                      <Clock className="h-2.5 w-2.5" />
                      {new Date(snap.createdAt).toLocaleString()}
                    </div>
                    <button
                      onClick={() => handleRestoreSnapshot(snap.id)}
                      disabled={snapshotBusy}
                      className={cn(
                        'w-full flex items-center justify-center gap-1 px-2 py-1 rounded text-xs font-medium transition-colors',
                        'border border-border hover:bg-muted',
                        'disabled:opacity-50 disabled:cursor-not-allowed'
                      )}
                    >
                      <RotateCw className="h-2.5 w-2.5" />
                      Restore
                    </button>
                  </div>
                ))
              )}
            </div>
          </div>
        )}

        {/* Quick Reference Sidebar */}
        {!snapshotPanelOpen && !presetsOpen && viewMode === 'text' && (
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
                      const editor = editorRef.current
                      if (editor) {
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
        )}
      </div>
    </div>
  )
}
