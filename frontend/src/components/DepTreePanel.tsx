import { useEffect, useState, useCallback } from 'react'
import { ChevronRight, ChevronDown, RefreshCw, Download, Loader2, CheckCircle2, XCircle, AlertTriangle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { CheckDependencies, InstallDependency } from '../../wailsjs/go/main/App'

interface RuntimeStatus {
  id: string
  name: string
  installed: boolean
  version: string
  path: string
  required: boolean
  tools: string[]
}

interface DepCheckResult {
  runtimes: RuntimeStatus[]
  allMet: boolean
}

export function DepTreePanel() {
  const { t } = useTranslation()
  const [deps, setDeps] = useState<DepCheckResult | null>(null)
  const [loading, setLoading] = useState(true)
  const [installing, setInstalling] = useState<Record<string, boolean>>({})
  const [expanded, setExpanded] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadDeps = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const result = await CheckDependencies()
      setDeps(result)
      // Auto-expand if there are missing dependencies
      if (result && !result.allMet) {
        setExpanded(true)
      }
    } catch (err) {
      setError(`${err}`)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadDeps()
  }, [loadDeps])

  const handleInstall = async (runtimeId: string) => {
    setInstalling((prev) => ({ ...prev, [runtimeId]: true }))
    setError(null)
    try {
      const result = await InstallDependency(runtimeId)
      if (!result.success) {
        setError(result.message)
      }
      await loadDeps()
    } catch (err) {
      setError(`${err}`)
    } finally {
      setInstalling((prev) => ({ ...prev, [runtimeId]: false }))
    }
  }

  const handleInstallAll = async () => {
    if (!deps) return
    const missing = deps.runtimes.filter((r) => r.required && !r.installed)
    for (const runtime of missing) {
      await handleInstall(runtime.id)
    }
  }

  if (loading && !deps) {
    return (
      <div className="border border-border rounded-lg px-4 py-3 bg-card">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          {t('status.loading')}
        </div>
      </div>
    )
  }

  if (!deps) return null

  const missingCount = deps.runtimes.filter((r) => r.required && !r.installed).length

  return (
    <div className="border border-border rounded-lg bg-card overflow-hidden">
      {/* Header — always visible */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center justify-between px-4 py-3 text-sm font-medium hover:bg-muted/50 transition-colors"
      >
        <div className="flex items-center gap-2">
          {expanded ? (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-4 w-4 text-muted-foreground" />
          )}
          <span>{t('dashboard.deps.title')}</span>
        </div>
        <div className="flex items-center gap-2">
          {missingCount > 0 ? (
            <span className="flex items-center gap-1 text-xs text-amber-500">
              <AlertTriangle className="h-3.5 w-3.5" />
              {t('dashboard.deps.missing', { count: missingCount })}
            </span>
          ) : (
            <span className="flex items-center gap-1 text-xs text-green-500">
              <CheckCircle2 className="h-3.5 w-3.5" />
              {t('dashboard.deps.allMet')}
            </span>
          )}
        </div>
      </button>

      {/* Expanded content */}
      {expanded && (
        <div className="px-4 pb-4 space-y-3 border-t border-border pt-3">
          {error && (
            <div className="text-xs text-red-500 bg-red-500/10 rounded px-3 py-2">
              {error}
            </div>
          )}

          {/* Runtime tree */}
          <div className="space-y-2">
            {deps.runtimes.map((runtime) => {
              if (runtime.id === 'none') {
                // Standalone section
                return (
                  <div key={runtime.id} className="flex items-center gap-2 text-xs text-muted-foreground pl-2">
                    <span className="font-mono">{'└─'}</span>
                    <span>{t('dashboard.deps.standalone')}</span>
                    <span className="text-foreground/70">
                      {'─── '}
                      {runtime.tools.map((tool) => toolLabel(tool)).join(', ')}
                    </span>
                  </div>
                )
              }

              const isNodeJS = runtime.id === 'nodejs'
              const isBun = runtime.id === 'bun'
              // Bun is blocked if Node.js is not installed
              const nodeInstalled = deps.runtimes.find((r) => r.id === 'nodejs')?.installed ?? true
              const blocked = isBun && !nodeInstalled

              return (
                <div
                  key={runtime.id}
                  className={cn(
                    'flex items-center gap-2 text-xs',
                    isNodeJS ? 'pl-2' : 'pl-8'
                  )}
                >
                  <span className="font-mono text-muted-foreground">
                    {isNodeJS ? '┌─' : '└─'}
                  </span>
                  <span className="font-medium">{runtime.name}</span>
                  <span className="font-mono text-muted-foreground">───</span>

                  {runtime.installed ? (
                    <>
                      <span className="text-muted-foreground">
                        v{runtime.version || '?'}
                      </span>
                      <CheckCircle2 className="h-3.5 w-3.5 text-green-500" />
                    </>
                  ) : blocked ? (
                    <>
                      <span className="text-muted-foreground">
                        {t('dashboard.deps.notFound')}
                      </span>
                      <XCircle className="h-3.5 w-3.5 text-muted-foreground" />
                      <span className="text-muted-foreground italic">
                        ({t('dashboard.deps.blocked', { dep: 'Node.js' })})
                      </span>
                    </>
                  ) : (
                    <>
                      <span className="text-red-500">{t('dashboard.deps.notFound')}</span>
                      <XCircle className="h-3.5 w-3.5 text-red-500" />
                      <button
                        onClick={() => handleInstall(runtime.id)}
                        disabled={installing[runtime.id]}
                        className={cn(
                          'flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium transition-colors',
                          'bg-primary text-primary-foreground hover:bg-primary/90',
                          'disabled:opacity-50 disabled:cursor-not-allowed'
                        )}
                      >
                        {installing[runtime.id] ? (
                          <Loader2 className="h-3 w-3 animate-spin" />
                        ) : (
                          <Download className="h-3 w-3" />
                        )}
                        {t('dashboard.deps.install')}
                      </button>
                    </>
                  )}

                  {runtime.installed && runtime.tools.length > 0 && (
                    <span className="text-muted-foreground ml-1">
                      {'─── '}
                      {runtime.tools.map((tool) => toolLabel(tool)).join(', ')}
                    </span>
                  )}
                </div>
              )
            })}
          </div>

          {/* Bottom actions */}
          <div className="flex items-center gap-2 pt-1">
            <button
              onClick={loadDeps}
              disabled={loading}
              className={cn(
                'flex items-center gap-1 px-2.5 py-1 rounded text-xs font-medium transition-colors',
                'border border-border hover:bg-muted',
                'disabled:opacity-50 disabled:cursor-not-allowed'
              )}
            >
              {loading ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                <RefreshCw className="h-3 w-3" />
              )}
              {t('dashboard.refresh')}
            </button>

            {missingCount > 0 && (
              <button
                onClick={handleInstallAll}
                disabled={Object.values(installing).some(Boolean)}
                className={cn(
                  'flex items-center gap-1 px-2.5 py-1 rounded text-xs font-medium transition-colors',
                  'bg-primary text-primary-foreground hover:bg-primary/90',
                  'disabled:opacity-50 disabled:cursor-not-allowed'
                )}
              >
                <Download className="h-3 w-3" />
                {t('dashboard.deps.installAll')}
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

// Map tool name to display label
const toolLabels: Record<string, string> = {
  claude: 'Claude Code',
  codex: 'Codex',
  gemini: 'Gemini CLI',
  picoclaw: 'PicoClaw',
  nullclaw: 'NullClaw',
  zeroclaw: 'ZeroClaw',
  openclaw: 'OpenClaw',
}

function toolLabel(name: string): string {
  return toolLabels[name] || name
}
