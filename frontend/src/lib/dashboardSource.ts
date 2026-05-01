// dashboardSource.ts — mode-aware data source for the Gateway dashboard.
//
// Both Personal and Reseller modes return the same logical bundle —
// summary counters + 14-day quota series + runtime perf snapshot. The Hub
// implementation routes through Wails bindings (which talk to the
// configured Reseller Hub URL); local talks to the in-process gateway.

import { GatewayAPI, type GatewayDashboardData, type GatewayQuotaDate, type GatewayPerformanceStats } from './gateway-api'
import {
  HubGetDashboardSummary,
  HubGetQuotaDates,
  HubGetPerformanceStats,
} from '../../wailsjs/go/main/App'

export interface DashboardBundle {
  summary: GatewayDashboardData | null
  quota: GatewayQuotaDate[]
  performance: GatewayPerformanceStats | null
}

export interface DashboardSource {
  readonly kind: 'local' | 'hub'
  fetch(startDate: string, endDate: string): Promise<DashboardBundle>
}

class LocalDashboardSource implements DashboardSource {
  readonly kind = 'local' as const
  private api: GatewayAPI

  constructor(baseURL: string, token: string) {
    this.api = new GatewayAPI(baseURL, token)
  }

  async fetch(startDate: string, endDate: string): Promise<DashboardBundle> {
    const [s, q, p] = await Promise.all([
      this.api.getDashboardData(),
      this.api.getQuotaDates(startDate, endDate),
      this.api.getPerformanceStats(),
    ])
    return {
      summary: s.data ?? null,
      quota: q.data ?? [],
      performance: p.data ?? null,
    }
  }
}

class HubDashboardSource implements DashboardSource {
  readonly kind = 'hub' as const

  async fetch(startDate: string, endDate: string): Promise<DashboardBundle> {
    // Run in parallel; if any single call rejects we let the caller see the
    // first error (Promise.all semantic) rather than papering over a
    // partial dashboard.
    const [s, q, p] = await Promise.all([
      HubGetDashboardSummary(),
      HubGetQuotaDates(startDate, endDate),
      HubGetPerformanceStats(),
    ])
    return {
      summary: {
        user_count: s.user_count,
        channel_count: s.channel_count,
        token_count: s.token_count,
        today_request: Number(s.today_request),
        today_quota: Number(s.today_quota),
        today_tokens: Number(s.today_tokens),
      },
      quota: (q ?? []).map((row) => ({
        date: row.date,
        quota: Number(row.quota),
        request_count: row.request_count,
        token_count: Number(row.token_count),
        model_usage: row.model_usage ?? {},
      })),
      performance: {
        goroutines: p.goroutines,
        memory_alloc: Number(p.memory_alloc),
        uptime: Number(p.uptime),
        requests_total: Number(p.requests_total),
        requests_per_sec: p.requests_per_sec,
      },
    }
  }
}

export type { GatewayDashboardData, GatewayQuotaDate, GatewayPerformanceStats } from './gateway-api'

export function makeDashboardSource(args:
  | { mode: 'local'; baseURL: string; token: string }
  | { mode: 'hub' }
): DashboardSource {
  if (args.mode === 'hub') return new HubDashboardSource()
  return new LocalDashboardSource(args.baseURL, args.token)
}
