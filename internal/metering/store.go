package metering

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"lurus-switch/internal/pricing"
)

const (
	meteringDir      = "metering"
	bufferFlushSize  = 100 // flush to disk after this many records
	bufferFlushAge   = 30 * time.Second
	maxMemoryRecords = 5000 // keep recent records in memory
	recentActivityN  = 50   // max entries in activity feed

	// Dedup guard bounds. A record whose stable correlation ID was already
	// seen within dedupTTL is dropped as a client retry; the seen-set is
	// capped at dedupMaxIDs and evicts its oldest entry when full so it can't
	// grow without bound on a long-running gateway.
	dedupTTL    = 10 * time.Minute
	dedupMaxIDs = 4096
)

// Store records and queries API usage metrics.
// Data is kept in memory for fast access and periodically flushed to
// daily JSON files for persistence.
type Store struct {
	mu        sync.RWMutex
	baseDir   string
	buffer    []Record            // unflushed records
	recent    []Record            // recent records (circular, capped)
	daily     map[string][]Record // date → records (loaded on demand)
	lastFlush time.Time

	// Idempotency guard: record ID → first-seen unix-millis. Bounded by
	// dedupMaxIDs with oldest-first eviction. Guarded by mu.
	seenIDs map[string]int64
	// now is the clock used for dedup TTL comparisons; injectable so tests are
	// deterministic. Defaults to time.Now via NewStore (nil falls back to
	// time.Now in clock(), so literal-constructed Stores in tests stay safe).
	now func() time.Time
}

// clock returns the store's current time, defaulting to time.Now when unset.
func (s *Store) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// NewStore creates a metering store rooted at appDataDir/metering/.
func NewStore(appDataDir string) (*Store, error) {
	dir := filepath.Join(appDataDir, meteringDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create metering directory: %w", err)
	}
	s := &Store{
		baseDir:   dir,
		buffer:    make([]Record, 0, bufferFlushSize),
		recent:    make([]Record, 0, recentActivityN),
		daily:     make(map[string][]Record),
		lastFlush: time.Now(),
		seenIDs:   make(map[string]int64),
		now:       time.Now,
	}
	// Pre-load today's records so aggregation is instant.
	today := time.Now().Format("2006-01-02")
	s.daily[today] = s.loadDayFile(today)
	return s, nil
}

// Record writes a single API call record.
//
// Records carrying a stable correlation ID (set by the gateway from the
// client's Idempotency-Key / X-Request-Id) are deduped: a repeat of the same
// ID within dedupTTL is dropped as a client retry, so a retried request bills
// once. A generated-id record (the no-header case) is effectively never a
// duplicate since each id is unique.
func (s *Store) Record(r Record) {
	if r.ID == "" {
		r.ID = generateRecordID()
	}
	if r.Timestamp.IsZero() {
		r.Timestamp = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isDuplicateLocked(r.ID) {
		return // already booked this correlation id within the TTL window
	}

	s.buffer = append(s.buffer, r)

	// Maintain recent activity ring.
	if len(s.recent) >= recentActivityN {
		s.recent = s.recent[1:]
	}
	s.recent = append(s.recent, r)

	// Add to in-memory daily cache.
	day := r.Timestamp.Format("2006-01-02")
	s.daily[day] = append(s.daily[day], r)

	// Flush if buffer is full or old enough.
	if len(s.buffer) >= bufferFlushSize || time.Since(s.lastFlush) > bufferFlushAge {
		s.flushLocked()
	}
}

// Flush writes all buffered records to disk.
func (s *Store) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flushLocked()
}

// TodaySummary returns aggregated usage for today.
func (s *Store) TodaySummary() DailySummary {
	today := time.Now().Format("2006-01-02")
	return s.daySummary(today)
}

// DaySummaries returns daily summaries for the last N days.
func (s *Store) DaySummaries(days int) []DailySummary {
	now := time.Now()
	out := make([]DailySummary, 0, days)
	for i := days - 1; i >= 0; i-- {
		day := now.AddDate(0, 0, -i).Format("2006-01-02")
		out = append(out, s.daySummary(day))
	}
	return out
}

// AppSummaries returns per-app usage for a date range.
func (s *Store) AppSummaries(from, to time.Time) []AppSummary {
	records := s.recordsInRange(from, to)
	byApp := make(map[string]*AppSummary)
	for _, r := range records {
		as, ok := byApp[r.AppID]
		if !ok {
			as = &AppSummary{AppID: r.AppID}
			byApp[r.AppID] = as
		}
		as.TotalCalls++
		as.TokensIn += r.TokensIn
		as.TokensOut += r.TokensOut
		if r.CachedHit {
			as.CacheHits++
		}
		as.CostUSD += pricing.Cost(r.Model, r.TokensIn, r.TokensOut, r.CacheCreateTokens, r.CacheReadTokens)
	}
	out := make([]AppSummary, 0, len(byApp))
	for _, as := range byApp {
		out = append(out, *as)
	}
	sort.Slice(out, func(i, j int) bool {
		return (out[i].TokensIn + out[i].TokensOut) > (out[j].TokensIn + out[j].TokensOut)
	})
	return out
}

// CostCenterSummaries aggregates usage by cost-center for the given
// range. Records without a CostCenter are bucketed under "" (empty);
// the Enterprise dashboard hides that bucket but a Personal/Reseller
// install will only ever produce that bucket.
func (s *Store) CostCenterSummaries(from, to time.Time) []CostCenterSummary {
	records := s.recordsInRange(from, to)
	byCc := make(map[string]*CostCenterSummary)
	emp := make(map[string]map[string]struct{}) // cc -> set of employeeIds
	for _, r := range records {
		cs, ok := byCc[r.CostCenter]
		if !ok {
			cs = &CostCenterSummary{CostCenter: r.CostCenter}
			byCc[r.CostCenter] = cs
			emp[r.CostCenter] = make(map[string]struct{})
		}
		cs.TotalCalls++
		cs.TokensIn += r.TokensIn
		cs.TokensOut += r.TokensOut
		if r.EmployeeID != "" {
			emp[r.CostCenter][r.EmployeeID] = struct{}{}
		}
	}
	out := make([]CostCenterSummary, 0, len(byCc))
	for cc, cs := range byCc {
		cs.UniqueEmps = len(emp[cc])
		out = append(out, *cs)
	}
	sort.Slice(out, func(i, j int) bool {
		return (out[i].TokensIn + out[i].TokensOut) > (out[j].TokensIn + out[j].TokensOut)
	})
	return out
}

// EmployeeSummaries aggregates per-employee usage for the chargeback
// dashboard. Records without an EmployeeID end up under "" — the UI
// labels that bucket as "unattributed" so the admin sees the gap.
func (s *Store) EmployeeSummaries(from, to time.Time) []EmployeeSummary {
	records := s.recordsInRange(from, to)
	byEmp := make(map[string]*EmployeeSummary)
	for _, r := range records {
		es, ok := byEmp[r.EmployeeID]
		if !ok {
			es = &EmployeeSummary{
				EmployeeID: r.EmployeeID,
				CostCenter: r.CostCenter,
			}
			byEmp[r.EmployeeID] = es
		}
		es.TotalCalls++
		es.TokensIn += r.TokensIn
		es.TokensOut += r.TokensOut
		// Last-write-wins on CostCenter — should be stable across the
		// range, but if an admin re-binds an app mid-period the most
		// recent attribution is the one auditors expect.
		if r.CostCenter != "" {
			es.CostCenter = r.CostCenter
		}
	}
	out := make([]EmployeeSummary, 0, len(byEmp))
	for _, es := range byEmp {
		out = append(out, *es)
	}
	sort.Slice(out, func(i, j int) bool {
		return (out[i].TokensIn + out[i].TokensOut) > (out[j].TokensIn + out[j].TokensOut)
	})
	return out
}

// ModelSummaries returns per-model usage for a date range.
func (s *Store) ModelSummaries(from, to time.Time) []ModelSummary {
	records := s.recordsInRange(from, to)
	byModel := make(map[string]*ModelSummary)
	for _, r := range records {
		ms, ok := byModel[r.Model]
		if !ok {
			ms = &ModelSummary{Model: r.Model}
			byModel[r.Model] = ms
		}
		ms.TotalCalls++
		ms.TokensIn += r.TokensIn
		ms.TokensOut += r.TokensOut
		ms.CostUSD += pricing.Cost(r.Model, r.TokensIn, r.TokensOut, r.CacheCreateTokens, r.CacheReadTokens)
	}
	out := make([]ModelSummary, 0, len(byModel))
	for _, ms := range byModel {
		out = append(out, *ms)
	}
	sort.Slice(out, func(i, j int) bool {
		return (out[i].TokensIn + out[i].TokensOut) > (out[j].TokensIn + out[j].TokensOut)
	})
	return out
}

// RecentActivity returns the N most recent API calls for the activity feed.
func (s *Store) RecentActivity(n int) []ActivityEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := 0
	if len(s.recent) > n {
		start = len(s.recent) - n
	}
	slice := s.recent[start:]

	out := make([]ActivityEntry, len(slice))
	for i, r := range slice {
		out[len(slice)-1-i] = ActivityEntry{
			Timestamp: r.Timestamp.Format("15:04"),
			AppID:     r.AppID,
			Model:     r.Model,
			Tokens:    r.TokensIn + r.TokensOut,
		}
	}
	return out
}

// Insights returns aggregated insight data for a date range.
func (s *Store) Insights(from, to time.Time) InsightsRaw {
	records := s.recordsInRange(from, to)
	ins := InsightsRaw{
		ModelTokensIn:  make(map[string]int64),
		ModelTokensOut: make(map[string]int64),
	}
	for _, r := range records {
		ins.TotalCalls++
		ins.TotalTokensIn += r.TokensIn
		ins.TotalTokensOut += r.TokensOut
		ins.TotalLatencyMs += r.LatencyMs
		if r.CachedHit {
			ins.CacheHits++
		}
		if r.StatusCode == 429 {
			ins.RateLimitEvents++
		}
		if r.StatusCode >= 500 {
			ins.ErrorEvents++
		}
		ins.ModelTokensIn[r.Model] += r.TokensIn
		ins.ModelTokensOut[r.Model] += r.TokensOut
		ins.TotalCostUSD += pricing.Cost(r.Model, r.TokensIn, r.TokensOut, r.CacheCreateTokens, r.CacheReadTokens)
	}
	if ins.TotalCalls > 0 {
		ins.AvgLatencyMs = ins.TotalLatencyMs / ins.TotalCalls
	}
	return ins
}

// RecentRecords returns the N most recent raw records from memory.
func (s *Store) RecentRecords(n int) []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n <= 0 || n > len(s.recent) {
		n = len(s.recent)
	}
	start := len(s.recent) - n
	if start < 0 {
		start = 0
	}
	out := make([]Record, len(s.recent)-start)
	copy(out, s.recent[start:])
	return out
}

// TotalRequests returns the lifetime request count (today + buffer).
func (s *Store) TotalRequests() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	today := time.Now().Format("2006-01-02")
	if recs, ok := s.daily[today]; ok {
		return int64(len(recs))
	}
	return 0
}

// --- internal helpers ---

func (s *Store) daySummary(day string) DailySummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records, ok := s.daily[day]
	if !ok {
		records = s.loadDayFile(day)
		// Don't cache old days in memory to save RAM.
	}

	sum := DailySummary{Date: day}
	for _, r := range records {
		sum.TotalCalls++
		sum.TokensIn += r.TokensIn
		sum.TokensOut += r.TokensOut
		if r.CachedHit {
			sum.CacheHits++
		}
		sum.CostUSD += pricing.Cost(r.Model, r.TokensIn, r.TokensOut, r.CacheCreateTokens, r.CacheReadTokens)
	}
	return sum
}

func (s *Store) recordsInRange(from, to time.Time) []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fromY, fromM, fromD := from.Date()
	day := time.Date(fromY, fromM, fromD, 0, 0, 0, 0, from.Location())
	toY, toM, toD := to.Date()
	end := time.Date(toY, toM, toD, 0, 0, 0, 0, to.Location())

	var out []Record
	for d := day; !d.After(end); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		records, ok := s.daily[key]
		if !ok {
			records = s.loadDayFile(key)
		}
		for _, r := range records {
			if !r.Timestamp.Before(from) && !r.Timestamp.After(to) {
				out = append(out, r)
			}
		}
	}
	return out
}

func (s *Store) flushLocked() {
	if len(s.buffer) == 0 {
		return
	}

	// Group by day.
	byDay := make(map[string][]Record)
	for _, r := range s.buffer {
		day := r.Timestamp.Format("2006-01-02")
		byDay[day] = append(byDay[day], r)
	}

	for day, records := range byDay {
		s.appendToDayFile(day, records)
	}

	s.buffer = s.buffer[:0]
	s.lastFlush = time.Now()
}

func (s *Store) dayFilePath(day string) string {
	return filepath.Join(s.baseDir, day+".json")
}

func (s *Store) loadDayFile(day string) []Record {
	fp := s.dayFilePath(day)
	data, err := os.ReadFile(fp)
	if err != nil {
		return nil
	}
	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		return nil
	}
	return records
}

func (s *Store) appendToDayFile(day string, newRecords []Record) {
	fp := s.dayFilePath(day)
	existing := s.loadDayFile(day)
	all := append(existing, newRecords...)
	data, err := json.Marshal(all)
	if err != nil {
		log.Printf("metering: appendToDayFile failed for %s: %v", fp, err)
		return
	}
	// Crash-atomic write: serialize to a sibling temp file, then rename over
	// the day file. os.WriteFile would truncate-then-write in place, so a
	// crash mid-write leaves a half-written (corrupt) ledger for the whole
	// day. Rename is atomic within the same directory, so a reader/recovery
	// either sees the old complete file or the new complete file — never a
	// torn one. The temp file shares the day file's directory to guarantee
	// the rename stays on one filesystem.
	tmp := fp + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		log.Printf("metering: appendToDayFile failed for %s: %v", fp, err)
		return
	}
	if err := os.Rename(tmp, fp); err != nil {
		// Rename failed — drop the temp file so a stale partial doesn't linger.
		_ = os.Remove(tmp)
		log.Printf("metering: appendToDayFile failed for %s: %v", fp, err)
	}
}

// isDuplicateLocked reports whether id was already recorded within dedupTTL
// and, as a side effect, registers id with the current timestamp. Must be
// called with s.mu held. Enforces the dedupMaxIDs bound by evicting the oldest
// entry before inserting a new one.
func (s *Store) isDuplicateLocked(id string) bool {
	if id == "" {
		return false
	}
	if s.seenIDs == nil {
		s.seenIDs = make(map[string]int64)
	}
	nowMs := s.clock().UnixMilli()
	if firstSeen, ok := s.seenIDs[id]; ok {
		if nowMs-firstSeen < dedupTTL.Milliseconds() {
			return true // within the window → a duplicate booking
		}
		// Stale: the TTL lapsed, so treat this as a fresh request and refresh
		// the timestamp rather than dropping it.
		s.seenIDs[id] = nowMs
		return false
	}
	if len(s.seenIDs) >= dedupMaxIDs {
		s.evictOldestLocked()
	}
	s.seenIDs[id] = nowMs
	return false
}

// evictOldestLocked removes the single oldest entry from seenIDs. Must be
// called with s.mu held. O(n) over the bounded map — fine at dedupMaxIDs and
// only runs when the cap is hit.
func (s *Store) evictOldestLocked() {
	var oldestID string
	var oldestTs int64 = math.MaxInt64
	for id, ts := range s.seenIDs {
		if ts < oldestTs {
			oldestTs = ts
			oldestID = id
		}
	}
	if oldestID != "" {
		delete(s.seenIDs, oldestID)
	}
}

func generateRecordID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
