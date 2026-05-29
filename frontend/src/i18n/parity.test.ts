import { describe, it, expect } from 'vitest'
import zh from './zh.json'
import en from './en.json'

// Walk a translation tree and look up a dotted path. Returns the
// resolved string, or undefined if any segment is missing. Mirrors what
// i18next does internally for `t('a.b.c')` — keeping the helper local
// avoids pulling i18next instance config into a JSON-shape test.
function lookup(root: unknown, path: string): unknown {
  return path.split('.').reduce<unknown>((acc, seg) => {
    if (acc && typeof acc === 'object' && seg in (acc as Record<string, unknown>)) {
      return (acc as Record<string, unknown>)[seg]
    }
    return undefined
  }, root)
}

// Keys added in the multi-step click-flow batch. Each one MUST exist in
// both locales — divergent locale shape is the most common UX bug we
// ship (a Chinese user sees a stub key like "switch.openOrgChart" in
// the UI), and a synchronous parity test catches it at PR time.
const NEW_KEYS = [
  // OwnerBindingModal jump-to-OrgChart
  'switch.openOrgChart',
  'switch.ownerReturnHint',

  // Chargeback empty-state CTA
  'chargeback.gotoBinding',

  // EndUserActivationPage per-kind error CTAs
  'enduser.error.action.retry',
  'enduser.error.action.contact',
  'enduser.error.action.clear',
  'enduser.error.action.noSupport',
  'enduser.error.action.mailSubject',
  'enduser.error.action.mailBody',

  // CommandPalette navigation entries (newapi + governance)
  'commandPalette.commands.goConnectedApps',
  'commandPalette.commands.goChargeback',
  'commandPalette.commands.goOrgChart',
  'commandPalette.commands.goDlp',
  'commandPalette.commands.goAgentTemplates',
  'commandPalette.commands.goPromotion',
  'commandPalette.commands.goPackager',
  'commandPalette.commands.goRelay',
  'commandPalette.commands.goChannels',
  'commandPalette.commands.goTokens',
  'commandPalette.commands.goRedemptions',
  'commandPalette.commands.goLogs',
  'commandPalette.commands.goAudit',
  'commandPalette.commands.undoLastAudit',

  // CommandPalette undo-action user-facing toasts
  'commandPalette.undoNothing',
  'commandPalette.undoDone',

  // NotifyTab — Telegram + Slack transport fields (PR-B push channels)
  'settings.notify.transportHint',
  'settings.notify.telegram.botToken',
  'settings.notify.telegram.botTokenHint',
  'settings.notify.telegram.chatId',
  'settings.notify.telegram.chatIdHint',
  'settings.notify.telegram.bothRequired',
  'settings.notify.slack.webhookUrl',
  'settings.notify.slack.webhookUrlHint',
  'settings.notify.slack.httpsRequired',

  // Settings — OpenTelemetry observability toggle (PR-C)
  'settings.observability.title',
  'settings.observability.desc',
  'settings.observability.endpoint',
  'settings.observability.endpointHint',
] as const

describe('i18n parity — newly added keys', () => {
  for (const key of NEW_KEYS) {
    it(`${key} exists in zh.json`, () => {
      const v = lookup(zh, key)
      expect(v, `Missing in zh.json: ${key}`).toBeTypeOf('string')
      expect(String(v).length).toBeGreaterThan(0)
    })

    it(`${key} exists in en.json`, () => {
      const v = lookup(en, key)
      expect(v, `Missing in en.json: ${key}`).toBeTypeOf('string')
      expect(String(v).length).toBeGreaterThan(0)
    })
  }

  it('mailBody preserves {{code}}, {{hub}}, {{kind}} placeholders in both locales', () => {
    // Without these placeholders the mailto body becomes useless — the
    // user emails an empty body and the reseller has no diagnostics.
    const requireVars = ['{{code}}', '{{hub}}', '{{kind}}']
    for (const locale of ['zh', 'en'] as const) {
      const body = lookup(locale === 'zh' ? zh : en, 'enduser.error.action.mailBody') as string
      for (const v of requireVars) {
        expect(body, `${locale}: mailBody missing ${v}`).toContain(v)
      }
    }
  })

  it('ownerReturnHint preserves {{name}} placeholder', () => {
    expect(lookup(zh, 'switch.ownerReturnHint')).toContain('{{name}}')
    expect(lookup(en, 'switch.ownerReturnHint')).toContain('{{name}}')
  })

  it('undoDone preserves {{op}} placeholder', () => {
    expect(lookup(zh, 'commandPalette.undoDone')).toContain('{{op}}')
    expect(lookup(en, 'commandPalette.undoDone')).toContain('{{op}}')
  })
})
