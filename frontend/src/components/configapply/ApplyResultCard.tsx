import { useTranslation } from 'react-i18next'
import { AlertTriangle, CheckCircle2, RotateCcw, ExternalLink } from 'lucide-react'
import type { ApplyResult, NextStep } from './types'

// 4-element failure card per configuration-rollback-design.md Feature 4. Always
// renders 4 sections: WhatHappened / WhatExpected / RollbackDone / NextSteps,
// even on success (in which case we collapse to a single confirmation row).
//
// Wiring: every call site that invokes ApplyChangePlan MUST render this card
// when result.success is false. The eslint rule that enforces this is a
// follow-up; until then, code reviewers please check.

interface ApplyResultCardProps {
  result: ApplyResult
  onNextStep?: (step: NextStep) => void
  onDismiss?: () => void
}

export function ApplyResultCard({ result, onNextStep, onDismiss }: ApplyResultCardProps) {
  const { i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh')

  if (result.success) {
    return (
      <div className="rounded-md border border-green-500/30 bg-green-500/10 p-3 flex items-start gap-2">
        <CheckCircle2 className="h-5 w-5 text-green-500 mt-0.5 flex-shrink-0" />
        <div className="flex-1">
          <p className="text-sm font-medium text-foreground">
            {isZh ? '应用成功' : 'Applied successfully'}
          </p>
          {result.filesWritten && result.filesWritten.length > 0 && (
            <p className="text-xs text-muted-foreground mt-1">
              {(isZh ? '已写入 ' : 'Wrote ') + result.filesWritten.length +
                (isZh ? ' 个文件' : ' file(s)')}
            </p>
          )}
        </div>
        {onDismiss && (
          <button
            onClick={onDismiss}
            className="text-xs text-muted-foreground hover:text-foreground"
          >
            {isZh ? '关闭' : 'Dismiss'}
          </button>
        )}
      </div>
    )
  }

  return (
    <div className="rounded-md border border-red-500/30 bg-red-500/5 overflow-hidden">
      <div className="flex items-start gap-2 px-3 py-2 border-b border-red-500/20 bg-red-500/10">
        <AlertTriangle className="h-5 w-5 text-red-500 mt-0.5 flex-shrink-0" />
        <div className="flex-1">
          <p className="text-sm font-semibold text-foreground">
            {isZh ? '应用失败' : 'Apply failed'}
          </p>
          <p className="text-xs text-muted-foreground mt-0.5">
            {isZh ? '阶段:' : 'Phase: '}
            <span className="font-mono">{result.phase}</span>
          </p>
        </div>
      </div>

      <div className="px-3 py-2 space-y-3 text-sm">
        <Section
          label={isZh ? '发生了什么' : 'What happened'}
          body={result.whatHappened || (isZh ? '(未提供)' : '(not provided)')}
          tone="danger"
        />
        <Section
          label={isZh ? '期望是什么' : 'What was expected'}
          body={result.whatExpected || (isZh ? '(未提供)' : '(not provided)')}
        />
        <Section
          label={isZh ? '已回滚?' : 'Rolled back?'}
          body={
            result.rollbackDone
              ? (result.rollbackNote || (isZh ? '是,已回滚' : 'Yes, rolled back'))
              : (isZh
                  ? '否 — 部分文件可能已写入,请手动检查'
                  : 'No — some files may have been written, please verify')
          }
          tone={result.rollbackDone ? undefined : 'danger'}
        />
        {result.nextSteps && result.nextSteps.length > 0 && (
          <div>
            <p className="text-xs uppercase tracking-wider text-muted-foreground mb-1.5">
              {isZh ? '建议的下一步' : 'Suggested next steps'}
            </p>
            <div className="flex flex-wrap gap-1.5">
              {result.nextSteps.map((step, i) => (
                <NextStepButton
                  key={i}
                  step={step}
                  onClick={() => onNextStep?.(step)}
                />
              ))}
            </div>
          </div>
        )}
        {result.rawError && (
          <details className="text-xs text-muted-foreground">
            <summary className="cursor-pointer hover:text-foreground">
              {isZh ? '原始错误' : 'Raw error'}
            </summary>
            <pre className="mt-1 p-2 bg-muted/30 rounded font-mono whitespace-pre-wrap break-all">
              {result.rawError}
            </pre>
          </details>
        )}
      </div>

      {onDismiss && (
        <div className="border-t border-red-500/20 px-3 py-2 flex justify-end">
          <button
            onClick={onDismiss}
            className="text-xs text-muted-foreground hover:text-foreground"
          >
            {isZh ? '关闭' : 'Dismiss'}
          </button>
        </div>
      )}
    </div>
  )
}

function Section({ label, body, tone }: { label: string; body: string; tone?: 'danger' }) {
  return (
    <div>
      <p className="text-xs uppercase tracking-wider text-muted-foreground mb-0.5">{label}</p>
      <p className={'text-sm ' + (tone === 'danger' ? 'text-red-400' : 'text-foreground')}>
        {body}
      </p>
    </div>
  )
}

function NextStepButton({ step, onClick }: { step: NextStep; onClick: () => void }) {
  const handleClick = () => {
    if (step.url) {
      window.open(step.url, '_blank', 'noopener')
      return
    }
    onClick()
  }
  return (
    <button
      onClick={handleClick}
      className="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium border border-border rounded hover:bg-muted transition-colors"
    >
      <RotateCcw className="h-3 w-3" />
      <span>{step.label}</span>
      {step.url && <ExternalLink className="h-3 w-3" />}
    </button>
  )
}
