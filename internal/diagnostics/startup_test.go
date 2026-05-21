package diagnostics

import (
	"testing"
	"time"
)

func TestRecorder_MarkSequence(t *testing.T) {
	r := &Recorder{}
	r.MarkStart()
	time.Sleep(2 * time.Millisecond)
	r.Mark("phase-a")
	time.Sleep(2 * time.Millisecond)
	r.Mark("phase-b")

	snap := r.Snapshot()
	if len(snap.Phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(snap.Phases))
	}
	if snap.Phases[0].Name != "phase-a" || snap.Phases[1].Name != "phase-b" {
		t.Errorf("phase order wrong: %+v", snap.Phases)
	}
	if snap.ColdStartMS <= 0 {
		t.Errorf("ColdStartMS = %d, want > 0", snap.ColdStartMS)
	}
}

func TestRecorder_MarkBeforeStartIsNoop(t *testing.T) {
	r := &Recorder{}
	r.Mark("ignored") // no MarkStart yet
	snap := r.Snapshot()
	if len(snap.Phases) != 0 {
		t.Errorf("expected Mark before MarkStart to be ignored, got %+v", snap.Phases)
	}
	if snap.ColdStartMS != 0 {
		t.Errorf("ColdStartMS should be 0 before start, got %d", snap.ColdStartMS)
	}
}

func TestRecorder_MarkStartIdempotent(t *testing.T) {
	r := &Recorder{}
	r.MarkStart()
	first := r.start
	time.Sleep(2 * time.Millisecond)
	r.MarkStart() // must not reset
	if !r.start.Equal(first) {
		t.Error("second MarkStart reset the clock")
	}
}

func TestRecorder_GUIReady(t *testing.T) {
	r := &Recorder{}
	r.MarkStart()
	time.Sleep(2 * time.Millisecond)
	r.MarkGUIReady()
	snap := r.Snapshot()
	if snap.GUIReadyMS <= 0 {
		t.Errorf("GUIReadyMS = %d, want > 0", snap.GUIReadyMS)
	}
	// GUI-ready should be <= total cold start (services settle after).
	if snap.GUIReadyMS > snap.ColdStartMS {
		t.Errorf("GUIReadyMS(%d) > ColdStartMS(%d)", snap.GUIReadyMS, snap.ColdStartMS)
	}
}

func TestPersist_RoundTripAndCap(t *testing.T) {
	dir := t.TempDir()
	// Write more than maxKeptTraces to verify the cap holds.
	for i := 0; i < maxKeptTraces+3; i++ {
		r := &Recorder{}
		r.MarkStart()
		r.Mark("only")
		if _, err := r.Persist(dir); err != nil {
			t.Fatal(err)
		}
	}
	hist := History(dir)
	if len(hist) != maxKeptTraces {
		t.Fatalf("history len = %d, want %d (capped)", len(hist), maxKeptTraces)
	}
	if len(hist[0].Phases) == 0 {
		t.Error("round-tripped trace lost its phases")
	}
}

func TestHistory_MissingFile(t *testing.T) {
	if h := History(t.TempDir()); len(h) != 0 {
		t.Errorf("expected empty history for fresh dir, got %d", len(h))
	}
}
