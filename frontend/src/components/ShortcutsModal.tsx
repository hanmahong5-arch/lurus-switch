import { useEffect } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { useTranslation } from 'react-i18next'
import { X, Keyboard } from 'lucide-react'

interface Props {
  open: boolean
  onClose: () => void
}

// Surfaces the keyboard shortcuts users actually have. Plain table — no
// fancy categorisation; if it grows past a screenful we'll group.
export function ShortcutsModal({ open, onClose }: Props) {
  const { t, i18n } = useTranslation()
  const isMac = typeof navigator !== 'undefined' && /Mac/i.test(navigator.platform)
  const mod = isMac ? '⌘' : 'Ctrl'

  const items: Array<{ keys: string; desc: string }> = [
    { keys: `${mod} K`, desc: t('shortcuts.cmdPalette', '打开命令面板') },
    { keys: `${mod} 1 – 5`, desc: t('shortcuts.switchPage', '切换主页面（首页 / 助理 / 会话 / 工具 / 网关）') },
    { keys: `${mod} S`, desc: t('shortcuts.save', '保存当前编辑') },
    { keys: 'Esc', desc: t('shortcuts.escape', '关闭弹窗 / 退出当前模式') },
    { keys: `${mod} C`, desc: t('shortcuts.copy', '复制选中（也可点顶栏复制按钮）') },
    { keys: `${mod} F`, desc: t('shortcuts.findInPage', '在当前内容中查找') },
  ]

  // Close on Escape (Dialog handles this but be explicit for clarity).
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose])

  const isZh = (i18n.language || '').startsWith('zh')

  return (
    <Dialog.Root open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/30 backdrop-blur-sm z-40" />
        <Dialog.Content className="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 w-[440px] max-w-[90vw] rounded-lg border border-border bg-card shadow-xl">
          <div className="flex items-center justify-between px-4 py-3 border-b border-border">
            <Dialog.Title className="text-sm font-semibold flex items-center gap-2">
              <Keyboard className="h-4 w-4" />
              {isZh ? '键盘快捷键' : 'Keyboard shortcuts'}
            </Dialog.Title>
            <button
              onClick={onClose}
              className="p-1 rounded hover:bg-muted text-muted-foreground"
              aria-label="Close"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <div className="p-4 space-y-2">
            {items.map((it) => (
              <div key={it.keys} className="flex items-center justify-between gap-3 text-xs">
                <span className="text-muted-foreground">{it.desc}</span>
                <kbd className="px-2 py-0.5 rounded border border-border bg-muted font-mono text-[11px]">{it.keys}</kbd>
              </div>
            ))}
          </div>
          <div className="px-4 py-2 border-t border-border text-[11px] text-muted-foreground">
            {isZh
              ? '提示：顶栏按钮可点 — 复制选中文本、快速启动 CLI、刷新当前页、命令面板。'
              : 'Tip: the top bar exposes copy-selection, quick CLI launch, page refresh, and the command palette.'}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
