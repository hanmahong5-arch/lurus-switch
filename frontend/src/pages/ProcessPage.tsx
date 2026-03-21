import { useEffect, useState, useRef, useCallback } from 'react'
import { Play, Square, Trash2, RefreshCw, Loader2, Activity, Pause } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useClassifiedError } from '../lib/useClassifiedError'
import { InlineError } from '../components/InlineError'
import { ListCLIProcesses, KillCLIProcess, LaunchTool, GetToolOutput, StopToolSession } from '../../wailsjs/go/main/App'

interface ProcessInfo {
  pid: number
  tool: string
  command: string
  status: string
  memory: number
  since: string
}

const toolOptions = ['claude', 'codex', 'gemini', 'picoclaw', 'nullclaw']

export function ProcessPage() {
  const { t } = useTranslation()
  const [processes, setProcesses] = useState<ProcessInfo[]>([])
  const [loading, setLoading] = useState(false)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const { classified: error, setError, clearError } = useClassifiedError()
  const [killing, setKilling] = useState<Record<number, boolean>>({})

  // Launch panel
  const [launchTool, setLaunchTool] = useState('claude')
  const [launchArgs, setLaunchArgs] = useState('')
  const [launching, setLaunching] = useState(false)
  const [sessionID, setSessionID] = useState<string | null>(null)
  const [output, setOutput] = useState<string[]>([])
  const [stoppingSession, setStoppingSession] = useState(false)
  const outputRef = useRef<HTMLDivElement>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchProcesses = useCallback(async () => {
    setLoading(true)
    try {
      const list = await ListCLIProcesses()
      setProcesses(list || [])
      clearError()
    } catch (err) {
      setError(err)
    } finally {
      setLoading(false)
    }
  }, [])

  // Auto-refresh every 3 seconds when enabled
  useEffect(() => {
    fetchProcesses()
    if (autoRefresh) {
      intervalRef.current = setInterval(fetchProcesses, 3000)
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [autoRefresh, fetchProcesses])

  // Poll session output when a session is active
  useEffect(() => {
    if (!sessionID) return
    const poll = setInterval(async () => {
      try {
        const lines = await GetToolOutput(sessionID, 100)
        setOutput(lines || [])
        // Auto-scroll to bottom
        if (outputRef.current) {
          outputRef.current.scrollTop = outputRef.current.scrollHeight
        }
      } catch {
        // Session may have ended
      }
    }, 1000)
    return () => clearInterval(poll)
  }, [sessionID])

  const handleKill = async (pid: number) => {
    setKilling((prev) => ({ ...prev, [pid]: true }))
    try {
      await KillCLIProcess(pid)
      await fetchProcesses()
    } catch (err) {
      setError(err)
    } finally {
      setKilling((prev) => ({ ...prev, [pid]: false }))
    }
  }

  const handleLaunch = async () => {
    setLaunching(true)
    clearError()
    setOutput([])
    try {
      const args = launchArgs.trim() ? launchArgs.trim().split(/\s+/) : []
      const id = await LaunchTool(launchTool, args)
      setSessionID(id)
    } catch (err) {
      setError(err)
    } finally {
      setLaunching(false)
    }
  }

  const handleStopSession = async () => {
    if (!sessionID) return
    setStoppingSession(true)
    try {
      await StopToolSession(sessionID)
      setSessionID(null)
    } catch (err) {
      setError(err)
    } finally {
      setStoppingSession(false)
    }
  }

  const formatMemory = (bytes: number) => {
    if (bytes === 0) return '-'
    if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  }

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-3xl mx-auto p-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold">{t('process.title')}</h2>
            <p className="text-sm text-muted-foreground">{t('process.subtitle')}</p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setAutoRefresh(!autoRefresh)}
              className={cn(
                'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border transition-colors',
                autoRefresh
                  ? 'border-primary text-primary bg-primary/10'
                  : 'border-border text-muted-foreground hover:bg-muted'
              )}
            >
              {autoRefresh ? <Pause className="h-3.5 w-3.5" /> : <Activity className="h-3.5 w-3.5" />}
              {autoRefresh ? t('process.pauseRefresh') : t('process.autoRefresh')}
            </button>
            <button
              onClick={fetchProcesses}
              disabled={loading}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border border-border hover:bg-muted transition-colors disabled:opacity-50"
            >
              {loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
              {t('dashboard.refresh')}
            </button>
          </div>
        </div>

        {error && (
          <InlineError
            category={error.category}
            message={error.message}
            details={error.details}
            onDismiss={clearError}
          />
        )}

        {/* Process List */}
        <div className="border border-border rounded-lg overflow-hidden">
          <div className="bg-muted/50 px-4 py-2 border-b border-border">
            <div className="grid grid-cols-5 text-xs font-medium text-muted-foreground uppercase tracking-wider">
              <span>{t('process.col.tool')}</span>
              <span>{t('process.col.pid')}</span>
              <span>{t('process.col.status')}</span>
              <span>{t('process.col.memory')}</span>
              <span className="text-right">{t('process.col.actions')}</span>
            </div>
          </div>
          {processes.length === 0 ? (
            <div className="py-8 text-center text-sm text-muted-foreground">
              {loading ? t('process.detecting') : t('process.empty')}
            </div>
          ) : (
            processes.map((proc) => (
              <div key={proc.pid} className="px-4 py-3 border-b border-border last:border-0 hover:bg-muted/30">
                <div className="grid grid-cols-5 items-center text-sm">
                  <span className="font-medium capitalize">{proc.tool}</span>
                  <span className="text-muted-foreground font-mono">{proc.pid}</span>
                  <span className={cn(
                    'text-xs px-1.5 py-0.5 rounded-full inline-block w-fit',
                    proc.status === 'running'
                      ? 'bg-green-500/10 text-green-500'
                      : 'bg-muted text-muted-foreground'
                  )}>
                    {proc.status}
                  </span>
                  <span className="text-muted-foreground text-xs">{formatMemory(proc.memory)}</span>
                  <div className="flex justify-end">
                    <button
                      onClick={() => handleKill(proc.pid)}
                      disabled={killing[proc.pid]}
                      className="flex items-center gap-1 px-2 py-1 text-xs border border-red-500/30 text-red-500 rounded hover:bg-red-500/10 transition-colors disabled:opacity-50"
                    >
                      {killing[proc.pid] ? <Loader2 className="h-3 w-3 animate-spin" /> : <Trash2 className="h-3 w-3" />}
                      {t('process.kill')}
                    </button>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>

        {/* Launch Panel */}
        <div className="border border-border rounded-lg p-4 space-y-4">
          <h3 className="text-sm font-semibold">{t('process.launchTool')}</h3>
          <div className="flex gap-2">
            <select
              value={launchTool}
              onChange={(e) => setLaunchTool(e.target.value)}
              className="px-3 py-1.5 text-sm bg-muted border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
            >
              {toolOptions.map((tool) => (
                <option key={tool} value={tool}>{tool}</option>
              ))}
            </select>
            <input
              type="text"
              value={launchArgs}
              onChange={(e) => setLaunchArgs(e.target.value)}
              placeholder={t('process.argsPlaceholder')}
              className="flex-1 px-3 py-1.5 text-sm bg-muted border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
            />
            {sessionID ? (
              <button
                onClick={handleStopSession}
                disabled={stoppingSession}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm bg-red-500 text-white hover:bg-red-600 transition-colors disabled:opacity-50"
              >
                {stoppingSession ? <Loader2 className="h-4 w-4 animate-spin" /> : <Square className="h-4 w-4" />}
                {t('process.stop')}
              </button>
            ) : (
              <button
                onClick={handleLaunch}
                disabled={launching}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {launching ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
                {t('process.launch')}
              </button>
            )}
          </div>

          {/* Output window */}
          {(sessionID || output.length > 0) && (
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-xs text-muted-foreground">
                  {t('process.output')} {sessionID ? `(session: ${sessionID.slice(-8)})` : ''}
                </span>
                <button
                  onClick={() => setOutput([])}
                  className="text-xs text-muted-foreground hover:text-foreground"
                >
                  <Trash2 className="h-3 w-3" />
                </button>
              </div>
              <div
                ref={outputRef}
                className="bg-black/90 text-green-400 font-mono text-xs p-3 rounded-md h-48 overflow-y-auto"
              >
                {output.length === 0 ? (
                  <span className="text-muted-foreground">{t('process.waitingOutput')}</span>
                ) : (
                  output.map((line, i) => (
                    <div key={i}>{line}</div>
                  ))
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
