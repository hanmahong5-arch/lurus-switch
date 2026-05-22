// Resume helper for the OwnerBindingModal flow.
//
// When the user clicks "Open Org Chart" inside the modal because they
// haven't imported employees yet, we stash a sessionStorage hint. After
// they finish import and navigate back to Connected Apps, SwitchHubPage
// reads the hint and re-opens the modal against the same app — turning
// a multi-step click into a coherent flow instead of a dead-end.

const KEY = 'switch.pendingOwnerBind'
// Half-hour sanity window. Older hints belong to a previous session and
// would surprise the user if revived.
const MAX_AGE_MS = 30 * 60 * 1000

interface PendingHint {
  appId: string
  appName: string
  ts: number
}

interface AppLike {
  id: string
}

// Storage shape for tests / non-browser environments. The wider `Storage`
// type carries DOM-only members (length, key) we don't need.
export interface MinimalStorage {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
  removeItem(key: string): void
}

function defaultStorage(): MinimalStorage | null {
  // sessionStorage may throw on access in Safari private mode or be
  // missing entirely in non-browser test envs.
  try {
    if (typeof sessionStorage !== 'undefined') return sessionStorage
  } catch { /* fall through */ }
  return null
}

// Persist a hint that the user wanted to bind ownership for this app
// but had to leave first. Failures (storage disabled / quota) are
// swallowed — the resume flow is a nice-to-have, never a blocker.
export function savePendingOwnerBind(
  appId: string,
  appName: string,
  storage: MinimalStorage | null = defaultStorage(),
  now: number = Date.now(),
): void {
  if (!storage) return
  if (!appId) return
  try {
    storage.setItem(KEY, JSON.stringify({ appId, appName, ts: now }))
  } catch { /* swallow */ }
}

// Read-and-consume the hint, returning the matched app from the loaded
// list (or null). The hint is removed regardless of match so the caller
// is never re-prompted on a refresh.
//
// Returns null when:
//   - no hint stored
//   - hint is malformed JSON
//   - hint older than MAX_AGE_MS
//   - hint's appId no longer exists in `apps` (e.g. deleted while away)
export function resolvePendingOwnerBind<T extends AppLike>(
  apps: readonly T[],
  storage: MinimalStorage | null = defaultStorage(),
  now: number = Date.now(),
): T | null {
  if (!storage) return null
  let raw: string | null = null
  try { raw = storage.getItem(KEY) } catch { return null }
  if (!raw) return null
  // Single-shot semantics: always consume, even when stale.
  try { storage.removeItem(KEY) } catch { /* swallow */ }
  let hint: PendingHint
  try {
    hint = JSON.parse(raw) as PendingHint
  } catch {
    return null
  }
  if (!hint || typeof hint.appId !== 'string' || !hint.appId) return null
  if (typeof hint.ts !== 'number' || now - hint.ts > MAX_AGE_MS) return null
  return apps.find((a) => a.id === hint.appId) ?? null
}
