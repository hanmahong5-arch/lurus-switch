package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"lurus-switch/internal/appreg"
	"lurus-switch/internal/audit"
	"lurus-switch/internal/capability"
	"lurus-switch/internal/hub/admin"
	"lurus-switch/internal/metering"
)

// TestUndoChannelDelete_Roundtrip exercises the full undo flow:
//
//  1. Wire a fake hub admin server that records inbound calls.
//  2. Override hubClientFactory so undo handlers reach the fake.
//  3. Build a minimal App with a real audit Journal in TempDir.
//  4. Record a channel.delete entry with Before populated (mimicking
//     what HubDeleteChannel would do).
//  5. Invoke journal.Undo and verify the fake server received an
//     equivalent re-create POST.
//
// Tests the wiring (registerAuditUndoHandlers + decodeMap + admin
// client glue), not the journal mechanics — those have their own unit
// coverage in internal/audit.
func TestUndoChannelDelete_Roundtrip(t *testing.T) {
	srv, recorded := newFakeHub(t)
	defer srv.Close()

	withFakeHub(t, srv.URL)

	app := newTestApp(t)
	app.registerAuditUndoHandlers()

	before := &admin.Channel{
		ID:    42,
		Name:  "deepseek-prod",
		Type:  100, // some upstream code
		Group: "default",
	}
	entry := app.auditJournal.Record(auditOpChannelDelete, "42", before, map[string]any{"id": 42}, nil)

	if err := app.auditJournal.Undo(entry.ID); err != nil {
		t.Fatalf("undo failed: %v", err)
	}

	calls := recorded.snapshot()
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 fake-hub call, got %d: %+v", len(calls), calls)
	}
	if calls[0].method != http.MethodPost || calls[0].path != "/api/channel/" {
		t.Errorf("expected POST /api/channel/, got %s %s", calls[0].method, calls[0].path)
	}
	// The wrapper payload is `{"channel": {…}}`. Confirm the prior
	// channel name made it into the re-create.
	wrapper, _ := calls[0].body["channel"].(map[string]any)
	if wrapper == nil {
		t.Fatalf("re-create payload missing channel wrapper: %+v", calls[0].body)
	}
	if wrapper["name"] != "deepseek-prod" {
		t.Errorf("expected re-created channel name=deepseek-prod, got %v", wrapper["name"])
	}
	// Server-assigned id should have been stripped on undo.
	if _, has := wrapper["id"]; has {
		t.Errorf("re-create payload should NOT carry the original id, got: %+v", wrapper)
	}
}

// TestUndoChannelUpdate_Roundtrip confirms update undo posts the
// captured Before back as a PUT — the inverse of the original PATCH.
func TestUndoChannelUpdate_Roundtrip(t *testing.T) {
	srv, recorded := newFakeHub(t)
	defer srv.Close()
	withFakeHub(t, srv.URL)

	app := newTestApp(t)
	app.registerAuditUndoHandlers()

	before := map[string]any{
		"id":      float64(7),
		"name":    "old-name",
		"weight":  float64(1),
	}
	entry := app.auditJournal.Record(auditOpChannelUpdate, "7", before, map[string]any{"id": 7, "name": "new-name"}, nil)

	if err := app.auditJournal.Undo(entry.ID); err != nil {
		t.Fatalf("undo failed: %v", err)
	}
	calls := recorded.snapshot()
	if len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != "/api/channel/" {
		t.Fatalf("expected PUT /api/channel/, got %+v", calls)
	}
	if calls[0].body["name"] != "old-name" {
		t.Errorf("expected name=old-name in undo PUT, got %v", calls[0].body["name"])
	}
}

// TestUndoTokenDeleteBatch_Roundtrip checks the multi-target inverse
// fires one re-create per snapshotted token.
func TestUndoTokenDeleteBatch_Roundtrip(t *testing.T) {
	srv, recorded := newFakeHub(t)
	defer srv.Close()
	withFakeHub(t, srv.URL)

	app := newTestApp(t)
	app.registerAuditUndoHandlers()

	before := []*admin.Token{
		{ID: 1, Name: "t1", Key: "secret-1"},
		{ID: 2, Name: "t2", Key: "secret-2"},
	}
	entry := app.auditJournal.Record(auditOpTokenDeleteBatch, "", before, map[string]any{"ids": []int{1, 2}}, nil)

	if err := app.auditJournal.Undo(entry.ID); err != nil {
		t.Fatalf("undo failed: %v", err)
	}
	calls := recorded.snapshot()
	if len(calls) != 2 {
		t.Fatalf("expected 2 re-creates, got %d", len(calls))
	}
	for _, c := range calls {
		if c.method != http.MethodPost || c.path != "/api/token/" {
			t.Errorf("expected POST /api/token/, got %+v", c)
		}
		// Hub regenerates secrets — undo must drop the old key.
		if _, has := c.body["key"]; has {
			t.Errorf("undo payload must not carry original key (Hub regenerates), got: %+v", c.body)
		}
	}
}

// TestUndoNonReversibleEntry confirms ops without a registered handler
// (e.g. channel.create) are correctly refused.
func TestUndoNonReversibleEntry(t *testing.T) {
	app := newTestApp(t)
	app.registerAuditUndoHandlers()

	// channel.create has no handler in registerAuditUndoHandlers —
	// Reversible is false at journal-write time.
	entry := app.auditJournal.Record(auditOpChannelCreate, "", nil, map[string]any{"name": "x"}, nil)
	err := app.auditJournal.Undo(entry.ID)
	if err == nil {
		t.Error("expected error on non-reversible op, got nil")
	}
	if !strings.Contains(err.Error(), "not reversible") {
		t.Errorf("error should mention non-reversibility, got: %v", err)
	}
}

// --- fake hub ---

type recordedCall struct {
	method string
	path   string
	body   map[string]any
}

type recorder struct {
	mu    sync.Mutex
	calls []recordedCall
}

func (r *recorder) record(c recordedCall) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, c)
}

func (r *recorder) snapshot() []recordedCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]recordedCall, len(r.calls))
	copy(out, r.calls)
	return out
}

// newFakeHub returns an httptest.Server that records every inbound
// call and replies with a minimal Hub-shaped JSON envelope. The
// recorder is returned so tests can assert on what was received.
func newFakeHub(t *testing.T) (*httptest.Server, *recorder) {
	t.Helper()
	rec := &recorder{}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		rec.record(recordedCall{
			method: r.Method,
			path:   r.URL.Path,
			body:   body,
		})
		// Hub's `do` helper expects {"success":true,"data":...} or a
		// bare object. Bare object works for our purposes.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"","data":{}}`))
	})
	return httptest.NewServer(mux), rec
}

// withFakeHub swaps the package-level hubClient factory for the
// duration of a test, restoring it via t.Cleanup.
func withFakeHub(t *testing.T, baseURL string) {
	t.Helper()
	orig := hubClientFactory
	hubClientFactory = func() (*admin.Client, error) {
		return admin.New(admin.Config{BaseURL: baseURL, Token: "test-token"})
	}
	t.Cleanup(func() { hubClientFactory = orig })
}

// newTestApp builds a minimal App carrying a real audit journal in a
// TempDir. The capability gate is set to the all-grant token so undo
// passes the cap check.
func newTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	journal, err := audit.NewJournal(dir)
	if err != nil {
		t.Fatalf("audit.NewJournal: %v", err)
	}
	reg, err := appreg.NewRegistry(dir)
	if err != nil {
		t.Fatalf("appreg.NewRegistry: %v", err)
	}
	meter, err := metering.NewStore(dir)
	if err != nil {
		t.Fatalf("metering.NewStore: %v", err)
	}
	capability.SetCurrent(capability.AllToken("test-user"))
	return &App{services: &services{
		auditJournal: journal,
		appRegistry:  reg,
		meterStore:   meter,
	}}
}
