// channelSource.ts — mode-aware data source for the Channel admin page.
//
// In Personal mode the user runs an in-process newapi gateway (a local
// process, talked to over HTTP via createGatewayClient). In Reseller mode
// the user is running a remote lurus-newhub instance — calls go through
// Wails Hub bindings (bindings_hub.go) which talk to the saved Hub URL +
// admin token from AppSettings.Reseller.
//
// The component-facing surface (`ChannelSource`) abstracts both. The
// `capabilities` field signals which advanced operations the source
// supports — pages should hide UI controls for capabilities that read
// false rather than calling the optional methods.

import { GatewayAPI, type GatewayChannel } from './gateway-api'
import {
  HubListChannels,
  HubSearchChannels,
  HubAddChannel,
  HubUpdateChannel,
  HubDeleteChannel,
  HubDeleteChannelBatch,
  HubTestChannel,
  HubCopyChannel,
  HubBatchSetChannelTag,
  HubEnableChannelsByTag,
  HubDisableChannelsByTag,
  HubEditChannelTag,
  HubFetchChannelModels,
  HubFixChannelAbilities,
} from '../../wailsjs/go/main/App'
import type { admin } from '../../wailsjs/go/models'

export interface ChannelSourceCapabilities {
  search: boolean
  copy: boolean
  fetchModels: boolean
  batchEnableDisable: boolean
  batchSetTag: boolean
  tagOperations: boolean
  fixAbilities: boolean
}

export interface ChannelSource {
  readonly kind: 'local' | 'hub'
  readonly capabilities: ChannelSourceCapabilities

  list(page: number, perPage: number, opts?: { keyword?: string }): Promise<{ items: GatewayChannel[]; total: number }>
  create(input: Partial<GatewayChannel>): Promise<void>
  update(input: Partial<GatewayChannel> & { id: number }): Promise<void>
  delete(id: number): Promise<void>
  batchDelete(ids: number[]): Promise<void>
  test(id: number): Promise<string>

  // Optional: only call when capability flag is true.
  copy?(id: number): Promise<void>
  fetchModels?(id: number): Promise<string[]>
  batchEnable?(ids: number[]): Promise<void>
  batchDisable?(ids: number[]): Promise<void>
  batchSetTag?(ids: number[], tag: string): Promise<void>
  enableByTag?(tag: string): Promise<void>
  disableByTag?(tag: string): Promise<void>
  editTag?(oldTag: string, newTag: string): Promise<void>
  fixAbilities?(): Promise<void>
}

// LocalChannelSource — Personal mode, talks to embedded newapi.
class LocalChannelSource implements ChannelSource {
  readonly kind = 'local' as const
  readonly capabilities: ChannelSourceCapabilities = {
    search: true,
    copy: true,
    fetchModels: true,
    batchEnableDisable: true,
    batchSetTag: true,
    tagOperations: true,
    fixAbilities: true,
  }
  private api: GatewayAPI

  constructor(baseURL: string, token: string) {
    this.api = new GatewayAPI(baseURL, token)
  }

  async list(page: number, perPage: number, opts?: { keyword?: string }) {
    const kw = opts?.keyword?.trim()
    const res = kw
      ? await this.api.searchChannels(kw, page, perPage)
      : await this.api.getChannels(page, perPage)
    const items = res.data ?? []
    return { items, total: items.length }
  }

  async create(input: Partial<GatewayChannel>) {
    await this.api.createChannel(input)
  }
  async update(input: Partial<GatewayChannel> & { id: number }) {
    await this.api.updateChannel(input)
  }
  async delete(id: number) {
    await this.api.deleteChannel(id)
  }
  async batchDelete(ids: number[]) {
    await this.api.batchDeleteChannels(ids)
  }
  async test(id: number) {
    const r = await this.api.testChannel(id)
    return r.data ?? r.message ?? 'OK'
  }
  async copy(id: number) {
    await this.api.copyChannel(id)
  }
  async fetchModels(id: number) {
    const r = await this.api.fetchChannelModels(id)
    return r.data ?? []
  }
  async batchEnable(ids: number[]) {
    await this.api.batchEnableChannels(ids)
  }
  async batchDisable(ids: number[]) {
    await this.api.batchDisableChannels(ids)
  }
  async batchSetTag(ids: number[], tag: string) {
    await this.api.batchSetChannelTag(ids, tag)
  }
  async enableByTag(tag: string) {
    await this.api.enableChannelsByTag(tag)
  }
  async disableByTag(tag: string) {
    await this.api.disableChannelsByTag(tag)
  }
  async editTag(oldTag: string, newTag: string) {
    await this.api.editChannelTag(oldTag, newTag)
  }
  async fixAbilities() {
    await this.api.fixChannelAbilities()
  }
}

// HubChannelSource — Reseller mode, talks to remote lurus-newhub via Wails.
//
// Wave 5 W5.3: capability gap closed — Hub now matches Personal mode on
// tag-scoped enable/disable, batch-tag, fetch-models, and fix-abilities.
//
// `batchEnableDisable` stays false because newhub has no by-ids enable /
// disable endpoint (POST /api/channel/batch is delete-only on Hub) — the
// upstream equivalent is to tag the affected channels then call enable/
// disable by tag. Surfacing that as a separate flow keeps the contract
// honest rather than silently routing batch-enable to a destructive op.
class HubChannelSource implements ChannelSource {
  readonly kind = 'hub' as const
  readonly capabilities: ChannelSourceCapabilities = {
    search: true,
    copy: true,
    fetchModels: true,
    batchEnableDisable: false,
    batchSetTag: true,
    tagOperations: true,
    fixAbilities: true,
  }

  async list(page: number, perPage: number, opts?: { keyword?: string }) {
    // Hub's search endpoint returns the full match set without pagination —
    // when a keyword is set we route to search and ignore page params; the
    // page picks them up after the call as-if total === length.
    const kw = opts?.keyword?.trim()
    if (kw) {
      const items = await HubSearchChannels(kw)
      return { items: items as unknown as GatewayChannel[], total: items.length }
    }
    // Hub paginates from page=1 (1-indexed); the local UI uses 0-indexed.
    const resp = await HubListChannels(page + 1, perPage, '')
    return {
      items: (resp.items ?? []) as unknown as GatewayChannel[],
      total: resp.total ?? 0,
    }
  }

  async create(input: Partial<GatewayChannel>) {
    await HubAddChannel(input as Record<string, unknown>)
  }
  async update(input: Partial<GatewayChannel> & { id: number }) {
    await HubUpdateChannel(input as Record<string, unknown>)
  }
  async delete(id: number) {
    await HubDeleteChannel(id)
  }
  async batchDelete(ids: number[]) {
    await HubDeleteChannelBatch(ids)
  }
  async test(id: number) {
    await HubTestChannel(id, '')
    return 'OK'
  }
  async copy(id: number) {
    await HubCopyChannel(id)
  }
  async fetchModels(id: number) {
    const models = await HubFetchChannelModels(id)
    return models ?? []
  }
  async batchSetTag(ids: number[], tag: string) {
    await HubBatchSetChannelTag(ids, tag)
  }
  async enableByTag(tag: string) {
    await HubEnableChannelsByTag(tag)
  }
  async disableByTag(tag: string) {
    await HubDisableChannelsByTag(tag)
  }
  async editTag(oldTag: string, newTag: string) {
    await HubEditChannelTag(oldTag, newTag)
  }
  async fixAbilities() {
    await HubFixChannelAbilities()
  }
}

// Re-export for component use.
export type { GatewayChannel } from './gateway-api'

// Suppress unused-namespace-import lint without forcing component-level imports.
export type _AdminChannel = admin.Channel

// Create a source for the given mode. `local` is only valid when the
// in-process gateway is running and adminToken is non-empty.
export function makeChannelSource(args:
  | { mode: 'local'; baseURL: string; token: string }
  | { mode: 'hub' }
): ChannelSource {
  if (args.mode === 'hub') return new HubChannelSource()
  return new LocalChannelSource(args.baseURL, args.token)
}
