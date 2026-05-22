import { useEffect, useState } from 'react'

// useTickingNow — returns a Date.now() value that re-renders the caller at
// a steady cadence (default 1 Hz). The intended use is a single page-level
// timer that drives many time-derived UI bits (stopwatch labels, decay
// styling, etc.); pushing the timer below per-row keeps each row's render
// cost independent of timer overhead.
//
// The interval is cleared on unmount so navigation away from the page
// does not leak a setInterval. We snapshot Date.now() once at mount so
// the first render already reflects "now" rather than the previous mount.
export function useTickingNow(intervalMs: number = 1000): number {
  const [now, setNow] = useState<number>(() => Date.now())

  useEffect(() => {
    // Guard against pathological intervals — anything <50ms is almost
    // certainly a mistake and would burn CPU. Clamp to a sane minimum.
    const ms = Math.max(50, intervalMs)
    const id = setInterval(() => setNow(Date.now()), ms)
    return () => clearInterval(id)
  }, [intervalMs])

  return now
}
