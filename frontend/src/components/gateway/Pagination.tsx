import { ChevronLeft, ChevronRight } from 'lucide-react'
import { cn } from '../../lib/utils'

interface PaginationProps {
  page: number
  total: number
  perPage: number
  onPageChange: (page: number) => void
}

export function Pagination({ page, total, perPage, onPageChange }: PaginationProps) {
  const totalPages = Math.max(1, Math.ceil(total / perPage))
  if (totalPages <= 1) return null

  const pages: (number | '...')[] = []
  for (let i = 0; i < totalPages; i++) {
    if (i === 0 || i === totalPages - 1 || Math.abs(i - page) <= 1) {
      pages.push(i)
    } else if (pages[pages.length - 1] !== '...') {
      pages.push('...')
    }
  }

  return (
    <div className="flex items-center gap-1 text-xs">
      <button
        onClick={() => onPageChange(Math.max(0, page - 1))}
        disabled={page === 0}
        className="p-1 rounded hover:bg-muted disabled:opacity-30"
      >
        <ChevronLeft className="h-4 w-4" />
      </button>
      {pages.map((p, i) =>
        p === '...' ? (
          <span key={`e${i}`} className="px-1.5 text-muted-foreground">...</span>
        ) : (
          <button
            key={p}
            onClick={() => onPageChange(p)}
            className={cn(
              'min-w-[28px] h-7 rounded text-center',
              p === page ? 'bg-primary text-primary-foreground' : 'hover:bg-muted text-muted-foreground'
            )}
          >
            {p + 1}
          </button>
        )
      )}
      <button
        onClick={() => onPageChange(Math.min(totalPages - 1, page + 1))}
        disabled={page >= totalPages - 1}
        className="p-1 rounded hover:bg-muted disabled:opacity-30"
      >
        <ChevronRight className="h-4 w-4" />
      </button>
      <span className="ml-2 text-muted-foreground">{total} items</span>
    </div>
  )
}
