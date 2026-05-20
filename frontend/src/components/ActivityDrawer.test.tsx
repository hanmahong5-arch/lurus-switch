import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallbackOrOpts?: string | Record<string, unknown>) => {
      return typeof fallbackOrOpts === 'string' ? fallbackOrOpts : key
    },
    i18n: { language: 'zh', changeLanguage: vi.fn() },
  }),
  Trans: ({ children, i18nKey }: { children?: React.ReactNode; i18nKey?: string }) =>
    children ?? i18nKey ?? null,
  initReactI18next: { type: '3rdParty', init: () => {} },
}))

vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn(() => () => undefined),
}))

import { ActivityDrawer } from './ActivityDrawer'
import { useActivityStore, type ActivityEvent } from '../stores/activityStore'

const ev = (overrides: Partial<ActivityEvent> = {}): ActivityEvent => ({
  id: 'ev-' + Math.random().toString(36).slice(2, 8),
  phase: 'done',
  titleZh: '测试事件',
  titleEn: 'Test event',
  startedAt: '2026-05-20T10:00:00Z',
  updatedAt: '2026-05-20T10:00:01Z',
  ...overrides,
})

beforeEach(() => {
  localStorage.removeItem('switch.activity-drawer')
  useActivityStore.setState({
    events: [],
    filter: 'all',
    drawerOpen: false,
    lastSeenAt: null,
  })
})

describe('ActivityDrawer', () => {
  it('renders hidden by default (aria-hidden true)', () => {
    render(<ActivityDrawer />)
    const drawer = screen.getByTestId('activity-drawer')
    expect(drawer.getAttribute('aria-hidden')).toBe('true')
  })

  it('opens when drawerOpen flips true', () => {
    useActivityStore.setState({ drawerOpen: true })
    render(<ActivityDrawer />)
    const drawer = screen.getByTestId('activity-drawer')
    expect(drawer.getAttribute('aria-hidden')).toBe('false')
  })

  it('shows "no activity" placeholder when empty', () => {
    useActivityStore.setState({ drawerOpen: true })
    render(<ActivityDrawer />)
    expect(screen.getByText(/暂无活动记录/)).toBeInTheDocument()
  })

  it('lists events in reverse chronological order', () => {
    useActivityStore.setState({
      drawerOpen: true,
      events: [
        ev({ id: 'a', titleZh: '事件 A', updatedAt: '2026-05-20T10:00:00Z' }),
        ev({ id: 'b', titleZh: '事件 B', updatedAt: '2026-05-20T10:01:00Z' }),
        ev({ id: 'c', titleZh: '事件 C', updatedAt: '2026-05-20T10:02:00Z' }),
      ],
    })
    render(<ActivityDrawer />)
    const titles = screen.getAllByText(/事件 [ABC]/).map((el) => el.textContent)
    expect(titles).toEqual(['事件 C', '事件 B', '事件 A'])
  })

  it('filters by mutation tag', () => {
    useActivityStore.setState({
      drawerOpen: true,
      filter: 'mutation',
      events: [
        ev({ id: 'a', titleZh: '配置变更', tags: ['mutation'] }),
        ev({ id: 'b', titleZh: '系统就绪' }),
      ],
    })
    render(<ActivityDrawer />)
    expect(screen.getByText('配置变更')).toBeInTheDocument()
    expect(screen.queryByText('系统就绪')).not.toBeInTheDocument()
  })

  it('filters by error phase', () => {
    useActivityStore.setState({
      drawerOpen: true,
      filter: 'error',
      events: [
        ev({ id: 'a', titleZh: '错误事件', phase: 'error', error: 'oops' }),
        ev({ id: 'b', titleZh: '成功事件', phase: 'done' }),
      ],
    })
    render(<ActivityDrawer />)
    expect(screen.getByText('错误事件')).toBeInTheDocument()
    expect(screen.queryByText('成功事件')).not.toBeInTheDocument()
  })

  it('clears history when clear button clicked', () => {
    useActivityStore.setState({
      drawerOpen: true,
      events: [ev({ titleZh: '需清空' })],
    })
    render(<ActivityDrawer />)
    expect(screen.getByText('需清空')).toBeInTheDocument()
    const clearBtn = screen.getByTitle(/清空历史/)
    fireEvent.click(clearBtn)
    expect(useActivityStore.getState().events).toEqual([])
  })

  it('closes drawer when close button clicked', () => {
    useActivityStore.setState({ drawerOpen: true })
    render(<ActivityDrawer />)
    const closeBtn = screen.getByTitle(/关闭/)
    fireEvent.click(closeBtn)
    expect(useActivityStore.getState().drawerOpen).toBe(false)
  })
})
