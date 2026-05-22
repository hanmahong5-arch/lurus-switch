import { type ReactNode } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { AnimatePresence, motion } from 'framer-motion'
import { X, type LucideIcon } from 'lucide-react'
import { cn } from '../../lib/utils'

export type ModalSize = 'sm' | 'md' | 'lg' | 'xl'

interface ModalProps {
  open: boolean
  onClose: () => void
  title: ReactNode
  icon?: LucideIcon
  size?: ModalSize
  footer?: ReactNode
  children: ReactNode
  className?: string
}

const SIZE: Record<ModalSize, string> = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-xl',
  xl: 'max-w-2xl',
}

export function Modal({
  open, onClose, title, icon: Icon, size = 'md', footer, children, className,
}: ModalProps) {
  return (
    <AnimatePresence>
      {open && (
        <Dialog.Root open={open} onOpenChange={(o) => { if (!o) onClose() }}>
          <Dialog.Portal forceMount>
            <Dialog.Overlay asChild forceMount>
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.15 }}
                className="fixed inset-0 z-50 bg-black/40 backdrop-blur-sm"
              />
            </Dialog.Overlay>
            <Dialog.Content asChild forceMount aria-describedby={undefined}>
              <motion.div
                initial={{ opacity: 0, y: 8, scale: 0.98 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: 4, scale: 0.99 }}
                transition={{ duration: 0.2, ease: 'easeOut' }}
                className={cn(
                  'fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50',
                  'w-full max-h-[88vh] flex flex-col',
                  'bg-card-elevated border border-rule-strong rounded-xl shadow-2xl',
                  SIZE[size],
                  className,
                )}
              >
                <header className="flex items-center justify-between px-4 py-3 border-b border-border">
                  <Dialog.Title className="flex items-center gap-2 text-sm font-semibold">
                    {Icon && <Icon className="h-4 w-4 text-primary" />}
                    <span>{title}</span>
                  </Dialog.Title>
                  <button
                    onClick={onClose}
                    aria-label="Close"
                    className="h-7 w-7 inline-flex items-center justify-center rounded hover:bg-muted text-muted-foreground transition-colors"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </header>
                <div className="flex-1 overflow-y-auto p-4">{children}</div>
                {footer && (
                  <footer className="flex items-center justify-end gap-2 px-4 py-3 border-t border-border">
                    {footer}
                  </footer>
                )}
              </motion.div>
            </Dialog.Content>
          </Dialog.Portal>
        </Dialog.Root>
      )}
    </AnimatePresence>
  )
}
