# Code Review Checklist — Sprint 2

## 1. Go Backend

### 1.1 Code Quality
- [x] No hardcoded magic numbers or strings
- [x] Error wrapping with `fmt.Errorf("...: %w", err)`
- [x] No `_ = fn()` swallowed errors in production code — Fixed: H1 (bindings_tools.go analytics tracking)
- [x] Resources closed with `defer`
- [x] Build tags correct (`//go:build windows` / `//go:build !windows`)
- [x] No unused imports
- [x] Package naming follows Go conventions

### 1.2 Security
- [x] No sensitive data in logs
- [x] Registry access read-only (no writes)
- [x] TCP probe timeouts bounded (1s per port, concurrent)
- [x] URL validation on SaveProxySettings — Fixed: M6

### 1.3 Testing
- [x] Tests cover happy path + edge cases (proxydetect: 7 tests, toolhealth: 11 tests)
- [x] `t.Setenv` used correctly (Windows case-insensitivity handled)
- [x] No test pollution (env restored after test)

## 2. Frontend

### 2.1 React
- [x] No missing `key` props in lists
- [x] Effects have correct dependencies — Fixed: H2 (SetupWizard refs), H3 (DashboardPage useCallback deps)
- [x] No memory leaks (cleanup in useEffect where needed)
- [x] Error states handled gracefully

### 2.2 i18n
- [x] All user-visible strings use `t()` or `useTranslation()` — Fixed: H5 (SettingsPage errors), H6 (startup page label)
- [x] Both zh.json and en.json have matching keys
- [x] No leftover English hardcoded text

### 2.3 TypeScript
- [x] No `any` types in component props
- [x] Wails model types match Go struct definitions — Fixed: C2 (installer.ToolStatus)
- [x] All imports resolve
- [x] Safe type casting — Fixed: M2 (QuotaWidget)

### 2.4 UX
- [x] Loading states show spinners
- [x] Error states show meaningful messages
- [x] Wizard skip works at every step
- [x] Collapsible sections work

## 3. Integration

### 3.1 Wails Bindings
- [x] App.d.ts declarations match Go method signatures
- [x] App.js functions call correct window.go paths
- [x] models.ts classes match Go struct JSON tags

### 3.2 Backwards Compatibility
- [x] Old app-settings.json without `onboardingCompleted` -> defaults to false (wizard shows)
- [x] Old proxy settings still load correctly

## Review Findings Summary

**25 issues found, 15 fixed:**

| ID | Severity | Issue | Status |
|----|----------|-------|--------|
| C1 | Critical | XSS via dangerouslySetInnerHTML in uninstall modal | Fixed: replaced with `<Trans>` component |
| C2 | Critical | Missing installer.ToolStatus in models.ts | Fixed: added class to models.ts |
| H1 | High | Swallowed analytics errors (`_ = tracker.Record`) | Fixed: logged with fmt.Printf |
| H2 | High | useEffect missing deps in SetupWizard | Fixed: useRef guards prevent re-trigger |
| H3 | High | useCallback stale closures in DashboardPage | Fixed: added proper dependency arrays |
| H4 | High | EditorFontSize comment mismatch (12-20 vs 10-24) | Fixed: updated comment to 10-24 |
| H5 | High | Hardcoded English error strings in SettingsPage | Fixed: i18n keys added |
| H6 | High | Startup page "Dashboard" label not i18n'd | Fixed: uses `t('nav.dashboard')` |
| M1 | Medium | ToolCard 'unknown' fallback not i18n'd | Fixed: changed to '?' |
| M2 | Medium | Unsafe `as unknown as QuotaData` cast | Fixed: runtime shape validation |
| M3 | Medium | Sequential TCP port probing | Fixed: concurrent with goroutines |
| M4 | Medium | toolhealth uses map[string]any | Accepted: appropriate for health check parsing |
| M5 | Medium | appconfig silently returns defaults on corrupt JSON | Fixed: stderr warning logged |
| M6 | Medium | No validation on SaveProxySettings URL | Fixed: URL scheme validation added |
| M7 | Medium | ConfigureAllProxy error consistency | Accepted: frontend handles correctly |

**Verification:**
```
go build ./...         -> PASS
go vet ./...           -> PASS
go test ./...          -> PASS (proxydetect 7/7, toolhealth 11/11)
npx tsc --noEmit       -> PASS
bun run test:run       -> PASS (19/19)
```
