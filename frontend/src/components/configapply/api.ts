import type { ChangePlan, ApplyResult } from './types'

// Wails injects bound methods at window.go.main.App.* at runtime. We avoid
// global declaration merging (other files declare a narrower window.go) and
// instead cast through unknown with a local interface. If you add more
// planners in bindings_apply.go, extend WailsConfigApply below.
interface WailsConfigApply {
  BuildChangePlan: (intent: string, params: Record<string, unknown>) => Promise<ChangePlan>
  ApplyChangePlan: (plan: ChangePlan) => Promise<ApplyResult>
  ListApplyIntents: () => Promise<string[] | null>
}

function requireApp(): WailsConfigApply {
  const root = (window as unknown as { go?: { main?: { App?: WailsConfigApply } } }).go
  const app = root?.main?.App
  if (!app || typeof app.BuildChangePlan !== 'function') {
    throw new Error('configapply: Wails bindings not available; ensure the desktop app has loaded')
  }
  return app
}

export async function buildChangePlan(
  intent: string,
  params: Record<string, unknown>,
): Promise<ChangePlan> {
  return requireApp().BuildChangePlan(intent, params)
}

export async function applyChangePlan(plan: ChangePlan): Promise<ApplyResult> {
  return requireApp().ApplyChangePlan(plan)
}

export async function listApplyIntents(): Promise<string[]> {
  const intents = await requireApp().ListApplyIntents()
  return intents ?? []
}
