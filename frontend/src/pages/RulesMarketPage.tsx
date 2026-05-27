import { useEffect, useState, useCallback } from 'react'
import { Search, Download, BookMarked, Globe } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { Button, Modal } from '../components/ui'
import { useToastStore } from '../stores/toastStore'
import { RulesMarketList, RulesMarketWrite } from '../../wailsjs/go/main/App'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface RuleTemplate {
  id: string
  name: string
  category: string
  framework: string
  description: string
  format: string
  source_url: string
  content: string
}

type TargetFormat = 'agents_md' | 'claude_md' | 'cursorrules'

const TARGET_FORMATS: TargetFormat[] = ['agents_md', 'claude_md', 'cursorrules']

const CATEGORIES = ['all', 'framework', 'language', 'custom', 'tool', 'testing'] as const
type Category = typeof CATEGORIES[number]

// ---------------------------------------------------------------------------
// Install modal
// ---------------------------------------------------------------------------

interface InstallModalProps {
  template: RuleTemplate | null
  onClose: () => void
  onInstall: (template: RuleTemplate, format: TargetFormat, projectDir: string, overwrite: boolean) => Promise<void>
  installing: boolean
}

export function InstallModal({ template, onClose, onInstall, installing }: InstallModalProps) {
  const { t } = useTranslation()
  const [format, setFormat] = useState<TargetFormat>('agents_md')
  const [projectDir, setProjectDir] = useState('')
  const [overwrite, setOverwrite] = useState(false)

  if (!template) return null

  return (
    <Modal
      open={!!template}
      onClose={onClose}
      title={t('rulesmarket.modal.title', 'Install rule template')}
      icon={Download}
      size="md"
      footer={
        <div className="flex items-center justify-end gap-2 p-4 border-t border-border">
          <Button variant="ghost" size="sm" onClick={onClose} disabled={installing}>
            {t('rulesmarket.modal.cancel', 'Cancel')}
          </Button>
          <Button
            variant="primary"
            size="sm"
            loading={installing}
            disabled={!projectDir.trim() || installing}
            onClick={() => onInstall(template, format, projectDir.trim(), overwrite)}
          >
            {t('rulesmarket.modal.confirm', 'Install')}
          </Button>
        </div>
      }
    >
      <div className="p-4 space-y-4">
        <p className="text-xs text-muted-foreground">
          {t('rulesmarket.modal.desc', 'Select the target format and the project directory where the rule file will be written.')}
        </p>

        {/* Target format selector */}
        <div>
          <label className="block text-xs font-medium text-foreground mb-1.5">
            {t('rulesmarket.targetFormat', 'Target format')}
          </label>
          <div className="flex gap-2 flex-wrap">
            {TARGET_FORMATS.map((f) => (
              <button
                key={f}
                onClick={() => setFormat(f)}
                className={cn(
                  'px-3 py-1 rounded-md text-xs border transition-colors',
                  format === f
                    ? 'bg-primary/15 border-primary text-primary font-medium'
                    : 'border-border text-muted-foreground hover:border-primary/50 hover:text-foreground',
                )}
              >
                {t(`rulesmarket.formatLabel.${f}`, f)}
              </button>
            ))}
          </div>
        </div>

        {/* Project directory input */}
        <div>
          <label className="block text-xs font-medium text-foreground mb-1.5">
            {t('rulesmarket.projectDir', 'Project directory')}
          </label>
          <input
            type="text"
            value={projectDir}
            onChange={(e) => setProjectDir(e.target.value)}
            placeholder="/path/to/your/project"
            className={cn(
              'w-full h-8 px-3 text-xs rounded-md border border-border',
              'bg-input text-foreground placeholder:text-muted-foreground',
              'focus:outline-none focus:ring-1 focus:ring-primary',
            )}
          />
        </div>

        {/* Overwrite toggle */}
        <label className="flex items-center gap-2 cursor-pointer select-none">
          <input
            type="checkbox"
            checked={overwrite}
            onChange={(e) => setOverwrite(e.target.checked)}
            className="h-3.5 w-3.5 rounded accent-primary"
          />
          <span className="text-xs text-muted-foreground">
            {overwrite
              ? t('rulesmarket.modal.overwrite', 'Replace existing file (overwrite)')
              : t('rulesmarket.modal.append', 'Append to existing file (default)')}
          </span>
        </label>
      </div>
    </Modal>
  )
}

// ---------------------------------------------------------------------------
// Template card
// ---------------------------------------------------------------------------

interface TemplateCardProps {
  template: RuleTemplate
  onInstall: (template: RuleTemplate) => void
}

export function TemplateCard({ template, onInstall }: TemplateCardProps) {
  const { t } = useTranslation()
  const isBuiltin = !template.source_url
  return (
    <div
      className={cn(
        'group flex flex-col gap-2 p-3 rounded-lg border border-border',
        'bg-card hover:border-primary/40 transition-colors',
      )}
      data-testid="template-card"
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-foreground truncate">{template.name}</p>
          <p className="text-[11px] text-muted-foreground mt-0.5 truncate">{template.framework}</p>
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-muted text-muted-foreground font-mono uppercase tracking-wide">
            {template.category}
          </span>
          {isBuiltin ? (
            <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-primary/10 text-primary">
              {t('rulesmarket.builtin', 'Built-in')}
            </span>
          ) : (
            <Globe className="h-3 w-3 text-muted-foreground" aria-label="remote" />
          )}
        </div>
      </div>
      <p className="text-xs text-muted-foreground line-clamp-2 leading-relaxed">
        {template.description}
      </p>
      <div className="flex items-center justify-between mt-1">
        <span className="text-[10px] font-mono text-muted-foreground">
          {t(`rulesmarket.formatLabel.${template.format}`, template.format)}
        </span>
        <Button
          variant="secondary"
          size="sm"
          icon={<Download className="h-3 w-3" />}
          onClick={() => onInstall(template)}
        >
          {t('rulesmarket.install', 'Install')}
        </Button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------

export function RulesMarketPage() {
  const { t } = useTranslation()
  const addToast = useToastStore((s) => s.addToast)

  const [templates, setTemplates] = useState<RuleTemplate[]>([])
  const [loading, setLoading] = useState(true)
  const [category, setCategory] = useState<Category>('all')
  const [search, setSearch] = useState('')
  const [installing, setInstalling] = useState(false)
  const [selectedTemplate, setSelectedTemplate] = useState<RuleTemplate | null>(null)

  const loadTemplates = useCallback(async () => {
    setLoading(true)
    try {
      const list = await RulesMarketList()
      setTemplates(list || [])
    } catch {
      addToast('error', t('rulesmarket.installFailed', 'Install failed'))
    } finally {
      setLoading(false)
    }
  }, [addToast, t])

  useEffect(() => { loadTemplates() }, [loadTemplates])

  const filtered = templates.filter((tmpl) => {
    const matchCat = category === 'all' || tmpl.category === category
    const q = search.toLowerCase()
    const matchSearch =
      q === '' ||
      tmpl.name.toLowerCase().includes(q) ||
      tmpl.framework.toLowerCase().includes(q) ||
      tmpl.description.toLowerCase().includes(q)
    return matchCat && matchSearch
  })

  const handleInstall = async (
    template: RuleTemplate,
    format: TargetFormat,
    projectDir: string,
    overwrite: boolean,
  ) => {
    setInstalling(true)
    try {
      const result = await RulesMarketWrite(projectDir, template as any, format, overwrite)
      if (result && result.success === false) {
        addToast('error', result.message || t('rulesmarket.installFailed', 'Install failed'))
      } else {
        addToast('success', t('rulesmarket.installed', 'Rule installed'))
        setSelectedTemplate(null)
      }
    } catch (err) {
      addToast('error', err instanceof Error ? err.message : t('rulesmarket.installFailed', 'Install failed'))
    } finally {
      setInstalling(false)
    }
  }

  return (
    <div className="h-full flex overflow-hidden">
      {/* Sidebar: categories */}
      <div className="w-44 border-r border-border bg-card-recessed flex flex-col shrink-0">
        <div className="p-3 border-b border-border">
          <h2 className="text-sm font-semibold flex items-center gap-2">
            <BookMarked className="h-4 w-4 text-primary" />
            {t('rulesmarket.title', 'Rules Market')}
          </h2>
        </div>
        <nav className="p-2 space-y-0.5">
          {CATEGORIES.map((cat) => {
            const active = category === cat
            return (
              <button
                key={cat}
                onClick={() => setCategory(cat)}
                className={cn(
                  'w-full text-left px-3 py-1.5 rounded-md transition-all duration-150 text-sm',
                  active
                    ? 'bg-primary/15 text-primary border-l-2 border-l-primary font-mono text-xs tracking-[0.06em]'
                    : 'text-muted-foreground hover:bg-muted hover:text-foreground',
                )}
              >
                {active
                  ? `[ ${t(`rulesmarket.category.${cat}`, cat).toUpperCase()} ]`
                  : t(`rulesmarket.category.${cat}`, cat)}
              </button>
            )
          })}
        </nav>
      </div>

      {/* Main content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Search bar */}
        <div className="p-3 border-b border-border shrink-0">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('rulesmarket.search', 'Search templates…')}
              className={cn(
                'w-full h-8 pl-8 pr-3 text-xs rounded-md border border-border',
                'bg-input text-foreground placeholder:text-muted-foreground',
                'focus:outline-none focus:ring-1 focus:ring-primary',
              )}
            />
          </div>
        </div>

        {/* Template grid */}
        <div className="flex-1 overflow-y-auto p-3">
          {loading ? (
            <div className="flex items-center justify-center h-32 text-xs text-muted-foreground">
              {t('rulesmarket.loading', 'Loading templates…')}
            </div>
          ) : filtered.length === 0 ? (
            <div className="flex items-center justify-center h-32 text-xs text-muted-foreground">
              {t('rulesmarket.noResults', 'No templates match your search.')}
            </div>
          ) : (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-2.5">
              {filtered.map((tmpl) => (
                <TemplateCard
                  key={tmpl.id}
                  template={tmpl}
                  onInstall={(t) => setSelectedTemplate(t)}
                />
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Install modal */}
      <InstallModal
        template={selectedTemplate}
        onClose={() => setSelectedTemplate(null)}
        onInstall={handleInstall}
        installing={installing}
      />
    </div>
  )
}
