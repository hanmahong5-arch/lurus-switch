import { describe, it, expect, beforeEach } from 'vitest'
import {
  useActivityStore,
  matchesFilter,
  unreadCount,
  type ActivityEvent,
} from './activityStore'

const ev = (overrides: Partial<ActivityEvent> = {}): ActivityEvent => ({
  id: 'ev-1',
  phase: 'done',
  titleZh: '测试',
  titleEn: 'Test',
  startedAt: '2026-05-20T10:00:00Z',
  updatedAt: '2026-05-20T10:00:01Z',
  ...overrides,
})

beforeEach(() => {
  // Reset store + persisted localStorage so each test starts clean.
  localStorage.removeItem('switch.activity-drawer')
  useActivityStore.setState({
    events: [],
    filter: 'all',
    drawerOpen: false,
    lastSeenAt: null,
  })
})

describe('activityStore.ingest', () => {
  it('prepends new events', () => {
    useActivityStore.getState().ingest(ev({ id: 'a', updatedAt: '2026-05-20T10:00:00Z' }))
    useActivityStore.getState().ingest(ev({ id: 'b', updatedAt: '2026-05-20T10:00:01Z' }))
    const events = useActivityStore.getState().events
    expect(events.map((e) => e.id)).toEqual(['b', 'a'])
  })

  it('updates in place when id already exists', () => {
    useActivityStore.getState().ingest(ev({ id: 'a', phase: 'start' }))
    useActivityStore.getState().ingest(ev({ id: 'a', phase: 'done' }))
    const events = useActivityStore.getState().events
    expect(events.length).toBe(1)
    expect(events[0].phase).toBe('done')
  })

  it('caps history at 100 entries (drops oldest settled first)', () => {
    for (let i = 0; i < 105; i++) {
      useActivityStore.getState().ingest(
        ev({
          id: `ev-${i}`,
          phase: 'done',
          updatedAt: new Date(2026, 4, 20, 10, i, 0).toISOString(),
        }),
      )
    }
    const events = useActivityStore.getState().events
    expect(events.length).toBeLessThanOrEqual(100)
  })
})

describe('activityStore.clear', () => {
  it('empties events and stamps lastSeenAt', () => {
    useActivityStore.getState().ingest(ev())
    useActivityStore.getState().clear()
    const s = useActivityStore.getState()
    expect(s.events).toEqual([])
    expect(s.lastSeenAt).toBeTruthy()
  })
})

describe('matchesFilter', () => {
  it('"all" matches everything', () => {
    expect(matchesFilter(ev({ phase: 'done' }), 'all')).toBe(true)
    expect(matchesFilter(ev({ phase: 'error' }), 'all')).toBe(true)
  })

  it('"active" matches only start/progress', () => {
    expect(matchesFilter(ev({ phase: 'start' }), 'active')).toBe(true)
    expect(matchesFilter(ev({ phase: 'progress' }), 'active')).toBe(true)
    expect(matchesFilter(ev({ phase: 'done' }), 'active')).toBe(false)
    expect(matchesFilter(ev({ phase: 'error' }), 'active')).toBe(false)
  })

  it('"error" matches errored phase or error-tagged event', () => {
    expect(matchesFilter(ev({ phase: 'error' }), 'error')).toBe(true)
    expect(matchesFilter(ev({ tags: ['error'] }), 'error')).toBe(true)
    expect(matchesFilter(ev({ phase: 'done' }), 'error')).toBe(false)
  })

  it('"mutation" matches only mutation-tagged events', () => {
    expect(matchesFilter(ev({ tags: ['mutation'] }), 'mutation')).toBe(true)
    expect(matchesFilter(ev({ tags: ['auth'] }), 'mutation')).toBe(false)
    expect(matchesFilter(ev({}), 'mutation')).toBe(false)
  })

  it('"system" matches events without business tags', () => {
    expect(matchesFilter(ev({}), 'system')).toBe(true)
    expect(matchesFilter(ev({ tags: [] }), 'system')).toBe(true)
    expect(matchesFilter(ev({ tags: ['mutation'] }), 'system')).toBe(false)
    expect(matchesFilter(ev({ tags: ['auth'] }), 'system')).toBe(false)
  })
})

describe('unreadCount', () => {
  it('returns full count when no lastSeenAt', () => {
    const events = [ev({ id: 'a' }), ev({ id: 'b' })]
    expect(unreadCount(events, null)).toBe(2)
  })

  it('returns 0 when all events predate lastSeenAt', () => {
    const events = [
      ev({ updatedAt: '2026-05-20T09:00:00Z' }),
      ev({ updatedAt: '2026-05-20T09:30:00Z' }),
    ]
    expect(unreadCount(events, '2026-05-20T10:00:00Z')).toBe(0)
  })

  it('counts only events newer than lastSeenAt', () => {
    const events = [
      ev({ id: 'a', updatedAt: '2026-05-20T09:00:00Z' }),
      ev({ id: 'b', updatedAt: '2026-05-20T11:00:00Z' }),
      ev({ id: 'c', updatedAt: '2026-05-20T11:30:00Z' }),
    ]
    expect(unreadCount(events, '2026-05-20T10:00:00Z')).toBe(2)
  })
})
