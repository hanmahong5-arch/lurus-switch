package metering

import (
	"testing"
	"time"
)

func TestStore_RecordAndSummary(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// Record some usage.
	store.Record(Record{AppID: "claude", Model: "claude-sonnet-4-6", TokensIn: 100, TokensOut: 200})
	store.Record(Record{AppID: "claude", Model: "claude-sonnet-4-6", TokensIn: 50, TokensOut: 100})
	store.Record(Record{AppID: "cursor", Model: "gpt-4o", TokensIn: 80, TokensOut: 150})

	// Today's summary.
	summary := store.TodaySummary()
	if summary.TotalCalls != 3 {
		t.Fatalf("expected 3 calls, got %d", summary.TotalCalls)
	}
	if summary.TokensIn != 230 {
		t.Fatalf("expected 230 tokens in, got %d", summary.TokensIn)
	}
	if summary.TokensOut != 450 {
		t.Fatalf("expected 450 tokens out, got %d", summary.TokensOut)
	}
}

func TestStore_AppSummaries(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	store.Record(Record{AppID: "claude", Model: "claude-sonnet-4-6", TokensIn: 100, TokensOut: 200})
	store.Record(Record{AppID: "claude", Model: "claude-haiku-4-5", TokensIn: 50, TokensOut: 100})
	store.Record(Record{AppID: "cursor", Model: "gpt-4o", TokensIn: 80, TokensOut: 150})

	now := time.Now()
	summaries := store.AppSummaries(now.Truncate(24*time.Hour), now)

	if len(summaries) != 2 {
		t.Fatalf("expected 2 app summaries, got %d", len(summaries))
	}

	// Should be sorted by total tokens (descending).
	if summaries[0].AppID != "claude" {
		t.Fatalf("expected claude first (most tokens), got %s", summaries[0].AppID)
	}
	if summaries[0].TotalCalls != 2 {
		t.Fatalf("expected 2 calls for claude, got %d", summaries[0].TotalCalls)
	}
}

func TestStore_ModelSummaries(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	store.Record(Record{AppID: "claude", Model: "claude-sonnet-4-6", TokensIn: 100, TokensOut: 200})
	store.Record(Record{AppID: "cursor", Model: "gpt-4o", TokensIn: 80, TokensOut: 150})
	store.Record(Record{AppID: "codex", Model: "gpt-4o", TokensIn: 60, TokensOut: 100})

	now := time.Now()
	summaries := store.ModelSummaries(now.Truncate(24*time.Hour), now)

	if len(summaries) != 2 {
		t.Fatalf("expected 2 model summaries, got %d", len(summaries))
	}

	// gpt-4o should have more total tokens than claude-sonnet.
	if summaries[0].Model != "gpt-4o" {
		t.Fatalf("expected gpt-4o first, got %s", summaries[0].Model)
	}
	if summaries[0].TotalCalls != 2 {
		t.Fatalf("expected 2 calls for gpt-4o, got %d", summaries[0].TotalCalls)
	}
}

func TestStore_RecentActivity(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	store.Record(Record{AppID: "claude", Model: "claude-sonnet-4-6", TokensIn: 100, TokensOut: 200})
	store.Record(Record{AppID: "cursor", Model: "gpt-4o", TokensIn: 80, TokensOut: 150})

	activity := store.RecentActivity(10)
	if len(activity) != 2 {
		t.Fatalf("expected 2 activity entries, got %d", len(activity))
	}
	// Most recent first.
	if activity[0].AppID != "cursor" {
		t.Fatalf("expected cursor first (most recent), got %s", activity[0].AppID)
	}
}

func TestStore_FlushAndReload(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	store.Record(Record{AppID: "test", Model: "test-model", TokensIn: 42, TokensOut: 84})
	store.Flush()

	// Reload from disk.
	store2, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore reload: %v", err)
	}

	summary := store2.TodaySummary()
	if summary.TotalCalls != 1 {
		t.Fatalf("expected 1 call after reload, got %d", summary.TotalCalls)
	}
	if summary.TokensIn != 42 {
		t.Fatalf("expected 42 tokens in, got %d", summary.TokensIn)
	}
}

func TestStore_CacheHitTracking(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	store.Record(Record{AppID: "test", Model: "m", TokensIn: 10, TokensOut: 20, CachedHit: true})
	store.Record(Record{AppID: "test", Model: "m", TokensIn: 10, TokensOut: 20, CachedHit: false})

	summary := store.TodaySummary()
	if summary.CacheHits != 1 {
		t.Fatalf("expected 1 cache hit, got %d", summary.CacheHits)
	}
}

func TestStore_DaySummaries(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	store.Record(Record{AppID: "test", Model: "m", TokensIn: 10, TokensOut: 20})

	summaries := store.DaySummaries(7)
	if len(summaries) != 7 {
		t.Fatalf("expected 7 daily summaries, got %d", len(summaries))
	}
	// Last entry should be today with 1 call.
	last := summaries[6]
	if last.TotalCalls != 1 {
		t.Fatalf("expected today to have 1 call, got %d", last.TotalCalls)
	}
}
