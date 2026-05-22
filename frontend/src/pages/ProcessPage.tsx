import { useEffect, useState, useRef, useCallback } from 'react'
import { Play, Square, Trash2, RefreshCw, Loader2, Activity, Pause } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
import { useClassifiedError } from '../lib/useClassifiedError'
import { InlineError } from '../components/InlineError'
import { Button, Card } from '../components/ui'
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
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setAutoRefresh(!autoRefresh)}
              icon={autoRefresh ? <Pause className="h-3.5 w-3.5" /> : <Activity className="h-3.5 w-3.5" />}
              className={autoRefresh ? 'border-primary text-primary bg-primary/10 hover:bg-primary/15' : ''}
            >
              {autoRefresh ? t('process.pauseRefresh') : t('process.autoRefresh')}
            </Button>
            <Button
              variant="secondary"
              size="sm"
              onClick={fetchProcesses}
              disabled={loading}
              loading={loading}
              icon={!loading ? <RefreshCw className="h-3.5 w-3.5" /> : undefined}
            >
              {t('dashboard.refresh')}
            </Button>
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
        <Card variant="default" className="overflow-hidden">
          <div className="bg-card-recessed px-4 py-2 border-b border-border">
            <div className="grid grid-cols-5 font-mono text-[10px] font-medium text-muted-foreground uppercase tracking-[0.18em]">
              <span>[ {t('process.col.tool').toUpperCase()} ]</span>
              <span>[ {t('process.col.pid').toUpperCase()} ]</span>
              <span>[ {t('process.col.status').toUpperCase()} ]</span>
              <span>[ {t('process.col.memory').toUpperCase()} ]</span>
              <span className="text-right">[ {t('process.col.actions').toUpperCase()} ]</span>
            </div>
          </div>
          {processes.length === 0 ? (
            <div className="py-8 text-center text-sm text-muted-foreground font-mono">
              {loading ? t('process.detecting') : t('process.empty')}
            </div>
          ) : (
            processes.map((proc) => (
              <div key={proc.pid} className="px-4 py-3 border-b border-border last:border-0 hover:bg-muted/30 transition-colors">
                <div className="grid grid-cols-5 items-center text-sm">
                  <span className="font-medium capitalize">{proc.tool}</span>
                  <span className="text-muted-foreground font-mono tabular-nums">{proc.pid}</span>
                  <span className={cn(
                    'text-xs px-1.5 py-0.5 rounded font-mono inline-block w-fit',
                    proc.status === 'running'
                      ? 'bg-emerald-500/15 text-emerald-400'
                      : 'bg-card-recessed text-muted-foreground'
                  )}>
                    ▸ {proc.status}
                  </span>
                  <span className="text-muted-foreground text-xs font-mono tabular-nums">{formatMemory(proc.memory)}</span>
                  <div className="flex justify-end">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleKill(proc.pid)}
                      disabled={killing[proc.pid]}
                      loading={killing[proc.pid]}
                      icon={!killing[proc.pid] ? <Trash2 className="h-3 w-3" /> : undefined}
                      className="border border-red-500/30 text-red-400 hover:bg-red-500/10"
                    >
                      {t('process.kill')}
                    </Button>
                  </div>
                </div>
              </div>
            ))
          )}
        </Card>

        {/* Launch Panel */}
        <Card variant="default" className="p-4 space-y-4">
          <h3 className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground">
            [ {t('process.launchTool').toUpperCase()} ]
          </h3>
          <div className="flex gap-2">
            <select
              value={launchTool}
              onChange={(e) => setLaunchTool(e.target.value)}
              className="px-3 py-1.5 text-sm bg-muted border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
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
              className="flex-1 px-3 py-1.5 text-sm bg-muted border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary font-mono"
            />
            {sessionID ? (
              <Button
                variant="danger"
                onClick={handleStopSession}
                disabled={stoppingSession}
                loading={stoppingSession}
                icon={!stoppingSession ? <Square className="h-4 w-4" /> : undefined}
              >
                {t('process.stop')}
              </Button>
            ) : (
              <Button
                onClick={handleLaunch}
                disabled={launching}
                loading={launching}
                icon={!launching ? <Play className="h-4 w-4" /> : undefined}
              >
                {t('process.launch')}
              </Button>
            )}
          </div>

          {/* Output window */}
          {(sessionID || output.length > 0) && (
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
                  [ {t('process.output').toUpperCase()} ]
                  {sessionID && <span className="ml-2 tabular-nums normal-case">session: {sessionID.slice(-8)}</span>}
                </span>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setOutput([])}
                  icon={<Trash2 className="h-3 w-3" />}
                />
              </div>
              <div
                ref={outputRef}
                className="bg-card-recessed text-emerald-400 font-mono text-xs p-3 rounded-md h-48 overflow-y-auto border border-border"
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
        </Card>
      </div>
    </div>
  )
}
