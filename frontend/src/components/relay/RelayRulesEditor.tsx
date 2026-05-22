import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Save, RotateCcw, Loader2, AlertCircle } from 'lucide-react'
import { GetRelayRules, SaveRelayRules } from '../../../wailsjs/go/main/App'

// RelayRulesEditor is the YAML editor for the optional routing rules
// that decide which RelayEndpoint should serve a given request. Strict
// schema parsing happens on the Go side; we just surface the resulting
// error to the user.
const PLACEHOLDER = `# Routing rules (optional). First match wins.
#
# rules:
#   - name: long-context-to-fast-cluster
#     match_model_prefix: claude-opus
#     min_tokens: 50000
#     prefer_endpoint_id: <relay-endpoint-id>
`

export function RelayRulesEditor() {
  const { t } = useTranslation()
  const [text, setText] = useState('')
  const [savedText, setSavedText] = useState('')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [savedAt, setSavedAt] = useState<number>(0)

  useEffect(() => {
    setLoading(true)
    GetRelayRules()
      .then((s) => {
        setText(s || '')
        setSavedText(s || '')
      })
      .catch((e) => setError(e?.message ?? String(e)))
      .finally(() => setLoading(false))
  }, [])

  const dirty = text !== savedText

  const save = async () => {
    setSaving(true)
    setError(null)
    try {
      await SaveRelayRules(text)
      setSavedText(text)
      setSavedAt(Date.now())
    } catch (e: any) {
      setError(e?.message ?? String(e))
    } finally {
      setSaving(false)
    }
  }

  const revert = () => {
    setText(savedText)
    setError(null)
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2 text-xs">
        <span className="font-medium">
          {t('relay.rulesEditor.title', '路由规则 · Routing rules (YAML)')}
        </span>
        <span className="text-muted-foreground">
          {t('relay.rulesEditor.subtitle', 'First match wins; predicates AND together.')}
        </span>
        <div className="ml-auto flex items-center gap-1">
          <button
            disabled={!dirty || saving}
            onClick={revert}
            className="px-2 py-1 rounded text-xs hover:bg-muted disabled:opacity-40"
          >
            <RotateCcw className="h-3 w-3 inline-block mr-1" />
            {t('common.revert', 'Revert')}
          </button>
          <button
            disabled={!dirty || saving}
            onClick={save}
            className="px-2 py-1 rounded text-xs bg-primary text-primary-foreground hover:opacity-90 disabled:opacity-40"
          >
            {saving ? <Loader2 className="h-3 w-3 inline-block mr-1 animate-spin" /> : <Save className="h-3 w-3 inline-block mr-1" />}
            {t('common.save', 'Save')}
          </button>
        </div>
      </div>
      {loading ? (
        <div className="h-32 flex items-center justify-center">
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        </div>
      ) : (
        <textarea
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder={PLACEHOLDER}
          spellCheck={false}
          className="w-full h-48 px-2 py-1.5 rounded bg-background border border-border text-xs font-mono focus:border-primary outline-none"
        />
      )}
      {error && (
        <div className="text-xs text-red-500 flex items-center gap-1">
          <AlertCircle className="h-3 w-3" />
          {error}
        </div>
      )}
      {savedAt > 0 && !error && !dirty && (
        <div className="text-[11px] text-emerald-500">
          {t('relay.rulesEditor.saved', 'Saved.')}
        </div>
      )}
    </div>
  )
}
