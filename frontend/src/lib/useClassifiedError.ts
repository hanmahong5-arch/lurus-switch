import { useCallback, useState } from 'react'
import { classifyError, type ClassifiedError } from './errorClassifier'

/**
 * Hook for inline classified error display.
 * Returns a setter (accepts raw err), classified result, and clear function.
 *
 * Usage:
 *   const { classified, setError, clearError } = useClassifiedError()
 *   try { ... } catch (err) { setError(err) }
 *   {classified && <InlineError {...classified} onDismiss={clearError} />}
 */
export function useClassifiedError() {
  const [classified, setClassified] = useState<ClassifiedError | null>(null)

  const setError = useCallback((err: unknown) => {
    if (err == null) {
      setClassified(null)
      return
    }
    setClassified(classifyError(err))
  }, [])

  const clearError = useCallback(() => setClassified(null), [])

  return { classified, setError, clearError }
}
