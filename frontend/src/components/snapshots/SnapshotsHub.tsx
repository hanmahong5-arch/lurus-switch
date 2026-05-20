import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import * as Dialog from '@radix-ui/react-dialog'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import {
  X, Camera, RotateCcw, Trash2, Loader2, Clock, AlertCircle,
  CheckCircle2,
} from 'lucide-react'
import { cn } from '../../lib/utils'
import { TOOL_ORDER, TOOL_DISPLAY } from '../../lib/toolMeta'
import { useSnapshotsHubStore } from '../../stores/snapshotsHubStore'
import { useToastStore } from '../../stores/toastStore'
import {
  ListConfigSnapshots,
  TakeConfigSnapshot,
  RestoreConfigSnapshot,
  DeleteConfigSnapshot,
} from '../../../wailsjs/go/main/App'
import type { snapshot } from '../../../wailsjs/go/models'

// One snapshot row, augmented with the tool it belongs to. The Wails
// SnapshotMeta has a `tool` field already, but we re-key the array by
// tool in the hub so the UI can render grouped sections.
type Snap = snapshot.SnapshotMeta

interface ToolBucket {
  tool: string
  display: string
  snaps: Snap[]
}

// Restore confirmation modal state. When non-null, blocks the hub UI
// until the user confirms or cancels.
interface PendingRestore {
  tool: string
  toolDisplay: string
  snap: Snap
}

export function SnapshotsHub() {
  const { t } = useTranslation()
  const open = useSnapshotsHubStore((s) => s.open)
  const setOpen = useSnapshotsHubStore((s) => s.setOpen)
  const focusTool = useSnapshotsHubStore((s) => s.focusTool)
  const toast = useToastStore((s) => s.addToast)

  const [loading, setLoading] = useState(false)
  const [buckets, setBuckets] = useState<ToolBucket[]>([])
  const [selectedTool, setSelectedTool] = useState<string>('')
  const [busyId, setBusyId] = useState<string | null>(null)
  const [pendingRestore, setPendingRestore] = useState<PendingRestore | null>(null)
  const [creatingFor, setCreatingFor] = useState<string | null>(null)
  const [labelDraft, setLabelDraft] = useState('')

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const results = await Promise.all(
        TOOL_ORDER.map(async (tool) => {
          try {
            const snaps = (await ListConfigSnapshots(tool)) ?? []
            return { tool, snaps }
          } catch {
            // A missing config dir or unknown tool isn't an error — show
            // it as an empty bucket so the user can still take a snapshot
            // for that tool from the hub.
            return { tool, snaps: [] as Snap[] }
          }
        }),
      )
      setBuckets(
        results.map(({ tool, snaps }) => ({
          tool,
          display: TOOL_DISPLAY[tool] ?? tool,
          snaps: [...snaps].sort(
            (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
          ),
        })),
      )
    } finally {
      setLoading(false)
    }
  }, [])

  // Refresh on open + when focus tool changes.
  useEffect(() => {
    if (!open) return
    setSelectedTool(focusTool ?? '')
    refresh()
  }, [open, focusTool, refresh])

  const visibleBuckets = useMemo(
    () => (selectedTool ? buckets.filter((b) => b.tool === selectedTool) : buckets),
    [buckets, selectedTool],
  )

  const totalCount = useMemo(
    () => buckets.reduce((s, b) => s + b.snaps.length, 0),
    [buckets],
  )

  const handleTakeForTool = async (tool: string) => {
    const display = TOOL_DISPLAY[tool] ?? tool
    setBusyId(`take-${tool}`)
    try {
      await TakeConfigSnapshot(
        tool,
        labelDraft.trim() ||
          t('snapshotsHub.defaultLabel', '手动快照 {{date}}', {
            date: new Date().toLocaleString(),
          }),
      )
      setLabelDraft('')
      setCreatingFor(null)
      toast('success', t('snapshotsHub.takeSuccess', '已为 {{tool}} 创建快照', { tool: display }))
      await refresh()
    } catch (e) {
      toast('error', t('snapshotsHub.takeFailed', '快照创建失败：{{err}}', { err: String(e) }))
    } finally {
      setBusyId(null)
    }
  }

  const handleRestore = async () => {
    if (!pendingRestore) return
    const { tool, toolDisplay, snap } = pendingRestore
    setBusyId(`restore-${snap.id}`)
    setPendingRestore(null)
    try {
      await RestoreConfigSnapshot(tool, snap.id)
      toast('success', t('snapshotsHub.restoreSuccess', '已恢复 {{tool}} 到 {{label}}', {
        tool: toolDisplay,
        label: snap.label || snap.id,
      }))
      await refresh()
    } catch (e) {
      toast('error', t('snapshotsHub.restoreFailed', '恢复失败：{{err}}', { err: String(e) }))
    } finally {
      setBusyId(null)
    }
  }

  const handleDelete = async (tool: string, snap: Snap) => {
    if (
      !confirm(
        t('snapshotsHub.deleteConfirm', '确定删除快照 {{label}}？此操作不可恢复。', {
          label: snap.label || snap.id,
        }),
      )
    )
      return
    setBusyId(`del-${snap.id}`)
    try {
      await DeleteConfigSnapshot(tool, snap.id)
      toast('success', t('snapshotsHub.deleteSuccess', '已删除快照'))
      await refresh()
    } catch (e) {
      toast('error', t('snapshotsHub.deleteFailed', '删除失败：{{err}}', { err: String(e) }))
    } finally {
      setBusyId(null)
    }
  }

  const formatBytes = (n: number) => {
    if (n < 1024) return `${n} B`
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
    return `${(n / 1024 / 1024).toFixed(1)} MB`
  }

  return (
    <>
      <Dialog.Root open={open} onOpenChange={setOpen}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50 animate-in fade-in-0" />
          <Dialog.Content
            data-testid="snapshots-hub"
            className="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-[860px] max-w-[95vw] max-h-[85vh] bg-card border border-border rounded-xl shadow-2xl z-50 flex flex-col overflow-hidden"
            aria-describedby={undefined}
          >
            <VisuallyHidden>
              <Dialog.Title>{t('snapshotsHub.title', '配置快照中心')}</Dialog.Title>
            </VisuallyHidden>

            {/* Header */}
            <header className="flex items-center justify-between px-5 py-3 border-b border-border">
              <div className="flex items-center gap-2">
                <Camera className="h-4 w-4 text-primary" />
                <h2 className="text-sm font-semibold">{t('snapshotsHub.title', '配置快照中心')}</h2>
                <span className="text-[11px] text-muted-foreground tabular-nums">
                  {t('snapshotsHub.total', '共 {{n}} 个', { n: totalCount })}
                </span>
              </div>
              <button
                onClick={() => setOpen(false)}
                className="h-7 w-7 inline-flex items-center justify-center rounded text-muted-foreground hover:bg-muted"
                title={t('common.close', '关闭')}
              >
                <X className="h-4 w-4" />
              </button>
            </header>

            {/* Body */}
            <div className="flex-1 overflow-hidden grid grid-cols-[180px_1fr]">
              {/* Tool sidebar */}
              <nav className="border-r border-border overflow-y-auto p-2 space-y-0.5">
                <button
                  onClick={() => setSelectedTool('')}
                  className={cn(
                    'w-full text-left px-3 py-1.5 rounded text-xs',
                    selectedTool === ''
                      ? 'bg-primary/10 text-primary font-medium'
                      : 'text-muted-foreground hover:bg-muted',
                  )}
                >
                  {t('snapshotsHub.allTools', '全部工具')}
                  <span className="ml-2 text-[10px] tabular-nums opacity-70">{totalCount}</span>
                </button>
                {buckets.map((b) => (
                  <button
                    key={b.tool}
                    onClick={() => setSelectedTool(b.tool)}
                    className={cn(
                      'w-full text-left px-3 py-1.5 rounded text-xs flex items-center justify-between',
                      selectedTool === b.tool
                        ? 'bg-primary/10 text-primary font-medium'
                        : 'text-muted-foreground hover:bg-muted',
                    )}
                  >
                    <span className="truncate">{b.display}</span>
                    <span className="text-[10px] tabular-nums opacity-70">{b.snaps.length}</span>
                  </button>
                ))}
              </nav>

              {/* Content */}
              <div className="overflow-y-auto p-4">
                {loading && (
                  <div className="flex items-center justify-center py-10 text-muted-foreground">
                    <Loader2 className="h-4 w-4 animate-spin" />
                  </div>
                )}

                {!loading &&
                  visibleBuckets.map((b) => (
                    <section key={b.tool} className="mb-6 last:mb-0">
                      <header className="flex items-center justify-between mb-2">
                        <h3 className="text-sm font-medium">{b.display}</h3>
                        {creatingFor === b.tool ? (
                          <div className="flex items-center gap-1">
                            <input
                              autoFocus
                              value={labelDraft}
                              onChange={(e) => setLabelDraft(e.target.value)}
                              placeholder={t('snapshotsHub.labelPlaceholder', '快照标签（可选）')}
                              className="px-2 py-1 text-[11px] bg-background border border-border rounded w-44 focus:outline-none focus:ring-1 focus:ring-primary"
                              onKeyDown={(e) => {
                                if (e.key === 'Enter') handleTakeForTool(b.tool)
                                if (e.key === 'Escape') {
                                  setCreatingFor(null)
                                  setLabelDraft('')
                                }
                              }}
                            />
                            <button
                              onClick={() => handleTakeForTool(b.tool)}
                              disabled={busyId === `take-${b.tool}`}
                              className="px-2 py-1 rounded text-[11px] bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
                            >
                              {busyId === `take-${b.tool}` ? (
                                <Loader2 className="h-3 w-3 animate-spin" />
                              ) : (
                                t('snapshotsHub.save', '保存')
                              )}
                            </button>
                            <button
                              onClick={() => {
                                setCreatingFor(null)
                                setLabelDraft('')
                              }}
                              className="px-2 py-1 rounded text-[11px] border border-border hover:bg-muted"
                            >
                              {t('common.cancel', '取消')}
                            </button>
                          </div>
                        ) : (
                          <button
                            onClick={() => setCreatingFor(b.tool)}
                            className="px-2 py-1 rounded text-[11px] border border-border text-muted-foreground hover:bg-muted inline-flex items-center gap-1"
                            title={t('snapshotsHub.takeNew', '为 {{tool}} 创建快照', { tool: b.display })}
                          >
                            <Camera className="h-3 w-3" />
                            {t('snapshotsHub.takeNewShort', '新建快照')}
                          </button>
                        )}
                      </header>

                      {b.snaps.length === 0 ? (
                        <p className="text-xs text-muted-foreground italic px-2 py-3 border border-dashed border-border rounded">
                          {t('snapshotsHub.emptyTool', '{{tool}} 暂无快照', { tool: b.display })}
                        </p>
                      ) : (
                        <ul className="space-y-1.5">
                          {b.snaps.map((snap) => (
                            <li
                              key={snap.id}
                              className="border border-border rounded-lg p-3 hover:border-primary/40 transition-colors"
                            >
                              <div className="flex items-start justify-between gap-3">
                                <div className="min-w-0 flex-1">
                                  <div className="text-sm font-medium truncate" title={snap.label || snap.id}>
                                    {snap.label || snap.id}
                                  </div>
                                  <div className="text-[11px] text-muted-foreground mt-0.5 flex items-center gap-2 flex-wrap">
                                    <span className="inline-flex items-center gap-1">
                                      <Clock className="h-2.5 w-2.5" />
                                      {new Date(snap.createdAt).toLocaleString()}
                                    </span>
                                    <span className="opacity-70 font-mono">{formatBytes(snap.size)}</span>
                                  </div>
                                </div>
                                <div className="flex items-center gap-1 shrink-0">
                                  <button
                                    onClick={() =>
                                      setPendingRestore({
                                        tool: b.tool,
                                        toolDisplay: b.display,
                                        snap,
                                      })
                                    }
                                    disabled={busyId === `restore-${snap.id}`}
                                    className="px-2 py-1 rounded text-[11px] font-medium bg-primary/10 text-primary hover:bg-primary/20 disabled:opacity-50 inline-flex items-center gap-1"
                                    title={t('snapshotsHub.restore', '恢复')}
                                  >
                                    {busyId === `restore-${snap.id}` ? (
                                      <Loader2 className="h-3 w-3 animate-spin" />
                                    ) : (
                                      <RotateCcw className="h-3 w-3" />
                                    )}
                                    {t('snapshotsHub.restore', '恢复')}
                                  </button>
                                  <button
                                    onClick={() => handleDelete(b.tool, snap)}
                                    disabled={busyId === `del-${snap.id}`}
                                    className="px-2 py-1 rounded text-[11px] border border-border text-muted-foreground hover:text-red-400 hover:bg-red-950/20 disabled:opacity-50"
                                    title={t('snapshotsHub.delete', '删除')}
                                  >
                                    {busyId === `del-${snap.id}` ? (
                                      <Loader2 className="h-3 w-3 animate-spin" />
                                    ) : (
                                      <Trash2 className="h-3 w-3" />
                                    )}
                                  </button>
                                </div>
                              </div>
                            </li>
                          ))}
                        </ul>
                      )}
                    </section>
                  ))}

                {!loading && visibleBuckets.length === 0 && (
                  <p className="text-sm text-muted-foreground italic text-center py-10">
                    {t('snapshotsHub.empty', '尚无任何快照。新建一个吧。')}
                  </p>
                )}
              </div>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>

      {/* Restore confirmation dialog — surfaces "what's about to change" so
          the user sees the cost of rolling back before they commit. */}
      <Dialog.Root open={!!pendingRestore} onOpenChange={(v) => !v && setPendingRestore(null)}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-black/60 z-[60]" />
          <Dialog.Content
            data-testid="snapshots-restore-confirm"
            className="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-[480px] max-w-[95vw] bg-card border border-border rounded-xl shadow-2xl z-[70] p-5"
            aria-describedby={undefined}
          >
            <Dialog.Title className="text-sm font-semibold flex items-center gap-2 mb-2">
              <AlertCircle className="h-4 w-4 text-amber-400" />
              {t('snapshotsHub.confirmTitle', '确认恢复快照？')}
            </Dialog.Title>
            {pendingRestore && (
              <div className="space-y-3 text-xs">
                <p className="text-muted-foreground">
                  {t('snapshotsHub.confirmIntro', '此操作会覆盖当前 {{tool}} 的配置：', {
                    tool: pendingRestore.toolDisplay,
                  })}
                </p>
                <ul className="border border-border rounded p-3 bg-muted/30 space-y-1.5">
                  <li className="flex items-center gap-2">
                    <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400 shrink-0" />
                    <span>
                      {t('snapshotsHub.willChange', '{{tool}} 的本地配置文件', {
                        tool: pendingRestore.toolDisplay,
                      })}
                    </span>
                  </li>
                  <li className="flex items-center gap-2 text-muted-foreground/70">
                    <X className="h-3.5 w-3.5 text-muted-foreground/50 shrink-0" />
                    <span>
                      {t('snapshotsHub.wontChange', '其他工具 / Hub 账号 / 已下发的渠道')}
                    </span>
                  </li>
                </ul>
                <p className="text-muted-foreground">
                  {t('snapshotsHub.restoreTo', '将恢复到 {{label}}（{{date}}）', {
                    label: pendingRestore.snap.label || pendingRestore.snap.id,
                    date: new Date(pendingRestore.snap.createdAt).toLocaleString(),
                  })}
                </p>
                <div className="flex justify-end gap-2 pt-1">
                  <button
                    onClick={() => setPendingRestore(null)}
                    className="px-3 py-1.5 rounded text-xs border border-border hover:bg-muted"
                  >
                    {t('common.cancel', '取消')}
                  </button>
                  <button
                    data-testid="snapshots-restore-confirm-btn"
                    onClick={handleRestore}
                    className="px-3 py-1.5 rounded text-xs bg-primary text-primary-foreground hover:bg-primary/90 inline-flex items-center gap-1"
                  >
                    <RotateCcw className="h-3 w-3" />
                    {t('snapshotsHub.restoreNow', '立即恢复')}
                  </button>
                </div>
              </div>
            )}
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </>
  )
}
