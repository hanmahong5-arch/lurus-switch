// redemptionSource.ts — mode-aware data source for redemption codes.
//
// `create()` always returns an array of newly-issued codes — even when
// count=1 — so the page can flow CSV export uniformly. The legacy local
// gateway-api typed createRedemption as singleton; the underlying newapi
// endpoint actually returns the full array, so we cast through.

import { GatewayAPI, type GatewayRedemption } from './gateway-api'
import {
  HubListRedemptions,
  HubCreateRedemptions,
  HubDeleteRedemption,
  HubDeleteInvalidRedemptions,
} from '../../wailsjs/go/main/App'

export interface CreateRedemptionInput {
  name: string
  quota: number
  count: number
  expiredTime?: number // unix seconds; 0 means never expires
}

export interface RedemptionSource {
  readonly kind: 'local' | 'hub'

  list(page: number, perPage: number, opts?: { keyword?: string }): Promise<{ items: GatewayRedemption[]; total: number }>
  create(input: CreateRedemptionInput): Promise<GatewayRedemption[]>
  delete(id: number): Promise<void>
  deleteInvalid(): Promise<void>
}

class LocalRedemptionSource implements RedemptionSource {
  readonly kind = 'local' as const
  private api: GatewayAPI

  constructor(baseURL: string, token: string) {
    this.api = new GatewayAPI(baseURL, token)
  }

  async list(page: number, perPage: number, opts?: { keyword?: string }) {
    const kw = opts?.keyword?.trim()
    const res = kw
      ? await this.api.searchRedemptions(kw, page, perPage)
      : await this.api.getRedemptions(page, perPage)
    const items = res.data ?? []
    return { items, total: items.length }
  }

  async create(input: CreateRedemptionInput): Promise<GatewayRedemption[]> {
    // newapi accepts count and returns an array; the legacy TS type is singleton.
    const r = await this.api.createRedemption({
      name: input.name,
      quota: input.quota,
      count: input.count,
      ...(input.expiredTime !== undefined && input.expiredTime > 0
        ? { /* legacy API uses different field; left to backend default */ }
        : {}),
    })
    const data = r.data as unknown
    if (Array.isArray(data)) return data as GatewayRedemption[]
    if (data && typeof data === 'object') return [data as GatewayRedemption]
    return []
  }

  async delete(id: number) { await this.api.deleteRedemption(id) }
  async deleteInvalid() { await this.api.deleteInvalidRedemptions() }
}

class HubRedemptionSource implements RedemptionSource {
  readonly kind = 'hub' as const

  async list(page: number, perPage: number, opts?: { keyword?: string }) {
    const resp = await HubListRedemptions(page + 1, perPage)
    let items = (resp.items ?? []) as unknown as GatewayRedemption[]
    const kw = opts?.keyword?.trim().toLowerCase()
    if (kw) items = items.filter((r) => r.name?.toLowerCase().includes(kw))
    return { items, total: resp.total ?? 0 }
  }

  async create(input: CreateRedemptionInput): Promise<GatewayRedemption[]> {
    const codes = await HubCreateRedemptions(
      input.name,
      input.quota,
      input.count,
      input.expiredTime ?? 0,
    )
    return (codes ?? []) as unknown as GatewayRedemption[]
  }

  async delete(id: number) { await HubDeleteRedemption(id) }
  async deleteInvalid() { await HubDeleteInvalidRedemptions() }
}

export type { GatewayRedemption } from './gateway-api'

export function makeRedemptionSource(args:
  | { mode: 'local'; baseURL: string; token: string }
  | { mode: 'hub' }
): RedemptionSource {
  if (args.mode === 'hub') return new HubRedemptionSource()
  return new LocalRedemptionSource(args.baseURL, args.token)
}

// CSV export helper. Format: id,name,key,quota,created_at — keys are quoted
// because they may contain hyphens. Browser-side download via Blob.
export function downloadRedemptionsCSV(rows: GatewayRedemption[], filename: string) {
  const header = 'id,name,key,quota,created_at\n'
  const lines = rows.map((r) => {
    const created = r.created_time > 0
      ? new Date(r.created_time * 1000).toISOString()
      : ''
    const escape = (s: string) => `"${(s ?? '').replace(/"/g, '""')}"`
    return [r.id, escape(r.name ?? ''), escape(r.key ?? ''), r.quota, escape(created)].join(',')
  }).join('\n')
  const csv = header + lines + '\n'
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
