package installer

import (
	"context"
	"testing"
)

func TestOpenClawInstaller_Detect_NotInstalled(t *testing.T) {
	rt := NewBunRuntime()
	inst := NewOpenClawInstaller(rt)

	status, err := inst.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect returned unexpected error: %v", err)
	}
	if status == nil {
		t.Fatal("Detect returned nil status")
	}
	if status.Name != ToolOpenClaw {
		t.Errorf("expected Name=%q, got %q", ToolOpenClaw, status.Name)
	}
}

func TestOpenClawInstaller_ConfigureProxy_CreatesConfig(t *testing.T) {
	t.Skip("skipping: requires writable home directory; run manually")
}

func TestOpenClawInstaller_Runtime(t *testing.T) {
	rt := NewBunRuntime()
	inst := NewOpenClawInstaller(rt)
	if inst == nil {
		t.Error("NewOpenClawInstaller returned nil")
	}
	if inst.runtime == nil {
		t.Error("OpenClawInstaller.runtime is nil")
	}
}
