import { useTranslation } from 'react-i18next'
import { Loader2, ChevronDown, ChevronUp } from 'lucide-react'
import { useState } from 'react'
import { cn } from '../lib/utils'
import { HelpTip } from './HelpTip'
import type { ScoreReport } from '../stores/homeStore'

function scoreColor(score: number): string {
  if (score < 30) return '#ef4444'  // red-500
  if (score <= 70) return '#eab308' // yellow-500
  return '#22c55e'                   // green-500
}

function scoreColorClass(score: number): string {
  if (score < 30) return 'text-red-500'
  if (score <= 70) return 'text-yellow-500'
  return 'text-green-500'
}

function scoreBgClass(score: number): string {
  if (score < 30) return 'bg-red-500/10'
  if (score <= 70) return 'bg-yellow-500/10'
  return 'bg-green-500/10'
}

interface Props {
  report: ScoreReport | null
  loading: boolean
  onOptimize: () => void
  optimizing: boolean
}

export function HealthScoreGauge({ report, loading, onOptimize, optimizing }: Props) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)

  const score = report?.totalScore ?? 0
  const maxScore = report?.maxScore ?? 100

  // SVG circle parameters
  const size = 160
  const strokeWidth = 10
  const radius = (size - strokeWidth) / 2
  const circumference = 2 * Math.PI * radius
  const progress = maxScore > 0 ? score / maxScore : 0
  const dashOffset = circumference * (1 - progress)
  const color = scoreColor(score)

  if (loading && !report) {
    return (
      <div className="flex items-center justify-center p-8">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className={cn('rounded-xl border border-border p-6', scoreBgClass(score))}>
      <div className="flex items-center gap-8">
        {/* Score Ring */}
        <div className="relative flex-shrink-0">
          <svg width={size} height={size} className="transform -rotate-90">
            {/* Background circle */}
            <circle
              cx={size / 2}
              cy={size / 2}
              r={radius}
              fill="none"
              stroke="currentColor"
              strokeWidth={strokeWidth}
              className="text-muted/30"
            />
            {/* Progress circle */}
            <circle
              cx={size / 2}
              cy={size / 2}
              r={radius}
              fill="none"
              stroke={color}
              strokeWidth={strokeWidth}
              strokeLinecap="round"
              strokeDasharray={circumference}
              strokeDashoffset={dashOffset}
              className="transition-all duration-1000 ease-out"
            />
          </svg>
          {/* Center text */}
          <div className="absolute inset-0 flex flex-col items-center justify-center">
            <span className={cn('text-3xl font-bold', scoreColorClass(score))}>
              {score}
            </span>
            <span className="text-xs text-muted-foreground">{t('home.healthScore')}</span>
          </div>
        </div>

        {/* Action area */}
        <div className="flex-1 space-y-3">
          <div>
            <h3 className="text-sm font-semibold">
              {score >= 70
                ? t('home.envHealthy')
                : score >= 30
                  ? t('home.envNeedsWork')
                  : t('home.envCritical')}
              <HelpTip
                titleKey="help.healthScore.title"
                bodyKey="help.healthScore.body"
                showFor={['beginner', 'regular']}
                placement="bottom"
              />
            </h3>
            <p className="text-xs text-muted-foreground mt-1">
              {t('home.healthDesc')}
            </p>
          </div>

          <button
            onClick={onOptimize}
            disabled={optimizing || loading}
            className={cn(
              'flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors',
              'bg-primary text-primary-foreground hover:bg-primary/90',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            {optimizing ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : null}
            {t('home.optimize')}
          </button>
        </div>
      </div>

      {/* Category breakdown toggle */}
      {report && report.categories.length > 0 && (
        <div className="mt-4">
          <button
            onClick={() => setExpanded(!expanded)}
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            {expanded ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
            {t('home.scoreDetails')}
          </button>

          {expanded && (
            <div className="mt-3 space-y-2">
              {report.categories.map((cat) => (
                <div key={cat.category} className="space-y-1">
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-muted-foreground">{t(`home.cat.${cat.category}`)}</span>
                    <span className={cn('font-medium', scoreColorClass(cat.max > 0 ? (cat.score / cat.max) * 100 : 0))}>
                      {cat.score}/{cat.max}
                    </span>
                  </div>
                  <div className="w-full h-1.5 bg-muted rounded-full overflow-hidden">
                    <div
                      className="h-full rounded-full transition-all duration-500"
                      style={{
                        width: `${cat.max > 0 ? (cat.score / cat.max) * 100 : 0}%`,
                        backgroundColor: scoreColor(cat.max > 0 ? (cat.score / cat.max) * 100 : 0),
                      }}
                    />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
