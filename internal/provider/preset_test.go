package provider

import "testing"

func TestPresets_NotEmpty(t *testing.T) {
	presets := Presets()
	if len(presets) == 0 {
		t.Fatal("expected at least one preset")
	}
	// Verify Lurus is included
	lurus := PresetByID("lurus")
	if lurus == nil {
		t.Fatal("expected Lurus preset to exist")
	}
	if lurus.BaseURL != "https://api.lurus.cn" {
		t.Errorf("Lurus baseURL = %q, want https://api.lurus.cn", lurus.BaseURL)
	}
}

func TestPresetByID_NotFound(t *testing.T) {
	p := PresetByID("nonexistent-provider-xyz")
	if p != nil {
		t.Errorf("expected nil for unknown ID, got %+v", p)
	}
}

func TestPresetsByCategory(t *testing.T) {
	china := PresetsByCategory("china")
	if len(china) < 5 {
		t.Errorf("expected at least 5 China providers, got %d", len(china))
	}
	for _, p := range china {
		if p.Category != "china" {
			t.Errorf("preset %s has category %q, want 'china'", p.ID, p.Category)
		}
	}
}

func TestPresets_UniqueIDs(t *testing.T) {
	seen := make(map[string]bool)
	for _, p := range Presets() {
		if seen[p.ID] {
			t.Errorf("duplicate preset ID: %s", p.ID)
		}
		seen[p.ID] = true
	}
}

func TestPresets_AllHaveRequiredFields(t *testing.T) {
	for _, p := range Presets() {
		if p.Name == "" {
			t.Errorf("preset %s has empty name", p.ID)
		}
		if p.BaseURL == "" {
			t.Errorf("preset %s has empty baseURL", p.ID)
		}
		if p.Category == "" {
			t.Errorf("preset %s has empty category", p.ID)
		}
		if p.IconColor == "" {
			t.Errorf("preset %s has empty iconColor", p.ID)
		}
	}
}
