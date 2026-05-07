// Scans the frontend src for `t('key', '中文 fallback')` calls, then reports
// which keys are missing from frontend/src/i18n/en.json. Run with:
//   bun scripts/i18n_audit.ts
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, extname } from 'node:path'

const ROOT = 'frontend/src'
const EN_PATH = 'frontend/src/i18n/en.json'

function walk(dir: string, out: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry)
    const st = statSync(full)
    if (st.isDirectory()) walk(full, out)
    else if (['.ts', '.tsx'].includes(extname(full))) out.push(full)
  }
  return out
}

function getNested(obj: any, path: string): unknown {
  return path.split('.').reduce((o, k) => (o == null ? undefined : o[k]), obj)
}

const en = JSON.parse(readFileSync(EN_PATH, 'utf8'))
const zh = JSON.parse(readFileSync('frontend/src/i18n/zh.json', 'utf8'))
const files = walk(ROOT)
// Match: t('some.key', '中文 fallback...') or t("...", "...")
// Captures key (group 1) and Chinese fallback (group 2)
const re = /t\(\s*['"]([a-zA-Z0-9._-]+)['"]\s*,\s*['"]([^'"\\]*[一-鿿][^'"\\]*)['"]/g

const missing = new Map<string, { fallback: string; locations: string[] }>()
for (const file of files) {
  const text = readFileSync(file, 'utf8')
  let m: RegExpExecArray | null
  while ((m = re.exec(text)) !== null) {
    const [, key, fallback] = m
    const enHas = getNested(en, key) !== undefined
    const zhHas = getNested(zh, key) !== undefined
    if (!enHas || !zhHas) {
      const entry = missing.get(key) ?? { fallback, locations: [] }
      entry.fallback = fallback
      entry.locations.push(`${file.replace(/\\/g, '/')}${!enHas ? ' [en]' : ''}${!zhHas ? ' [zh]' : ''}`)
      missing.set(key, entry)
    }
  }
}

console.log(`Missing EN keys: ${missing.size}`)
const sorted = [...missing.entries()].sort(([a], [b]) => a.localeCompare(b))
for (const [key, info] of sorted) {
  console.log(`  ${key}\n    zh: ${info.fallback}`)
}
console.log('\nGrouped by namespace:')
const byNs = new Map<string, string[]>()
for (const [key] of sorted) {
  const ns = key.split('.')[0]
  ;(byNs.get(ns) ?? byNs.set(ns, []).get(ns)!).push(key)
}
for (const [ns, keys] of byNs) console.log(`  ${ns}: ${keys.length}`)
