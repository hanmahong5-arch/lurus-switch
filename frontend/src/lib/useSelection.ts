import { useEffect, useState, useCallback } from 'react'

// Tracks the currently selected text across the document, so a header
// "Copy" button can light up only when there is actually something to
// copy. Listens to selectionchange + focusin/out (selections inside
// inputs/textareas don't show up in window.getSelection on all browsers).
export function useSelection() {
  const [text, setText] = useState('')

  const compute = useCallback(() => {
    // Native input/textarea selections: pull from the focused element.
    const ae = document.activeElement as HTMLInputElement | HTMLTextAreaElement | null
    if (ae && (ae.tagName === 'INPUT' || ae.tagName === 'TEXTAREA')) {
      const start = ae.selectionStart ?? 0
      const end = ae.selectionEnd ?? 0
      if (end > start) {
        setText(ae.value.slice(start, end))
        return
      }
    }
    const sel = window.getSelection()
    setText(sel ? sel.toString() : '')
  }, [])

  useEffect(() => {
    compute()
    document.addEventListener('selectionchange', compute)
    window.addEventListener('focusin', compute)
    window.addEventListener('focusout', compute)
    return () => {
      document.removeEventListener('selectionchange', compute)
      window.removeEventListener('focusin', compute)
      window.removeEventListener('focusout', compute)
    }
  }, [compute])

  return text
}
