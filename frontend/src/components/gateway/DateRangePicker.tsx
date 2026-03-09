interface DateRangePickerProps {
  start: string
  end: string
  onChange: (start: string, end: string) => void
}

export function DateRangePicker({ start, end, onChange }: DateRangePickerProps) {
  return (
    <div className="flex items-center gap-2 text-sm">
      <input
        type="date"
        value={start}
        onChange={(e) => onChange(e.target.value, end)}
        className="px-2 py-1 rounded border border-border bg-background text-sm focus:outline-none focus:ring-1 focus:ring-primary"
      />
      <span className="text-muted-foreground">—</span>
      <input
        type="date"
        value={end}
        onChange={(e) => onChange(start, e.target.value)}
        className="px-2 py-1 rounded border border-border bg-background text-sm focus:outline-none focus:ring-1 focus:ring-primary"
      />
    </div>
  )
}
