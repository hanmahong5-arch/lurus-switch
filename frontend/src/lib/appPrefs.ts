// Centralised helpers for the two app-wide preferences exposed in the
// PageHeader quick-switch (language + theme). Both helpers apply the
// change immediately AND persist it to disk via SaveAppSettings so that
// the choice survives a relaunch.
import i18n from '../i18n'
import { GetAppSettings, SaveAppSettings } from '../../wailsjs/go/main/App'
import { appconfig } from '../../wailsjs/go/models'

export type Theme = 'dark' | 'light' | 'auto'
export type Language = 'zh' | 'en'

// Apply a theme to <html> immediately. 'auto' follows the OS via the
// prefers-color-scheme media query.
export function applyTheme(theme: Theme) {
  const root = document.documentElement
  if (theme === 'auto') {
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    root.classList.toggle('dark', prefersDark)
    return
  }
  root.classList.toggle('dark', theme === 'dark')
}

// Persist the new value alongside whatever else is in app-settings.json.
// We re-read the current settings rather than tracking them in a store
// because the user might have changed unrelated fields (e.g. on the
// Settings page) since this client booted.
async function persistPatch(patch: Partial<appconfig.AppSettings>) {
  try {
    const current = await GetAppSettings()
    const merged = appconfig.AppSettings.createFrom({ ...current, ...patch })
    await SaveAppSettings(merged)
  } catch (e) {
    // Non-fatal — the in-memory change still applies for the current
    // session. Surface to console for diagnostics.
    console.error('Failed to persist app prefs:', e)
  }
}

export async function setLanguage(lang: Language) {
  await i18n.changeLanguage(lang)
  await persistPatch({ language: lang })
}

export async function setTheme(theme: Theme) {
  applyTheme(theme)
  await persistPatch({ theme })
}
