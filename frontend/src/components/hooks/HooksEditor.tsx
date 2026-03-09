import { useEffect, useState } from 'react'
import { Plus, Trash2, Loader2, Zap } from 'lucide-react'
import { GetClaudeHooks, SaveClaudeHooks } from '../../../wailsjs/go/main/App'

type HookEvent = 'preToolUse' | 'postToolUse' | 'preCompact'

interface HookEntry {
  matcher?: string
  hooks: { type: string; command: string }[]
}

interface HooksConfig {
  preToolUse?: HookEntry[]
  postToolUse?: HookEntry[]
  preCompact?: HookEntry[]
}

const EVENT_LABELS: Record<HookEvent, string> = {
  preToolUse: '工具调用前',
  postToolUse: '工具调用后',
  preCompact: '压缩前',
}

const TEMPLATES: { label: string; event: HookEvent; matcher: string; command: string }[] = [
  {
    label: '记录工具调用日志',
    event: 'postToolUse',
    matcher: '.*',
    command: 'echo "Tool: $TOOL_NAME Exit: $EXIT_CODE" >> ~/.lurus-switch/tool-calls.log',
  },
  {
    label: 'Git 自动提交',
    event: 'postToolUse',
    matcher: 'Edit|Write|MultiEdit',
    command: 'git add -A && git commit -m "auto: tools changes" --no-verify 2>/dev/null || true',
  },
  {
    label: '安全检查（阻断敏感操作）',
    event: 'preToolUse',
    matcher: 'Bash',
    command: 'bash ~/.lurus-switch/security-check.sh "$TOOL_INPUT"',
  },
]

export function HooksEditor() {
  const [hooks, setHooks] = useState<HooksConfig>({})
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')
  const [activeEvent, setActiveEvent] = useState<HookEvent>('preToolUse')

  useEffect(() => {
    GetClaudeHooks()
      .then((h) => setHooks((h as HooksConfig) || {}))
      .catch((err) => setError(`Failed to load hooks: ${err}`))
      .finally(() => setLoading(false))
  }, [])

  const handleSave = async () => {
    setSaving(true)
    setError('')
    try {
      await SaveClaudeHooks(hooks as Record<string, unknown>)
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch (err) {
      setError(`Failed to save: ${err}`)
    } finally {
      setSaving(false)
    }
  }

  const getEntries = () => hooks[activeEvent] || []

  const setEntries = (entries: HookEntry[]) => {
    setHooks({ ...hooks, [activeEvent]: entries })
  }

  const addEntry = () => {
    setEntries([...getEntries(), { matcher: '.*', hooks: [{ type: 'command', command: '' }] }])
  }

  const removeEntry = (i: number) => {
    const next = [...getEntries()]
    next.splice(i, 1)
    setEntries(next)
  }

  const updateEntry = (i: number, field: keyof HookEntry, value: string | { type: string; command: string }[]) => {
    const next = [...getEntries()]
    next[i] = { ...next[i], [field]: value }
    setEntries(next)
  }

  const addHookToEntry = (entryIdx: number) => {
    const entry = getEntries()[entryIdx]
    updateEntry(entryIdx, 'hooks', [...entry.hooks, { type: 'command', command: '' }])
  }

  const updateHookCommand = (entryIdx: number, hookIdx: number, command: string) => {
    const entry = getEntries()[entryIdx]
    const updatedHooks = entry.hooks.map((h, i) => i === hookIdx ? { ...h, command } : h)
    updateEntry(entryIdx, 'hooks', updatedHooks)
  }

  const removeHook = (entryIdx: number, hookIdx: number) => {
    const entry = getEntries()[entryIdx]
    const updatedHooks = entry.hooks.filter((_, i) => i !== hookIdx)
    updateEntry(entryIdx, 'hooks', updatedHooks)
  }

  const applyTemplate = (tpl: typeof TEMPLATES[0]) => {
    if (tpl.event !== activeEvent) {
      setActiveEvent(tpl.event)
    }
    const entries = hooks[tpl.event] || []
    setHooks({
      ...hooks,
      [tpl.event]: [
        ...entries,
        { matcher: tpl.matcher, hooks: [{ type: 'command', command: tpl.command }] },
      ],
    })
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {error && (
        <div className="px-3 py-2 text-xs text-red-500 bg-red-500/10 rounded border border-red-500/20 flex items-center justify-between">
          {error}
          <button onClick={() => setError('')}>✕</button>
        </div>
      )}

      {/* Templates */}
      <div className="border border-border rounded-lg p-3 space-y-2 bg-muted/10">
        <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider flex items-center gap-1">
          <Zap className="h-3 w-3" /> 快速模板
        </p>
        <div className="flex flex-wrap gap-1">
          {TEMPLATES.map((t) => (
            <button
              key={t.label}
              onClick={() => applyTemplate(t)}
              className="px-2 py-1 text-xs border border-border rounded hover:bg-muted transition-colors"
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>

      {/* Event Tabs */}
      <div className="flex gap-1 border-b border-border">
        {(Object.keys(EVENT_LABELS) as HookEvent[]).map((ev) => (
          <button
            key={ev}
            onClick={() => setActiveEvent(ev)}
            className={`px-3 py-1.5 text-xs font-medium border-b-2 transition-colors ${
              activeEvent === ev
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {EVENT_LABELS[ev]}
            {(hooks[ev]?.length ?? 0) > 0 && (
              <span className="ml-1 bg-primary/20 text-primary rounded-full px-1.5 py-0.5 text-xs">
                {hooks[ev]?.length}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Hook Entries */}
      <div className="space-y-3">
        {getEntries().length === 0 ? (
          <p className="text-xs text-muted-foreground text-center py-4">暂无 hooks，点击下方按钮添加</p>
        ) : (
          getEntries().map((entry, i) => (
            <div key={i} className="border border-border rounded-lg p-3 space-y-2">
              <div className="flex items-center gap-2">
                <div className="flex-1">
                  <label className="text-xs text-muted-foreground">Matcher (正则)</label>
                  <input
                    type="text"
                    value={entry.matcher || ''}
                    onChange={(e) => updateEntry(i, 'matcher', e.target.value)}
                    placeholder=".*"
                    className="w-full mt-1 px-2 py-1 text-xs font-mono bg-muted border border-border rounded focus:outline-none"
                  />
                </div>
                <button
                  onClick={() => removeEntry(i)}
                  className="p-1 text-muted-foreground hover:text-red-500 transition-colors mt-4 shrink-0"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
              <div className="space-y-1.5">
                <label className="text-xs text-muted-foreground">命令</label>
                {entry.hooks.map((h, j) => (
                  <div key={j} className="flex items-center gap-1">
                    <input
                      type="text"
                      value={h.command}
                      onChange={(e) => updateHookCommand(i, j, e.target.value)}
                      placeholder="shell 命令..."
                      className="flex-1 px-2 py-1 text-xs font-mono bg-muted border border-border rounded focus:outline-none"
                    />
                    {entry.hooks.length > 1 && (
                      <button
                        onClick={() => removeHook(i, j)}
                        className="p-1 text-muted-foreground hover:text-red-500"
                      >
                        <Trash2 className="h-3 w-3" />
                      </button>
                    )}
                  </div>
                ))}
                <button
                  onClick={() => addHookToEntry(i)}
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
                >
                  <Plus className="h-3 w-3" /> 添加命令
                </button>
              </div>
            </div>
          ))
        )}

        <button
          onClick={addEntry}
          className="w-full flex items-center justify-center gap-1.5 px-3 py-2 text-xs border border-dashed border-border rounded-lg hover:bg-muted/50 transition-colors"
        >
          <Plus className="h-3.5 w-3.5" /> 添加 Hook 规则
        </button>
      </div>

      <button
        onClick={handleSave}
        disabled={saving}
        className={`w-full flex items-center justify-center gap-1.5 px-4 py-2 text-sm rounded transition-colors disabled:opacity-50 ${
          saved
            ? 'bg-green-500/20 text-green-500 border border-green-500/30'
            : 'bg-primary text-primary-foreground hover:bg-primary/90'
        }`}
      >
        {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
        {saved ? '已保存' : '保存 Hooks 配置'}
      </button>
    </div>
  )
}
