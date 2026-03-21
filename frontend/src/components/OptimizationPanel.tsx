import { useEffect, useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Loader2, Wrench, Download, ArrowUpCircle, Play, Zap,
  CheckCircle2, XCircle, SkipForward, AlertTriangle,
} from 'lucide-react'
import { cn } from '../lib/utils'
import { FeatureGate } from './HelpTip'

interface Optimization {
  id: string
  category: string
  priority: number
  title: string
  description: string
  action: string
  target: string
  autoFixable: boolean
  status: string
  error?: string
}

interface AnalysisResult {
  optimizations: Optimization[]
  fixableCount: number
  totalCount: number
}

interface FixResult {
  id: string
  status: string
  message?: string
  error?: string
}

// Lazy-resolve Wails bindings
let _analyzeOpt: ((  ) => Promise<AnalysisResult>) | null | undefined = undefined
let _applyOpt: ((id: string) => Promise<FixResult>) | null | undefined = undefined
let _applyAllOpt: (() => Promise<FixResult[]>) | null | undefined = undefined

async function resolveBindings() {
  if (_analyzeOpt !== undefined) return
  try {
    const mod = await import('../../wailsjs/go/main/App')
    _analyzeOpt = ('AnalyzeOptimizations' in mod) ? (mod as any).AnalyzeOptimizations : null
    _applyOpt = ('ApplyOptimization' in mod) ? (mod as any).ApplyOptimization : null
    _applyAllOpt = ('ApplyAllOptimizations' in mod) ? (mod as any).ApplyAllOptimizations : null
  } catch {
    _analyzeOpt = null
    _applyOpt = null
    _applyAllOpt = null
  }
}

const ACTION_ICONS: Record<string, React.ComponentType<{ className?: string }>> = {
  'install-runtime': Download,
  'install-tool': Download,
  'update-tool': ArrowUpCircle,
  'start-gateway': Play,
  'connect-gateway': Zap,
  'fix-config': Wrench,
  'install-git': Download,
}

const STATUS_ICONS: Record<string, React.ComponentType<{ className?: string }>> = {
  success: CheckCircle2,
  failed: XCircle,
  skipped: SkipForward,
}

const PRIORITY_STYLES: Record<number, string> = {
  1: 'border-red-500/30 bg-red-500/5',
  2: 'border-yellow-500/30 bg-yellow-500/5',
  3: 'border-border bg-card',
}

interface Props {
  onRefresh?: () => void
}

export function OptimizationPanel({ onRefresh }: Props) {
  const { t } = useTranslation()
  const [analysis, setAnalysis] = useState<AnalysisResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [applying, setApplying] = useState<Record<string, boolean>>({})
  const [applyingAll, setApplyingAll] = useState(false)
  const [results, setResults] = useState<Record<string, FixResult>>({})

  const loadAnalysis = useCallback(async () => {
    await resolveBindings()
    if (!_analyzeOpt) return
    setLoading(true)
    try {
      const result = await _analyzeOpt()
      setAnalysis(result)
      setResults({})
    } catch {
      // Non-critical
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadAnalysis()
  }, [loadAnalysis])

  const handleApply = async (opt: Optimization) => {
    if (!_applyOpt) return
    setApplying(prev => ({ ...prev, [opt.id]: true }))
    try {
      const result = await _applyOpt(opt.id)
      setResults(prev => ({ ...prev, [opt.id]: result }))
      // Reload after applying
      await loadAnalysis()
      onRefresh?.()
    } catch {
      setResults(prev => ({
        ...prev,
        [opt.id]: { id: opt.id, status: 'failed', error: 'unexpected error' },
      }))
    } finally {
      setApplying(prev => ({ ...prev, [opt.id]: false }))
    }
  }

  const handleApplyAll = async () => {
    if (!_applyAllOpt) return
    setApplyingAll(true)
    try {
      const fixResults = await _applyAllOpt()
      const resultMap: Record<string, FixResult> = {}
      for (const r of fixResults) resultMap[r.id] = r
      setResults(resultMap)
      await loadAnalysis()
      onRefresh?.()
    } catch {
      // Non-critical
    } finally {
      setApplyingAll(false)
    }
  }

  if (loading && !analysis) {
    return (
      <div className="flex items-center justify-center p-4">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!analysis || analysis.totalCount === 0) return null

  const allResolved = analysis.optimizations.every(
    opt => results[opt.id]?.status === 'success' || results[opt.id]?.status === 'skipped'
  )

  if (allResolved && analysis.totalCount > 0) {
    return (
      <div className="rounded-xl border border-green-500/30 bg-green-500/5 p-4 flex items-center gap-3">
        <CheckCircle2 className="h-5 w-5 text-green-500 flex-shrink-0" />
        <span className="text-sm font-medium text-green-600 dark:text-green-400">
          {t('optimizer.allFixed')}
        </span>
      </div>
    )
  }

  return (
    <FeatureGate minLevel="regular">
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Wrench className="h-4 w-4 text-muted-foreground" />
            <h3 className="text-sm font-semibold">{t('optimizer.title')}</h3>
            {analysis.fixableCount > 0 && (
              <span className="text-xs text-muted-foreground">
                {t('optimizer.fixableCount', { count: analysis.fixableCount })}
              </span>
            )}
          </div>
          {analysis.fixableCount > 1 && (
            <button
              onClick={handleApplyAll}
              disabled={applyingAll}
              className={cn(
                'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
                'bg-primary text-primary-foreground hover:bg-primary/90',
                'disabled:opacity-50 disabled:cursor-not-allowed'
              )}
            >
              {applyingAll ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Zap className="h-3.5 w-3.5" />}
              {t('optimizer.applyAll')}
            </button>
          )}
        </div>

        <div className="space-y-2">
          {analysis.optimizations.map((opt) => {
            const ActionIcon = ACTION_ICONS[opt.action] || Wrench
            const result = results[opt.id]
            const isApplying = applying[opt.id] || false
            const colorClass = PRIORITY_STYLES[opt.priority] || PRIORITY_STYLES[3]

            // Determine display status
            const StatusIcon = result ? STATUS_ICONS[result.status] : null

            return (
              <div
                key={opt.id}
                className={cn(
                  'flex items-center gap-3 rounded-lg border p-3',
                  result?.status === 'success' ? 'border-green-500/30 bg-green-500/5 opacity-60' : colorClass,
                )}
              >
                {StatusIcon ? (
                  <StatusIcon className={cn(
                    'h-4 w-4 flex-shrink-0',
                    result?.status === 'success' ? 'text-green-500' :
                    result?.status === 'failed' ? 'text-red-500' : 'text-muted-foreground'
                  )} />
                ) : (
                  <ActionIcon className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                )}
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{opt.title}</p>
                  <p className="text-xs text-muted-foreground truncate">{opt.description}</p>
                  {result?.error && (
                    <p className="text-xs text-red-500 mt-0.5 truncate">{result.error}</p>
                  )}
                </div>
                {opt.autoFixable && !result?.status && (
                  <button
                    onClick={() => handleApply(opt)}
                    disabled={isApplying || applyingAll}
                    className={cn(
                      'px-3 py-1 rounded-md text-xs font-medium transition-colors flex-shrink-0',
                      'bg-primary text-primary-foreground hover:bg-primary/90',
                      'disabled:opacity-50 disabled:cursor-not-allowed'
                    )}
                  >
                    {isApplying ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : t(`home.actionLabel.${opt.action}`)}
                  </button>
                )}
                {!opt.autoFixable && !result?.status && (
                  <span className="text-xs text-muted-foreground flex-shrink-0 flex items-center gap-1">
                    <AlertTriangle className="h-3 w-3" />
                  </span>
                )}
              </div>
            )
          })}
        </div>
      </div>
    </FeatureGate>
  )
}
