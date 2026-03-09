import { useRef, useState } from 'react'
import { Play, Square, Trash2, Loader2 } from 'lucide-react'
import { LaunchTool, GetToolOutput, StopToolSession } from '../../wailsjs/go/main/App'

const TOOLS = ['claude', 'codex', 'gemini', 'picoclaw', 'nullclaw']

const QUICK_COMMANDS: { label: string; args: string }[] = [
  { label: '--version', args: '--version' },
  { label: '--help', args: '--help' },
  { label: 'doctor', args: 'doctor' },
  { label: 'update', args: 'update' },
]

export function CLIRunner() {
  const [tool, setTool] = useState('claude')
  const [args, setArgs] = useState('')
  const [launching, setLaunching] = useState(false)
  const [sessionID, setSessionID] = useState<string | null>(null)
  const [output, setOutput] = useState<string[]>([])
  const [stopping, setStopping] = useState(false)
  const outputRef = useRef<HTMLDivElement>(null)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const startPoll = (id: string) => {
    if (pollRef.current) clearInterval(pollRef.current)
    pollRef.current = setInterval(async () => {
      try {
        const lines = await GetToolOutput(id, 200)
        setOutput(lines || [])
        if (outputRef.current) {
          outputRef.current.scrollTop = outputRef.current.scrollHeight
        }
      } catch {
        // session ended
        if (pollRef.current) clearInterval(pollRef.current)
      }
    }, 800)
  }

  const handleLaunch = async (customArgs?: string) => {
    const argStr = customArgs ?? args
    setLaunching(true)
    setOutput([])
    try {
      const argList = argStr.trim() ? argStr.trim().split(/\s+/) : []
      const id = await LaunchTool(tool, argList)
      setSessionID(id)
      startPoll(id)
    } catch (err) {
      setOutput([`Error: ${err}`])
    } finally {
      setLaunching(false)
    }
  }

  const handleStop = async () => {
    if (!sessionID) return
    setStopping(true)
    try {
      await StopToolSession(sessionID)
      if (pollRef.current) clearInterval(pollRef.current)
      setSessionID(null)
    } catch {
      // ignore
    } finally {
      setStopping(false)
    }
  }

  const handleQuick = (cmd: string) => {
    setArgs(cmd)
    handleLaunch(cmd)
  }

  return (
    <div className="space-y-3">
      {/* Tool + Args */}
      <div className="flex gap-2">
        <select
          value={tool}
          onChange={(e) => setTool(e.target.value)}
          className="px-2 py-1.5 text-sm bg-muted border border-border rounded focus:outline-none focus:ring-1 focus:ring-primary"
        >
          {TOOLS.map((t) => <option key={t} value={t}>{t}</option>)}
        </select>
        <input
          type="text"
          value={args}
          onChange={(e) => setArgs(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && !sessionID && handleLaunch()}
          placeholder="参数..."
          className="flex-1 px-2 py-1.5 text-sm bg-muted border border-border rounded focus:outline-none focus:ring-1 focus:ring-primary font-mono"
        />
        {sessionID ? (
          <button
            onClick={handleStop}
            disabled={stopping}
            className="flex items-center gap-1 px-3 py-1.5 text-sm bg-red-500 text-white rounded hover:bg-red-600 transition-colors disabled:opacity-50"
          >
            {stopping ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Square className="h-3.5 w-3.5" />}
            Stop
          </button>
        ) : (
          <button
            onClick={() => handleLaunch()}
            disabled={launching}
            className="flex items-center gap-1 px-3 py-1.5 text-sm bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            {launching ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Play className="h-3.5 w-3.5" />}
            运行
          </button>
        )}
      </div>

      {/* Quick Commands */}
      <div className="flex items-center gap-1 flex-wrap">
        <span className="text-xs text-muted-foreground mr-1">快速:</span>
        {QUICK_COMMANDS.map((q) => (
          <button
            key={q.label}
            onClick={() => handleQuick(q.args)}
            disabled={!!sessionID}
            className="px-2 py-0.5 text-xs border border-border rounded hover:bg-muted transition-colors disabled:opacity-50 font-mono"
          >
            {q.label}
          </button>
        ))}
      </div>

      {/* Output */}
      {output.length > 0 && (
        <div className="space-y-1">
          <div className="flex items-center justify-between">
            <span className="text-xs text-muted-foreground">
              输出 {sessionID ? `(${sessionID.slice(-8)})` : ''}
            </span>
            <button
              onClick={() => setOutput([])}
              className="text-muted-foreground hover:text-foreground"
            >
              <Trash2 className="h-3 w-3" />
            </button>
          </div>
          <div
            ref={outputRef}
            className="bg-black/90 text-green-400 font-mono text-xs p-3 rounded h-36 overflow-y-auto"
          >
            {output.map((line, i) => (
              <div key={i}>{line || '\u00A0'}</div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
