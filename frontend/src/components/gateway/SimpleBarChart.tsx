interface SimpleBarChartProps {
  data: Record<string, unknown>[]
  labelKey: string
  valueKey: string
  height?: number
}

export function SimpleBarChart({
  data,
  labelKey,
  valueKey,
  height = 120,
}: SimpleBarChartProps) {
  if (data.length === 0) {
    return <p className="text-sm text-muted-foreground py-4 text-center">No data</p>
  }

  const values = data.map((d) => Number(d[valueKey]) || 0)
  const max = Math.max(...values, 1)

  return (
    <div className="flex items-end gap-1" style={{ height }}>
      {data.map((d, i) => {
        const val = values[i]
        const pct = (val / max) * 100
        const label = String(d[labelKey])
        return (
          <div key={i} className="flex-1 flex flex-col items-center gap-1 min-w-0">
            <div
              className="w-full bg-indigo-500/60 rounded-t transition-all hover:bg-indigo-400/70"
              style={{ height: `${Math.max(pct, 2)}%` }}
              title={`${label}: ${val}`}
            />
            <span className="text-[10px] text-muted-foreground truncate w-full text-center">
              {label.length > 5 ? label.slice(-5) : label}
            </span>
          </div>
        )
      })}
    </div>
  )
}
