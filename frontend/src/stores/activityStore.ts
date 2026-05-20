import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'

// Mirrors internal/activity.Event wire format. Keep in sync with the Go
// struct (activity:event payload) — ActivityPane.tsx has the same shape.
export interface ActivityEvent {
  id: string
  phase: 'start' | 'progress' | 'done' | 'error'
  titleZh: string
  titleEn: string
  detailZh?: string
  detailEn?: string
  progress?: number
  total?: number
  step?: number
  error?: string
  startedAt: string
  updatedAt: string
  // Free-form tags assigned by the emitter. We use this for filter
  // categories. Common values: mutation / error / auth / system / install.
  // Missing → treated as "system".
  tags?: string[]
}

// Top-level filter buckets surfaced in the drawer UI. Maps loosely to
// ActivityEvent.tags + phase. The "all" bucket bypasses filtering.
export type ActivityFilter = 'all' | 'active' | 'mutation' | 'error' | 'auth' | 'system'

// Persist up to this many events. Beyond this we drop oldest settled
// entries first so an active op never gets evicted by old history.
const HISTORY_CAP = 100

interface State {
  events: ActivityEvent[]
  filter: ActivityFilter
  drawerOpen: boolean
  // Last-seen timestamp the user dismissed the drawer with — used to
  // compute the "unread" badge on the sidebar button.
  lastSeenAt: string | null

  ingest: (ev: ActivityEvent) => void
  clear: () => void
  setFilter: (f: ActivityFilter) => void
  setDrawerOpen: (open: boolean) => void
  markAllSeen: () => void
}

// Filter predicate. Pure function — exported so tests can drive it
// without mounting the store.
export function matchesFilter(ev: ActivityEvent, filter: ActivityFilter): boolean {
  if (filter === 'all') return true
  if (filter === 'active') return ev.phase === 'start' || ev.phase === 'progress'
  if (filter === 'error') return ev.phase === 'error' || ev.tags?.includes('error') === true
  if (filter === 'mutation') return ev.tags?.includes('mutation') === true
  if (filter === 'auth') return ev.tags?.includes('auth') === true
  if (filter === 'system') {
    // System bucket = anything with no specific business tag.
    const t = ev.tags ?? []
    return !t.includes('mutation') && !t.includes('auth') && !t.includes('error')
  }
  return true
}

// unreadCount: pure helper used by the sidebar badge.
export function unreadCount(events: ActivityEvent[], lastSeenAt: string | null): number {
  if (!lastSeenAt) return events.length
  const cutoff = new Date(lastSeenAt).getTime()
  return events.filter((e) => new Date(e.updatedAt).getTime() > cutoff).length
}

export const useActivityStore = create<State>()(
  persist(
    (set) => ({
      events: [],
      filter: 'all',
      drawerOpen: false,
      lastSeenAt: null,

      ingest: (ev) =>
        set((s) => {
          // Update-in-place if we already have this id (start → progress →
          // done lifecycle); otherwise prepend.
          const idx = s.events.findIndex((e) => e.id === ev.id)
          let next: ActivityEvent[]
          if (idx >= 0) {
            next = s.events.slice()
            next[idx] = ev
          } else {
            next = [ev, ...s.events]
          }
          // Cap history. Drop oldest settled first.
          if (next.length > HISTORY_CAP) {
            const settled = next
              .map((e, i) => ({ e, i }))
              .filter(({ e }) => e.phase === 'done' || e.phase === 'error')
              .sort((a, b) => new Date(a.e.updatedAt).getTime() - new Date(b.e.updatedAt).getTime())
            const toDropIdx = new Set<number>()
            for (const { i } of settled) {
              if (next.length - toDropIdx.size <= HISTORY_CAP) break
              toDropIdx.add(i)
            }
            next = next.filter((_, i) => !toDropIdx.has(i))
            // If active ops alone exceed the cap, fall back to chronological
            // trim so the array can't grow without bound.
            if (next.length > HISTORY_CAP) next = next.slice(0, HISTORY_CAP)
          }
          return { events: next }
        }),

      clear: () => set({ events: [], lastSeenAt: new Date().toISOString() }),
      setFilter: (filter) => set({ filter }),
      setDrawerOpen: (drawerOpen) => set({ drawerOpen }),
      markAllSeen: () => set({ lastSeenAt: new Date().toISOString() }),
    }),
    {
      name: 'switch.activity-drawer',
      storage: createJSONStorage(() => localStorage),
      // Only persist the fields that make sense across reloads. Filter +
      // drawer-open are session-local.
      partialize: (s) => ({ events: s.events, lastSeenAt: s.lastSeenAt }),
    },
  ),
)
