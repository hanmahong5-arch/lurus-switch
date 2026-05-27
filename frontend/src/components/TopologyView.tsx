import { useEffect, useMemo, useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { BorderBeam } from './ui/magicui/BorderBeam'
import {
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  Background,
  Controls,
  MiniMap,
  Handle,
  Position,
  type Node as RFNode,
  type Edge as RFEdge,
  type NodeTypes,
  type NodeMouseHandler,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import dagre from '@dagrejs/dagre'
import {
  RefreshCw,
  Loader2,
  Wrench,
  Terminal,
  Server,
  Network,
  ShieldCheck,
  Cloud,
  Cpu,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  CircleDot,
  PlugZap,
  Play,
  LogIn,
  Download,
  ArrowUpCircle,
  Activity as ActivityIcon,
} from 'lucide-react'
import { cn } from '../lib/utils'
import { useTopologyStore } from '../stores/topologyStore'
import { useConfigStore } from '../stores/configStore'
import { useToastStore } from '../stores/toastStore'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import {
  StartGateway,
  InstallTool,
  UpdateTool,
  AutoFixToolConfig,
  Login,
  LaunchToolInTerminal,
} from '../../wailsjs/go/main/App'
import type { topology } from '../../wailsjs/go/models'

// Architecture topology view rendered with React Flow + dagre auto-layout.
// Node clicks navigate; all install/launch/repair actions live in a single
// action bar BELOW the canvas (the in-card buttons were too small to hit
// reliably). The history strip surfaces recent activity (installs, gateway
// starts, logins, errors) so users see what just happened.

type SnapNode = topology.Node
type SnapEdge = topology.Edge

interface NodeData extends Record<string, unknown> {
  snap: SnapNode
}

interface ActivityEvent {
  id: string
  phase: 'start' | 'progress' | 'done' | 'error'
  titleZh: string
  titleEn: string
  detailZh?: string
  detailEn?: string
  error?: string
  updatedAt: string
}

const STATUS_BORDER: Record<string, string> = {
  ok: 'border-[var(--lt-ok)] bg-[var(--lt-ok)]/8 text-[var(--lt-ok)]',
  degraded: 'border-[var(--lt-warn)] bg-[var(--lt-warn)]/10 text-[var(--lt-warn)]',
  down: 'border-[var(--lt-err)] bg-[var(--lt-err)]/10 text-[var(--lt-err)]',
  notconfigured: 'border-border bg-muted/40 text-muted-foreground',
  unknown: 'border-border bg-muted/30 text-muted-foreground',
}

const STATUS_DOT: Record<string, string> = {
  ok: 'bg-[var(--lt-ok)]',
  degraded: 'bg-[var(--lt-warn)]',
  down: 'bg-[var(--lt-err)]',
  notconfigured: 'bg-muted-foreground/40',
  unknown: 'bg-muted-foreground/30',
}

const STATUS_STROKE: Record<string, string> = {
  ok: 'var(--lt-ok)',
  degraded: 'var(--lt-warn)',
  down: 'var(--lt-err)',
  notconfigured: 'hsl(var(--muted-foreground) / 0.4)',
  unknown: 'hsl(var(--muted-foreground) / 0.3)',
}

const ICON_FOR_KIND: Record<string, React.ComponentType<{ className?: string }>> = {
  tool: Terminal,
  gateway: Server,
  proxy: Network,
  auth: ShieldCheck,
  hub: Cloud,
  provider: Cpu,
  mcp: PlugZap,
}

const TOOL_DISPLAY: Record<string, string> = {
  claude: 'Claude Code',
  codex: 'Codex',
  gemini: 'Gemini',
  picoclaw: 'PicoClaw',
  nullclaw: 'NullClaw',
  zeroclaw: 'ZeroClaw',
  openclaw: 'OpenClaw',
}

const NODE_WIDTH = 260
const NODE_HEIGHT = 116

function StatusNode({ data }: { data: NodeData }) {
  const { snap } = data
  const Icon = ICON_FOR_KIND[snap.kind] ?? Server
  const cls = STATUS_BORDER[snap.status] ?? STATUS_BORDER.unknown
  const dot = STATUS_DOT[snap.status] ?? STATUS_DOT.unknown

  return (
    <div
      className={cn(
        'rounded-lg border-2 px-3.5 py-2.5 select-none cursor-pointer transition-shadow hover:shadow-md',
        cls,
        snap.highlight && 'ring-2 ring-[var(--lt-accent)] ring-offset-1',
      )}
      style={{ width: NODE_WIDTH }}
    >
      <Handle type="target" position={Position.Top} style={{ opacity: 0, pointerEvents: 'none' }} />
      <Handle type="source" position={Position.Bottom} style={{ opacity: 0, pointerEvents: 'none' }} />

      <div className="flex items-start gap-2">
        <Icon className="h-4 w-4 flex-shrink-0 mt-0.5" />
        <div className="min-w-0 flex-1">
          <div className="text-sm font-semibold leading-tight truncate">{snap.label}</div>
          {snap.badge && <div className="text-xs text-muted-foreground truncate mt-0.5">{snap.badge}</div>}
        </div>
        <span className={cn('h-2.5 w-2.5 rounded-full flex-shrink-0 mt-1.5', dot)} />
      </div>
      {snap.detail && (
        <div className="text-xs mt-1.5 opacity-80 truncate" title={snap.detail}>
          {snap.detail}
        </div>
      )}
      {snap.latencyMs ? <div className="text-xs opacity-60 mt-0.5">{snap.latencyMs} ms</div> : null}
    </div>
  )
}

const nodeTypes: NodeTypes = { status: StatusNode }

// dagreLayout uses TB (top-bottom) so the data flow reads as a vertical
// tree: CLI tools on top → local gateway → hub → providers at the bottom.
// Most screens are landscape, so a tall thin tree wastes width; TB fans
// the wide-fanout layers (7 CLI tools, 5 providers) out horizontally to
// fill the canvas.
function dagreLayout(nodes: RFNode[], edges: RFEdge[]): RFNode[] {
  const g = new dagre.graphlib.Graph()
  g.setDefaultEdgeLabel(() => ({}))
  g.setGraph({ rankdir: 'TB', ranksep: 90, nodesep: 32, marginx: 24, marginy: 24 })

  nodes.forEach((n) => g.setNode(n.id, { width: NODE_WIDTH, height: NODE_HEIGHT }))
  edges.forEach((e) => g.setEdge(e.source, e.target))
  dagre.layout(g)

  return nodes.map((n) => {
    const pos = g.node(n.id)
    return {
      ...n,
      position: { x: pos.x - NODE_WIDTH / 2, y: pos.y - NODE_HEIGHT / 2 },
    }
  })
}

export function TopologyView() {
  const { t, i18n } = useTranslation()
  const isZh = i18n.language?.startsWith('zh') ?? true
  const { snapshot, loading, error, refresh, startPolling, stopPolling, lastUpdated } = useTopologyStore()
  const { setActiveTool, setSubTab } = useConfigStore()
  const toast = useToastStore((s) => s.addToast)
  const [acting, setActing] = useState<string | null>(null)
  const [selected, setSelected] = useState<SnapNode | null>(null)
  const [history, setHistory] = useState<ActivityEvent[]>([])

  useEffect(() => {
    startPolling()
    return () => stopPolling()
  }, [])

  // Subscribe to the global activity bus — install / start-gateway / login
  // events all land here. We keep a 15-entry rolling buffer so the user can
  // see *what just happened* without opening the ActivityPane.
  useEffect(() => {
    const unsub = EventsOn('activity:event', (ev: ActivityEvent) => {
      setHistory((prev) => {
        const without = prev.filter((e) => e.id !== ev.id)
        return [ev, ...without].slice(0, 15)
      })
    })
    return () => {
      if (unsub) unsub()
    }
  }, [])

  // React Flow's onNodeClick fires on the wrapper that actually receives the
  // mouse event. Doing the click from inside the custom node component was
  // unreliable because RF intercepts pointer events at the wrapper level
  // (this was the user's "selected nothing" bug).
  const handleNodeClick: NodeMouseHandler = useCallback((_e, node) => {
    const data = node.data as NodeData
    setSelected(data.snap)
    if (data.snap.navPage) {
      setActiveTool(data.snap.navPage as any)
      if (data.snap.navSubTab) setSubTab(data.snap.navPage as any, data.snap.navSubTab)
    }
  }, [setActiveTool, setSubTab])

  const runAction = useCallback(
    async (key: string, fn: () => Promise<unknown>, successMsg?: string) => {
      if (acting) return
      setActing(key)
      try {
        await fn()
        if (successMsg) toast('success', successMsg)
        await refresh()
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e)
        // Typed errors from LaunchToolInTerminal: route to the right toast
        // tone so the user gets an actionable hint instead of a red wall.
        if (msg.startsWith('not-found:')) {
          toast('warning', msg.replace(/^not-found:\s*/, ''))
        } else if (msg.startsWith('broken-bin:')) {
          // Pre-launch probe detected a corrupted bun-shim or similar —
          // the message already includes the exact reinstall command.
          toast('warning', msg.replace(/^broken-bin:\s*/, ''))
        } else {
          toast('error', msg)
        }
      } finally {
        setActing(null)
      }
    },
    [acting, refresh, toast],
  )

  const { rfNodes, rfEdges } = useMemo(() => {
    if (!snapshot) return { rfNodes: [], rfEdges: [] }
    const nodes: RFNode[] = snapshot.nodes.map((sn) => ({
      id: sn.id,
      type: 'status',
      position: { x: 0, y: 0 },
      draggable: false,
      data: { snap: sn },
    }))
    const edges: RFEdge[] = snapshot.edges.map((se, i) => {
      const status = se.status || 'unknown'
      return {
        id: `e${i}`,
        source: se.from,
        target: se.to,
        animated: status === 'ok',
        style: {
          stroke: STATUS_STROKE[status] ?? STATUS_STROKE.unknown,
          strokeWidth: 1.6,
          strokeDasharray: status === 'notconfigured' || status === 'unknown' ? '4 4' : undefined,
        },
        label: [se.credential, se.label].filter(Boolean).join(' · ') || undefined,
        labelStyle: { fontSize: 10, fill: 'hsl(var(--muted-foreground))' },
        labelBgPadding: [4, 2] as [number, number],
        labelBgStyle: { fill: 'hsl(var(--card))', fillOpacity: 0.85 },
      }
    })
    return { rfNodes: dagreLayout(nodes, edges), rfEdges: edges }
  }, [snapshot])

  // Derive the action bar's button list from the snapshot. The bar is the
  // single source of truth for "what can I do right now" — fix actions
  // (red/yellow nodes) on the left, launch actions (installed tools) in
  // the middle, config jumps on the right.
  const actions = useMemo(() => {
    const fix: ActionItem[] = []
    const launch: ActionItem[] = []
    const config: ActionItem[] = []
    if (!snapshot) return { fix, launch, config }

    for (const n of snapshot.nodes) {
      if (n.kind === 'tool') {
        const toolName = n.id.replace(/^tool:/, '')
        const display = TOOL_DISPLAY[toolName] ?? n.label
        // InstallTool / UpdateTool / AutoFixToolConfig all return a Result
        // object whose .success can be false WITHOUT throwing (e.g. GitHub
        // release API blocked in CN). The action bar surfaces those as
        // real errors so the user doesn't see a fake "已安装" toast while
        // the tool's status stays grey.
        const ensureSuccess = (label: string) => (result: { success?: boolean; message?: string } | null | undefined) => {
          if (result && result.success === false) {
            const msg = result.message || `${label} ${isZh ? '操作失败' : 'failed'}`
            throw new Error(msg)
          }
        }
        if (n.status === 'notconfigured' && n.fixAction?.startsWith('install-tool:')) {
          fix.push({
            key: `install-${toolName}`,
            label: isZh ? `安装 ${display}` : `Install ${display}`,
            icon: Download,
            tone: 'fix',
            run: () =>
              runAction(
                `install-${toolName}`,
                async () => ensureSuccess(display)(await InstallTool(toolName)),
                isZh ? `${display} 已安装` : `${display} installed`,
              ),
          })
        } else if (n.fixAction?.startsWith('update-tool:')) {
          fix.push({
            key: `update-${toolName}`,
            label: isZh ? `更新 ${display}` : `Update ${display}`,
            icon: ArrowUpCircle,
            tone: 'warn',
            run: () =>
              runAction(
                `update-${toolName}`,
                async () => ensureSuccess(display)(await UpdateTool(toolName)),
                isZh ? `${display} 已更新` : `${display} updated`,
              ),
          })
        } else if (n.fixAction?.startsWith('fix-tool:')) {
          fix.push({
            key: `fix-${toolName}`,
            label: isZh ? `修复 ${display}` : `Repair ${display}`,
            icon: Wrench,
            tone: 'warn',
            run: () =>
              runAction(
                `fix-${toolName}`,
                async () => ensureSuccess(display)(await AutoFixToolConfig(toolName)),
                isZh ? `${display} 配置已修复` : `${display} repaired`,
              ),
          })
        }
        // Installed (ok / degraded but installed) — offer a launch shortcut.
        if (n.status === 'ok' || (n.status === 'degraded' && n.badge)) {
          launch.push({
            key: `launch-${toolName}`,
            label: isZh ? `启动 ${display}` : `Launch ${display}`,
            icon: Play,
            tone: 'launch',
            run: () =>
              runAction(
                `launch-${toolName}`,
                () => LaunchToolInTerminal(toolName),
                isZh ? `已在终端启动 ${display}` : `Launched ${display} in terminal`,
              ),
          })
        }
      } else if (n.id === 'gateway') {
        if (n.status === 'down' && n.fixAction === 'start-gateway') {
          fix.push({
            key: 'start-gateway',
            label: isZh ? '启动本地网关' : 'Start local gateway',
            icon: Play,
            tone: 'fix',
            run: () => runAction('start-gateway', () => StartGateway(), isZh ? '网关已启动' : 'Gateway started'),
          })
        }
        config.push({
          key: 'cfg-gateway',
          label: isZh ? '网关设置' : 'Gateway settings',
          icon: Server,
          tone: 'nav',
          run: async () => {
            setActiveTool('gateway' as any)
          },
        })
      } else if (n.id === 'auth') {
        if (n.fixAction === 'login') {
          fix.push({
            key: 'login',
            label: isZh ? '登录' : 'Sign in',
            icon: LogIn,
            tone: 'fix',
            run: () => runAction('login', () => Login(), isZh ? '已登录' : 'Signed in'),
          })
        }
      } else if (n.id === 'proxy' && n.status !== 'ok') {
        config.push({
          key: 'cfg-proxy',
          label: isZh ? '配置上游代理' : 'Configure upstream proxy',
          icon: Network,
          tone: 'nav',
          run: async () => {
            setActiveTool('settings' as any)
            setSubTab('settings' as any, 'proxy')
          },
        })
      }
    }
    return { fix, launch, config }
  }, [snapshot, isZh, runAction, setActiveTool, setSubTab])

  if (!snapshot && loading) {
    return (
      <div className="rounded-lg border border-border bg-card p-10 flex items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
        <span className="ml-3 text-sm text-muted-foreground">{t('topology.loading')}</span>
      </div>
    )
  }

  if (error && !snapshot) {
    return (
      <div className="rounded-lg border border-[var(--lt-err)]/30 bg-[var(--lt-err)]/5 p-6">
        <div className="flex items-center gap-2 text-[var(--lt-err)] mb-2">
          <AlertTriangle className="h-4 w-4" />
          <span className="text-sm font-medium">{t('topology.failed')}</span>
        </div>
        <p className="text-xs text-muted-foreground">{error}</p>
        <button onClick={refresh} className="mt-3 px-3 py-1.5 rounded-md text-xs border border-border hover:bg-muted">
          {t('topology.retry')}
        </button>
      </div>
    )
  }

  if (!snapshot) return null

  const summary = snapshot.summary
  const total = summary.ok + summary.degraded + summary.down + summary.notconfigured + summary.unknown

  // Show BorderBeam only when all nodes are ok (healthy glow) or when
  // there are nodes that need attention (warn glow) — not when completely
  // unconfigured (no data yet).
  const beamColor =
    summary.down > 0
      ? 'hsl(var(--destructive) / 0.6)'
      : summary.ok > 0
        ? 'hsl(var(--primary) / 0.55)'
        : undefined

  return (
    <div className="relative rounded-lg border border-border bg-card overflow-hidden">
      {beamColor && (
        <BorderBeam
          colorFrom={beamColor}
          colorTo="transparent"
          duration={8}
          borderRadius="0.5rem"
          borderWidth={1.5}
        />
      )}
      {/* Headline + summary chips + refresh */}
      <div className="flex items-start justify-between p-4 border-b border-border">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 text-sm">
            <HeadlineIcon summary={summary} />
            <span className="font-medium truncate">{summary.headline}</span>
          </div>
          <div className="flex flex-wrap gap-1.5 mt-2 text-[11px]">
            <Chip color="ok" label={t('topology.ok')} value={summary.ok} />
            {summary.degraded > 0 && <Chip color="warn" label={t('topology.degraded')} value={summary.degraded} />}
            {summary.down > 0 && <Chip color="err" label={t('topology.down')} value={summary.down} />}
            {summary.notconfigured > 0 && (
              <Chip color="muted" label={t('topology.notconfigured')} value={summary.notconfigured} />
            )}
            <Chip color="muted" label={t('topology.total')} value={total} />
          </div>
        </div>
        <button
          onClick={refresh}
          disabled={loading}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs border border-border hover:bg-muted disabled:opacity-50"
        >
          {loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
          {t('topology.refresh')}
        </button>
      </div>

      {/* React Flow canvas — height follows the viewport so a wide/tall
          window gets a roomy diagram. clamp() caps the extremes. The
          canvas is wrapped in a ReactFlowProvider so the inner component
          can call fitView() on window resize, keeping the diagram
          comfortably full regardless of how the user drags the window. */}
      <div style={{ height: 'clamp(520px, 68vh, 900px)' }} className="bg-muted/20">
        <ReactFlowProvider>
          <TopologyCanvas nodes={rfNodes} edges={rfEdges} onNodeClick={handleNodeClick} />
        </ReactFlowProvider>
      </div>

      {/* Selected-node detail strip — fires on click, shows full label + hint */}
      {selected && (
        <div className="flex items-center justify-between gap-3 px-4 py-2 border-t border-border bg-muted/30 text-xs">
          <div className="min-w-0">
            <span className="font-medium">{selected.label}</span>
            {selected.detail && <span className="ml-2 text-muted-foreground truncate">· {selected.detail}</span>}
            {selected.hint && <span className="ml-2 text-[var(--lt-accent)]">→ {selected.hint}</span>}
          </div>
          <button onClick={() => setSelected(null)} className="text-muted-foreground hover:text-foreground">
            ✕
          </button>
        </div>
      )}

      {/* Action bar: one row, scrollable horizontally if narrow */}
      <div className="border-t border-border bg-card">
        <ActionBar
          fix={actions.fix}
          launch={actions.launch}
          config={actions.config}
          acting={acting}
          emptyLabel={isZh ? '当前没有需要修复或启动的项' : 'Nothing to fix or launch right now'}
        />
      </div>

      {/* History strip: last 15 activity events from the global bus */}
      <HistoryStrip events={history} isZh={isZh} />

      {/* Footer legend + last-updated */}
      <div className="flex items-center justify-between px-4 py-2 border-t border-border text-[11px] text-muted-foreground">
        <div className="flex items-center gap-3">
          <LegendDot color="ok" label={t('topology.ok')} />
          <LegendDot color="warn" label={t('topology.degraded')} />
          <LegendDot color="err" label={t('topology.down')} />
          <LegendDot color="muted" label={t('topology.notconfigured')} />
        </div>
        {lastUpdated && (
          <span>{t('topology.lastUpdated', { time: new Date(lastUpdated).toLocaleTimeString() })}</span>
        )}
      </div>
    </div>
  )
}

// TopologyCanvas wraps the actual ReactFlow component so we can use
// useReactFlow() to call fitView() on window resize. Without this the
// diagram stays at its initial scale when the user enlarges the window —
// fitView() with maxZoom 1.8 keeps the tree comfortably filling the
// available space, both wider and taller.
function TopologyCanvas({
  nodes,
  edges,
  onNodeClick,
}: {
  nodes: RFNode[]
  edges: RFEdge[]
  onNodeClick: NodeMouseHandler
}) {
  const { fitView } = useReactFlow()

  useEffect(() => {
    // Re-fit when the window dimensions change. Debounce via rAF so a
    // fast drag doesn't fire fitView dozens of times per second.
    let raf = 0
    const handler = () => {
      cancelAnimationFrame(raf)
      raf = requestAnimationFrame(() => {
        fitView({ padding: 0.06, maxZoom: 1.8, duration: 200 })
      })
    }
    window.addEventListener('resize', handler)
    return () => {
      cancelAnimationFrame(raf)
      window.removeEventListener('resize', handler)
    }
  }, [fitView])

  // Re-fit when the node set changes (snapshot poll → relayout).
  useEffect(() => {
    const id = requestAnimationFrame(() => {
      fitView({ padding: 0.06, maxZoom: 1.8, duration: 200 })
    })
    return () => cancelAnimationFrame(id)
  }, [nodes, fitView])

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      nodeTypes={nodeTypes}
      onNodeClick={onNodeClick}
      fitView
      fitViewOptions={{ padding: 0.06, maxZoom: 1.8 }}
      minZoom={0.3}
      maxZoom={2.5}
      nodesDraggable={false}
      nodesConnectable={false}
      proOptions={{ hideAttribution: true }}
    >
      <Background gap={20} size={1} color="hsl(var(--muted-foreground) / 0.15)" />
      <Controls showInteractive={false} className="!border-border !bg-card" />
      <MiniMap
        nodeColor={(n) => {
          const status = (n.data as NodeData | undefined)?.snap?.status
          return STATUS_STROKE[status ?? 'unknown'] ?? STATUS_STROKE.unknown
        }}
        maskColor="hsl(var(--background) / 0.7)"
        className="!border-border !bg-card"
      />
    </ReactFlow>
  )
}

// ============================================================
// Action bar
// ============================================================

interface ActionItem {
  key: string
  label: string
  icon: React.ComponentType<{ className?: string }>
  tone: 'fix' | 'warn' | 'launch' | 'nav'
  run: () => Promise<void> | void
}

const TONE_CLS: Record<ActionItem['tone'], string> = {
  fix: 'border-[var(--lt-err)]/40 text-[var(--lt-err)] hover:bg-[var(--lt-err)]/10',
  warn: 'border-[var(--lt-warn)]/40 text-[var(--lt-warn)] hover:bg-[var(--lt-warn)]/10',
  launch: 'border-[var(--lt-accent)]/40 text-[var(--lt-accent)] hover:bg-[var(--lt-accent)]/10',
  nav: 'border-border text-foreground hover:bg-muted',
}

function ActionBar({
  fix,
  launch,
  config,
  acting,
  emptyLabel,
}: {
  fix: ActionItem[]
  launch: ActionItem[]
  config: ActionItem[]
  acting: string | null
  emptyLabel: string
}) {
  const isEmpty = fix.length === 0 && launch.length === 0 && config.length === 0
  if (isEmpty) {
    return <div className="px-4 py-3 text-xs text-muted-foreground italic">{emptyLabel}</div>
  }
  return (
    <div className="flex flex-wrap items-center gap-2 px-4 py-3">
      {fix.map((a) => (
        <ActionButton key={a.key} item={a} acting={acting === a.key} />
      ))}
      {fix.length > 0 && launch.length > 0 && <Divider />}
      {launch.map((a) => (
        <ActionButton key={a.key} item={a} acting={acting === a.key} />
      ))}
      {(fix.length > 0 || launch.length > 0) && config.length > 0 && <Divider />}
      {config.map((a) => (
        <ActionButton key={a.key} item={a} acting={acting === a.key} />
      ))}
    </div>
  )
}

function ActionButton({ item, acting }: { item: ActionItem; acting: boolean }) {
  const Icon = item.icon
  return (
    <button
      onClick={() => void item.run()}
      disabled={acting}
      className={cn(
        'inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-md border text-[12px] transition-colors',
        TONE_CLS[item.tone],
        'disabled:opacity-50 disabled:cursor-not-allowed',
      )}
    >
      {acting ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Icon className="h-3.5 w-3.5" />}
      <span>{item.label}</span>
    </button>
  )
}

function Divider() {
  return <span className="h-5 w-px bg-border" />
}

// ============================================================
// History strip
// ============================================================

function HistoryStrip({ events, isZh }: { events: ActivityEvent[]; isZh: boolean }) {
  if (events.length === 0) return null
  return (
    <div className="border-t border-border bg-muted/15 px-4 py-2">
      <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground mb-1.5">
        <ActivityIcon className="h-3 w-3" />
        <span>{isZh ? '最近活动' : 'Recent activity'}</span>
      </div>
      <div className="flex gap-1.5 overflow-x-auto pb-1">
        {events.map((ev) => (
          <HistoryPill key={ev.id + ev.updatedAt} ev={ev} isZh={isZh} />
        ))}
      </div>
    </div>
  )
}

function HistoryPill({ ev, isZh }: { ev: ActivityEvent; isZh: boolean }) {
  const title = (isZh ? ev.titleZh : ev.titleEn) || ev.titleEn || ev.titleZh
  let toneCls = 'border-border bg-card text-foreground'
  if (ev.phase === 'done') toneCls = 'border-[var(--lt-ok)]/40 bg-[var(--lt-ok)]/10 text-[var(--lt-ok)]'
  else if (ev.phase === 'error') toneCls = 'border-[var(--lt-err)]/40 bg-[var(--lt-err)]/10 text-[var(--lt-err)]'
  else if (ev.phase === 'progress' || ev.phase === 'start')
    toneCls = 'border-[var(--lt-accent)]/40 bg-[var(--lt-accent)]/10 text-[var(--lt-accent)]'
  const time = new Date(ev.updatedAt).toLocaleTimeString(undefined, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
  return (
    <div
      title={ev.error || (isZh ? ev.detailZh : ev.detailEn) || title}
      className={cn('inline-flex items-center gap-1.5 px-2 py-1 rounded-md border text-[11px] whitespace-nowrap', toneCls)}
    >
      <span className="opacity-60">{time}</span>
      <span className="font-medium truncate max-w-[160px]">{title}</span>
      {ev.phase === 'progress' && <Loader2 className="h-3 w-3 animate-spin" />}
    </div>
  )
}

// ============================================================
// Header bits
// ============================================================

function HeadlineIcon({ summary }: { summary: topology.Summary }) {
  if (summary.down > 0) return <XCircle className="h-4 w-4 text-[var(--lt-err)] flex-shrink-0" />
  if (summary.degraded > 0) return <AlertTriangle className="h-4 w-4 text-[var(--lt-warn)] flex-shrink-0" />
  if (summary.notconfigured > 0) return <CircleDot className="h-4 w-4 text-muted-foreground flex-shrink-0" />
  return <CheckCircle2 className="h-4 w-4 text-[var(--lt-ok)] flex-shrink-0" />
}

function Chip({ color, label, value }: { color: 'ok' | 'warn' | 'err' | 'muted'; label: string; value: number }) {
  const cls: Record<string, string> = {
    ok: 'bg-[var(--lt-ok)]/10 text-[var(--lt-ok)] border-[var(--lt-ok)]/40',
    warn: 'bg-[var(--lt-warn)]/10 text-[var(--lt-warn)] border-[var(--lt-warn)]/40',
    err: 'bg-[var(--lt-err)]/10 text-[var(--lt-err)] border-[var(--lt-err)]/40',
    muted: 'bg-muted text-muted-foreground border-border',
  }
  return (
    <span className={cn('inline-flex items-center gap-1 px-1.5 py-0.5 rounded border', cls[color])}>
      <span className="font-semibold">{value}</span>
      <span>{label}</span>
    </span>
  )
}

function LegendDot({ color, label }: { color: 'ok' | 'warn' | 'err' | 'muted'; label: string }) {
  const cls: Record<string, string> = {
    ok: 'bg-[var(--lt-ok)]',
    warn: 'bg-[var(--lt-warn)]',
    err: 'bg-[var(--lt-err)]',
    muted: 'bg-muted-foreground/40',
  }
  return (
    <span className="flex items-center gap-1">
      <span className={cn('h-2 w-2 rounded-full', cls[color])} />
      {label}
    </span>
  )
}
