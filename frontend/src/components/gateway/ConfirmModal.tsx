interface ConfirmModalProps {
  open: boolean
  title: string
  desc: string
  danger?: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmModal({ open, title, desc, danger, onConfirm, onCancel }: ConfirmModalProps) {
  if (!open) return null

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="bg-card border border-border rounded-lg p-6 w-96 space-y-4">
        <h3 className="font-semibold text-lg">{title}</h3>
        <p className="text-sm text-muted-foreground">{desc}</p>
        <div className="flex justify-end gap-2 pt-2">
          <button
            onClick={onCancel}
            className="px-4 py-1.5 rounded border border-border text-sm hover:bg-muted"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className={`px-4 py-1.5 rounded text-sm text-white ${
              danger
                ? 'bg-red-600 hover:bg-red-500'
                : 'bg-indigo-600 hover:bg-indigo-500'
            }`}
          >
            Confirm
          </button>
        </div>
      </div>
    </div>
  )
}
