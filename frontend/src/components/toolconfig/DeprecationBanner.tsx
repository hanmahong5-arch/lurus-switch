import { useState, useEffect } from 'react'
import { AlertTriangle, ArrowRight, Loader2, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../../lib/utils'
import {
  GetGeminiDeprecationStatus,
  BuildGeminiMigrationPlan,
  ApplyGeminiMigration,
} from '../../../wailsjs/go/main/App'
import { useToastStore } from '../../stores/toastStore'

// FieldMigration describes how one Gemini config field maps to its successor.
interface FieldMigration {
  geminiField: string
  antigravityField: string
  value: string
  needsManualReview: boolean
  note?: string
}

// MigrationPlan is the plan returned by BuildGeminiMigrationPlan.
interface MigrationPlan {
  sourcePath: string
  targetPath: string
  fields: FieldMigration[]
  warnings?: string[]
}

export interface GeminiDeprecationStatus {
  isDeprecated: boolean
  eolDate: string
  daysRemaining: number
  successorTool: string
}

interface DeprecationBannerProps {
  /** The CLI tool that is deprecated. Currently only "gemini" is supported. */
  cli: string
  /** The deprecation status object (pre-fetched by the parent). */
  status: GeminiDeprecationStatus
  /** Called after a successful migration so the parent can refresh its state. */
  onMigrated?: () => void
}

/**
 * DeprecationBanner renders an inline alert above a CLI tool's config editor
 * when that tool is approaching or has passed end-of-life. For Gemini CLI it
 * provides a two-step migration flow:
 *   1. Show plan (field mapping preview)
 *   2. Apply (write successor config)
 */
export function DeprecationBanner({ cli, status, onMigrated }: DeprecationBannerProps) {
  const { t } = useTranslation()
  const toast = useToastStore((s) => s.addToast)

  const [dismissed, setDismissed] = useState(false)
  const [planOpen, setPlanOpen] = useState(false)
  const [plan, setPlan] = useState<MigrationPlan | null>(null)
  const [planLoading, setPlanLoading] = useState(false)
  const [applying, setApplying] = useState(false)

  if (dismissed) return null
  if (!status.isDeprecated) return null

  const daysLabel =
    status.daysRemaining > 0
      ? t('toolConfig.deprecation.daysLeft', { days: status.daysRemaining })
      : t('toolConfig.deprecation.eolPassed')

  const successorLabel = status.successorTool.charAt(0).toUpperCase() + status.successorTool.slice(1)

  const handleOpenPlan = async () => {
    setPlanLoading(true)
    setPlanOpen(true)
    try {
      const p = await BuildGeminiMigrationPlan()
      setPlan(p as MigrationPlan)
    } catch (err) {
      toast('error', String(err))
      setPlanOpen(false)
    } finally {
      setPlanLoading(false)
    }
  }

  const handleApply = async () => {
    setApplying(true)
    try {
      const result = await ApplyGeminiMigration()
      if (!result.success) {
        toast('error', t('toolConfig.deprecation.migrateError', { error: result.message }))
        return
      }
      toast('success', t('toolConfig.deprecation.migrateSuccess', { successor: successorLabel }))
      setPlanOpen(false)
      onMigrated?.()
    } catch (err) {
      toast('error', t('toolConfig.deprecation.migrateError', { error: String(err) }))
    } finally {
      setApplying(false)
    }
  }

  return (
    <>
      {/* Inline banner */}
      <div
        data-testid="deprecation-banner"
        className="flex items-center gap-3 px-4 py-2.5 bg-amber-500/10 border-b border-amber-500/20 text-xs shrink-0"
      >
        <AlertTriangle className="h-4 w-4 text-amber-500 shrink-0" />
        <div className="flex-1 min-w-0">
          <span className="font-medium text-amber-600">
            {t('toolConfig.deprecation.title', {
              tool: cli.charAt(0).toUpperCase() + cli.slice(1) + ' CLI',
              date: status.eolDate,
            })}
          </span>
          <span className="text-muted-foreground ml-2">{daysLabel}</span>
        </div>

        <button
          data-testid="migrate-btn"
          onClick={handleOpenPlan}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium bg-amber-500/20 hover:bg-amber-500/30 text-amber-700 transition-colors whitespace-nowrap"
        >
          <ArrowRight className="h-3.5 w-3.5" />
          {t('toolConfig.deprecation.migrateBtn', { successor: successorLabel })}
        </button>

        <button
          data-testid="dismiss-banner-btn"
          onClick={() => setDismissed(true)}
          className="p-1 hover:bg-amber-500/20 rounded text-amber-500/70 hover:text-amber-500 transition-colors"
          title="Dismiss"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>

      {/* Migration plan modal */}
      {planOpen && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
          <div className="bg-card border border-border rounded-lg w-full max-w-2xl shadow-2xl flex flex-col max-h-[80vh]">
            {/* Header */}
            <div className="flex items-center justify-between px-4 py-3 border-b border-border shrink-0">
              <h3 className="text-sm font-semibold">
                {t('toolConfig.deprecation.confirmTitle')}
              </h3>
              <button
                onClick={() => setPlanOpen(false)}
                className="p-1 hover:bg-muted rounded transition-colors"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Body */}
            <div className="flex-1 overflow-y-auto p-4 space-y-4">
              {planLoading ? (
                <div className="flex items-center justify-center gap-2 py-8">
                  <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                </div>
              ) : plan ? (
                <>
                  {/* Warnings */}
                  {(plan.warnings ?? []).length > 0 && (
                    <div className="bg-amber-500/10 border border-amber-500/20 rounded-md p-3 space-y-1">
                      <p className="text-xs font-semibold text-amber-600">
                        {t('toolConfig.deprecation.warnings')}
                      </p>
                      {plan.warnings!.map((w, i) => (
                        <p key={i} className="text-xs text-amber-700">{w}</p>
                      ))}
                    </div>
                  )}

                  {/* Paths */}
                  <div className="text-xs text-muted-foreground space-y-1">
                    <p><span className="font-medium">Source:</span> {plan.sourcePath}</p>
                    <p><span className="font-medium">Target:</span> {plan.targetPath}</p>
                  </div>

                  {/* Field mapping */}
                  {(plan.fields ?? []).length > 0 && (
                    <div>
                      <p className="text-xs font-semibold text-muted-foreground mb-2">
                        {t('toolConfig.deprecation.fieldMap')}
                      </p>
                      <div className="space-y-2">
                        {plan.fields.map((f, i) => (
                          <div
                            key={i}
                            className={cn(
                              'rounded-md border px-3 py-2 text-xs',
                              f.needsManualReview
                                ? 'border-amber-500/30 bg-amber-500/5'
                                : 'border-border bg-muted/30'
                            )}
                          >
                            <div className="flex items-center gap-2 font-mono">
                              <span className="text-muted-foreground">{f.geminiField}</span>
                              <ArrowRight className="h-3 w-3 text-muted-foreground shrink-0" />
                              <span className="text-primary">{f.antigravityField}</span>
                              {f.needsManualReview && (
                                <span className="ml-auto text-amber-600 font-sans">
                                  {t('toolConfig.deprecation.needsReview')}
                                </span>
                              )}
                            </div>
                            {f.value && (
                              <p className="text-muted-foreground mt-1 truncate">
                                Value: <span className="font-mono">{f.value}</span>
                              </p>
                            )}
                            {f.note && (
                              <p className="text-amber-600 mt-1">{f.note}</p>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </>
              ) : null}
            </div>

            {/* Footer */}
            <div className="flex justify-end gap-2 px-4 py-3 border-t border-border shrink-0">
              <button
                onClick={() => setPlanOpen(false)}
                className="px-4 py-1.5 text-xs rounded-md border border-border hover:bg-muted transition-colors"
              >
                {t('toolConfig.deprecation.confirmCancel')}
              </button>
              <button
                data-testid="apply-migration-btn"
                onClick={handleApply}
                disabled={applying || planLoading}
                className={cn(
                  'flex items-center gap-1.5 px-4 py-1.5 text-xs rounded-md transition-colors',
                  'bg-primary text-primary-foreground hover:bg-primary/90',
                  'disabled:opacity-50 disabled:cursor-not-allowed'
                )}
              >
                {applying ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <ArrowRight className="h-3.5 w-3.5" />
                )}
                {applying
                  ? t('toolConfig.deprecation.migrating')
                  : t('toolConfig.deprecation.confirmApply')}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}

/**
 * useGeminiDeprecationStatus fetches Gemini deprecation status once on mount.
 * Returns null while loading.
 */
export function useGeminiDeprecationStatus() {
  const [status, setStatus] = useState<GeminiDeprecationStatus | null>(null)

  // Fetch once — status is static (it just returns today vs EOL date).
  useEffect(() => {
    GetGeminiDeprecationStatus()
      .then(setStatus)
      .catch(() => {/* non-fatal — banner simply won't render */})
  }, [])

  return status
}
