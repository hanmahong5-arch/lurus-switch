package toolmanifest

import "testing"

// Builtin manifest currently flags nullclaw/picoclaw/zeroclaw as coming-soon
// (no installable artifact yet). A wrong upstream manifest that lists URLs
// for these tools must not flip them to stable — the install flow short-
// circuits on IsComingSoon so users never hit unresolvable download hosts.
func TestApplyComingSoonFloor_OverridesUpstreamUrls(t *testing.T) {
	builtin := Builtin()
	if builtin == nil || !builtin.IsComingSoon("picoclaw") {
		t.Fatalf("builtin should mark picoclaw coming-soon; got %+v", builtin)
	}

	// Simulate api.lurus.cn returning placeholder URLs without a coming-soon
	// flag — the historical bad-data shape that triggered this code path.
	upstream := &Manifest{
		Tools: map[string]ToolEntry{
			"picoclaw": {
				Type:          "binary",
				LatestVersion: "0.1.0-test",
				Platforms: map[string]PlatformAsset{
					"windows/amd64": {URL: "https://minio-api.lurus.cn/lurus-releases/tools/picoclaw/v0.1.0-test/picoclaw-windows-amd64.exe"},
				},
			},
			"claude": {
				Type:          "npm",
				NpmPackage:    "@anthropic-ai/claude-code",
				LatestVersion: "1.0.30",
				Status:        "stable",
			},
		},
	}

	floored := applyComingSoonFloor(upstream, builtin)
	if !floored.IsComingSoon("picoclaw") {
		t.Errorf("picoclaw should remain coming-soon after floor; got status %q", floored.Tools["picoclaw"].Status)
	}
	if len(floored.Tools["picoclaw"].Platforms) != 0 {
		t.Errorf("picoclaw platforms should be cleared to prevent unreachable downloads; got %+v", floored.Tools["picoclaw"].Platforms)
	}
	if floored.IsComingSoon("claude") {
		t.Errorf("claude (stable) should not be marked coming-soon")
	}
	if floored.Tools["claude"].LatestVersion != "1.0.30" {
		t.Errorf("claude entry should pass through untouched; got %+v", floored.Tools["claude"])
	}
}

// Operator override (Reseller admin uploading their own CDN URL) must beat
// the builtin coming-soon floor — that's the whole point of the override
// layer. Floor runs before Merge so this is just a regression check on the
// pipeline ordering inside Fetch.
func TestComingSoonFloor_OperatorOverrideWins(t *testing.T) {
	upstream := &Manifest{Tools: map[string]ToolEntry{
		"picoclaw": {Type: "binary", Status: "coming-soon"},
	}}
	overrides := &OverridesFile{Tools: map[string]ToolEntry{
		"picoclaw": {
			Type:          "binary",
			LatestVersion: "1.0.0",
			Status:        "stable",
			Platforms: map[string]PlatformAsset{
				"windows/amd64": {URL: "https://cdn.reseller.example/picoclaw.exe"},
			},
		},
	}}
	floored := applyComingSoonFloor(upstream, Builtin())
	merged := Merge(floored, overrides)
	if merged.IsComingSoon("picoclaw") {
		t.Errorf("operator override should flip picoclaw back to stable; entry=%+v", merged.Tools["picoclaw"])
	}
	if merged.Tools["picoclaw"].Platforms["windows/amd64"].URL != "https://cdn.reseller.example/picoclaw.exe" {
		t.Errorf("operator override URL lost; got %+v", merged.Tools["picoclaw"].Platforms)
	}
}
