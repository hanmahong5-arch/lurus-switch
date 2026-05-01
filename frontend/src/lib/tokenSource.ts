// tokenSource.ts — mode-aware data source for the Token admin page.
//
// Mirrors channelSource.ts. Personal mode → in-process newapi; Reseller →
// remote newhub via Wails Hub bindings. Tokens are simpler than channels —
// no copy/test/tag operations on the upstream — so the surface is just
// list/create/update/delete/batchDelete.

import { GatewayAPI, type GatewayToken } from './gateway-api'
import {
  HubListTokens,
  HubAddToken,
  HubUpdateToken,
  HubDeleteToken,
  HubDeleteTokenBatch,
} from '../../wailsjs/go/main/App'

export interface TokenSourceCapabilities {
  search: boolean
}

export interface TokenSource {
  readonly kind: 'local' | 'hub'
  readonly capabilities: TokenSourceCapabilities

  list(page: number, perPage: number, opts?: { keyword?: string }): Promise<{ items: GatewayToken[]; total: number }>
  create(input: Partial<GatewayToken>): Promise<void>
  update(input: Partial<GatewayToken> & { id: number }): Promise<void>
  delete(id: number): Promise<void>
  batchDelete(ids: number[]): Promise<void>
}

class LocalTokenSource implements TokenSource {
  readonly kind = 'local' as const
  readonly capabilities: TokenSourceCapabilities = { search: true }
  private api: GatewayAPI

  constructor(baseURL: string, token: string) {
    this.api = new GatewayAPI(baseURL, token)
  }

  async list(page: number, perPage: number, opts?: { keyword?: string }) {
    const kw = opts?.keyword?.trim()
    const res = kw
      ? await this.api.searchTokens(kw, page, perPage)
      : await this.api.getTokens(page, perPage)
    const items = res.data ?? []
    return { items, total: items.length }
  }
  async create(input: Partial<GatewayToken>) { await this.api.createToken(input) }
  async update(input: Partial<GatewayToken> & { id: number }) { await this.api.updateToken(input) }
  async delete(id: number) { await this.api.deleteToken(id) }
  async batchDelete(ids: number[]) { await this.api.batchDeleteTokens(ids) }
}

class HubTokenSource implements TokenSource {
  readonly kind = 'hub' as const
  // Hub doesn't expose a token search endpoint yet — keyword is filtered
  // client-side after listing.
  readonly capabilities: TokenSourceCapabilities = { search: false }

  async list(page: number, perPage: number, opts?: { keyword?: string }) {
    const resp = await HubListTokens(page + 1, perPage)
    let items = (resp.items ?? []) as unknown as GatewayToken[]
    const kw = opts?.keyword?.trim().toLowerCase()
    if (kw) {
      items = items.filter((t) => t.name?.toLowerCase().includes(kw))
    }
    return { items, total: resp.total ?? 0 }
  }
  async create(input: Partial<GatewayToken>) {
    await HubAddToken(input as Record<string, unknown>)
  }
  async update(input: Partial<GatewayToken> & { id: number }) {
    await HubUpdateToken(input as Record<string, unknown>)
  }
  async delete(id: number) { await HubDeleteToken(id) }
  async batchDelete(ids: number[]) { await HubDeleteTokenBatch(ids) }
}

export type { GatewayToken } from './gateway-api'

export function makeTokenSource(args:
  | { mode: 'local'; baseURL: string; token: string }
  | { mode: 'hub' }
): TokenSource {
  if (args.mode === 'hub') return new HubTokenSource()
  return new LocalTokenSource(args.baseURL, args.token)
}
