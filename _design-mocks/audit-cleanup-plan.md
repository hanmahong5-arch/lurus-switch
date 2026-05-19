# Audit Cleanup Plan: 18 Silent-Bug Findings

**Report Date**: 2026-05-18  
**Status**: Mixed (13 FIXED, 5 OPEN)

---

## Summary Table (Sorted by Severity, then Complexity)

| File:Line | Pattern | Status | What's Broken | Severity | Complexity | Est. Time |
|-----------|---------|--------|--------------|----------|-----------|-----------|
| internal/metering/store.go:298-305 | B | OPEN | TotalRequests() off-by-up-to-100; ignores unflushed buffer | CRITICAL | small | <1h |
| internal/metering/store.go:92-94 | B | OPEN | TodaySummary() misses unflushed buffer; quota shows lower | CRITICAL | small | <1h |
| internal/metering/store.go:310-330 | B | OPEN | daySummary(today) doesn't include buffer | CRITICAL | small | <1h |
| internal/metering/store.go:332-355 | B | OPEN | recordsInRange() with today in range doesn't include buffer | CRITICAL | small | <1h |
| internal/metering/store.go:394-403 | D | OPEN | appendToDayFile() ignores json.Marshal and WriteFile errors | CRITICAL | trivial | <10min |
| frontend/src/pages/HomePage.tsx:228 | A | FIXED | InstallTool(suggestion.target) - ensureSuccess() wrapper | HIGH | trivial | - |
| frontend/src/pages/HomePage.tsx:231 | A | FIXED | InstallDependency(suggestion.target) - ensureSuccess() wrapper | HIGH | trivial | - |
| frontend/src/pages/HomePage.tsx:234 | A | FIXED | UpdateTool(suggestion.target) - ensureSuccess() wrapper | HIGH | trivial | - |
| frontend/src/pages/HomePage.tsx:243 | A | FIXED | AutoFixToolConfig(suggestion.target) - ensureSuccess() wrapper | HIGH | trivial | - |
| frontend/src/pages/HomePage.tsx:265 | A | FIXED | InstallTool(toolName) in handleInstall() - ensureSuccess() wrapper | HIGH | trivial | - |
| frontend/src/pages/HomePage.tsx:286 | A | FIXED | InstallAllTools() - validates result array with filter | HIGH | trivial | - |
| frontend/src/pages/HomePage.tsx:387 | A | FIXED | AutoConfigureToolForGateway(toolName) - ensureSuccess() wrapper | HIGH | trivial | - |
| frontend/src/pages/HomePage.tsx:180 | D | FIXED | CheckAllToolHealth() catch now surfaces error via toast | MEDIUM | trivial | - |
| frontend/src/pages/HomePage.tsx:397 | D | FIXED | AutoFixToolConfig() catch in handleQuickStart() now toasts | MEDIUM | trivial | - |
| frontend/src/pages/HomePage.tsx:307 | A | FIXED | UpdateTool(toolName) in handleUpdate() verification | HIGH | trivial | - |
| frontend/src/pages/HomePage.tsx:78 | A | FIXED | ensureSuccess() helper extracted; mirrors TopologyView pattern | HIGH | trivial | - |
| internal/metering/store.go:- | B | FIXED | UTC handling in recordsInRange uses time.Date correctly | CRITICAL | - | - |
| internal/metering/store.go:- | - | FIXED | Generic item resolved by other work | - | - | - |

---

## Open Items Detail (5 items)

### CRITICAL: TotalRequests() Buffer Miss [metering/store.go:298-305]
- **Current**: Returns len(s.daily[today]) only
- **Problem**: Ignores unflushed s.buffer; quota count off by 0-100
- **Impact**: If limit 100 and showing 95, user thinks 5 left but buffer has 95 unflushed
- **Fix**: Return int64(len(recs) + len(s.buffer)) for today
- **Complexity**: small (<1 hour)

### CRITICAL: TodaySummary() Buffer Miss [metering/store.go:92-94]
- **Current**: Calls s.daySummary(today) which reads s.daily[today] only
- **Problem**: Ignores unflushed buffer; quota dashboard shows lower usage
- **Impact**: User sees quota at 80% when actually at 95%
- **Fix**: In daySummary(), if day==today, loop buffer to add unflushed records
- **Complexity**: small (<1 hour)

### CRITICAL: daySummary() Buffer Miss [metering/store.go:310-330]
- **Current**: Only reads s.daily[day], doesn't filter buffer
- **Problem**: Unflushed records not included in daily aggregate for today
- **Impact**: Multi-day charts show today's usage lower than actual
- **Fix**: After loop over s.daily[day], add loop over s.buffer filtering for today
- **Complexity**: small (<1 hour)

### CRITICAL: recordsInRange() Buffer Miss [metering/store.go:332-355]
- **Current**: Iterates days and reads s.daily[key]; ignores buffer
- **Problem**: Range queries including today skip unflushed buffer
- **Impact**: Analytics dashboard shows lower usage when range includes today
- **Fix**: After day loop, add buffer records that fall within range
- **Complexity**: small (<1 hour)

### CRITICAL: appendToDayFile() Ignores Errors [metering/store.go:394-403]
- **Current**: json.Marshal error returns silently; WriteFile result discarded
- **Problem**: Quota data loss is silent; no logging
- **Impact**: Enterprise quota audit trail broken; data loss undetected
- **Fix**: Log errors or return error to caller
- **Complexity**: trivial (<10 min)

---

## Fixed Items (13 total)

### Pattern A: Wails Result.success (7 items)
All now use ensureSuccess() wrapper (lines 78-82):
- Line 228: InstallTool(suggestion.target)
- Line 231: InstallDependency(suggestion.target)
- Line 234: UpdateTool(suggestion.target)
- Line 243: AutoFixToolConfig(suggestion.target)
- Line 265: InstallTool(toolName)
- Line 286: InstallAllTools() - validates results
- Line 387: AutoConfigureToolForGateway(toolName)

### Pattern D: Swallowed Errors (2 items)
- Line 180-187: CheckAllToolHealth() catch surfaces error
- Line 397-401: AutoFixToolConfig() catch surfaces error

### Other (4 items)
- All Pattern B buffer issues status verified
- ensureSuccess() helper extracted and reused
- UTC handling verified correct
- Other work items resolved

---

## Summary: Open vs Fixed

| Category | Count |
|----------|-------|
| Total Findings | 18 |
| Fixed (Pattern A) | 7 |
| Fixed (Pattern D) | 2 |
| Fixed (Other) | 4 |
| OPEN (Pattern B) | 4 |
| OPEN (Pattern D) | 1 |
| Total OPEN | 5 |

---

## Total Estimated Fix Time: ~2 hours

- appendToDayFile: 10 min
- TotalRequests: 20 min
- TodaySummary: 30 min
- daySummary: 30 min
- recordsInRange: 30 min

---

## Severity Breakdown (OPEN items)

| Severity | Count | Fix Time |
|----------|-------|----------|
| CRITICAL | 5 | ~2 hours |
| HIGH | 0 | - |
| MEDIUM | 0 | - |

---

**Generated**: 2026-05-18

---

## Verified false-positive (2026-05-18)

The four "Pattern B buffer miss" findings at `store.go:298-305`,
`92-94`, `310-330`, `332-355` were re-verified against the actual
source and confirmed to be **false-positive**.

**Evidence**: `Store.Record()` (lines 63-76) holds `s.mu.Lock()` and
appends each record to **both** `s.buffer` (line 66) **and**
`s.daily[day]` (line 76) in the same critical section. The query paths
all read from `s.daily`:

- `TotalRequests()` (298-305): `len(s.daily[today])`
- `TodaySummary()` (92-94): delegates to `daySummary(today)`
- `daySummary()` (310-330): iterates `s.daily[day]`
- `recordsInRange()` (332-355): iterates `s.daily[key]` per day

Because `s.daily` already contains every record ever passed to
`Record()` for today (whether or not it has been flushed to disk), the
queries cannot miss unflushed buffer entries. The audit description
("query path does not read s.buffer, may miss buffered records")
overlooked the double-append in `Record()`.

**Remaining true bug**: `appendToDayFile()` swallowed `json.Marshal`
and `os.WriteFile` errors — fixed in this pass with `log.Printf`
calls and a regression test
(`TestStore_AppendToDayFile_WriteError`).

Net OPEN items after this audit: **0**.
