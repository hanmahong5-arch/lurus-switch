import { useEffect, useState, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Save, FolderOpen, FileText, Loader2, RefreshCw, Plus, FolderSearch } from 'lucide-react'
import { cn } from '../lib/utils'
import { errorToast } from '../lib/errorToast'
import { useToastStore } from '../stores/toastStore'
import { Button, Card, Modal } from '../components/ui'
import { GetContextFile, SaveContextFile, OpenFolderAndScanContext } from '../../wailsjs/go/main/App'

interface ContextFile {
  tool: string
  scope: string
  path: string
  content: string
  exists: boolean
}

type Tool = 'claude' | 'gemini' | 'picoclaw' | 'nullclaw'
type Scope = 'global' | 'project'

const TOOLS: { id: Tool; label: string; color: string }[] = [
  { id: 'claude', label: 'Claude', color: 'text-orange-400' },
  { id: 'gemini', label: 'Gemini', color: 'text-blue-400' },
  { id: 'picoclaw', label: 'PicoClaw', color: 'text-green-400' },
  { id: 'nullclaw', label: 'NullClaw', color: 'text-cyan-400' },
]

const TOOL_FILE_NAMES: Record<Tool, Record<Scope, string>> = {
  claude: { global: '~/.claude/CLAUDE.md', project: '<项目目录>/CLAUDE.md' },
  gemini: { global: '~/.gemini/GEMINI.md', project: '<项目目录>/.gemini/GEMINI.md' },
  picoclaw: { global: '~/.picoclaw/SYSTEM.md', project: '~/.picoclaw/SYSTEM.md' },
  nullclaw: { global: '~/.nullclaw/SYSTEM.md', project: '~/.nullclaw/SYSTEM.md' },
}

export function DocumentPage() {
  const { t } = useTranslation()
  const [activeTool, setActiveTool] = useState<Tool>('claude')
  const [activeScope, setActiveScope] = useState<Scope>('global')
  const [contextFile, setContextFile] = useState<ContextFile | null>(null)
  const [content, setContent] = useState('')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const toast = useToastStore((s) => s.addToast)
  const [scannedFiles, setScannedFiles] = useState<ContextFile[]>([])
  const [showScanned, setShowScanned] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const loadFile = useCallback(async (tool: Tool, scope: Scope) => {
    setLoading(true)
    try {
      const f = await GetContextFile(tool, scope)
      setContextFile(f)
      setContent(f?.content || '')
    } catch (err) {
      errorToast(toast, err, { currentPage: 'workspace' })
    } finally {
      setLoading(false)
    }
  }, [toast])

  useEffect(() => {
    loadFile(activeTool, activeScope)
  }, [activeTool, activeScope, loadFile])

  const handleSave = async () => {
    if (!contextFile) return
    setSaving(true)
    try {
      await SaveContextFile({
        ...contextFile,
        content,
      })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
      await loadFile(activeTool, activeScope)
      toast('success', 'Context file saved')
    } catch (err) {
      errorToast(toast, err, { currentPage: 'workspace', retry: () => handleSave() })
    } finally {
      setSaving(false)
    }
  }

  const handleScanFolder = async () => {
    try {
      const files = await OpenFolderAndScanContext()
      setScannedFiles(files || [])
      setShowScanned(true)
    } catch (err) {
      errorToast(toast, err, { currentPage: 'workspace' })
    }
  }

  const handleLoadScanned = (f: ContextFile) => {
    setContent(f.content)
    setContextFile(f)
    setShowScanned(false)
  }

  const insertTemplate = (template: string) => {
    const textarea = textareaRef.current
    if (!textarea) return
    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const newContent = content.slice(0, start) + template + content.slice(end)
    setContent(newContent)
    setTimeout(() => {
      textarea.selectionStart = start + template.length
      textarea.selectionEnd = start + template.length
      textarea.focus()
    }, 0)
  }

  const TEMPLATES: { label: string; content: string }[] = [
    {
      label: 'contextDoc.templates.codeStandards',
      content: '\n## 代码规范\n\n- 使用 TypeScript strict 模式\n- 函数命名采用 camelCase\n- 组件命名采用 PascalCase\n',
    },
    {
      label: 'contextDoc.templates.projectStructure',
      content: '\n## 项目结构\n\n```\nsrc/\n  components/  # 可复用组件\n  pages/       # 页面组件\n  stores/      # 状态管理\n```\n',
    },
    {
      label: 'contextDoc.templates.forbidden',
      content: '\n## 禁止事项\n\n- 禁止硬编码配置值\n- 禁止忽略错误处理\n- 禁止提交测试文件\n',
    },
  ]

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* Tool Tabs */}
      <div className="border-b border-border bg-muted/20 px-4 pt-3 shrink-0">
        <div className="flex items-center gap-1 mb-3">
          <FileText className="h-4 w-4 text-teal-400 mr-1" />
          <span className="text-sm font-semibold">{t('contextDoc.title')}</span>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex gap-1">
            {TOOLS.map((t) => (
              <button
                key={t.id}
                onClick={() => setActiveTool(t.id)}
                className={cn(
                  'px-3 py-1.5 text-xs font-medium rounded-t-md border border-b-0 transition-colors',
                  activeTool === t.id
                    ? 'border-border bg-background text-foreground'
                    : 'border-transparent text-muted-foreground hover:text-foreground'
                )}
              >
                <span className={activeTool === t.id ? t.color : ''}>{t.label}</span>
              </button>
            ))}
          </div>
          <div className="ml-auto flex items-center gap-2">
            <div className="flex rounded-md border border-border overflow-hidden text-xs">
              {(['global', 'project'] as Scope[]).map((s) => (
                <button
                  key={s}
                  onClick={() => setActiveScope(s)}
                  className={cn(
                    'px-3 py-1.5 transition-colors',
                    activeScope === s
                      ? 'bg-primary text-primary-foreground'
                      : 'text-muted-foreground hover:bg-muted'
                  )}
                >
                  {s === 'global' ? t('contextDoc.scopeGlobal') : t('contextDoc.scopeProject')}
                </button>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Toolbar */}
      <div className="flex items-center gap-2 px-4 py-2 border-b border-border bg-muted/10 shrink-0">
        <span className="text-xs text-muted-foreground font-mono truncate flex-1 tabular-nums">
          {contextFile ? contextFile.path : '...'}
          {contextFile && !contextFile.exists && (
            <span className="ml-2 text-amber-400">▸ {t('contextDoc.fileNotExist')}</span>
          )}
        </span>
        <div className="flex items-center gap-1">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => loadFile(activeTool, activeScope)}
            disabled={loading}
            loading={loading}
            icon={!loading ? <RefreshCw className="h-3 w-3" /> : undefined}
          >
            {t('contextDoc.refresh')}
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={handleScanFolder}
            icon={<FolderOpen className="h-3 w-3" />}
          >
            {t('contextDoc.scanDir')}
          </Button>
          <Button
            size="sm"
            onClick={handleSave}
            disabled={saving || loading}
            loading={saving}
            icon={!saving ? <Save className="h-3 w-3" /> : undefined}
            className={saved ? 'bg-emerald-500/20 text-emerald-400 ring-emerald-500/40 hover:bg-emerald-500/25' : ''}
          >
            {saved ? t('contextDoc.saved') : t('contextDoc.save')}
          </Button>
        </div>
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* Editor */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {loading ? (
            <div className="flex-1 flex items-center justify-center">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : (
            <textarea
              ref={textareaRef}
              value={content}
              onChange={(e) => setContent(e.target.value)}
              className="flex-1 p-4 font-mono text-sm bg-background text-foreground resize-none focus:outline-none"
              placeholder={t('contextDoc.editorPlaceholder', { tool: activeTool, file: activeTool === 'claude' ? 'CLAUDE' : activeTool === 'gemini' ? 'GEMINI' : 'SYSTEM' })}
              spellCheck={false}
            />
          )}
        </div>

        {/* Right Panel */}
        <div className="w-48 border-l border-border flex flex-col shrink-0 bg-card-recessed">
          <div className="p-3 border-b border-border">
            <p className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground">
              [ {t('contextDoc.quickInsert').toUpperCase()} ]
            </p>
          </div>
          <div className="flex-1 overflow-y-auto p-2 space-y-1">
            {TEMPLATES.map((tpl) => (
              <button
                key={tpl.label}
                onClick={() => insertTemplate(tpl.content)}
                className="w-full flex items-center gap-2 px-2 py-1.5 text-xs text-left rounded hover:bg-muted transition-colors"
              >
                <Plus className="h-3 w-3 text-primary shrink-0" />
                {t(tpl.label)}
              </button>
            ))}
          </div>
          <div className="p-3 border-t border-border">
            <div className="text-xs text-muted-foreground space-y-1">
              <p className="font-mono text-[10px] uppercase tracking-[0.18em]">[ {t('contextDoc.filePath').toUpperCase()} ]</p>
              <p className="font-mono text-xs break-all opacity-70 tabular-nums">
                {TOOL_FILE_NAMES[activeTool][activeScope]}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Scanned files modal */}
      <Modal
        open={showScanned}
        onClose={() => setShowScanned(false)}
        title={t('contextDoc.scannedFiles')}
        icon={FolderSearch}
        size="md"
        footer={
          <Button variant="secondary" size="sm" onClick={() => setShowScanned(false)}>
            {t('contextDoc.close')}
          </Button>
        }
      >
        {scannedFiles.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('contextDoc.noFilesFound')}</p>
        ) : (
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {scannedFiles.map((f, i) => (
              <Card
                key={i}
                as="button"
                variant="default"
                onClick={() => handleLoadScanned(f)}
                className="w-full text-left px-3 py-2 text-sm hover:bg-muted"
              >
                <p className="font-medium">
                  <span className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground mr-1.5">[ {f.tool.toUpperCase()} ]</span>
                  {f.scope}
                </p>
                <p className="text-xs text-muted-foreground font-mono truncate tabular-nums">{f.path}</p>
              </Card>
            ))}
          </div>
        )}
      </Modal>
    </div>
  )
}
